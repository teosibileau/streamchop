use chrono::Utc;
use notify::{Event, EventKind, RecursiveMode, Watcher};
use rumqttc::{Client, MqttOptions, QoS};
use serde::Serialize;
use std::env;
use std::path::Path;
use std::sync::mpsc;
use std::time::Duration;

#[derive(Serialize)]
struct SegmentEvent {
    camera_id: String,
    segment: String,
    playlist: String,
    segment_url: String,
    timestamp: String,
}

fn main() {
    let mqtt_host = env::var("MQTT_HOST").expect("MQTT_HOST is required");
    let mqtt_port: u16 = env::var("MQTT_PORT")
        .unwrap_or_else(|_| "1883".to_string())
        .parse()
        .expect("MQTT_PORT must be a number");
    let topic_prefix = env::var("MQTT_TOPIC_PREFIX").unwrap_or_else(|_| "streamchop".to_string());
    let hls_base_url = env::var("HLS_BASE_URL").expect("HLS_BASE_URL is required");
    let watch_dir = env::var("WATCH_DIR").unwrap_or_else(|_| "/output".to_string());

    let mut mqtt_opts = MqttOptions::new("streamchop-emitter", &mqtt_host, mqtt_port);
    mqtt_opts.set_keep_alive(Duration::from_secs(30));

    let (client, mut connection) = Client::new(mqtt_opts, 64);

    // Spawn connection loop in background
    std::thread::spawn(move || {
        for notification in connection.iter() {
            if let Err(e) = notification {
                eprintln!("MQTT connection error: {e}");
            }
        }
    });

    let (tx, rx) = mpsc::channel::<notify::Result<Event>>();

    let mut watcher = notify::recommended_watcher(tx).expect("Failed to create file watcher");
    watcher
        .watch(Path::new(&watch_dir), RecursiveMode::Recursive)
        .expect("Failed to watch output directory");

    println!("Watching {watch_dir} for new segments...");

    for event in rx {
        let event = match event {
            Ok(e) => e,
            Err(e) => {
                eprintln!("Watch error: {e}");
                continue;
            }
        };

        if !matches!(event.kind, EventKind::Create(_)) {
            continue;
        }

        for path in &event.paths {
            let ext = path.extension().and_then(|e| e.to_str()).unwrap_or("");
            if ext != "ts" {
                continue;
            }

            let filename = match path.file_name().and_then(|f| f.to_str()) {
                Some(f) => f.to_string(),
                None => continue,
            };

            // Extract camera_id from path: /output/<camera_id>/segment_001.ts
            let camera_id = match path.parent() {
                Some(parent) => match parent.file_name().and_then(|f| f.to_str()) {
                    Some(id) => id.to_string(),
                    None => continue,
                },
                None => continue,
            };

            let base = hls_base_url.trim_end_matches('/');
            let event = SegmentEvent {
                playlist: format!("{base}/{camera_id}/stream.m3u8"),
                segment_url: format!("{base}/{camera_id}/{filename}"),
                camera_id: camera_id.clone(),
                segment: filename,
                timestamp: Utc::now().to_rfc3339(),
            };

            let topic = format!("{topic_prefix}/{camera_id}/segment");
            let payload = match serde_json::to_string(&event) {
                Ok(p) => p,
                Err(e) => {
                    eprintln!("JSON error: {e}");
                    continue;
                }
            };

            println!("Publishing to {topic}: {payload}");
            if let Err(e) = client.publish(&topic, QoS::AtLeastOnce, false, payload.as_bytes()) {
                eprintln!("MQTT publish error: {e}");
            }
        }
    }
}

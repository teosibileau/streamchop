mod events;

use chrono::Utc;
use events::{SegmentEvent, SnapshotEvent};
use log::{debug, error, info, warn};
use notify::{Event, EventKind, RecursiveMode, Watcher};
use rumqttc::{Client, MqttOptions, QoS};
use serde::Serialize;
use std::env;
use std::path::Path;
use std::sync::mpsc;
use std::time::Duration;

fn main() {
    env_logger::init();

    let mqtt_host = env::var("MQTT_HOST").expect("MQTT_HOST is required");
    let mqtt_port: u16 = env::var("MQTT_PORT")
        .unwrap_or_else(|_| "1883".to_string())
        .parse()
        .expect("MQTT_PORT must be a number");
    let topic_prefix = env::var("MQTT_TOPIC_PREFIX").unwrap_or_else(|_| "streamchop".to_string());
    let hls_base_url = env::var("HLS_BASE_URL").expect("HLS_BASE_URL is required");
    let watch_dir = env::var("WATCH_DIR").unwrap_or_else(|_| "/output".to_string());
    let segment_duration: u64 = env::var("HLS_TIME")
        .unwrap_or_else(|_| "10".to_string())
        .parse()
        .expect("HLS_TIME must be a number");

    let mut mqtt_opts = MqttOptions::new("streamchop-emitter", &mqtt_host, mqtt_port);
    mqtt_opts.set_keep_alive(Duration::from_secs(30));

    let (client, mut connection) = Client::new(mqtt_opts, 64);

    // Spawn connection loop in background
    std::thread::spawn(move || {
        for notification in connection.iter() {
            if let Err(e) = notification {
                error!("MQTT connection error: {e}");
            }
        }
    });

    let (tx, rx) = mpsc::channel::<notify::Result<Event>>();

    let mut watcher = notify::recommended_watcher(tx).expect("Failed to create file watcher");
    watcher
        .watch(Path::new(&watch_dir), RecursiveMode::Recursive)
        .expect("Failed to watch output directory");

    info!("Watching {watch_dir} for new segments and snapshots...");

    for event in rx {
        let event = match event {
            Ok(e) => e,
            Err(e) => {
                warn!("Watch error: {e}");
                continue;
            }
        };

        if !matches!(event.kind, EventKind::Create(_)) {
            continue;
        }

        for path in &event.paths {
            let ext = path.extension().and_then(|e| e.to_str()).unwrap_or("");
            let filename = match path.file_name().and_then(|f| f.to_str()) {
                Some(f) => f.to_string(),
                None => continue,
            };

            match ext {
                "ts" => handle_segment(path, &filename, &hls_base_url, &topic_prefix, &client),
                "jpg" => handle_snapshot(path, &filename, &hls_base_url, &topic_prefix, segment_duration, &client),
                _ => {}
            }
        }
    }
}

fn handle_segment(path: &Path, filename: &str, base_url: &str, topic_prefix: &str, client: &Client) {
    let camera_id = match extract_camera_id(path) {
        Some(id) => id,
        None => return,
    };

    let epoch = match extract_epoch(filename, "segment_") {
        Some(e) => e,
        None => return,
    };

    let base = base_url.trim_end_matches('/');
    let event = SegmentEvent {
        playlist: format!("{base}/{camera_id}/stream.m3u8"),
        segment_url: format!("{base}/{camera_id}/{filename}"),
        camera_id: camera_id.clone(),
        segment: filename.to_string(),
        segment_epoch: epoch,
        timestamp: Utc::now().to_rfc3339(),
    };

    let topic = format!("{topic_prefix}/{camera_id}/segment");
    publish(client, &topic, &event);
}

fn handle_snapshot(path: &Path, filename: &str, base_url: &str, topic_prefix: &str, segment_duration: u64, client: &Client) {
    let camera_id = match path
        .parent()
        .and_then(|p| p.parent())
        .and_then(|p| p.file_name())
        .and_then(|f| f.to_str())
    {
        Some(id) => id.to_string(),
        None => return,
    };

    let snap_epoch = match extract_epoch(filename, "snap_") {
        Some(e) => e,
        None => return,
    };

    let seg_epoch = snap_epoch - (snap_epoch % segment_duration);
    let segment_name = format!("segment_{seg_epoch}.ts");

    let base = base_url.trim_end_matches('/');
    let event = SnapshotEvent {
        snapshot_url: format!("{base}/{camera_id}/snapshots/{filename}"),
        segment_url: format!("{base}/{camera_id}/{segment_name}"),
        camera_id: camera_id.clone(),
        snapshot: filename.to_string(),
        snapshot_epoch: snap_epoch,
        segment: segment_name,
        segment_epoch: seg_epoch,
        timestamp: Utc::now().to_rfc3339(),
    };

    let topic = format!("{topic_prefix}/{camera_id}/snapshot");
    publish(client, &topic, &event);
}

fn extract_camera_id(path: &Path) -> Option<String> {
    path.parent()
        .and_then(|p| p.file_name())
        .and_then(|f| f.to_str())
        .map(|s| s.to_string())
}

fn extract_epoch(filename: &str, prefix: &str) -> Option<u64> {
    filename
        .strip_prefix(prefix)
        .and_then(|rest| rest.split('.').next())
        .and_then(|num| num.parse().ok())
}

fn publish<T: Serialize>(client: &Client, topic: &str, event: &T) {
    let payload = match serde_json::to_string(event) {
        Ok(p) => p,
        Err(e) => {
            error!("JSON serialization error: {e}");
            return;
        }
    };

    debug!("Publishing to {topic}: {payload}");
    if let Err(e) = client.publish(topic, QoS::AtLeastOnce, false, payload.as_bytes()) {
        error!("MQTT publish error: {e}");
    }
}

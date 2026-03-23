mod config;
mod events;
mod handler;
mod mqtt;

use config::Config;
use handler::Handler;
use log::{info, warn};
use mqtt::MqttPublisher;
use notify::{Event, EventKind, RecursiveMode, Watcher};
use std::path::Path;
use std::sync::mpsc;

fn main() {
    env_logger::init();

    let config = Config::from_env();
    let publisher = MqttPublisher::new(&config);
    let handler = Handler::new(&config, &publisher);

    let (tx, rx) = mpsc::channel::<notify::Result<Event>>();

    let mut watcher = notify::recommended_watcher(tx).expect("Failed to create file watcher");
    watcher
        .watch(Path::new(&config.watch_dir), RecursiveMode::Recursive)
        .expect("Failed to watch output directory");

    info!(
        "Watching {} for new segments and snapshots...",
        config.watch_dir
    );

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
                "ts" => handler.handle_segment(path, &filename),
                "jpg" => handler.handle_snapshot(path, &filename),
                _ => {}
            }
        }
    }
}

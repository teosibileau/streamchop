use log::{debug, error};
use rumqttc::{Client, MqttOptions, QoS};
use serde::Serialize;
use std::time::Duration;

use crate::config::Config;
use crate::events::{SegmentEvent, SnapshotEvent};

pub struct MqttPublisher {
    client: Client,
    topic_prefix: String,
}

impl MqttPublisher {
    pub fn new(config: &Config) -> Self {
        let mut opts = MqttOptions::new("streamchop-emitter", &config.mqtt_host, config.mqtt_port);
        opts.set_keep_alive(Duration::from_secs(30));

        let (client, mut connection) = Client::new(opts, 64);

        std::thread::spawn(move || {
            for notification in connection.iter() {
                if let Err(e) = notification {
                    error!("MQTT connection error: {e}");
                }
            }
        });

        Self {
            client,
            topic_prefix: config.topic_prefix.clone(),
        }
    }

    pub fn publish_segment(&self, camera_id: &str, event: &SegmentEvent) {
        self.publish(camera_id, "segment", event);
    }

    pub fn publish_snapshot(&self, camera_id: &str, event: &SnapshotEvent) {
        self.publish(camera_id, "snapshot", event);
    }

    fn publish<T: Serialize>(&self, camera_id: &str, event_type: &str, event: &T) {
        let topic = format!("{}/{camera_id}/{event_type}", self.topic_prefix);

        let payload = match serde_json::to_string(event) {
            Ok(p) => p,
            Err(e) => {
                error!("JSON serialization error: {e}");
                return;
            }
        };

        debug!("Publishing to {topic}: {payload}");
        if let Err(e) = self.client.publish(&topic, QoS::AtLeastOnce, false, payload.as_bytes()) {
            error!("MQTT publish error: {e}");
        }
    }
}

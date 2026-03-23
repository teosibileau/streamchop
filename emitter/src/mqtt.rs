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
        let topic = Self::build_topic(&self.topic_prefix, camera_id, event_type);

        let payload = match serde_json::to_string(event) {
            Ok(p) => p,
            Err(e) => {
                error!("JSON serialization error: {e}");
                return;
            }
        };

        debug!("Publishing to {topic}: {payload}");
        if let Err(e) = self
            .client
            .publish(&topic, QoS::AtLeastOnce, false, payload.as_bytes())
        {
            error!("MQTT publish error: {e}");
        }
    }

    fn build_topic(prefix: &str, camera_id: &str, event_type: &str) -> String {
        format!("{prefix}/{camera_id}/{event_type}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::Config;

    fn test_config() -> Config {
        Config {
            mqtt_host: "127.0.0.1".to_string(),
            mqtt_port: 1883,
            topic_prefix: "streamchop".to_string(),
            hls_base_url: "http://localhost:8080".to_string(),
            watch_dir: "/output".to_string(),
            segment_duration: 10,
        }
    }

    #[test]
    fn test_build_topic_segment() {
        let topic = MqttPublisher::build_topic("streamchop", "cam1", "segment");
        assert_eq!(topic, "streamchop/cam1/segment");
    }

    #[test]
    fn test_build_topic_snapshot() {
        let topic = MqttPublisher::build_topic("streamchop", "cam1", "snapshot");
        assert_eq!(topic, "streamchop/cam1/snapshot");
    }

    #[test]
    fn test_build_topic_custom_prefix() {
        let topic = MqttPublisher::build_topic("myprefix", "cam3", "segment");
        assert_eq!(topic, "myprefix/cam3/segment");
    }

    #[test]
    fn test_publisher_stores_topic_prefix() {
        let config = test_config();
        let publisher = MqttPublisher::new(&config);
        assert_eq!(publisher.topic_prefix, "streamchop");
    }

    #[test]
    fn test_publisher_construction_does_not_panic() {
        let config = test_config();
        let _publisher = MqttPublisher::new(&config);
    }
}

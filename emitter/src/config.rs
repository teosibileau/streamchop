use std::env;

pub struct Config {
    pub mqtt_host: String,
    pub mqtt_port: u16,
    pub topic_prefix: String,
    pub hls_base_url: String,
    pub watch_dir: String,
    pub segment_duration: u64,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            mqtt_host: env::var("MQTT_HOST").expect("MQTT_HOST is required"),
            mqtt_port: env::var("MQTT_PORT")
                .unwrap_or_else(|_| "1883".to_string())
                .parse()
                .expect("MQTT_PORT must be a number"),
            topic_prefix: env::var("MQTT_TOPIC_PREFIX")
                .unwrap_or_else(|_| "streamchop".to_string()),
            hls_base_url: env::var("HLS_BASE_URL").expect("HLS_BASE_URL is required"),
            watch_dir: env::var("WATCH_DIR").unwrap_or_else(|_| "/output".to_string()),
            segment_duration: env::var("HLS_TIME")
                .unwrap_or_else(|_| "10".to_string())
                .parse()
                .expect("HLS_TIME must be a number"),
        }
    }
}

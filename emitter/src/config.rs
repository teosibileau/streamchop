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

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    // Mutex to prevent tests from running in parallel (env vars are global state)
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    fn set_required_env() {
        env::set_var("MQTT_HOST", "localhost");
        env::set_var("HLS_BASE_URL", "http://localhost:8080");
    }

    fn clear_env() {
        env::remove_var("MQTT_HOST");
        env::remove_var("MQTT_PORT");
        env::remove_var("MQTT_TOPIC_PREFIX");
        env::remove_var("HLS_BASE_URL");
        env::remove_var("WATCH_DIR");
        env::remove_var("HLS_TIME");
    }

    #[test]
    fn test_from_env_with_defaults() {
        let _lock = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        clear_env();
        set_required_env();

        let config = Config::from_env();

        assert_eq!(config.mqtt_host, "localhost");
        assert_eq!(config.mqtt_port, 1883);
        assert_eq!(config.topic_prefix, "streamchop");
        assert_eq!(config.hls_base_url, "http://localhost:8080");
        assert_eq!(config.watch_dir, "/output");
        assert_eq!(config.segment_duration, 10);
    }

    #[test]
    fn test_from_env_with_overrides() {
        let _lock = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        clear_env();
        set_required_env();
        env::set_var("MQTT_PORT", "1884");
        env::set_var("MQTT_TOPIC_PREFIX", "cameras");
        env::set_var("WATCH_DIR", "/data");
        env::set_var("HLS_TIME", "5");

        let config = Config::from_env();

        assert_eq!(config.mqtt_port, 1884);
        assert_eq!(config.topic_prefix, "cameras");
        assert_eq!(config.watch_dir, "/data");
        assert_eq!(config.segment_duration, 5);
    }

    #[test]
    #[should_panic(expected = "MQTT_HOST is required")]
    fn test_missing_mqtt_host() {
        let _lock = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        clear_env();
        env::set_var("HLS_BASE_URL", "http://localhost:8080");

        Config::from_env();
    }

    #[test]
    #[should_panic(expected = "HLS_BASE_URL is required")]
    fn test_missing_hls_base_url() {
        let _lock = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        clear_env();
        env::set_var("MQTT_HOST", "localhost");

        Config::from_env();
    }

    #[test]
    #[should_panic(expected = "MQTT_PORT must be a number")]
    fn test_invalid_mqtt_port() {
        let _lock = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        clear_env();
        set_required_env();
        env::set_var("MQTT_PORT", "not_a_number");

        Config::from_env();
    }
}

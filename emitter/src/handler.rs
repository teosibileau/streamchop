use chrono::Utc;
use std::path::Path;

use crate::config::Config;
use crate::events::{SegmentEvent, SnapshotEvent};
use crate::mqtt::MqttPublisher;

pub struct Handler<'a> {
    config: &'a Config,
    publisher: &'a MqttPublisher,
}

impl<'a> Handler<'a> {
    pub fn new(config: &'a Config, publisher: &'a MqttPublisher) -> Self {
        Self { config, publisher }
    }

    pub fn handle_segment(&self, path: &Path, filename: &str) {
        let camera_id = match Self::extract_camera_id(path) {
            Some(id) => id,
            None => return,
        };

        let epoch = match Self::extract_epoch(filename, "segment_") {
            Some(e) => e,
            None => return,
        };

        let base = self.config.hls_base_url.trim_end_matches('/');
        let event = SegmentEvent {
            playlist: format!("{base}/{camera_id}/stream.m3u8"),
            segment_url: format!("{base}/{camera_id}/{filename}"),
            camera_id: camera_id.clone(),
            segment: filename.to_string(),
            segment_epoch: epoch,
            timestamp: Utc::now().to_rfc3339(),
        };

        self.publisher.publish_segment(&camera_id, &event);
    }

    pub fn handle_snapshot(&self, path: &Path, filename: &str) {
        let snapshots_dir = match path.parent() {
            Some(p) => p,
            None => return,
        };
        let camera_id = match Self::extract_camera_id(snapshots_dir) {
            Some(id) => id,
            None => return,
        };

        let snap_epoch = match Self::extract_epoch(filename, "snap_") {
            Some(e) => e,
            None => return,
        };

        let seg_epoch = snap_epoch - (snap_epoch % self.config.segment_duration);
        let segment_name = format!("segment_{seg_epoch}.ts");

        let base = self.config.hls_base_url.trim_end_matches('/');
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

        self.publisher.publish_snapshot(&camera_id, &event);
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
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_camera_id_from_segment_path() {
        let path = Path::new("/output/cam1/segment_1711195200.ts");
        assert_eq!(Handler::extract_camera_id(path), Some("cam1".to_string()));
    }

    #[test]
    fn test_extract_camera_id_from_snapshots_dir() {
        let snap_path = Path::new("/output/cam2/snapshots/snap_1711195203.jpg");
        let snapshots_dir = snap_path.parent().unwrap();
        assert_eq!(
            Handler::extract_camera_id(snapshots_dir),
            Some("cam2".to_string())
        );
    }

    #[test]
    fn test_extract_camera_id_root_returns_none() {
        let path = Path::new("/");
        assert_eq!(Handler::extract_camera_id(path), None);
    }

    #[test]
    fn test_extract_epoch_segment() {
        assert_eq!(
            Handler::extract_epoch("segment_1711195200.ts", "segment_"),
            Some(1711195200)
        );
    }

    #[test]
    fn test_extract_epoch_snapshot() {
        assert_eq!(
            Handler::extract_epoch("snap_1711195203.jpg", "snap_"),
            Some(1711195203)
        );
    }

    #[test]
    fn test_extract_epoch_wrong_prefix() {
        assert_eq!(
            Handler::extract_epoch("segment_1711195200.ts", "snap_"),
            None
        );
    }

    #[test]
    fn test_extract_epoch_invalid_number() {
        assert_eq!(Handler::extract_epoch("segment_abc.ts", "segment_"), None);
    }
}

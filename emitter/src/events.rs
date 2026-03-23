use serde::Serialize;

#[derive(Serialize)]
pub struct SegmentEvent {
    pub camera_id: String,
    pub segment: String,
    pub playlist: String,
    pub segment_url: String,
    pub segment_epoch: u64,
    pub timestamp: String,
}

#[derive(Serialize)]
pub struct SnapshotEvent {
    pub camera_id: String,
    pub snapshot: String,
    pub snapshot_url: String,
    pub snapshot_epoch: u64,
    pub segment: String,
    pub segment_url: String,
    pub segment_epoch: u64,
    pub timestamp: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_segment_event_serializes() {
        let event = SegmentEvent {
            camera_id: "cam1".to_string(),
            segment: "segment_1711195200.ts".to_string(),
            playlist: "http://localhost:8080/cam1/stream.m3u8".to_string(),
            segment_url: "http://localhost:8080/cam1/segment_1711195200.ts".to_string(),
            segment_epoch: 1711195200,
            timestamp: "2026-03-23T12:00:00+00:00".to_string(),
        };

        let json = serde_json::to_value(&event).unwrap();

        assert_eq!(json["camera_id"], "cam1");
        assert_eq!(json["segment"], "segment_1711195200.ts");
        assert_eq!(json["segment_epoch"], 1711195200);
        assert_eq!(json["playlist"], "http://localhost:8080/cam1/stream.m3u8");
        assert_eq!(json["segment_url"], "http://localhost:8080/cam1/segment_1711195200.ts");
    }

    #[test]
    fn test_snapshot_event_serializes() {
        let event = SnapshotEvent {
            camera_id: "cam1".to_string(),
            snapshot: "snap_1711195203.jpg".to_string(),
            snapshot_url: "http://localhost:8080/cam1/snapshots/snap_1711195203.jpg".to_string(),
            snapshot_epoch: 1711195203,
            segment: "segment_1711195200.ts".to_string(),
            segment_url: "http://localhost:8080/cam1/segment_1711195200.ts".to_string(),
            segment_epoch: 1711195200,
            timestamp: "2026-03-23T12:00:03+00:00".to_string(),
        };

        let json = serde_json::to_value(&event).unwrap();

        assert_eq!(json["camera_id"], "cam1");
        assert_eq!(json["snapshot"], "snap_1711195203.jpg");
        assert_eq!(json["snapshot_epoch"], 1711195203);
        assert_eq!(json["segment"], "segment_1711195200.ts");
        assert_eq!(json["segment_epoch"], 1711195200);
    }

    #[test]
    fn test_snapshot_segment_epoch_alignment() {
        let snap_epoch: u64 = 1711195207;
        let segment_duration: u64 = 10;
        let seg_epoch = snap_epoch - (snap_epoch % segment_duration);

        assert_eq!(seg_epoch, 1711195200);
    }

    #[test]
    fn test_segment_event_has_all_fields() {
        let event = SegmentEvent {
            camera_id: "cam2".to_string(),
            segment: "segment_1711195210.ts".to_string(),
            playlist: "http://host/cam2/stream.m3u8".to_string(),
            segment_url: "http://host/cam2/segment_1711195210.ts".to_string(),
            segment_epoch: 1711195210,
            timestamp: "2026-03-23T12:00:10+00:00".to_string(),
        };

        let json = serde_json::to_string(&event).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&json).unwrap();

        for field in ["camera_id", "segment", "playlist", "segment_url", "segment_epoch", "timestamp"] {
            assert!(parsed.get(field).is_some(), "missing field: {field}");
        }
    }

    #[test]
    fn test_snapshot_event_has_all_fields() {
        let event = SnapshotEvent {
            camera_id: "cam2".to_string(),
            snapshot: "snap_1711195213.jpg".to_string(),
            snapshot_url: "http://host/cam2/snapshots/snap_1711195213.jpg".to_string(),
            snapshot_epoch: 1711195213,
            segment: "segment_1711195210.ts".to_string(),
            segment_url: "http://host/cam2/segment_1711195210.ts".to_string(),
            segment_epoch: 1711195210,
            timestamp: "2026-03-23T12:00:13+00:00".to_string(),
        };

        let json = serde_json::to_string(&event).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&json).unwrap();

        for field in ["camera_id", "snapshot", "snapshot_url", "snapshot_epoch", "segment", "segment_url", "segment_epoch", "timestamp"] {
            assert!(parsed.get(field).is_some(), "missing field: {field}");
        }
    }
}

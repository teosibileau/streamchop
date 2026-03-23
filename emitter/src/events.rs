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

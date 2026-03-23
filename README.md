# StreamChop

A multi-service Docker pipeline that ingests RTSP camera streams, chops them into HLS segments with JPEG snapshots, serves them over HTTP, and emits MQTT events.

## Architecture

```
N x Cameras (RTSP) --> Chopper (FFmpeg) --> .ts segments + .m3u8 + snapshots
                                                     |
                                     Emitter (Rust) --> MQTT events
                                     Nginx -----------> HLS + snapshots over HTTP
```

FFmpeg remuxes H.264 without re-encoding (`-c copy`) for HLS segments. Snapshots are decoded at 1 fps. Runs fine on a Pi.

## Services

| Service | Image | Description |
|---------|-------|-------------|
| `chopper_cam_1` | Alpine + FFmpeg | RTSP to HLS segments + JPEG snapshots (one per camera) |
| `emitter` | Rust (Alpine) | Watches output dirs, publishes MQTT on new segments and snapshots |
| `mqtt` | RabbitMQ + MQTT plugin | Message broker with management UI on `:15672` |
| `nginx` | Nginx (Alpine) | Serves HLS files and snapshots over HTTP on `:8080` |

## Quick start

```bash
cp .env.example .env   # fill in camera URL + HLS_BASE_URL with your host IP
ahoy setup             # install pre-commit hooks
ahoy docker up         # start all services
```

HLS stream available at `http://<host-ip>:8080/cam1/stream.m3u8`
Latest snapshot at `http://<host-ip>:8080/cam1/snapshots/snap_<epoch>.jpg`
RabbitMQ management at `http://localhost:15672` (guest/guest)

## Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `CAM1_RTSP_URL` | RTSP stream URL for camera 1 | `rtsp://admin:pass@192.168.1.100:554/cam/realmonitor?channel=1&subtype=0` |
| `MQTT_HOST` | MQTT broker hostname | `mqtt` |
| `MQTT_PORT` | MQTT broker port | `1883` |
| `MQTT_TOPIC_PREFIX` | Prefix for MQTT topics | `streamchop` |
| `HLS_BASE_URL` | Public base URL for HLS files | `http://192.168.1.237:8080` |

## MQTT events

### Segment events

Topic: `<prefix>/<camera_id>/segment`

```json
{
  "camera_id": "cam1",
  "segment": "segment_1711195200.ts",
  "playlist": "http://192.168.1.237:8080/cam1/stream.m3u8",
  "segment_url": "http://192.168.1.237:8080/cam1/segment_1711195200.ts",
  "segment_epoch": 1711195200,
  "timestamp": "2026-03-23T12:00:00+00:00"
}
```

### Snapshot events

Topic: `<prefix>/<camera_id>/snapshot`

```json
{
  "camera_id": "cam1",
  "snapshot": "snap_1711195203.jpg",
  "snapshot_url": "http://192.168.1.237:8080/cam1/snapshots/snap_1711195203.jpg",
  "snapshot_epoch": 1711195203,
  "segment": "segment_1711195200.ts",
  "segment_url": "http://192.168.1.237:8080/cam1/segment_1711195200.ts",
  "segment_epoch": 1711195200,
  "timestamp": "2026-03-23T12:00:03+00:00"
}
```

Snapshot-to-segment matching: the snapshot epoch is rounded down to the nearest 10-second boundary to find its parent segment.

## File naming

Segments and snapshots use Unix epoch timestamps:
- `segment_1711195200.ts` — segment starting at epoch 1711195200
- `snap_1711195203.jpg` — snapshot taken at epoch 1711195203

The playlist keeps 360 segments (1 hour of rewind). Old segments are retained on disk.

## Adding cameras

Duplicate the chopper service in `docker-compose.yml`:

```yaml
  chopper_cam_2:
    build: ./chopper
    container_name: streamchop-chopper-cam2
    restart: unless-stopped
    environment:
      - RTSP_URL=${CAM2_RTSP_URL}
    volumes:
      - ./output/cam2:/output
```

Add `CAM2_RTSP_URL` to `.env`. The emitter automatically picks up new subdirectories under `output/`.

## Ahoy commands

| Command | Description |
|---------|-------------|
| `ahoy setup` | Install pre-commit hooks |
| `ahoy clean` | Remove all segments and snapshots from the output folder |
| `ahoy docker up` | Start containers |
| `ahoy docker stop` | Stop containers (non-destructive) |
| `ahoy docker build` | Build containers |
| `ahoy docker ps` | List running containers |
| `ahoy docker log` | Follow container logs |
| `ahoy docker reset` | Stop, remove, and restart containers |
| `ahoy docker destroy` | Stop and remove containers |
| `ahoy docker cleanup` | Remove unused images and volumes |
| `ahoy docker exec` | Execute a command in a running container |
| `ahoy docker run` | Run a one-off command |

## Project structure

```
chopper/                   # FFmpeg chopper image
  Dockerfile
  entrypoint.sh
emitter/                   # Rust MQTT emitter image
  Cargo.toml
  src/main.rs
  Dockerfile
nginx/                     # Nginx HLS server image
  nginx.conf
  Dockerfile
.ahoy.yml                  # Root ahoy config
.ahoy/docker.ahoy.yml      # Docker commands
.env.example                # Environment template
docker-compose.yml          # Service definitions
output/                     # Per-camera HLS segments + snapshots (gitignored)
  cam1/
    segment_<epoch>.ts
    stream.m3u8
    snapshots/
      snap_<epoch>.jpg
```

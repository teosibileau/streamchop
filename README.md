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

## Edge deployment

Deploy minimal edge units close to the cameras — a Jetson Nano or Pi on the same VLAN. No repo clone needed.

### 1. Install

```bash
curl -fsSL https://raw.githubusercontent.com/teosibileau/streamchop/main/install.sh | sh
```

### 2. Setup

```bash
./streamchop setup
```

The wizard will:
- Scan your network for ONVIF cameras via WS-Discovery
- Let you select which cameras to configure
- Prompt for credentials per camera (with a "same for all" shortcut)
- Probe each camera for its RTSP stream URI
- Scan for an MQTT broker on the network (or skip MQTT entirely)
- Auto-detect the host IP for HLS access
- Generate `docker-compose.dist.yml` and `.env`
- Offer to install as a systemd service

### 3. Service management

```bash
./streamchop install     # install and start the systemd service
./streamchop status      # check service health
./streamchop uninstall   # remove the service
```

The systemd service uses a watchdog — if any container exits, the entire stack is restarted.

### Re-running setup

Run `./streamchop setup` again at any time. It will pre-select previously configured cameras and pre-fill credentials from the existing `.env`.

### Pinning image versions

Edit `.env` to pin to a specific tag:

```
TAG=sha-abc123d
```

## Development

### Prerequisites

- Docker and Docker Compose
- [Ahoy](https://github.com/ahoy-cli/ahoy) task runner
- Go 1.23+ (for TUI development)
- [pre-commit](https://pre-commit.com/) (for linting hooks)

### Quick start

```bash
git clone git@github.com:teosibileau/streamchop.git
cd streamchop
cp .env.example .env   # fill in camera URL + HLS_BASE_URL with your host IP
ahoy setup             # install pre-commit hooks
ahoy profile dev docker up  # start all services including local MQTT broker
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

## Ahoy commands

### General

| Command | Description |
|---------|-------------|
| `ahoy setup` | Install pre-commit hooks |
| `ahoy clean` | Remove all segments and snapshots from the output folder |

### Docker

| Command | Description |
|---------|-------------|
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

### Profiles

| Command | Description |
|---------|-------------|
| `ahoy profile dev` | Include dev services (MQTT broker) in compose commands |

Usage: `ahoy profile dev docker up` starts all services including the local MQTT broker.

### TUI setup tool

| Command | Description |
|---------|-------------|
| `ahoy tui build` | Build the streamchop binary locally |
| `ahoy tui run` | Run the setup wizard |
| `ahoy tui test` | Run TUI tests |

## Project structure

```
chopper/                        # FFmpeg chopper image
  Dockerfile
  entrypoint.sh
emitter/                        # Rust MQTT emitter image
  Cargo.toml
  src/main.rs
  Dockerfile
nginx/                          # Nginx HLS server image
  nginx.conf
  Dockerfile
tui/                            # streamchop CLI tool (Go)
  main.go
  model.go
  onvif/                        # ONVIF camera discovery + RTSP probing
  compose/                      # docker-compose.dist.yml + .env generation
  steps/                        # TUI wizard steps
  systemd/                      # Embedded systemd service template + watchdog
.ahoy.yml                       # Root ahoy config
.ahoy/
  docker.ahoy.yml               # Docker commands
  profile.ahoy.yml              # Compose profile helpers
  tui.ahoy.yml                  # TUI build/run/test commands
.env.example                     # Environment template
docker-compose.yml               # Dev service definitions (includes MQTT via profile)
docker-compose.dist.yml          # Production compose (generated by streamchop setup, gitignored)
install.sh                       # One-liner installer for the streamchop binary
output/                          # Per-camera HLS segments + snapshots (gitignored)
  cam1/
    segment_<epoch>.ts
    stream.m3u8
    snapshots/
      snap_<epoch>.jpg
```

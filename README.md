# StreamChop

A multi-service Docker pipeline that ingests RTSP camera streams, chops them into HLS segments, serves them over HTTP, and emits MQTT events for each new segment.

## Architecture

```
N x Cameras (RTSP) --> Chopper (FFmpeg -c copy) --> .ts segments + .m3u8
                                                          |
                                          Emitter (Rust) --> MQTT events
                                          Nginx -----------> HLS over HTTP
```

FFmpeg remuxes H.264 without re-encoding (`-c copy`). Bottleneck is network, not CPU — runs fine on a Pi.

## Services

| Service | Image | Description |
|---------|-------|-------------|
| `chopper_cam_1` | Alpine + FFmpeg | RTSP to HLS segmenter (one per camera) |
| `emitter` | Rust (Alpine) | Watches output dirs, publishes MQTT on new segments |
| `mqtt` | RabbitMQ + MQTT plugin | Message broker with management UI on `:15672` |
| `nginx` | Nginx (Alpine) | Serves HLS files over HTTP on `:8080` |

## Quick start

```bash
cp .env.example .env   # fill in camera URL + HLS_BASE_URL with your host IP
ahoy setup             # install pre-commit hooks
ahoy docker up         # start all services
```

HLS stream available at `http://<host-ip>:8080/cam1/stream.m3u8`
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

Each new `.ts` segment triggers a message on topic `<prefix>/<camera_id>/segment`:

```json
{
  "camera_id": "cam1",
  "segment": "segment_042.ts",
  "playlist": "http://192.168.1.237:8080/cam1/stream.m3u8",
  "segment_url": "http://192.168.1.237:8080/cam1/segment_042.ts",
  "timestamp": "2026-03-23T12:00:00+00:00"
}
```

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
output/                     # HLS segments per camera (gitignored)
```

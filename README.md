# StreamChop

A Docker container that consumes RTSP streams and chops them into HLS segments with a `.m3u8` playlist.

## How it works

```
Camera (RTSP) → FFmpeg (-c copy) → .ts segments + .m3u8
```

FFmpeg remuxes the H.264 stream without re-encoding. Bottleneck is network, not CPU — runs fine on a Pi.

## Quick start

```bash
cp .env.example .env   # fill in your RTSP_URL
ahoy docker up         # or: docker compose up -d --build
```

HLS output lands in `./output/` — point any HLS player (VLC, Safari) at `output/stream.m3u8`.

## Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `RTSP_URL` | Full RTSP stream URL | `rtsp://admin:pass@192.168.1.100:554/cam/realmonitor?channel=1&subtype=0` |

## Ahoy commands

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

## Project structure

```
.ahoy.yml                 # Root ahoy config
.ahoy/docker.ahoy.yml     # Docker commands
.env.example               # Environment template
Dockerfile                 # Alpine + FFmpeg
docker-compose.yml         # Service definition
entrypoint.sh              # FFmpeg launch script
output/                    # HLS segments (gitignored)
```

#!/bin/bash
set -e
INTERVAL=20

trap 'echo "[watchdog] Caught SIGTERM, exiting"; exit 0' SIGTERM

systemd-notify READY=1
docker compose -f docker-compose.dist.yml up -d

while sleep "$INTERVAL"; do
  if [[ -n "$(docker compose -f docker-compose.dist.yml ps --status exited --format json)" ]]; then
    echo "[watchdog] Container exited — restarting stack"
    exit 1
  fi
  systemd-notify WATCHDOG=1
done

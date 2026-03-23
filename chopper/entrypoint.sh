#!/bin/sh
set -e

if [ -z "$RTSP_URL" ]; then
  echo "ERROR: RTSP_URL is not set" >&2
  exit 1
fi

mkdir -p /output/snapshots

exec ffmpeg -rtsp_transport tcp -i "$RTSP_URL" \
  -c copy \
  -f hls \
  -hls_time 10 \
  -hls_list_size 360 \
  -strftime 1 \
  -hls_segment_filename /output/segment_%s.ts \
  /output/stream.m3u8 \
  -vf fps=1 \
  -q:v 2 \
  -strftime 1 \
  /output/snapshots/snap_%s.jpg

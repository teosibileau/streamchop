#!/bin/sh
set -e

if [ -z "$RTSP_URL" ]; then
  echo "ERROR: RTSP_URL is not set" >&2
  exit 1
fi

exec ffmpeg -rtsp_transport tcp -i "$RTSP_URL" \
  -c copy \
  -f hls \
  -hls_time 10 \
  -hls_list_size 6 \
  -hls_flags delete_segments \
  -hls_segment_filename /output/segment_%03d.ts \
  /output/stream.m3u8

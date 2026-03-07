#!/bin/bash
set -e

VIDEO_DEV=/dev/baby-cam
AUDIO_DEV=hw:WEBCAM,0
AUDIO_RATE=48000

arecord -D "$AUDIO_DEV" -f S16_LE -c1 -r "$AUDIO_RATE" --buffer-time=200000 2>/dev/null | \
ffmpeg -hide_banner -loglevel warning \
  -f v4l2 -framerate 15 -video_size 1280x720 -i "$VIDEO_DEV" \
  -f wav -i - \
  -vf fps=15,format=yuv420p \
  -c:v libx264 -preset veryfast -tune zerolatency -g 15 \
  -b:v 1500k \
  -c:a libopus -b:a 96k -ar "$AUDIO_RATE" -ac 1 \
  -f rtsp rtsp://localhost:8554/baby 2>&1 | \
while IFS= read -r line; do echo "[$(date '+%Y-%m-%d %H:%M:%S')] $line"; done

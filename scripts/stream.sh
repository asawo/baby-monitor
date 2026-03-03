#!/bin/bash
set -e

VIDEO_DEV=/dev/video0
AUDIO_DEV=hw:3,0
AUDIO_RATE=48000

exec arecord -D "$AUDIO_DEV" -f S16_LE -c1 -r "$AUDIO_RATE" --buffer-time=200000 | \
ffmpeg -hide_banner -loglevel info \
  -f v4l2 -framerate 15 -video_size 1280x720 -i "$VIDEO_DEV" \
  -f wav -i - \
  -vf fps=15,format=yuv420p \
  -c:v libx264 -preset veryfast -tune zerolatency -g 15 \
  -b:v 1500k \
  -c:a libopus -b:a 96k -ar "$AUDIO_RATE" -ac 1 \
  -f rtsp rtsp://localhost:8554/baby

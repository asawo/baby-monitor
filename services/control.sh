#!/bin/bash
set -e

[ "$(id -u)" -ne 0 ] && exec sudo "$0" "$@"

start() {
  echo "Starting mediamtx..."
  systemctl start mediamtx
  echo "Starting stream service..."
  systemctl start stream.service
  echo "Starting HTTP server..."
  systemctl start monitor-http.service
  echo "Starting cry detection..."
  systemctl start detect.service
  echo "Done."
}

stop() {
  echo "Stopping cry detection..."
  systemctl stop detect.service
  echo "Stopping HTTP server..."
  systemctl stop monitor-http.service
  echo "Stopping stream service..."
  systemctl stop stream.service
  echo "Stopping mediamtx..."
  systemctl stop mediamtx
  echo "Done."
}

status() {
  systemctl status mediamtx --no-pager
  systemctl status stream.service --no-pager
  systemctl status monitor-http.service --no-pager
  systemctl status detect.service --no-pager
}

case "$1" in
  start)  start  ;;
  stop)   stop   ;;
  status) status ;;
  *)      echo "Usage: $0 {start|stop|status}"; exit 1 ;;
esac

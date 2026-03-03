#!/bin/bash
set -e

INSTALL_DIR="${HOME}/monitor"
MEDIAMTX_VERSION=v1.16.3
MEDIAMTX_URL="https://github.com/bluenviron/mediamtx/releases/download/${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_arm64.tar.gz"

echo "==> Downloading mediamtx ${MEDIAMTX_VERSION}..."
wget -O /tmp/mediamtx.tar.gz "$MEDIAMTX_URL"
tar -xzf /tmp/mediamtx.tar.gz -C "$INSTALL_DIR" mediamtx
rm /tmp/mediamtx.tar.gz
chmod +x "$INSTALL_DIR/mediamtx"
echo "    mediamtx installed to $INSTALL_DIR/mediamtx"

echo "==> Installing systemd services..."
bash "$INSTALL_DIR/scripts/install-services.sh"

echo "==> Enabling services..."
sudo systemctl enable mediamtx stream.service monitor-http.service

echo "Done. Reboot or run './scripts/monitor.sh start' to start the monitor."

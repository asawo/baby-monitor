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

echo "==> Installing udev rules..."
sudo cp "$INSTALL_DIR/scripts/udev/99-baby-monitor.rules" /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger --action=add --subsystem-match=video4linux

echo "==> Installing Python dependencies for cry detection..."
pip3 install --quiet -r "$INSTALL_DIR/requirements.txt"

echo "==> Downloading YAMNet cry detection model..."
mkdir -p "$INSTALL_DIR/models"
wget -q -O "$INSTALL_DIR/models/yamnet.tflite" \
  "https://storage.googleapis.com/download.tensorflow.org/models/tflite/task_library/audio_classification/rpi/yamnet_float32.tflite"
echo "    YAMNet model installed to $INSTALL_DIR/models/yamnet.tflite"

echo "==> Installing systemd services..."
bash "$INSTALL_DIR/scripts/install-services.sh"

echo "==> Enabling services..."
sudo systemctl enable mediamtx stream.service monitor-http.service detect.service

echo "Done. Reboot or run './scripts/monitor.sh start' to start the monitor."
echo "NOTE: Edit .env and set NTFY_TOPIC before starting detect.service."

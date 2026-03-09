#!/bin/bash
set -e

INSTALL_DIR="${HOME}/monitor"
MEDIAMTX_VERSION=v1.16.3
MEDIAMTX_URL="https://github.com/bluenviron/mediamtx/releases/download/${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_arm64.tar.gz"
YAMNET_URL="https://storage.googleapis.com/mediapipe-models/audio_classifier/yamnet/float32/latest/yamnet.tflite"
YAMNET_CLASS_MAP_URL="https://raw.githubusercontent.com/tensorflow/models/master/research/audioset/yamnet/yamnet_class_map.csv"

if [ ! -f "$INSTALL_DIR/mediamtx" ]; then
    echo "==> Downloading mediamtx ${MEDIAMTX_VERSION}..."
    wget -O /tmp/mediamtx.tar.gz "$MEDIAMTX_URL"
    tar -xzf /tmp/mediamtx.tar.gz -C "$INSTALL_DIR" mediamtx
    rm /tmp/mediamtx.tar.gz
    chmod +x "$INSTALL_DIR/mediamtx"
    echo "    mediamtx installed to $INSTALL_DIR/mediamtx"
else
    echo "==> mediamtx already installed, skipping."
fi

echo "==> Installing udev rules..."
sudo cp "$INSTALL_DIR/config/99-baby-monitor.udev.rules" /etc/udev/rules.d/99-baby-monitor.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --action=add --subsystem-match=video4linux

echo "==> Installing Python dependencies for cry detection..."
[ -d "$INSTALL_DIR/venv" ] || python3 -m venv "$INSTALL_DIR/venv"
"$INSTALL_DIR/venv/bin/pip" install --quiet -r "$INSTALL_DIR/services/detect/requirements.txt"

if [ ! -f "$INSTALL_DIR/models/yamnet.tflite" ]; then
    echo "==> Downloading YAMNet model..."
    mkdir -p "$INSTALL_DIR/models"
    wget -q -O "$INSTALL_DIR/models/yamnet.tflite" "$YAMNET_URL"
    echo "    YAMNet model installed to $INSTALL_DIR/models/yamnet.tflite"
else
    echo "==> YAMNet model already present, skipping."
fi

if [ ! -f "$INSTALL_DIR/models/yamnet_class_map.csv" ]; then
    echo "==> Downloading YAMNet class map..."
    wget -q -O "$INSTALL_DIR/models/yamnet_class_map.csv" "$YAMNET_CLASS_MAP_URL"
    echo "    Class map installed to $INSTALL_DIR/models/yamnet_class_map.csv"
else
    echo "==> YAMNet class map already present, skipping."
fi

echo "==> Installing systemd services..."
bash "$INSTALL_DIR/setup/systemd.sh"

echo "==> Enabling services..."
sudo systemctl enable mediamtx stream.service monitor-http.service detect.service

echo "Done. Reboot or run 'services/control.sh start' to start the monitor."
echo "NOTE: Edit .env and set NTFY_TOPIC before starting detect.service."

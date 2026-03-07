INSTALL_USER="${USER}"
INSTALL_DIR="${HOME}/monitor"
LOG_DIR="${INSTALL_DIR}/logs"
mkdir -p "$LOG_DIR"

sudo tee /etc/systemd/system/mediamtx.service > /dev/null <<UNIT
[Unit]
Description=mediamtx WebRTC server
After=network.target

[Service]
User=${INSTALL_USER}
Group=${INSTALL_USER}
ExecStart=${INSTALL_DIR}/mediamtx ${INSTALL_DIR}/mediamtx.yml
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/stream.service > /dev/null <<UNIT
[Unit]
Description=Baby Monitor Stream
After=network.target sound.target mediamtx.service dev-baby\x2dcam.device
BindsTo=dev-baby\x2dcam.device

[Service]
User=${INSTALL_USER}
Group=${INSTALL_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/services/stream/stream.sh
Restart=always
RestartSec=3
StandardOutput=append:${LOG_DIR}/ffmpeg.log
StandardError=append:${LOG_DIR}/ffmpeg.log

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/monitor-http.service > /dev/null <<UNIT
[Unit]
Description=Baby Monitor HTTP Server
After=network.target mediamtx.service stream.service

[Service]
User=${INSTALL_USER}
Group=${INSTALL_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStartPre=/bin/sh -c 'fuser -k 80/tcp 2>/dev/null || true'
ExecStart=${INSTALL_DIR}/monitor
Restart=always
RestartSec=3
StandardOutput=append:${LOG_DIR}/monitor.log
StandardError=append:${LOG_DIR}/monitor.log
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/detect.service > /dev/null <<UNIT
[Unit]
Description=Baby Monitor Cry Detection
After=network.target stream.service mediamtx.service

[Service]
User=${INSTALL_USER}
Group=${INSTALL_USER}
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/services/detect/detect.sh
Restart=always
RestartSec=5
StandardOutput=append:${LOG_DIR}/detect.log
StandardError=append:${LOG_DIR}/detect.log

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload

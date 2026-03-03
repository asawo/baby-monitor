INSTALL_USER="${USER}"
INSTALL_DIR="${HOME}/monitor"

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
After=network.target sound.target mediamtx.service

[Service]
User=${INSTALL_USER}
Group=${INSTALL_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/scripts/stream.sh
Restart=always
RestartSec=3
StandardOutput=append:${INSTALL_DIR}/ffmpeg.log
StandardError=append:${INSTALL_DIR}/ffmpeg.log

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
StandardOutput=append:${INSTALL_DIR}/monitor.log
StandardError=append:${INSTALL_DIR}/monitor.log
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload

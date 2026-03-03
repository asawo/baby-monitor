# Baby Monitor

A self-hosted baby monitor that streams live video and audio from a Raspberry Pi to any browser on your local network using WebRTC.

The webcam I'm using is `Logicool Web Camera C270nd HD 720P`.

## How it works

Three systemd services run on the Pi:

| Service | Role |
|---|---|
| `stream` | FFmpeg captures video (v4l2) and audio (ALSA) and pushes an RTSP stream |
| `mediamtx` | Receives the RTSP stream and re-publishes it over WebRTC (WHEP) |
| `monitor-http` | Go HTTP server serves the web UI on port 80 |

Open `http://monitor.local` in a browser and the page connects directly to the WebRTC stream.

## Requirements

- Raspberry Pi (arm64) with a USB or CSI camera and microphone
- SSH access from your dev machine
- Go 1.21+ (to build the HTTP server binary)
- `rsync` and `scp` on your dev machine

## Setup

**1. Configure your environment**

```bash
cp .env.example .env
```

Edit `.env` with your Pi's details:

```
PI_USER=pi           # username on the Pi
PI_HOST=monitor.local  # hostname or IP of the Pi
REMOTE=/home/pi/monitor  # deployment path on the Pi
PI_IP=192.168.1.100  # Pi's LAN IP (used for WebRTC ICE candidates)
```

**2. Deploy and install**

```bash
make install
```

This builds the Go binary, syncs all files to the Pi (including a generated `mediamtx.yml`), downloads mediamtx, and registers the three systemd services.

**3. Start the monitor**

```bash
make start
```

Then open `http://<PI_HOST>` in your browser.

## Makefile targets

| Target | Description |
|---|---|
| `make build` | Cross-compile the Go HTTP server for linux/arm64 |
| `make setup` | Generate `mediamtx.yml` from `mediamtx.yml.example` using `PI_IP` |
| `make sync` | Sync project files to the Pi |
| `make deploy` | Build + sync + install binary |
| `make install` | Full install: sync files + download mediamtx + register services |
| `make start` | Start all services on the Pi |
| `make stop` | Stop all services on the Pi |
| `make status` | Show systemd status for all services |
| `make logs` | Tail the FFmpeg log |
| `make logs-http` | Tail the HTTP server log |

## Hardware tested on

- Raspberry Pi 4 (arm64)
- USB webcam with built-in microphone (`Logicool C270nd HD 720P`)

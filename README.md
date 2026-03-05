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

### Web UI

The single-page UI (`public/index.html`) provides:

- **Live video** — WebRTC stream at a fixed 16:9 aspect ratio
- **Audio waveform** — scrolling amplitude visualiser with colour-coded levels (green → yellow → red); resizes with the window while preserving history
- **Mute / fullscreen** — overlay buttons on the video
- **Connectivity status** — live indicator in the controls bar that polls `/api/status` every 10 seconds; clicking it opens a log modal that fetches recent service logs from `/api/logs`

### Device stability

USB device paths (`/dev/videoN`, ALSA card numbers) can change across reboots depending on enumeration order. To prevent this:

- **Video**: a udev rule (`scripts/udev/99-baby-monitor.rules`) matches the webcam by USB vendor/product ID and creates a stable symlink at `/dev/baby-cam`. The stream service is bound to this device unit, so systemd waits for the camera to appear before starting and stops the service if it's unplugged.
- **Audio**: ALSA is addressed by card name (`hw:WEBCAM,0`) rather than card number, which is derived from the USB device descriptor and stays consistent regardless of enumeration order.

## Requirements

- Raspberry Pi (arm64) with a USB or CSI camera and microphone
- SSH access from your dev machine
- [`just`](https://github.com/casey/just) — install with `brew install just`
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
just install
```

This builds the Go binary, syncs all files to the Pi (including a generated `mediamtx.yml`), downloads mediamtx, installs the udev rules for stable device paths, and registers the three systemd services.

**3. Start the monitor**

```bash
just start
```

Then open `http://<PI_HOST>` in your browser.

## Commands

Run `just --list` to see all available recipes.

| Command | Description |
|---|---|
| `just build` | Cross-compile the Go HTTP server for linux/arm64 |
| `just setup` | Generate `mediamtx.yml` from `mediamtx.yml.example` using `PI_IP` |
| `just sync` | Sync project files to the Pi |
| `just deploy` | Build + sync + install binary |
| `just install` | Full install: sync files + download mediamtx + register services |
| `just start` | Start all services on the Pi |
| `just stop` | Stop all services on the Pi |
| `just status` | Show systemd status for all services |
| `just logs` | Tail the FFmpeg log |
| `just logs-http` | Tail the HTTP server log |

## Hardware tested on

- Raspberry Pi 4 (arm64)
- USB webcam with built-in microphone (`Logicool C270nd HD 720P`)

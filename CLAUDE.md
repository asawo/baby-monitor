# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Raspberry Pi baby monitor that streams webcam/microphone via RTSP → WebRTC to a browser. Target audience: owner/developer with admin access to the Pi. Requires a **64-bit Raspberry Pi OS** (`uname -m` → `aarch64`) — the Go binary cross-compiles to `linux/arm64` and will not run on 32-bit OS.

Four services run on the Pi: FFmpeg (capture), mediamtx (WebRTC relay), a Go HTTP server (serves the web UI), and a Python ML cry detector.

## Development Commands

Requires [`just`](https://github.com/casey/just) and a configured `.env` file (copy from `.env.example`). All commands that SSH/rsync to the Pi require `PI_USER`, `PI_HOST`, `REMOTE`, and `PI_IP` to be set in `.env`.

```sh
just build       # Cross-compile Go server for Linux ARM64 (runs gen + tsc first)
just deploy      # Build + rsync + restart HTTP service on Pi
just install     # Full setup: sync files, download mediamtx, register systemd services
just start       # Start all services on Pi
just stop        # Stop all services
just restart     # Restart all services on Pi
just status      # Show systemd status for all four services
just logs        # Tail FFmpeg stream log (~/.monitor/logs/ffmpeg.log)
just logs-http   # Tail HTTP server log (~/.monitor/logs/monitor.log)
just logs-detect # Tail sound detection log (~/.monitor/logs/detect.log)
just setup       # Generate mediamtx.yml from template (substitutes YOUR_PI_IP → PI_IP)
just sync        # Rsync source files to Pi (without binary)
just gen         # Generate JS/TS types from Go API structs (via tygo)
just test        # Run cry detection unit tests (test_cry.py) on the Pi — unit only, not integration
```

No build/lint/test tooling beyond `go build` — the Go server is minimal stdlib-only.

## Architecture

```
Pi:
  USB Webcam → FFmpeg (stream.sh) → RTSP :8554/baby
                                         ↓
                                    mediamtx → WebRTC :8889
                                         ↓
                                    Go HTTP :80 → index.html
Browser:
  index.html → WebRTC WHEP (WebRTC-HTTP Egress Protocol) → mediamtx:8889/baby/whep
```

**Services** (managed by systemd, all with `Restart=on-failure`):
- `stream.service` — FFmpeg captures `/dev/baby-cam` (v4l2) + `hw:WEBCAM,0` (ALSA), outputs RTSP
- `mediamtx.service` — Bridges RTSP to WebRTC (WHEP protocol); restarts after 3s
- `monitor-http.service` — Go binary serves `./server/web` on port 80 via `CAP_NET_BIND_SERVICE` (not root); restarts after 3s
- `detect.service` — Python YAMNet ML cry detection, reports to Go server; restarts after 5s

**State**: All detection state (last cry, last fart, notifications toggle) is held in-memory by the Go server. Data resets on `monitor-http.service` restart — there is no persistence layer.

**Directory structure**:
```
server/
  cmd/main.go                # Go HTTP server entry point + routing
  cmd/logger.go              # Logging setup
  cmd/middleware.go          # HTTP middleware
  internal/api/              # API handler functions + types
  internal/state/            # Mutex-protected shared state
  web/                       # Frontend static assets (served by Go)
    index.html
    css/style.css
    src/                     # TypeScript source (compiled to dist/ by tsc)
      app.ts, waveform.ts, webrtc.ts, controls.ts,
      detection.ts, logs.ts, notifications.ts,
      status.ts, theme.ts
    dist/                    # Compiled JS output
services/
  control.sh                 # Start/stop/status all services
  stream/stream.sh           # FFmpeg capture
  detect/sound_detection.py  # YAMNet ML cry + fart/poop detection
  detect/detect.sh           # Python wrapper for systemd
  detect/requirements.txt    # Python deps (run in ~/monitor/venv)
  detect/test_cry.py         # Unit tests for detection logic
  detect/README.md           # Detect service docs
setup/
  install.sh                 # Full Pi setup
  systemd.sh                 # Systemd unit generator
config/
  mediamtx.yml.example       # mediamtx config template
  99-baby-monitor.udev.rules # Udev rule for stable /dev/baby-cam
```

**Device stability**: A udev rule ([config/99-baby-monitor.udev.rules](config/99-baby-monitor.udev.rules)) creates a stable `/dev/baby-cam` symlink by matching the Logitech C270 USB IDs, so stream.service binds to `dev-baby-cam.device`.

**Web UI** ([server/web/index.html](server/web/index.html)): SPA. HTML in `index.html`, styles in [server/web/css/style.css](server/web/css/style.css), TypeScript source in [server/web/src/](server/web/src/), compiled to `server/web/dist/` by `tsc` (run as part of `just build`). Uses RTCPeerConnection for WebRTC, Web Audio API for the canvas waveform visualizer. Tuning constants at top of `waveform.ts`:
- `SCROLL_MS` — pixels per ms of scroll speed
- `SENSITIVITY` — visual amplitude scale
- `YELLOW_THRESHOLD` / `RED_THRESHOLD` — color thresholds

**Go server** ([server/cmd/main.go](server/cmd/main.go)): Entry point, routing, middleware. API handlers in [server/internal/api/handlers.go](server/internal/api/handlers.go). Shared state in [server/internal/state/state.go](server/internal/state/state.go). Serves `./server/web` as static files on `:80`. No dedicated `/health` endpoint — use `GET /api/status` or `just status` to check service health.

**Cry/fart detector** ([services/detect/sound_detection.py](services/detect/sound_detection.py)): YAMNet TFLite model (`~/monitor/models/yamnet.tflite`) runs inference on 0.975s audio windows (15,600 samples at 16kHz). FFmpeg pipes RTSP audio as raw PCM. Auto-reconnects on stream failure with 5s retry. Fart wetness (poop detection) uses FFT-based spectral analysis comparing high-frequency energy ratio against `WETNESS_THRESHOLD`. Python virtualenv at `~/monitor/venv` (created by `just install`).

## Environment

`.env` (copied from `.env.example`) — all vars consumed on Pi by `detect.service` and during `just` commands from dev machine:

| Variable | Required | Default | Description |
|---|---|---|---|
| `PI_USER` | yes | — | SSH user (e.g. `pi`) |
| `PI_HOST` | yes | — | SSH hostname (e.g. `monitor.local`) |
| `REMOTE` | yes | — | Remote install path (e.g. `/home/pi/monitor`) |
| `PI_IP` | yes | — | Pi's LAN IP for WebRTC ICE candidates |
| `NTFY_TOPIC` | yes | — | ntfy.sh topic for push notifications (no default — required for alerts) |
| `NTFY_URL` | no | `https://ntfy.sh` | ntfy.sh base URL |
| `CRY_THRESHOLD` | no | `0.3` | YAMNet confidence threshold for cry detection |
| `CRY_COOLDOWN` | no | `120` | Seconds between cry notifications |
| `FART_THRESHOLD` | no | `0.3` | YAMNet confidence threshold for fart detection |
| `FART_COOLDOWN` | no | `120` | Seconds between fart notifications |
| `WETNESS_THRESHOLD` | no | `0.5` | High-frequency energy ratio for poop classification |
| `WETNESS_HF_CUTOFF` | no | `1000` | Hz cutoff for high-frequency spectral analysis |
| `DEBUG_AUDIO` | no | `0` | Set to `1` to log top 5 YAMNet class scores per window |

`mediamtx.yml` is generated by `just setup` from `config/mediamtx.yml.example` — do not edit the generated file directly. The only substitution is `YOUR_PI_IP` → `PI_IP`.

## Security

- **No authentication** — the web UI on port 80 is open to anyone on the network. Relies on local network isolation.
- **No HTTPS** — traffic (video, audio, API) is plaintext on the LAN.
- **SSH**: `just` commands assume key-based SSH auth to `PI_USER@PI_HOST`. Password auth should be disabled on the Pi.
- **Firewall**: Restrict ports 80 (HTTP), 8554 (RTSP), and 8889 (WebRTC) to your LAN subnet via `ufw` or `iptables` to prevent exposure if the Pi is on a shared or bridged network.

## Logs & Observability

Log file paths on the Pi:

| Service | Log |
|---|---|
| `stream.service` | `~/monitor/logs/ffmpeg.log` |
| `monitor-http.service` | `~/monitor/logs/monitor.log` |
| `detect.service` | `~/monitor/logs/detect.log` |
| `mediamtx.service` | journalctl only |

Follow logs live on Pi: `journalctl -u mediamtx.service -f`

From dev machine: `just logs`, `just logs-http`, `just logs-detect`

## Troubleshooting

**Camera not found / stream fails**
```sh
ls -la /dev/baby-cam           # Check udev symlink exists
udevadm trigger                # Re-apply udev rules
v4l2-ctl --list-formats-ext -d /dev/baby-cam  # Verify supported formats
ffplay rtsp://localhost:8554/baby             # Test RTSP stream locally on Pi
```

**Audio / detection issues**
```sh
arecord -D hw:WEBCAM,0 -f S16_LE -r 16000 -c 1 /tmp/test.wav  # Test audio capture
# Set DEBUG_AUDIO=1 in .env and restart detect.service to log YAMNet scores per window
just logs-detect               # Watch detection output
```

**WebRTC not connecting in browser**
- Verify `PI_IP` in `mediamtx.yml` matches the Pi's current LAN IP (`hostname -I`)
- Re-run `just setup && just deploy` after any IP change
- Check `journalctl -u mediamtx.service -f` for ICE/WHEP errors

**Service not restarting**
- All services have `Restart=on-failure` — check `just status` and `journalctl -u <service> -n 50`

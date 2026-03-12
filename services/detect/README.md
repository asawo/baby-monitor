# Sound Detection Service

Python ML service that classifies audio from the RTSP stream to detect baby cries, dry farts, and wet farts (poops). Sends push notifications via ntfy.sh and reports detections to the Go HTTP server for UI display.

## How it works

### Audio pipeline

FFmpeg is spawned as a subprocess, decoding the RTSP stream to raw 16kHz mono PCM on stdout. The detector reads in 0.975s chunks (15,600 samples at 16kHz — exactly one YAMNet input window), classifies each chunk, and loops forever. If the stream drops, it retries every 5 seconds.

### Cry detection

Each audio window is fed to [YAMNet](https://github.com/google-research/audioset-ontology), a TFLite model trained on Google's AudioSet. YAMNet outputs per-frame scores for 521 audio classes; the detector takes the mean across frames and checks class 20 ("Baby cry, infant cry"). If the score exceeds `CRY_THRESHOLD`, a cry is reported.

### Fart / poop detection

Same YAMNet pass — class 55 ("Fart") is checked against `FART_THRESHOLD`. If triggered, a second analysis runs on the same audio window to determine wetness.

### Wetness analysis (poop vs dry fart)

A wet fart has more high-frequency energy than a dry one. The detector splits the window into 1024-sample Hann-windowed frames with 50% overlap, computes the fraction of spectral energy above `WETNESS_HF_CUTOFF` Hz for each frame, and takes the **peak** (not mean) across frames. Peak is used so a brief wet event isn't diluted by surrounding silence.

If the peak ratio exceeds `WETNESS_THRESHOLD`, the fart is classified as wet (poop) and a push notification is sent. Dry farts are UI-only.

```
FFmpeg stdout (raw PCM)
  └─ 15600-sample window
       ├─ YAMNet → class 20 score → cry?
       └─ YAMNet → class 55 score → fart?
                       └─ FFT wetness → wet or dry?
```

### Notifications

Push notifications are sent via [ntfy.sh](https://ntfy.sh) (requires `NTFY_TOPIC`). The notification state (enabled/disabled) is polled from the Go server every 30 seconds, so the UI toggle takes effect within half a minute. Separate cooldown timers prevent notification spam.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `NTFY_TOPIC` | *(required)* | ntfy.sh topic for push notifications |
| `NTFY_URL` | `https://ntfy.sh` | ntfy server base URL (override for self-hosted) |
| `CRY_THRESHOLD` | `0.6` | YAMNet score threshold for cry detection (0.0–1.0) |
| `CRY_COOLDOWN` | `120` | Minimum seconds between cry notifications |
| `FART_THRESHOLD` | `0.6` | YAMNet score threshold for fart detection (0.0–1.0) |
| `FART_COOLDOWN` | `120` | Minimum seconds between fart/poop notifications |
| `WETNESS_THRESHOLD` | `0.5` | Peak HF energy ratio above which a fart is classified as wet |
| `WETNESS_HF_CUTOFF` | `1000` | Frequency cutoff in Hz separating dry (low) from wet (high) energy |
| `DEBUG_AUDIO` | `0` | Set to `1` to log top-5 YAMNet class scores per window |
| `RTSP_URL` | `rtsp://localhost:8554/baby` | Audio source |
| `MODEL_PATH` | `~/monitor/models/yamnet.tflite` | Path to the YAMNet TFLite model file |

## Files

| File | Purpose |
|---|---|
| `sound_detection.py` | Main detector — audio pipeline, ML inference, wetness analysis, notifications |
| `detect.sh` | Shell wrapper used by systemd to launch the Python script inside the venv |
| `test_cry.py` | Offline unit test — loads the model, runs inference on synthetic audio, asserts output shape and score range |
| `requirements.txt` | Python dependencies (`numpy`, `ai-edge-litert`) |

## Running locally / testing

```bash
# Run unit tests on the Pi (via justfile)
just test

# Or directly on the Pi
~/monitor/venv/bin/python3 ~/monitor/services/detect/test_cry.py

# Enable verbose class score logging
DEBUG_AUDIO=1 ~/monitor/venv/bin/python3 ~/monitor/services/detect/sound_detection.py
```

## Model

YAMNet TFLite (`yamnet.tflite`) and its class map (`yamnet_class_map.csv`) live in `~/monitor/models/` on the Pi and are downloaded during `just install`. The model expects 16kHz mono float32 audio and outputs scores for 521 AudioSet classes.

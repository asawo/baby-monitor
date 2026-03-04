#!/usr/bin/env python3
"""
Baby cry detection using YAMNet TFLite model.

Reads audio from the existing RTSP stream (decoded by FFmpeg), classifies
each ~0.975s window with YAMNet, and sends a push notification via ntfy.sh
when a baby cry is detected.

Environment variables:
  NTFY_TOPIC        (required) Unique ntfy.sh topic
  NTFY_URL          (default: https://ntfy.sh) ntfy server base URL
  CRY_THRESHOLD     (default: 0.3) Confidence score threshold (0.0–1.0)
  NOTIFY_COOLDOWN   (default: 120) Minimum seconds between notifications
  RTSP_URL          (default: rtsp://localhost:8554/baby) Audio source
  MODEL_PATH        (default: ~/monitor/models/yamnet.tflite)
"""

import os
import sys
import time
import subprocess
import urllib.request
import urllib.error
import numpy as np

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
NTFY_TOPIC = os.environ.get("NTFY_TOPIC", "")
NTFY_URL = os.environ.get("NTFY_URL", "https://ntfy.sh").rstrip("/")
THRESHOLD = float(os.environ.get("CRY_THRESHOLD", "0.3"))
COOLDOWN = int(os.environ.get("NOTIFY_COOLDOWN", "120"))
RTSP_URL = os.environ.get("RTSP_URL", "rtsp://localhost:8554/baby")

_default_model = os.path.join(os.path.dirname(__file__), "..", "models", "yamnet.tflite")
MODEL_PATH = os.path.abspath(os.environ.get("MODEL_PATH", _default_model))

# YAMNet expects 16kHz mono; window is exactly 0.975s = 15600 samples
SAMPLE_RATE = 16000
WINDOW_SAMPLES = 15600
BYTES_PER_SAMPLE = 2  # int16

# YAMNet class 20 = "Baby cry, infant cry" (AudioSet ontology)
BABY_CRY_CLASS = 20

# ---------------------------------------------------------------------------
# Model loading
# ---------------------------------------------------------------------------
def load_model(path):
    try:
        import tflite_runtime.interpreter as tflite
    except ImportError:
        import tensorflow.lite as tflite

    if not os.path.exists(path):
        print(f"ERROR: Model not found at {path}", file=sys.stderr)
        print("Run the installer or: wget -O models/yamnet.tflite <model_url>", file=sys.stderr)
        sys.exit(1)

    interpreter = tflite.Interpreter(model_path=path)
    interpreter.allocate_tensors()
    return interpreter


def classify(interpreter, window_int16):
    """Run YAMNet on a 15600-sample int16 window. Returns mean scores array."""
    audio = window_int16.astype(np.float32) / 32768.0
    input_details = interpreter.get_input_details()
    output_details = interpreter.get_output_details()
    interpreter.set_tensor(input_details[0]["index"], audio)
    interpreter.invoke()
    scores = interpreter.get_tensor(output_details[0]["index"])  # shape: (frames, 521)
    return np.mean(scores, axis=0)  # shape: (521,)


# ---------------------------------------------------------------------------
# Notification
# ---------------------------------------------------------------------------
_last_notification = 0.0


def notify(message):
    global _last_notification
    now = time.time()

    if now - _last_notification < COOLDOWN:
        remaining = int(COOLDOWN - (now - _last_notification))
        print(f"  (cooldown: {remaining}s remaining, skipping notification)")
        return

    if not NTFY_TOPIC:
        print("WARNING: NTFY_TOPIC not set — skipping notification (set it in .env)")
        return

    url = f"{NTFY_URL}/{NTFY_TOPIC}"
    req = urllib.request.Request(url, data=message.encode("utf-8"), method="POST")
    req.add_header("Title", "Baby Monitor")
    req.add_header("Priority", "high")
    req.add_header("Tags", "baby,rotating_light")

    try:
        with urllib.request.urlopen(req, timeout=5):
            pass
        _last_notification = now
        print(f"  Notification sent to ntfy.sh/{NTFY_TOPIC}")
    except urllib.error.URLError as e:
        print(f"  Notification failed: {e}", file=sys.stderr)


# ---------------------------------------------------------------------------
# Audio source
# ---------------------------------------------------------------------------
def open_ffmpeg_stream(rtsp_url):
    """Spawn FFmpeg to decode audio from RTSP as raw 16kHz mono PCM on stdout."""
    cmd = [
        "ffmpeg",
        "-hide_banner", "-loglevel", "error",
        "-i", rtsp_url,
        "-vn",                         # drop video
        "-f", "s16le",                 # raw signed 16-bit little-endian PCM
        "-ar", str(SAMPLE_RATE),       # resample to 16kHz
        "-ac", "1",                    # mono
        "pipe:1",                      # write to stdout
    ]
    return subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------
def main():
    print(f"Loading model: {MODEL_PATH}")
    interpreter = load_model(MODEL_PATH)
    print(f"Model loaded.")
    print(f"Starting cry detection (threshold={THRESHOLD}, cooldown={COOLDOWN}s)")
    print(f"Audio source: {RTSP_URL}")
    if not NTFY_TOPIC:
        print("WARNING: NTFY_TOPIC is not set — detections will be logged but not sent")
    print("Listening...")

    chunk_bytes = WINDOW_SAMPLES * BYTES_PER_SAMPLE

    while True:
        proc = open_ffmpeg_stream(RTSP_URL)
        try:
            while True:
                data = proc.stdout.read(chunk_bytes)
                if len(data) < chunk_bytes:
                    # Stream ended or FFmpeg exited
                    break

                window = np.frombuffer(data, dtype=np.int16)
                scores = classify(interpreter, window)
                cry_score = float(scores[BABY_CRY_CLASS])

                if cry_score >= THRESHOLD:
                    print(f"Cry detected! Score: {cry_score:.3f}")
                    notify(f"Baby is crying! (confidence: {cry_score:.0%})")

        except KeyboardInterrupt:
            proc.terminate()
            sys.exit(0)
        except Exception as e:
            print(f"Stream error: {e}", file=sys.stderr)
        finally:
            proc.terminate()
            proc.wait()

        print("Stream disconnected, retrying in 5s...")
        time.sleep(5)


if __name__ == "__main__":
    main()

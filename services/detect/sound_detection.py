#!/usr/bin/env python3
"""
Baby sound detection using YAMNet TFLite model.

Detects baby cries and farts/poops from the RTSP audio stream. Classifies
each ~0.975s window with YAMNet and applies spectral analysis to distinguish
wet farts (poop) from dry farts. Sends push notifications via ntfy.sh.

Environment variables:
  NTFY_TOPIC        (required) Unique ntfy.sh topic
  NTFY_URL          (default: https://ntfy.sh) ntfy server base URL
  CRY_THRESHOLD     (default: 0.3) Cry confidence score threshold (0.0–1.0)
  CRY_COOLDOWN      (default: 120) Minimum seconds between cry notifications
  FART_THRESHOLD    (default: 0.3) Fart confidence score threshold (0.0–1.0)
  FART_COOLDOWN     (default: 120) Minimum seconds between fart notifications
  WETNESS_THRESHOLD (default: 0.4) High-freq energy ratio to classify as wet/poop
  RTSP_URL          (default: rtsp://localhost:8554/baby) Audio source
  MODEL_PATH        (default: ~/monitor/models/yamnet.tflite)
"""

import json
import os
import sys
import time
import subprocess
import urllib.request
import urllib.error
from datetime import datetime
import numpy as np


def log(*args, **kwargs):
    ts = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    print(f'[{ts}]', *args, **kwargs)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
NTFY_TOPIC = os.environ.get("NTFY_TOPIC", "")
NTFY_URL = os.environ.get("NTFY_URL", "https://ntfy.sh").rstrip("/")
CRY_THRESHOLD = float(os.environ.get("CRY_THRESHOLD", "0.3"))
CRY_COOLDOWN = int(os.environ.get("CRY_COOLDOWN", "120"))
FART_THRESHOLD = float(os.environ.get("FART_THRESHOLD", "0.3"))
FART_COOLDOWN = int(os.environ.get("FART_COOLDOWN", "120"))
WETNESS_THRESHOLD = float(os.environ.get("WETNESS_THRESHOLD", "0.4"))
RTSP_URL = os.environ.get("RTSP_URL", "rtsp://localhost:8554/baby")

_default_model = os.path.join(os.path.dirname(__file__), "..", "..", "models", "yamnet.tflite")
MODEL_PATH = os.path.abspath(os.environ.get("MODEL_PATH", _default_model))

# YAMNet expects 16kHz mono; window is exactly 0.975s = 15600 samples
SAMPLE_RATE = 16000
WINDOW_SAMPLES = 15600
BYTES_PER_SAMPLE = 2  # int16

# YAMNet class indices (AudioSet ontology)
# Verify against yamnet_class_map.csv if model is updated.
BABY_CRY_CLASS = 20   # "Baby cry, infant cry"
FART_CLASS = 369      # "Fart"

# ---------------------------------------------------------------------------
# Model loading
# ---------------------------------------------------------------------------
def load_model(path):
    from ai_edge_litert.interpreter import Interpreter

    interpreter = Interpreter(model_path=path)
    interpreter.allocate_tensors()
    input_idx = interpreter.get_input_details()[0]["index"]
    output_idx = interpreter.get_output_details()[0]["index"]
    return interpreter, input_idx, output_idx


def classify(interpreter, input_idx, output_idx, window_int16):
    """Run YAMNet on a 15600-sample int16 window. Returns mean scores array."""
    audio = window_int16.astype(np.float32) / 32768.0
    interpreter.set_tensor(input_idx, audio)
    interpreter.invoke()
    scores = interpreter.get_tensor(output_idx)  # shape: (frames, 521)
    return np.mean(scores, axis=0)  # shape: (521,)


def compute_wetness(window_int16):
    """
    Returns a wetness score 0.0–1.0 based on high-frequency energy ratio.
    Wet/sloppy sounds (liquid, poop farts) have more broadband high-freq energy.
    Dry farts are more tonal with lower high-freq energy.
    """
    spectrum = np.abs(np.fft.rfft(window_int16.astype(np.float32)))
    total_energy = np.sum(spectrum ** 2) + 1e-10
    # High frequencies = top 40% of spectrum
    hf_start = int(len(spectrum) * 0.6)
    hf_energy = np.sum(spectrum[hf_start:] ** 2)
    return float(hf_energy / total_energy)


# ---------------------------------------------------------------------------
# Server reporting
# ---------------------------------------------------------------------------
GO_SERVER = "http://localhost"


def _post_to_server(endpoint, payload, label):
    try:
        data = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(f"{GO_SERVER}{endpoint}", data=data, method="POST")
        req.add_header("Content-Type", "application/json")
        with urllib.request.urlopen(req, timeout=2):
            pass
    except Exception as e:
        log(f"  (could not report {label} to server: {e})", file=sys.stderr)


def report_detect_status(error_message):
    """Report detector error (or clear it with empty string) to the Go server."""
    _post_to_server("/api/detect-status", {"error": error_message}, "status")


def report_cry(confidence):
    """Report a cry detection to the Go server so the UI can show an indicator."""
    _post_to_server("/api/cry", {"confidence": confidence}, "cry")


def report_fart(confidence, wetness, is_wet):
    """Report a fart detection to the Go server so the UI can show an indicator."""
    _post_to_server("/api/fart", {"confidence": confidence, "wetness": wetness, "wet": is_wet}, "fart")


# ---------------------------------------------------------------------------
# Notifications
# ---------------------------------------------------------------------------
_notifications_enabled = True
_notifications_last_checked = 0.0
NOTIFICATIONS_POLL_INTERVAL = 30  # seconds

_last_cry_notification = 0.0
_last_fart_notification = 0.0


def _refresh_notifications_enabled():
    """Fetch current notifications state from the Go server (cached, polled every 30s)."""
    global _notifications_enabled, _notifications_last_checked
    now = time.time()
    if now - _notifications_last_checked < NOTIFICATIONS_POLL_INTERVAL:
        return
    try:
        with urllib.request.urlopen(f"{GO_SERVER}/api/notifications", timeout=2) as resp:
            _notifications_enabled = json.loads(resp.read()).get("enabled", True)
    except Exception:
        pass  # keep previous cached value
    _notifications_last_checked = now


def _send_ntfy(message, title, tags):
    """Send a push notification via ntfy.sh. Returns True on success."""
    if not NTFY_TOPIC:
        log("WARNING: NTFY_TOPIC not set — skipping notification (set it in .env)")
        return False
    url = f"{NTFY_URL}/{NTFY_TOPIC}"
    req = urllib.request.Request(url, data=message.encode("utf-8"), method="POST")
    req.add_header("Title", title)
    req.add_header("Priority", "high")
    req.add_header("Tags", tags)
    try:
        with urllib.request.urlopen(req, timeout=5):
            pass
        log(f"  Notification sent to ntfy.sh/{NTFY_TOPIC}")
        return True
    except urllib.error.URLError as e:
        log(f"  Notification failed: {e}", file=sys.stderr)
        return False


def notify_cry(message):
    global _last_cry_notification
    now = time.time()
    if now - _last_cry_notification < CRY_COOLDOWN:
        remaining = int(CRY_COOLDOWN - (now - _last_cry_notification))
        log(f"  (cry cooldown: {remaining}s remaining, skipping notification)")
        return
    _refresh_notifications_enabled()
    if not _notifications_enabled:
        log("  (notifications disabled via UI, skipping)")
        return
    if _send_ntfy(message, "Mayday!", "baby,rotating_light"):
        _last_cry_notification = now


def notify_fart(message):
    global _last_fart_notification
    now = time.time()
    if now - _last_fart_notification < FART_COOLDOWN:
        remaining = int(FART_COOLDOWN - (now - _last_fart_notification))
        log(f"  (fart cooldown: {remaining}s remaining, skipping notification)")
        return
    _refresh_notifications_enabled()
    if not _notifications_enabled:
        log("  (notifications disabled via UI, skipping)")
        return
    if _send_ntfy(message, "Poop Alert!", "poop,baby"):
        _last_fart_notification = now


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
    log(f"Loading model: {MODEL_PATH}")
    try:
        interpreter, input_idx, output_idx = load_model(MODEL_PATH)
    except Exception as e:
        msg = f"Model load failed: {e}"
        log(msg, file=sys.stderr)
        report_detect_status(msg)
        sys.exit(1)
    report_detect_status("")  # clear any previous error
    log(f"Model loaded.")
    log(f"Starting sound detection")
    log(f"  cry:  threshold={CRY_THRESHOLD}, cooldown={CRY_COOLDOWN}s")
    log(f"  fart: threshold={FART_THRESHOLD}, cooldown={FART_COOLDOWN}s, wetness_threshold={WETNESS_THRESHOLD}")
    log(f"Audio source: {RTSP_URL}")
    if not NTFY_TOPIC:
        log("WARNING: NTFY_TOPIC is not set — detections will be logged but not sent")
    log("Listening...")

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
                scores = classify(interpreter, input_idx, output_idx, window)

                cry_score = float(scores[BABY_CRY_CLASS])
                if cry_score >= CRY_THRESHOLD:
                    log(f"Cry detected! Score: {cry_score:.3f}")
                    report_cry(cry_score)
                    notify_cry(f"Crying detected (confidence: {cry_score:.0%})")

                fart_score = float(scores[FART_CLASS])
                if fart_score >= FART_THRESHOLD:
                    wetness = compute_wetness(window)
                    is_wet = wetness >= WETNESS_THRESHOLD
                    kind = "Wet fart (poop)" if is_wet else "Dry fart"
                    log(f"{kind} detected! Score: {fart_score:.3f}, Wetness: {wetness:.3f}")
                    report_fart(fart_score, wetness, is_wet)
                    if is_wet:
                        notify_fart(f"Poop detected (confidence: {fart_score:.0%})")

        except KeyboardInterrupt:
            proc.terminate()
            sys.exit(0)
        except Exception as e:
            msg = f"Stream error: {e}"
            log(msg, file=sys.stderr)
            report_detect_status(msg)
        finally:
            proc.terminate()
            proc.wait()

        log("Stream disconnected, retrying in 5s...")
        time.sleep(5)


if __name__ == "__main__":
    main()

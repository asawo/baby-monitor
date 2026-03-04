#!/usr/bin/env python3
"""
Offline unit test for the YAMNet cry detector.

Validates that:
  1. The TFLite model file loads without error
  2. Inference runs on synthetic audio without crashing
  3. Output shape is (N, 521) as expected
  4. Class 20 ("Baby cry, infant cry") exists in the output

Usage:
  python3 scripts/test_cry.py [path/to/yamnet.tflite]
"""

import os
import sys
import numpy as np

MODEL_PATH = sys.argv[1] if len(sys.argv) > 1 else os.path.join(
    os.path.dirname(__file__), "..", "models", "yamnet.tflite"
)
MODEL_PATH = os.path.abspath(MODEL_PATH)

SAMPLE_RATE = 16000
WINDOW_SAMPLES = 15600   # 0.975s at 16kHz — YAMNet's expected input length
BABY_CRY_CLASS = 20

# ---------------------------------------------------------------------------
# Load model
# ---------------------------------------------------------------------------
print(f"Loading model: {MODEL_PATH}")
if not os.path.exists(MODEL_PATH):
    print(f"ERROR: Model not found at {MODEL_PATH}")
    print("Download it first:\n  mkdir -p models && wget -O models/yamnet.tflite <url>")
    sys.exit(1)

try:
    import tflite_runtime.interpreter as tflite
    print("Using tflite_runtime")
except ImportError:
    import tensorflow.lite as tflite
    print("Using tensorflow.lite (tflite_runtime not installed)")

interpreter = tflite.Interpreter(model_path=MODEL_PATH)
interpreter.allocate_tensors()
print("Model loaded OK")

input_details = interpreter.get_input_details()
output_details = interpreter.get_output_details()
print(f"Input shape:  {input_details[0]['shape']}")
print(f"Output shape: {output_details[0]['shape']}")

# ---------------------------------------------------------------------------
# Synthetic audio: 440Hz sine wave (~baby babble-ish tone)
# ---------------------------------------------------------------------------
t = np.linspace(0, WINDOW_SAMPLES / SAMPLE_RATE, WINDOW_SAMPLES)
sine = (np.sin(2 * np.pi * 440 * t) * 32767).astype(np.int16)

audio_float = sine.astype(np.float32) / 32768.0
interpreter.set_tensor(input_details[0]["index"], audio_float)
interpreter.invoke()

scores = interpreter.get_tensor(output_details[0]["index"])  # (frames, 521)
mean_scores = np.mean(scores, axis=0)                         # (521,)

# ---------------------------------------------------------------------------
# Assertions
# ---------------------------------------------------------------------------
assert scores.ndim == 2, f"Expected 2D output, got shape {scores.shape}"
assert scores.shape[1] == 521, f"Expected 521 classes, got {scores.shape[1]}"
assert 0.0 <= mean_scores[BABY_CRY_CLASS] <= 1.0, "Baby cry score out of [0,1] range"

print(f"\nInference OK — {scores.shape[0]} frame(s), 521 classes")

# Top-5 detected classes
top5_idx = np.argsort(mean_scores)[::-1][:5]
print("\nTop-5 detected classes (on synthetic 440Hz sine):")
for i, idx in enumerate(top5_idx):
    print(f"  {i+1}. class {idx:3d}: {mean_scores[idx]:.4f}")

print(f"\nClass {BABY_CRY_CLASS} (Baby cry, infant cry): {mean_scores[BABY_CRY_CLASS]:.4f}")
print("\nAll assertions passed.")

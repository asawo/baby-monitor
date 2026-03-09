#!/bin/bash
set -e

exec "${HOME}/monitor/venv/bin/python3" "${HOME}/monitor/services/detect/sound_detection.py"

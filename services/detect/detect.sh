#!/bin/bash
set -e

exec "${HOME}/monitor/venv/bin/python3" "${HOME}/monitor/services/detect/detect_cry.py"

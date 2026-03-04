#!/bin/bash
set -e

exec "${HOME}/monitor/venv/bin/python3" "${HOME}/monitor/scripts/detect_cry.py"

#!/usr/bin/env bash
# tapes/demo.tape — Automated demo recording for parkranger
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/../lib/tape.sh"

tape_init "parkranger-demo" 120 35

# Build parkranger first
tape_run "go build -o parkranger ./cmd/parkranger"
tape_wait_for "\\$" 15
tape_sleep 0.5

# Launch parkranger
tape_run "./parkranger"
tape_sleep 2

# Let the dashboard render
tape_wait_for "parkranger" 5
tape_sleep 3

# Exit the app
tape_key q
tape_sleep 0.5

tape_finish

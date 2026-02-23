#!/usr/bin/env bash
# tapes/demo.tape — Automated walkthrough of parkranger
#
# Demonstrates:
#   1. parkranger help   — show available commands
#   2. parkranger ls     — list worktrees with live status
#   3. parkranger        — interactive picker with j/k navigation
#
# Prerequisites: go toolchain, tmux, asciinema
# Run: ./tapes/demo.tape
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$SCRIPT_DIR/.."
source "$PROJECT_DIR/lib/tape.sh"

# ── Build binary before recording ────────────────────────────
echo "Building parkranger..."
(cd "$PROJECT_DIR" && go build -o parkranger ./cmd/parkranger)

# ── Start recording ──────────────────────────────────────────
tape_init "parkranger-demo" 120 35

# Set up: cd to the project so paths look clean, then clear
tape_run "cd '$PROJECT_DIR'"
tape_wait_for "\$" 3
tape_run "clear"
tape_sleep 0.5

# ── 1. Show help ─────────────────────────────────────────────
tape_type "./parkranger help" 0.06
tape_key Enter
tape_wait_for "delete" 5
tape_sleep 3

# ── 2. List worktrees with status ────────────────────────────
tape_type "./parkranger ls" 0.06
tape_key Enter
tape_wait_for "\$" 5
tape_sleep 3

# ── 3. Interactive picker ────────────────────────────────────
tape_type "./parkranger" 0.06
tape_key Enter
tape_sleep 2

# Browse through worktree items
tape_key j
tape_sleep 0.8
tape_key j
tape_sleep 0.8
tape_key j
tape_sleep 0.8

# Scroll back up
tape_key k
tape_sleep 0.8
tape_key k
tape_sleep 0.8

# Pause on first item to show the picker
tape_sleep 2

# Exit without selecting
tape_key C-c
tape_sleep 1

# ── Done ─────────────────────────────────────────────────────
tape_finish

echo ""
echo "Next steps:"
echo "  Clean:   ./scripts/clean-cast.sh casts/parkranger-demo.cast --inplace"
echo "  MP4:     ./scripts/cast-to-mp4.sh casts/parkranger-demo.cast -o casts/parkranger-demo.mp4"
echo "  GIF:     agg casts/parkranger-demo.cast casts/parkranger-demo.gif"

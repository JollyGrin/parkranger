#!/usr/bin/env bash
# tapes/demo.tape — Automated walkthrough of parkranger
#
# Demonstrates:
#   1. parkranger help     — available commands
#   2. parkranger ls       — worktrees with live session status
#   3. parkranger          — interactive picker (navigate branches)
#   4. Ctrl-b w            — tmux tree showing parkranger-managed windows
#
# Prerequisites: go, tmux, asciinema, existing parkranger sessions
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

tape_run "cd '$PROJECT_DIR'"
tape_wait_for "\$" 3
tape_run "clear"
tape_sleep 0.5

# ── 1. Show help ─────────────────────────────────────────────
tape_type "./parkranger help" 0.06
tape_key Enter
tape_wait_for "delete" 5
tape_sleep 3

# ── 2. List worktrees with live status ───────────────────────
tape_type "./parkranger ls" 0.06
tape_key Enter
tape_wait_for "\$" 5
tape_sleep 3

# ── 3. Interactive picker — browse branches ──────────────────
tape_type "./parkranger" 0.06
tape_key Enter
tape_sleep 2

# Navigate through worktree items
tape_key j
tape_sleep 0.8
tape_key j
tape_sleep 0.8
tape_key j
tape_sleep 0.8

# Back up
tape_key k
tape_sleep 0.8
tape_key k
tape_sleep 1.5

# Open the selected branch
tape_key Enter
tape_sleep 2

# ── 4. Show tmux tree (Ctrl-b w) ────────────────────────────
tape_key C-b
tape_sleep 0.3
tape_key w
tape_sleep 2

# Navigate the session/window tree
tape_key Down
tape_sleep 0.5
tape_key Down
tape_sleep 0.5
tape_key Down
tape_sleep 0.5
tape_key Down
tape_sleep 0.5
tape_sleep 2

# Close the tree
tape_key q
tape_sleep 0.5

# ── 5. Ctrl-C a few times and exit ──────────────────────────
tape_key C-c
tape_sleep 0.3
tape_key C-c
tape_sleep 0.3
tape_key C-c
tape_sleep 0.5

tape_type "exit" 0.08
tape_key Enter
tape_sleep 0.5

tape_finish

echo ""
echo "Next steps:"
echo "  Clean:   ./scripts/clean-cast.sh casts/parkranger-demo.cast --inplace"
echo "  MP4:     ./scripts/cast-to-mp4.sh casts/parkranger-demo.cast -o casts/parkranger-demo.mp4"
echo "  GIF:     agg casts/parkranger-demo.cast casts/parkranger-demo.gif"

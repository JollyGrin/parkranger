#!/usr/bin/env bash
# lib/tape.sh — DSL library for automated terminal recordings
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAPE_NAME=""
TAPE_SESSION=""
CAST_FILE=""

# ── tape_init ────────────────────────────────────────────────
# Create tmux session and start asciinema recording.
#   tape_init <name> [cols] [rows] [cast_dir]
tape_init() {
  local name="${1:?tape_init requires a name}"
  local cols="${2:-100}"
  local rows="${3:-30}"
  local cast_dir="${4:-${SCRIPT_ROOT}/casts}"
  TAPE_NAME="$name"
  TAPE_SESSION="tape-${name}-$$"
  CAST_FILE="${cast_dir}/${name}.cast"

  mkdir -p "$cast_dir"

  # Check dependencies
  for cmd in tmux asciinema; do
    if ! command -v "$cmd" &>/dev/null; then
      echo "ERROR: $cmd not found. Install it first." >&2
      exit 1
    fi
  done

  # Clean up stale sessions with this name prefix
  tmux list-sessions -F '#{session_name}' 2>/dev/null \
    | grep "^tape-${name}-" \
    | xargs -I{} tmux kill-session -t {} 2>/dev/null || true

  # Create tmux session
  tmux new-session -d -s "$TAPE_SESSION" -x "$cols" -y "$rows"

  # Trap for cleanup on unexpected exit
  trap '_tape_cleanup' EXIT INT TERM

  # Start asciinema recording inside the tmux session
  tmux send-keys -t "$TAPE_SESSION" \
    "asciinema rec --cols ${cols} --rows ${rows} --overwrite '${CAST_FILE}'" Enter
  sleep 1  # Let asciinema start
}

# ── tape_run ─────────────────────────────────────────────────
# Launch a command inside the recording session.
#   tape_run "./my-app --flag"
tape_run() {
  local command="${1:?tape_run requires a command}"
  tmux send-keys -t "$TAPE_SESSION" "$command" Enter
}

# ── tape_key ─────────────────────────────────────────────────
# Send keystrokes. Supports tmux key names:
#   Enter, Up, Down, Left, Right, Tab, Escape, BTab (Shift-Tab)
#   tape_key Up Up Enter
#   tape_key q
tape_key() {
  for key in "$@"; do
    tmux send-keys -t "$TAPE_SESSION" "$key"
    sleep 0.05
  done
}

# ── tape_type ────────────────────────────────────────────────
# Type text char-by-char with realistic delay.
#   tape_type "hello world" 0.08
tape_type() {
  local text="${1:?tape_type requires text}"
  local delay="${2:-0.05}"
  for (( i=0; i<${#text}; i++ )); do
    local char="${text:$i:1}"
    tmux send-keys -t "$TAPE_SESSION" -l "$char"
    sleep "$delay"
  done
}

# ── tape_sleep ───────────────────────────────────────────────
# Pause for N seconds (visible in recording).
#   tape_sleep 2
tape_sleep() {
  sleep "${1:?tape_sleep requires seconds}"
}

# ── tape_wait_for ────────────────────────────────────────────
# Poll tmux pane content until a pattern appears.
#   tape_wait_for "\\$" 10       # wait for shell prompt
#   tape_wait_for "Done" 30      # wait for a process to finish
tape_wait_for() {
  local pattern="${1:?tape_wait_for requires a pattern}"
  local timeout="${2:-10}"
  local elapsed=0
  while (( elapsed < timeout )); do
    if tmux capture-pane -t "$TAPE_SESSION" -p | grep -q "$pattern"; then
      return 0
    fi
    sleep 0.3
    elapsed=$(( elapsed + 1 ))
  done
  echo "WARN: tape_wait_for '$pattern' timed out after ${timeout}s" >&2
}

# ── tape_finish ──────────────────────────────────────────────
# Stop recording and kill the tmux session.
#   tape_finish
tape_finish() {
  # Stop asciinema (Ctrl-D or exit)
  tmux send-keys -t "$TAPE_SESSION" "exit" Enter
  sleep 1

  # Kill session
  tmux kill-session -t "$TAPE_SESSION" 2>/dev/null || true
  trap - EXIT INT TERM

  echo "Recording saved: $CAST_FILE"
}

# ── _tape_cleanup (internal) ─────────────────────────────────
_tape_cleanup() {
  tmux kill-session -t "$TAPE_SESSION" 2>/dev/null || true
}

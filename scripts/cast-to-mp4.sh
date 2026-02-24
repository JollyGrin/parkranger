#!/usr/bin/env bash
# cast-to-mp4.sh — Convert an asciicast .cast file to MP4 via agg (GIF) + ffmpeg.
#
# Usage:
#   ./scripts/cast-to-mp4.sh demo.cast -o demo.mp4
#   ./scripts/cast-to-mp4.sh demo.cast --font-size 20
set -euo pipefail

# --- Defaults ---
OUTPUT=""
FONT_SIZE=16
COLS=""
ROWS=""

# --- Parse args ---
POSITIONAL=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o)          OUTPUT="$2"; shift 2 ;;
    --font-size) FONT_SIZE="$2"; shift 2 ;;
    --cols)      COLS="$2"; shift 2 ;;
    --rows)      ROWS="$2"; shift 2 ;;
    -*)          echo "Unknown option: $1" >&2; exit 1 ;;
    *)           POSITIONAL+=("$1"); shift ;;
  esac
done

if [[ ${#POSITIONAL[@]} -lt 1 ]]; then
  echo "Usage: $0 <file.cast> [-o output.mp4] [--font-size N] [--cols N] [--rows N]" >&2
  exit 1
fi

INPUT="${POSITIONAL[0]}"

if [[ -z "$OUTPUT" ]]; then
  OUTPUT="${INPUT%.cast}.mp4"
fi

# --- Check dependencies ---
missing=()
if ! command -v agg &>/dev/null; then
  missing+=("agg  — install with: cargo install --git https://github.com/asciinema/agg")
fi
if ! command -v ffmpeg &>/dev/null; then
  missing+=("ffmpeg — install with: brew install ffmpeg")
fi
if [[ ${#missing[@]} -gt 0 ]]; then
  echo "Missing dependencies:" >&2
  for m in "${missing[@]}"; do
    echo "  $m" >&2
  done
  exit 1
fi

# --- Convert .cast → .gif via agg ---
TMP_GIF="$(mktemp "${TMPDIR:-/tmp}/cast-XXXXXX.gif")"
trap 'rm -f "$TMP_GIF"' EXIT

agg_args=("--font-size" "$FONT_SIZE")
[[ -n "$COLS" ]] && agg_args+=("--cols" "$COLS")
[[ -n "$ROWS" ]] && agg_args+=("--rows" "$ROWS")

echo "Running agg: .cast → .gif"
agg "${agg_args[@]}" "$INPUT" "$TMP_GIF"

# --- Convert .gif → .mp4 via ffmpeg ---
echo "Running ffmpeg: .gif → .mp4"
ffmpeg -y -i "$TMP_GIF" -movflags faststart -pix_fmt yuv420p \
  -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" "$OUTPUT" </dev/null

echo "Wrote $OUTPUT"

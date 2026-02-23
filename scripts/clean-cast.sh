#!/usr/bin/env bash
# clean-cast.sh — Clean an asciicast v3 file:
#   - Strip hostname/username from shell prompts
#   - Cap delays to remove long pauses
#
# Usage:
#   ./scripts/clean-cast.sh recording.cast --inplace
#   ./scripts/clean-cast.sh recording.cast --inplace --max-delay 1.0
#   ./scripts/clean-cast.sh recording.cast -o cleaned.cast --strip-prefix "user@host dir"
set -euo pipefail

python3 -c '
import json, sys, argparse, subprocess, os

def detect_prompt_prefix():
    """Auto-detect the shell prompt prefix: user@hostname cwd"""
    user = os.environ.get("USER", "")
    try:
        hostname = subprocess.check_output(["hostname", "-s"], text=True).strip()
    except Exception:
        hostname = ""
    cwd = os.path.basename(os.getcwd())
    if user and hostname:
        return f"{user}@{hostname} {cwd}"
    return ""

def main():
    p = argparse.ArgumentParser(description="Clean an asciicast v3 file")
    p.add_argument("input", help="Path to .cast file")
    p.add_argument("--max-delay", type=float, default=0.5,
                   help="Cap delays at this value in seconds (default: 0.5)")
    p.add_argument("--strip-prefix", default=None,
                   help="Prompt prefix to strip (default: auto-detect from $USER@hostname cwd)")
    p.add_argument("--inplace", action="store_true", help="Overwrite the input file")
    p.add_argument("-o", "--output", help="Write to a different file")
    args = p.parse_args()

    if not args.inplace and not args.output:
        p.error("Specify --inplace or -o OUTPUT")

    prompt_prefix = args.strip_prefix if args.strip_prefix else detect_prompt_prefix()
    if prompt_prefix:
        print(f"Stripping prompt prefix: {prompt_prefix!r}")
    else:
        print("No prompt prefix detected; skipping prompt stripping")

    with open(args.input, "r") as f:
        lines = f.readlines()

    out = []
    # Line 1: JSON header, pass through unchanged
    out.append(lines[0])

    for line in lines[1:]:
        line = line.strip()
        if not line:
            continue
        event = json.loads(line)
        delay, etype, data = event[0], event[1], event[2]

        # Strip prompt prefix
        if prompt_prefix and prompt_prefix in data:
            data = data.replace(prompt_prefix, "")

        # Cap delay
        if delay > args.max_delay:
            delay = args.max_delay

        out.append(json.dumps([delay, etype, data], ensure_ascii=False) + "\n")

    dest = args.input if args.inplace else args.output
    with open(dest, "w") as f:
        f.writelines(out)

    print(f"Wrote {len(out)} lines to {dest}")

main()
' "$@"

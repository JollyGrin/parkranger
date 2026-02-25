#!/usr/bin/env bash
set -euo pipefail

# Resolve repo root from this script's location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$SCRIPT_DIR"
WORKTREES_DIR="$(dirname "$REPO_ROOT")/.worktrees/$(basename "$REPO_ROOT")"

# Discover buildable worktrees
declare -a names=()
declare -A paths=()

# Main repo
if [[ -f "$REPO_ROOT/cmd/parkranger/main.go" ]]; then
    names+=("main")
    paths["main"]="$REPO_ROOT"
fi

# Sibling worktrees
if [[ -d "$WORKTREES_DIR" ]]; then
    for wt in "$WORKTREES_DIR"/*/; do
        [[ -f "${wt}cmd/parkranger/main.go" ]] || continue
        name="$(basename "$wt")"
        names+=("$name")
        paths["$name"]="$wt"
    done
fi

if [[ ${#names[@]} -eq 0 ]]; then
    echo "No buildable worktrees found" >&2
    exit 1
fi

if [[ ${#names[@]} -eq 1 ]]; then
    selected="${names[0]}"
else
    # Picker: fzf if available, else bash select
    if command -v fzf &>/dev/null; then
        selected="$(printf '%s\n' "${names[@]}" | fzf --prompt="worktree> " --height=~10)" || exit 0
    else
        echo "Select worktree:" >&2
        PS3="> "
        select selected in "${names[@]}"; do
            [[ -n "$selected" ]] && break
        done
    fi
fi

src="${paths[$selected]}"
bin="/tmp/parkranger-${selected}"

echo "Building $selected → $bin" >&2
(cd "$src" && go build -o "$bin" ./cmd/parkranger)

exec "$bin" "$@"

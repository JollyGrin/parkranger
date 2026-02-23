# CLAUDE.md - parkranger

## What this project is

parkranger is a Go TUI that manages parallel AI coding sessions across multiple git repos and worktrees using tmux. It is NOT another agent orchestrator -- it is a workspace manager that gives visibility into agent status.

## End goal configuration

The user runs `parkranger`, sees a dashboard of all their active coding sessions across repos, can create new worktree+tmux combos with one action, and at a glance knows which Claude agents need attention. Sessions persist across restarts.

## Key design principles

- **Engine first, TUI second.** The core logic (tmux polling, agent detection, state persistence) lives in `internal/engine/` and must work without a TUI attached. The TUI is a consumer of engine events.
- **Minimal polling overhead.** The engine polls tmux pane output to detect agent state. This must be lightweight -- no spawning shells per-poll, no unbounded buffers. Use `tmux capture-pane` with line limits.
- **No memory leaks.** Long-running process. All goroutines must have cancellation. All channels must be drained or closed. Use `context.Context` everywhere.
- **Single binary.** No runtime dependencies beyond `tmux` and `git` on PATH.
- **Multi-repo by default.** The data model supports multiple repos, each with multiple worktrees, each with one tmux session.

## Architecture

```
cmd/parkranger/main.go    -- CLI entrypoint (cobra or bare flag parsing)
internal/
  engine/                  -- Core state machine, polling loop, event bus
  tmux/                    -- Tmux CLI wrapper (sessions, windows, panes, capture)
  worktree/                -- Git worktree CRUD
  agent/                   -- Claude session detection, status classification
  store/                   -- BoltDB/SQLite persistence for sessions & state
  tui/                     -- Bubble Tea models (dashboard, detail, preview)
  config/                  -- YAML config loading via Viper
```

## Agent status detection

Proven approach (claude-squad + ccmanager both converge here): `tmux capture-pane -p -J -t <target>` every 500ms. Captures visible pane content (~30-50 lines). No subshell spawned inside the target pane.

Detection priority (highest first -- short-circuit on match):

1. **idle override** -- `"⌕ Search…"` (U+2315) present anywhere → always idle, skip everything else
2. **state hold** -- `"ctrl+r to toggle"` present → maintain previous state (history search UI)
3. **waiting** -- pattern match on last 30 lines:
   - `/(do you want|would you like).+\n+[\s\S]*?(?:yes|❯)/i`
   - `"esc to cancel"` (text input mode)
   - `"No, and tell Claude what to do differently"` (full permission prompt, used by claude-squad)
4. **busy** -- two signals:
   - Pattern: `"esc to interrupt"` or `"ctrl+c to interrupt"` (shown while Claude is working)
   - Hash diff: SHA-256 of captured output changed since last poll → streaming
5. **idle** -- default: output stable, no patterns matched

**Debounce state transitions** for 200ms minimum before confirming (prevents flicker during rapid output). ccmanager polls at 100ms with 200ms persistence requirement. claude-squad polls at 500ms with immediate transitions.

**Bonus signals** from bottom 3 lines of output:
- Background tasks: `/(\d+)\s+(?:background\s+task|local\s+agent)/` or `"(running)"`
- Team members: `/@[\w-]+/g` on lines containing `"shift+↑ to expand"`

When a session has multiple Claude panes, report the most actionable state (waiting > error > busy > idle).

## Tmux conventions

- Session names prefixed `pr-` to isolate parkranger-managed sessions
- Default layout: vertical split, left=editor (vim/nvim), right=Claude Code
- Both panes cd'd into the worktree directory
- Sessions created via `tmux new-session -d -s pr-<name> -c <worktree-path>`

## Config format (~/.config/parkranger/config.yaml)

```yaml
repos:
  - path: /path/to/repo
    default_branch: main
poll_interval: 2s
editor: nvim
session_prefix: "pr-"
```

## Tech stack decisions

- **Bubble Tea** for TUI (not tview) -- Elm architecture, async-friendly, proven in claude-squad
- **Lip Gloss** for styling, **Glamour** for markdown rendering in preview pane
- **os/exec wrapping tmux CLI** for tmux interaction (not a library -- keep it simple, tmux CLI is stable)
- **os/exec wrapping git** for worktree ops (go-git doesn't support worktrees well)
- **BoltDB** for state persistence (single-file, no CGo, embedded)
- **Viper** for YAML config

## Session history discovery

Claude Code stores sessions in `~/.claude/projects/[path-with-slashes-replaced-by-dashes]/`. Known issues with worktree session isolation:

- `sessions-index.json` in worktree dirs often references the parent repo's `projectPath`, not the worktree
- Some worktree dirs have no `sessions-index.json` at all (sessions silently lost)
- Related upstream issues: anthropics/claude-code#15776, #27676

**Parkranger's approach:** Don't trust sessions-index.json. Scan JSONL files directly and match to worktrees by the `cwd` field inside each file. Build and cache our own index. Provide built-in session search (replacing the manual `ccfind` bash script pattern).

```
internal/agent/sessions.go:
  - ScanSessions(projectsDir) → []Session
  - MatchToWorktree(session, worktreePath) → bool
  - SearchSessions(keyword, projectFilter) → []SessionMatch
```

## What NOT to build

- Not an agent. parkranger does not send prompts to Claude or any LLM.
- Not a git client. Worktree create/delete only. No staging, committing, merging.
- Not a tmux replacement. It orchestrates tmux, doesn't reimplement it.
- No web UI. Terminal only.
- No plugin system in v1. Keep it monolithic.

## Build & run

```bash
go build -o parkranger ./cmd/parkranger
./parkranger
```

## Testing

- Unit tests for engine, agent detection, tmux output parsing
- Integration tests that create real tmux sessions (skip in CI with `-short`)
- `go test ./...` must pass before any commit

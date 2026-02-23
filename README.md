# parkranger

A Go TUI for managing parallel AI coding sessions across multiple repos, worktrees, and tmux.

## Problem

Working with multiple Claude Code agents across different repos and worktrees involves too much manual ceremony:

1. Create a tmux session per task
2. Create or select a git worktree
3. Open split panes for editor + Claude
4. `ctrl-b w` to juggle between sessions
5. No visibility into which agents need attention
6. Close the manager tool, lose all session state

Existing tools (claude-squad, ccmanager, workmux) each solve pieces of this but none provide a unified multi-repo dashboard with custom pane layouts and persistent state.

## What parkranger does

- **Multi-repo worktree management** -- add repos, create/select worktrees from one place
- **Tmux session orchestration** -- each worktree gets a tmux session with a split-pane layout (editor + Claude Code), prefixed `pr-` for easy filtering
- **Agent status dashboard** -- overview of all sessions showing agent state (busy / waiting / idle). When a session has multiple Claude panes, surfaces the most actionable state
- **Session persistence** -- conversation hashes and session metadata survive restarts so state reattaches to existing worktrees
- **Streaming preview** -- see Claude's response streaming in the dashboard without switching sessions

## Architecture

```
parkranger
├── cmd/            # CLI entrypoint
├── internal/
│   ├── engine/     # Core polling loop, state machine
│   ├── tmux/       # Tmux interaction (sessions, panes, capture)
│   ├── worktree/   # Git worktree operations
│   ├── agent/      # Claude session detection & status parsing
│   ├── store/      # Persistent state (sessions, conversation hashes)
│   └── tui/        # Bubble Tea views (dashboard, detail, preview)
└── docs/
    └── research.md # Prior art & ecosystem research
```

### Engine-first approach

The TUI is a view layer on top of an engine that can run headless. The engine handles:

- Polling tmux sessions for Claude activity at configurable intervals
- Classifying agent state from pane output (busy / waiting for input / idle / error)
- Persisting session metadata so sessions survive restarts
- Emitting events the TUI (or future integrations) can subscribe to

## Tech stack

| Layer | Choice |
|-------|--------|
| Language | Go |
| TUI | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Markdown preview | [Glamour](https://github.com/charmbracelet/glamour) |
| Tmux interaction | `os/exec` wrapping `tmux` CLI (evaluate [gotmux](https://github.com/GianlucaP106/gotmux) later) |
| Git worktrees | `os/exec` wrapping `git worktree` |
| Config | YAML via [Viper](https://github.com/spf13/viper) |
| State | BoltDB or SQLite (pure Go via modernc) |

## Prior art

See [docs/research.md](docs/research.md) for a detailed landscape analysis of 30+ related projects including claude-squad, ccmanager, workmux, overmind, and the full Charm ecosystem.

## Status

Pre-implementation. Engine design phase.

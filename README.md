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

## Prerequisites

- **Go 1.24+** -- to build from source
- **tmux** -- parkranger creates and manages tmux sessions (`brew install tmux` / `apt install tmux`)
- **git** -- worktree operations require git 2.15+ (`git worktree` support)
- **Claude Code** -- the CLI (`claude`) must be on PATH for agent panes

## Sessions in workspaces

Each repo gets one tmux session. Each worktree gets a window inside it with a vertical split: editor on the left, Claude Code on the right.

```
tmux session: pr-myrepo
├── window: dashboard          # parkranger TUI
├── window: feat-auth          # worktree window
│   ├── pane 0 (left):  nvim   # editor, cd'd into worktree
│   └── pane 1 (right): claude  # Claude Code, cd'd into worktree
└── window: fix-login          # another worktree window
    ├── pane 0 (left):  nvim
    └── pane 1 (right): claude
```

On disk, worktrees live as siblings to the repo:

```
~/git/
├── myrepo/                           # main checkout
└── .worktrees/
    └── myrepo/
        ├── feat-auth/                # worktree (own branch)
        └── fix-login/                # worktree (own branch)
```

Claude Code sessions are stored per-worktree path in `~/.claude/projects/`:

```
~/.claude/projects/
├── -Users-me-git-myrepo/                          # main repo sessions
│   ├── a1b2c3.jsonl
│   └── d4e5f6.jsonl
└── -Users-me-git--worktrees-myrepo-feat-auth/     # worktree sessions
    ├── g7h8i9.jsonl
    └── j0k1l2.jsonl
```

The dashboard polls each Claude pane (`tmux capture-pane`) to classify agent status:

| Status      | Meaning            | How detected                                              |
| ----------- | ------------------ | --------------------------------------------------------- |
| **waiting** | Claude needs input | Permission prompts, yes/no questions, text input mode     |
| **busy**    | Claude is working  | `"esc to interrupt"` present, or pane output hash changed |
| **idle**    | Nothing happening  | Output stable, no active patterns                         |

When opening a worktree, parkranger shows the live session (if the window exists) plus historical Claude sessions from the JSONL files. Resuming a session uses `claude --resume <session-id>`.

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

| Layer            | Choice                                                                                                                                                                  |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Language         | Go                                                                                                                                                                      |
| TUI              | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Markdown preview | [Glamour](https://github.com/charmbracelet/glamour)                                                                                                                     |
| Tmux interaction | `os/exec` wrapping `tmux` CLI (evaluate [gotmux](https://github.com/GianlucaP106/gotmux) later)                                                                         |
| Git worktrees    | `os/exec` wrapping `git worktree`                                                                                                                                       |
| Config           | YAML via [Viper](https://github.com/spf13/viper)                                                                                                                        |
| State            | BoltDB or SQLite (pure Go via modernc)                                                                                                                                  |

## Prior art

See [docs/research.md](docs/research.md) for a detailed landscape analysis of 30+ related projects including claude-squad, ccmanager, workmux, overmind, and the full Charm ecosystem.

## Status

Pre-implementation. Engine design phase.

## Record ascii

Structure:
lib/tape.sh — DSL library (source this from tape
scripts)
scripts/clean-cast.sh — Strip hostnames + cap delays in .cast
files
scripts/cast-to-mp4.sh — Convert .cast → GIF → MP4 via agg +
ffmpeg
tapes/demo.tape — Example automated recording for
parkranger
casts/ — Output directory for recordings
(gitignored)

Workflow:

1. Record — run a tape: ./tapes/demo.tape
2. Clean — strip personal info: ./scripts/clean-cast.sh
   casts/parkranger-demo.cast --inplace
3. Convert — to MP4: ./scripts/cast-to-mp4.sh
   casts/parkranger-demo.cast -o demo.mp4

The tape DSL gives you tape_run, tape_type, tape_key,
tape_wait_for, and tape_sleep to script repeatable demos. The
example tape in tapes/demo.tape builds parkranger, launches it,
waits for the dashboard, then exits — customize it for your
actual demo flow.

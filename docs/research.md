# Parkranger - Prior Art & Ecosystem Research

> Research conducted 2026-02-23. Star counts are approximate and may have changed since.

## Table of Contents

1. [Claude Code / AI Agent Session Managers](#1-claude-code--ai-agent-session-managers)
2. [Git Worktree Managers (TUI)](#2-git-worktree-managers-tui)
3. [Tmux Session Managers](#3-tmux-session-managers)
4. [Go TUI Frameworks](#4-go-tui-frameworks)
5. [Tmux Go Libraries](#5-tmux-go-libraries)
6. [Composite Tools (Worktree + Tmux + Agents)](#6-composite-tools-worktree--tmux--agents)
7. [Key Takeaways for Parkranger](#7-key-takeaways-for-parkranger)

---

## 1. Claude Code / AI Agent Session Managers

These tools directly address the problem of running and monitoring multiple AI coding agents in parallel.

### claude-squad

- **GitHub:** [smtg-ai/claude-squad](https://github.com/smtg-ai/claude-squad)
- **Stars:** ~5,600
- **Language:** Go
- **Actively maintained:** Yes (very active, frequent releases)
- **Description:** Terminal app that manages multiple AI coding agents (Claude Code, Aider, Codex, OpenCode, Amp) in separate workspaces. Uses tmux for isolated terminal sessions and git worktrees to isolate codebases per session.
- **Key Features:**
  - TUI dashboard listing all active agent instances
  - Each instance gets its own tmux session + git worktree + branch
  - Auto-accept mode (`-y` / `--autoyes`) for unattended operation
  - Agent status monitoring (busy/idle/waiting for input)
  - Preview of agent output from the dashboard
  - Session persistence -- sessions survive restarts
  - Built with Go, uses bubbletea for the TUI
- **Relevance to Parkranger:** This is the **closest existing project** to what parkranger aims to be. It already combines tmux sessions, git worktrees, agent monitoring, and a Go TUI. Parkranger could differentiate by offering a richer dashboard, multi-repo management, custom pane layouts (editor + agent side-by-side), and deeper tmux integration for split-pane workflows.

### ccmanager

- **GitHub:** [kbwo/ccmanager](https://github.com/kbwo/ccmanager)
- **Stars:** ~810
- **Language:** TypeScript
- **Actively maintained:** Yes
- **Description:** CLI application for managing multiple AI coding assistant sessions (Claude Code, Gemini CLI, Codex CLI, Cursor Agent, Copilot CLI, Cline CLI, OpenCode, Kimi CLI) across git worktrees and projects.
- **Key Features:**
  - Session data copying when creating new worktrees (preserves conversation history)
  - Status hooks -- execute custom commands when agent status changes (notifications, logging)
  - Devcontainer support -- run sessions inside devcontainers
  - Teammate mode for Claude Code integration
  - Worktree hooks -- run custom commands on worktree creation
  - Multi-agent support (widest agent compatibility of any tool)
- **Relevance to Parkranger:** The status hooks concept is excellent -- allowing arbitrary automation when agent state changes. The session data copying across worktrees is also a feature worth considering. Written in TypeScript, so not directly reusable, but the architecture and feature design are worth studying.

### dmux ★ UI INSPIRATION

- **GitHub:** [standardagents/dmux](https://github.com/standardagents/dmux)
- **Stars:** ~729
- **Language:** TypeScript (React TUI via [Ink](https://github.com/vadimdemedes/ink))
- **Actively maintained:** Yes
- **Description:** Dev agent multiplexer for git worktrees and Claude Code (or other agents). Creates a tmux pane for each task with its own worktree. Best-in-class sidebar UI for agent session management.
- **Key Features:**
  - Press `n` to create a new pane, type a prompt, pick an agent
  - Left sidebar with pane cards: status icon + task slug + agent type (`[cc]`, `[oc]`)
  - Status icons: `*` working (blue), `~` analyzing (yellow), `!` waiting (red), `o` idle (dim)
  - `j`/`k` nav, `Enter` to focus tmux pane, `n` new, `x` close, `m` merge
  - Auto-calculated grid layout for tmux panes (min 60, max 120 char width)
  - A/B launches -- run two agents on the same prompt side-by-side
  - Smart merging with auto-commit and cleanup
  - Lifecycle hooks (worktree create, pre-merge, post-merge)
  - Two-tier status detection: deterministic pattern matching + LLM fallback via OpenRouter
  - Autopilot mode: auto-accepts low-risk prompts based on LLM risk assessment
- **Limitations (gaps parkranger fills):**
  - Only manages worktrees it creates (`<repo>/.dmux/worktrees/`) -- no discovery of pre-existing worktrees
  - No discovery of existing Claude Code sessions started outside dmux
  - Requires OpenRouter API key for idle vs waiting detection
  - Worktrees placed inside the repo (file watcher/IDE issues -- see deep-dive.md §2b)
  - No session persistence across restarts
  - No Claude Code session history browsing or resume
- **Relevance to Parkranger:** Best UI reference. The sidebar card layout, status iconography, and keyboard shortcuts are the UX target. The two-tier status detection is clever but parkranger should achieve idle/waiting/busy purely from pane output patterns (no external LLM dependency). See `docs/deep-dive.md` §4 for full analysis.

### amux

- **GitHub:** [andyrewlee/amux](https://github.com/andyrewlee/amux)
- **Stars:** ~200+
- **Language:** Go
- **Actively maintained:** Yes
- **Description:** TUI for running multiple coding agents in parallel with a workspace-first model.
- **Key Features:**
  - Each agent runs in its own tmux session for isolation and persistence
  - Workspace-first model that can import git worktrees
  - Prompt enqueuing and job status polling
  - Configurable timeouts and intervals
  - Structured logging (~/.amux/logs/)
- **Relevance to Parkranger:** Another Go-based tool in this space. The job queue / prompt enqueue pattern is interesting for batch workflows.

### Coder Mux

- **GitHub:** [coder/mux](https://github.com/coder/mux)
- **Stars:** ~300+
- **Language:** TypeScript (Electron/Tauri)
- **Actively maintained:** Yes (backed by Coder, Inc.)
- **Description:** Desktop and browser application for parallel agentic development. Runs on local or remote compute.
- **Key Features:**
  - Spec-driven development -- agents write detailed plan files for review before coding
  - Sub-agent delegation -- spawn background agents to explore codebases
  - Runs on your infrastructure, fully open-source (AGPL)
  - Desktop app (not terminal-based)
- **Relevance to Parkranger:** Different approach (desktop app vs TUI) but the spec-driven development and sub-agent delegation patterns are architecturally interesting.

### cc-sessions

- **GitHub:** [GWUDCAP/cc-sessions](https://github.com/GWUDCAP/cc-sessions)
- **Stars:** Small
- **Language:** Shell
- **Description:** An opinionated approach to productive development with Claude Code. Focuses on session management conventions rather than a full TUI.
- **Relevance to Parkranger:** Useful for understanding session management patterns and conventions.

---

## 2. Git Worktree Managers (TUI)

### lazyworktree

- **GitHub:** [chmouel/lazyworktree](https://github.com/chmouel/lazyworktree)
- **Stars:** ~100+
- **Language:** Go (BubbleTea)
- **Actively maintained:** Yes (migrated from Python/Textual to Go/BubbleTea)
- **Description:** TUI for git worktrees with a keyboard-driven workflow. The most feature-rich standalone worktree TUI available.
- **Key Features:**
  - Create, rename, remove, absorb, and prune merged worktrees
  - View CI logs from GitHub Actions
  - Display linked PR/MR, CI status, and checks
  - Stage, unstage, commit, edit, and diff files
  - Optional delta integration for diffs
  - Per-worktree tmux/zellij session management
  - Cherry-pick commits between worktrees
  - Custom commands with keybindings
  - GitHub/GitLab/Gitea forge integration
- **Relevance to Parkranger:** **Highly relevant.** Written in Go with BubbleTea, tackles the worktree management problem with the deepest feature set. The forge integration (CI status, PR links) is a feature parkranger could incorporate. Could potentially be used as a library or reference implementation.

### forestui

- **GitHub:** [flipbit03/forestui](https://github.com/flipbit03/forestui)
- **Stars:** Small
- **Language:** Python 3.14+ (Textual 7.2+, Pydantic, Click)
- **Actively maintained:** Yes (newer project)
- **Description:** TUI for managing git worktrees with Claude Code integration. Sidebar tree of repos/worktrees, detail panel with action buttons, session history list with Resume/YOLO options.
- **Key Features:**
  - Sidebar tree view: repos as parent nodes, worktrees nested with branch names
  - Detail panel: worktree info, quick-action buttons (editor, terminal, file manager, claude)
  - Claude session history: reads JSONL files from `~/.claude/projects/`, shows last 5 sessions with title/timestamp/message count
  - Resume (`-r <id>`) and YOLO (`--dangerously-skip-permissions`) modes per session
  - GitHub issues display with "Create Worktree from Issue" button
  - Separate tmux windows per tool: `edit:<name>`, `claude:<name>`, `term:<name>`
- **Limitations (confirmed via source code analysis):**
  - **No agent status detection.** Zero idle/busy/waiting indicators. Only reads historical JSONL session files, no pane capture or real-time monitoring.
  - **No state persistence across restarts.** Forest directories are CLI args, not saved.
  - **Separate windows, not split panes.** Each tool gets its own tmux window rather than a side-by-side layout.
  - **Single forest at a time.** No unified multi-repo dashboard.
- **Relevance to Parkranger:** Closest UX inspiration -- the sidebar+detail layout, session resume buttons, and GitHub issues integration are all patterns parkranger should adopt. But parkranger fills the critical gaps: agent status detection, split-pane layouts, multi-repo persistence, and reliable session discovery across worktrees. See [docs/deep-dive.md](deep-dive.md) for full source code analysis.

### claude-worktree (cwt)

- **GitHub:** [bucket-robotics/claude-worktree](https://github.com/bucket-robotics/claude-worktree)
- **Stars:** Small
- **Language:** Ruby (ratatui-ruby)
- **Description:** Simple TUI built on the premise that git worktrees are the best way to isolate AI coding sessions. Uses thread pool for git operations so the UI stays responsive.
- **Relevance to Parkranger:** The async git operations pattern is worth noting for UI responsiveness.

### branchlet

- **URL:** [terminaltrove.com/branchlet](https://terminaltrove.com/branchlet/)
- **Language:** Go
- **Description:** Interactive TUI for managing git worktrees with create, list, and delete operations.
- **Relevance to Parkranger:** Simpler Go implementation for worktree management.

---

## 3. Tmux Session Managers

### smug

- **GitHub:** [ivaaaan/smug](https://github.com/ivaaaan/smug)
- **Stars:** ~715
- **Language:** Go
- **Actively maintained:** Yes (latest release Dec 2024)
- **Description:** Session manager and task runner for tmux. Configuration-driven setup of windows and panes via YAML files.
- **Key Features:**
  - YAML configuration for defining windows, panes, and commands
  - Support for custom variables passed via CLI arguments
  - Start/stop/list/edit/new project commands
  - `.smug.yml` file in project directory or `~/.config/smug/`
  - Inspired by tmuxinator and tmuxp
- **Relevance to Parkranger:** Good reference for YAML-driven tmux session configuration in Go. The configuration schema could inform parkranger's session definition format.

### sesh

- **GitHub:** [joshmedeski/sesh](https://github.com/joshmedeski/sesh)
- **Stars:** ~1,600
- **Language:** Go
- **Actively maintained:** Yes
- **Description:** Smart session manager for the terminal, built with Go. Successor to t-smart-tmux-session-manager. Multiplexer-agnostic (tmux, zellij, WezTerm).
- **Key Features:**
  - Zoxide integration for directory-based session discovery
  - Auto-create sessions on connect
  - Custom configurations and startup scripts
  - Raycast extension for GUI access
  - Fast compiled Go binary
- **Relevance to Parkranger:** The multiplexer-agnostic design is interesting if parkranger ever needs to support zellij/WezTerm. Zoxide integration for smart directory matching is a nice UX touch.

### tmux-sessionx

- **GitHub:** [omerxx/tmux-sessionx](https://github.com/omerxx/tmux-sessionx)
- **Stars:** ~913
- **Language:** Shell (tmux plugin)
- **Actively maintained:** Yes
- **Description:** Tmux plugin providing a session manager with preview, fuzzy finding via fzf, and tmuxinator integration.
- **Key Features:**
  - fzf-tmux popup with fuzzy search over sessions
  - Create new sessions by typing non-existing names
  - Tmuxinator project integration
  - fzf-marks integration
  - Custom paths always visible in results
- **Relevance to Parkranger:** The fuzzy-finding session selection UX is worth incorporating.

### tmuxinator

- **GitHub:** [tmuxinator/tmuxinator](https://github.com/tmuxinator/tmuxinator)
- **Stars:** ~13,400
- **Language:** Ruby
- **Actively maintained:** Yes (mature project)
- **Description:** The original tmux session manager. Manages complex tmux sessions via YAML configuration.
- **Relevance to Parkranger:** The gold standard for tmux session configuration. Its YAML schema has influenced every subsequent tool (smug, tmuxp, etc.). Worth studying the configuration format.

### tmuxp

- **GitHub:** [tmux-python/tmuxp](https://github.com/tmux-python/tmuxp)
- **Stars:** ~4,300
- **Language:** Python
- **Actively maintained:** Yes
- **Description:** Session manager for tmux built on libtmux. Load sessions via JSON/YAML, tmuxinator/teamocil format compatible.
- **Relevance to Parkranger:** Built on libtmux (a Python tmux API wrapper). The architectural separation of "tmux library" from "session manager" is a pattern parkranger should follow.

### overmind

- **GitHub:** [DarthSim/overmind](https://github.com/DarthSim/overmind)
- **Stars:** ~3,500
- **Language:** Go
- **Actively maintained:** Yes
- **Description:** Process manager for Procfile-based applications using tmux. Each process runs in its own tmux pane, allowing individual process control, restart, and monitoring.
- **Key Features:**
  - Processes in tmux sessions -- connect and interact with any process
  - Restart individual processes without restarting the stack
  - Let processes die without killing others
  - Auto-restart specified processes on death
  - Uses tmux control mode for clean output capture
  - Environment variable management via `.overmind.env`
- **Relevance to Parkranger:** **Very relevant architecture.** Overmind's approach of using tmux as an execution substrate for managed processes is exactly the pattern parkranger needs. The control mode output capture is particularly relevant for monitoring Claude agent output. Written in Go, so the tmux interaction patterns are directly reusable.

---

## 4. Go TUI Frameworks

### Bubble Tea (charmbracelet/bubbletea)

- **GitHub:** [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- **Stars:** ~39,300
- **Language:** Go
- **Actively maintained:** Yes (very active, Charm is a funded company)
- **Description:** The dominant Go TUI framework. Based on The Elm Architecture (TEA) with a functional, message-passing design.
- **Key Features:**
  - Model-Update-View architecture (functional, predictable state management)
  - Inline, full-window, or mixed rendering modes
  - Built-in mouse support
  - Extensive ecosystem of components (Bubbles)
  - Production-tested in numerous projects (Glow, claude-squad, etc.)
  - Async command support via `tea.Cmd`
  - Sub-model composition for complex UIs
- **Relevance to Parkranger:** **This is the recommended framework.** claude-squad already uses it, it has the largest ecosystem, and its Elm-style architecture is well-suited for a dashboard with multiple async data sources (agent status, tmux state, git status). The functional architecture makes testing straightforward.

### Bubbles (charmbracelet/bubbles)

- **GitHub:** [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- **Stars:** ~7,800
- **Language:** Go
- **Description:** Official component library for Bubble Tea. Provides ready-made, composable TUI widgets.
- **Key Components:**
  - **Spinner** -- activity indicators
  - **Text Input** -- single-line input with unicode/paste support
  - **Text Area** -- multi-line input with scrolling
  - **Table** -- tabular data display with selection
  - **List** -- filterable, navigable lists
  - **Viewport** -- scrollable content region (perfect for log/output viewing)
  - **Progress** -- progress bars
  - **File Picker** -- file/directory browser
  - **Paginator** -- page navigation
  - **Timer / Stopwatch** -- time tracking
  - **Help** -- automatic keybinding documentation
- **Relevance to Parkranger:** Essential companion to Bubble Tea. The **viewport** component is key for streaming Claude output preview. The **table** and **list** components are ideal for the session dashboard. The **spinner** is useful for status indicators.

### Lip Gloss (charmbracelet/lipgloss)

- **GitHub:** [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)
- **Stars:** ~10,600
- **Language:** Go
- **Description:** Declarative, CSS-like styling for terminal UIs. Part of the Charm ecosystem.
- **Key Features:**
  - CSS-like style definitions (padding, margin, border, colors)
  - Composable styles
  - Adaptive color profiles (true color, 256 color, ANSI)
  - Built on termenv
- **Relevance to Parkranger:** Required for styling the parkranger TUI. Provides the visual polish layer on top of Bubble Tea components.

### Glamour (charmbracelet/glamour)

- **GitHub:** [charmbracelet/glamour](https://github.com/charmbracelet/glamour)
- **Stars:** ~2,500+
- **Language:** Go
- **Description:** Stylesheet-based markdown rendering for terminal apps. Used by GitHub CLI, GitLab CLI, and Glow.
- **Key Features:**
  - Multiple built-in themes (dark, light, dracula, tokyo-night, etc.)
  - Syntax-highlighted code blocks
  - Tables, lists, headings, links
  - Custom stylesheets
- **Relevance to Parkranger:** **Very useful** for rendering Claude's markdown responses in the preview pane. Claude outputs markdown-formatted text, and glamour can render it beautifully in the terminal.

### Huh (charmbracelet/huh)

- **GitHub:** [charmbracelet/huh](https://github.com/charmbracelet/huh)
- **Stars:** ~5,400
- **Language:** Go
- **Description:** Library for building interactive terminal forms and prompts. Can be used standalone or embedded in Bubble Tea apps.
- **Key Features:**
  - Input, text area, select, multi-select, confirm fields
  - Theming with 5 predefined themes
  - Accessibility mode for screen readers
  - Embeddable in Bubble Tea applications
- **Relevance to Parkranger:** Useful for configuration dialogs, session creation forms, and prompt input within the TUI.

### tview (rivo/tview)

- **GitHub:** [rivo/tview](https://github.com/rivo/tview)
- **Stars:** ~13,400
- **Language:** Go
- **Actively maintained:** Yes
- **Description:** Traditional widget-based terminal UI library with rich, interactive widgets. Built on tcell.
- **Key Features:**
  - Box, List, Form, TextView, TextArea, Table, TreeView widgets
  - Grid and Flexbox layout systems
  - Pages for view switching
  - Modal dialogs
  - Backward compatibility commitment
- **Relevance to Parkranger:** Alternative to Bubble Tea. tview uses a more traditional object-oriented/widget approach (like Qt/GTK) vs. Bubble Tea's functional Elm architecture. **Recommendation: Use Bubble Tea instead** -- it has a larger ecosystem, is more actively developed, and claude-squad already demonstrates it works well for this problem domain.

### Comparison: Bubble Tea vs tview

| Feature | Bubble Tea | tview |
|---------|-----------|-------|
| Architecture | Functional (Elm/TEA) | Object-oriented (widget tree) |
| Stars | ~39k | ~13k |
| Ecosystem | Large (bubbles, lipgloss, glamour, huh) | Self-contained |
| Learning Curve | Steeper (functional paradigm) | Gentler (familiar OOP) |
| Composability | Excellent (sub-models) | Good (widget nesting) |
| Async | Built-in (tea.Cmd) | Manual (goroutines + app.QueueUpdate) |
| Testing | Easy (pure functions) | Harder (stateful widgets) |
| Styling | Lip Gloss (CSS-like) | Built-in (tcell colors) |
| Used by | claude-squad, gh, glow | lazygit (older versions), k9s |

---

## 5. Tmux Go Libraries

### gotmux

- **GitHub:** [GianlucaP106/gotmux](https://github.com/GianlucaP106/gotmux)
- **Stars:** ~34
- **Language:** Go
- **Actively maintained:** Yes
- **Description:** The most comprehensive Go library for programmatic tmux interaction. Type-safe API with full session/window/pane management.
- **Key Features:**
  - Session management (create, list, attach, rename, kill)
  - Window management (create, get by index, select layout, move between sessions)
  - Pane operations (split, send keys, capture output)
  - Server information access
  - Robust error handling
  - Production-ready core (not all tmux features yet)
- **Relevance to Parkranger:** **Primary candidate for tmux integration.** Despite low star count, it provides the cleanest API for the operations parkranger needs. Worth evaluating API completeness for split-pane creation and output capture.

### go-tmux (jubnzv)

- **GitHub:** [jubnzv/go-tmux](https://github.com/jubnzv/go-tmux)
- **Stars:** ~20
- **Language:** Go
- **Description:** Library for managing tmux sessions, windows, and panes. Simpler API than gotmux.
- **Relevance to Parkranger:** Alternative to gotmux with a simpler interface. May be sufficient for basic operations.

### gomux (wricardo)

- **GitHub:** [wricardo/gomux](https://github.com/wricardo/gomux)
- **Stars:** ~9
- **Language:** Go
- **Description:** Go wrapper to create tmux sessions, windows, and panes. Minimal API surface.
- **Relevance to Parkranger:** Simplest option. Good for understanding the minimal tmux CLI wrapper pattern.

### libtmux-go

- **GitHub:** [philipgraf/libtmux-go](https://github.com/philipgraf/libtmux-go)
- **Stars:** Very small
- **Language:** Go
- **Description:** Go port of Python's libtmux. Aims to provide a complete tmux API.
- **Relevance to Parkranger:** If it mirrors libtmux's comprehensive API, it could be useful. Needs maturity evaluation.

### Alternative: Shell out to tmux directly

Many Go projects (including claude-squad and overmind) simply shell out to the `tmux` CLI via `os/exec`. This is pragmatic and avoids library dependencies. The tmux CLI is stable and well-documented. Parkranger could start with direct CLI calls and wrap them in its own internal package, potentially adopting gotmux later if the abstraction proves valuable.

---

## 6. Composite Tools (Worktree + Tmux + Agents)

These projects combine multiple aspects of what parkranger aims to do.

### workmux

- **GitHub:** [raine/workmux](https://github.com/raine/workmux)
- **Stars:** ~643
- **Language:** Rust
- **Actively maintained:** Yes
- **Description:** Zero-friction workflow for git worktrees + tmux windows. Each worktree gets a dedicated, pre-configured tmux window.
- **Key Features:**
  - `add` command creates worktree + tmux window in one step
  - `merge` command merges branch, removes worktree + window + local branch
  - YAML layout configuration (`.workmux.yaml`)
  - Post-creation hooks (install deps, setup DB, etc.)
  - Symlink/copy config files into new worktrees
  - Session mode -- each worktree gets its own tmux session (not just window)
  - AI agent support for parallel development
- **Relevance to Parkranger:** **Very close to parkranger's vision** but in Rust and without a TUI dashboard. The YAML config, lifecycle hooks, and "one command to create everything" workflow are excellent design references.

### treemux

- **URL:** [treemux.com](https://treemux.com/)
- **Language:** Unknown
- **Description:** Pairs git worktrees with tmux sessions. Create, jump, and delete from a TUI.
- **Relevance to Parkranger:** Simpler version of the worktree+tmux concept.

### devx

- **GitHub:** [jfox85/devx](https://github.com/jfox85/devx)
- **Language:** Unknown
- **Description:** Manage parallel development sessions with a terminal UI, git worktrees, and automatic HTTPS hostnames. Each session gets its own git worktree, unique ports, HTTPS via Caddy, and tmux session.
- **Relevance to Parkranger:** The HTTPS hostname and port management per-session is interesting for web development workflows but probably out of scope for parkranger v1.

### twig

- **GitHub:** [andersonkrs/twig](https://github.com/andersonkrs/twig)
- **Language:** Go
- **Description:** A glamorous tmux session manager with git worktree support.
- **Relevance to Parkranger:** Another Go project combining tmux + worktrees.

### Jean

- **GitHub:** [coollabsio/jean](https://github.com/coollabsio/jean)
- **Language:** TypeScript (Tauri + React)
- **Description:** Desktop AI assistant for managing multiple projects, worktrees, and chat sessions with Claude CLI. Built with Tauri v2 and React 19.
- **Relevance to Parkranger:** Desktop app approach (vs TUI). Interesting if parkranger ever considers a GUI mode.

---

## 7. Key Takeaways for Parkranger

### The Landscape

The space of "manage multiple AI agents with worktrees and tmux" has **exploded** in 2025-2026. There are dozens of tools, but most are young, narrowly scoped, or written in languages other than Go. The two most mature projects are:

1. **claude-squad** (Go, ~5.6k stars) -- the market leader, but focused on agent management with worktrees as an implementation detail
2. **ccmanager** (TypeScript, ~810 stars) -- most flexible multi-agent support with status hooks

### Where Parkranger Can Differentiate

Based on gaps observed across all projects:

1. **Multi-repo management** -- Most tools assume a single repo. Parkranger could manage worktrees across multiple repositories from one dashboard.

2. **Custom pane layouts** -- Most tools create simple single-pane sessions. Parkranger's split-pane concept (editor + Claude side-by-side) with configurable layouts is unique.

3. **Rich dashboard** -- claude-squad has a basic list view. A richer dashboard with status indicators, session previews, resource usage, and session grouping would be valuable.

4. **Streaming output preview** -- Using glamour to render Claude's markdown output in real-time within the dashboard would be a standout feature.

5. **Session persistence and state** -- Going beyond simple session restoration to include full state serialization (conversation context, branch state, editor state) across restarts.

6. **Declarative configuration** -- A YAML/TOML-based project configuration (like workmux's `.workmux.yaml` or smug's config) that defines the default layout, repos, and agent configurations.

### Recommended Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Language | Go | Performance, single binary, ecosystem maturity |
| TUI Framework | Bubble Tea | Largest ecosystem, proven for this domain (claude-squad uses it) |
| TUI Components | Bubbles | Viewport for streaming, table for dashboard, list for navigation |
| Styling | Lip Gloss | CSS-like terminal styling, part of Charm ecosystem |
| Markdown Rendering | Glamour | Render Claude's markdown output beautifully |
| Forms/Input | Huh | Session creation dialogs, configuration |
| Tmux Interaction | gotmux or direct CLI | Start with CLI via `os/exec`, consider gotmux for cleaner API |
| Git Operations | go-git or CLI | go-git for programmatic access, CLI for worktree operations |
| Configuration | Viper + YAML | Standard Go config management |
| State Persistence | SQLite (via modernc) or BoltDB | Session state, history, preferences |

### Architecture Lessons from Prior Art

1. **From claude-squad:** The Bubble Tea model works well for this. Use sub-models for different views (dashboard, session detail, preview).

2. **From ccmanager:** Status hooks are powerful. Allow users to define custom actions on agent state transitions.

3. **From workmux:** YAML config per project is the right UX. One command to create the full environment (worktree + tmux + agent).

4. **From overmind:** Using tmux as an execution substrate (not just display) is the right pattern. tmux control mode enables clean output capture.

5. **From smug:** YAML-driven session definition is well-understood. Follow the convention of `~/.config/parkranger/` for global config and `.parkranger.yaml` for per-project config.

6. **From dmux:** UI is the reference target -- sidebar card layout with status icons, `j`/`k` navigation, auto-grid tmux layout. Deterministic status detection should NOT require an external LLM. Lifecycle hooks add flexibility without complexity.

### Reference Repositories to Study

| Priority | Repository | Why |
|----------|-----------|-----|
| 1 | [standardagents/dmux](https://github.com/standardagents/dmux) | **UI inspiration target.** Sidebar card layout, status icons, keyboard shortcuts, grid layout calculator |
| 2 | [smtg-ai/claude-squad](https://github.com/smtg-ai/claude-squad) | Closest competitor, same tech stack (Go + Bubble Tea + tmux + worktrees) |
| 3 | [DarthSim/overmind](https://github.com/DarthSim/overmind) | Best Go reference for tmux process management patterns |
| 4 | [chmouel/lazyworktree](https://github.com/chmouel/lazyworktree) | Best Go/BubbleTea reference for worktree TUI |
| 5 | [raine/workmux](https://github.com/raine/workmux) | Best worktree+tmux workflow design (Rust, but great config/UX) |
| 6 | [ivaaaan/smug](https://github.com/ivaaaan/smug) | Clean Go reference for YAML-driven tmux session management |
| 7 | [kbwo/ccmanager](https://github.com/kbwo/ccmanager) | Status hooks, multi-agent patterns, deterministic status detection |
| 8 | [GianlucaP106/gotmux](https://github.com/GianlucaP106/gotmux) | Go tmux library API design |

### Additional Resource

- [awesome-agent-orchestrators](https://github.com/andyrewlee/awesome-agent-orchestrators) -- Maintained list of all coding agent orchestration tools. Useful for tracking the evolving landscape.
- [awesome-claude-code](https://github.com/hesreallyhim/awesome-claude-code) -- Curated list of Claude Code tools, skills, and integrations.

# UX Flow: parkranger

---

## Phase 0: CLI Mode (MVP)

Minimal interactive CLI. No dashboard, no polling, no persistence layer. Just the workflow shortcut you actually need right now.

### Launch

```
$ cd ~/git/sorcery-tcg-playtest
$ parkranger
```

Detects the repo from CWD. Shows a picker:

```
 sorcery-tcg-playtest (main)

 Worktrees:
   1. task-fix-curiosa        (3 ahead)
   2. task-improve-dragdrop   (1 ahead)
   3. task-update-deckbuilder (5 ahead, dirty)

 [n] New worktree
 [m] Merge a worktree
 [d] Delete a worktree
 [q] Quit

 >
```

That's it. One screen. Detects existing worktrees via `git worktree list`, shows branch status (ahead/behind/dirty) at a glance.

### Select existing → open session

```
> 1

 Opening task-fix-curiosa...
 tmux session: pr-task-fix-curiosa
```

What happens:
1. Checks if `pr-task-fix-curiosa` tmux session already exists
   - **Yes →** attaches to it
   - **No →** creates it:
     - `tmux new-session -d -s pr-task-fix-curiosa -c <worktree-path>`
     - Splits vertical: left = `$EDITOR .`, right = `claude`
     - Both panes cd'd into the worktree
2. Attaches to the session (parkranger exits, you're in tmux)

Back to parkranger later: detach (`ctrl-b d`) and run `parkranger` again.

### New worktree

```
> n

 Branch name: task-new-feature
 Base branch: main ▾

 Creating worktree...
 Created: ~/git/.worktrees/sorcery-tcg-playtest/task-new-feature
 Opening tmux session: pr-task-new-feature
```

What happens:
1. `git worktree add <parent>/.worktrees/<repo>/task-new-feature -b task-new-feature`
2. Creates + attaches tmux session (same as selecting existing)

### Merge

```
> m

 Merge which worktree?
   1. task-fix-curiosa        (3 ahead, clean)
   2. task-improve-dragdrop   (1 ahead, clean)
   3. task-update-deckbuilder (5 ahead, dirty!) ← can't merge

 > 1

 Merge task-fix-curiosa → main?
 [y] Yes  [n] No  [d] Yes + delete worktree + branch

 > d

 Switching to main...
 Merging task-fix-curiosa...
 ✓ Merged (fast-forward)
 Removing worktree...
 ✓ Worktree removed
 Deleting branch...
 ✓ Branch deleted
```

Dirty worktrees are flagged and blocked from merge. Simple fast-forward or merge commit -- no rebase, no squash.

### Delete

```
> d

 Delete which worktree?
   1. task-fix-curiosa
   2. task-improve-dragdrop
   3. task-update-deckbuilder

 > 2

 Delete task-improve-dragdrop?
 This will:
   - Close tmux session pr-task-improve-dragdrop (if running)
   - Remove worktree
   - Optionally delete branch

 [y] Yes  [b] Yes + delete branch  [n] No

 > y
 ✓ Done
```

### CLI flags for scripting

```bash
parkranger                     # interactive picker (default)
parkranger open <name>         # skip picker, open directly
parkranger new <name>          # create + open
parkranger ls                  # list worktrees (non-interactive, for scripts)
parkranger merge <name>        # merge flow
parkranger delete <name>       # delete flow
```

### What this does NOT do (saved for Phase 1)

- No agent status detection (no polling, no busy/idle/waiting)
- No streaming preview
- No multi-repo (works from CWD repo only)
- No session history / resume
- No persistent state (no BoltDB)
- No background process

It's just a smart launcher: pick a worktree (or make one), get a tmux session with editor + claude, both in the right directory. Everything else comes later.

### Implementation scope

```
cmd/parkranger/main.go       -- CLI entrypoint, interactive picker
internal/
  worktree/worktree.go        -- git worktree list/add/remove/merge
  tmux/tmux.go                -- session create/attach/split/exists
  git/git.go                  -- branch ahead/behind/dirty checks
```

Three packages. No engine, no store, no TUI framework. Just `os/exec` wrapping git and tmux, and a simple terminal picker (could be [survey](https://github.com/AlecAivazis/survey), [huh](https://github.com/charmbracelet/huh), or raw stdin).

---

## Phase 1: Dashboard Mode (full TUI)

> Replaces: `ccmanager (create worktree) → tmux new → workmux ls → workmux open <task> → ccmanager merge`

## Launch

```
$ parkranger
```

Opens the dashboard. If repos are already configured, shows them immediately. If first run, prompts to add a repo.

## Dashboard (home screen)

dmux-inspired sidebar layout. Left panel is the parkranger TUI. Right side is the focused tmux pane (editor or Claude).

```
╭─ parkranger ──────────────╮┬─────────────────────────────────────╮
│                            ││                                     │
│  sorcery-tcg-playtest      ││  (focused pane content)             │
│  ├─ ! task-fix-curiosa     ││                                     │
│  ├─ * task-improve-drag    ││  vim / claude / terminal            │
│  └─ o main                 ││                                     │
│                            ││                                     │
│  sorcery-cards             ││                                     │
│  └─ * task-blurhash        ││                                     │
│                            ││                                     │
│  parkranger                ││                                     │
│  └─ o main                 ││                                     │
│                            │├─────────────────────────────────────┤
│                            ││  (streaming preview of selected     │
│  ─────────────────         ││   Claude pane, if not focused)      │
│  [n]ew  [a]dd repo         ││                                     │
│  [d]elete  [m]erge         ││                                     │
│  [s]earch sessions         ││                                     │
│  [?] help                  ││                                     │
╰────────────────────────────╯┴─────────────────────────────────────╯
```

**Status icons:**
- `!` waiting for input (red) -- needs your attention
- `*` busy/working (blue) -- Claude is actively responding
- `~` analyzing (yellow) -- transitional
- `o` idle (dim) -- done or waiting for you to start

**Sorted by actionability:** waiting floats to top within each repo.

## Flow 1: New task (replaces ccmanager + tmux new + workmux open)

```
Dashboard → press [n]

  ┌─ New Task ─────────────────────┐
  │                                │
  │  Repo:  sorcery-tcg-playtest ▾ │
  │  Name:  task-fix-curiosa       │
  │  From:  main ▾                 │
  │                                │
  │  [Enter] Create  [Esc] Cancel  │
  └────────────────────────────────┘
```

One action does all of:
1. `git worktree add <parent>/.worktrees/<repo>/<name> -b <name>` (from selected branch)
2. `tmux new-session -d -s pr-<name> -c <worktree-path>`
3. Split pane: left=editor (`$EDITOR`), right=`claude`
4. Both panes cd'd into the worktree
5. Session metadata saved to BoltDB
6. Dashboard updates, new task appears under the repo

## Flow 2: Open existing worktree (replaces workmux ls + workmux open)

```
Dashboard → j/k to navigate → Enter on a task
```

- If tmux session `pr-<name>` exists → reattach/focus it
- If session was closed → recreate it from persisted metadata (same worktree path, same layout)
- If Claude was running → session history shown, [r]esume button available

No `workmux ls`. The dashboard IS the list. No `workmux open <name>`. Just navigate and press Enter.

## Flow 3: Add a repo (replaces nothing -- new capability)

```
Dashboard → press [a]

  ┌─ Add Repo ─────────────────────┐
  │                                │
  │  Path: ~/git/my-new-repo       │
  │  (tab-completion)              │
  │                                │
  │  [Enter] Add  [Esc] Cancel     │
  └────────────────────────────────┘
```

Validates it's a git repo. Scans for existing worktrees (`git worktree list`). Discovers any existing `pr-` tmux sessions. Adds to config. Persisted across restarts.

## Flow 4: Monitor agents (replaces ctrl-b w)

You never leave the dashboard. Status updates in real-time via polling.

- Waiting tasks float to top (red `!`) -- you see immediately which agents need input
- Select a task → bottom panel shows streaming preview of Claude's output
- Press Enter → jump into that tmux session to respond
- Press Esc → back to dashboard

When a session has multiple Claude panes, the most actionable state is surfaced (waiting > error > busy > idle).

## Flow 5: Session history & resume (replaces ccfind + manual --resume)

```
Dashboard → select a task → press [h]

  ┌─ Sessions: task-fix-curiosa ───────────────────┐
  │                                                │
  │  2026-02-23 14:30  "Fix the curiosa card..."   │
  │    42 messages · 1.2MB · session: a3b7c9d...   │
  │    [r]esume                                    │
  │                                                │
  │  2026-02-22 09:15  "Debug the drag handler"    │
  │    18 messages · 340KB · session: e5f8a1b...    │
  │    [r]esume                                    │
  │                                                │
  │  [s] Search all sessions                       │
  │  [Esc] Back                                    │
  └────────────────────────────────────────────────┘
```

**Tier 1 (default):** Shows Claude's own session index for this worktree path.
**Tier 2 ([s] search):** Deep scan across ALL `~/.claude/projects/` JSONL files. Keyword search, cross-worktree discovery, finds orphaned sessions.

Resume always uses `claude --resume <session-id>`, bypassing Claude's broken path-based lookup.

## Flow 6: Merge (replaces ccmanager merge)

```
Dashboard → select a task → press [m]

  ┌─ Merge: task-fix-curiosa ──────────────────────┐
  │                                                │
  │  Branch: task-fix-curiosa → main               │
  │  Commits: 3 ahead, 0 behind                   │
  │  Status: clean (no uncommitted changes)        │
  │                                                │
  │  [ ] Delete worktree after merge               │
  │  [ ] Delete branch after merge                 │
  │                                                │
  │  [Enter] Merge  [Esc] Cancel                   │
  └────────────────────────────────────────────────┘
```

Later feature. Simple git merge (not rebase, not squash -- keep it simple). Option to clean up worktree + branch after.

## Flow 7: Delete/cleanup

```
Dashboard → select a task → press [d]

  Delete task-fix-curiosa?
  [x] Close tmux session
  [x] Remove worktree (git worktree remove)
  [ ] Delete branch
  [Enter] Confirm  [Esc] Cancel
```

## Keyboard shortcut summary

| Key | Context | Action |
|-----|---------|--------|
| `j`/`k` or `↑`/`↓` | Dashboard | Navigate tasks |
| `Enter` | Dashboard | Focus selected task (jump into tmux session) |
| `Esc` | Anywhere | Back to dashboard |
| `n` | Dashboard | New task (create worktree + session) |
| `a` | Dashboard | Add repo |
| `d` | Dashboard | Delete/cleanup task |
| `m` | Dashboard | Merge task branch |
| `h` | Dashboard | Session history for selected task |
| `s` | Dashboard or History | Deep search across all sessions |
| `p` | Dashboard | Toggle streaming preview panel |
| `?` | Anywhere | Help |
| `q` | Dashboard | Quit (sessions keep running) |

## What happens on quit and relaunch

**Quit (`q`):** parkranger exits. All tmux sessions keep running. Agents keep working.

**Relaunch (`parkranger`):**
1. Loads config (repos, preferences)
2. Loads persisted state from BoltDB (task metadata, session hashes)
3. Scans for existing `pr-` tmux sessions → reattaches to any still running
4. For tasks with no tmux session → shows as "stopped" with option to relaunch
5. Starts polling for agent status
6. Dashboard is back exactly where you left it

## Comparison

| Step | Before (5 tools) | parkranger |
|------|-------------------|------------|
| Create worktree | `ccmanager` → navigate → create | `n` → type name → Enter |
| Open tmux session | `tmux new -s <name>` | (automatic on create) |
| List tasks | `workmux ls` | Dashboard IS the list |
| Open task | `workmux open <name>` | `j`/`k` → Enter |
| Check agent status | `ctrl-b w` → squint | Glance at status icons |
| Find old session | `ccfind <keyword>` | `h` → browse or `s` → search |
| Resume session | `claude --resume <id>` | `r` in history view |
| Merge | `ccmanager` → merge flow | `m` → confirm |
| Survive restart | Start over | Automatic |

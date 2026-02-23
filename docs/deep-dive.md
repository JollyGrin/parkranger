# Deep Dive: forestui, Session Persistence, and Agent Detection

> Research conducted 2026-02-23. Source code analysis from current main branches.

## 1. forestui Analysis

**Repo:** [flipbit03/forestui](https://github.com/flipbit03/forestui)
**Stack:** Python 3.14+, Textual 7.2+, Pydantic, Click, requires tmux

### What it does well (and what parkranger should learn from)

**UI Layout:**
- Sidebar tree: repos as parent nodes, worktrees nested beneath with branch names
- Detail panel: worktree info (branch, commit hash, path), action buttons, session list
- Quick-action buttons: Editor, Terminal, File Manager, Claude (with Resume & YOLO modes)
- GitHub integration: shows open issues, can create worktrees from issues

**Session History Display:**
- Reads Claude JSONL files from `~/.claude/projects/[encoded-path]/`
- Shows last 5 sessions per worktree with title, last message preview, timestamp, message count
- Resume button passes `-r <session-id>` to Claude Code
- YOLO button adds `--dangerously-skip-permissions`

**Tmux Integration:**
- Creates separate named windows: `edit:<name>`, `claude:<name>`, `term:<name>`, `files:<name>`
- Singleton TmuxService detects if running inside tmux via `TMUX` env var
- Unique window naming with auto-incrementing suffixes to avoid collisions
- TUI editor detection (vim, emacs, nano, helix, etc.)

### What it does NOT do (gaps parkranger fills)

1. **No agent status detection.** Zero idle/busy/waiting indicators. The `claude_session.py` service only reads historical JSONL files -- it has no pane capture, no process monitoring, no real-time state classification.

2. **No state persistence across restarts.** Forest directories are CLI arguments (`forestui ~/forest`). Config is per-forest (`.forestui-config.json`) but the app doesn't remember which forests you had open. Close it, reopen it, start from scratch.

3. **Separate windows, not split panes.** Creates individual tmux windows for editor/claude/terminal -- not a split-pane layout where you see both side by side.

4. **Single forest at a time.** While you can pass multiple forest dirs as args, the UI treats them independently. No unified multi-repo dashboard.

### forestui's session discovery approach (for reference)

```python
# Path encoding: /Users/grins/git/thegrid/.trees/dev-3383 →
#   ~/.claude/projects/-Users-grins-git-thegrid--trees-dev-3383/

# Session data extracted per JSONL file:
# - session_id (filename stem)
# - message_count (entries with type=="user" or role=="user")
# - last_timestamp (from data["timestamp"])
# - title (first 100 chars of first user message)
# - last_message (first 100 chars of most recent user message)
# - git_branches (from gitBranches array)
# - Skips "agent-" prefixed files
```

---

## 2. Claude Code Session Persistence Problem

### How Claude Code stores sessions

```
~/.claude/
├── projects/
│   ├── -Users-grins-git-thegrid-thegrid-discovery/     # main repo
│   │   ├── sessions-index.json
│   │   ├── <session-id-1>.jsonl
│   │   └── <session-id-2>.jsonl
│   ├── -Users-grins-git-thegrid--trees-dev-3383-new-profile-cards/  # worktree
│   │   ├── sessions-index.json
│   │   └── <session-id-3>.jsonl
│   └── ...
├── file-history/
│   └── <session-id>/         # per-session file edit history
└── history.jsonl              # global history (display, pastedContents, timestamp, project, sessionId)
```

**Path encoding rule:** Replace `/` with `-` in the absolute path.

### The worktree session isolation problem

Verified on your actual `~/.claude/projects/` directory:

| Worktree dir | Sessions | projectPath in sessions-index.json |
|---|---|---|
| `-Users-grins-git-thegrid--trees-dev-3383-new-profile-cards` | 34 | `/Users/grins/git/thegrid/thegrid-discovery` (main repo!) |
| `-Users-grins-git-thegrid--trees-dat-445-claims-csv` | 137 | Main monorepo paths |
| `-Users-grins-git--trees-task-add-cards` | 0 | No sessions-index.json |

**Key findings:**

1. **sessions-index.json in worktree dirs references the main repo's projectPath**, not the worktree path. This means Claude Code internally resolves worktrees back to their parent repo.

2. **Some worktree dirs have sessions, some don't.** The `--trees/task-*` dirs have no session data at all, while `--trees/dev-*` dirs do. This may depend on whether Claude was launched from the worktree directory or from elsewhere.

3. **The CWD in individual JSONL files IS correct** -- a session created in a worktree records `cwd: /Users/grins/git/thegrid/.trees/dev-3383-new-profile-cards`. The issue is at the index level.

4. **Session count inflation.** Multiple worktree dirs share the same 137-entry sessions-index with the main repo, suggesting the index is being duplicated or linked rather than isolated.

### Root cause

Claude Code identifies projects by **filesystem path**, not by **git repository identity** (remote URL or `.git` location). The path encoding scheme (`/` → `-`) creates unique project directories per CWD. Git worktrees create multiple filesystem paths that all refer to the same repo. Claude's `git worktree list` integration attempts to bridge this but has multiple failure modes:

1. **Worktree deletion orphans sessions.** Worktrees are disposable (create, work, merge, delete), but sessions tied to that path become unreachable -- `/resume` from other paths won't find them since the path no longer appears in `git worktree list`.
2. **No session inheritance.** No mechanism to continue a discussion from `main` into a new worktree. Every worktree starts from zero context.
3. **Submodule path mismatch.** `git worktree list` reports submodule paths differently from their actual filesystem paths, causing lookup failures.
4. **Session index cross-contamination.** Multiple worktree dirs share/duplicate the main repo's sessions-index (the 137-session duplication seen above).

### Related GitHub issues

- **[anthropics/claude-code#15776](https://github.com/anthropics/claude-code/issues/15776):** "Session state should persist across git worktrees" -- requests `claude --resume-from-parent` or automatic detection. The core UX pain: 50-100 messages of architecture context lost when switching to a worktree.
- **[anthropics/claude-code#27676](https://github.com/anthropics/claude-code/issues/27676):** "Task list state leaks across git worktrees" -- the inverse problem. Task lists scoped to shared `.git` dir, not per-worktree. Workaround: `CLAUDE_CODE_TASK_LIST_ID=my-feature claude`.
- **[kbwo/ccmanager#196](https://github.com/kbwo/ccmanager/issues/196):** ccmanager maintainer confirmed this is upstream in `claude` CLI, not a ccmanager bug. Workaround: configure `--resume <id>` as a preset argument.

### Existing workarounds

1. **Resume by session ID:** `claude --resume <uuid>` bypasses path-based lookup entirely
2. **Manual JSONL search:** `grep -rl "keyword" ~/.claude/projects/*/` (or the ccfind script below)
3. **ccmanager preset:** Configure `--resume` as a command argument rather than using `/resume` interactively

### Parkranger's approach to session discovery

Since Claude Code's own indexing is unreliable across worktrees, parkranger should:

1. **Build its own session index** by scanning `~/.claude/projects/` JSONL files directly
2. **Match sessions to worktrees** by checking the `cwd` field inside each JSONL file, not relying on the directory name encoding or sessions-index.json
3. **Provide the user's ccfind functionality built-in** -- search across all sessions with keyword matching
4. **Cache the index** and update incrementally (watch for new/modified JSONL files)

### User's existing ccfind script (reference)

```bash
ccfind() {
  if [ -z "$1" ]; then
    echo "Usage: ccfind <keyword> [project-filter]"
    return 1
  fi
  for f in $(find "$HOME/.claude/projects" -maxdepth 2 -name "*.jsonl" ${2:+-path "*$2*"} -type f); do
    if grep -ql "$1" "$f" 2>/dev/null; then
      local id=$(basename "$f" .jsonl)
      local project=$(basename "$(dirname "$f")")
      local mod=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$f")
      local size=$(du -h "$f" | cut -f1)
      local snippet=$(grep -m1 "$1" "$f" | sed 's/.*"text":"//;s/".*//' | cut -c1-80)
      echo "$mod  $size  $id"
      echo "  project: $project"
      echo "  match:   $snippet"
      echo "  resume:  claude --resume $id"
      echo ""
    fi
  done
}
```

This should be internalized into parkranger's session index with better performance (Go's concurrent file scanning vs sequential bash).

---

## 3. Agent Status Detection: How the Existing Tools Do It

### claude-squad approach (Go)

Source: [smtg-ai/claude-squad](https://github.com/smtg-ai/claude-squad)

**Mechanism:** `tmux capture-pane -p -t <session>:<window>.<pane>` every poll interval.

**State classification (from source code):**

```
1. Capture last ~50 lines of pane output
2. Strip ANSI escape codes
3. Hash the cleaned output
4. Compare hash to previous poll:
   - Hash changed → BUSY (output is streaming)
   - Hash unchanged → check patterns:
     a. Prompt patterns (waiting for input) → WAITING
     b. Error patterns → ERROR
     c. Otherwise → IDLE
```

**Key patterns matched:**
- Busy: output hash changed between two consecutive polls
- Waiting: looks for the `?` character in specific ANSI formatting contexts (Claude's permission prompts)
- Idle: stable output with no prompt indicators

### ccmanager approach (TypeScript)

Source: [kbwo/ccmanager](https://github.com/kbwo/ccmanager)

**Mechanism:** Per-agent state detector classes. Each agent type has its own pattern matchers.

**Claude Code detector (`claude.ts`):**
```typescript
// Captures last 30 lines via getTerminalContent(terminal, 30)
// Priority order:

// 1. WAITING_INPUT (highest priority):
"Do you want to proceed?"
"Allow command?"
"[Y/n]" or "[y/N]"
"yes (y)"
"press enter to confirm or esc to cancel"
/confirm with .+ enter/i

// 2. BUSY:
/esc.*interrupt/i  // Claude shows "Esc to interrupt" while working

// 3. IDLE (default):
// No patterns matched = idle
```

**Codex detector (`codex.ts`) for comparison:**
```typescript
// Same structure, different patterns:
// WAITING: "press enter to confirm", "allow command?", "[y/n]"
// BUSY: /esc.*interrupt/i
// IDLE: default
```

### Recommended detection strategy for parkranger

Based on both implementations:

```
Poll: tmux capture-pane -p -S -30 -t <target>  (last 30 lines, stdout, fast)

Detection priority:
1. WAITING — Pattern match on last 30 lines:
   - "Do you want to proceed"
   - "[Y/n]" / "[y/N]"
   - "Allow" + "?"
   - "press enter to confirm"
   - "yes (y)"

2. ERROR — Pattern match:
   - "Error:" / "error:" at line start
   - "panic:" / "fatal:"
   - Exit codes in prompt

3. BUSY — Output diff:
   - Hash current capture, compare to previous
   - If changed → busy (streaming)
   - Also: /esc.*interrupt/i pattern (Claude's interrupt hint)

4. IDLE — Default:
   - Output stable (hash unchanged) + no waiting patterns
```

**Performance notes:**
- `tmux capture-pane -p -S -30` is ~0.5ms per call (negligible)
- 10 sessions at 2s interval = 5 calls/second = trivial CPU
- Hash comparison avoids string scanning on every poll when nothing changed
- Strip ANSI codes before hashing for stability (color changes shouldn't trigger false "busy")

### Claude Code statusline (alternative detection source)

Claude Code has a built-in statusline feature (`~/.claude/statusline.sh`) that receives JSON on stdin with:
```json
{
  "model": {"display_name": "Opus"},
  "context_window": {"used_percentage": 25},
  "cost": {"total_cost_usd": 0.50, "total_duration_ms": 120000},
  "workspace": {"current_dir": "/path/to/project"}
}
```

This only runs while Claude Code is active and attached to a terminal. It doesn't expose busy/idle/waiting state directly, but the `context_window.used_percentage` changing between polls could be used as a secondary signal for "busy." This is **not** a reliable primary detection method since it requires Claude to be running and calling the script.

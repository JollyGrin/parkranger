package session

import (
	"crypto/sha256"
	"regexp"
	"strings"

	"github.com/grins/parkranger/internal/tmux"
)

// AgentStatus represents the detected state of a Claude agent in a tmux pane.
type AgentStatus int

const (
	StatusUnknown AgentStatus = iota
	StatusIdle
	StatusBusy
	StatusWaiting
)

func (s AgentStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusBusy:
		return "busy"
	case StatusWaiting:
		return "waiting"
	default:
		return "unknown"
	}
}

// busyActivityRe matches Claude's activity stats line, e.g.
// "↓ 20.1k tokens · thought for 288s" or "tokens · thinking"
var busyActivityRe = regexp.MustCompile(`(?i)\d+\.?\d*k?\s+tokens?\s*·\s*(?:thinking|thought)`)

// LiveInfo describes the live state of a tmux session's Claude pane.
type LiveInfo struct {
	Exists      bool        // tmux session exists
	HasClaude   bool        // Claude UI detected in pane
	Status      AgentStatus // idle/busy/waiting
	PaneContent string      // raw captured pane text (for preview)
}

// Detector wraps stateful agent detection with hash-based change tracking.
// Create one per worktree and reuse across polls.
type Detector struct {
	prevHash   [32]byte
	prevStatus AgentStatus
	hasHistory bool
}

// Detect checks the tmux window for a running Claude agent, using hash-based
// change detection to upgrade Unknown→Busy when pane content is changing.
func (d *Detector) Detect(sessionName, windowName string) LiveInfo {
	if !tmux.WindowExists(sessionName, windowName) {
		return LiveInfo{}
	}

	target := sessionName + ":" + windowName + ".1"
	output, err := tmux.CapturePaneBottom(target, 30)
	if err != nil || output == "" {
		return LiveInfo{Exists: true}
	}

	status, hasClaude := classifyPaneOutput(output)

	// Hash-based change detection: if pattern detection is inconclusive
	// but the pane content changed, the agent is likely streaming output.
	hash := sha256.Sum256([]byte(output))
	if status == StatusUnknown && d.hasHistory && hash != d.prevHash {
		status = StatusBusy
		hasClaude = true
	}

	d.prevHash = hash
	d.prevStatus = status
	d.hasHistory = true

	return LiveInfo{
		Exists:      true,
		HasClaude:   hasClaude,
		Status:      status,
		PaneContent: output,
	}
}

// DetectLive checks the tmux window for a running Claude agent.
// Captures the bottom 30 lines of pane <session>:<window>.1 (the right pane
// where claude runs) to always see the most recent activity.
// This is the stateless version — prefer Detector for repeated polling.
func DetectLive(sessionName, windowName string) LiveInfo {
	if !tmux.WindowExists(sessionName, windowName) {
		return LiveInfo{}
	}

	target := sessionName + ":" + windowName + ".1"
	output, err := tmux.CapturePaneBottom(target, 30)
	if err != nil || output == "" {
		return LiveInfo{Exists: true}
	}

	status, hasClaude := classifyPaneOutput(output)
	return LiveInfo{
		Exists:      true,
		HasClaude:   hasClaude,
		Status:      status,
		PaneContent: output,
	}
}

// bottomN returns the last n elements of a string slice.
func bottomN(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// classifyPaneOutput determines the agent status from captured pane content.
//
// Status indicators in Claude Code appear at the bottom of the terminal pane.
// We restrict pattern matching to the bottom N lines to avoid false positives
// from stale indicators that scrolled up but remain visible in the capture.
//
// Detection priority (from CLAUDE.md):
//  1. ⌕ (U+2315) → idle, not claude (search UI overlay)
//  2. "ctrl+r to toggle" → unknown (history search UI)
//  3. Permission prompts / "esc to cancel" → waiting
//  4. ✢ (U+2722) spinner / activity stats / "esc to interrupt" → busy
//  5. Claude UI elements visible → idle
//  6. Default → unknown
func classifyPaneOutput(output string) (AgentStatus, bool) {
	// Split into lines and trim trailing blanks (tmux capture-pane may include them)
	lines := strings.Split(output, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return StatusUnknown, false
	}

	lower := strings.ToLower(output)

	// Bottom slices for targeted detection.
	// Claude Code's status bar, prompts, and input area are always at the
	// bottom of the pane. Checking only the bottom avoids stale patterns
	// from earlier output that scrolled up but remain visible.
	b15 := bottomN(lines, 15)
	b10 := bottomN(lines, 10)
	b5 := bottomN(lines, 5)
	bot10 := strings.ToLower(strings.Join(b10, "\n"))
	bot5 := strings.ToLower(strings.Join(b5, "\n"))
	raw5 := strings.Join(b5, "\n")       // preserve case for unicode (❯)
	raw15 := strings.Join(b15, "\n")     // preserve case for unicode (✢)

	// 1. Search UI override — ⌕ (U+2315) anywhere → idle, not Claude
	if strings.Contains(output, "\u2315") {
		return StatusIdle, false
	}

	// 2. History search — bottom 10 → unknown (state hold)
	if strings.Contains(bot10, "ctrl+r to toggle") {
		return StatusUnknown, false
	}

	// 3. Waiting — needs user attention (bottom of pane only)
	//    "esc to cancel": active input/dialog UI element (bottom 5)
	//    "No, and tell Claude...": ephemeral selection text, only visible when active (bottom 10)
	//    "do you want" / "would you like": question text (bottom 5 to avoid stale matches)
	if strings.Contains(bot5, "esc to cancel") ||
		strings.Contains(bot10, "no, and tell claude what to do differently") ||
		strings.Contains(bot5, "do you want") ||
		strings.Contains(bot5, "would you like") {
		return StatusWaiting, true
	}

	// 4. Busy — Claude is working
	//    ✢ (U+2722) spinner: only visible during active processing (bottom 15)
	//    Activity stats: "tokens · thinking/thought" (bottom 10)
	//    Legacy: "esc to interrupt" / "ctrl+c to interrupt" (bottom 5)
	if strings.Contains(raw15, "\u2722") {
		return StatusBusy, true
	}
	if busyActivityRe.MatchString(bot10) {
		return StatusBusy, true
	}
	if strings.Contains(bot5, "esc to interrupt") ||
		strings.Contains(bot5, "ctrl+c to interrupt") {
		return StatusBusy, true
	}

	// 5. Claude UI elements → idle
	//    "claude" branding anywhere in output + prompt/hint indicators at bottom
	//    Also detect model names (opus, sonnet, haiku) and "ctx:" in status bar
	hasPrompt := strings.Contains(raw5, "❯") ||
		strings.Contains(bot10, "type a message") ||
		strings.Contains(bot10, "type your message")
	hasHints := strings.Contains(bot10, "/help") ||
		strings.Contains(bot10, "shift+")

	// Model name or context indicator in status bar → Claude session
	hasModelBar := strings.Contains(bot5, "ctx:") ||
		strings.Contains(bot5, "opus") ||
		strings.Contains(bot5, "sonnet") ||
		strings.Contains(bot5, "haiku")

	if strings.Contains(lower, "claude") && (hasPrompt || hasHints) {
		return StatusIdle, true
	}

	// Also idle when Claude header scrolled off-screen but prompt + hints visible
	if hasPrompt && hasHints {
		return StatusIdle, true
	}

	// Model/ctx bar + prompt → idle Claude (header may have scrolled off)
	if hasModelBar && hasPrompt {
		return StatusIdle, true
	}

	return StatusUnknown, false
}

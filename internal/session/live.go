package session

import (
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

// LiveInfo describes the live state of a tmux session's Claude pane.
type LiveInfo struct {
	Exists    bool        // tmux session exists
	HasClaude bool        // Claude UI detected in pane
	Status    AgentStatus // idle/busy/waiting
}

// DetectLive checks the tmux window for a running Claude agent.
// Captures the bottom 30 lines of pane <session>:<window>.1 (the right pane
// where claude runs) to always see the most recent activity.
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
		Exists:    true,
		HasClaude: hasClaude,
		Status:    status,
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
//  4. "esc to interrupt" / "ctrl+c to interrupt" → busy
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
	b10 := bottomN(lines, 10)
	b5 := bottomN(lines, 5)
	bot10 := strings.ToLower(strings.Join(b10, "\n"))
	bot5 := strings.ToLower(strings.Join(b5, "\n"))
	raw5 := strings.Join(b5, "\n") // preserve case for unicode (❯)

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

	// 4. Busy — Claude is working (bottom 5 only — status bar indicators)
	if strings.Contains(bot5, "esc to interrupt") ||
		strings.Contains(bot5, "ctrl+c to interrupt") {
		return StatusBusy, true
	}

	// 5. Claude UI elements → idle
	//    "claude" branding anywhere in output + prompt/hint indicators at bottom
	hasPrompt := strings.Contains(raw5, "❯") ||
		strings.Contains(bot10, "type a message") ||
		strings.Contains(bot10, "type your message")
	hasHints := strings.Contains(bot10, "/help") ||
		strings.Contains(bot10, "shift+")

	if strings.Contains(lower, "claude") && (hasPrompt || hasHints) {
		return StatusIdle, true
	}

	// Also idle when Claude header scrolled off-screen but prompt + hints visible
	if hasPrompt && hasHints {
		return StatusIdle, true
	}

	return StatusUnknown, false
}

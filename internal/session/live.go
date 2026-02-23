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
// Captures pane <session>:<window>.1 (the right pane where claude runs).
func DetectLive(sessionName, windowName string) LiveInfo {
	if !tmux.WindowExists(sessionName, windowName) {
		return LiveInfo{}
	}

	output, err := tmux.CapturePane(sessionName + ":" + windowName + ".1")
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

// classifyPaneOutput determines the agent status from captured pane content.
// Detection priority (from CLAUDE.md):
//  1. ⌕ (U+2315) → idle, not claude
//  2. "ctrl+r to toggle" → unknown (history search UI)
//  3. Permission prompts / "esc to cancel" → waiting
//  4. "esc to interrupt" / "ctrl+c to interrupt" → busy
//  5. Claude UI elements visible → idle
//  6. Default → unknown
func classifyPaneOutput(output string) (AgentStatus, bool) {
	lower := strings.ToLower(output)

	// 1. Search UI override
	if strings.Contains(output, "\u2315") {
		return StatusIdle, false
	}

	// 2. History search — maintain unknown
	if strings.Contains(lower, "ctrl+r to toggle") {
		return StatusUnknown, false
	}

	// 3. Waiting — needs user attention
	if strings.Contains(lower, "esc to cancel") ||
		strings.Contains(lower, "do you want") ||
		strings.Contains(lower, "would you like") ||
		strings.Contains(lower, "no, and tell claude what to do differently") {
		return StatusWaiting, true
	}

	// 4. Busy — claude is working
	if strings.Contains(lower, "esc to interrupt") ||
		strings.Contains(lower, "ctrl+c to interrupt") {
		return StatusBusy, true
	}

	// 5. Claude UI elements — idle
	if strings.Contains(lower, "claude") && (strings.Contains(lower, "/help") ||
		strings.Contains(output, "❯") ||
		strings.Contains(lower, "shift+") ||
		strings.Contains(lower, "type a message")) {
		return StatusIdle, true
	}

	return StatusUnknown, false
}

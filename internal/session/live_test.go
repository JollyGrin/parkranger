package session

import (
	"strings"
	"testing"
)

func TestClassifyPaneOutput_SearchUI(t *testing.T) {
	output := "some stuff\n⌕ Search…\nmore stuff"
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || hasClaude {
		t.Errorf("got status=%v hasClaude=%v, want idle/false", status, hasClaude)
	}
}

func TestClassifyPaneOutput_HistorySearch(t *testing.T) {
	output := "bck-search: something\nctrl+r to toggle"
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusUnknown || hasClaude {
		t.Errorf("got status=%v hasClaude=%v, want unknown/false", status, hasClaude)
	}
}

func TestClassifyPaneOutput_Waiting(t *testing.T) {
	tests := []string{
		"Do you want to proceed?\n❯ Yes",
		"Press esc to cancel",
		"No, and tell Claude what to do differently",
	}
	for _, output := range tests {
		status, hasClaude := classifyPaneOutput(output)
		if status != StatusWaiting || !hasClaude {
			t.Errorf("output=%q: got status=%v hasClaude=%v, want waiting/true", output, status, hasClaude)
		}
	}
}

func TestClassifyPaneOutput_Busy(t *testing.T) {
	tests := []string{
		"Working on it...\nesc to interrupt",
		"Processing...\nctrl+c to interrupt",
	}
	for _, output := range tests {
		status, hasClaude := classifyPaneOutput(output)
		if status != StatusBusy || !hasClaude {
			t.Errorf("output=%q: got status=%v hasClaude=%v, want busy/true", output, status, hasClaude)
		}
	}
}

func TestClassifyPaneOutput_Idle(t *testing.T) {
	output := "Claude Code v1.0.0\n❯ type a message\n/help for commands"
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_Unknown(t *testing.T) {
	output := "$ ls\nfile1.go  file2.go"
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusUnknown || hasClaude {
		t.Errorf("got status=%v hasClaude=%v, want unknown/false", status, hasClaude)
	}
}

func TestAgentStatusString(t *testing.T) {
	tests := []struct {
		s    AgentStatus
		want string
	}{
		{StatusUnknown, "unknown"},
		{StatusIdle, "idle"},
		{StatusBusy, "busy"},
		{StatusWaiting, "waiting"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

// --- Bottom-line restriction tests ---
// These verify that stale indicators in scrollback don't cause false positives.

func TestClassifyPaneOutput_StaleBusyAboveIdle(t *testing.T) {
	// Claude was busy, finished, now idle. "esc to interrupt" visible higher up.
	var lines []string
	lines = append(lines, "Working on the task...")
	lines = append(lines, "Reading file.go...")
	lines = append(lines, "esc to interrupt") // stale busy indicator
	for i := 0; i < 12; i++ {
		lines = append(lines, "  Output line from claude")
	}
	lines = append(lines, "Claude Code v1.0.0")
	lines = append(lines, "❯ type a message")
	lines = append(lines, "/help for commands")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("stale busy above idle: got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_StaleWaitingAboveBusy(t *testing.T) {
	// Claude asked "do you want", user said yes, now Claude is busy.
	// "Do you want" visible higher up in scrollback.
	var lines []string
	lines = append(lines, "Do you want to apply this edit?")
	lines = append(lines, "Applied edit to file.go")
	for i := 0; i < 12; i++ {
		lines = append(lines, "  Processing changes...")
	}
	lines = append(lines, "esc to interrupt")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusBusy || !hasClaude {
		t.Errorf("stale waiting above busy: got status=%v hasClaude=%v, want busy/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_StaleWaitingAboveIdle(t *testing.T) {
	// Permission prompt answered long ago, now idle.
	var lines []string
	lines = append(lines, "Do you want to proceed?")
	lines = append(lines, "Proceeding with changes...")
	for i := 0; i < 10; i++ {
		lines = append(lines, "  Completed step")
	}
	lines = append(lines, "Claude Code v1.0.0")
	lines = append(lines, "❯ type a message")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("stale waiting above idle: got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_StaleEscCancelAboveIdle(t *testing.T) {
	// "esc to cancel" from an old text input, now idle.
	var lines []string
	lines = append(lines, "Enter file path:")
	lines = append(lines, "esc to cancel")
	for i := 0; i < 12; i++ {
		lines = append(lines, "  Some output")
	}
	lines = append(lines, "Claude Code v1.0.0")
	lines = append(lines, "❯")
	lines = append(lines, "/help for commands")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("stale esc-to-cancel above idle: got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_ActiveWaitingOverStalebusy(t *testing.T) {
	// "esc to interrupt" visible from before, but now there's an active permission prompt.
	// Waiting should win (higher priority + at bottom).
	var lines []string
	lines = append(lines, "esc to interrupt") // stale
	for i := 0; i < 8; i++ {
		lines = append(lines, "  Some output")
	}
	lines = append(lines, "Do you want to apply this edit?")
	lines = append(lines, "  ❯ Yes")
	lines = append(lines, "    No, and tell Claude what to do differently")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusWaiting || !hasClaude {
		t.Errorf("active waiting over stale busy: got status=%v hasClaude=%v, want waiting/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_IdleWithoutHeader(t *testing.T) {
	// Claude header scrolled off-screen, but prompt and hints visible at bottom.
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "  Long output from a task")
	}
	lines = append(lines, "❯")
	lines = append(lines, "/help for commands  shift+enter for newline")

	output := strings.Join(lines, "\n")
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("idle without header: got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_TrailingBlanks(t *testing.T) {
	// tmux capture-pane may include trailing blank lines
	output := "Claude Code v1.0.0\n❯ type a message\n/help for commands\n\n\n\n"
	status, hasClaude := classifyPaneOutput(output)
	if status != StatusIdle || !hasClaude {
		t.Errorf("trailing blanks: got status=%v hasClaude=%v, want idle/true", status, hasClaude)
	}
}

func TestClassifyPaneOutput_EmptyOutput(t *testing.T) {
	status, hasClaude := classifyPaneOutput("")
	if status != StatusUnknown || hasClaude {
		t.Errorf("empty: got status=%v hasClaude=%v, want unknown/false", status, hasClaude)
	}
}

func TestClassifyPaneOutput_OnlyBlanks(t *testing.T) {
	status, hasClaude := classifyPaneOutput("\n\n   \n\n")
	if status != StatusUnknown || hasClaude {
		t.Errorf("only blanks: got status=%v hasClaude=%v, want unknown/false", status, hasClaude)
	}
}

func TestBottomN(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	got := bottomN(lines, 3)
	if len(got) != 3 || got[0] != "c" || got[1] != "d" || got[2] != "e" {
		t.Errorf("bottomN(5, 3) = %v, want [c d e]", got)
	}

	got = bottomN(lines, 10)
	if len(got) != 5 {
		t.Errorf("bottomN(5, 10) = %v, want all 5", got)
	}

	got = bottomN(nil, 3)
	if len(got) != 0 {
		t.Errorf("bottomN(nil, 3) = %v, want empty", got)
	}
}

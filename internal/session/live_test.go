package session

import "testing"

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

package tmux

import (
	"testing"
)

func TestSessionName(t *testing.T) {
	tests := []struct {
		repo, wt string
		want     string
	}{
		{"myrepo", "feat-x", "pr-myrepo-feat-x"},
		{"my.repo", "fix:bug", "pr-my-repo-fix-bug"},
		{"repo", "task.123", "pr-repo-task-123"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := SessionName(tt.repo, tt.wt)
			if got != tt.want {
				t.Errorf("SessionName(%q, %q) = %q, want %q", tt.repo, tt.wt, got, tt.want)
			}
		})
	}
}

func TestSessionExistsNonexistent(t *testing.T) {
	// A session with this name should never exist
	if SessionExists("pr-test-nonexistent-session-xyz") {
		t.Error("expected false for nonexistent session")
	}
}

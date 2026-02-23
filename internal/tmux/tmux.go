// Package tmux provides wrappers around the tmux CLI for session lifecycle.
package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// run executes tmux with the given args, returning trimmed stdout.
func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// SessionName builds a sanitized tmux session name: pr-<repo>-<worktree>.
// Replaces characters that tmux forbids in session names (. and :).
func SessionName(repo, worktree string) string {
	name := fmt.Sprintf("pr-%s-%s", repo, worktree)
	r := strings.NewReplacer(".", "-", ":", "-")
	return r.Replace(name)
}

// SessionExists returns true if the named tmux session exists.
// Returns false (not an error) if the tmux server is not running.
func SessionExists(name string) bool {
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

// CreateSession creates a detached tmux session in the given directory.
func CreateSession(name, workDir string) error {
	_, err := run("new-session", "-d", "-s", name, "-c", workDir)
	return err
}

// SplitVertical splits the session's current window horizontally (side-by-side).
func SplitVertical(session, workDir string) error {
	_, err := run("split-window", "-h", "-t", session, "-c", workDir)
	return err
}

// SendKeys sends keystrokes to the given tmux target followed by Enter.
func SendKeys(target, keys string) error {
	_, err := run("send-keys", "-t", target, keys, "Enter")
	return err
}

// AttachSession attaches to (or switches to) the named session.
// Inside tmux: uses switch-client. Outside: replaces the process with tmux attach via syscall.Exec.
func AttachSession(name string) error {
	if IsInsideTmux() {
		_, err := run("switch-client", "-t", name)
		return err
	}

	// Find the tmux binary
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	// syscall.Exec replaces this process entirely — no Go code runs after this
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}

// CapturePane captures the visible content of a tmux pane.
// Returns "" (not an error) if the pane or session doesn't exist.
func CapturePane(target string) (string, error) {
	out, err := run("capture-pane", "-p", "-J", "-t", target)
	if err != nil {
		// Pane doesn't exist — not an error for our purposes
		return "", nil
	}
	return out, nil
}

// KillSession destroys the named tmux session.
func KillSession(name string) error {
	_, err := run("kill-session", "-t", name)
	return err
}

// IsInsideTmux returns true if the current process is running inside tmux.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

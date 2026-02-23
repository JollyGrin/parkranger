// Package session provides Claude Code session discovery and live detection.
package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// sessionMeta holds metadata extracted from the first few lines of a JSONL session file.
type sessionMeta struct {
	ID          string
	CWD         string
	FirstPrompt string
	GitBranch   string
}

// parseJSONLMeta reads the first 10 lines of a JSONL session file and extracts metadata.
// Returns nil if cwd doesn't match worktreePath (filters out duplicated parent sessions).
func parseJSONLMeta(filePath, worktreePath string) (*sessionMeta, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	meta := &sessionMeta{
		ID: strings.TrimSuffix(filepath.Base(filePath), ".jsonl"),
	}

	for i := 0; i < 20 && scanner.Scan(); i++ {
		line := scanner.Bytes()

		var entry struct {
			Type      string          `json:"type"`
			CWD       string          `json:"cwd"`
			Message   json.RawMessage `json:"message"`
			GitBranch string          `json:"gitBranch"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.CWD != "" && meta.CWD == "" {
			meta.CWD = entry.CWD
		}

		if entry.GitBranch != "" && meta.GitBranch == "" {
			meta.GitBranch = entry.GitBranch
		}

		if entry.Type == "user" && meta.FirstPrompt == "" {
			text := extractMessageText(entry.Message)
			if text != "" && !isBoilerplate(text) {
				meta.FirstPrompt = text
			}
		}
	}

	if meta.CWD == "" {
		return nil, nil
	}

	if !pathsMatch(meta.CWD, worktreePath) {
		return nil, nil
	}

	return meta, nil
}

// extractMessageText pulls display text from a Claude JSONL message field.
// The message is {role, content} where content is a string or [{type:"text", text:"..."}].
func extractMessageText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// The message field is {role: "user", content: <string or array>}
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil || len(msg.Content) == 0 {
		return ""
	}

	return extractContent(msg.Content)
}

// extractContent extracts text from a content field that can be a string or array of blocks.
func extractContent(raw json.RawMessage) string {
	// Try plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return truncate(s, 80)
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				return truncate(b.Text, 80)
			}
		}
	}

	return ""
}

// truncate cuts s to maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	// Collapse newlines to spaces for display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// isBoilerplate returns true for messages that aren't useful as session descriptions.
func isBoilerplate(text string) bool {
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "[request interrupted") ||
		strings.HasPrefix(lower, "resume")
}

// pathsMatch compares two paths after cleaning (handles trailing slashes, symlink differences).
func pathsMatch(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

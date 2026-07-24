package jarvis

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session is a local Claude Code session, summarized for the dashboard.
type Session struct {
	ID           string
	Title        string
	Cwd          string
	Branch       string
	LastActivity time.Time
	WaitingOnMe  bool // the last turn was Claude's — the ball is in your court
}

// rawEntry is the subset of a transcript line we care about.
type rawEntry struct {
	Type        string `json:"type"`
	Cwd         string `json:"cwd"`
	GitBranch   string `json:"gitBranch"`
	Timestamp   string `json:"timestamp"`
	AiTitle     string `json:"aiTitle"`
	LastPrompt  string `json:"lastPrompt"`
	IsMeta      bool   `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Message     *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// DiscoverSessions scans ~/.claude/projects for sessions whose last turn was
// Claude's (waiting on the user) and that saw activity within maxAgeDays.
// Best-effort: unreadable files are skipped, not fatal.
func DiscoverSessions(maxAgeDays int) ([]Session, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, ".claude", "projects")
	projectDirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	var sessions []Session
	for _, pd := range projectDirs {
		if !pd.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, pd.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			s, ok := parseSession(filepath.Join(root, pd.Name(), f.Name()))
			if !ok || !s.WaitingOnMe || s.LastActivity.Before(cutoff) {
				continue
			}
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

// parseSession reads a transcript and summarizes it. The session is "waiting on
// me" when the last assistant turn is newer than the last real user prompt.
func parseSession(path string) (Session, bool) {
	file, err := os.Open(path)
	if err != nil {
		return Session{}, false
	}
	defer file.Close()

	s := Session{ID: strings.TrimSuffix(filepath.Base(path), ".jsonl")}
	var lastUser, lastAssistant time.Time
	var lastPrompt, firstUserText string

	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024) // tolerate long lines
	for sc.Scan() {
		var e rawEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		if e.Cwd != "" {
			s.Cwd = e.Cwd
		}
		if e.GitBranch != "" {
			s.Branch = e.GitBranch
		}
		if e.AiTitle != "" {
			s.Title = e.AiTitle
		}
		if e.LastPrompt != "" {
			lastPrompt = e.LastPrompt
		}
		ts := parseTime(e.Timestamp)
		if !ts.IsZero() && ts.After(s.LastActivity) {
			s.LastActivity = ts
		}
		if e.Message == nil || e.IsMeta || e.IsSidechain {
			continue
		}
		switch e.Type {
		case "assistant":
			if ts.After(lastAssistant) {
				lastAssistant = ts
			}
		case "user":
			// A real user prompt has string content; tool results are arrays.
			if isUserPrompt(e.Message.Content) {
				if ts.After(lastUser) {
					lastUser = ts
				}
				if firstUserText == "" {
					firstUserText = stringContent(e.Message.Content)
				}
			}
		}
	}

	s.WaitingOnMe = lastAssistant.After(lastUser)
	if s.Title == "" {
		if lastPrompt != "" {
			s.Title = lastPrompt
		} else if firstUserText != "" {
			s.Title = firstUserText
		} else {
			s.Title = "(untitled session)"
		}
	}
	s.Title = firstLine(s.Title)
	return s, true
}

// isUserPrompt reports whether message content is a plain string (a typed prompt)
// rather than a JSON array (a tool result).
func isUserPrompt(content json.RawMessage) bool {
	t := strings.TrimSpace(string(content))
	return strings.HasPrefix(t, `"`)
}

func stringContent(content json.RawMessage) string {
	var s string
	if err := json.Unmarshal(content, &s); err != nil {
		return ""
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

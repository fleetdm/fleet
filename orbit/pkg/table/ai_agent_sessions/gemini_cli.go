package ai_agent_sessions

import (
	"path/filepath"
	"sort"
	"time"
)

// geminiLogEntry is one record of a Gemini CLI logs.json file.
type geminiLogEntry struct {
	SessionID string `json:"sessionId"`
	MessageID int64  `json:"messageId"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
}

// collectGeminiCLI reads Gemini CLI history from
// ~/.gemini/tmp/<project-hash>/logs.json, a single array covering every
// session run in that project.
//
// Gemini identifies projects by an opaque hash of their path, so project_path
// stays empty. The log records prompts only, so assistant_messages, tool_calls,
// model, and token usage are all unavailable.
func collectGeminiCLI(u userHome, cutoff time.Time) []Session {
	tmpDir := filepath.Join(u.HomeDir, ".gemini", "tmp")

	var sessions []Session
	for _, project := range subdirs(tmpDir) {
		path := filepath.Join(tmpDir, project, "logs.json")
		if isStale(path, cutoff) {
			continue
		}
		sessions = append(sessions, parseGeminiLog(path, u.Username)...)
	}
	return sessions
}

// parseGeminiLog splits one project log into its constituent sessions.
func parseGeminiLog(path, username string) []Session {
	var entries []geminiLogEntry
	if err := readJSONFile(path, &entries); err != nil {
		return nil
	}

	size, modTime := fileInfo(path)

	// The log interleaves sessions, so accumulate per session id.
	byID := map[string]*Session{}
	var order []string
	for _, entry := range entries {
		if entry.SessionID == "" {
			continue
		}
		s, seen := byID[entry.SessionID]
		if !seen {
			s = &Session{
				Agent:           agentGeminiCLI,
				Username:        username,
				SessionID:       entry.SessionID,
				TranscriptPath:  path,
				TranscriptBytes: size,
			}
			byID[entry.SessionID] = s
			order = append(order, entry.SessionID)
		}
		s.observe(parseTime(entry.Timestamp))
		if entry.Type == "user" {
			s.UserMessages++
		}
	}

	sort.Strings(order)
	sessions := make([]Session, 0, len(order))
	for _, id := range order {
		s := byID[id]
		if s == nil {
			continue
		}
		if s.EndedAt.IsZero() {
			s.EndedAt = modTime
		}
		sessions = append(sessions, *s)
	}
	return sessions
}

// geminiMessageEntry is one record of a Gemini CLI logs.json, read for its
// content.
type geminiMessageEntry struct {
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

// discoverGeminiTranscripts lists this user's Gemini CLI project logs. One log
// holds many sessions, so the ref carries no session id and the driver filters
// per message instead.
func discoverGeminiTranscripts(u userHome, cutoff time.Time) []transcriptRef {
	tmpDir := filepath.Join(u.HomeDir, ".gemini", "tmp")

	var refs []transcriptRef
	for _, project := range subdirs(tmpDir) {
		path := filepath.Join(tmpDir, project, "logs.json")
		size, modTime := fileInfo(path)
		if size == 0 || isStale(path, cutoff) {
			continue
		}
		refs = append(refs, transcriptRef{
			Agent:    agentGeminiCLI,
			Username: u.Username,
			Path:     path,
			ModTime:  modTime,
			Parse:    parseGeminiMessages,
		})
	}
	return refs
}

// parseGeminiMessages reads the prompts out of one project log. Gemini records
// user input only, so there are no assistant rows.
func parseGeminiMessages(ref transcriptRef, remaining, maxContentBytes int) []Message {
	var entries []geminiMessageEntry
	if err := readJSONFile(ref.Path, &entries); err != nil {
		return nil
	}

	b := newBlockBuilder(ref, remaining, maxContentBytes)
	for _, entry := range entries {
		if b.full() {
			break
		}
		if entry.Type != "user" {
			continue
		}
		// Each entry belongs to its own session, so the row carries the id
		// rather than inheriting it from the file.
		b.ref.SessionID = entry.SessionID
		b.nextMessage()
		b.add(roleUser, blockText, "", entry.Message, parseTime(entry.Timestamp), "")
	}
	return b.messages
}

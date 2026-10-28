package ai_agent_sessions

import (
	"path/filepath"
	"time"
)

// cursorRecord is one line of a Cursor agent transcript.
type cursorRecord struct {
	Role    string `json:"role"`
	Message struct {
		Content contentBlocks `json:"content"`
	} `json:"message"`
}

// collectCursor reads Cursor agent transcripts from
// ~/.cursor/projects/<project-slug>/agent-transcripts/<session-id>/<session-id>.jsonl.
//
// These transcripts carry no timestamps, model name, or token usage, so those
// columns stay empty and the session's end time comes from the file's last
// write.
func collectCursor(u userHome, cutoff time.Time) []Session {
	projectsDir := filepath.Join(u.HomeDir, ".cursor", "projects")

	var sessions []Session
	for _, project := range subdirs(projectsDir) {
		transcriptsDir := filepath.Join(projectsDir, project, "agent-transcripts")
		projectPath := resolveSlugPath(cursorSlugRoot(u.HomeDir), project)

		for _, sessionID := range subdirs(transcriptsDir) {
			path := filepath.Join(transcriptsDir, sessionID, sessionID+".jsonl")
			if isStale(path, cutoff) {
				continue
			}
			if s, ok := parseCursorTranscript(path, sessionID, projectPath, u.Username); ok {
				sessions = append(sessions, s)
			}
		}
	}
	return sessions
}

// cursorSlugRoot is the directory a Cursor project slug is relative to. Slugs
// encode an absolute path with its separators replaced by dashes.
func cursorSlugRoot(homeDir string) string {
	if vol := filepath.VolumeName(homeDir); vol != "" {
		return vol + string(filepath.Separator)
	}
	return string(filepath.Separator)
}

// parseCursorTranscript summarizes one Cursor session.
func parseCursorTranscript(path, sessionID, projectPath, username string) (Session, bool) {
	size, modTime := fileInfo(path)
	if size == 0 && modTime.IsZero() {
		return Session{}, false
	}

	s := Session{
		Agent:           agentCursor,
		Username:        username,
		SessionID:       sessionID,
		ProjectPath:     projectPath,
		EndedAt:         modTime,
		TranscriptPath:  path,
		TranscriptBytes: size,
	}

	var sawRecord bool
	err := scanJSONL(path, func(rec *cursorRecord) {
		sawRecord = true
		switch rec.Role {
		case "user":
			if rec.Message.Content.hasOtherThan("tool_result") {
				s.UserMessages++
			}
		case "assistant":
			s.AssistantMessages++
			s.ToolCalls += rec.Message.Content.count("tool_use")
		}
	})
	if err != nil || !sawRecord {
		return Session{}, false
	}
	return s, true
}

// cursorMessageRecord is one transcript line, read for its content.
type cursorMessageRecord struct {
	Role    string `json:"role"`
	Message struct {
		Content richContent `json:"content"`
	} `json:"message"`
}

// discoverCursorTranscripts lists this user's Cursor agent sessions.
func discoverCursorTranscripts(u userHome, cutoff time.Time) []transcriptRef {
	projectsDir := filepath.Join(u.HomeDir, ".cursor", "projects")

	var refs []transcriptRef
	for _, project := range subdirs(projectsDir) {
		transcriptsDir := filepath.Join(projectsDir, project, "agent-transcripts")
		for _, sessionID := range subdirs(transcriptsDir) {
			path := filepath.Join(transcriptsDir, sessionID, sessionID+".jsonl")
			size, modTime := fileInfo(path)
			if size == 0 || isStale(path, cutoff) {
				continue
			}
			refs = append(refs, transcriptRef{
				Agent:     agentCursor,
				SessionID: sessionID,
				Username:  u.Username,
				Path:      path,
				ModTime:   modTime,
				Parse:     parseCursorMessages,
			})
		}
	}
	return refs
}

// parseCursorMessages reads the conversation out of one transcript. Cursor
// records no per-message timestamps, so the timestamp column stays 0.
func parseCursorMessages(ref transcriptRef, remaining, maxContentBytes int) []Message {
	b := newBlockBuilder(ref, remaining, maxContentBytes)

	_ = scanJSONL(ref.Path, func(rec *cursorMessageRecord) {
		if b.full() || rec.Role == "" {
			return
		}
		b.nextMessage()
		b.addBlocks(contentRole(rec.Role, rec.Message.Content), rec.Message.Content, time.Time{}, "")
	})

	return b.messages
}

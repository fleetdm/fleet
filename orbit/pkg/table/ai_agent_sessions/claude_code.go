package ai_agent_sessions

import (
	"path/filepath"
	"strings"
	"time"
)

// claudeRecord is one line of a Claude Code transcript. Only the metadata
// fields are declared; message text is skipped by the decoder.
type claudeRecord struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
	CWD       string `json:"cwd"`
	Version   string `json:"version"`
	GitBranch string `json:"gitBranch"`
	Message   struct {
		Model   string        `json:"model"`
		Content contentBlocks `json:"content"`
		Usage   struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// claudeTranscriptPaths lists the transcripts under
// ~/.claude/projects/<project-slug>/<session-id>.jsonl. One file is one
// session, including any subagent turns it spawned.
func claudeTranscriptPaths(u userHome, cutoff time.Time) []string {
	projectsDir := filepath.Join(u.HomeDir, ".claude", "projects")

	var paths []string
	for _, project := range subdirs(projectsDir) {
		matches, err := filepath.Glob(filepath.Join(projectsDir, project, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, path := range matches {
			if !isStale(path, cutoff) {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

// collectClaudeCode summarizes every Claude Code session for one user.
func collectClaudeCode(u userHome, cutoff time.Time) []Session {
	var sessions []Session
	for _, path := range claudeTranscriptPaths(u, cutoff) {
		if s, ok := parseClaudeTranscript(path, u.Username); ok {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// parseClaudeTranscript summarizes a single transcript file. It reports false
// when the file yielded no usable records.
func parseClaudeTranscript(path, username string) (Session, bool) {
	size, modTime := fileInfo(path)

	s := Session{
		Agent:           agentClaudeCode,
		Username:        username,
		SessionID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		TranscriptPath:  path,
		TranscriptBytes: size,
	}

	var sawRecord bool
	err := scanJSONL(path, func(rec *claudeRecord) {
		sawRecord = true
		s.observe(parseTime(rec.Timestamp))

		// The first directory seen is the one the session was launched in. A
		// later record can report a subdirectory the agent happened to change
		// into, which is not the project.
		if rec.CWD != "" && s.ProjectPath == "" {
			s.ProjectPath = rec.CWD
		}
		// Branch and version take the last value instead, so a session that
		// switched branches or survived an upgrade reports where it ended up.
		if rec.GitBranch != "" {
			s.GitBranch = rec.GitBranch
		}
		if rec.Version != "" {
			s.AgentVersion = rec.Version
		}

		switch rec.Type {
		case "user":
			// Turns carrying only tool results back to the model are part of
			// the agent loop, not something the person typed.
			if rec.Message.Content.hasOtherThan("tool_result") {
				s.UserMessages++
			}
		case "assistant":
			s.AssistantMessages++
			s.ToolCalls += rec.Message.Content.count("tool_use")
			if model := rec.Message.Model; model != "" && !strings.HasPrefix(model, "<") {
				s.Model = model
			}
			usage := rec.Message.Usage
			s.InputTokens += usage.InputTokens
			s.OutputTokens += usage.OutputTokens
			s.CacheReadTokens += usage.CacheReadInputTokens
			s.CacheWriteTokens += usage.CacheCreationInputTokens
		}
	})
	if err != nil || !sawRecord {
		return Session{}, false
	}

	// Transcripts written without timestamps still have a last-write time.
	if s.EndedAt.IsZero() {
		s.EndedAt = modTime
	}
	return s, true
}

// claudeMessageRecord is one transcript line, read for its content rather
// than only its shape.
type claudeMessageRecord struct {
	Type      string   `json:"type"`
	Timestamp string   `json:"timestamp"`
	Content   flexText `json:"content"` // top level, on "system" records
	Message   struct {
		Role    string      `json:"role"`
		Model   string      `json:"model"`
		Content richContent `json:"content"`
	} `json:"message"`
}

// discoverClaudeTranscripts lists this user's Claude Code sessions.
func discoverClaudeTranscripts(u userHome, cutoff time.Time) []transcriptRef {
	paths := claudeTranscriptPaths(u, cutoff)

	refs := make([]transcriptRef, 0, len(paths))
	for _, path := range paths {
		_, modTime := fileInfo(path)
		refs = append(refs, transcriptRef{
			Agent:     agentClaudeCode,
			SessionID: strings.TrimSuffix(filepath.Base(path), ".jsonl"),
			Username:  u.Username,
			Path:      path,
			ModTime:   modTime,
			Parse:     parseClaudeMessages,
		})
	}
	return refs
}

// parseClaudeMessages reads the conversation out of one transcript.
func parseClaudeMessages(ref transcriptRef, remaining, maxContentBytes int) []Message {
	b := newBlockBuilder(ref, remaining, maxContentBytes)

	_ = scanJSONL(ref.Path, func(rec *claudeMessageRecord) {
		if b.full() {
			return
		}
		ts := parseTime(rec.Timestamp)

		switch rec.Type {
		case "user":
			b.nextMessage()
			b.addBlocks(contentRole(roleUser, rec.Message.Content), rec.Message.Content, ts, "")
		case "assistant":
			model := rec.Message.Model
			if strings.HasPrefix(model, "<") {
				model = ""
			}
			b.nextMessage()
			b.addBlocks(roleAssistant, rec.Message.Content, ts, model)
		case "system":
			b.nextMessage()
			b.add(roleSystem, blockText, "", string(rec.Content), ts, "")
		}
		// Bookkeeping records - attachments, queued prompts, generated titles -
		// are not part of the conversation and are skipped.
	})

	return b.messages
}

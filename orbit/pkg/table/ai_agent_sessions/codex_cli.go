package ai_agent_sessions

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// codexRecord is one line of an OpenAI Codex CLI rollout file.
//
// Codex has used two layouts. Current rollouts wrap everything in a typed
// envelope ("session_meta", "turn_context", "response_item", "event_msg") with
// the detail under "payload"; older ones wrote the session header and the
// response items at the top level. Both are declared here and disambiguated in
// parseCodexRollout.
type codexRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`

	// Older flat layout.
	ID      string        `json:"id"`
	Role    string        `json:"role"`
	Content contentBlocks `json:"content"`

	Payload struct {
		Type       string        `json:"type"`
		ID         string        `json:"id"`
		Timestamp  string        `json:"timestamp"`
		CWD        string        `json:"cwd"`
		CLIVersion string        `json:"cli_version"`
		Model      string        `json:"model"`
		Role       string        `json:"role"`
		Content    contentBlocks `json:"content"`
		Git        struct {
			Branch        string `json:"branch"`
			RepositoryURL string `json:"repository_url"`
		} `json:"git"`
		Info struct {
			TotalTokenUsage struct {
				InputTokens       int64 `json:"input_tokens"`
				CachedInputTokens int64 `json:"cached_input_tokens"`
				OutputTokens      int64 `json:"output_tokens"`
			} `json:"total_token_usage"`
		} `json:"info"`
	} `json:"payload"`
}

// codexEnvelopeTypes are the record types of the current rollout layout.
var codexEnvelopeTypes = map[string]struct{}{
	"session_meta": {}, "turn_context": {},
	"response_item": {}, "event_msg": {},
	"compacted": {}, "turn_aborted": {},
}

// codexRolloutID pulls the session UUID out of a rollout filename such as
// rollout-2025-08-01T12-00-00-<uuid>.jsonl.
var codexRolloutID = regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)

// collectCodexCLI reads Codex CLI rollouts from ~/.codex/sessions, which are
// filed under a YYYY/MM/DD directory tree.
func collectCodexCLI(u userHome, cutoff time.Time) []Session {
	sessionsDir := filepath.Join(u.HomeDir, ".codex", "sessions")
	if info, err := os.Stat(sessionsDir); err != nil || !info.IsDir() {
		return nil
	}

	var sessions []Session
	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable subtree should not abort the rest of the walk.
			return nil //nolint:nilerr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		if isStale(path, cutoff) {
			return nil
		}
		if s, ok := parseCodexRollout(path, u.Username); ok {
			sessions = append(sessions, s)
		}
		return nil
	})
	return sessions
}

// parseCodexRollout summarizes one rollout file.
func parseCodexRollout(path, username string) (Session, bool) {
	size, modTime := fileInfo(path)

	s := Session{
		Agent:           agentCodexCLI,
		Username:        username,
		TranscriptPath:  path,
		TranscriptBytes: size,
	}
	if match := codexRolloutID.FindString(filepath.Base(path)); match != "" {
		s.SessionID = match
	}

	var sawRecord bool
	err := scanJSONL(path, func(rec *codexRecord) {
		sawRecord = true
		s.observe(parseTime(rec.Timestamp))

		if _, ok := codexEnvelopeTypes[rec.Type]; !ok {
			// Flat layout: the record is the item itself.
			if rec.ID != "" && s.SessionID == "" {
				s.SessionID = rec.ID
			}
			countCodexItem(&s, rec.Type, rec.Role, rec.Content)
			return
		}

		p := rec.Payload
		switch rec.Type {
		case "session_meta":
			if p.ID != "" {
				s.SessionID = p.ID
			}
			s.ProjectPath = p.CWD
			s.AgentVersion = p.CLIVersion
			s.GitBranch = p.Git.Branch
			s.GitRepository = p.Git.RepositoryURL
			s.observe(parseTime(p.Timestamp))
		case "turn_context":
			// Turn context can move the working directory or switch models
			// mid-session; the last one wins.
			if p.CWD != "" {
				s.ProjectPath = p.CWD
			}
			if p.Model != "" {
				s.Model = p.Model
			}
		case "response_item":
			countCodexItem(&s, p.Type, p.Role, p.Content)
		case "event_msg":
			if p.Type == "token_count" {
				// These totals are cumulative for the session, so the last
				// event replaces rather than adds to what came before.
				usage := p.Info.TotalTokenUsage
				s.InputTokens = usage.InputTokens
				s.OutputTokens = usage.OutputTokens
				s.CacheReadTokens = usage.CachedInputTokens
			}
		}
	})
	if err != nil || !sawRecord {
		return Session{}, false
	}

	if s.EndedAt.IsZero() {
		s.EndedAt = modTime
	}
	return s, true
}

// countCodexItem tallies a single conversation item.
func countCodexItem(s *Session, itemType, role string, content contentBlocks) {
	switch itemType {
	case "message":
		switch role {
		case "user":
			s.UserMessages++
		case "assistant":
			s.AssistantMessages++
		}
	case "function_call", "local_shell_call", "custom_tool_call", "web_search_call":
		s.ToolCalls++
	}
	// Some rollouts inline tool calls as content blocks instead of items.
	s.ToolCalls += content.count("tool_use")
}

// codexMessageRecord is one rollout line, read for its content.
type codexMessageRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`

	// Older flat layout.
	Role    string          `json:"role"`
	Content richContent     `json:"content"`
	Name    string          `json:"name"`
	Args    json.RawMessage `json:"arguments"`

	Payload struct {
		Type    string          `json:"type"`
		Role    string          `json:"role"`
		Model   string          `json:"model"`
		Content richContent     `json:"content"`
		Name    string          `json:"name"`
		Args    json.RawMessage `json:"arguments"`
		Output  flexText        `json:"output"`
	} `json:"payload"`
}

// discoverCodexTranscripts lists this user's Codex CLI rollouts.
func discoverCodexTranscripts(u userHome, cutoff time.Time) []transcriptRef {
	sessionsDir := filepath.Join(u.HomeDir, ".codex", "sessions")
	if info, err := os.Stat(sessionsDir); err != nil || !info.IsDir() {
		return nil
	}

	var refs []transcriptRef
	_ = filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") || isStale(path, cutoff) {
			return nil
		}
		_, modTime := fileInfo(path)
		refs = append(refs, transcriptRef{
			Agent:     agentCodexCLI,
			SessionID: codexRolloutID.FindString(d.Name()),
			Username:  u.Username,
			Path:      path,
			ModTime:   modTime,
			Parse:     parseCodexMessages,
		})
		return nil
	})
	return refs
}

// parseCodexMessages reads the conversation out of one rollout.
func parseCodexMessages(ref transcriptRef, remaining, maxContentBytes int) []Message {
	b := newBlockBuilder(ref, remaining, maxContentBytes)

	var model string
	_ = scanJSONL(ref.Path, func(rec *codexMessageRecord) {
		if b.full() {
			return
		}
		ts := parseTime(rec.Timestamp)

		if _, ok := codexEnvelopeTypes[rec.Type]; !ok {
			// Flat layout: the record is the item itself.
			addCodexItem(b, rec.Type, rec.Role, rec.Content, rec.Name, string(rec.Args), "", ts, model)
			return
		}

		p := rec.Payload
		switch rec.Type {
		case "turn_context":
			if p.Model != "" {
				model = p.Model
			}
		case "response_item":
			addCodexItem(b, p.Type, p.Role, p.Content, p.Name, string(p.Args), string(p.Output), ts, model)
		}
	})

	return b.messages
}

// addCodexItem emits the rows for a single conversation item.
func addCodexItem(b *blockBuilder, itemType, role string, content richContent, name, args, output string, ts time.Time, model string) {
	switch itemType {
	case "message":
		if role == "" {
			return
		}
		b.nextMessage()
		b.addBlocks(role, content, ts, model)
	case "function_call", "local_shell_call", "custom_tool_call":
		b.nextMessage()
		b.add(roleAssistant, blockToolUse, name, args, ts, model)
	case "function_call_output", "custom_tool_call_output":
		b.nextMessage()
		b.add(roleSystem, blockToolResult, name, output, ts, model)
	}
}

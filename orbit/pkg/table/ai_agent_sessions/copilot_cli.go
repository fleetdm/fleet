package ai_agent_sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// copilotTokenCount is the shape Copilot uses for each token bucket.
type copilotTokenCount struct {
	TokenCount int64 `json:"tokenCount"`
}

// copilotRecord is one event from a Copilot CLI events.jsonl. The Data field
// is a union across event types; every member is optional.
type copilotRecord struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      struct {
		// session.start
		SessionID      string `json:"sessionId"`
		CopilotVersion string `json:"copilotVersion"`
		StartTime      string `json:"startTime"`
		SelectedModel  string `json:"selectedModel"`
		Context        struct {
			CWD        string `json:"cwd"`
			Branch     string `json:"branch"`
			Repository string `json:"repository"`
		} `json:"context"`

		// assistant.message
		Model string `json:"model"`

		// session.shutdown
		CurrentModel string `json:"currentModel"`
		TokenDetails struct {
			Input      copilotTokenCount `json:"input"`
			CacheRead  copilotTokenCount `json:"cache_read"`
			CacheWrite copilotTokenCount `json:"cache_write"`
			Output     copilotTokenCount `json:"output"`
		} `json:"tokenDetails"`
	} `json:"data"`
}

// collectCopilotCLI reads GitHub Copilot CLI sessions from
// ~/.copilot/session-state/<session-id>/events.jsonl.
func collectCopilotCLI(u userHome, cutoff time.Time) []Session {
	stateDir := filepath.Join(u.HomeDir, ".copilot", "session-state")

	var sessions []Session
	for _, sessionID := range subdirs(stateDir) {
		path := filepath.Join(stateDir, sessionID, "events.jsonl")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if isStale(path, cutoff) {
			continue
		}
		if s, ok := parseCopilotEvents(path, sessionID, u.Username); ok {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

// parseCopilotEvents summarizes one session's event log.
func parseCopilotEvents(path, sessionID, username string) (Session, bool) {
	size, modTime := fileInfo(path)

	s := Session{
		Agent:           agentCopilotCLI,
		Username:        username,
		SessionID:       sessionID,
		TranscriptPath:  path,
		TranscriptBytes: size,
	}

	// Copilot reports totals once at shutdown. A session killed before it could
	// write that event leaves the token columns at zero.
	var lastAssistantModel, shutdownModel, selectedModel string

	var sawRecord bool
	err := scanJSONL(path, func(rec *copilotRecord) {
		sawRecord = true
		s.observe(parseTime(rec.Timestamp))

		switch rec.Type {
		case "session.start":
			if rec.Data.SessionID != "" {
				s.SessionID = rec.Data.SessionID
			}
			s.AgentVersion = rec.Data.CopilotVersion
			s.ProjectPath = rec.Data.Context.CWD
			s.GitBranch = rec.Data.Context.Branch
			s.GitRepository = rec.Data.Context.Repository
			selectedModel = rec.Data.SelectedModel
			s.observe(parseTime(rec.Data.StartTime))
		case "user.message":
			s.UserMessages++
		case "assistant.message":
			s.AssistantMessages++
			if rec.Data.Model != "" {
				lastAssistantModel = rec.Data.Model
			}
		case "session.shutdown":
			shutdownModel = rec.Data.CurrentModel
			details := rec.Data.TokenDetails
			s.InputTokens = details.Input.TokenCount
			s.OutputTokens = details.Output.TokenCount
			s.CacheReadTokens = details.CacheRead.TokenCount
			s.CacheWriteTokens = details.CacheWrite.TokenCount
		default:
			if isCopilotToolEvent(rec.Type) {
				s.ToolCalls++
			}
		}
	})
	if err != nil || !sawRecord {
		return Session{}, false
	}

	// "auto" is a routing preference rather than a model, so prefer whichever
	// model actually served the session.
	s.Model = firstNonEmpty(shutdownModel, lastAssistantModel, selectedModel)

	if s.EndedAt.IsZero() {
		s.EndedAt = modTime
	}
	return s, true
}

// isCopilotToolEvent matches the event types Copilot emits when the model
// invokes a tool. Copilot has used more than one naming scheme for these, so
// this matches on substring rather than an exact set.
func isCopilotToolEvent(eventType string) bool {
	if !strings.Contains(eventType, "tool") {
		return false
	}
	// Only count the invocation, not the matching result or approval events.
	return strings.HasSuffix(eventType, "tool_call") ||
		strings.HasSuffix(eventType, "tool_use") ||
		eventType == "tool.start"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// copilotMessageRecord is one event, read for its content.
type copilotMessageRecord struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      struct {
		Content  flexText        `json:"content"`
		Model    string          `json:"model"`
		Name     string          `json:"name"`
		ToolName string          `json:"toolName"`
		Input    json.RawMessage `json:"input"`
		Result   flexText        `json:"result"`
	} `json:"data"`
}

// discoverCopilotTranscripts lists this user's Copilot CLI sessions.
func discoverCopilotTranscripts(u userHome, cutoff time.Time) []transcriptRef {
	stateDir := filepath.Join(u.HomeDir, ".copilot", "session-state")

	var refs []transcriptRef
	for _, sessionID := range subdirs(stateDir) {
		path := filepath.Join(stateDir, sessionID, "events.jsonl")
		if _, err := os.Stat(path); err != nil || isStale(path, cutoff) {
			continue
		}
		_, modTime := fileInfo(path)
		refs = append(refs, transcriptRef{
			Agent:     agentCopilotCLI,
			SessionID: sessionID,
			Username:  u.Username,
			Path:      path,
			ModTime:   modTime,
			Parse:     parseCopilotMessages,
		})
	}
	return refs
}

// parseCopilotMessages reads the conversation out of one event log. Copilot
// writes one flat string per turn rather than typed blocks, so each turn
// yields a single text row.
func parseCopilotMessages(ref transcriptRef, remaining, maxContentBytes int) []Message {
	b := newBlockBuilder(ref, remaining, maxContentBytes)

	var model string
	_ = scanJSONL(ref.Path, func(rec *copilotMessageRecord) {
		if b.full() {
			return
		}
		if rec.Data.Model != "" {
			model = rec.Data.Model
		}
		ts := parseTime(rec.Timestamp)

		switch {
		case rec.Type == "system.message":
			b.nextMessage()
			b.add(roleSystem, blockText, "", string(rec.Data.Content), ts, "")
		case rec.Type == "user.message":
			b.nextMessage()
			b.add(roleUser, blockText, "", string(rec.Data.Content), ts, "")
		case rec.Type == "assistant.message":
			b.nextMessage()
			b.add(roleAssistant, blockText, "", string(rec.Data.Content), ts, model)
		case isCopilotToolEvent(rec.Type):
			b.nextMessage()
			b.add(roleAssistant, blockToolUse, firstNonEmpty(rec.Data.Name, rec.Data.ToolName),
				string(rec.Data.Input), ts, model)
		case strings.HasSuffix(rec.Type, "tool_call_result"):
			b.nextMessage()
			b.add(roleSystem, blockToolResult, firstNonEmpty(rec.Data.Name, rec.Data.ToolName),
				string(rec.Data.Result), ts, model)
		}
	})

	return b.messages
}

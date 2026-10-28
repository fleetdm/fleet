package ai_agent_sessions

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// claudeTranscript is a trimmed-down transcript in the shape Claude Code
// writes: a user prompt, an assistant turn with two tool calls, the tool
// results coming back, and a final assistant reply.
var claudeTranscript = strings.Join([]string{
	`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-08-11T13:27:08.788Z","sessionId":"5d1f26db","content":"..."}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:10.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"role":"user","content":"fix the build"}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:20.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"role":"assistant","model":"claude-opus-5","content":[{"type":"text","text":"Looking."},{"type":"tool_use","name":"Read"},{"type":"tool_use","name":"Bash"}],"usage":{"input_tokens":10,"output_tokens":248,"cache_read_input_tokens":30802,"cache_creation_input_tokens":22539}}}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:25.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"role":"user","content":[{"type":"tool_result","content":"file contents"}]}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:28:10.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"allenhouchins/fix","message":{"role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"Done."}],"usage":{"input_tokens":5,"output_tokens":100,"cache_read_input_tokens":1000,"cache_creation_input_tokens":0}}}`,
}, "\n") + "\n"

func TestCollectClaudeCode(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet", "5d1f26db.jsonl"), claudeTranscript)

	sessions := collectClaudeCode(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"Agent", s.Agent, agentClaudeCode},
		{"SessionID", s.SessionID, "5d1f26db"},
		{"Username", s.Username, "alice"},
		{"ProjectPath", s.ProjectPath, "/Users/alice/GitHub/fleet"},
		// The last record wins, so a mid-session branch switch is reflected.
		{"GitBranch", s.GitBranch, "allenhouchins/fix"},
		{"AgentVersion", s.AgentVersion, "2.1.222"},
		// "<synthetic>" is not a real model and must not overwrite the last
		// genuine one.
		{"Model", s.Model, "claude-opus-5"},
		// Only the typed prompt counts; the tool_result turn does not.
		{"UserMessages", s.UserMessages, int64(1)},
		{"AssistantMessages", s.AssistantMessages, int64(2)},
		{"ToolCalls", s.ToolCalls, int64(2)},
		{"InputTokens", s.InputTokens, int64(15)},
		{"OutputTokens", s.OutputTokens, int64(348)},
		{"CacheReadTokens", s.CacheReadTokens, int64(31802)},
		{"CacheWriteTokens", s.CacheWriteTokens, int64(22539)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}

	// The range spans the earliest and latest timestamps in the file, which
	// includes the queue-operation record ahead of the first prompt.
	wantStart := time.Date(2026, 8, 11, 13, 27, 8, 788000000, time.UTC)
	wantEnd := time.Date(2026, 8, 11, 13, 28, 10, 0, time.UTC)
	if !s.StartedAt.Equal(wantStart) {
		t.Errorf("StartedAt = %v, want %v", s.StartedAt, wantStart)
	}
	if !s.EndedAt.Equal(wantEnd) {
		t.Errorf("EndedAt = %v, want %v", s.EndedAt, wantEnd)
	}
	if s.durationSeconds() != 61 {
		t.Errorf("durationSeconds() = %d, want 61", s.durationSeconds())
	}
	if s.TranscriptBytes != int64(len(claudeTranscript)) {
		t.Errorf("TranscriptBytes = %d, want %d", s.TranscriptBytes, len(claudeTranscript))
	}
}

func TestCollectClaudeCodeSkipsStaleAndEmptyTranscripts(t *testing.T) {
	home := t.TempDir()
	projects := filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet")
	writeFile(t, filepath.Join(projects, "5d1f26db.jsonl"), claudeTranscript)
	writeFile(t, filepath.Join(projects, "empty.jsonl"), "")
	writeFile(t, filepath.Join(projects, "notes.txt"), claudeTranscript)

	u := userHome{Username: "alice", HomeDir: home}

	// An empty transcript and a non-.jsonl file both contribute nothing.
	if got := len(collectClaudeCode(u, time.Time{})); got != 1 {
		t.Errorf("got %d sessions, want 1", got)
	}
	// Everything predates a cutoff in the future.
	if got := len(collectClaudeCode(u, time.Now().Add(time.Hour))); got != 0 {
		t.Errorf("got %d sessions past the cutoff, want 0", got)
	}
}

// copilotEvents is the event sequence a short Copilot CLI session produces.
var copilotEvents = strings.Join([]string{
	`{"type":"session.start","data":{"sessionId":"24b9ca9f","version":1,"producer":"copilot-agent","copilotVersion":"1.0.65","startTime":"2026-07-06T02:36:32.043Z","selectedModel":"auto","context":{"cwd":"/Users/alice/GitHub/fleet","gitRoot":"/Users/alice/GitHub/fleet","branch":"dogfood","repository":"fleetdm/fleet"}},"id":"4191c26f","timestamp":"2026-07-06T02:36:32.058Z"}`,
	`{"type":"system.message","data":{"role":"system","content":"You are the GitHub Copilot CLI"},"timestamp":"2026-07-06T02:36:32.100Z"}`,
	`{"type":"user.message","data":{"content":"summarize this diff"},"timestamp":"2026-07-06T02:36:32.200Z"}`,
	`{"type":"assistant.turn_start","data":{"turnId":"0"},"timestamp":"2026-07-06T02:36:32.742Z"}`,
	`{"type":"assistant.tool_call","data":{"name":"str_replace_editor"},"timestamp":"2026-07-06T02:36:33.000Z"}`,
	`{"type":"assistant.tool_call_result","data":{"name":"str_replace_editor"},"timestamp":"2026-07-06T02:36:33.500Z"}`,
	`{"type":"assistant.message","data":{"messageId":"a2312369","model":"claude-haiku-4.5","content":"Here is the summary."},"timestamp":"2026-07-06T02:36:36.000Z"}`,
	`{"type":"assistant.turn_end","data":{"turnId":"0"},"timestamp":"2026-07-06T02:36:37.382Z"}`,
	`{"type":"session.shutdown","data":{"shutdownType":"routine","totalPremiumRequests":0.33,"tokenDetails":{"input":{"tokenCount":9},"cache_read":{"tokenCount":0},"cache_write":{"tokenCount":21105},"output":{"tokenCount":355}},"currentModel":"claude-haiku-4.5"},"timestamp":"2026-07-06T02:36:37.407Z"}`,
}, "\n") + "\n"

func TestCollectCopilotCLI(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "24b9ca9f", "events.jsonl"), copilotEvents)

	sessions := collectCopilotCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"Agent", s.Agent, agentCopilotCLI},
		{"SessionID", s.SessionID, "24b9ca9f"},
		{"ProjectPath", s.ProjectPath, "/Users/alice/GitHub/fleet"},
		{"GitBranch", s.GitBranch, "dogfood"},
		{"GitRepository", s.GitRepository, "fleetdm/fleet"},
		{"AgentVersion", s.AgentVersion, "1.0.65"},
		// "auto" is a routing preference, so the model that actually ran wins.
		{"Model", s.Model, "claude-haiku-4.5"},
		{"UserMessages", s.UserMessages, int64(1)},
		{"AssistantMessages", s.AssistantMessages, int64(1)},
		// The invocation counts; the matching result event does not.
		{"ToolCalls", s.ToolCalls, int64(1)},
		// tokenDetails separates real input from cache, unlike modelMetrics.
		{"InputTokens", s.InputTokens, int64(9)},
		{"OutputTokens", s.OutputTokens, int64(355)},
		{"CacheReadTokens", s.CacheReadTokens, int64(0)},
		{"CacheWriteTokens", s.CacheWriteTokens, int64(21105)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}

	if s.durationSeconds() != 5 {
		t.Errorf("durationSeconds() = %d, want 5", s.durationSeconds())
	}
}

func TestCollectCopilotCLIWithoutShutdownEvent(t *testing.T) {
	home := t.TempDir()
	// A session killed mid-flight never writes session.shutdown.
	truncated := strings.Join(strings.Split(strings.TrimSpace(copilotEvents), "\n")[:7], "\n") + "\n"
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "24b9ca9f", "events.jsonl"), truncated)

	sessions := collectCopilotCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	// Token totals are unavailable, but the model still comes from the
	// assistant turn rather than falling back to "auto".
	if s.Model != "claude-haiku-4.5" {
		t.Errorf("Model = %q, want %q", s.Model, "claude-haiku-4.5")
	}
	if s.InputTokens != 0 || s.OutputTokens != 0 {
		t.Errorf("tokens = (%d, %d), want (0, 0)", s.InputTokens, s.OutputTokens)
	}
}

func TestIsCopilotToolEvent(t *testing.T) {
	tests := map[string]bool{
		"assistant.tool_call":        true,
		"tool.start":                 true,
		"assistant.tool_use":         true,
		"assistant.tool_call_result": false,
		"assistant.message":          false,
		"session.start":              false,
	}
	for eventType, want := range tests {
		if got := isCopilotToolEvent(eventType); got != want {
			t.Errorf("isCopilotToolEvent(%q) = %v, want %v", eventType, got, want)
		}
	}
}

func TestCollectCursor(t *testing.T) {
	home := t.TempDir()
	transcript := strings.Join([]string{
		`{"role":"user","message":{"content":[{"type":"text","text":"<user_query>fix this</user_query>"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"text","text":"Reading the script."},{"type":"tool_use","name":"Read","input":{}}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"WebFetch","input":{}}]}}`,
		`{"role":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
	}, "\n") + "\n"

	sessionID := "010bfa5d-28b2-4862-8867-b4108668e138"
	writeFile(t, filepath.Join(home, ".cursor", "projects", "tmp-fleet", "agent-transcripts", sessionID, sessionID+".jsonl"), transcript)

	sessions := collectCursor(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	if s.Agent != agentCursor {
		t.Errorf("Agent = %q, want %q", s.Agent, agentCursor)
	}
	if s.SessionID != sessionID {
		t.Errorf("SessionID = %q, want %q", s.SessionID, sessionID)
	}
	if s.UserMessages != 1 {
		t.Errorf("UserMessages = %d, want 1", s.UserMessages)
	}
	if s.AssistantMessages != 2 {
		t.Errorf("AssistantMessages = %d, want 2", s.AssistantMessages)
	}
	if s.ToolCalls != 2 {
		t.Errorf("ToolCalls = %d, want 2", s.ToolCalls)
	}
	// Cursor transcripts carry no timestamps, so only the file's last write
	// time is known.
	if !s.StartedAt.IsZero() {
		t.Errorf("StartedAt = %v, want zero", s.StartedAt)
	}
	if s.EndedAt.IsZero() {
		t.Error("EndedAt should fall back to the transcript's modification time")
	}
	if s.Model != "" || s.InputTokens != 0 {
		t.Errorf("Cursor reports no model or token usage, got %q / %d", s.Model, s.InputTokens)
	}
}

func TestCollectCursorSkipsSessionsWithoutATranscript(t *testing.T) {
	home := t.TempDir()
	// The session directory exists but the .jsonl inside it does not.
	writeFile(t, filepath.Join(home, ".cursor", "projects", "tmp-fleet", "agent-transcripts", "abc", "other.txt"), "x")

	if got := len(collectCursor(userHome{Username: "alice", HomeDir: home}, time.Time{})); got != 0 {
		t.Errorf("got %d sessions, want 0", got)
	}
}

func TestCollectCodexCLI(t *testing.T) {
	home := t.TempDir()
	rollout := strings.Join([]string{
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"7f3c1a2b","timestamp":"2026-08-01T12:00:00.000Z","cwd":"/Users/alice/GitHub/fleet","originator":"codex_cli_rs","cli_version":"0.20.0","git":{"branch":"main","repository_url":"https://github.com/fleetdm/fleet"}}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"turn_context","payload":{"cwd":"/Users/alice/GitHub/fleet","model":"gpt-5","effort":"medium"}}`,
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"run the tests"}]}}`,
		`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","call_id":"c1"}}`,
		`{"timestamp":"2026-08-01T12:00:04.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c1"}}`,
		`{"timestamp":"2026-08-01T12:00:05.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Tests pass."}]}}`,
		`{"timestamp":"2026-08-01T12:00:05.500Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":20}}}}`,
		`{"timestamp":"2026-08-01T12:00:30.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":250,"cached_input_tokens":90,"output_tokens":75}}}}`,
	}, "\n") + "\n"

	writeFile(t, filepath.Join(home, ".codex", "sessions", "2026", "08", "01",
		"rollout-2026-08-01T12-00-00-7f3c1a2b-1111-2222-3333-444455556666.jsonl"), rollout)

	sessions := collectCodexCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"Agent", s.Agent, agentCodexCLI},
		// session_meta's id takes precedence over the one in the filename.
		{"SessionID", s.SessionID, "7f3c1a2b"},
		{"ProjectPath", s.ProjectPath, "/Users/alice/GitHub/fleet"},
		{"GitBranch", s.GitBranch, "main"},
		{"GitRepository", s.GitRepository, "https://github.com/fleetdm/fleet"},
		{"AgentVersion", s.AgentVersion, "0.20.0"},
		{"Model", s.Model, "gpt-5"},
		{"UserMessages", s.UserMessages, int64(1)},
		{"AssistantMessages", s.AssistantMessages, int64(1)},
		{"ToolCalls", s.ToolCalls, int64(1)},
		// token_count totals are cumulative, so the last event replaces the
		// earlier one rather than adding to it.
		{"InputTokens", s.InputTokens, int64(250)},
		{"OutputTokens", s.OutputTokens, int64(75)},
		{"CacheReadTokens", s.CacheReadTokens, int64(90)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}
}

func TestCollectCodexCLIFlatLayout(t *testing.T) {
	home := t.TempDir()
	// Older rollouts wrote the header and items at the top level.
	rollout := strings.Join([]string{
		`{"id":"9a8b7c6d","timestamp":"2026-08-01T12:00:00.000Z","instructions":"..."}`,
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}`,
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`,
		`{"type":"function_call","name":"shell","call_id":"c1"}`,
	}, "\n") + "\n"

	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-08-01T12-00-00-old.jsonl"), rollout)

	sessions := collectCodexCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	s := sessions[0]

	if s.SessionID != "9a8b7c6d" {
		t.Errorf("SessionID = %q, want %q", s.SessionID, "9a8b7c6d")
	}
	if s.UserMessages != 1 || s.AssistantMessages != 1 || s.ToolCalls != 1 {
		t.Errorf("counts = (%d, %d, %d), want (1, 1, 1)", s.UserMessages, s.AssistantMessages, s.ToolCalls)
	}
	// No file-level timestamps on the items, so the fallback fills in the end.
	if s.EndedAt.IsZero() {
		t.Error("EndedAt should be set")
	}
}

func TestCollectGeminiCLI(t *testing.T) {
	home := t.TempDir()
	// One project log covering two interleaved sessions.
	logs := `[
	  {"sessionId":"aaa","messageId":0,"timestamp":"2026-08-01T10:00:00.000Z","type":"user","message":"first"},
	  {"sessionId":"bbb","messageId":0,"timestamp":"2026-08-02T10:00:00.000Z","type":"user","message":"other project"},
	  {"sessionId":"aaa","messageId":1,"timestamp":"2026-08-01T10:05:00.000Z","type":"user","message":"second"},
	  {"sessionId":"","messageId":0,"timestamp":"2026-08-01T10:06:00.000Z","type":"user","message":"no session id"}
	]`
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "d41d8cd9", "logs.json"), logs)

	sessions := collectGeminiCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}

	byID := map[string]Session{}
	for _, s := range sessions {
		byID[s.SessionID] = s
	}

	first, ok := byID["aaa"]
	if !ok {
		t.Fatal("session aaa is missing")
	}
	if first.Agent != agentGeminiCLI {
		t.Errorf("Agent = %q, want %q", first.Agent, agentGeminiCLI)
	}
	if first.UserMessages != 2 {
		t.Errorf("UserMessages = %d, want 2", first.UserMessages)
	}
	if first.durationSeconds() != 300 {
		t.Errorf("durationSeconds() = %d, want 300", first.durationSeconds())
	}
	// Gemini hashes the project path, so it cannot be recovered.
	if first.ProjectPath != "" {
		t.Errorf("ProjectPath = %q, want empty", first.ProjectPath)
	}
}

func TestCollectGeminiCLIMalformedLog(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "d41d8cd9", "logs.json"), `{"not":"an array"}`)

	if got := len(collectGeminiCLI(userHome{Username: "alice", HomeDir: home}, time.Time{})); got != 0 {
		t.Errorf("got %d sessions from a malformed log, want 0", got)
	}
}

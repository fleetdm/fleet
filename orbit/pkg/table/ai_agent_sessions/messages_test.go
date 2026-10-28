package ai_agent_sessions

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

func TestRichContentUnmarshal(t *testing.T) {
	t.Run("plain string body", func(t *testing.T) {
		var c richContent
		if err := json.Unmarshal([]byte(`"fix the build"`), &c); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if !c.isPlainText() || c.Text != "fix the build" {
			t.Errorf("got %+v, want plain text %q", c, "fix the build")
		}
	})

	t.Run("typed blocks", func(t *testing.T) {
		var c richContent
		body := `[{"type":"text","text":"Looking."},{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]`
		if err := json.Unmarshal([]byte(body), &c); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if c.isPlainText() || len(c.Blocks) != 2 {
			t.Fatalf("got %+v, want 2 blocks", c)
		}
		if c.Blocks[0].Text != "Looking." {
			t.Errorf("block 0 text = %q", c.Blocks[0].Text)
		}
		if c.Blocks[1].Name != "Bash" || string(c.Blocks[1].Input) != `{"command":"ls"}` {
			t.Errorf("block 1 = %+v", c.Blocks[1])
		}
	})

	t.Run("unexpected shape", func(t *testing.T) {
		var c richContent
		if err := json.Unmarshal([]byte(`{"type":"text"}`), &c); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if c.isPlainText() || len(c.Blocks) != 0 {
			t.Errorf("got %+v, want empty", c)
		}
	})
}

func TestFlexTextUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"string", `"plain output"`, "plain output"},
		{"array of text blocks", `[{"type":"text","text":"line one"},{"type":"text","text":"line two"}]`, "line one\nline two"},
		{"empty array", `[]`, ""},
		{"unexpected object", `{"a":1}`, ""},
		{"null", `null`, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var f flexText
			if err := json.Unmarshal([]byte(tc.input), &f); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if string(f) != tc.want {
				t.Errorf("got %q, want %q", string(f), tc.want)
			}
		})
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		limit int
		want  string
	}{
		{"under the limit", "hello", 10, "hello"},
		{"exactly the limit", "hello", 5, "hello"},
		{"plain ascii", "hello world", 5, "hello"},
		// "é" is two bytes, so a cut at 3 must not split it.
		{"does not split a rune", "abé", 3, "ab"},
		{"multibyte at the boundary", "aé", 3, "aé"},
		{"emoji", "ab🙂", 4, "ab"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateUTF8(tc.input, tc.limit)
			if got != tc.want {
				t.Errorf("truncateUTF8(%q, %d) = %q, want %q", tc.input, tc.limit, got, tc.want)
			}
			if !isValidUTF8(got) {
				t.Errorf("result %q is not valid UTF-8", got)
			}
		})
	}
}

// isValidUTF8 checks the result never ends mid-rune.
func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

func TestContentRoleReclassifiesToolResultTurns(t *testing.T) {
	toolResultOnly := richContent{Blocks: []contentBlockDetail{{Type: "tool_result", Content: "output"}}}
	mixed := richContent{Blocks: []contentBlockDetail{
		{Type: "tool_result", Content: "output"},
		{Type: "text", Text: "and also this"},
	}}

	// A user turn carrying only tool results is machine output.
	if got := contentRole(roleUser, toolResultOnly); got != roleSystem {
		t.Errorf("tool-result-only user turn = %q, want %q", got, roleSystem)
	}
	if got := contentRole(roleUser, mixed); got != roleUser {
		t.Errorf("mixed user turn = %q, want %q", got, roleUser)
	}
	if got := contentRole(roleUser, richContent{Text: "typed this"}); got != roleUser {
		t.Errorf("plain text user turn = %q, want %q", got, roleUser)
	}
	// Assistant turns are never reclassified.
	if got := contentRole(roleAssistant, toolResultOnly); got != roleAssistant {
		t.Errorf("assistant turn = %q, want %q", got, roleAssistant)
	}
}

func TestBlockBuilderRespectsRowBudgetAndTruncation(t *testing.T) {
	ref := transcriptRef{Agent: agentClaudeCode, SessionID: "s1", Username: "alice", Path: "/tmp/s1.jsonl"}

	b := newBlockBuilder(ref, 2, 4)
	b.nextMessage()
	b.add(roleUser, blockText, "", "abcdefgh", time.Time{}, "")
	b.add(roleAssistant, blockText, "", "ok", time.Time{}, "")
	// Past the budget.
	b.add(roleAssistant, blockText, "", "dropped", time.Time{}, "")

	if len(b.messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(b.messages))
	}
	first := b.messages[0]
	if first.Content != "abcd" {
		t.Errorf("Content = %q, want %q", first.Content, "abcd")
	}
	// The pre-truncation length is preserved so the loss is measurable.
	if first.ContentLength != 8 {
		t.Errorf("ContentLength = %d, want 8", first.ContentLength)
	}
	if !first.Truncated {
		t.Error("Truncated should be true")
	}
	if b.messages[1].Truncated {
		t.Error("a short block should not be marked truncated")
	}
	if !b.full() {
		t.Error("builder should report full")
	}
}

func TestBlockBuilderSkipsEmptyBlocksAndIndexes(t *testing.T) {
	b := newBlockBuilder(transcriptRef{}, 0, 0)

	b.nextMessage()
	b.add(roleUser, blockText, "", "first", time.Time{}, "")
	b.add(roleUser, blockText, "", "", time.Time{}, "") // dropped: no text, no tool
	b.add(roleUser, blockText, "", "second", time.Time{}, "")
	b.nextMessage()
	b.add(roleAssistant, blockText, "", "reply", time.Time{}, "")

	if len(b.messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(b.messages))
	}
	want := []struct{ msg, blk int64 }{{0, 0}, {0, 1}, {1, 0}}
	for i, w := range want {
		if b.messages[i].MessageIndex != w.msg || b.messages[i].BlockIndex != w.blk {
			t.Errorf("message %d index = (%d, %d), want (%d, %d)",
				i, b.messages[i].MessageIndex, b.messages[i].BlockIndex, w.msg, w.blk)
		}
	}
}

func TestBlockBuilderKeepsToolCallWithEmptyInput(t *testing.T) {
	b := newBlockBuilder(transcriptRef{}, 0, 0)
	b.nextMessage()
	// A tool call with no arguments still matters; only fully empty blocks go.
	b.add(roleAssistant, blockToolUse, "ListFiles", "", time.Time{}, "")

	if len(b.messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(b.messages))
	}
	if b.messages[0].ToolName != "ListFiles" {
		t.Errorf("ToolName = %q, want %q", b.messages[0].ToolName, "ListFiles")
	}
}

func TestCollectClaudeMessages(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet", "5d1f26db.jsonl"), claudeTranscript)

	refs := discoverClaudeTranscripts(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(refs) != 1 {
		t.Fatalf("got %d transcripts, want 1", len(refs))
	}
	messages := refs[0].Parse(refs[0], 0, 0)

	want := []struct {
		role      string
		blockType string
		toolName  string
		content   string
	}{
		{roleUser, blockText, "", "fix the build"},
		{roleAssistant, blockText, "", "Looking."},
		{roleAssistant, blockToolUse, "Read", ""},
		{roleAssistant, blockToolUse, "Bash", ""},
		// The tool-result turn is machine output, not a typed prompt.
		{roleSystem, blockToolResult, "", "file contents"},
		{roleAssistant, blockText, "", "Done."},
	}
	if len(messages) != len(want) {
		for i, m := range messages {
			t.Logf("row %d: %s/%s %q", i, m.Role, m.BlockType, m.Content)
		}
		t.Fatalf("got %d messages, want %d", len(messages), len(want))
	}
	for i, w := range want {
		m := messages[i]
		if m.Role != w.role || m.BlockType != w.blockType || m.ToolName != w.toolName {
			t.Errorf("row %d = %s/%s/%s, want %s/%s/%s",
				i, m.Role, m.BlockType, m.ToolName, w.role, w.blockType, w.toolName)
		}
		if w.content != "" && m.Content != w.content {
			t.Errorf("row %d content = %q, want %q", i, m.Content, w.content)
		}
	}

	// The model is attributed to assistant rows only, and the synthetic
	// placeholder is not reported as a model.
	if messages[1].Model != "claude-opus-5" {
		t.Errorf("assistant model = %q, want %q", messages[1].Model, "claude-opus-5")
	}
	if messages[5].Model != "" {
		t.Errorf("synthetic model = %q, want empty", messages[5].Model)
	}
	if messages[0].Model != "" {
		t.Errorf("user row should carry no model, got %q", messages[0].Model)
	}
	if messages[0].SessionID != "5d1f26db" || messages[0].Username != "alice" {
		t.Errorf("identity = %s/%s", messages[0].SessionID, messages[0].Username)
	}
	if messages[0].Timestamp.IsZero() {
		t.Error("timestamp should be parsed from the record")
	}
}

func TestCollectCopilotMessages(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "24b9ca9f", "events.jsonl"), copilotEvents)

	refs := discoverCopilotTranscripts(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(refs) != 1 {
		t.Fatalf("got %d transcripts, want 1", len(refs))
	}
	messages := refs[0].Parse(refs[0], 0, 0)

	var roles []string
	for _, m := range messages {
		roles = append(roles, m.Role+"/"+m.BlockType)
	}
	want := []string{
		roleSystem + "/" + blockText,
		roleUser + "/" + blockText,
		roleAssistant + "/" + blockToolUse,
		roleSystem + "/" + blockToolResult,
		roleAssistant + "/" + blockText,
	}
	if strings.Join(roles, ",") != strings.Join(want, ",") {
		t.Fatalf("rows = %v, want %v", roles, want)
	}
	if messages[1].Content != "summarize this diff" {
		t.Errorf("user content = %q", messages[1].Content)
	}
	if messages[4].Model != "claude-haiku-4.5" {
		t.Errorf("assistant model = %q", messages[4].Model)
	}
}

func TestCollectCursorMessages(t *testing.T) {
	home := t.TempDir()
	sessionID := "010bfa5d"
	transcript := strings.Join([]string{
		`{"role":"user","message":{"content":[{"type":"text","text":"fix this"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"text","text":"Reading."},{"type":"tool_use","name":"Read","input":{"path":"/a"}}]}}`,
		`{"role":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
	}, "\n") + "\n"
	writeFile(t, filepath.Join(home, ".cursor", "projects", "tmp-fleet", "agent-transcripts", sessionID, sessionID+".jsonl"), transcript)

	refs := discoverCursorTranscripts(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(refs) != 1 {
		t.Fatalf("got %d transcripts, want 1", len(refs))
	}
	messages := refs[0].Parse(refs[0], 0, 0)

	if len(messages) != 4 {
		t.Fatalf("got %d messages, want 4", len(messages))
	}
	if messages[0].Content != "fix this" {
		t.Errorf("user content = %q", messages[0].Content)
	}
	if messages[2].BlockType != blockToolUse || messages[2].Content != `{"path":"/a"}` {
		t.Errorf("tool call = %s / %q", messages[2].BlockType, messages[2].Content)
	}
	if messages[3].Role != roleSystem || messages[3].BlockType != blockToolResult {
		t.Errorf("tool result row = %s/%s", messages[3].Role, messages[3].BlockType)
	}
	// Cursor records no timestamps.
	if !messages[0].Timestamp.IsZero() {
		t.Errorf("timestamp = %v, want zero", messages[0].Timestamp)
	}
}

func TestCollectCodexMessages(t *testing.T) {
	home := t.TempDir()
	rollout := strings.Join([]string{
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"7f3c1a2b","cwd":"/w","cli_version":"0.20.0"}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"turn_context","payload":{"model":"gpt-5"}}`,
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"run the tests"}]}}`,
		`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"cmd\":\"go test\"}"}}`,
		`{"timestamp":"2026-08-01T12:00:04.000Z","type":"response_item","payload":{"type":"function_call_output","name":"shell","output":"ok"}}`,
		`{"timestamp":"2026-08-01T12:00:05.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Tests pass."}]}}`,
	}, "\n") + "\n"
	writeFile(t, filepath.Join(home, ".codex", "sessions", "2026", "08", "01",
		"rollout-2026-08-01T12-00-00-7f3c1a2b-1111-2222-3333-444455556666.jsonl"), rollout)

	refs := discoverCodexTranscripts(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(refs) != 1 {
		t.Fatalf("got %d transcripts, want 1", len(refs))
	}
	messages := refs[0].Parse(refs[0], 0, 0)

	if len(messages) != 4 {
		for i, m := range messages {
			t.Logf("row %d: %s/%s %q", i, m.Role, m.BlockType, m.Content)
		}
		t.Fatalf("got %d messages, want 4", len(messages))
	}
	if messages[0].Content != "run the tests" || messages[0].Role != roleUser {
		t.Errorf("user row = %s %q", messages[0].Role, messages[0].Content)
	}
	if messages[1].BlockType != blockToolUse || messages[1].ToolName != "shell" {
		t.Errorf("tool call row = %s/%s", messages[1].BlockType, messages[1].ToolName)
	}
	if messages[2].BlockType != blockToolResult || messages[2].Content != "ok" {
		t.Errorf("tool result row = %s %q", messages[2].BlockType, messages[2].Content)
	}
	// The model comes from turn_context and applies to later rows.
	if messages[3].Model != "gpt-5" {
		t.Errorf("assistant model = %q, want gpt-5", messages[3].Model)
	}
}

func TestCollectGeminiMessages(t *testing.T) {
	home := t.TempDir()
	logs := `[
	  {"sessionId":"aaa","messageId":0,"timestamp":"2026-08-01T10:00:00.000Z","type":"user","message":"first prompt"},
	  {"sessionId":"bbb","messageId":0,"timestamp":"2026-08-02T10:00:00.000Z","type":"user","message":"other session"},
	  {"sessionId":"aaa","messageId":1,"timestamp":"2026-08-01T10:05:00.000Z","type":"assistant","message":"ignored"}
	]`
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "d41d8cd9", "logs.json"), logs)

	refs := discoverGeminiTranscripts(userHome{Username: "alice", HomeDir: home}, time.Time{})
	if len(refs) != 1 {
		t.Fatalf("got %d transcripts, want 1", len(refs))
	}
	// One log holds many sessions, so the ref cannot name one up front.
	if refs[0].SessionID != "" {
		t.Errorf("ref session id = %q, want empty", refs[0].SessionID)
	}

	messages := refs[0].Parse(refs[0], 0, 0)
	if len(messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(messages))
	}
	if messages[0].SessionID != "aaa" || messages[0].Content != "first prompt" {
		t.Errorf("row 0 = %s %q", messages[0].SessionID, messages[0].Content)
	}
	if messages[1].SessionID != "bbb" {
		t.Errorf("row 1 session = %q, want bbb", messages[1].SessionID)
	}
}

// constraintContext builds a QueryContext with equality constraints, as
// osquery would push down from a WHERE clause.
func constraintContext(column string, values ...string) table.QueryContext {
	list := table.ConstraintList{Affinity: table.ColumnTypeText}
	for _, v := range values {
		list.Constraints = append(list.Constraints, table.Constraint{
			Operator:   table.OperatorEquals,
			Expression: v,
		})
	}
	return table.QueryContext{Constraints: map[string]table.ConstraintList{column: list}}
}

func TestCollectMessagesAppliesConstraints(t *testing.T) {
	home := t.TempDir()
	projects := filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet")
	writeFile(t, filepath.Join(projects, "sess-a.jsonl"),
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"alpha"}}`+"\n")
	writeFile(t, filepath.Join(projects, "sess-b.jsonl"),
		`{"type":"user","timestamp":"2026-08-02T10:00:00.000Z","message":{"content":"beta"}}`+"\n")
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "cop-1", "events.jsonl"),
		`{"type":"user.message","data":{"content":"gamma"},"timestamp":"2026-08-03T10:00:00.000Z"}`+"\n")

	users := []userHome{{Username: "alice", HomeDir: home}}

	t.Run("unconstrained returns everything", func(t *testing.T) {
		got := collectMessages(users, time.Time{}, messageFilter{}, 0, 0)
		if len(got) != 3 {
			t.Fatalf("got %d messages, want 3", len(got))
		}
	})

	t.Run("session_id", func(t *testing.T) {
		f := newMessageFilter(constraintContext("session_id", "sess-b"))
		got := collectMessages(users, time.Time{}, f, 0, 0)
		if len(got) != 1 || got[0].Content != "beta" {
			t.Fatalf("got %+v, want just beta", got)
		}
	})

	t.Run("agent", func(t *testing.T) {
		f := newMessageFilter(constraintContext("agent", agentCopilotCLI))
		got := collectMessages(users, time.Time{}, f, 0, 0)
		if len(got) != 1 || got[0].Content != "gamma" {
			t.Fatalf("got %+v, want just gamma", got)
		}
	})

	t.Run("IN list becomes several equality constraints", func(t *testing.T) {
		f := newMessageFilter(constraintContext("session_id", "sess-a", "sess-b"))
		got := collectMessages(users, time.Time{}, f, 0, 0)
		if len(got) != 2 {
			t.Fatalf("got %d messages, want 2", len(got))
		}
	})

	t.Run("no match", func(t *testing.T) {
		f := newMessageFilter(constraintContext("username", "bob"))
		if got := collectMessages(users, time.Time{}, f, 0, 0); len(got) != 0 {
			t.Fatalf("got %d messages, want 0", len(got))
		}
	})
}

func TestCollectMessagesCapsRowsNewestFirst(t *testing.T) {
	home := t.TempDir()
	projects := filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet")

	// Two prompts per session so the cap has to cut inside a transcript.
	writeFile(t, filepath.Join(projects, "older.jsonl"), strings.Join([]string{
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"old one"}}`,
		`{"type":"user","timestamp":"2026-08-01T10:01:00.000Z","message":{"content":"old two"}}`,
	}, "\n")+"\n")
	writeFile(t, filepath.Join(projects, "newer.jsonl"), strings.Join([]string{
		`{"type":"user","timestamp":"2026-08-10T10:00:00.000Z","message":{"content":"new one"}}`,
		`{"type":"user","timestamp":"2026-08-10T10:01:00.000Z","message":{"content":"new two"}}`,
	}, "\n")+"\n")

	// newer.jsonl must be written last so its mtime is the more recent.
	now := time.Now()
	touch(t, filepath.Join(projects, "older.jsonl"), now.Add(-48*time.Hour))
	touch(t, filepath.Join(projects, "newer.jsonl"), now)

	users := []userHome{{Username: "alice", HomeDir: home}}
	got := collectMessages(users, time.Time{}, messageFilter{}, 3, 0)

	if len(got) != 3 {
		t.Fatalf("got %d messages, want 3", len(got))
	}
	// The newest transcript is read first, so the dropped row is the oldest.
	if got[0].Content != "new one" || got[1].Content != "new two" || got[2].Content != "old one" {
		t.Errorf("rows = %q, %q, %q", got[0].Content, got[1].Content, got[2].Content)
	}
}

func TestMessageRowRendersEveryColumn(t *testing.T) {
	m := Message{
		Agent:     agentClaudeCode,
		SessionID: "abc",
		Role:      roleUser,
		BlockType: blockText,
		Content:   "hi",
		Truncated: true,
	}

	row := m.row()
	for _, col := range aiAgentSessionMessagesColumns() {
		if _, ok := row[col.Name]; !ok {
			t.Errorf("row is missing column %q", col.Name)
		}
	}
	if len(row) != len(aiAgentSessionMessagesColumns()) {
		t.Errorf("row has %d keys, want %d", len(row), len(aiAgentSessionMessagesColumns()))
	}
	if row["truncated"] != "1" {
		t.Errorf("truncated = %q, want %q", row["truncated"], "1")
	}
	if got := (Message{}).row()["truncated"]; got != "0" {
		t.Errorf("untruncated = %q, want %q", got, "0")
	}
}

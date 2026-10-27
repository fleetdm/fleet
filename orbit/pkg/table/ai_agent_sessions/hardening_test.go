package ai_agent_sessions

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestDeeplyNestedContentIsCheap(t *testing.T) {
	home := setup(t)
	deep := strings.Repeat("[", 5000) + `"x"` + strings.Repeat("]", 5000)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"), lines(
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t","content":`+deep+`}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"content":"after"}}`,
	))
	start := time.Now()
	rows := messageRows(t, eq("session_id", "s"))
	require.Less(t, time.Since(start), 2*time.Second)
	require.Len(t, rows, 1)
	require.Equal(t, "after", rows[0]["content"])
}

func TestHugeMetadataIsCapped(t *testing.T) {
	home := setup(t)
	cwd := "/" + strings.Repeat("d", 1<<20)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"),
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","cwd":"`+cwd+`","gitBranch":"`+cwd+`","message":{"content":"hi"}}`+"\n")
	writeClaudePrompts(t, home, "other", "still here")

	s := sessionRows(t, eq("session_id", "s"))[0]
	require.Len(t, s["project_path"], maxFieldBytes)
	require.Len(t, s["git_branch"], maxFieldBytes)
	rows := messageRows(t)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.LessOrEqual(t, len(r["cwd"]), maxFieldBytes)
	}
}

func TestDecisionSettledPastThePage(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"), lines(
		`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"m1","content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"one"}}]}}`,
		`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"m1","content":[{"type":"tool_use","id":"b","name":"Bash","input":{"command":"two"}}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","content":"ok"}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","toolDenialKind":"user-rejected","message":{"content":[{"type":"tool_result","tool_use_id":"b","is_error":true,"content":"no"}]}}`,
	))
	rows := messageRows(t, eq("session_id", "s"), cons{"message_index", table.OperatorLessThanOrEquals, "1"})
	require.Equal(t, []string{"Bash | execute | one | allowed", "Bash | execute | two | denied"}, summary(rows))
}

func TestMessageIndexDoesNotDependOnPage(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"), lines(
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"first"}}`,
		// Decodes as a user record but not as a full one.
		`{"type":"user","timestamp":12345,"message":{"content":"odd"}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:02.000Z","message":{"content":"third"}}`,
	))
	all := messageRows(t, eq("session_id", "s"))
	paged := messageRows(t, eq("session_id", "s"), cons{"message_index", table.OperatorGreaterThanOrEquals, "2"})
	require.Equal(t, "third", all[len(all)-1]["content"])
	require.Len(t, paged, 1)
	require.Equal(t, all[len(all)-1]["message_index"], paged[0]["message_index"])
}

func TestDecisionTextIsNotSpoofable(t *testing.T) {
	out := func(s string) json.RawMessage {
		b, err := json.Marshal([]map[string]any{{"functionResponse": map[string]any{"response": map[string]any{"output": s}}}})
		require.NoError(t, err)
		return b
	}
	require.Equal(t, decisionError, geminiDecision(geminiToolCall{Status: "success", Result: out("Exit Code: 0\nlater\nExit Code: 1")}))
	require.Equal(t, decisionAllowed, geminiDecision(geminiToolCall{Status: "success", Result: out("printed Exit Code: 3 inline\nExit Code: 0")}))
	require.Equal(t, decisionError, geminiDecision(geminiToolCall{Status: "cancelled", Result: out("file says User denied execution.")}))
	require.Equal(t, decisionAllowed, codexDecision(json.RawMessage(`"README: rejected by configuration happens when..."`), false))
}

package ai_agent_sessions

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// claudeTranscript is a trimmed-down session: a prompt, an assistant turn
// written as two records of one API message (so usage repeats), a failed
// command, a refused command, a refused-looking MCP result and a final reply.
var claudeTranscript = lines(
	`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-08-11T13:27:08.788Z","sessionId":"5d1f26db","content":"..."}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:10.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"role":"user","content":"fix the build"}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:20.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"id":"m1","role":"assistant","model":"claude-opus-5","content":[{"type":"text","text":"Looking."},{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/Users/alice/GitHub/fleet/Makefile"}}],"usage":{"input_tokens":10,"output_tokens":248,"cache_read_input_tokens":30802,"cache_creation_input_tokens":22539}}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:21.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"main","message":{"id":"m1","role":"assistant","model":"claude-opus-5","content":[{"type":"tool_use","id":"t2","name":"Bash","input":{"command":"make build\necho done"}}],"usage":{"input_tokens":10,"output_tokens":248,"cache_read_input_tokens":30802,"cache_creation_input_tokens":22539}}}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:25.000Z","cwd":"/Users/alice/GitHub/fleet","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"file contents"},{"type":"tool_result","tool_use_id":"t2","is_error":true,"content":"Exit code 2\nboom"}]}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:40.000Z","cwd":"/Users/alice/GitHub/fleet","message":{"id":"m2","role":"assistant","model":"claude-opus-5","content":[{"type":"tool_use","id":"t3","name":"Bash","input":{"command":"rm -rf /tmp/x"}}],"usage":{"input_tokens":5,"output_tokens":100,"cache_read_input_tokens":1000,"cache_creation_input_tokens":0}}}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:45.000Z","toolDenialKind":"user-rejected","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t3","is_error":true,"content":"The user doesn't want to proceed with this tool use."}]}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:50.000Z","message":{"id":"m3","role":"assistant","model":"claude-opus-5","content":[{"type":"tool_use","id":"t4","name":"mcp__github__create_issue","input":{"title":"x"}}],"usage":{"input_tokens":1,"output_tokens":1}}}`,
	`{"type":"user","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:55.000Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t4","is_error":true,"content":[{"type":"text","text":"The user doesn't want to proceed"}]}]}}`,
	`{"type":"assistant","sessionId":"5d1f26db","timestamp":"2026-08-11T13:28:10.000Z","cwd":"/Users/alice/GitHub/fleet","version":"2.1.222","gitBranch":"allenhouchins/fix","message":{"id":"m4","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"Done."}]}}`,
)

var claudeSubagent = lines(
	`{"type":"user","isSidechain":true,"agentId":"a1","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:30.000Z","message":{"role":"user","content":"explore src"}}`,
	`{"type":"assistant","isSidechain":true,"agentId":"a1","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:31.000Z","message":{"id":"s1","role":"assistant","model":"claude-haiku-5","content":[{"type":"tool_use","id":"t9","name":"Grep","input":{"pattern":"TODO","path":"src"}}],"usage":{"input_tokens":2,"output_tokens":3,"cache_read_input_tokens":4,"cache_creation_input_tokens":5}}}`,
	`{"type":"user","isSidechain":true,"agentId":"a1","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:32.000Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t9","content":"found"}]}}`,
)

// claudeProject is the project directory the generic Claude transcripts go in.
func claudeProject(home string) string {
	return filepath.Join(home, ".claude", "projects", "-p")
}

func claudePath(home, id string) string {
	return filepath.Join(claudeProject(home), id+".jsonl")
}

// writeClaudeTranscript writes session id's transcript, one record per line.
func writeClaudeTranscript(t *testing.T, home, id string, records ...string) string {
	t.Helper()
	path := claudePath(home, id)
	writeFile(t, path, lines(records...))
	return path
}

// writeClaudePrompts writes a Claude Code transcript made of plain prompts.
func writeClaudePrompts(t *testing.T, home, id string, prompts ...string) string {
	t.Helper()
	var ls []string
	for i, p := range prompts {
		b, err := json.Marshal(p)
		require.NoError(t, err)
		ls = append(ls, fmt.Sprintf(`{"type":"user","sessionId":%q,"timestamp":"2026-08-01T10:%02d:%02d.000Z","cwd":"/Users/alice/proj","message":{"role":"user","content":%s}}`,
			id, (i/60)%60, i%60, b))
	}
	path := claudePath(home, id)
	writeFile(t, path, lines(ls...))
	return path
}

func writeClaudeFixture(t *testing.T, home string) (main, sub string) {
	t.Helper()
	proj := filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet")
	main = filepath.Join(proj, "5d1f26db.jsonl")
	sub = filepath.Join(proj, "5d1f26db", "subagents", "agent-a1.jsonl")
	writeFile(t, main, claudeTranscript)
	writeFile(t, sub, claudeSubagent)
	return main, sub
}

func TestClaudeCodeSession(t *testing.T) {
	home := setup(t)
	main, _ := writeClaudeFixture(t, home)

	s := onlySession(t, agentClaudeCode)
	requireRow(t, map[string]string{
		"agent":          agentClaudeCode,
		"session_id":     "5d1f26db",
		"username":       "alice",
		"project_path":   "/Users/alice/GitHub/fleet",
		"git_branch":     "allenhouchins/fix",
		"git_repository": "",
		"agent_version":  "2.1.222",
		"model":          "claude-opus-5",
		"started_at":     "1786454828", // the queue-operation record
		"ended_at":       "1786454890",
		// Only the typed prompt counts; tool results and the sub-agent's
		// instructions don't.
		"user_messages": "1",
		// One per API message, including the sub-agent's.
		"assistant_messages": "5",
		"tool_calls":         "5",
		// Usage repeated on the split record counts once; the sub-agent's adds.
		"input_tokens":       "18",
		"output_tokens":      "352",
		"cache_read_tokens":  "31806",
		"cache_write_tokens": "22544",
		"transcript_path":    main,
		"duration_seconds":   "61",
	}, s)
}

func TestClaudeCodeMessages(t *testing.T) {
	home := setup(t)
	main, sub := writeClaudeFixture(t, home)

	rows := messageRows(t, eq("session_id", "5d1f26db"))
	require.Equal(t, []string{
		"0.0 user/text ",
		"1.0 assistant/text ",
		"1.1 assistant/tool_use Read",
		"2.0 assistant/tool_use Bash",
		"3.0 system/tool_result ",
		"3.1 system/tool_result ",
		// The sub-agent's turns fall where their timestamps put them, its
		// instructions as system text.
		"4.0 system/text ",
		"5.0 assistant/tool_use Grep",
		"6.0 system/tool_result ",
		"7.0 assistant/tool_use Bash",
		"8.0 system/tool_result ",
		"9.0 assistant/tool_use mcp__github__create_issue",
		"10.0 system/tool_result ",
		"11.0 assistant/text ",
	}, render(rows, true, "tool_name"))
	require.Equal(t, []string{"", "", "0", "1", "", "", "", "2", "", "3", "", "4", "", ""}, column(rows, "tool_call_index"))

	require.Equal(t, "fix the build", rows[0]["content"])
	require.Equal(t, "claude-opus-5", rows[1]["model"])
	require.Empty(t, rows[0]["model"])
	require.Equal(t, "Done.", rows[13]["content"])
	require.Empty(t, rows[13]["model"], "synthetic model is dropped")
	require.JSONEq(t, `{"file_path":"/Users/alice/GitHub/fleet/Makefile"}`, rows[2]["content"])
	require.Equal(t, "t1", rows[2]["tool_use_id"])
	require.Equal(t, "t1", rows[4]["tool_use_id"])
	require.Equal(t, "/Users/alice/GitHub/fleet", rows[2]["cwd"])
	require.Equal(t, "1786454840", rows[2]["timestamp"])
	require.Equal(t, main, rows[0]["transcript_path"])
	require.Equal(t, main, rows[len(rows)-1]["transcript_path"], "sub-agent rows report the session's transcript")
	require.FileExists(t, sub)

	require.Equal(t, []string{
		"Read | read | /Users/alice/GitHub/fleet/Makefile | allowed",
		"Bash | execute | make build | error",
		`Grep | search | "TODO" in src | allowed`,
		"Bash | execute | rm -rf /tmp/x | denied",
		// An MCP server's text never counts as a refusal.
		"mcp__github__create_issue | mcp | github: create_issue | error",
	}, summary(toolRows(t, "5d1f26db")))
}

func TestClaudeCodeDecisionFallbackForOlderTranscripts(t *testing.T) {
	home := setup(t)
	results := []string{
		"The user doesn't want to take this action right now.",
		"Permission to use Bash has been denied.",
		"Hook PreToolUse:Bash denied this tool",
		"PreToolUse:Bash hook error: exit 1",
		"[Request interrupted by user for tool use]",
		"<tool_use_error>File not found</tool_use_error>",
	}
	var ls []string
	for i, r := range results {
		ls = append(ls,
			`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"m`+string(rune('a'+i))+`","content":[{"type":"tool_use","id":"t`+string(rune('a'+i))+`","name":"Bash","input":{"command":"x"}}]}}`,
			`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t`+string(rune('a'+i))+`","is_error":true,"content":"`+r+`"}]}}`,
		)
	}
	ls = append(ls,
		`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"mz","content":[{"type":"tool_use","id":"tz","name":"Bash","input":{"command":"x"}}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","toolDenialKind":"interrupted","message":{"content":[{"type":"tool_result","tool_use_id":"tz","is_error":true,"content":"Interrupted"}]}}`,
	)
	writeClaudeTranscript(t, home, "old", ls...)

	require.Equal(t, []string{"denied", "denied", "denied", "error", "error", "error", "error"}, column(toolRows(t, "old"), "decision"))
}

func TestClaudeCodeInjectedUserRecordsAreNotPrompts(t *testing.T) {
	home := setup(t)
	writeClaudeTranscript(t, home, "s",
		`{"type":"user","isMeta":true,"timestamp":"2026-08-01T10:00:00.000Z","message":{"role":"user","content":"Caveat: The messages below were generated by the user while running local commands."}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"role":"user","content":"<command-message>clear</command-message>\n<command-name>/clear</command-name>"}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:02.000Z","message":{"role":"user","content":"<local-command-stdout>ok</local-command-stdout>"}}`,
		`{"type":"user","isCompactSummary":true,"timestamp":"2026-08-01T10:00:03.000Z","message":{"role":"user","content":"This session is being continued from a previous conversation."}}`,
		// The same records in block form.
		`{"type":"user","isMeta":true,"timestamp":"2026-08-01T10:00:03.100Z","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>attached file</system-reminder>"}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:03.200Z","message":{"role":"user","content":[{"type":"text","text":"<command-name>/model</command-name>"}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:04.000Z","message":{"role":"user","content":"real prompt"}}`,
		// Older versions wrote sub-agent turns into the same file.
		`{"type":"user","isSidechain":true,"timestamp":"2026-08-01T10:00:05.000Z","message":{"role":"user","content":"explore src"}}`,
		`{"type":"assistant","isSidechain":true,"timestamp":"2026-08-01T10:00:06.000Z","message":{"id":"a1","model":"claude-haiku-5","content":[{"type":"text","text":"found"}],"usage":{"input_tokens":1,"output_tokens":1}}}`,
	)
	// ... and sub-agent transcripts flat in the project directory, named by
	// their parent's session id inside.
	writeFile(t, filepath.Join(claudeProject(home), "agent-a2.jsonl"), lines(
		`{"type":"user","isSidechain":true,"agentId":"a2","sessionId":"s","timestamp":"2026-08-01T10:00:07.000Z","message":{"role":"user","content":"sub task"}}`,
		`{"type":"assistant","isSidechain":true,"agentId":"a2","sessionId":"s","timestamp":"2026-08-01T10:00:08.000Z","message":{"id":"a2","model":"claude-haiku-5","content":[{"type":"tool_use","id":"t","name":"Glob","input":{"pattern":"*"}}]}}`,
	))

	rows := sessionRows(t, eq("agent", agentClaudeCode))
	require.Len(t, rows, 1, "the flat agent file isn't a session of its own")
	s := rows[0]
	require.Equal(t, "s", s["session_id"])
	require.Equal(t, "1", s["user_messages"])
	require.Equal(t, "2", s["assistant_messages"])
	require.Equal(t, "1", s["tool_calls"])
	require.Empty(t, s["model"], "sub-agent models never set the session's")

	require.Equal(t, []string{"system", "system", "system", "system", "system", "system", "user", "system", "assistant", "system"},
		column(messageRows(t, eq("session_id", "s"), eq("block_type", "text")), "role"))
}

package ai_agent_sessions

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/stretchr/testify/require"
)

const codexID = "7f3c1a2b-1111-2222-3333-444455556666"

var codexLegacy = lines(
	`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","session_id":"`+codexID+`","timestamp":"2026-08-01T12:00:00.000Z","cwd":"/Users/alice/GitHub/fleet","originator":"codex_cli_rs","cli_version":"0.20.0","model_provider":"openai","git":{"branch":"main","repository_url":"https://github.com/fleetdm/fleet"}}}`,
	`{"timestamp":"2026-08-01T12:00:01.000Z","type":"turn_context","payload":{"cwd":"/Users/alice/GitHub/fleet","model":"gpt-5","effort":"medium"}}`,
	`{"timestamp":"2026-08-01T12:00:01.500Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>cwd</environment_context>"}]}}`,
	`{"timestamp":"2026-08-01T12:00:02.000Z","type":"event_msg","payload":{"type":"user_message","message":"run the tests"}}`,
	`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"run the tests"}]}}`,
	`{"timestamp":"2026-08-01T12:00:02.500Z","type":"response_item","payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about tests"}]}}`,
	`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"go test ./...\"}","call_id":"c1"}}`,
	`{"timestamp":"2026-08-01T12:00:04.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"Process exited with code 1\nFAIL"}}`,
	`{"timestamp":"2026-08-01T12:00:05.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":[\"bash\",\"-lc\",\"ls\"]}","call_id":"c2"}}`,
	`{"timestamp":"2026-08-01T12:00:06.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c2","output":"exec command rejected by user"}}`,
	`{"timestamp":"2026-08-01T12:00:07.000Z","type":"response_item","payload":{"type":"custom_tool_call","name":"apply_patch","input":"*** Begin Patch\n*** Update File: a.go\n*** Add File: b.go\n*** End Patch","call_id":"c3"}}`,
	`{"timestamp":"2026-08-01T12:00:08.000Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c3","output":"Success. Updated the following files"}}`,
	`{"timestamp":"2026-08-01T12:00:09.000Z","type":"response_item","payload":{"type":"function_call","namespace":"mcp__github","name":"create_issue","arguments":"{\"title\":\"x\"}","call_id":"c4"}}`,
	`{"timestamp":"2026-08-01T12:00:10.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c4","output":"created"}}`,
	`{"timestamp":"2026-08-01T12:00:11.000Z","type":"response_item","payload":{"type":"web_search_call","status":"completed","action":{"type":"search","query":"golang generics"}}}`,
	`{"timestamp":"2026-08-01T12:00:11.500Z","type":"response_item","payload":{"type":"local_shell_call","call_id":"c5","action":{"type":"exec","command":["git","status"]}}}`,
	`{"timestamp":"2026-08-01T12:00:12.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Tests fail."}]}}`,
	`{"timestamp":"2026-08-01T12:00:12.500Z","type":"event_msg","payload":{"type":"agent_message","message":"Tests fail."}}`,
	`{"timestamp":"2026-08-01T12:00:13.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":20}}}}`,
	`{"timestamp":"2026-08-01T12:00:14.000Z","type":"event_msg","payload":{"type":"token_count","info":null}}`,
	`{"timestamp":"2026-08-01T12:00:30.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":250,"cached_input_tokens":90,"output_tokens":75}}}}`,
	// Overflowing the context window rewrites the totals to zero.
	`{"timestamp":"2026-08-01T12:00:31.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":0,"cached_input_tokens":0,"output_tokens":0}}}}`,
)

func codexPath(home, name string) string {
	return filepath.Join(home, ".codex", "sessions", "2026", "08", "01", name)
}

func TestCodexCLISession(t *testing.T) {
	home := setup(t)
	path := codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl")
	writeFile(t, path, codexLegacy)

	s := onlySession(t, agentCodexCLI)
	requireRow(t, map[string]string{
		"session_id":         codexID,
		"project_path":       "/Users/alice/GitHub/fleet",
		"git_branch":         "main",
		"git_repository":     "https://github.com/fleetdm/fleet",
		"agent_version":      "0.20.0",
		"model":              "gpt-5",
		"user_messages":      "1",
		"assistant_messages": "1",
		"tool_calls":         "6",
		"input_tokens":       "160",
		"output_tokens":      "75",
		"cache_read_tokens":  "90",
		"cache_write_tokens": "0",
		"duration_seconds":   "31",
		"transcript_path":    path,
	}, s)
}

func TestCodexCLIMessages(t *testing.T) {
	home := setup(t)
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), codexLegacy)

	rows := messageRows(t, eq("session_id", codexID))
	require.Equal(t, []string{
		"user/text ",
		"assistant/thinking ",
		"assistant/tool_use exec_command",
		"system/tool_result ",
		"assistant/tool_use shell",
		"system/tool_result ",
		"assistant/tool_use apply_patch",
		"system/tool_result ",
		"assistant/tool_use create_issue",
		"system/tool_result ",
		"assistant/tool_use web_search",
		"assistant/tool_use local_shell",
		"assistant/text ",
	}, render(rows, false, "tool_name"))
	require.Equal(t, "run the tests", rows[0]["content"])
	require.Equal(t, "thinking about tests", rows[1]["content"])
	require.Equal(t, "gpt-5", rows[2]["model"])
	require.Equal(t, "c1", rows[3]["tool_use_id"])

	require.Equal(t, []string{
		"exec_command | execute | go test ./... | error",
		"shell | execute | bash -lc ls | denied",
		"apply_patch | write | a.go, b.go | allowed",
		"create_issue | mcp | github: create_issue | allowed",
		"web_search | network | golang generics | ",
		"local_shell | execute | git status | ",
	}, summary(toolRows(t, codexID)))
}

func TestCodexCLIPaginatedSubagentAndFork(t *testing.T) {
	home := setup(t)
	root := "11111111-0000-0000-0000-000000000001"
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+root+".jsonl"), lines(
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+root+`","session_id":"`+root+`","cwd":"/w","cli_version":"0.130.0","history_mode":"paginated"}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"event_msg","payload":{"type":"item_completed","thread_id":"`+root+`","turn_id":"1","item":{"type":"UserMessage","id":"u1","content":[{"type":"text","text":"hello"}]}}}`,
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"c1"}}`,
		`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"Process exited with code 0\nok"}}`,
		`{"timestamp":"2026-08-01T12:00:04.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":999,"cached_input_tokens":0,"output_tokens":999}}}}`,
		`{"timestamp":"2026-08-01T12:00:05.000Z","type":"token_usage_record","payload":{"thread_token_usage":{"input_tokens":50,"cached_input_tokens":10,"cache_write_input_tokens":3,"output_tokens":5}}}`,
	))
	// A sub-agent's rollout is merged into its root session.
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-02-22222222-0000-0000-0000-000000000002.jsonl"), lines(
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"session_meta","payload":{"id":"22222222-0000-0000-0000-000000000002","session_id":"`+root+`","parent_thread_id":"`+root+`","cwd":"/w","cli_version":"0.130.0","history_mode":"paginated"}}`,
		`{"timestamp":"2026-08-01T12:00:02.500Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}","call_id":"s1"}}`,
	))
	// A fork is its own session, without the history it copied.
	fork := "33333333-0000-0000-0000-000000000003"
	writeFile(t, codexPath(home, "rollout-2026-08-01T13-00-00-"+fork+".jsonl"), lines(
		`{"timestamp":"2026-08-01T13:00:00.000Z","type":"session_meta","payload":{"id":"`+fork+`","session_id":"`+fork+`","forked_from_id":"`+root+`","timestamp":"2026-08-01T13:00:00.000Z","cwd":"/w","cli_version":"0.130.0"}}`,
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+root+`","session_id":"`+root+`","cwd":"/w","cli_version":"0.130.0"}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}`,
		`{"timestamp":"2026-08-01T12:00:04.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":10}}}}`,
		`{"timestamp":"2026-08-01T13:00:05.000Z","type":"event_msg","payload":{"type":"user_message","message":"forked prompt"}}`,
		`{"timestamp":"2026-08-01T13:00:06.000Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":160,"cached_input_tokens":20,"output_tokens":15}}}}`,
	))

	rows := sessionRows(t, eq("agent", agentCodexCLI))
	require.Len(t, rows, 2)
	byID := rowsByID(rows)
	require.Equal(t, []string{"1", "2", "40", "5", "10", "3"}, []string{
		byID[root]["user_messages"], byID[root]["tool_calls"], byID[root]["input_tokens"],
		byID[root]["output_tokens"], byID[root]["cache_read_tokens"], byID[root]["cache_write_tokens"],
	})
	require.Equal(t, []string{"1", "50", "5", "10"}, []string{
		byID[fork]["user_messages"], byID[fork]["input_tokens"], byID[fork]["output_tokens"], byID[fork]["cache_read_tokens"],
	})

	forkRows := messageRows(t, eq("session_id", fork))
	require.Len(t, forkRows, 1)
	require.Equal(t, "forked prompt", forkRows[0]["content"])
	require.Equal(t, []string{
		"exec_command | execute | ls | allowed",
		"exec_command | execute | pwd | ",
	}, summary(toolRows(t, root)))
}

func TestCodexCLIFlatLayout(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-08-01T12-00-00-old.jsonl"), lines(
		`{"id":"9a8b7c6d","timestamp":"2026-08-01T12:00:00.000Z","instructions":"..."}`,
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}`,
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`,
		`{"type":"function_call","name":"shell","arguments":"{\"command\":[\"ls\"]}","call_id":"c1"}`,
		`{"type":"function_call_output","call_id":"c1","output":"{\"output\":\"x\",\"metadata\":{\"exit_code\":2}}"}`,
	))

	s := onlySession(t, agentCodexCLI)
	require.Equal(t, "9a8b7c6d", s["session_id"])
	require.Equal(t, []string{"1", "1", "1"}, []string{s["user_messages"], s["assistant_messages"], s["tool_calls"]})
	require.NotEqual(t, "0", s["ended_at"])
	require.Equal(t, []string{"shell | execute | ls | error"}, summary(toolRows(t, "9a8b7c6d")))
}

func TestCodexCLISubagentWithOlderRoot(t *testing.T) {
	home := setup(t)
	root := "11111111-0000-0000-0000-000000000001"
	rootPath := codexPath(home, "rollout-2026-08-01T12-00-00-"+root+".jsonl")
	writeFile(t, rootPath, lines(
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+root+`","session_id":"`+root+`","cwd":"/w","cli_version":"0.130.0"}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}`,
	))
	touch(t, rootPath, time.Now().AddDate(0, 0, -40))
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-02-22222222-0000-0000-0000-000000000002.jsonl"), lines(
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"session_meta","payload":{"id":"22222222-0000-0000-0000-000000000002","session_id":"`+root+`","parent_thread_id":"`+root+`","cwd":"/w"}}`,
		`{"timestamp":"2026-08-01T12:00:02.500Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}","call_id":"s1"}}`,
	))
	s := onlySession(t, agentCodexCLI)
	require.Equal(t, root, s["session_id"])
	require.Equal(t, "/w", s["project_path"])
	require.Equal(t, "1", s["user_messages"])
	require.Equal(t, "1", s["tool_calls"])
	require.Equal(t, rootPath, s["transcript_path"])
}

func TestCodexCLISessionIDFallsBackToFileName(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-odd.jsonl"), lines(
		`not json`,
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}`,
	))
	require.Equal(t, "rollout-odd", onlySession(t, agentCodexCLI)["session_id"])
}

func TestCodexCLIMessageIndexDoesNotDependOnPage(t *testing.T) {
	home := setup(t)
	// A record the full decode rejects (name isn't a string) still counts.
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), codexLegacy+
		`{"timestamp":"2026-08-01T12:00:40.000Z","type":"response_item","payload":{"type":"function_call","name":5,"arguments":"{}","call_id":"c9"}}`+"\n"+
		`{"timestamp":"2026-08-01T12:00:41.000Z","type":"event_msg","payload":{"type":"user_message","message":"after"}}`+"\n")
	all := messageRows(t, eq("session_id", codexID))
	require.Equal(t, "after", all[len(all)-1]["content"])
	require.Equal(t, "14", all[len(all)-1]["message_index"])
	for i := range all {
		paged := messageRows(t, eq("session_id", codexID), cons{"message_index", table.OperatorGreaterThanOrEquals, all[i]["message_index"]})
		require.Equal(t, all[i:], paged, "page from %s", all[i]["message_index"])
	}
}

func TestCodexCLIDiscoveryReadsUnchangedFilesOnce(t *testing.T) {
	home := setup(t)
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), codexLegacy)
	sessionRows(t, eq("agent", agentCodexCLI))
	before := discoveryReads.Load()
	require.Len(t, sessionRows(t, eq("agent", agentCodexCLI)), 1)
	require.Equal(t, before, discoveryReads.Load())
}

func TestCodexCLIArchivedSessions(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".codex", "archived_sessions", "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), codexLegacy)
	writeFile(t, filepath.Join(home, ".codex", "sessions", "2026", "08", "01", "rollout-x.jsonl.zst"), "compressed")
	require.Equal(t, codexID, onlySession(t, agentCodexCLI)["session_id"])
}

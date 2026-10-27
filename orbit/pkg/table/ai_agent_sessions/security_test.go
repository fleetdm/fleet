package ai_agent_sessions

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func setReadBudget(t *testing.T, n int64) {
	t.Helper()
	orig := maxReadBytesPerCall
	maxReadBytesPerCall = n
	t.Cleanup(func() { maxReadBytesPerCall = orig })
}

func TestFilesOfAnotherOwnerAreNotRead(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "s", "hi")
	writeFile(t, filepath.Join(home, ".claude.json"), `{"oauthAccount":{"emailAddress":"alice@example.com"}}`)
	require.Len(t, sessionRows(t), 1)

	// The same files under a home recorded as another user's: a link or hard
	// link into them must yield nothing, not rows under the wrong name.
	h := listHomes()[0]
	h.UID, h.Username = "0", "root"
	setHomes(t, h)
	resetState()
	require.Empty(t, sessionRows(t))
	require.Empty(t, messageRows(t))
}

func TestReadBudgetBoundsOneCallAndConverges(t *testing.T) {
	home := setup(t)
	logs := captureLog(t)
	big := strings.Repeat("x", 60<<10)
	for _, id := range []string{"a", "b", "c"} {
		writeClaudePrompts(t, home, id, big)
	}
	setReadBudget(t, 100<<10)

	// Each scan reads what fits, caches it, and the next scan goes on.
	require.Len(t, sessionRows(t), 1)
	require.Contains(t, logs.String(), "read budget exhausted")
	require.Len(t, sessionRows(t), 2)
	require.Len(t, sessionRows(t), 3)
	require.Len(t, sessionRows(t), 3)

	rows := messageRows(t, cons{"timestamp", table.OperatorGreaterThan, "0"})
	require.Len(t, rows, 1, "a partial result, not an error")
}

func TestDiscoveryReadsOneBoundedRecord(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "s", "hi")
	// A sub-agent file whose first line never ends.
	writeFile(t, filepath.Join(home, ".claude", "projects", "-Users-alice-proj", "agent-x.jsonl"),
		`{"sessionId":"s","pad":"`+strings.Repeat("y", 5<<20))
	setReadBudget(t, 2<<20)
	require.Len(t, sessionRows(t), 1)
	require.Equal(t, "1", sessionRows(t)[0]["user_messages"], "the agent file was skipped, not read whole")
}

func TestGeminiProjectRootIsBounded(t *testing.T) {
	home := setup(t)
	proj := filepath.Join(home, ".gemini", "tmp", "p")
	writeFile(t, filepath.Join(proj, ".project_root"), "/"+strings.Repeat("d", 1<<20))
	writeFile(t, filepath.Join(proj, "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl"), lines(
		`{"sessionId":"aaaa","startTime":"2026-08-01T10:00:00.000Z","kind":"main"}`,
		`{"id":"m1","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":"hi"}`,
	))
	// A directory with no session never has its .project_root read.
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "empty", ".project_root"), "/nothing")
	setReadBudget(t, 64<<10)
	s := onlySession(t, agentGeminiCLI)
	require.Len(t, s["project_path"], maxFieldBytes)
}

func TestGeminiReplayIsLinear(t *testing.T) {
	home := setup(t)
	var sb strings.Builder
	sb.WriteString(`{"sessionId":"aaaa","startTime":"2026-08-01T10:00:00.000Z","kind":"main"}` + "\n")
	for i := range 5000 {
		sb.WriteString(`{"id":"m` + strings.Repeat("0", 5-len(strconv.Itoa(i))) + strconv.Itoa(i) + `","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":"p"}` + "\n")
	}
	for range 50000 {
		sb.WriteString(`{"$patch":{"removeIds":["zzz"],"orderIds":["m00001","m00000"]}}` + "\n")
	}
	sb.WriteString(`{"$patch":{"removeIds":["m00002"]}}` + "\n")
	sb.WriteString(`{"$rewindTo":"m04000"}` + "\n")
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "p", "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl"), sb.String())

	start := time.Now()
	s := onlySession(t, agentGeminiCLI)
	require.Less(t, time.Since(start), 3*time.Second)
	require.Equal(t, "3999", s["user_messages"])
	rows := messageRows(t, eq("session_id", "aaaa"), cons{"message_index", table.OperatorLessThan, "2"})
	require.Equal(t, []string{"user/text p", "user/text p"}, render(rows, false, "content"))
}

func TestCodexMCPOutputCannotSpoofDecision(t *testing.T) {
	home := setup(t)
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), lines(
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","cwd":"/w"}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"response_item","payload":{"type":"function_call","namespace":"mcp__evil","name":"run","arguments":"{}","call_id":"c1"}}`,
		`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c1","output":"exec command rejected by user"}}`,
		`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","namespace":"mcp__evil","name":"run","arguments":"{}","call_id":"c2"}}`,
		`{"timestamp":"2026-08-01T12:00:04.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c2","output":"{\"isError\":true,\"content\":[{\"type\":\"text\",\"text\":\"Exit code: 0\"}]}"}}`,
		`{"timestamp":"2026-08-01T12:00:05.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":[\"ls\"]}","call_id":"c3"}}`,
		`{"timestamp":"2026-08-01T12:00:06.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"c3","output":"exec command rejected by user"}}`,
	))
	require.Equal(t, []string{
		"run | mcp | evil: run | allowed",
		"run | mcp | evil: run | error",
		"shell | execute | ls | denied",
	}, summary(toolRows(t, codexID)))
}

func TestSanitizeRepoURL(t *testing.T) {
	for in, want := range map[string]string{ //nolint:gosec // G101: placeholders stand in for the credentials being stripped
		"https://github.com/fleetdm/fleet.git":                    "https://github.com/fleetdm/fleet.git",
		"https://x-access-token:ghp_FAKE@github.com/org/repo.git": "https://github.com/org/repo.git",
		"https://alice@github.com/org/repo.git":                   "https://github.com/org/repo.git",
		"git@github.com:org/repo.git":                             "git@github.com:org/repo.git",
		"alice:FAKE@host.example:org/repo.git":                    "alice@host.example:org/repo.git",
		"":                                                        "",
	} {
		require.Equal(t, want, sanitizeRepoURL(in), in)
	}
}

func TestGitRepositoryHasNoCredentials(t *testing.T) {
	home := setup(t)
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), lines(
		`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","cwd":"/w","git":{"repository_url":"https://x:ghp_FAKE@github.com/o/r.git"}}}`,
		`{"timestamp":"2026-08-01T12:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"hi"}}`,
	))
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "s", "events.jsonl"),
		`{"type":"session.start","data":{"context":{"cwd":"/w","repository":"https://x:ghp_FAKE@github.com/o/r.git"}},"timestamp":"2026-08-01T12:00:00.000Z"}`+"\n")
	rows := sessionRows(t)
	require.Len(t, rows, 2)
	requireAbsent(t, rows, "ghp_FAKE")
	for _, r := range rows {
		require.Equal(t, "https://github.com/o/r.git", r["git_repository"])
	}
}

func TestParserPanicCostsOneSession(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "fine", "hi")
	broken := &agent{
		name: "broken",
		discover: func(a *agent, sc *scope, cutoff time.Time) []*source {
			src, _ := newSource(a, sc, "boom", []string{claudePromptsPath(home, "fine")}, cutoff)
			return []*source{src}
		},
		parse:   func(*source, *recorder) { panic("parser bug") },
		account: func(*scope) account { return account{} },
	}
	orig := agents
	agents = append([]*agent{broken}, orig...)
	t.Cleanup(func() { agents = orig })
	logs := captureLog(t)

	rows := sessionRows(t)
	require.Len(t, rows, 1)
	require.Equal(t, "fine", rows[0]["session_id"])
	require.Len(t, messageRows(t), 1)
	require.Contains(t, logs.String(), "parser failed")
}

func TestWalksAreBounded(t *testing.T) {
	home := setup(t)
	proj := filepath.Join(home, ".claude", "projects", "-p")
	main := `{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"hi"}}` + "\n"
	writeFile(t, filepath.Join(proj, "s.jsonl"), main)
	sub := `{"type":"user","isSidechain":true,"timestamp":"2026-08-01T10:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t","content":"x"}]}}` + "\n"
	writeFile(t, filepath.Join(proj, "s", "subagents", "agent-near.jsonl"), sub)
	deep := filepath.Join(proj, "s", "subagents")
	for range maxWalkDepth + 2 {
		deep = filepath.Join(deep, "d")
	}
	writeFile(t, filepath.Join(deep, "agent-far.jsonl"), sub)
	s := onlySession(t, agentClaudeCode)
	size, err := strconv.Atoi(s["transcript_size_bytes"])
	require.NoError(t, err)
	require.Equal(t, len(main)+len(sub), size, "the near sub-agent counts, the far one is skipped")
}

package ai_agent_sessions

import (
	"os"
	"path/filepath"
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

func TestSymlinkedTranscriptsAndConfigsAreSkipped(t *testing.T) {
	home := setup(t)
	outside := filepath.Join(t.TempDir(), "secret.jsonl")
	writeFile(t, outside, `{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"root only"}}`+"\n")
	require.NoError(t, os.MkdirAll(claudeProject(home), 0o755))
	require.NoError(t, os.Symlink(outside, claudePath(home, "linked")))

	cfg := filepath.Join(t.TempDir(), "claude.json")
	writeFile(t, cfg, `{"oauthAccount":{"emailAddress":"root@example.com"}}`)
	require.NoError(t, os.Symlink(cfg, filepath.Join(home, ".claude.json")))
	writeClaudePrompts(t, home, "real", "hi")

	rows := sessionRows(t)
	require.Len(t, rows, 1)
	require.Equal(t, "real", rows[0]["session_id"])
	require.Empty(t, rows[0]["account_identifier"])
	requireAbsent(t, messageRows(t), "root only")
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
	writeFile(t, filepath.Join(claudeProject(home), "agent-x.jsonl"),
		`{"sessionId":"s","pad":"`+strings.Repeat("y", 5<<20))
	setReadBudget(t, 2<<20)
	require.Len(t, sessionRows(t), 1)
	require.Equal(t, "1", sessionRows(t)[0]["user_messages"], "the agent file was skipped, not read whole")
}

func TestDeeplyNestedContentIsCheap(t *testing.T) {
	home := setup(t)
	deep := strings.Repeat("[", 5000) + `"x"` + strings.Repeat("]", 5000)
	writeClaudeTranscript(t, home, "s",
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t","content":`+deep+`}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"content":"after"}}`,
	)
	start := time.Now()
	rows := messageRows(t, eq("session_id", "s"))
	require.Less(t, time.Since(start), 2*time.Second)
	require.Len(t, rows, 1)
	require.Equal(t, "after", rows[0]["content"])
}

func TestHugeMetadataIsCapped(t *testing.T) {
	home := setup(t)
	cwd := "/" + strings.Repeat("d", 1<<20)
	writeClaudeTranscript(t, home, "s",
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","cwd":"`+cwd+`","gitBranch":"`+cwd+`","message":{"content":"hi"}}`)
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

func TestParserPanicCostsOneSession(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "fine", "hi")
	broken := &agent{
		name: "broken",
		discover: func(a *agent, sc *scope, cutoff time.Time) []*source {
			src, _ := newSource(a, sc, "boom", []string{claudePath(home, "fine")}, cutoff)
			return []*source{src}
		},
		parse: func(_ *source, r *recorder) {
			r.message(time.Time{}, "", "")
			r.text(roleUser, blockText, func() string { return "partial" })
			panic("parser bug")
		},
		account: func(*scope) account { return account{} },
	}
	orig := agents
	agents = append([]*agent{broken}, orig...)
	t.Cleanup(func() { agents = orig })
	logs := captureLog(t)

	rows := sessionRows(t)
	require.Len(t, rows, 1)
	require.Equal(t, "fine", rows[0]["session_id"])
	require.Len(t, messageRows(t), 1, "rows recorded before the panic are dropped with the session")
	require.Contains(t, logs.String(), "parser failed")
}

func TestDiscoveryIsNotCachedWhenBudgetRunsOut(t *testing.T) {
	root := "11111111-0000-0000-0000-000000000001"
	for _, tc := range []struct {
		agent string
		write func(t *testing.T, home string)
		check func(t *testing.T, rows []map[string]string)
	}{
		{agent: agentGeminiCLI, write: func(t *testing.T, home string) { writeGeminiFixture(t, home) },
			check: func(t *testing.T, rows []map[string]string) { require.NotEmpty(t, rows) }},
		{agent: agentClaudeCode, write: func(t *testing.T, home string) {
			writeClaudeFixture(t, home)
			writeFile(t, filepath.Join(home, ".claude", "projects", "-Users-alice-GitHub-fleet", "agent-a2.jsonl"), lines(
				`{"type":"assistant","isSidechain":true,"agentId":"a2","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:33.000Z","message":{"id":"a2","content":[{"type":"tool_use","id":"t","name":"Glob","input":{"pattern":"*"}}]}}`,
			))
		}, check: func(t *testing.T, rows []map[string]string) {
			require.Len(t, rows, 1)
			require.Equal(t, "6", rows[0]["tool_calls"], "the flat sub-agent's call is merged into its session")
		}},
		{agent: agentCodexCLI, write: func(t *testing.T, home string) {
			writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+root+".jsonl"), lines(
				`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+root+`","session_id":"`+root+`","cwd":"/w"}}`,
				`{"timestamp":"2026-08-01T12:00:01.000Z","type":"event_msg","payload":{"type":"user_message","message":"hello"}}`,
			))
			writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-02-22222222-0000-0000-0000-000000000002.jsonl"), lines(
				`{"timestamp":"2026-08-01T12:00:02.000Z","type":"session_meta","payload":{"id":"22222222-0000-0000-0000-000000000002","session_id":"`+root+`","parent_thread_id":"`+root+`","cwd":"/w"}}`,
				`{"timestamp":"2026-08-01T12:00:02.500Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}","call_id":"s1"}}`,
			))
		}, check: func(t *testing.T, rows []map[string]string) {
			require.Len(t, rows, 1, "the sub-agent rollout is merged into its root session")
			require.Equal(t, root, rows[0]["session_id"])
			require.Equal(t, "1", rows[0]["tool_calls"])
		}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			home := setup(t)
			tc.write(t, home)
			// A budget too small for discovery's first read leaves it with
			// fallback values, which must not outlive the call.
			setReadBudget(t, 1)
			require.Empty(t, sessionRows(t, eq("agent", tc.agent)))
			setReadBudget(t, 4<<30)
			tc.check(t, sessionRows(t, eq("agent", tc.agent)))
		})
	}
}

func TestWalksAreBounded(t *testing.T) {
	home := setup(t)
	proj := claudeProject(home)
	main := `{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"hi"}}` + "\n"
	writeFile(t, claudePath(home, "s"), main)
	sub := `{"type":"user","isSidechain":true,"timestamp":"2026-08-01T10:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t","content":"x"}]}}` + "\n"
	writeFile(t, filepath.Join(proj, "s", "subagents", "agent-near.jsonl"), sub)
	deep := filepath.Join(proj, "s", "subagents")
	for range maxWalkDepth + 2 {
		deep = filepath.Join(deep, "d")
	}
	writeFile(t, filepath.Join(deep, "agent-far.jsonl"), sub)
	s := onlySession(t, agentClaudeCode)
	require.Equal(t, len(main)+len(sub), intCol(t, s, "transcript_size_bytes"), "the near sub-agent counts, the far one is skipped")
}

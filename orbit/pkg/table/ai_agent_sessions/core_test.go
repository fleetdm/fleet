package ai_agent_sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestColumnsMatchRows(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "s1", "hello")

	for _, tc := range []struct {
		cols []table.ColumnDefinition
		rows []map[string]string
	}{
		{SessionsColumns(), sessionRows(t)},
		{MessagesColumns(), messageRows(t)},
	} {
		require.Len(t, tc.rows, 1)
		require.Len(t, tc.rows[0], len(tc.cols))
		for _, c := range tc.cols {
			require.Contains(t, tc.rows[0], c.Name)
		}
	}
}

func TestEmptyHomeHasNoRows(t *testing.T) {
	setup(t)
	require.Empty(t, sessionRows(t))
	require.Empty(t, messageRows(t))
}

func TestUsernameComesFromHomeOwner(t *testing.T) {
	setup(t)
	// The directory's name says nothing about who owns it.
	dir := filepath.Join(t.TempDir(), "not-bob")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	h := ownedHome(t, dir)
	h.Username = "bob"
	setHomes(t, h)
	writeClaudePrompts(t, dir, "s1", "hi")

	rows := sessionRows(t)
	require.Len(t, rows, 1)
	require.Equal(t, "bob", rows[0]["username"])
	require.Equal(t, "bob", messageRows(t)[0]["username"])

	require.Empty(t, sessionRows(t, eq("username", "alice")))
	require.Len(t, sessionRows(t, eq("username", "bob")), 1)
}

func TestCacheIsKeyedByHomeOwner(t *testing.T) {
	setup(t)
	dir := t.TempDir()
	h := ownedHome(t, dir)
	setHomes(t, h)
	writeClaudePrompts(t, dir, "s1", "hi")
	require.Equal(t, "alice", sessionRows(t)[0]["username"])

	// The same, unchanged transcript under a renamed account is reported
	// under the new name.
	h.Username = "bob"
	setHomes(t, h)
	require.Equal(t, "bob", sessionRows(t)[0]["username"])
}

func TestSessionsNewestFirst(t *testing.T) {
	home := setup(t)
	writeClaudeTranscript(t, home, "older", `{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"hi"}}`)
	writeClaudeTranscript(t, home, "newer", `{"type":"user","timestamp":"2026-08-10T10:00:00.000Z","message":{"content":"hi"}}`)

	require.Equal(t, []string{"newer", "older"}, column(sessionRows(t), "session_id"))
}

func TestDefaultAgeCutoff(t *testing.T) {
	home := setup(t)
	old := writeClaudePrompts(t, home, "old", "old prompt")
	touch(t, old, time.Now().AddDate(0, 0, -40))
	writeClaudePrompts(t, home, "new", "new prompt")

	since := func(d int) string { return strconv.FormatInt(time.Now().AddDate(0, 0, -d).Unix(), 10) }

	t.Run("default is 30 days", func(t *testing.T) {
		require.Equal(t, []string{"new"}, column(sessionRows(t), "session_id"))
		require.Equal(t, []string{"new"}, column(messageRows(t), "session_id"))
	})
	t.Run("a lower bound replaces it", func(t *testing.T) {
		require.ElementsMatch(t, []string{"new", "old"}, column(sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, since(50)}), "session_id"))
		require.ElementsMatch(t, []string{"new", "old"}, column(messageRows(t, cons{"timestamp", table.OperatorGreaterThanOrEquals, since(50)}), "session_id"))
		require.Empty(t, sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)}))
	})
	t.Run("only an upper bound drops it", func(t *testing.T) {
		require.ElementsMatch(t, []string{"new", "old"}, column(sessionRows(t, cons{"ended_at", table.OperatorLessThan, since(-1)}), "session_id"))
		require.ElementsMatch(t, []string{"new", "old"}, column(messageRows(t, cons{"timestamp", table.OperatorLessThanOrEquals, since(-1)}), "session_id"))
	})
	t.Run("a named session ignores it", func(t *testing.T) {
		rows := messageRows(t, eq("session_id", "old"))
		require.Len(t, rows, 1)
		require.Equal(t, "old prompt", rows[0]["content"])
	})
}

// Fleet asks for sessions started in the last 30 days with started_at >=,
// a filter osquery applies to the rows the table returns: the table selects
// sessions by when their transcript was last written. A session whose start
// is unknown reports 0, so the filter leaves it out.
func TestStartedAtFilter(t *testing.T) {
	home := setup(t)
	at := func(d int) time.Time { return time.Now().AddDate(0, 0, -d).UTC() }
	rec := func(ts time.Time) string {
		return fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"content":"hi"}}`, ts.Format(time.RFC3339Nano))
	}
	writeClaudeTranscript(t, home, "recent", rec(at(5)), rec(at(4)))
	writeClaudeTranscript(t, home, "older", rec(at(40)), rec(at(10)))
	writeClaudeTranscript(t, home, "undated", `{"type":"user","message":{"content":"hi"}}`)

	since := at(30).Unix()
	rows := sessionRows(t, cons{"started_at", table.OperatorGreaterThanOrEquals, strconv.FormatInt(since, 10)})
	require.Len(t, rows, 3, "every session written in the window is returned, for osquery to filter")
	require.Equal(t, "0", rowsByID(rows)["undated"]["started_at"])

	var kept []string
	for _, r := range rows {
		if v, err := strconv.ParseInt(r["started_at"], 10, 64); err == nil && v >= since {
			kept = append(kept, r["session_id"])
		}
	}
	require.Equal(t, []string{"recent"}, kept)
}

func TestSessionIDOpensOneTranscript(t *testing.T) {
	home := setup(t)
	for _, id := range []string{"a", "b", "c"} {
		writeClaudePrompts(t, home, id, "prompt "+id)
	}
	before := transcriptsParsed.Load()
	rows := messageRows(t, eq("session_id", "b"))
	require.Len(t, rows, 1)
	require.Equal(t, "prompt b", rows[0]["content"])
	require.EqualValues(t, 1, transcriptsParsed.Load()-before)
}

// TestFiltersReturnASubsetOfAllRows checks that pushed-down filters only
// select rows: indexes, decisions and content match the unfiltered query.
func TestFiltersReturnASubsetOfAllRows(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		write    func(t *testing.T, home string)
	}{
		{"claude code", "s", func(t *testing.T, home string) {
			writeClaudeTranscript(t, home, "s",
				`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"first"}}`,
				// Decodes as a user record but not as a full one.
				`{"type":"user","timestamp":12345,"message":{"content":"odd"}}`,
				// The empty text block yields no row but still takes its index.
				`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m1","content":[{"type":"text","text":"ok"},{"type":"text","text":""},{"type":"tool_use","id":"a","name":"Bash","input":{"command":"one"}}]}}`,
				`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m1","content":[{"type":"tool_use","id":"b","name":"Bash","input":{"command":"two"}}]}}`,
				// Results past an upper bound still decide the calls before it.
				`{"type":"user","timestamp":"2026-08-01T10:00:02.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","is_error":true,"content":"Exit code 1"}]}}`,
				`{"type":"user","timestamp":"2026-08-01T10:00:02.000Z","toolDenialKind":"user-rejected","message":{"content":[{"type":"tool_result","tool_use_id":"b","is_error":true,"content":"no"}]}}`,
				`{"type":"user","timestamp":"2026-08-01T10:00:03.000Z","message":{"content":"last"}}`,
			)
		}},
		{"codex cli", codexID, func(t *testing.T, home string) {
			// A record the full decode rejects (name isn't a string) still counts.
			writeCodexRollout(t, home, codexLegacy+lines(
				`{"timestamp":"2026-08-01T12:00:40.000Z","type":"response_item","payload":{"type":"function_call","name":5,"arguments":"{}","call_id":"c9"}}`,
				`{"timestamp":"2026-08-01T12:00:41.000Z","type":"event_msg","payload":{"type":"user_message","message":"after"}}`,
			))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := setup(t)
			tc.write(t, home)
			all := messageRows(t, eq("session_id", tc.id))
			require.NotEmpty(t, all)

			check := func(keep func(map[string]string) bool, cs ...cons) {
				t.Helper()
				var want []map[string]string
				for _, r := range all {
					if keep(r) {
						want = append(want, r)
					}
				}
				got := messageRows(t, append([]cons{eq("session_id", tc.id)}, cs...)...)
				if len(want) == 0 {
					require.Empty(t, got, "%v", cs)
					return
				}
				require.Equal(t, want, got, "%v", cs)
			}
			for _, bt := range []string{"text", "thinking", "tool_use", "tool_result"} {
				check(func(r map[string]string) bool { return r["block_type"] == bt }, eq("block_type", bt))
			}
			idx := func(r map[string]string) int { return intCol(t, r, "message_index") }
			for k := 0; k <= idx(all[len(all)-1])+1; k++ {
				v := strconv.Itoa(k)
				check(func(r map[string]string) bool { return idx(r) >= k }, cons{"message_index", table.OperatorGreaterThanOrEquals, v})
				check(func(r map[string]string) bool { return idx(r) <= k }, cons{"message_index", table.OperatorLessThanOrEquals, v})
			}
			// A tool_call_index bound selects tool calls only, numbered the
			// same whether or not the calls before them were decoded.
			calls := 0
			for _, r := range all {
				if r["block_type"] == blockToolUse {
					calls++
				}
			}
			call := func(r map[string]string) (int, bool) {
				if r["block_type"] != blockToolUse {
					return 0, false
				}
				return intCol(t, r, "tool_call_index"), true
			}
			for k := 0; k <= calls; k++ {
				v := strconv.Itoa(k)
				check(func(r map[string]string) bool { i, ok := call(r); return ok && i >= k }, cons{"tool_call_index", table.OperatorGreaterThanOrEquals, v})
				check(func(r map[string]string) bool { i, ok := call(r); return ok && i <= k }, cons{"tool_call_index", table.OperatorLessThanOrEquals, v})
				check(func(r map[string]string) bool { i, ok := call(r); return ok && i == k }, eq("tool_call_index", v))
			}
		})
	}
}

// TestDiscoveryCachesUnchangedFiles checks that a repeated scan reads
// nothing that hasn't changed, for every agent whose discovery reads files.
func TestDiscoveryCachesUnchangedFiles(t *testing.T) {
	for _, tc := range []struct {
		agent string
		extra func(t *testing.T, home string)
		after func(t *testing.T)
	}{
		{agent: agentClaudeCode, extra: func(t *testing.T, home string) {
			// Flat sub-agent files are opened to learn their parent.
			main, _ := writeClaudeFixture(t, home)
			writeFile(t, filepath.Join(filepath.Dir(main), "agent-a2.jsonl"),
				`{"type":"user","isSidechain":true,"agentId":"a2","sessionId":"5d1f26db","timestamp":"2026-08-11T13:27:33.000Z","message":{"content":"sub task"}}`+"\n")
		}},
		{agent: agentCodexCLI},
		{agent: agentGeminiCLI, after: func(t *testing.T) {
			require.Len(t, messageRows(t, eq("session_id", "bbbb")), 2, "log-only prompts are still read for rows")
		}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			home := setup(t)
			sessionFixtures[tc.agent](t, home)
			if tc.extra != nil {
				tc.extra(t, home)
			}
			reads := discoveryReads.Load()
			want := sessionRows(t, eq("agent", tc.agent))
			require.NotEmpty(t, want)
			require.Greater(t, discoveryReads.Load(), reads, "the fixture exercises discovery reads")

			reads, parsed := discoveryReads.Load(), transcriptsParsed.Load()
			require.Equal(t, want, sessionRows(t, eq("agent", tc.agent)))
			require.Equal(t, reads, discoveryReads.Load())
			require.Equal(t, parsed, transcriptsParsed.Load())
			if tc.after != nil {
				tc.after(t)
			}
		})
	}
}

func TestContentTruncation(t *testing.T) {
	home := setup(t)
	long := strings.Repeat("x", 2*maxBlockBytes)
	writeClaudePrompts(t, home, "s1", long)

	t.Run("8 KB per block", func(t *testing.T) {
		r := messageRows(t)[0]
		require.Equal(t, "1", r["truncated"])
		require.Len(t, r["content"], maxBlockBytes)
		require.Equal(t, strconv.Itoa(len(long)), r["content_length"])
	})
	t.Run("lifted for a named session", func(t *testing.T) {
		r := messageRows(t, eq("session_id", "s1"))[0]
		require.Equal(t, "0", r["truncated"])
		require.Equal(t, long, r["content"])
	})
}

func TestTruncateUTF8(t *testing.T) {
	for _, tc := range []struct {
		in    string
		limit int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello"},
		{"abé", 3, "ab"},
		{"aé", 3, "aé"},
		{"ab🙂", 4, "ab"},
	} {
		require.Equal(t, tc.want, truncateUTF8(tc.in, tc.limit), "%q/%d", tc.in, tc.limit)
	}
}

func TestSizeLimit(t *testing.T) {
	t.Run("counts whole rows, not just content", func(t *testing.T) {
		home := setup(t)
		writeClaudePrompts(t, home, "many", slices.Repeat([]string{"p"}, 6000)...)
		logs := captureLog(t)

		rows := messageRows(t, eq("session_id", "many"))
		require.Less(t, len(rows), 6000)
		size := wireBytes(t, rows)
		require.LessOrEqual(t, size, maxResultBytes)
		require.Greater(t, size, maxResultBytes*9/10)
		require.Contains(t, logs.String(), "size limit")
		require.Contains(t, logs.String(), rows[len(rows)-1]["message_index"])
	})

	t.Run("shared by back-to-back calls, reset after idle", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, time.Now())
		prompts := slices.Repeat([]string{strings.Repeat("x", 7000)}, 100) // ~700 KB each
		writeClaudePrompts(t, home, "a", prompts...)
		writeClaudePrompts(t, home, "b", prompts...)
		writeClaudePrompts(t, home, "c", "one", "two")
		writeClaudePrompts(t, home, "d", "three")

		// osquery runs one call per value of session_id IN ('a', 'b', ...).
		call := func(id string) []map[string]string {
			*clock = clock.Add(10 * time.Millisecond)
			return messageRows(t, eq("session_id", id))
		}
		first, second := call("a"), call("b")
		require.Len(t, first, 100)
		require.Less(t, len(second), 100)
		require.LessOrEqual(t, wireBytes(t, first)+wireBytes(t, second), maxResultBytes+1024)

		// Later values of the IN list aren't mistaken for empty sessions.
		third := call("c")
		require.Len(t, third, 1)
		require.Equal(t, "one", third[0]["content"])
		require.Len(t, call("d"), 1)

		*clock = clock.Add(budgetIdle + time.Millisecond)
		require.Len(t, messageRows(t, eq("session_id", "b")), 100)
	})

	t.Run("one oversized block still fits one row", func(t *testing.T) {
		prompt := func(content string) func(string) {
			return func(home string) { writeClaudePrompts(t, home, "huge", content) }
		}
		for _, tc := range []struct {
			name  string
			write func(home string)
			json  bool
		}{
			{"plain", prompt(strings.Repeat("y", 3*maxResultBytes)), false},
			{"escaped", prompt(strings.Repeat("<", 600<<10)), false}, // six bytes each once JSON encoded
			{"JSON arguments", func(home string) {
				writeClaudeToolUse(t, home, "huge", "Write", `{"file_path":"/a","content":"`+strings.Repeat("<", 600<<10)+`"}`)
			}, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				home := setup(t)
				tc.write(home)
				rows := messageRows(t, eq("session_id", "huge"))
				require.Len(t, rows, 1)
				require.Equal(t, "1", rows[0]["truncated"])
				require.LessOrEqual(t, wireBytes(t, rows), maxResultBytes)
				if tc.json {
					require.True(t, json.Valid([]byte(rows[0]["content"])))
				}
			})
		}
	})
}

func TestPagingByMessageIndex(t *testing.T) {
	home := setup(t)
	big := strings.Repeat("z", 6000)
	prompts := make([]string, 400)
	for i := range prompts {
		prompts[i] = fmt.Sprintf("%d %s", i, big)
	}
	writeClaudePrompts(t, home, "s", prompts...)
	clock := setClock(t, time.Now())

	seen := map[int]bool{}
	from := 0
	for page := 0; page < 10 && from < 400; page++ {
		rows := messageRows(t, eq("session_id", "s"), cons{"message_index", table.OperatorGreaterThanOrEquals, strconv.Itoa(from)})
		require.NotEmpty(t, rows)
		for i, r := range rows {
			idx := intCol(t, r, "message_index")
			require.Equal(t, from+i, idx, "pages are contiguous")
			seen[idx] = true
		}
		from = intCol(t, rows[len(rows)-1], "message_index") + 1
		*clock = clock.Add(time.Second)
	}
	require.Len(t, seen, 400)

	rows := messageRows(t, eq("session_id", "s"),
		cons{"message_index", table.OperatorGreaterThan, "10"},
		cons{"message_index", table.OperatorLessThanOrEquals, "12"})
	require.Len(t, rows, 2)
	require.Equal(t, "11", rows[0]["message_index"])
	require.Equal(t, "12", rows[1]["message_index"])
}

func TestPagingResumesInsideAMessage(t *testing.T) {
	home := setup(t)
	// One assistant message whose blocks together exceed the size limit.
	blocks := make([]string, 200)
	for i := range blocks {
		blocks[i] = fmt.Sprintf(`{"type":"text","text":"%d %s"}`, i, strings.Repeat("z", 7000))
	}
	writeClaudeTranscript(t, home, "s",
		`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"m1","model":"m","content":[`+strings.Join(blocks, ",")+`]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:01.000Z","message":{"role":"user","content":"after"}}`,
	)
	clock := setClock(t, time.Now())

	rows := messageRows(t, eq("session_id", "s"))
	require.NotEmpty(t, rows)
	require.Less(t, len(rows), 200)
	last := rows[len(rows)-1]
	require.Equal(t, "0", last["message_index"], "the cut fell inside the first message")

	// Two queries read on: the rest of the cut message, then the messages
	// after it. (osquery doesn't split the OR of the two into separate
	// calls; it only passes the message_index lower bound down.)
	seen := len(rows)
	for page := 0; page < 10 && seen < 200; page++ {
		*clock = clock.Add(time.Second)
		from := intCol(t, last, "block_index")
		rows = messageRows(t, eq("session_id", "s"), eq("message_index", "0"),
			cons{"block_index", table.OperatorGreaterThan, strconv.Itoa(from)})
		require.NotEmpty(t, rows, "paging must make progress")
		for i, r := range rows {
			require.Equal(t, "0", r["message_index"])
			require.Equal(t, from+1+i, intCol(t, r, "block_index"), "blocks are contiguous")
		}
		seen += len(rows)
		last = rows[len(rows)-1]
	}
	require.Equal(t, 200, seen, "every block of the cut message was returned once")

	*clock = clock.Add(time.Second)
	after := messageRows(t, eq("session_id", "s"), cons{"message_index", table.OperatorGreaterThan, "0"})
	require.Len(t, after, 1)
	require.Equal(t, "after", after[0]["content"])
}

// A transcript_path constraint names a session by its main transcript and
// selects all of it, sub-agent files included: osquery re-applies the
// constraint to the rows it gets back, so every row reports that path.
func TestPathConstraintSelectsWholeSession(t *testing.T) {
	root := "11111111-0000-0000-0000-000000000001"
	for _, tc := range []struct {
		agent string
		write func(t *testing.T, home string) (main, sub string)
	}{
		{agentClaudeCode, writeClaudeFixture},
		{agentGeminiCLI, func(t *testing.T, home string) (string, string) {
			chat := writeGeminiFixture(t, home)
			return chat, filepath.Join(filepath.Dir(chat), geminiID, "sub1.jsonl")
		}},
		{agentCodexCLI, func(t *testing.T, home string) (string, string) {
			main := codexPath(home, "rollout-2026-08-01T12-00-00-"+root+".jsonl")
			sub := codexPath(home, "rollout-2026-08-01T12-00-02-22222222-0000-0000-0000-000000000002.jsonl")
			writeFile(t, main, lines(
				`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+root+`","session_id":"`+root+`","cwd":"/w"}}`,
				`{"timestamp":"2026-08-01T12:00:01.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"c1"}}`,
			))
			writeFile(t, sub, lines(
				`{"timestamp":"2026-08-01T12:00:02.000Z","type":"session_meta","payload":{"id":"22222222-0000-0000-0000-000000000002","session_id":"`+root+`","parent_thread_id":"`+root+`","cwd":"/w"}}`,
				`{"timestamp":"2026-08-01T12:00:02.500Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"pwd\"}","call_id":"s1"}}`,
			))
			return main, sub
		}},
		{agentCursor, func(t *testing.T, home string) (string, string) {
			dir := filepath.Join(home, ".cursor", "projects", "tmp-x", "agent-transcripts", "p1")
			main, sub := filepath.Join(dir, "p1.jsonl"), filepath.Join(dir, "subagents", "c1.jsonl")
			writeFile(t, main, lines(
				`{"role":"user","message":{"content":[{"type":"text","text":"go"}]}}`,
				`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Shell","input":{"command":"ls"}}]}}`,
			))
			writeFile(t, sub, lines(
				`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"a"}}]}}`,
			))
			return main, sub
		}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			home := setup(t)
			_, sub := tc.write(t, home)
			clock := setClock(t, time.Now())
			for _, s := range sessionRows(t, eq("agent", tc.agent)) {
				*clock = clock.Add(time.Second)
				rows := messageRows(t, eq("transcript_path", s["transcript_path"]), eq("block_type", blockToolUse))
				require.Len(t, rows, intCol(t, s, "tool_calls"), "tool_use rows for %s match the session's tool_calls", s["transcript_path"])
				for _, r := range rows {
					require.Equal(t, s["transcript_path"], r["transcript_path"])
				}
			}
			*clock = clock.Add(time.Second)
			require.Empty(t, messageRows(t, eq("transcript_path", sub)), "a sub-agent file isn't a session of its own")
		})
	}
}

func TestPagingResumesInsideASubagentFile(t *testing.T) {
	home := setup(t)
	blocks := make([]string, 200)
	for i := range blocks {
		blocks[i] = fmt.Sprintf(`{"type":"text","text":"%d %s"}`, i, strings.Repeat("z", 7000))
	}
	main := writeClaudeTranscript(t, home, "s",
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"role":"user","content":"go"}}`,
	)
	writeFile(t, filepath.Join(claudeProject(home), "s", "subagents", "agent-a1.jsonl"), lines(
		`{"type":"assistant","isSidechain":true,"agentId":"a1","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"a1","content":[`+strings.Join(blocks, ",")+`]}}`,
		`{"type":"user","isSidechain":true,"agentId":"a1","timestamp":"2026-08-01T10:00:02.000Z","message":{"role":"user","content":"sub done"}}`,
	))
	clock := setClock(t, time.Now())

	// Indices run on across the session's files: the sub-agent's message
	// follows the prompt, so the pair stays unique for the whole session.
	type key struct{ m, b int }
	seen := map[key]bool{}
	rows := messageRows(t, eq("transcript_path", main))
	require.NotEmpty(t, rows)
	require.Equal(t, "go", rows[0]["content"])
	for page := 0; page < 10 && len(rows) > 0; page++ {
		for _, r := range rows {
			require.Equal(t, main, r["transcript_path"])
			k := key{intCol(t, r, "message_index"), intCol(t, r, "block_index")}
			require.False(t, seen[k], "row %v returned twice", k)
			seen[k] = true
		}
		last := rows[len(rows)-1]
		m, b := last["message_index"], last["block_index"]
		*clock = clock.Add(time.Second)
		rows = messageRows(t, eq("transcript_path", main), eq("message_index", m), cons{"block_index", table.OperatorGreaterThan, b})
		*clock = clock.Add(10 * time.Millisecond)
		rows = append(rows, messageRows(t, eq("transcript_path", main), cons{"message_index", table.OperatorGreaterThan, m})...)
	}
	require.Len(t, seen, 202, "the prompt, every block of the sub-agent's message, and its last message")
	require.True(t, seen[key{1, 199}])
	require.True(t, seen[key{2, 0}])
}

func TestLimitIsNotHonoured(t *testing.T) {
	home := setup(t)
	writeClaudePrompts(t, home, "s", "a", "b", "c")
	// osquery passes LIMIT down even when it must sort or filter afterwards.
	rows := messageRows(t, cons{"session_id", table.Operator(73), "1"})
	require.Len(t, rows, 3)
}

func TestParseCache(t *testing.T) {
	home := setup(t)
	path := writeClaudePrompts(t, home, "s", "one")

	parsed := transcriptsParsed.Load
	require.Equal(t, "1", sessionRows(t)[0]["user_messages"])
	before := parsed()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString(`{"type":"user","timestamp":"2026-08-01T11:00:00.000Z","message":{"content":"two"}}` + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, "2", sessionRows(t)[0]["user_messages"])
	require.Equal(t, before+1, parsed())

	// A query over a narrower window doesn't evict what it didn't look at.
	old := writeClaudePrompts(t, home, "old", "old")
	touch(t, old, time.Now().AddDate(0, 0, -10))
	sessionRows(t)
	before = parsed()
	require.Len(t, sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, strconv.FormatInt(time.Now().AddDate(0, 0, -1).Unix(), 10)}), 1)
	require.Len(t, sessionRows(t), 2)
	require.Equal(t, before, parsed(), "the narrower query evicted nothing")
}

func TestTopLevelString(t *testing.T) {
	for _, tc := range []struct {
		line, want string
		ok         bool
	}{
		{`{"timestamp":"2026-08-01T10:00:00Z"}`, "2026-08-01T10:00:00Z", true},
		{` { "a" : [1, {"timestamp":"nested"}, "x"] , "b":{"c":"timestamp"}, "timestamp" : "t" } `, "t", true},
		{`{"a":"say \"timestamp\":\"x\"","timestamp":"t\u0041\\"}`, "tA\\", true},
		{`{"a":-1.5e3,"b":true,"c":null,"timestamp":"t"}`, "t", true},
		{`{"timestamp":12345}`, "", false},
		{`{"a":"b"}`, "", false},
		{`{"a":"b",}`, "", false},
		{`{"a":{"b":1}`, "", false},
		{`{"a":"unterminated`, "", false},
		{`[1,2]`, "", false},
		{``, "", false},
	} {
		got, ok := topLevelString([]byte(tc.line), "timestamp")
		require.Equal(t, tc.ok, ok, tc.line)
		require.Equal(t, tc.want, got, tc.line)
	}
}

func TestScanJSONL(t *testing.T) {
	dir := t.TempDir()
	sc := &scope{home: ownedHome(t, dir), reads: newReadBudget()}

	t.Run("skips malformed and oversized records", func(t *testing.T) {
		path := filepath.Join(dir, "r.jsonl")
		writeFile(t, path, strings.Join([]string{
			`{"n":1}`, `not json`, `{"n":99,"pad":"` + strings.Repeat("x", maxRecordBytes) + `"}`, ``, `{"n":2}`,
		}, "\n"))
		var got []int
		require.NoError(t, scanJSONL(sc, path, func(line []byte) bool {
			var r struct{ N int }
			if json.Unmarshal(line, &r) == nil {
				got = append(got, r.N)
			}
			return true
		}))
		require.Equal(t, []int{1, 2}, got)
	})
	t.Run("record larger than the read buffer", func(t *testing.T) {
		path := filepath.Join(dir, "big.jsonl")
		writeFile(t, path, `{"n":7,"pad":"`+strings.Repeat("y", 256<<10)+`"}`+"\n")
		var n int
		require.NoError(t, scanJSONL(sc, path, func(line []byte) bool { n++; return true }))
		require.Equal(t, 1, n)
	})
	t.Run("stops when asked", func(t *testing.T) {
		path := filepath.Join(dir, "stop.jsonl")
		writeFile(t, path, lines(`{}`, `{}`, `{}`))
		var n int
		require.NoError(t, scanJSONL(sc, path, func([]byte) bool { n++; return n < 2 }))
		require.Equal(t, 2, n)
	})
	t.Run("missing file", func(t *testing.T) {
		require.Error(t, scanJSONL(sc, filepath.Join(dir, "absent"), func([]byte) bool { return true }))
	})
}

func TestResolveSlugPath(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Users", "alice", "GitHub", "fleet-extensions"), 0o755))
	for _, tc := range []struct{ slug, want string }{
		{"Users-alice-GitHub-fleet-extensions", filepath.Join(root, "Users", "alice", "GitHub", "fleet-extensions")},
		{"Users-alice-GitHub", filepath.Join(root, "Users", "alice", "GitHub")},
		{"Users-bob-Projects-thing", filepath.Join(root, "Users", "bob", "Projects", "thing")},
		{"-", ""},
	} {
		require.Equal(t, tc.want, resolveSlugPath(root, tc.slug), tc.slug)
	}
}

func TestDurationSeconds(t *testing.T) {
	start := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	require.EqualValues(t, 90, Session{StartedAt: start, EndedAt: start.Add(90 * time.Second)}.durationSeconds())
	require.Zero(t, Session{EndedAt: start}.durationSeconds())
	require.Zero(t, Session{StartedAt: start}.durationSeconds())
	require.Zero(t, Session{StartedAt: start, EndedAt: start.Add(-time.Hour)}.durationSeconds())
}

func TestMissingTimesAreZero(t *testing.T) {
	row := Session{}.row(account{})
	require.Equal(t, "0", row["started_at"])
	require.Equal(t, "0", row["ended_at"])
}

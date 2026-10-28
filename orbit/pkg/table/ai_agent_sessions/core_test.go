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

	// A rewrite that restores size and modification time is still caught.
	fi, err := os.Stat(path)
	require.NoError(t, err)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(content), "two", "TWO", 1)), 0o644))
	touch(t, path, fi.ModTime())
	sessionRows(t)
	require.Equal(t, before+2, parsed())

	// A query over a narrower window doesn't evict what it didn't look at.
	old := writeClaudePrompts(t, home, "old", "old")
	touch(t, old, time.Now().AddDate(0, 0, -10))
	sessionRows(t)
	before = parsed()
	require.Len(t, sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, strconv.FormatInt(time.Now().AddDate(0, 0, -1).Unix(), 10)}), 1)
	require.Len(t, sessionRows(t), 2)
	require.Equal(t, before, parsed(), "the narrower query evicted nothing")
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

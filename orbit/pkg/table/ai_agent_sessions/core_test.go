package ai_agent_sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

// claudePromptsPath is where writeClaudePrompts puts session id's transcript.
func claudePromptsPath(home, id string) string {
	return filepath.Join(home, ".claude", "projects", "-Users-alice-proj", id+".jsonl")
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
	path := claudePromptsPath(home, id)
	writeFile(t, path, lines(ls...))
	return path
}

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
	dir := filepath.Join(home, ".claude", "projects", "-tmp-proj")
	writeFile(t, filepath.Join(dir, "older.jsonl"),
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"hi"}}`+"\n")
	writeFile(t, filepath.Join(dir, "newer.jsonl"),
		`{"type":"user","timestamp":"2026-08-10T10:00:00.000Z","message":{"content":"hi"}}`+"\n")

	rows := sessionRows(t)
	require.Len(t, rows, 2)
	require.Equal(t, "newer", rows[0]["session_id"])
	require.Equal(t, "older", rows[1]["session_id"])
}

func TestSymlinkedTranscriptsAndConfigsAreSkipped(t *testing.T) {
	home := setup(t)
	outside := filepath.Join(t.TempDir(), "secret.jsonl")
	writeFile(t, outside, `{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"root only"}}`+"\n")
	dir := filepath.Join(home, ".claude", "projects", "-tmp-proj")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "linked.jsonl")))

	cfg := filepath.Join(t.TempDir(), "claude.json")
	writeFile(t, cfg, `{"oauthAccount":{"emailAddress":"root@example.com"}}`)
	require.NoError(t, os.Symlink(cfg, filepath.Join(home, ".claude.json")))
	writeClaudePrompts(t, home, "real", "hi")

	rows := sessionRows(t)
	require.Len(t, rows, 1)
	require.Equal(t, "real", rows[0]["session_id"])
	require.Empty(t, rows[0]["account_username"])
	requireAbsent(t, messageRows(t), "root only")
}

func TestDefaultAgeCutoff(t *testing.T) {
	home := setup(t)
	old := writeClaudePrompts(t, home, "old", "old prompt")
	touch(t, old, time.Now().AddDate(0, 0, -40))
	writeClaudePrompts(t, home, "new", "new prompt")

	ids := func(rows []map[string]string) []string {
		var out []string
		for _, r := range rows {
			out = append(out, r["session_id"])
		}
		return out
	}
	since := func(d int) string { return strconv.FormatInt(time.Now().AddDate(0, 0, -d).Unix(), 10) }

	t.Run("default is 30 days", func(t *testing.T) {
		require.Equal(t, []string{"new"}, ids(sessionRows(t)))
		require.Equal(t, []string{"new"}, ids(messageRows(t)))
	})
	t.Run("a lower bound replaces it", func(t *testing.T) {
		require.ElementsMatch(t, []string{"new", "old"}, ids(sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, since(50)})))
		require.ElementsMatch(t, []string{"new", "old"}, ids(messageRows(t, cons{"timestamp", table.OperatorGreaterThanOrEquals, since(50)})))
		require.Empty(t, sessionRows(t, cons{"ended_at", table.OperatorGreaterThan, strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)}))
	})
	t.Run("only an upper bound drops it", func(t *testing.T) {
		require.ElementsMatch(t, []string{"new", "old"}, ids(sessionRows(t, cons{"ended_at", table.OperatorLessThan, since(-1)})))
		require.ElementsMatch(t, []string{"new", "old"}, ids(messageRows(t, cons{"timestamp", table.OperatorLessThanOrEquals, since(-1)})))
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

func TestBlockTypePushdown(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"), lines(
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","message":{"content":"run it"}}`,
		`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m1","content":[{"type":"text","text":"ok"},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"false"}}]}}`,
		`{"type":"user","timestamp":"2026-08-01T10:00:02.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":"Exit code 1"}]}}`,
	))

	rows := messageRows(t, eq("block_type", "tool_use"))
	require.Len(t, rows, 1)
	require.Equal(t, "tool_use", rows[0]["block_type"])
	// The result is never returned, but it still decides the call's outcome.
	require.Equal(t, "error", rows[0]["decision"])
}

func TestBlockIndexIsStableUnderFilters(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-p", "s.jsonl"), lines(
		`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m1","content":[{"type":"text","text":"ok"},{"type":"text","text":""},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"false"}}]}}`,
	))
	// The empty text block yields no row but still takes its index.
	all := messageRows(t, eq("session_id", "s"))
	require.Len(t, all, 2)
	require.Equal(t, "2", all[1]["block_index"])
	only := messageRows(t, eq("session_id", "s"), eq("block_type", "tool_use"))
	require.Len(t, only, 1)
	require.Equal(t, "2", only[0]["block_index"])
}

func TestContentTruncation(t *testing.T) {
	home := setup(t)
	long := strings.Repeat("é", maxBlockBytes) // two bytes per rune
	writeClaudePrompts(t, home, "s1", long)

	t.Run("8 KB per block, UTF-8 safe", func(t *testing.T) {
		r := messageRows(t)[0]
		require.Equal(t, "1", r["truncated"])
		require.Len(t, r["content"], maxBlockBytes)
		require.True(t, strings.HasPrefix(long, r["content"]))
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
		prompts := make([]string, 6000)
		for i := range prompts {
			prompts[i] = "p"
		}
		writeClaudePrompts(t, home, "many", prompts...)
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
		big := strings.Repeat("x", 7000)
		prompts := make([]string, 100) // ~700 KB each
		for i := range prompts {
			prompts[i] = big
		}
		writeClaudePrompts(t, home, "a", prompts...)
		writeClaudePrompts(t, home, "b", prompts...)

		// osquery runs one call per value of session_id IN ('a', 'b').
		first := messageRows(t, eq("session_id", "a"))
		*clock = clock.Add(10 * time.Millisecond)
		second := messageRows(t, eq("session_id", "b"))
		require.Len(t, first, 100)
		require.Less(t, len(second), 100)
		require.LessOrEqual(t, wireBytes(t, first)+wireBytes(t, second), maxResultBytes+1024)

		*clock = clock.Add(budgetIdle + time.Millisecond)
		require.Len(t, messageRows(t, eq("session_id", "b")), 100)
	})

	t.Run("a call after the limit still returns its first row", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, time.Now())
		big := strings.Repeat("x", 7000)
		prompts := make([]string, 200)
		for i := range prompts {
			prompts[i] = big
		}
		writeClaudePrompts(t, home, "a", prompts...)
		writeClaudePrompts(t, home, "b", "one", "two")
		writeClaudePrompts(t, home, "c", "three")

		first := messageRows(t, eq("session_id", "a"))
		require.Less(t, len(first), 200)
		// Later values of an IN list aren't mistaken for empty sessions.
		*clock = clock.Add(10 * time.Millisecond)
		second := messageRows(t, eq("session_id", "b"))
		require.Len(t, second, 1)
		require.Equal(t, "one", second[0]["content"])
		*clock = clock.Add(10 * time.Millisecond)
		require.Len(t, messageRows(t, eq("session_id", "c")), 1)
	})

	t.Run("one oversized block still fits one row", func(t *testing.T) {
		for name, content := range map[string]string{
			"plain":          strings.Repeat("y", 3*maxResultBytes),
			"escaped":        strings.Repeat("<", 600<<10), // six bytes each once JSON encoded
			"JSON arguments": `{"file_path":"/a","content":"` + strings.Repeat("<", 600<<10) + `"}`,
		} {
			t.Run(name, func(t *testing.T) {
				home := setup(t)
				if strings.HasPrefix(content, "{") {
					writeClaudeToolUse(t, home, "huge", "Write", content)
				} else {
					writeClaudePrompts(t, home, "huge", content)
				}
				rows := messageRows(t, eq("session_id", "huge"))
				require.Len(t, rows, 1)
				require.Equal(t, "1", rows[0]["truncated"])
				require.LessOrEqual(t, wireBytes(t, rows), maxResultBytes)
				if strings.HasPrefix(content, "{") {
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
			idx, err := strconv.Atoi(r["message_index"])
			require.NoError(t, err)
			require.Equal(t, from+i, idx, "pages are contiguous")
			seen[idx] = true
		}
		last, _ := strconv.Atoi(rows[len(rows)-1]["message_index"])
		from = last + 1
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
	require.Equal(t, "1", sessionRows(t)[0]["user_messages"])
	require.Equal(t, before, parsed(), "unchanged file is served from the cache")

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
	require.Equal(t, "0", Session{}.row(account{})["started_at"])
}

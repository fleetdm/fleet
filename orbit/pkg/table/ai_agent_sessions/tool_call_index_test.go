package ai_agent_sessions

import (
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

// claudeCall is an assistant record with one Bash call, with its command
// padded so a few hundred of them fill the size limit.
func claudeCall(id, ts string, padding int) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"message":{"id":"m-%s","model":"m","content":[{"type":"tool_use","id":%q,"name":"Bash","input":{"command":"echo %s %s"}}]}}`,
		ts, id, id, id, strings.Repeat("x", padding))
}

func claudeResult(id, ts string) string {
	return fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":%q,"content":"ok"}]}}`, ts, id)
}

// writeClaudeToolCalls writes session id as n Bash calls, each followed by
// its result, one second apart from 2026-08-01T10:00:00Z.
func writeClaudeToolCalls(t *testing.T, home, id string, n, padding int) string {
	t.Helper()
	var ls []string
	for i := range n {
		at := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second)
		callID := fmt.Sprintf("%s-%d", id, i)
		ls = append(ls, claudeCall(callID, at.Format(time.RFC3339Nano), padding), claudeResult(callID, at.Add(500*time.Millisecond).Format(time.RFC3339Nano)))
	}
	return writeClaudeTranscript(t, home, id, ls...)
}

// callIndexes maps each tool call's id to its tool_call_index.
func callIndexes(t *testing.T, rows []map[string]string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, r := range rows {
		require.Equal(t, blockToolUse, r["block_type"])
		out[r["tool_use_id"]] = intCol(t, r, "tool_call_index")
	}
	return out
}

func TestToolCallIndex(t *testing.T) {
	home := setup(t)
	writeClaudeToolCalls(t, home, "s", 3, 0)

	rows := messageRows(t, eq("session_id", "s"))
	require.Len(t, rows, 6)
	require.Equal(t, []string{"0", "", "1", "", "2", ""}, column(rows, "tool_call_index"),
		"tool calls are numbered from 0 in order; other rows have no index")

	for _, tc := range []struct {
		op   table.Operator
		want []string
	}{
		{table.OperatorEquals, []string{"1"}},
		{table.OperatorGreaterThan, []string{"2"}},
		{table.OperatorGreaterThanOrEquals, []string{"1", "2"}},
		{table.OperatorLessThan, []string{"0"}},
		{table.OperatorLessThanOrEquals, []string{"0", "1"}},
	} {
		rows := messageRows(t, eq("session_id", "s"), cons{"tool_call_index", tc.op, "1"})
		require.Equal(t, tc.want, column(rows, "tool_call_index"), "operator %d", tc.op)
		require.Equal(t, slices.Repeat([]string{blockToolUse}, len(rows)), column(rows, "block_type"),
			"a tool_call_index bound selects tool calls only")
	}
}

// Fleet resumes a session with tool_call_index >= the number of calls it has
// stored. Once earlier calls of the same query have spent the shared size
// limit, that call's one admitted row must be the first call Fleet lacks.
func TestToolCallIndexResumesAfterBudgetSpent(t *testing.T) {
	home := setup(t)
	clock := setClock(t, time.Now())
	writeClaudePrompts(t, home, "big", slices.Repeat([]string{strings.Repeat("x", 7000)}, 200)...)
	main := writeClaudeToolCalls(t, home, "calls", 50, 0)

	resume := func(from int) []map[string]string {
		*clock = clock.Add(10 * time.Millisecond)
		return messageRows(t, eq("transcript_path", main), eq("block_type", blockToolUse),
			cons{"tool_call_index", table.OperatorGreaterThanOrEquals, strconv.Itoa(from)})
	}

	require.Less(t, len(messageRows(t, eq("session_id", "big"))), 200, "the first call spends the budget")
	rows := resume(17)
	require.Len(t, rows, 1)
	require.Equal(t, "17", rows[0]["tool_call_index"])
	require.Equal(t, "calls-17", rows[0]["tool_use_id"])

	*clock = clock.Add(budgetIdle + time.Millisecond)
	rows = resume(17)
	require.Len(t, rows, 33)
	for i, r := range rows {
		require.Equal(t, strconv.Itoa(17+i), r["tool_call_index"], "calls are contiguous")
	}
}

// Fleet can start collecting a session at its tool_calls count, so that count
// must be the index the session's next call will get, for every agent.
func TestToolCallsCountMatchesToolCallIndex(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(t *testing.T, home string)
	}{
		{"claude code with sub-agent", func(t *testing.T, home string) { writeClaudeFixture(t, home) }},
		{"codex cli", func(t *testing.T, home string) { writeCodexRollout(t, home, codexLegacy) }},
		{"copilot cli", func(t *testing.T, home string) { writeFile(t, copilotPath(home, "24b9ca9f"), copilotEvents) }},
		{"cursor", func(t *testing.T, home string) { copyCursorCapture(t, home) }},
		{"cursor with sub-agent", func(t *testing.T, home string) {
			dir := filepath.Join(home, ".cursor", "projects", "tmp-x", "agent-transcripts", "p1")
			writeFile(t, filepath.Join(dir, "p1.jsonl"), lines(
				`{"role":"user","message":{"content":[{"type":"text","text":"go"}]}}`,
				`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Shell","input":{"command":"ls"}}]}}`,
			))
			writeFile(t, filepath.Join(dir, "subagents", "c1.jsonl"), lines(
				`{"role":"user","message":{"content":[{"type":"text","text":"sub task"}]}}`,
				`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"a"}}]}}`,
			))
		}},
		{"gemini cli", func(t *testing.T, home string) { writeGeminiFixture(t, home) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := setup(t)
			tc.write(t, home)

			var total int
			for _, s := range sessionRows(t) {
				calls := toolRows(t, s["session_id"])
				require.Equal(t, strconv.Itoa(len(calls)), s["tool_calls"], s["session_id"])
				for i, r := range calls {
					require.Equal(t, strconv.Itoa(i), r["tool_call_index"], s["session_id"])
				}
				total += len(calls)
			}
			require.Positive(t, total, "the fixture has tool calls")
		})
	}
}

// Files past those read side by side are still merged by timestamp, here with
// more sub-agents running at once than that, named against their time order.
func TestToolCallIndexMergesEverySubagentFile(t *testing.T) {
	home := setup(t)
	const subagents, rounds = maxOpenMergedFiles + 8, 3
	at := func(sec int) string {
		return time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(sec) * time.Second).Format(time.RFC3339Nano)
	}

	var want []string
	var main []string
	subs := make([][]string, subagents)
	for round := range rounds {
		id := fmt.Sprintf("main-%d", round)
		main = append(main, claudeCall(id, at(round*1000), 0))
		want = append(want, id)
		for order := range subagents {
			k := subagents - 1 - order
			id := fmt.Sprintf("sub-%02d-%d", k, round)
			subs[k] = append(subs[k], claudeCall(id, at(round*1000+1+order), 0))
			want = append(want, id)
		}
	}
	writeClaudeTranscript(t, home, "s", main...)
	for k, records := range subs {
		writeFile(t, filepath.Join(claudeProject(home), "s", "subagents", fmt.Sprintf("agent-%02d.jsonl", k)), lines(records...))
	}

	calls := toolRows(t, "s")
	require.Equal(t, want, column(calls, "tool_use_id"))
	for i, r := range calls {
		require.Equal(t, strconv.Itoa(i), r["tool_call_index"])
	}
}

// A tool call keeps its index as the session goes on: calls across the
// session's files are ordered by when they were written, so a sub-agent's
// calls don't move when the main transcript grows after them.
func TestToolCallIndexIsStableWhenTheMainTranscriptGrows(t *testing.T) {
	home := setup(t)
	clock := setClock(t, time.Now())
	main := writeClaudeTranscript(t, home, "s",
		claudeCall("a", "2026-08-01T10:00:00.000Z", 0),
		claudeResult("a", "2026-08-01T10:00:00.500Z"),
		`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m-agent","content":[{"type":"tool_use","id":"agent","name":"Agent","input":{"prompt":"explore"}}]}}`,
	)
	writeFile(t, filepath.Join(claudeProject(home), "s", "subagents", "agent-a1.jsonl"), lines(
		`{"type":"user","isSidechain":true,"agentId":"a1","timestamp":"2026-08-01T10:00:01.100Z","message":{"role":"user","content":"explore"}}`,
		claudeCall("s1", "2026-08-01T10:00:02.000Z", 0),
		claudeResult("s1", "2026-08-01T10:00:02.500Z"),
		claudeCall("s2", "2026-08-01T10:00:03.000Z", 0),
		claudeResult("s2", "2026-08-01T10:00:03.500Z"),
	))

	calls := func() []map[string]string {
		*clock = clock.Add(time.Second)
		return messageRows(t, eq("transcript_path", main), eq("block_type", blockToolUse))
	}
	before := callIndexes(t, calls())
	require.Equal(t, map[string]int{"a": 0, "agent": 1, "s1": 2, "s2": 3}, before)

	// The sub-agent reports back and the main agent goes on.
	f, err := os.OpenFile(main, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString(lines(
		claudeResult("agent", "2026-08-01T10:00:04.000Z"),
		claudeCall("b", "2026-08-01T10:00:05.000Z", 0),
		claudeResult("b", "2026-08-01T10:00:05.500Z"),
	))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	after := callIndexes(t, calls())
	require.Equal(t, map[string]int{"a": 0, "agent": 1, "s1": 2, "s2": 3, "b": 4}, after)
	require.Equal(t, "allowed", rowsByToolUseID(calls())["agent"]["decision"], "the sub-agent call's result is still found")
}

func rowsByToolUseID(rows []map[string]string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, r := range rows {
		out[r["tool_use_id"]] = r
	}
	return out
}

// A call the agent recorded without a name or arguments takes an index and
// counts in tool_calls like any other, so it has a row like any other: Fleet
// resumes a session at the number of calls it has stored, and a call with no
// row would make it reread the calls after it forever.
func TestNamelessToolCallHasARow(t *testing.T) {
	for _, tc := range []struct {
		name, agent, id string
		write           func(t *testing.T, home string)
	}{
		{"claude code", agentClaudeCode, "b", func(t *testing.T, home string) {
			writeClaudeTranscript(t, home, "s",
				claudeCall("a", "2026-08-01T10:00:00.000Z", 0),
				`{"type":"assistant","timestamp":"2026-08-01T10:00:01.000Z","message":{"id":"m-b","model":"m","content":[{"type":"tool_use","id":"b"}]}}`,
				claudeCall("c", "2026-08-01T10:00:02.000Z", 0),
			)
		}},
		{"codex cli", agentCodexCLI, "b", func(t *testing.T, home string) {
			writeCodexRollout(t, home, lines(
				`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","timestamp":"2026-08-01T12:00:00.000Z","cwd":"/p"}}`,
				`{"timestamp":"2026-08-01T12:00:01.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"a"}}`,
				`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"function_call","call_id":"b"}}`,
				`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"c"}}`,
			))
		}},
		// A field of the wrong type doesn't lose the call either.
		{"codex cli call with a malformed name", agentCodexCLI, "b", func(t *testing.T, home string) {
			writeCodexRollout(t, home, lines(
				`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","timestamp":"2026-08-01T12:00:00.000Z","cwd":"/p"}}`,
				`{"timestamp":"2026-08-01T12:00:01.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"a"}}`,
				`{"timestamp":"2026-08-01T12:00:02.000Z","type":"response_item","payload":{"type":"function_call","name":5,"call_id":"b"}}`,
				`{"timestamp":"2026-08-01T12:00:03.000Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"ls\"}","call_id":"c"}}`,
			))
		}},
		{"copilot cli", agentCopilotCLI, "b", func(t *testing.T, home string) {
			writeFile(t, copilotPath(home, "s"), lines(
				`{"type":"tool.execution_start","data":{"toolCallId":"a","toolName":"bash","arguments":{"command":"ls"}},"timestamp":"2026-07-06T02:36:33.100Z"}`,
				`{"type":"tool.execution_start","data":{"toolCallId":"b"},"timestamp":"2026-07-06T02:36:33.200Z"}`,
				`{"type":"tool.execution_start","data":{"toolCallId":"c","toolName":"bash","arguments":{"command":"ls"}},"timestamp":"2026-07-06T02:36:33.300Z"}`,
			))
		}},
		{"cursor", agentCursor, "", func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".cursor", "projects", "tmp-x", "agent-transcripts", "s", "s.jsonl"), lines(
				`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Shell","input":{"command":"ls"}},{"type":"tool_use"},{"type":"tool_use","name":"Shell","input":{"command":"ls"}}]}}`,
			))
		}},
		{"gemini cli", agentGeminiCLI, "b", func(t *testing.T, home string) {
			writeFile(t, filepath.Join(home, ".gemini", "tmp", "p", "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl"), lines(
				`{"sessionId":"`+geminiID+`","startTime":"2026-08-01T10:00:00.000Z","lastUpdated":"2026-08-01T10:00:00.000Z","kind":"main"}`,
				`{"id":"m1","timestamp":"2026-08-01T10:00:05.000Z","type":"gemini","content":"","toolCalls":[{"id":"a","name":"glob","args":{"pattern":"*"},"status":"success"},{"id":"b"},{"id":"c","name":"glob","args":{"pattern":"*"},"status":"success"}]}`,
			))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := setup(t)
			tc.write(t, home)

			s := onlySession(t, tc.agent)
			require.Equal(t, "3", s["tool_calls"])
			calls := toolRows(t, s["session_id"])
			require.Equal(t, []string{"0", "1", "2"}, column(calls, "tool_call_index"))
			requireRow(t, map[string]string{"tool_name": "", "tool_use_id": tc.id, "content": "", "content_length": "0"}, calls[1])
			require.Equal(t, calls[1:], messageRows(t, eq("session_id", s["session_id"]), eq("block_type", blockToolUse),
				cons{"tool_call_index", table.OperatorGreaterThanOrEquals, "1"}), "resuming at the call finds it")
		})
	}
}

// Fleet reads a session it is backfilling 100 calls at a time, with
// tool_call_index >= M AND tool_call_index < N. Both bounds are pushed down:
// the result is exactly [M, N), and the calls past N aren't read.
func TestToolCallIndexRangeIsExact(t *testing.T) {
	home := setup(t)
	setReadBudget(t, 2<<20)
	at := func(i int) string {
		return time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano)
	}
	var ls []string
	for i := range 300 {
		padding := 0
		if i >= 200 {
			padding = 60 << 10 // the calls past the range outweigh the read budget
		}
		id := strconv.Itoa(i)
		ls = append(ls, claudeCall(id, at(i), padding), claudeResult(id, at(i)))
	}
	main := writeClaudeTranscript(t, home, "s", ls...)
	logs := captureLog(t)

	rows := messageRows(t, eq("transcript_path", main),
		cons{"tool_call_index", table.OperatorGreaterThanOrEquals, "100"},
		cons{"tool_call_index", table.OperatorLessThan, "200"})
	require.Len(t, rows, 100)
	for i, r := range rows {
		require.Equal(t, blockToolUse, r["block_type"])
		require.Equal(t, strconv.Itoa(100+i), r["tool_call_index"])
		require.Equal(t, strconv.Itoa(100+i), r["tool_use_id"])
	}
	require.NotContains(t, logs.String(), "read budget", "reading stops past the upper bound")
}

// Merging more files than are kept open reopens one for nearly every
// record. A reopen reads about that record, not a whole scan buffer of
// records the stream is closed before using: what it reads is charged to
// the read budget, which a session could otherwise spend many times over.
func TestMergeReopenReadsAboutOneRecord(t *testing.T) {
	home := setup(t)
	const files, rounds = maxOpenMergedFiles + 8, 200
	at := func(ms int) string {
		return time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(ms) * time.Millisecond).Format(time.RFC3339Nano)
	}
	var paths []string
	var total int64
	for k := range files {
		var records []string
		for round := range rounds {
			records = append(records, claudeCall(fmt.Sprintf("%02d-%d", k, round), at(round*1000+k), 900))
		}
		p := filepath.Join(claudeProject(home), "s", "subagents", fmt.Sprintf("agent-%02d.jsonl", k))
		writeFile(t, p, lines(records...))
		paths = append(paths, p)
		fi, ok := regularFile(p)
		require.True(t, ok)
		total += fi.Size()
	}

	sc := &scope{home: ownedHome(t, home), reads: newReadBudget()}
	n := 0
	scanFiles(sc, paths, recordStamp, func(int, []byte) bool { n++; return true })
	require.Equal(t, files*rounds, n)
	require.False(t, sc.reads.exhausted)
	read := maxReadBytesPerCall - sc.reads.left
	require.LessOrEqual(t, read, 4*total, "read %d bytes to merge %d", read, total)
}

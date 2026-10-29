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

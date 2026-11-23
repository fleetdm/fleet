package ai_agent_sessions

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func epoch(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }

// A timestamp bound selects rows as well as transcripts. Blocks outside it
// don't spend the size limit, which would otherwise stop the result before
// it reached the rows asked for.
func TestTimestampBoundSelectsRows(t *testing.T) {
	// many is more calls than one result holds.
	const many, few = 1000, 40
	at := time.Now().UTC()
	lastDay := cons{"timestamp", table.OperatorGreaterThan, epoch(at.Add(-24 * time.Hour))}

	t.Run("lower bound on a long session", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, at)
		writeClaudeTranscript(t, home, "long", append(
			claudeCalls("long", 0, many, at.AddDate(0, 0, -10), 700),
			claudeCalls("long", many, few, at.Add(-10*time.Minute), 700)...)...)
		require.Less(t, len(toolRows(t, "long")), many, "the older calls alone fill a result")
		logs := captureLog(t)

		for _, cs := range [][]cons{
			{eq("session_id", "long"), eq("block_type", blockToolUse), lastDay},
			{eq("block_type", blockToolUse), lastDay},
		} {
			*clock = clock.Add(time.Second)
			rows := messageRows(t, cs...)
			require.Len(t, rows, few)
			for i, r := range rows {
				require.Equal(t, strconv.Itoa(many+i), r["tool_call_index"], "a call keeps its position in the session")
				require.Equal(t, decisionAllowed, r["decision"])
			}
		}
		require.NotContains(t, logs.String(), "size limit")
	})

	t.Run("upper bound past a busy recent session", func(t *testing.T) {
		home := setup(t)
		setClock(t, at)
		quiet := writeClaudeTranscript(t, home, "quiet", claudeCalls("quiet", 0, few, at.AddDate(0, 0, -10), 700)...)
		touch(t, quiet, at.AddDate(0, 0, -10))
		// The newest transcript is read first, and nothing in it is that old.
		writeClaudeTranscript(t, home, "busy", claudeCalls("busy", 0, many, at.Add(-2*time.Hour), 700)...)

		rows := messageRows(t, eq("block_type", blockToolUse), cons{"timestamp", table.OperatorLessThan, epoch(at.AddDate(0, 0, -5))})
		require.Equal(t, slices.Repeat([]string{"quiet"}, few), column(rows, "session_id"))
	})

	t.Run("a cut inside the bound is read on by tool_call_index", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, at)
		writeClaudeTranscript(t, home, "long", append(
			claudeCalls("long", 0, many, at.AddDate(0, 0, -10), 700),
			claudeCalls("long", many, many, at.Add(-time.Hour), 700)...)...)

		read, last := 0, -1
		for page := 0; page < 5 && read < many; page++ {
			*clock = clock.Add(time.Second)
			rows := messageRows(t, eq("session_id", "long"), eq("block_type", blockToolUse), lastDay,
				cons{"tool_call_index", table.OperatorGreaterThan, strconv.Itoa(last)})
			require.NotEmpty(t, rows)
			for i, r := range rows {
				require.Equal(t, strconv.Itoa(many+read+i), r["tool_call_index"], "pages are contiguous")
			}
			read += len(rows)
			last = intCol(t, rows[len(rows)-1], "tool_call_index")
		}
		require.Equal(t, many, read)
	})

	t.Run("every operator, on the second a row reports", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, at)
		// Three calls a second apart. A result is written half a second after
		// its call, and reports the same second.
		writeClaudeToolCalls(t, home, "s", 3, 0)
		second := epoch(time.Date(2026, 8, 1, 10, 0, 1, 0, time.UTC))

		for _, tc := range []struct {
			op   table.Operator
			want []string
		}{
			{table.OperatorEquals, []string{"2", "3"}},
			{table.OperatorGreaterThan, []string{"4", "5"}},
			{table.OperatorGreaterThanOrEquals, []string{"2", "3", "4", "5"}},
			{table.OperatorLessThan, []string{"0", "1"}},
			{table.OperatorLessThanOrEquals, []string{"0", "1", "2", "3"}},
		} {
			*clock = clock.Add(time.Second)
			rows := messageRows(t, eq("session_id", "s"), cons{"timestamp", tc.op, second})
			require.Equal(t, tc.want, column(rows, "message_index"), "operator %d", tc.op)
		}
	})

	t.Run("an undated row is at 0, as it reports", func(t *testing.T) {
		home := setup(t)
		clock := setClock(t, at)
		writeClaudeTranscript(t, home, "s",
			`{"type":"user","message":{"role":"user","content":"undated"}}`,
			claudePromptAt(at.Add(-time.Minute), "dated"))

		rows := messageRows(t, eq("session_id", "s"), cons{"timestamp", table.OperatorLessThanOrEquals, "0"})
		require.Equal(t, []string{"undated"}, column(rows, "content"))
		*clock = clock.Add(time.Second)
		rows = messageRows(t, eq("session_id", "s"), cons{"timestamp", table.OperatorGreaterThan, "0"})
		require.Equal(t, []string{"dated"}, column(rows, "content"))
	})
}

// Every agent's rows carry their message's time, so each one's bounded result
// is the rows osquery would keep of the unbounded one.
func TestTimestampBoundForEveryAgent(t *testing.T) {
	for name, write := range sessionFixtures {
		t.Run(name, func(t *testing.T) {
			home := setup(t)
			write(t, home)
			clock := setClock(t, time.Now())
			all := messageRows(t, eq("agent", name))
			require.NotEmpty(t, all)

			kept := func(keep func(ts int) bool) []map[string]string {
				var out []map[string]string
				for _, r := range all {
					if keep(intCol(t, r, "timestamp")) {
						out = append(out, r)
					}
				}
				return out
			}
			stamps := column(all, "timestamp")
			slices.Sort(stamps)
			for _, v := range slices.Compact(stamps) {
				k, err := strconv.Atoi(v)
				require.NoError(t, err)
				*clock = clock.Add(time.Second)
				require.Equal(t, kept(func(ts int) bool { return ts >= k }),
					messageRows(t, eq("agent", name), cons{"timestamp", table.OperatorGreaterThanOrEquals, v}), ">= %s", v)
				*clock = clock.Add(time.Second)
				require.Equal(t, kept(func(ts int) bool { return ts <= k }),
					messageRows(t, eq("agent", name), cons{"timestamp", table.OperatorLessThanOrEquals, v}), "<= %s", v)
			}
		})
	}
}

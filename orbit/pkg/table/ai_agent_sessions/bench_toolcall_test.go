package ai_agent_sessions

import (
	"strconv"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
)

// BenchmarkResumeToolCallIndex resumes after the 1500th call the way Fleet
// does with tool_call_index: one arm, from the first call it lacks.
func BenchmarkResumeToolCallIndex(b *testing.B) {
	path := benchSession(b)
	from := strconv.Itoa(benchRecords / 2 * benchCallsPerRecord)
	for b.Loop() {
		messageBudget.reset()
		if n := len(benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse), cons{"tool_call_index", table.OperatorGreaterThanOrEquals, from})); n == 0 {
			b.Fatal("no rows")
		}
	}
}

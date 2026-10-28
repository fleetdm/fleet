package ai_agent_sessions

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

// Benchmarks of what one detail refresh costs fleetd: Fleet asks for the tool
// calls of a session, and for a session it has partly stored, from where it
// left off. Rows carry ~2 KB tool inputs, so the 1 MiB result cap admits a
// few hundred calls per read.

const (
	benchRecords        = 1500 // assistant records, each with benchCallsPerRecord calls
	benchCallsPerRecord = 2
	benchContentBytes   = 2048
)

func setupBench(b *testing.B) string {
	b.Helper()
	home := b.TempDir()
	uid, _, ok := homes.Owner(home)
	if !ok {
		uid = "unknown"
	}
	orig := listHomes
	listHomes = func() []homes.Home { return []homes.Home{{UID: uid, Username: "alice", Dir: home}} }
	b.Cleanup(func() { listHomes = orig })
	resetState()
	b.Cleanup(resetState)
	level := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	b.Cleanup(func() { zerolog.SetGlobalLevel(level) })
	return home
}

// benchClaudeRecords writes records assistant turns of calls calls each, with
// results, timestamps starting at start and step apart; ids are prefixed.
func benchClaudeRecords(prefix string, records, calls int, start time.Time, step time.Duration, sub bool) []string {
	side := ""
	if sub {
		side = `"isSidechain":true,`
	}
	padding := strings.Repeat("x", benchContentBytes)
	var ls []string
	for i := range records {
		at := start.Add(time.Duration(i) * step)
		var uses, results []string
		for c := range calls {
			id := fmt.Sprintf("%s-%d-%d", prefix, i, c)
			uses = append(uses, fmt.Sprintf(`{"type":"tool_use","id":%q,"name":"Bash","input":{"command":"echo %s %s"}}`, id, id, padding))
			results = append(results, fmt.Sprintf(`{"type":"tool_result","tool_use_id":%q,"content":"ok"}`, id))
		}
		ls = append(ls,
			fmt.Sprintf(`{"type":"assistant",%s"timestamp":%q,"message":{"id":"m-%s-%d","model":"m","content":[%s]}}`, side, at.Format(time.RFC3339Nano), prefix, i, strings.Join(uses, ",")),
			fmt.Sprintf(`{"type":"user",%s"timestamp":%q,"message":{"role":"user","content":[%s]}}`, side, at.Add(step/2).Format(time.RFC3339Nano), strings.Join(results, ",")))
	}
	return ls
}

func benchRows(b *testing.B, cs ...cons) []map[string]string {
	b.Helper()
	rows, err := GenerateMessages(b.Context(), qctx(cs...))
	if err != nil {
		b.Fatal(err)
	}
	return rows
}

func benchSession(b *testing.B) string {
	b.Helper()
	home := setupBench(b)
	start := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	path := claudePath(home, "s")
	writeFile(b, path, lines(benchClaudeRecords("m", benchRecords, benchCallsPerRecord, start, 5*time.Second, false)...))
	return path
}

// BenchmarkFirstRead is the first read of a session: from its first call
// until the result cap.
func BenchmarkFirstRead(b *testing.B) {
	path := benchSession(b)
	for b.Loop() {
		messageBudget.reset()
		if n := len(benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse))); n == 0 || n == benchRecords*benchCallsPerRecord {
			b.Fatalf("got %d rows", n)
		}
	}
}

// BenchmarkResumeTwoArms resumes after the 1500th call with the two
// single-column arms Fleet can send without a tool_call_index: the rest of
// the message it stopped in, then the messages after it.
func BenchmarkResumeTwoArms(b *testing.B) {
	path := benchSession(b)
	last := benchRecords/2 - 1 // assistant record of the last stored call
	m, blk := strconv.Itoa(2*last), strconv.Itoa(benchCallsPerRecord-1)
	for b.Loop() {
		messageBudget.reset()
		rest := benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse), eq("message_index", m), cons{"block_index", table.OperatorGreaterThan, blk})
		after := benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse), cons{"message_index", table.OperatorGreaterThan, m})
		if len(rest) != 0 || len(after) == 0 {
			b.Fatalf("got %d + %d rows", len(rest), len(after))
		}
	}
}

// BenchmarkResumeMessageIndex is what the OR form actually asks fleetd: from
// the message the last stored call is in, re-sending the rest of it.
func BenchmarkResumeMessageIndex(b *testing.B) {
	path := benchSession(b)
	m := strconv.Itoa(2 * (benchRecords/2 - 1))
	for b.Loop() {
		messageBudget.reset()
		if n := len(benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse), cons{"message_index", table.OperatorGreaterThanOrEquals, m})); n == 0 {
			b.Fatal("no rows")
		}
	}
}

// benchMultiFileSession is a session of the same size spread over a main
// transcript and four sub-agents whose records interleave with it.
func benchMultiFileSession(b *testing.B) string {
	b.Helper()
	home := setupBench(b)
	start := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	const files = 5
	path := claudePath(home, "s")
	writeFile(b, path, lines(benchClaudeRecords("m", benchRecords/files, benchCallsPerRecord, start, 5*time.Second, false)...))
	for k := 1; k < files; k++ {
		writeFile(b, filepath.Join(claudeProject(home), "s", "subagents", fmt.Sprintf("agent-%d.jsonl", k)),
			lines(benchClaudeRecords(fmt.Sprintf("a%d", k), benchRecords/files, benchCallsPerRecord, start.Add(time.Duration(k)*time.Second), 5*time.Second, true)...))
	}
	return path
}

func BenchmarkFirstReadMultiFile(b *testing.B) {
	path := benchMultiFileSession(b)
	for b.Loop() {
		messageBudget.reset()
		if n := len(benchRows(b, eq("transcript_path", path), eq("block_type", blockToolUse))); n == 0 {
			b.Fatal("no rows")
		}
	}
}

// BenchmarkSessionsScan is the scheduled ai_agent_sessions read of the
// session, parsed once (the cache is cleared each time).
func BenchmarkSessionsScan(b *testing.B) {
	benchMultiFileSession(b)
	for b.Loop() {
		resetState()
		rows, err := GenerateSessions(b.Context(), qctx())
		if err != nil || len(rows) != 1 {
			b.Fatalf("%d rows, %v", len(rows), err)
		}
	}
}

package ai_agent_sessions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

// setup gives the test one empty home owned by alice and a clean cache,
// budget and clock. It returns the home directory.
func setup(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	setHomes(t, ownedHome(t, home))
	resetState()
	t.Cleanup(resetState)
	return home
}

// ownedHome is alice's home at dir, owned (as the tables require of every file
// they read) by whoever runs the tests.
func ownedHome(t *testing.T, dir string) homes.Home {
	t.Helper()
	uid, _, ok := homes.Owner(dir)
	if !ok {
		uid = "unknown"
	}
	return homes.Home{UID: uid, Username: "alice", Dir: dir}
}

func setHomes(t *testing.T, hs ...homes.Home) {
	t.Helper()
	orig := listHomes
	listHomes = func() []homes.Home { return hs }
	t.Cleanup(func() { listHomes = orig })
}

// setClock replaces the clock with one the test advances by hand.
func setClock(t *testing.T, start time.Time) *time.Time {
	t.Helper()
	cur := start
	orig := now
	now = func() time.Time { return cur }
	t.Cleanup(func() { now = orig })
	return &cur
}

// captureLog sends the global logger to a buffer for the test's duration.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := log.Logger
	log.Logger = zerolog.New(&buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = orig })
	return &buf
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func touch(t *testing.T, path string, mt time.Time) {
	t.Helper()
	require.NoError(t, os.Chtimes(path, mt, mt))
}

func lines(ls ...string) string { return strings.Join(ls, "\n") + "\n" }

type cons struct {
	col string
	op  table.Operator
	val string
}

func eq(col, val string) cons { return cons{col, table.OperatorEquals, val} }

func qctx(cs ...cons) table.QueryContext {
	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{}}
	for _, c := range cs {
		l := qc.Constraints[c.col]
		l.Constraints = append(l.Constraints, table.Constraint{Operator: c.op, Expression: c.val})
		qc.Constraints[c.col] = l
	}
	return qc
}

func sessionRows(t *testing.T, cs ...cons) []map[string]string {
	t.Helper()
	rows, err := GenerateSessions(t.Context(), qctx(cs...))
	require.NoError(t, err)
	return rows
}

func messageRows(t *testing.T, cs ...cons) []map[string]string {
	t.Helper()
	rows, err := GenerateMessages(t.Context(), qctx(cs...))
	require.NoError(t, err)
	return rows
}

// onlySession returns the single session row for agent.
func onlySession(t *testing.T, agent string) map[string]string {
	t.Helper()
	rows := sessionRows(t, eq("agent", agent))
	require.Len(t, rows, 1)
	return rows[0]
}

// toolRows returns the tool_use rows of a session, in order.
func toolRows(t *testing.T, sessionID string) []map[string]string {
	t.Helper()
	return messageRows(t, eq("session_id", sessionID), eq("block_type", "tool_use"))
}

// render shows each row as "role/block_type" followed by the named columns,
// prefixed with "message_index.block_index" when indexed is set.
func render(rows []map[string]string, indexed bool, cols ...string) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		s := r["role"] + "/" + r["block_type"]
		if indexed {
			s = r["message_index"] + "." + r["block_index"] + " " + s
		}
		for _, c := range cols {
			s += " " + r[c]
		}
		out = append(out, s)
	}
	return out
}

// rowsByID indexes rows by session_id.
func rowsByID(rows []map[string]string) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, r := range rows {
		out[r["session_id"]] = r
	}
	return out
}

// requireRow checks the columns named in want.
func requireRow(t *testing.T, want, row map[string]string) {
	t.Helper()
	for col, v := range want {
		require.Equal(t, v, row[col], col)
	}
}

// summary renders the event columns of tool_use rows for compact comparison.
func summary(rows []map[string]string) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r["tool_name"]+" | "+r["action"]+" | "+r["target"]+" | "+r["decision"])
	}
	return out
}

// wireBytes is how large rows are as JSON, which is how they leave the host.
func wireBytes(t *testing.T, rows []map[string]string) int {
	t.Helper()
	b, err := json.Marshal(rows)
	require.NoError(t, err)
	return len(b)
}

// requireAbsent fails if s appears in any column of any row.
func requireAbsent(t *testing.T, rows []map[string]string, s string) {
	t.Helper()
	for _, r := range rows {
		for col, v := range r {
			require.NotContains(t, v, s, "column %s", col)
		}
	}
}

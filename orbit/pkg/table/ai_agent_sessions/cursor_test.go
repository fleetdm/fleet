package ai_agent_sessions

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	cursorSession = "d038b33d-9a98-4916-bade-b2d745d16afc"
	cursorAborted = "422caa19-fa3d-456b-8588-cd7bf524a337"
)

// copyCursorCapture installs the sanitised Cursor CLI capture under home.
func copyCursorCapture(t *testing.T, home string) {
	t.Helper()
	require.NoError(t, os.CopyFS(filepath.Join(home, ".cursor"), os.DirFS(filepath.Join("testdata", "cursor"))))
}

func TestCursorSessions(t *testing.T) {
	home := setup(t)
	copyCursorCapture(t, home)

	rows := sessionRows(t, eq("agent", agentCursor))
	require.Len(t, rows, 2)
	byID := rowsByID(rows)
	requireRow(t, map[string]string{
		"project_path":       "/home/alice/code",
		"started_at":         "1793130357",
		"ended_at":           "1793130562",
		"duration_seconds":   "205",
		"user_messages":      "9",
		"assistant_messages": "18",
		// The two schema lookups before dynamic tool calls aren't events.
		"tool_calls":   "9",
		"model":        "",
		"input_tokens": "0",
		"transcript_path": filepath.Join(home, ".cursor", "projects", "home-alice-code", "agent-transcripts",
			cursorSession, cursorSession+".jsonl"),
	}, byID[cursorSession])

	a := byID[cursorAborted]
	require.Equal(t, "1", a["user_messages"])
	require.Equal(t, "1", a["tool_calls"])
}

func TestCursorSessionWithoutMeta(t *testing.T) {
	home := setup(t)
	copyCursorCapture(t, home)
	require.NoError(t, os.RemoveAll(filepath.Join(home, ".cursor", "chats")))

	s := sessionRows(t, eq("session_id", cursorSession))[0]
	require.Equal(t, "/home/alice/code", s["project_path"])
	ended, err := strconv.ParseInt(s["ended_at"], 10, 64)
	require.NoError(t, err)
	require.Positive(t, ended)
	started, err := strconv.ParseInt(s["started_at"], 10, 64)
	require.NoError(t, err)
	require.LessOrEqual(t, started, ended)
}

func TestCursorMessages(t *testing.T) {
	home := setup(t)
	copyCursorCapture(t, home)

	rows := messageRows(t, eq("session_id", cursorSession))
	require.Equal(t, "user", rows[0]["role"])
	// The timestamp and user_query wrappers are stripped.
	require.Equal(t, "Read README.md and summarise it.", rows[0]["content"])
	require.Equal(t, "1793130420", rows[0]["timestamp"])
	// The turn's assistant rows inherit the prompt's time.
	require.Equal(t, "1793130420", rows[1]["timestamp"])
	require.Equal(t, "/home/alice/code", rows[1]["cwd"])
	for _, r := range rows {
		require.NotEqual(t, "GetDynamicTools", r["tool_name"])
	}

	require.Equal(t, []string{
		"Read | read | /home/alice/code/README.md | ",
		"Glob | search | **/*.{ts,tsx} in /home/alice/code/src | ",
		`Grep | search | "refreshToken" | `,
		"Shell | execute | cat .env.local | grep API_ | ",
		"Shell | execute | ls -la | ",
		"Write | write | /home/alice/code/NOTES.md | ",
		"WebFetch | network | https://example.com | ",
		"echo | mcp | everything: echo | ",
		"Shell | execute | false | ",
	}, summary(toolRows(t, cursorSession)))

	last := messageRows(t, eq("session_id", cursorSession), eq("block_type", "text"))
	require.Equal(t, "Echo: hello", last[len(last)-3]["content"])
	require.Equal(t, "1793130540", last[len(last)-1]["timestamp"])
}

func TestCursorUnparseableTimestamp(t *testing.T) {
	home := setup(t)
	id := "abc"
	writeFile(t, filepath.Join(home, ".cursor", "projects", "tmp-x", "agent-transcripts", id, id+".jsonl"), lines(
		`{"role":"user","message":{"content":[{"type":"text","text":"<timestamp>sometime</timestamp>\n<user_query>\nhi\n</user_query>"}]}}`,
		`{"role":"user","message":{"content":[{"type":"text","text":"no wrapper"}]}}`,
	))
	rows := messageRows(t, eq("session_id", id))
	require.Len(t, rows, 2)
	require.Equal(t, "hi", rows[0]["content"])
	require.Equal(t, "0", rows[0]["timestamp"])
	require.Equal(t, "no wrapper", rows[1]["content"])
}

func TestCursorSubagentAndLegacyLayout(t *testing.T) {
	home := setup(t)
	dir := filepath.Join(home, ".cursor", "projects", "tmp-x", "agent-transcripts")
	writeFile(t, filepath.Join(dir, "p1", "p1.jsonl"), lines(
		`{"role":"user","message":{"content":[{"type":"text","text":"go"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Shell","input":{"command":"ls"}}]}}`,
	))
	writeFile(t, filepath.Join(dir, "p1", "subagents", "c1.jsonl"), lines(
		`{"role":"user","message":{"content":[{"type":"text","text":"sub task"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"a"}}]}}`,
	))
	writeFile(t, filepath.Join(dir, "flat.jsonl"), lines(
		`{"role":"user","message":{"content":[{"type":"text","text":"old layout"}]}}`,
	))

	byID := rowsByID(sessionRows(t, eq("agent", agentCursor)))
	require.Len(t, byID, 2)
	require.Equal(t, "1", byID["p1"]["user_messages"])
	require.Equal(t, "2", byID["p1"]["tool_calls"])
	require.Equal(t, "1", byID["flat"]["user_messages"])
}

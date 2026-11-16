package osquery_utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestAISessionEventPayload(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"object", `{"command": "ls -la", "timeout": 5}`, `{"name":"Bash","id":"toolu_1","input":{"command":"ls -la","timeout":5}}`},
		{"array", `["a", "b"]`, `{"name":"Bash","id":"toolu_1","input":["a","b"]}`},
		{"shortened object", `{"content":"aaaa…"}`, `{"name":"Bash","id":"toolu_1","input":{"content":"aaaa…"}}`},
		{"plain text", `ls -la`, `{"name":"Bash","id":"toolu_1","input":"ls -la"}`},
		{"cut object", `{"content": "aaa`, `{"name":"Bash","id":"toolu_1","input":"{\"content\": \"aaa"}`},
		{"scalar", `42`, `{"name":"Bash","id":"toolu_1","input":"42"}`},
		{"empty", ``, `{"name":"Bash","id":"toolu_1","input":""}`},
		{"nested to the limit", nested(maxAISessionJSONDepth - 1), `{"name":"Bash","id":"toolu_1","input":` + nested(maxAISessionJSONDepth-1) + `}`},
		{"nested to the limit with a value", nestedValue(maxAISessionJSONDepth - 1), `{"name":"Bash","id":"toolu_1","input":` + nestedValue(maxAISessionJSONDepth-1) + `}`},
		// MySQL rejects a JSON document nested deeper than the limit, counting only objects and arrays.
		{"nested past the limit", nested(maxAISessionJSONDepth), `{"name":"Bash","id":"toolu_1","input":"` + nested(maxAISessionJSONDepth) + `"}`},
		// Valid JSON that MySQL rejects: a number past a double's range, and a lone UTF-16 surrogate.
		{"number too big", `{"a":1e999}`, `{"name":"Bash","id":"toolu_1","input":"{\"a\":1e999}"}`},
		{"negative number too big", `[-1e400]`, `{"name":"Bash","id":"toolu_1","input":"[-1e400]"}`},
		{"number too small", `[1e-999]`, `{"name":"Bash","id":"toolu_1","input":[1e-999]}`},
		{"lone surrogate", `{"a":"\ud800 x"}`, `{"name":"Bash","id":"toolu_1","input":{"a":"� x"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := aiSessionEventPayload("Bash", "toolu_1", c.content)
			require.True(t, json.Valid(got), string(got))
			require.JSONEq(t, c.want, string(got))
			// JSONEq decodes surrogate escapes, so they're checked on the bytes MySQL gets.
			require.NotRegexp(t, `\\u[dD][89a-fA-F]`, string(got))
		})
	}
}

// nested is a JSON array nested depth levels deep; nestedValue holds a number at the bottom.
func nested(depth int) string {
	return strings.Repeat("[", depth) + strings.Repeat("]", depth)
}

func nestedValue(depth int) string {
	return strings.Repeat("[", depth) + "1" + strings.Repeat("]", depth)
}

// TestAIAgentSessionsDiscovery runs the discovery queries against sqlite: a fleetd whose tables lack the columns Fleet
// reads must not be asked for them.
func TestAIAgentSessionsDiscovery(t *testing.T) {
	for _, c := range []struct {
		name      string
		discovery string
		table     string
		column    string
	}{
		{"sessions", aiAgentSessions.Discovery, "ai_agent_sessions", "account_identifier"},
		{"events", aiAgentSessionEventsQuery(0, 0).Discovery, "ai_agent_session_messages", "tool_call_index"},
	} {
		t.Run(c.name, func(t *testing.T) {
			discover := func(columns string) bool {
				db, err := sql.Open("sqlite3", ":memory:")
				require.NoError(t, err)
				defer db.Close()
				_, err = db.Exec(`CREATE TABLE osquery_registry (registry TEXT, name TEXT, active INTEGER)`)
				require.NoError(t, err)
				if columns != "" {
					_, err = db.Exec(`CREATE TABLE ` + c.table + ` (` + columns + `)`)
					require.NoError(t, err)
					_, err = db.Exec(`INSERT INTO osquery_registry VALUES ('table', ?, 1)`, c.table)
					require.NoError(t, err)
				}
				rows, err := db.Query(c.discovery)
				require.NoError(t, err)
				defer rows.Close()
				found := rows.Next()
				require.NoError(t, rows.Err())
				return found
			}
			require.False(t, discover(""))
			require.False(t, discover("agent TEXT, session_id TEXT"))
			require.True(t, discover("agent TEXT, session_id TEXT, "+c.column+" TEXT"))
		})
	}
}

// TestAIAgentSessionsQuery runs the sessions query against sqlite to check that only sessions active in the last 7
// days, by the host's clock, are reported, however long ago they started.
func TestAIAgentSessionsQuery(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE ai_agent_sessions (
		agent TEXT, session_id TEXT, username TEXT, account_identifier TEXT, project_path TEXT, git_branch TEXT,
		git_repository TEXT, agent_version TEXT, model TEXT, started_at BIGINT, ended_at BIGINT, duration_seconds BIGINT,
		user_messages BIGINT, assistant_messages BIGINT, tool_calls BIGINT, input_tokens BIGINT, output_tokens BIGINT,
		cache_read_tokens BIGINT, cache_write_tokens BIGINT, transcript_path TEXT, transcript_size_bytes BIGINT)`)
	require.NoError(t, err)
	// The host's clock, not Fleet's.
	hostNow := int64(1_800_000_000)
	_, err = db.Exec(`CREATE TABLE time (unix_time BIGINT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO time VALUES (?)`, hostNow)
	require.NoError(t, err)
	const week = 7 * 24 * 3600
	for id, times := range map[string][2]int64{
		"recent":       {hostNow - 7200, hostNow - 3600},
		"long_running": {hostNow - 10*week, hostNow - 60},
		"at_cutoff":    {hostNow - 2*week, hostNow - week},
		"idle":         {hostNow - 2*week, hostNow - week - 1},
		"unknown_end":  {hostNow - 3600, 0},
		"future_skew":  {hostNow - 3600, hostNow + 600},
	} {
		_, err = db.Exec(`INSERT INTO ai_agent_sessions (agent, session_id, started_at, ended_at) VALUES ('claude_code', ?, ?, ?)`,
			id, times[0], times[1])
		require.NoError(t, err)
	}

	rows, err := db.Query(aiAgentSessions.Query)
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var ids []string
	for rows.Next() {
		values := make([]sql.RawBytes, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		require.NoError(t, rows.Scan(dest...))
		ids = append(ids, string(values[slices.Index(cols, "session_id")]))
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, []string{"recent", "long_running", "at_cutoff", "future_skew"}, ids)
}

func TestBuildAIAgentSessionEventsQuery(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	host := &fleet.Host{ID: 1}
	endedAt := time.Unix(1_800_000_000, 0).UTC()
	const maxEvents = 40

	ds := new(mock.Store)
	var cursors []*fleet.HostAISessionEventsCursor
	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int, maxEventsArg uint) ([]*fleet.HostAISessionEventsCursor, error) {
		require.EqualValues(t, 1, hostID)
		require.Equal(t, aiSessionsPerEventsQuery, limit)
		require.EqualValues(t, maxEvents, maxEventsArg)
		return cursors, nil
	}

	_, ok := buildAIAgentSessionEventsQuery(ctx, logger, host, ds, maxEvents, 0)
	require.False(t, ok)

	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int, maxEvents uint) ([]*fleet.HostAISessionEventsCursor, error) {
		return nil, errors.New("boom")
	}
	_, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds, maxEvents, 0)
	require.False(t, ok)

	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int, maxEvents uint) ([]*fleet.HostAISessionEventsCursor, error) {
		return cursors, nil
	}
	cursors = []*fleet.HostAISessionEventsCursor{
		// Never read.
		{Agent: "claude_code", AgentSessionID: "fresh", TranscriptPath: "/u/a/fresh.jsonl", EndedAt: &endedAt},
		// The last result was cut after three calls.
		{Agent: "claude_code", AgentSessionID: "cut", TranscriptPath: "/u/a/cut.jsonl", EndedAt: &endedAt, EventCount: 3},
		// Caught up, with an undecided call at 2, and active since.
		{Agent: "codex_cli", AgentSessionID: "undecided", TranscriptPath: "/u/a/undecided.jsonl", EventCount: 6, RereadFromToolCallIndex: new(uint(2))},
		// Paths come from the host.
		{Agent: "claude_code", AgentSessionID: "quoted", TranscriptPath: "/u/o'brien/x' OR '1'='1.jsonl", EndedAt: &endedAt},
	}
	query, ok := buildAIAgentSessionEventsQuery(ctx, logger, host, ds, maxEvents, 0)
	require.True(t, ok)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE ai_agent_session_messages (
		agent TEXT, session_id TEXT, username TEXT, message_index BIGINT, block_index BIGINT, tool_call_index BIGINT,
		role TEXT, block_type TEXT, tool_name TEXT, tool_use_id TEXT, action TEXT, target TEXT, decision TEXT, cwd TEXT,
		timestamp BIGINT, model TEXT, content TEXT, content_length BIGINT, truncated INTEGER, transcript_path TEXT)`)
	require.NoError(t, err)
	// Only tool_use rows have a tool_call_index.
	insert := func(sessionID, path string, toolCallIndex any, blockType string) {
		_, err := db.Exec(`INSERT INTO ai_agent_session_messages
			(agent, session_id, tool_call_index, block_type, tool_name, tool_use_id, action, target, decision, timestamp, content, content_length, truncated, transcript_path)
			VALUES ('claude_code', ?, ?, ?, 'Bash', 'toolu', 'execute', 'ls', 'allowed', 1800000000, '{}', 9000, 1, ?)`,
			sessionID, toolCallIndex, blockType, path)
		require.NoError(t, err)
	}
	for _, s := range []struct{ id, path string }{
		{"fresh", "/u/a/fresh.jsonl"},
		{"cut", "/u/a/cut.jsonl"},
		{"undecided", "/u/a/undecided.jsonl"},
		{"quoted", "/u/o'brien/x' OR '1'='1.jsonl"},
		{"other", "/u/a/other.jsonl"},
	} {
		insert(s.id, s.path, 0, "tool_use")
		insert(s.id, s.path, nil, "text")
		for i := 1; i < 6; i++ {
			insert(s.id, s.path, i, "tool_use")
		}
	}
	// Calls from the limit on aren't asked for.
	insert("fresh", "/u/a/fresh.jsonl", maxEvents-1, "tool_use")
	insert("fresh", "/u/a/fresh.jsonl", maxEvents, "tool_use")

	rows, err := db.Query(query)
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.Equal(t, []string{
		"agent", "session_id", "tool_call_index", "tool_name", "tool_use_id", "action", "target",
		"decision", "timestamp", "content", "content_length", "truncated", "session_ended_at", "requested",
	}, cols)

	got := map[string][]int{}
	endedAts := map[string]int64{}
	var requestedSessions []string
	for rows.Next() {
		var agent, sessionID, toolName, toolUseID, action, target, decision, content string
		var toolCallIndex, requested, truncated int
		var timestamp, sessionEndedAt, contentLength int64
		require.NoError(t, rows.Scan(&agent, &sessionID, &toolCallIndex, &toolName, &toolUseID, &action, &target, &decision, &timestamp, &content, &contentLength, &truncated, &sessionEndedAt, &requested))
		endedAts[sessionID] = sessionEndedAt
		if requested == 1 {
			requestedSessions = append(requestedSessions, agent+"/"+sessionID)
			require.Zero(t, contentLength)
			require.Zero(t, truncated)
			continue
		}
		require.EqualValues(t, 9000, contentLength)
		require.Equal(t, 1, truncated)
		got[sessionID] = append(got[sessionID], toolCallIndex)
	}
	require.NoError(t, rows.Err())

	// Every session asked for is named, whether or not it has tool calls to return. A session is resumed after the
	// calls stored, or from its first undecided call.
	require.Equal(t, []string{"claude_code/fresh", "claude_code/cut", "codex_cli/undecided", "claude_code/quoted"}, requestedSessions)
	all := []int{0, 1, 2, 3, 4, 5}
	require.Equal(t, map[string][]int{
		"fresh":     append(slices.Clone(all), maxEvents-1),
		"cut":       {3, 4, 5},
		"undecided": {2, 3, 4, 5},
		"quoted":    all,
	}, got)
	require.Equal(t, endedAt.Unix(), endedAts["fresh"])
	require.Zero(t, endedAts["undecided"])

	// A session ID is quoted as a path is.
	cursors = []*fleet.HostAISessionEventsCursor{{Agent: "claude_code", AgentSessionID: "x' OR '1'='1", TranscriptPath: "/u/a/x.jsonl"}}
	query, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds, maxEvents, 0)
	require.True(t, ok)
	var requestedID string
	require.NoError(t, db.QueryRow(`SELECT session_id FROM (`+query+`) WHERE requested = 1`).Scan(&requestedID))
	require.Equal(t, "x' OR '1'='1", requestedID)

	// With a limit per refresh, a session catching up is asked for that many calls after those stored. One rereading
	// its undecided calls isn't limited: those are stored already, and a reread cut short would be repeated.
	cursors = []*fleet.HostAISessionEventsCursor{
		{Agent: "claude_code", AgentSessionID: "fresh", TranscriptPath: "/u/a/fresh.jsonl"},
		{Agent: "claude_code", AgentSessionID: "cut", TranscriptPath: "/u/a/cut.jsonl", EventCount: 3},
		{Agent: "codex_cli", AgentSessionID: "undecided", TranscriptPath: "/u/a/undecided.jsonl", EventCount: 6, RereadFromToolCallIndex: new(uint(2))},
	}
	query, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds, maxEvents, 2)
	require.True(t, ok)
	limited, err := db.Query(query)
	require.NoError(t, err)
	defer limited.Close()
	got = map[string][]int{}
	for limited.Next() {
		var sessionID string
		var toolCallIndex, requested int
		var skip any
		require.NoError(t, limited.Scan(&skip, &sessionID, &toolCallIndex, &skip, &skip, &skip, &skip, &skip, &skip, &skip, &skip, &skip, &skip, &requested))
		if requested == 0 {
			got[sessionID] = append(got[sessionID], toolCallIndex)
		}
	}
	require.NoError(t, limited.Err())
	require.Equal(t, map[string][]int{"fresh": {0, 1}, "cut": {3, 4}, "undecided": {2, 3, 4, 5}}, got)
	// The limit per session still applies.
	query, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds, 4, 2)
	require.True(t, ok)
	require.Contains(t, query, "tool_call_index >= 3 AND tool_call_index < 4")

	cursors = cursors[:1]
	// The detail query asks with the configured limit.
	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int, maxEventsArg uint) ([]*fleet.HostAISessionEventsCursor, error) {
		require.EqualValues(t, 7, maxEventsArg)
		return cursors, nil
	}
	query, ok = aiAgentSessionEventsQuery(7, 0).QueryFunc(ctx, logger, host, ds)
	require.True(t, ok)
	require.Contains(t, query, "tool_call_index < 7")
}

func TestDirectIngestAIAgentSessions(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	host := &fleet.Host{ID: 7}

	ds := new(mock.Store)
	var got []*fleet.HostAISession
	ds.UpsertHostAISessionsFunc = func(ctx context.Context, hostID uint, sessions []*fleet.HostAISession) error {
		require.EqualValues(t, 7, hostID)
		got = sessions
		return nil
	}

	require.NoError(t, directIngestAIAgentSessions(ctx, logger, host, ds, nil))
	require.False(t, ds.UpsertHostAISessionsFuncInvoked)

	rows := []map[string]string{
		{
			"agent": "claude_code", "session_id": "s1", "agent_version": "2.1.0", "username": "alice",
			"account_identifier": "alice@example.com", "model": "claude-opus-5-5", "project_path": "/Users/alice/app",
			"git_repository": "github.com/acme/app", "git_branch": "main", "started_at": "1800000000",
			"ended_at": "1800003600", "user_messages": "3", "assistant_messages": "10", "tool_calls": "7",
			"input_tokens": "100", "output_tokens": "200", "cache_read_tokens": "5000000000", "cache_write_tokens": "400",
			"transcript_path": "/Users/alice/.claude/projects/app/s1.jsonl", "transcript_size_bytes": "12345",
		},
		{"agent": "cursor", "session_id": "s2", "started_at": "0", "ended_at": "", "tool_calls": "bogus"},
		// Values the columns can't hold are cut to fit rather than failing the host's sessions: a time past the
		// DATETIME range is unknown, and a count past INT UNSIGNED saturates.
		{
			"agent": "codex_cli", "session_id": "s3", "started_at": "-5", "ended_at": "253402300800",
			"user_messages": "4294967296", "tool_calls": "18446744073709551615", "input_tokens": "18446744073709551615",
		},
		{"agent": "", "session_id": "no-agent"},
		{"agent": "codex_cli", "session_id": ""},
		// The events query names a session by these, and SQL can't hold a NUL.
		{"agent": "claude_code\x00", "session_id": "nul-agent", "transcript_path": "/u/a.jsonl"},
		{"agent": "claude_code", "session_id": "nul\x00id", "transcript_path": "/u/a.jsonl"},
		{"agent": "claude_code", "session_id": "nul-path", "transcript_path": "/u/a\x00.jsonl"},
	}
	require.NoError(t, directIngestAIAgentSessions(ctx, logger, host, ds, rows))
	require.True(t, ds.UpsertHostAISessionsFuncInvoked)
	require.Len(t, got, 3)

	startedAt := time.Unix(1800000000, 0).UTC()
	endedAt := time.Unix(1800003600, 0).UTC()
	require.Equal(t, &fleet.HostAISession{
		Agent:                 "claude_code",
		AgentSessionID:        "s1",
		AgentVersion:          "2.1.0",
		Username:              "alice",
		AccountIdentifier:     "alice@example.com",
		Model:                 "claude-opus-5-5",
		ProjectPath:           "/Users/alice/app",
		GitRepository:         "github.com/acme/app",
		GitBranch:             "main",
		StartedAt:             &startedAt,
		EndedAt:               &endedAt,
		UserMessageCount:      3,
		AssistantMessageCount: 10,
		ToolCallCount:         7,
		InputTokenCount:       100,
		OutputTokenCount:      200,
		CacheReadTokenCount:   5000000000,
		CacheWriteTokenCount:  400,
		TranscriptPath:        "/Users/alice/.claude/projects/app/s1.jsonl",
		TranscriptSize:        12345,
	}, got[0])
	require.Equal(t, &fleet.HostAISession{Agent: "cursor", AgentSessionID: "s2"}, got[1])
	require.Equal(t, &fleet.HostAISession{
		Agent:            "codex_cli",
		AgentSessionID:   "s3",
		UserMessageCount: math.MaxUint32,
		ToolCallCount:    math.MaxUint32,
		InputTokenCount:  math.MaxUint64,
	}, got[2])

	ds.UpsertHostAISessionsFunc = func(ctx context.Context, hostID uint, sessions []*fleet.HostAISession) error {
		return errors.New("boom")
	}
	require.Error(t, directIngestAIAgentSessions(ctx, logger, host, ds, rows))
}

func TestDirectIngestAIAgentSessionEvents(t *testing.T) {
	ctx := t.Context()
	const maxEvents = 40
	logger := slog.New(slog.DiscardHandler)
	host := &fleet.Host{ID: 7}

	ds := new(mock.Store)
	var gotRequested []*fleet.HostAISessionEventsCursor
	var got []*fleet.HostAISessionEvent
	ds.UpsertHostAISessionEventsFunc = func(ctx context.Context, hostID uint, requested []*fleet.HostAISessionEventsCursor, events []*fleet.HostAISessionEvent) error {
		require.EqualValues(t, 7, hostID)
		gotRequested = requested
		got = events
		return nil
	}

	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, nil, maxEvents))
	require.False(t, ds.UpsertHostAISessionEventsFuncInvoked)

	// The rows naming the sessions asked for come back even when they have no tool calls to return.
	markers := []map[string]string{
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "0", "session_ended_at": "1800003600", "requested": "1"},
		{"agent": "cursor", "session_id": "s2", "tool_call_index": "0", "session_ended_at": "0", "requested": "1"},
		{"agent": "", "session_id": "s3", "tool_call_index": "0", "requested": "1"},
	}
	wantRequested := []*fleet.HostAISessionEventsCursor{
		{Agent: "claude_code", AgentSessionID: "s1"},
		{Agent: "cursor", AgentSessionID: "s2"},
	}
	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, markers, maxEvents))
	require.True(t, ds.UpsertHostAISessionEventsFuncInvoked)
	require.Equal(t, wantRequested, gotRequested)
	require.Empty(t, got)

	rows := slices.Concat(markers, []map[string]string{
		{
			"agent": "claude_code", "session_id": "s1", "tool_call_index": "3", "tool_name": "Bash",
			"tool_use_id": "toolu_1", "action": "execute", "target": "ls -la", "decision": "allowed",
			"timestamp": "1800000000", "content": `{"command":"ls -la"}`, "content_length": "20", "truncated": "0",
			"session_ended_at": "1800003600",
		},
		{
			"agent": "cursor", "session_id": "s2", "tool_call_index": "0", "tool_name": "read_file",
			"tool_use_id": "", "action": "read", "target": "main.go", "decision": "", "timestamp": "0",
			"content": "main.go", "content_length": "70000", "truncated": "1", "session_ended_at": "0",
		},
		// A time past the DATETIME range is unknown rather than failing the host's events.
		{
			"agent": "cursor", "session_id": "s2", "tool_call_index": "1", "tool_name": "read_file",
			"action": "something_new", "timestamp": "253402300800", "content": "main.go", "session_ended_at": "253402300800",
		},
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "bogus"},
		{"agent": "claude_code", "session_id": "s1", "tool_name": "Bash"},
		{"agent": "", "session_id": "s1", "tool_call_index": "4"},
		// A call from the limit on is dropped, should a host send one.
		{"agent": "claude_code", "session_id": "s1", "tool_call_index": "40", "tool_name": "Bash"},
	})
	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, rows, maxEvents))
	require.Equal(t, wantRequested, gotRequested)
	require.Len(t, got, 3)

	executedAt := time.Unix(1800000000, 0).UTC()
	endedAt := time.Unix(1800003600, 0).UTC()
	require.JSONEq(t, `{"name":"Bash","id":"toolu_1","input":{"command":"ls -la"}}`, string(got[0].Payload))
	got[0].Payload = nil
	require.Equal(t, &fleet.HostAISessionEvent{
		Agent:          "claude_code",
		AgentSessionID: "s1",
		SessionEndedAt: &endedAt,
		ToolCallIndex:  3,
		Action:         fleet.AISessionActionExecute,
		Target:         "ls -la",
		Decision:       "allowed",
		ExecutedAt:     &executedAt,
		ContentLength:  20,
	}, got[0])

	require.JSONEq(t, `{"name":"read_file","id":"","input":"main.go"}`, string(got[1].Payload))
	require.Equal(t, fleet.AISessionActionRead, got[1].Action)
	require.EqualValues(t, 70000, got[1].ContentLength)
	require.True(t, got[1].Truncated)
	require.Nil(t, got[1].ExecutedAt)
	require.Nil(t, got[1].SessionEndedAt)
	require.Empty(t, got[1].Decision)

	// An action a newer fleetd reports is kept as is.
	require.Equal(t, fleet.AISessionAction("something_new"), got[2].Action)
	require.Nil(t, got[2].ExecutedAt)
	require.Nil(t, got[2].SessionEndedAt)

	ds.UpsertHostAISessionEventsFunc = func(ctx context.Context, hostID uint, requested []*fleet.HostAISessionEventsCursor, events []*fleet.HostAISessionEvent) error {
		return errors.New("boom")
	}
	require.Error(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, rows, maxEvents))
}

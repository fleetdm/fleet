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

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := aiSessionEventPayload("Bash", "toolu_1", c.content)
			require.True(t, json.Valid(got), string(got))
			require.JSONEq(t, c.want, string(got))
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

func TestAIAgentSessionsDetailQueriesPremiumOnly(t *testing.T) {
	free := GetDetailQueries(t.Context(), config.FleetConfig{}, nil, nil, Integrations{}, nil)
	require.NotContains(t, free, "ai_agent_sessions")
	require.NotContains(t, free, "ai_agent_session_events")

	premiumCtx := license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	premium := GetDetailQueries(premiumCtx, config.FleetConfig{}, nil, nil, Integrations{}, nil)
	require.Contains(t, premium, "ai_agent_sessions")
	require.Contains(t, premium, "ai_agent_session_events")
	for _, name := range []string{"ai_agent_sessions", "ai_agent_session_events"} {
		q := premium[name]
		require.NotEmpty(t, q.Discovery)
		require.ElementsMatch(t, append([]string{"darwin", "windows"}, fleet.HostLinuxOSs...), q.Platforms)
		require.NotNil(t, q.DirectIngestFunc)
	}
	require.NotNil(t, premium["ai_agent_session_events"].QueryFunc)
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
		{"events", aiAgentSessionEvents.Discovery, "ai_agent_session_messages", "tool_call_index"},
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

// TestAIAgentSessionsQuery runs the sessions query against sqlite to check the columns the ingestion reads and that
// only sessions that started in the last 30 days, by the host's clock, are reported.
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
	for id, startedAt := range map[string]int64{
		"recent":      hostNow - 3600,
		"at_cutoff":   hostNow - 30*24*3600,
		"too_old":     hostNow - 30*24*3600 - 1,
		"unknown":     0,
		"future_skew": hostNow + 600,
	} {
		_, err = db.Exec(`INSERT INTO ai_agent_sessions (agent, session_id, started_at) VALUES ('claude_code', ?, ?)`, id, startedAt)
		require.NoError(t, err)
	}

	rows, err := db.Query(aiAgentSessions.Query)
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"agent", "session_id", "agent_version", "username", "account_identifier", "model", "project_path",
		"git_repository", "git_branch", "started_at", "ended_at", "user_messages", "assistant_messages", "tool_calls",
		"input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens", "transcript_path",
		"transcript_size_bytes",
	}, cols)

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
	require.ElementsMatch(t, []string{"recent", "at_cutoff", "future_skew"}, ids)
}

func TestBuildAIAgentSessionEventsQuery(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	host := &fleet.Host{ID: 1}
	endedAt := time.Unix(1_800_000_000, 0).UTC()

	ds := new(mock.Store)
	var cursors []*fleet.HostAISessionEventsCursor
	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int) ([]*fleet.HostAISessionEventsCursor, error) {
		require.EqualValues(t, 1, hostID)
		require.Equal(t, aiSessionsPerEventsQuery, limit)
		return cursors, nil
	}

	_, ok := buildAIAgentSessionEventsQuery(ctx, logger, host, ds)
	require.False(t, ok)

	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int) ([]*fleet.HostAISessionEventsCursor, error) {
		return nil, errors.New("boom")
	}
	_, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds)
	require.False(t, ok)

	ds.ListHostAISessionsNeedingEventsFunc = func(ctx context.Context, hostID uint, limit int) ([]*fleet.HostAISessionEventsCursor, error) {
		return cursors, nil
	}
	cursors = []*fleet.HostAISessionEventsCursor{
		// Never read.
		{Agent: "claude_code", AgentSessionID: "fresh", TranscriptPath: "/u/a/fresh.jsonl", EndedAt: &endedAt},
		// The last result was cut after three calls.
		{Agent: "claude_code", AgentSessionID: "cut", TranscriptPath: "/u/a/cut.jsonl", EndedAt: &endedAt, EventCount: 3},
		// A backfill with one call stored, read two calls at a time.
		{Agent: "claude_code", AgentSessionID: "backfill", TranscriptPath: "/u/a/backfill.jsonl", EndedAt: &endedAt, EventCount: 1, MaxToolCalls: 2},
		// Caught up, with an undecided call at 2, and active since.
		{Agent: "codex_cli", AgentSessionID: "undecided", TranscriptPath: "/u/a/undecided.jsonl", EventCount: 6, RereadFromToolCallIndex: new(uint(2))},
		// Paths come from the host.
		{Agent: "claude_code", AgentSessionID: "quoted", TranscriptPath: "/u/o'brien/x' OR '1'='1.jsonl", EndedAt: &endedAt},
		{Agent: "claude_code", AgentSessionID: "nul", TranscriptPath: "/u/a/nul\x00.jsonl"},
	}
	query, ok := buildAIAgentSessionEventsQuery(ctx, logger, host, ds)
	require.True(t, ok)
	require.NotContains(t, query, "\x00")

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
		{"backfill", "/u/a/backfill.jsonl"},
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
	// calls stored, or from its first undecided call, and a backfill reads at most its MaxToolCalls.
	require.Equal(t, []string{"claude_code/fresh", "claude_code/cut", "claude_code/backfill", "codex_cli/undecided", "claude_code/quoted"}, requestedSessions)
	all := []int{0, 1, 2, 3, 4, 5}
	require.Equal(t, map[string][]int{
		"fresh":     all,
		"cut":       {3, 4, 5},
		"backfill":  {1, 2},
		"undecided": {2, 3, 4, 5},
		"quoted":    all,
	}, got)
	require.Equal(t, endedAt.Unix(), endedAts["fresh"])
	require.Zero(t, endedAts["undecided"])

	// A session ID is quoted as a path is.
	cursors = []*fleet.HostAISessionEventsCursor{{Agent: "claude_code", AgentSessionID: "x' OR '1'='1", TranscriptPath: "/u/a/x.jsonl"}}
	query, ok = buildAIAgentSessionEventsQuery(ctx, logger, host, ds)
	require.True(t, ok)
	var requestedID string
	require.NoError(t, db.QueryRow(`SELECT session_id FROM (`+query+`) WHERE requested = 1`).Scan(&requestedID))
	require.Equal(t, "x' OR '1'='1", requestedID)
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

	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, nil))
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
	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, markers))
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
	})
	require.NoError(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, rows))
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
	require.Error(t, directIngestAIAgentSessionEvents(ctx, logger, host, ds, rows))
}

package mysql

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHostAISessions(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpsertSessions", testUpsertHostAISessions},
		{"IncrementalEvents", testHostAISessionEventsIncremental},
		{"Decisions", testHostAISessionEventsDecisions},
		{"DecisionsUnknownEnd", testHostAISessionEventsDecisionsUnknownEnd},
		{"UndecidedBeforeCatchUp", testHostAISessionEventsUndecidedBeforeCatchUp},
		{"Backfill", testHostAISessionEventsBackfill},
		{"EmptyReads", testHostAISessionEventsEmptyReads},
		{"EmptyReadsNewCalls", testHostAISessionEventsEmptyReadsNewCalls},
		{"ResumeAfterBudgetSpent", testHostAISessionEventsResumeAfterBudgetSpent},
		{"NeedingEventsOrderAndScope", testListHostAISessionsNeedingEventsOrderAndScope},
		{"Cleanup", testCleanupHostAISessions},
		{"CleanupBounded", testCleanupHostAISessionsBounded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

type storedAISession struct {
	ID                          uint       `db:"id"`
	AgentSessionID              string     `db:"agent_session_id"`
	AgentVersion                string     `db:"agent_version"`
	Model                       string     `db:"model"`
	StartedAt                   *time.Time `db:"started_at"`
	EndedAt                     *time.Time `db:"ended_at"`
	ToolCallCount               uint       `db:"tool_call_count"`
	CacheWriteTokenCount        uint64     `db:"cache_write_token_count"`
	Actions                     string     `db:"actions"`
	TranscriptPath              string     `db:"transcript_path"`
	Backfill                    bool       `db:"backfill"`
	EventCount                  uint       `db:"event_count"`
	FirstUndecidedToolCallIndex *uint      `db:"first_undecided_tool_call_index"`
	EventsEmptyReads            uint       `db:"events_empty_reads"`
}

type storedAISessionEvent struct {
	ToolCallIndex uint            `db:"tool_call_index"`
	Action        string          `db:"action"`
	Target        string          `db:"target"`
	Decision      string          `db:"decision"`
	ExecutedAt    *time.Time      `db:"executed_at"`
	Payload       json.RawMessage `db:"payload"`
	ContentLength uint            `db:"content_length"`
	Truncated     bool            `db:"truncated"`
}

func listStoredAISessions(t *testing.T, ds *Datastore, hostID uint) []storedAISession {
	var sessions []storedAISession
	require.NoError(t, sqlx.SelectContext(t.Context(), ds.reader(t.Context()), &sessions, `
		SELECT id, agent_session_id, agent_version, model, started_at, ended_at, tool_call_count, cache_write_token_count,
			actions, transcript_path, backfill, event_count, first_undecided_tool_call_index, events_empty_reads
		FROM host_ai_sessions WHERE host_id = ? ORDER BY agent_session_id`, hostID))
	return sessions
}

func listStoredAISessionEvents(t *testing.T, ds *Datastore, hostID uint, agentSessionID string) []storedAISessionEvent {
	var events []storedAISessionEvent
	require.NoError(t, sqlx.SelectContext(t.Context(), ds.reader(t.Context()), &events, `
		SELECT e.tool_call_index, e.action, e.target, e.decision, e.executed_at, e.payload, e.content_length, e.truncated
		FROM host_ai_session_events e JOIN host_ai_sessions s ON s.id = e.ai_session_id
		WHERE s.host_id = ? AND s.agent_session_id = ?
		ORDER BY e.tool_call_index`, hostID, agentSessionID))
	return events
}

func newTestAISession(id string, endedAt time.Time, toolCalls uint) *fleet.HostAISession {
	startedAt := endedAt.Add(-time.Hour)
	return &fleet.HostAISession{
		Agent:          "claude_code",
		AgentSessionID: id,
		AgentVersion:   "2.1.0",
		Username:       "alice",
		Model:          "claude-opus-5-5",
		ProjectPath:    "/Users/alice/src/app",
		StartedAt:      &startedAt,
		EndedAt:        &endedAt,
		ToolCallCount:  toolCalls,
		TranscriptPath: "/Users/alice/.claude/projects/app/" + id + ".jsonl",
	}
}

func newTestAISessionEvent(s *fleet.HostAISession, toolCallIndex uint, decision string) *fleet.HostAISessionEvent {
	return &fleet.HostAISessionEvent{
		Agent:          s.Agent,
		AgentSessionID: s.AgentSessionID,
		SessionEndedAt: s.EndedAt,
		ToolCallIndex:  toolCallIndex,
		Action:         fleet.AISessionActionExecute,
		Target:         "ls",
		Decision:       decision,
		ExecutedAt:     s.EndedAt,
		Payload:        json.RawMessage(`{"name":"Bash","id":"toolu_1","input":{"command":"ls"}}`),
		ContentLength:  17,
	}
}

func storeTestAISessions(t *testing.T, ds *Datastore, hostID uint, sessions ...*fleet.HostAISession) {
	require.NoError(t, ds.UpsertHostAISessions(t.Context(), hostID, sessions))
}

func testUpsertHostAISessions(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	s1 := newTestAISession("s1", now, 3)
	s2 := newTestAISession("s2", now, 0)
	s2.StartedAt = nil
	s2.EndedAt = nil
	s2.Model = strings.Repeat("m", 300)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s1, s2}))

	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Equal(t, "s1", stored[0].AgentSessionID)
	require.NotNil(t, stored[0].EndedAt)
	require.True(t, now.Equal(*stored[0].EndedAt))
	require.EqualValues(t, 3, stored[0].ToolCallCount)
	require.Nil(t, stored[1].StartedAt)
	require.Nil(t, stored[1].EndedAt)
	require.Len(t, stored[1].Model, 255)
	require.False(t, stored[0].Backfill)

	// s1 is updated, s3 is new, and s2, missing from the host's result, is kept.
	later := now.Add(time.Minute)
	s1 = newTestAISession("s1", later, 5)
	s1.AgentVersion = "2.2.0"
	s1.CacheWriteTokenCount = 1 << 40
	s3 := newTestAISession("s3", later, 1)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s1, s3}))

	stored = listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 3)
	require.Equal(t, []string{"s1", "s2", "s3"}, []string{stored[0].AgentSessionID, stored[1].AgentSessionID, stored[2].AgentSessionID})
	require.Equal(t, "2.2.0", stored[0].AgentVersion)
	require.EqualValues(t, 5, stored[0].ToolCallCount)
	require.Equal(t, uint64(1<<40), stored[0].CacheWriteTokenCount)
	require.True(t, later.Equal(*stored[0].EndedAt))
	require.False(t, stored[2].Backfill)

	// The same session ID from another agent is another session.
	s1Codex := newTestAISession("s1", later, 1)
	s1Codex.Agent = "codex_cli"
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s1Codex}))
	require.Len(t, listStoredAISessions(t, ds, host.ID), 4)

	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, nil))
}

func testHostAISessionEventsIncremental(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	s := newTestAISession("s1", now, 4)
	storeTestAISessions(t, ds, host.ID, s)

	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "claude_code", cursors[0].Agent)
	require.Equal(t, "s1", cursors[0].AgentSessionID)
	require.Equal(t, s.TranscriptPath, cursors[0].TranscriptPath)
	require.Zero(t, cursors[0].MaxToolCalls)
	require.Zero(t, cursors[0].EventCount)
	require.Nil(t, cursors[0].RereadFromToolCallIndex)

	// The first result is cut after two calls.
	read := newTestAISessionEvent(s, 1, "allowed")
	read.Action = fleet.AISessionActionRead
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 0, "allowed"),
		read,
	}))

	// The stored events are the first event_count calls from where collection started, so the next read resumes there.
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.EqualValues(t, 2, cursors[0].EventCount)
	require.Nil(t, cursors[0].RereadFromToolCallIndex)

	// An event already stored doesn't duplicate. An action Fleet doesn't know is kept on the event, as fleetd reported
	// it, but left out of the session's actions.
	unknownAction := newTestAISessionEvent(s, 2, "allowed")
	unknownAction.Action = "browse"
	unknownAction.ContentLength = 20000
	unknownAction.Truncated = true
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		read,
		unknownAction,
		newTestAISessionEvent(s, 3, "denied"),
	}))

	events := listStoredAISessionEvents(t, ds, host.ID, "s1")
	require.Len(t, events, 4)
	require.Equal(t, []uint{0, 1, 2, 3}, []uint{events[0].ToolCallIndex, events[1].ToolCallIndex, events[2].ToolCallIndex, events[3].ToolCallIndex})
	require.Equal(t, "execute", events[0].Action)
	require.Equal(t, "read", events[1].Action)
	require.Equal(t, "browse", events[2].Action)
	require.Equal(t, "denied", events[3].Decision)
	require.JSONEq(t, `{"name":"Bash","id":"toolu_1","input":{"command":"ls"}}`, string(events[0].Payload))
	require.EqualValues(t, 17, events[0].ContentLength)
	require.False(t, events[0].Truncated)
	require.EqualValues(t, 20000, events[2].ContentLength)
	require.True(t, events[2].Truncated)

	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 1)
	require.EqualValues(t, 4, stored[0].EventCount)
	require.Equal(t, "read,execute", stored[0].Actions)

	// Complete and decided: nothing left to read.
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// New tool calls on the host make it incomplete again, resuming after the stored events.
	s = newTestAISession("s1", now.Add(time.Minute), 5)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.EqualValues(t, 4, cursors[0].EventCount)

	// Long targets are cut to the column width.
	long := newTestAISessionEvent(s, 4, "allowed")
	long.Target = strings.Repeat("t", 2000)
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{long}))
	events = listStoredAISessionEvents(t, ds, host.ID, "s1")
	require.Len(t, events, 5)
	require.Len(t, events[4].Target, 256)

	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, nil))
}

func testHostAISessionEventsDecisions(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	s := newTestAISession("s1", now, 3)
	cursor := newTestAISession("c1", now, 2)
	cursor.Agent = fleet.AISessionAgentCursor
	storeTestAISessions(t, ds, host.ID, s, cursor)

	// Call 1 was still running when read.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 0, "allowed"),
		newTestAISessionEvent(s, 1, ""),
		newTestAISessionEvent(s, 2, "error"),
		newTestAISessionEvent(cursor, 0, ""),
		newTestAISessionEvent(cursor, 1, ""),
	}))
	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Equal(t, "c1", stored[0].AgentSessionID)
	require.NotNil(t, stored[0].FirstUndecidedToolCallIndex)
	require.EqualValues(t, 0, *stored[0].FirstUndecidedToolCallIndex)
	require.NotNil(t, stored[1].FirstUndecidedToolCallIndex)
	require.EqualValues(t, 1, *stored[1].FirstUndecidedToolCallIndex)

	// No activity since the events were read: no decision can have been recorded.
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// Both sessions are active again with no new tool calls; only s1 is reread, from its undecided call.
	s = newTestAISession("s1", now.Add(time.Minute), 3)
	cursor = newTestAISession("c1", now.Add(time.Minute), 2)
	cursor.Agent = fleet.AISessionAgentCursor
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s, cursor}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "s1", cursors[0].AgentSessionID)
	require.NotNil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 1, *cursors[0].RereadFromToolCallIndex)

	// Rereading records the decision, keeping the rest of the event as first stored.
	decided := newTestAISessionEvent(s, 1, "allowed")
	decided.Target = "changed"
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		decided,
		newTestAISessionEvent(s, 2, "error"),
	}))
	events := listStoredAISessionEvents(t, ds, host.ID, "s1")
	require.Len(t, events, 3)
	require.Equal(t, "allowed", events[1].Decision)
	require.Equal(t, "ls", events[1].Target)
	stored = listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Nil(t, stored[1].FirstUndecidedToolCallIndex)

	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// A decision is never cleared by a later read that has none.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{newTestAISessionEvent(s, 1, "")}))
	events = listStoredAISessionEvents(t, ds, host.ID, "s1")
	require.Len(t, events, 3)
	require.Equal(t, "allowed", events[1].Decision)

	// A session with new tool calls and an undecided call resumes after its stored events: catching up comes before
	// rereading, which would otherwise re-send the same events every refresh of an active session.
	s = newTestAISession("s1", now.Add(2*time.Minute), 5)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{newTestAISessionEvent(s, 3, "")}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Nil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 4, cursors[0].EventCount)

	// Once caught up, the undecided call is reread on the next activity.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{newTestAISessionEvent(s, 4, "allowed")}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)
	s = newTestAISession("s1", now.Add(3*time.Minute), 5)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.NotNil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 3, *cursors[0].RereadFromToolCallIndex)

	// A call that starts a sub-agent gets its result after the sub-agent's calls, so an undecided call stays eligible
	// however many calls follow it within the hour.
	var later []*fleet.HostAISessionEvent
	for i := uint(5); i < 75; i++ {
		later = append(later, newTestAISessionEvent(s, i, "allowed"))
	}
	s = newTestAISession("s1", now.Add(4*time.Minute), 75)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, later))
	stored = listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.NotNil(t, stored[1].FirstUndecidedToolCallIndex)
	require.EqualValues(t, 3, *stored[1].FirstUndecidedToolCallIndex)

	// One still undecided over an hour before the session's end is final and isn't reread.
	s = newTestAISession("s1", now.Add(2*time.Hour), 76)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{newTestAISessionEvent(s, 75, "allowed")}))
	stored = listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Nil(t, stored[1].FirstUndecidedToolCallIndex)
	s = newTestAISession("s1", now.Add(3*time.Hour), 76)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)
}

func testHostAISessionEventsDecisionsUnknownEnd(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	// The session's end was unknown when its events were read.
	s := newTestAISession("s1", now, 1)
	s.EndedAt = nil
	storeTestAISessions(t, ds, host.ID, s)
	undecided := newTestAISessionEvent(s, 0, "")
	undecided.ExecutedAt = &now
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{undecided}))
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// Once it is known, the undecided call is reread.
	s = newTestAISession("s1", now, 1)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.NotNil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 0, *cursors[0].RereadFromToolCallIndex)
}

func testHostAISessionEventsUndecidedBeforeCatchUp(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	// Call 2 was still running when read; its sub-agent calls 3 and 4 were.
	s := newTestAISession("s1", now, 5)
	storeTestAISessions(t, ds, host.ID, s)
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 0, "allowed"),
		newTestAISessionEvent(s, 1, "allowed"),
		newTestAISessionEvent(s, 2, ""),
		newTestAISessionEvent(s, 3, "allowed"),
		newTestAISessionEvent(s, 4, "allowed"),
	}))
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// One write on the host records call 2's result along with calls 5 to 8. Catching up comes first.
	s = newTestAISession("s1", now.Add(time.Minute), 9)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Nil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 5, cursors[0].EventCount)
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 5, "allowed"),
		newTestAISessionEvent(s, 6, "allowed"),
		newTestAISessionEvent(s, 7, "allowed"),
		newTestAISessionEvent(s, 8, "allowed"),
	}))

	// The read that caught up started after the undecided call, so the write it read may hold its decision: once caught
	// up, the session is reread from that call even though it hasn't been active since.
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.NotNil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 2, *cursors[0].RereadFromToolCallIndex)

	// A reread that finds it still undecided isn't repeated until the session is active again.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 2, ""),
		newTestAISessionEvent(s, 3, "allowed"),
	}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)
	s = newTestAISession("s1", now.Add(2*time.Minute), 9)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.NotNil(t, cursors[0].RereadFromToolCallIndex)
	require.EqualValues(t, 2, *cursors[0].RereadFromToolCallIndex)
}

func testHostAISessionEventsBackfill(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	// Sessions that started well before Fleet first stored them, as at a rollout, are backfills: their calls are read
	// one session and a few calls at a time, so every host's history isn't written at once. Recent ones are read as
	// usual.
	old1 := newTestAISession("old1", now.Add(-time.Hour), 250)
	old1.StartedAt = new(now.Add(-72 * time.Hour))
	old2 := newTestAISession("old2", now.Add(-2*time.Hour), 10)
	old2.StartedAt = new(now.Add(-48 * time.Hour))
	// One whose start isn't known may have been running for long, so it's a backfill too. One with no calls yet has
	// nothing to backfill.
	unknown := newTestAISession("unknown", now.Add(-3*time.Hour), 5)
	unknown.StartedAt = nil
	idle := newTestAISession("idle", now.Add(-3*time.Hour), 0)
	idle.StartedAt = new(now.Add(-72 * time.Hour))
	live := newTestAISession("live", now, 4)
	storeTestAISessions(t, ds, host.ID, old1, old2, unknown, idle, live)
	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 5)
	backfill := map[string]bool{}
	for _, s := range stored {
		backfill[s.AgentSessionID] = s.Backfill
	}
	require.Equal(t, map[string]bool{"old1": true, "old2": true, "unknown": true, "idle": false, "live": false}, backfill)

	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 2)
	require.Equal(t, "live", cursors[0].AgentSessionID)
	require.Zero(t, cursors[0].MaxToolCalls)
	require.Equal(t, "old1", cursors[1].AgentSessionID)
	require.EqualValues(t, hostAISessionBackfillCalls, cursors[1].MaxToolCalls)

	events := []*fleet.HostAISessionEvent{}
	for i := range uint(4) {
		events = append(events, newTestAISessionEvent(live, i, "allowed"))
	}
	for i := range uint(hostAISessionBackfillCalls) {
		events = append(events, newTestAISessionEvent(old1, i, "allowed"))
	}
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, events))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "old1", cursors[0].AgentSessionID)
	require.EqualValues(t, hostAISessionBackfillCalls, cursors[0].EventCount)

	// Once caught up, a backfill is read as usual, and the next one starts.
	events = nil
	for i := uint(hostAISessionBackfillCalls); i < 250; i++ {
		events = append(events, newTestAISessionEvent(old1, i, "allowed"))
	}
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, events))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "old2", cursors[0].AgentSessionID)
	require.EqualValues(t, hostAISessionBackfillCalls, cursors[0].MaxToolCalls)

	old1 = newTestAISession("old1", now, 260)
	old1.StartedAt = new(now.Add(-72 * time.Hour))
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{old1}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 2)
	require.Equal(t, "old1", cursors[0].AgentSessionID)
	require.Zero(t, cursors[0].MaxToolCalls)
	require.EqualValues(t, 250, cursors[0].EventCount)
	require.Equal(t, "old2", cursors[1].AgentSessionID)
}

func testHostAISessionEventsEmptyReads(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	gone := newTestAISession("gone", now, 2)
	other := newTestAISession("other", now.Add(-30*time.Minute), 1)
	storeTestAISessions(t, ds, host.ID, gone, other)

	// Requested sessions that return no events are given up on after a few reads, so one whose transcript is gone from
	// the host doesn't keep its place ahead of the others.
	for i := 1; i < hostAISessionMaxEmptyEventReads; i++ {
		cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 1)
		require.NoError(t, err)
		require.Len(t, cursors, 1)
		require.Equal(t, "gone", cursors[0].AgentSessionID)
		require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, nil))
		stored := listStoredAISessions(t, ds, host.ID)
		require.Len(t, stored, 2)
		require.EqualValues(t, i, stored[0].EventsEmptyReads)
		require.Zero(t, stored[1].EventsEmptyReads)
	}
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 1)
	require.NoError(t, err)
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, nil))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "other", cursors[0].AgentSessionID)

	// Reported again unchanged, it stays given up on; activity on the host makes it worth reading again.
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{gone}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "other", cursors[0].AgentSessionID)
	gone = newTestAISession("gone", now.Add(time.Minute), 2)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{gone}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 2)
	require.Equal(t, "gone", cursors[0].AgentSessionID)

	// Events arriving for a requested session clear its empty reads; the other requested session counts one.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, nil))
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, []*fleet.HostAISessionEvent{newTestAISessionEvent(gone, 0, "allowed")}))
	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Zero(t, stored[0].EventsEmptyReads)
	require.EqualValues(t, 2, stored[1].EventsEmptyReads)
}

func testHostAISessionEventsEmptyReadsNewCalls(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	s := newTestAISession("s1", now, 2)
	storeTestAISessions(t, ds, host.ID, s)
	for range hostAISessionMaxEmptyEventReads {
		cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
		require.NoError(t, err)
		require.Len(t, cursors, 1)
		require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, nil))
	}
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// New calls make it worth reading again even when its end time didn't move, as with Cursor, whose end time comes
	// from its own metadata.
	s = newTestAISession("s1", now, 3)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{s}))
	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "s1", cursors[0].AgentSessionID)
}

func testHostAISessionEventsResumeAfterBudgetSpent(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)

	// A session read behind newer ones that spend fleetd's result budget gets one row per read: the first call it
	// lacks, since Fleet resumes after its stored events. Each such read makes progress and isn't an empty read.
	newer := newTestAISession("newer", now.Add(time.Minute), 100)
	s := newTestAISession("s1", now, 5)
	storeTestAISessions(t, ds, host.ID, newer, s)
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(s, 0, "allowed"),
		newTestAISessionEvent(s, 1, "allowed"),
		newTestAISessionEvent(s, 2, "allowed"),
	}))

	for want := uint(3); want < 5; want++ {
		cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
		require.NoError(t, err)
		require.Len(t, cursors, 2)
		require.Equal(t, "newer", cursors[0].AgentSessionID)
		require.Equal(t, "s1", cursors[1].AgentSessionID)
		require.Equal(t, want, cursors[1].EventCount)

		require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, cursors, []*fleet.HostAISessionEvent{
			newTestAISessionEvent(newer, cursors[0].EventCount, "allowed"),
			newTestAISessionEvent(s, want, "allowed"),
		}))
		stored := listStoredAISessions(t, ds, host.ID)
		require.Len(t, stored, 2)
		require.Equal(t, "s1", stored[1].AgentSessionID)
		require.Equal(t, want+1, stored[1].EventCount)
		require.Zero(t, stored[1].EventsEmptyReads)
	}
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Len(t, cursors, 1)
	require.Equal(t, "newer", cursors[0].AgentSessionID)
	require.Len(t, listStoredAISessionEvents(t, ds, host.ID, "s1"), 5)
}

func testListHostAISessionsNeedingEventsOrderAndScope(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host1 := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	host2 := newTestHostWithPlatform(t, ds, "host2", "linux", nil)
	now := time.Now().UTC().Truncate(time.Second)

	noToolCalls := newTestAISession("none", now, 0)
	noPath := newTestAISession("nopath", now, 2)
	noPath.TranscriptPath = ""
	storeTestAISessions(t, ds, host1.ID,
		newTestAISession("oldest", now.Add(-50*time.Minute), 1),
		newTestAISession("latest", now, 1),
		newTestAISession("middle", now.Add(-20*time.Minute), 1),
		noToolCalls,
		noPath,
	)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host2.ID, []*fleet.HostAISession{newTestAISession("other", now.Add(time.Hour), 1)}))

	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host1.ID, 2)
	require.NoError(t, err)
	require.Len(t, cursors, 2)
	require.Equal(t, "latest", cursors[0].AgentSessionID)
	require.Equal(t, "middle", cursors[1].AgentSessionID)

	cursors, err = ds.ListHostAISessionsNeedingEvents(ctx, host1.ID, 10)
	require.NoError(t, err)
	require.Len(t, cursors, 3)
	require.Equal(t, "oldest", cursors[2].AgentSessionID)

	// Events name their session by agent and ID, which only identify it on its own host.
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host1.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(newTestAISession("other", now, 1), 0, "allowed"),
	}))
	require.Empty(t, listStoredAISessionEvents(t, ds, host2.ID, "other"))
}

func testCleanupHostAISessions(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)
	cutoff := now.Add(-fleet.HostAISessionsMaxAge)
	future := now.Add(365 * 24 * time.Hour)

	// Sessions are kept for as long after they started, with a grace for host clocks a little behind Fleet's.
	expired := newTestAISession("expired", cutoff.Add(-hostAISessionsCleanupGrace), 1)
	grace := newTestAISession("grace", cutoff, 1)
	kept := newTestAISession("kept", now, 1)
	sessions := []*fleet.HostAISession{expired, grace, kept}
	for _, id := range []string{"no-start-stale", "no-start-changed"} {
		s := newTestAISession(id, now, 1)
		s.StartedAt = nil
		sessions = append(sessions, s)
	}
	for _, id := range []string{"future-stale", "future-changed"} {
		sessions = append(sessions, newTestAISession(id, future, 1))
	}
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, sessions))
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, []*fleet.HostAISessionEvent{
		newTestAISessionEvent(expired, 0, "allowed"),
		newTestAISessionEvent(kept, 0, "allowed"),
	}))
	// Sessions the host's clock can't place, with no known start or one in the future, are kept for as long after
	// Fleet last saw them change.
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE host_ai_sessions SET created_at = ?, updated_at = ?
		WHERE agent_session_id IN ('no-start-stale', 'future-stale')`, cutoff.Add(-time.Hour), cutoff.Add(-time.Minute))
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_ai_sessions SET created_at = ?, updated_at = ?
		WHERE agent_session_id IN ('no-start-changed', 'future-changed')`, cutoff.Add(-time.Hour), cutoff.Add(time.Minute))
	require.NoError(t, err)

	deleted, err := ds.CleanupHostAISessions(ctx, cutoff)
	require.NoError(t, err)
	require.EqualValues(t, 3, deleted)

	stored := listStoredAISessions(t, ds, host.ID)
	var ids []string
	for _, s := range stored {
		ids = append(ids, s.AgentSessionID)
	}
	require.Equal(t, []string{"future-changed", "grace", "kept", "no-start-changed"}, ids)
	require.Len(t, listStoredAISessionEvents(t, ds, host.ID, "kept"), 1)

	var orphanEvents int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &orphanEvents,
		`SELECT COUNT(*) FROM host_ai_session_events WHERE ai_session_id NOT IN (SELECT id FROM host_ai_sessions)`))
	require.Zero(t, orphanEvents)

	deleted, err = ds.CleanupHostAISessions(ctx, cutoff)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func testCleanupHostAISessionsBounded(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)
	now := time.Now().UTC().Truncate(time.Second)
	cutoff := now.Add(-fleet.HostAISessionsMaxAge)

	var sessions []*fleet.HostAISession
	for _, id := range []string{"e1", "e2", "e3"} {
		sessions = append(sessions, newTestAISession(id, cutoff.Add(-hostAISessionsCleanupGrace), 3))
	}
	kept := newTestAISession("kept", now, 3)
	storeTestAISessions(t, ds, host.ID, append(sessions, kept)...)
	var events []*fleet.HostAISessionEvent
	for _, s := range append(sessions, kept) {
		for i := range uint(3) {
			events = append(events, newTestAISessionEvent(s, i, "allowed"))
		}
	}
	require.NoError(t, ds.UpsertHostAISessionEvents(ctx, host.ID, nil, events))

	// A budget of four rows covers e1's three calls and its row, so the run stops after it: a session is deleted
	// whole, with its events, or not at all.
	defer func(batchSize, maxBatches int) {
		hostAISessionsCleanupBatchSize, hostAISessionsCleanupMaxBatches = batchSize, maxBatches
	}(hostAISessionsCleanupBatchSize, hostAISessionsCleanupMaxBatches)
	hostAISessionsCleanupBatchSize, hostAISessionsCleanupMaxBatches = 2, 2
	deleted, err := ds.CleanupHostAISessions(ctx, cutoff)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	stored := listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 3)
	require.Equal(t, []string{"e2", "e3", "kept"}, []string{stored[0].AgentSessionID, stored[1].AgentSessionID, stored[2].AgentSessionID})
	for _, s := range stored {
		require.EqualValues(t, 3, s.EventCount)
		require.Len(t, listStoredAISessionEvents(t, ds, host.ID, s.AgentSessionID), 3)
	}

	// Reported again before the next run, e2 has nothing left to read.
	e2 := newTestAISession("e2", now, 3)
	require.NoError(t, ds.UpsertHostAISessions(ctx, host.ID, []*fleet.HostAISession{e2}))
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, 5)
	require.NoError(t, err)
	require.Empty(t, cursors)

	// The next run deletes what is still expired.
	hostAISessionsCleanupMaxBatches = 200
	deleted, err = ds.CleanupHostAISessions(ctx, cutoff)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	stored = listStoredAISessions(t, ds, host.ID)
	require.Len(t, stored, 2)
	require.Equal(t, "e2", stored[0].AgentSessionID)
	require.Equal(t, "kept", stored[1].AgentSessionID)
	require.Len(t, listStoredAISessionEvents(t, ds, host.ID, "kept"), 3)
}

package main

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/stretchr/testify/require"
)

// buildAISessionEventsQuery builds the events query the way Fleet does for a host whose stored sessions are cursors.
func buildAISessionEventsQuery(t *testing.T, cursors []*fleet.HostAISessionEventsCursor) string {
	ds := new(mock.Store)
	ds.ListHostAISessionsNeedingEventsFunc = func(_ context.Context, _ uint, _ int) ([]*fleet.HostAISessionEventsCursor, error) {
		return cursors, nil
	}
	ctx := license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	q, ok := osquery_utils.GetDetailQueries(ctx, config.FleetConfig{}, nil, nil, osquery_utils.Integrations{}, nil)["ai_agent_session_events"]
	require.True(t, ok)
	query, ok := q.QueryFunc(ctx, slog.New(slog.DiscardHandler), &fleet.Host{ID: 1}, ds)
	require.True(t, ok)
	return query
}

func testAISessionsSim(t *testing.T, cfg aiSessionsConfig, now time.Time) *aiSessionsSim {
	return newAISessionsSim(cfg, "macos", 1, now)
}

func aiSessionByAgent(t *testing.T, s *aiSessionsSim, agent string) *aiSession {
	for _, sess := range s.sessions {
		if sess.agentName() == agent {
			return sess
		}
	}
	t.Fatalf("no %s session", agent)
	return nil
}

func cursorFor(s *aiSession, now time.Time) *fleet.HostAISessionEventsCursor {
	return &fleet.HostAISessionEventsCursor{
		Agent:          s.agentName(),
		AgentSessionID: s.id(),
		TranscriptPath: s.transcriptPath(),
		EndedAt:        new(s.endedAt(now)),
	}
}

// aiEventIndexes are the tool_call_index values of the session's event rows.
func aiEventIndexes(t *testing.T, rows []map[string]string, sessionID string) []uint {
	var idx []uint
	for _, r := range rows {
		if r["requested"] != "0" || r["session_id"] != sessionID {
			continue
		}
		i, err := strconv.ParseUint(r["tool_call_index"], 10, 32)
		require.NoError(t, err)
		idx = append(idx, uint(i))
	}
	return idx
}

// seqUint is from, from+1, ..., to-1.
func seqUint(from, to uint) []uint {
	var s []uint
	for i := from; i < to; i++ {
		s = append(s, i)
	}
	return s
}

func TestAISessionsConfigValidate(t *testing.T) {
	require.NoError(t, aiSessionsConfig{prob: 0.5, initialCount: 100, perDay: 10}.validate())
	require.Error(t, aiSessionsConfig{prob: 1.5}.validate())
	require.Error(t, aiSessionsConfig{longProb: -0.1}.validate())
	require.Error(t, aiSessionsConfig{initialCount: -1}.validate())
	// fleetd reads at most 5,000 sessions per agent, so a host must stay under it over the 30 days it reports.
	require.Error(t, aiSessionsConfig{initialCount: 2000, perDay: 101}.validate())
}

func TestAISessionsSessionsRows(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s := testAISessionsSim(t, aiSessionsConfig{initialCount: 50, perDay: 24}, now)

	rows := s.sessionsRows(now)
	require.Len(t, rows, 50)
	agents := make(map[string]struct{})
	for _, r := range rows {
		// The rows have the shape of the ai_agent_sessions detail query.
		require.ElementsMatch(t, []string{
			"agent", "session_id", "agent_version", "username", "account_identifier", "model", "project_path",
			"git_repository", "git_branch", "started_at", "ended_at", "user_messages", "assistant_messages", "tool_calls",
			"input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens", "transcript_path",
			"transcript_size_bytes",
		}, slices.Collect(maps.Keys(r)))
		require.Contains(t, fleet.AISessionAgents, r["agent"])
		agents[r["agent"]] = struct{}{}
		ended, err := strconv.ParseInt(r["ended_at"], 10, 64)
		require.NoError(t, err)
		require.Greater(t, ended, now.Add(-30*24*time.Hour).Unix())
		require.LessOrEqual(t, ended, now.Unix())
	}
	require.Greater(t, len(agents), 1)

	// New sessions arrive at the configured rate.
	require.Len(t, s.sessionsRows(now.Add(10*time.Hour)), 60)

	// Sessions that ended over 30 days ago are no longer reported.
	later := now.Add(31 * 24 * time.Hour)
	for _, r := range s.sessionsRows(later) {
		ended, err := strconv.ParseInt(r["ended_at"], 10, 64)
		require.NoError(t, err)
		require.Greater(t, ended, later.Add(-30*24*time.Hour).Unix())
	}
}

func TestAISessionsEventsQuery(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s := testAISessionsSim(t, aiSessionsConfig{}, now)
	claude := s.addSession(0, now.Add(-2*time.Hour), 30*time.Second, 40, false)
	cursor := s.addSession(2, now.Add(-3*time.Hour), 30*time.Second, 40, false)
	codex := s.addSession(3, now.Add(-4*time.Hour), 30*time.Second, 40, false)
	claude.repo = 4 // a project path with a quote in it
	require.Contains(t, claude.transcriptPath(), "'")

	behind := cursorFor(cursor, now)
	behind.EventCount = 12
	reread := cursorFor(codex, now)
	reread.EventCount = 40
	reread.RereadFromToolCallIndex = new(uint(31))
	gone := &fleet.HostAISessionEventsCursor{
		Agent: "claude_code", AgentSessionID: "deleted", TranscriptPath: "/Users/fleet/.claude/projects/gone.jsonl",
		EndedAt: new(now.Add(-time.Hour)),
	}
	query := buildAISessionEventsQuery(t, []*fleet.HostAISessionEventsCursor{cursorFor(claude, now), behind, reread, gone})

	rows := s.eventsRows(query, now)
	requested := make(map[string]map[string]string)
	for _, r := range rows {
		// The rows have the shape of the ai_agent_session_events detail query.
		require.ElementsMatch(t, []string{
			"agent", "session_id", "tool_call_index", "tool_name", "tool_use_id", "action", "target",
			"decision", "timestamp", "content", "session_ended_at", "requested",
		}, slices.Collect(maps.Keys(r)))
		if r["requested"] == "1" {
			requested[r["session_id"]] = r
		}
	}
	require.Len(t, requested, 4)
	require.Equal(t, "cursor", requested[cursor.id()]["agent"])
	require.Equal(t, strconv.FormatInt(cursor.endedAt(now).Unix(), 10), requested[cursor.id()]["session_ended_at"])

	// A fresh session returns all its tool calls.
	require.Len(t, aiEventIndexes(t, rows, claude.id()), 40)
	// A session behind resumes after the calls it has stored.
	require.Equal(t, seqUint(12, 40), aiEventIndexes(t, rows, cursor.id()))
	// A reread starts at the first undecided call.
	require.Equal(t, seqUint(31, 40), aiEventIndexes(t, rows, codex.id()))
	// A transcript gone from the host returns only its requested row.
	require.Empty(t, aiEventIndexes(t, rows, "deleted"))

	for _, r := range rows {
		if r["requested"] != "0" {
			continue
		}
		switch r["session_id"] {
		case cursor.id():
			// Cursor records no tool call ids or outcomes, and its times are to the minute.
			require.Empty(t, r["tool_use_id"])
			require.Empty(t, r["decision"])
			ts, err := strconv.ParseInt(r["timestamp"], 10, 64)
			require.NoError(t, err)
			require.Zero(t, ts%60)
		case claude.id():
			require.NotEmpty(t, r["tool_use_id"])
			require.Contains(t, []string{"allowed", "denied", "error"}, r["decision"])
		}
		require.Contains(t, []string{"read", "write", "execute", "search", "network", "mcp", "other"}, r["action"])
		require.LessOrEqual(t, len(r["content"]), aiSessionMaxContent)
		require.LessOrEqual(t, len(r["target"]), 256)
	}
}

func TestAISessionsEventsPaging(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s := testAISessionsSim(t, aiSessionsConfig{}, now)
	var long []*aiSession
	for range 3 {
		long = append(long, s.addSession(0, now.Add(-20*time.Hour), 10*time.Second, 3000, true))
	}

	// Read the sessions the way Fleet does: resume each after the last call it stored, and give up on one after three
	// reads that return nothing.
	collected := make(map[string][]uint)
	emptyReads := make(map[string]int)
	refreshes := 0
	for ; ; refreshes++ {
		require.Less(t, refreshes, 100)
		var cursors []*fleet.HostAISessionEventsCursor
		for _, sess := range long {
			if len(collected[sess.id()]) >= 3000 || emptyReads[sess.id()] >= 3 {
				continue
			}
			c := cursorFor(sess, now)
			c.EventCount = uint(len(collected[sess.id()]))
			cursors = append(cursors, c)
		}
		if len(cursors) == 0 {
			break
		}
		rows := s.eventsRows(buildAISessionEventsQuery(t, cursors), now)
		// fleetd caps a result at about 1 MiB, letting each table call past it with its first row up to twice that.
		require.LessOrEqual(t, s.lastEventsBytes, 2*aiSessionMaxResultBytes)
		for _, c := range cursors {
			pos := aiEventIndexes(t, rows, c.AgentSessionID)
			if len(pos) == 0 {
				emptyReads[c.AgentSessionID]++
			}
			collected[c.AgentSessionID] = append(collected[c.AgentSessionID], pos...)
		}
	}
	require.Greater(t, refreshes, 2, "long sessions need several refreshes")
	for _, sess := range long {
		require.Equal(t, seqUint(0, 3000), collected[sess.id()])
	}
}

func TestAISessionsActive(t *testing.T) {
	start := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s := testAISessionsSim(t, aiSessionsConfig{}, start)
	sess := s.addSession(0, start, 30*time.Second, 100, false)

	undecided := func(now time.Time) []string {
		var idx []string
		rows := s.eventsRows(buildAISessionEventsQuery(t, []*fleet.HostAISessionEventsCursor{cursorFor(sess, now)}), now)
		for _, r := range rows {
			if r["requested"] == "0" && r["decision"] == "" {
				idx = append(idx, r["tool_call_index"])
			}
		}
		return idx
	}

	// While the agent is working, its session grows and the calls of its latest message have no outcome yet.
	t1 := start.Add(10 * time.Minute)
	require.Equal(t, 20, sess.callsAt(t1))
	require.Equal(t, start.Add(20*30*time.Second), sess.endedAt(t1))
	first := undecided(t1)
	require.NotEmpty(t, first)
	require.Len(t, s.eventsRows(buildAISessionEventsQuery(t, []*fleet.HostAISessionEventsCursor{cursorFor(sess, t1)}), t1), 21)

	// Once it moves on, those calls get their outcome.
	t2 := start.Add(30 * time.Minute)
	require.Equal(t, 60, sess.callsAt(t2))
	for _, p := range undecided(t2) {
		require.NotContains(t, first, p)
	}

	// An ended session has no undecided calls and stops growing.
	t3 := start.Add(2 * time.Hour)
	require.Equal(t, 100, sess.callsAt(t3))
	require.Empty(t, undecided(t3))
	require.Equal(t, start.Add(100*30*time.Second), sess.endedAt(t3))
}

func TestAISessionsIncomplete(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	t.Run("deleted transcripts", func(t *testing.T) {
		s := testAISessionsSim(t, aiSessionsConfig{initialCount: 30, deletedProb: 1}, now)
		paths := make(map[string]*fleet.HostAISessionEventsCursor)
		for _, sess := range s.sessions {
			paths[sess.id()] = cursorFor(sess, now)
		}
		// Each transcript is deleted after its session was reported one to three times.
		reported := 0
		for range 3 {
			reported += len(s.sessionsRows(now))
		}
		require.Greater(t, reported, 30)
		require.Empty(t, s.sessionsRows(now))
		rows := s.eventsRows(buildAISessionEventsQuery(t, slices.Collect(maps.Values(paths))[:5]), now)
		require.Len(t, rows, 5)
		for _, r := range rows {
			require.Equal(t, "1", r["requested"])
		}
	})

	t.Run("uncounted tool calls", func(t *testing.T) {
		s := testAISessionsSim(t, aiSessionsConfig{initialCount: 10, uncountedProb: 1}, now)
		rows := s.sessionsRows(now)
		require.Len(t, rows, 10)
		for _, r := range rows {
			sess := s.byID(r["session_id"])
			require.NotNil(t, sess)
			// fleetd counts more tool calls than it returns for the session.
			require.Equal(t, strconv.Itoa(sess.callsAt(now)+sess.uncounted), r["tool_calls"])
			require.Positive(t, sess.uncounted)
			events := s.eventsRows(buildAISessionEventsQuery(t, []*fleet.HostAISessionEventsCursor{cursorFor(sess, now)}), now)
			require.Len(t, aiEventIndexes(t, events, sess.id()), sess.callsAt(now))
		}
	})
}

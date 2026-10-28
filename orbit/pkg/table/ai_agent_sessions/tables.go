// Package ai_agent_sessions exposes the session history of AI coding agents
// (Claude Code, Copilot CLI, Cursor, Codex CLI, Gemini CLI) found in every
// user's home directory as two tables: ai_agent_sessions, one metadata row
// per session, and ai_agent_session_messages, one row per content block.
//
// Ported from github.com/allenhouchins/fleet-extensions/ai_agent_sessions
// (MIT). The extension's command-line flags became the constants below.
package ai_agent_sessions

import (
	"context"
	"sort"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

const (
	// maxAgeDays bounds how far back transcripts are reported (0 for all
	// history).
	maxAgeDays = 30

	// Limits for ai_agent_session_messages, which returns conversation text
	// and is unbounded without them.
	maxRows         = 5000 // rows per query (0 for no limit)
	maxContentBytes = 8192 // bytes per content block (0 for no limit)
)

// SessionsColumns returns the ai_agent_sessions schema.
func SessionsColumns() []table.ColumnDefinition { return aiAgentSessionsColumns() }

// MessagesColumns returns the ai_agent_session_messages schema.
func MessagesColumns() []table.ColumnDefinition { return aiAgentSessionMessagesColumns() }

// GenerateSessions produces the ai_agent_sessions rows.
func GenerateSessions(ctx context.Context, qc table.QueryContext) ([]map[string]string, error) {
	return generateAIAgentSessions(ctx, qc)
}

// GenerateMessages produces the ai_agent_session_messages rows.
func GenerateMessages(ctx context.Context, qc table.QueryContext) ([]map[string]string, error) {
	return generateAIAgentSessionMessages(ctx, qc)
}

func aiAgentSessionsColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("agent"),
		table.TextColumn("session_id"),
		table.TextColumn("username"),
		table.TextColumn("project_path"),
		table.TextColumn("git_branch"),
		table.TextColumn("git_repository"),
		table.TextColumn("agent_version"),
		table.TextColumn("model"),
		table.BigIntColumn("started_at"),
		table.BigIntColumn("ended_at"),
		table.BigIntColumn("duration_seconds"),
		table.BigIntColumn("user_messages"),
		table.BigIntColumn("assistant_messages"),
		table.BigIntColumn("tool_calls"),
		table.BigIntColumn("input_tokens"),
		table.BigIntColumn("output_tokens"),
		table.BigIntColumn("cache_read_tokens"),
		table.BigIntColumn("cache_write_tokens"),
		table.TextColumn("transcript_path"),
		table.BigIntColumn("transcript_size_bytes"),
	}
}

func generateAIAgentSessions(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	started := time.Now()

	users := discoverUsers()
	sessions := collectSessions(users, cutoffTime(started, maxAgeDays))

	log.Debug().Int("homes", len(users)).Int("sessions", len(sessions)).Dur("took", time.Since(started)).
		Msg("ai_agent_sessions: scan done")

	results := make([]map[string]string, 0, len(sessions))
	for _, s := range sessions {
		results = append(results, s.row())
	}
	return results, nil
}

func generateAIAgentSessionMessages(_ context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	started := time.Now()

	users := discoverUsers()
	filter := newMessageFilter(queryContext)
	messages := collectMessages(users, cutoffTime(started, maxAgeDays), filter, maxRows, maxContentBytes)

	log.Debug().Int("homes", len(users)).Int("rows", len(messages)).Dur("took", time.Since(started)).
		Msg("ai_agent_session_messages: scan done")
	if maxRows > 0 && len(messages) == maxRows {
		// Never let a capped result read as a complete one.
		log.Warn().Int("max_rows", maxRows).
			Msg("ai_agent_session_messages: result truncated at the row limit; narrow the query with session_id/agent/username")
	}

	results := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		results = append(results, m.row())
	}
	return results, nil
}

// collectSessions runs every agent collector against every user, newest
// session first.
func collectSessions(users []userHome, cutoff time.Time) []Session {
	var sessions []Session
	for _, u := range users {
		for _, collect := range collectors {
			sessions = append(sessions, collect(u, cutoff)...)
		}
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		if !sessions[i].EndedAt.Equal(sessions[j].EndedAt) {
			return sessions[i].EndedAt.After(sessions[j].EndedAt)
		}
		return sessions[i].SessionID < sessions[j].SessionID
	})
	return sessions
}

// cutoffTime is the oldest last-write time a transcript may have and still be
// reported. A non-positive maxAge disables the limit.
func cutoffTime(now time.Time, maxAgeDays int) time.Time {
	if maxAgeDays <= 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, -maxAgeDays)
}

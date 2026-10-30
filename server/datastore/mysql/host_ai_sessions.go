package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	hostAISessionsBatchSize      = 500
	hostAISessionEventsBatchSize = 200

	hostAISessionsCleanupBatchSize  = 500
	hostAISessionsCleanupMaxBatches = 200 // 100k rows per tick
)

const (
	// A session that had started this long before Fleet first stored it is a backfill, as when a host first reports
	// its sessions: its calls are read hostAISessionBackfillCalls at a time, one such session per read, so a rollout
	// doesn't write every host's history at once. One that started more recently is read as usual.
	hostAISessionBackfillAge   = 2 * time.Hour
	hostAISessionBackfillCalls = 100
	// hostAISessionsCleanupGrace keeps a session past the retention, so one whose host clock is a little behind
	// Fleet's isn't deleted while the host still reports it, then stored and read again.
	hostAISessionsCleanupGrace = 24 * time.Hour
)

// Widths of the host_ai_sessions VARCHAR columns. Values are cut to fit rather than failing the host's ingestion.
const (
	hostAISessionAgentMaxLen    = 32
	hostAISessionIDMaxLen       = 255
	hostAISessionVersionMaxLen  = 64
	hostAISessionFieldMaxLen    = 255
	hostAISessionActionMaxLen   = 16
	hostAISessionTargetMaxLen   = 256
	hostAISessionDecisionMaxLen = 16
)

const (
	// hostAISessionDecisionWindow is how long before the session's end a call can have run and still have its decision
	// looked for. A call's result is recorded when the call finishes, which for one that starts a sub-agent comes after
	// all the sub-agent's calls; one still undecided after that long is final, and rereading it on every refresh of an
	// active session would re-send the session's tail each time.
	hostAISessionDecisionWindow = time.Hour
	// hostAISessionMaxEmptyEventReads is how many reads of a session may return no events before Fleet stops asking
	// until the session is active again, so one whose transcript is gone from the host doesn't keep its place ahead
	// of the others.
	hostAISessionMaxEmptyEventReads = 3
)

type hostAISessionKey struct {
	agent     string
	sessionID string
}

func newHostAISessionKey(agent, sessionID string) hostAISessionKey {
	return hostAISessionKey{
		agent:     truncateRunes(agent, hostAISessionAgentMaxLen),
		sessionID: truncateRunes(sessionID, hostAISessionIDMaxLen),
	}
}

func (ds *Datastore) UpsertHostAISessions(ctx context.Context, hostID uint, sessions []*fleet.HostAISession) error {
	// backfill is set when the session is first stored. Assignments see the values of those before them, so
	// events_empty_reads compares ended_at and tool_call_count before they're assigned: a session given up on is read
	// again once it's active or has new calls, as Cursor's end time comes from its own metadata and may not move with
	// them.
	const upsertStmt = `
INSERT INTO host_ai_sessions (
	host_id, agent, agent_session_id, agent_version, username, account_identifier, model, project_path, git_repository,
	git_branch, started_at, ended_at, user_message_count, assistant_message_count, tool_call_count, input_token_count,
	output_token_count, cache_read_token_count, cache_write_token_count, transcript_path, transcript_size,
	backfill
) VALUES %s
ON DUPLICATE KEY UPDATE
	events_empty_reads = IF(ended_at <=> VALUES(ended_at) AND tool_call_count >= VALUES(tool_call_count), events_empty_reads, 0),
	agent_version = VALUES(agent_version),
	username = VALUES(username),
	account_identifier = VALUES(account_identifier),
	model = VALUES(model),
	project_path = VALUES(project_path),
	git_repository = VALUES(git_repository),
	git_branch = VALUES(git_branch),
	started_at = VALUES(started_at),
	ended_at = VALUES(ended_at),
	user_message_count = VALUES(user_message_count),
	assistant_message_count = VALUES(assistant_message_count),
	tool_call_count = VALUES(tool_call_count),
	input_token_count = VALUES(input_token_count),
	output_token_count = VALUES(output_token_count),
	cache_read_token_count = VALUES(cache_read_token_count),
	cache_write_token_count = VALUES(cache_write_token_count),
	transcript_path = VALUES(transcript_path),
	transcript_size = VALUES(transcript_size)`

	const updateStmt = `
UPDATE host_ai_sessions SET
	events_empty_reads = IF(ended_at <=> ? AND tool_call_count >= ?, events_empty_reads, 0),
	agent_version = ?, username = ?, account_identifier = ?, model = ?, project_path = ?, git_repository = ?,
	git_branch = ?, started_at = ?, ended_at = ?, user_message_count = ?, assistant_message_count = ?,
	tool_call_count = ?, input_token_count = ?, output_token_count = ?, cache_read_token_count = ?,
	cache_write_token_count = ?, transcript_path = ?, transcript_size = ?
WHERE id = ?`

	// Values as the columns hold them, so they compare equal to the stored ones. A session listed twice is kept as
	// last listed.
	var keys []hostAISessionKey
	reported := make(map[hostAISessionKey]*fleet.HostAISession, len(sessions))
	for _, s := range sessions {
		key := newHostAISessionKey(s.Agent, s.AgentSessionID)
		c := *s
		c.Agent, c.AgentSessionID = key.agent, key.sessionID
		c.AgentVersion = truncateRunes(s.AgentVersion, hostAISessionVersionMaxLen)
		c.Username = truncateRunes(s.Username, hostAISessionFieldMaxLen)
		c.AccountIdentifier = truncateRunes(s.AccountIdentifier, hostAISessionFieldMaxLen)
		c.Model = truncateRunes(s.Model, hostAISessionFieldMaxLen)
		c.GitRepository = truncateRunes(s.GitRepository, hostAISessionFieldMaxLen)
		c.GitBranch = truncateRunes(s.GitBranch, hostAISessionFieldMaxLen)
		if _, ok := reported[key]; !ok {
			keys = append(keys, key)
		}
		reported[key] = &c
	}

	if len(sessions) == 0 {
		return nil
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Only new and changed sessions are written. The host reports all its recent sessions at every refresh, and
		// rewriting the unchanged ones with INSERT ... ON DUPLICATE KEY UPDATE takes next-key locks at the end of the
		// primary key that deadlock other hosts' refreshes.
		var stored []*fleet.HostAISession
		if err := sqlx.SelectContext(ctx, tx, &stored, `
SELECT
	id, agent, agent_session_id, agent_version, username, account_identifier, model, project_path, git_repository,
	git_branch, started_at, ended_at, user_message_count, assistant_message_count, tool_call_count, input_token_count,
	output_token_count, cache_read_token_count, cache_write_token_count, transcript_path, transcript_size
FROM host_ai_sessions WHERE host_id = ?`, hostID); err != nil {
			return ctxerr.Wrap(ctx, err, "select stored host ai sessions")
		}
		storedByKey := make(map[hostAISessionKey]*fleet.HostAISession, len(stored))
		for _, s := range stored {
			storedByKey[hostAISessionKey{agent: s.Agent, sessionID: s.AgentSessionID}] = s
		}

		var toInsert []*fleet.HostAISession
		for _, key := range keys {
			s := reported[key]
			old, ok := storedByKey[key]
			if !ok {
				toInsert = append(toInsert, s)
				continue
			}
			if hostAISessionUnchanged(old, s) {
				continue
			}
			if _, err := tx.ExecContext(ctx, updateStmt,
				s.EndedAt, s.ToolCallCount, s.AgentVersion, s.Username, s.AccountIdentifier, s.Model, s.ProjectPath, s.GitRepository,
				s.GitBranch, s.StartedAt, s.EndedAt, s.UserMessageCount, s.AssistantMessageCount, s.ToolCallCount,
				s.InputTokenCount, s.OutputTokenCount, s.CacheReadTokenCount, s.CacheWriteTokenCount, s.TranscriptPath,
				s.TranscriptSize, old.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "update host ai session")
			}
		}

		// A session whose start isn't known may have been running for long, so it's a backfill too.
		backfillBefore := time.Now().Add(-hostAISessionBackfillAge)
		generateValueArgs := func(s *fleet.HostAISession) (string, []any) {
			backfill := s.ToolCallCount > 0 && (s.StartedAt == nil || s.StartedAt.Before(backfillBefore))
			return "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),", []any{
				hostID, s.Agent, s.AgentSessionID, s.AgentVersion, s.Username, s.AccountIdentifier, s.Model,
				s.ProjectPath, s.GitRepository, s.GitBranch, s.StartedAt, s.EndedAt,
				s.UserMessageCount, s.AssistantMessageCount, s.ToolCallCount,
				s.InputTokenCount, s.OutputTokenCount, s.CacheReadTokenCount, s.CacheWriteTokenCount,
				s.TranscriptPath, s.TranscriptSize,
				backfill,
			}
		}
		// New sessions are still upserted: a refresh racing this one may have stored them first.
		executeUpsertBatch := func(valuePart string, args []any) error {
			_, err := tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, strings.TrimSuffix(valuePart, ",")), args...)
			return ctxerr.Wrap(ctx, err, "upsert host ai sessions")
		}
		return batchProcessDB(toInsert, hostAISessionsBatchSize, generateValueArgs, executeUpsertBatch)
	})
}

func hostAISessionUnchanged(stored, reported *fleet.HostAISession) bool {
	return stored.AgentVersion == reported.AgentVersion &&
		stored.Username == reported.Username &&
		stored.AccountIdentifier == reported.AccountIdentifier &&
		stored.Model == reported.Model &&
		stored.ProjectPath == reported.ProjectPath &&
		stored.GitRepository == reported.GitRepository &&
		stored.GitBranch == reported.GitBranch &&
		sameAISessionTime(stored.StartedAt, reported.StartedAt) &&
		sameAISessionTime(stored.EndedAt, reported.EndedAt) &&
		stored.UserMessageCount == reported.UserMessageCount &&
		stored.AssistantMessageCount == reported.AssistantMessageCount &&
		stored.ToolCallCount == reported.ToolCallCount &&
		stored.InputTokenCount == reported.InputTokenCount &&
		stored.OutputTokenCount == reported.OutputTokenCount &&
		stored.CacheReadTokenCount == reported.CacheReadTokenCount &&
		stored.CacheWriteTokenCount == reported.CacheWriteTokenCount &&
		stored.TranscriptPath == reported.TranscriptPath &&
		stored.TranscriptSize == reported.TranscriptSize
}

func sameAISessionTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func (ds *Datastore) ListHostAISessionsNeedingEvents(ctx context.Context, hostID uint, limit int) ([]*fleet.HostAISessionEventsCursor, error) {
	// Catching up comes before rereading undecided events: an active session's ended_at moves at every refresh, so
	// rereading first would re-send the same events each time and never reach the new ones. An undecided event can
	// only get its decision if the agent wrote to the transcript after the event was read, so a caught-up session is
	// reread only once its ended_at moves past the one its events were read at. A backfill still behind is read
	// apart, at most one per read and a few calls at a time. Sessions within a day of the cleanup's cutoffs are left
	// out, so their events aren't being stored while the cleanup deletes them.
	const stmt = `
(SELECT
	agent, agent_session_id, transcript_path, ended_at, event_count,
	IF(event_count < tool_call_count, NULL, first_undecided_tool_call_index) AS reread_from_tool_call_index,
	0 AS max_tool_calls
FROM host_ai_sessions
WHERE host_id = ? AND transcript_path <> '' AND events_empty_reads < ? AND (started_at IS NULL OR started_at >= ?) AND
	updated_at >= ? AND NOT (backfill AND event_count < tool_call_count) AND (
	event_count < tool_call_count OR
	(agent <> ? AND first_undecided_tool_call_index IS NOT NULL AND ended_at > COALESCE(events_ended_at, '1000-01-01'))
)
ORDER BY ended_at DESC
LIMIT ?)
UNION ALL
(SELECT
	agent, agent_session_id, transcript_path, ended_at, event_count, NULL AS reread_from_tool_call_index,
	? AS max_tool_calls
FROM host_ai_sessions
WHERE host_id = ? AND transcript_path <> '' AND events_empty_reads < ? AND (started_at IS NULL OR started_at >= ?) AND
	updated_at >= ? AND backfill AND event_count < tool_call_count
ORDER BY ended_at DESC
LIMIT 1)`
	startedAfter := time.Now().UTC().Add(-fleet.HostAISessionsMaxAge)
	updatedAfter := startedAfter.Add(hostAISessionsCleanupGrace)
	var cursors []*fleet.HostAISessionEventsCursor
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cursors, stmt,
		hostID, hostAISessionMaxEmptyEventReads, startedAfter, updatedAfter, fleet.AISessionAgentCursor, limit,
		hostAISessionBackfillCalls, hostID, hostAISessionMaxEmptyEventReads, startedAfter, updatedAfter); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host ai sessions needing events")
	}
	return cursors, nil
}

// lockHostAISessions locks the host's stored sessions among keys until the transaction ends, so a session can't be
// deleted while its events are stored, nor its events stored by another transaction. It must be the transaction's
// first read: the reads after it then see every event stored before the locks were taken.
func lockHostAISessions(ctx context.Context, q sqlx.QueryerContext, hostID uint, keys []hostAISessionKey) (map[hostAISessionKey]uint, error) {
	var sb strings.Builder
	args := []any{hostID}
	for i, key := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?)")
		args = append(args, key.agent, key.sessionID)
	}
	var stored []struct {
		ID             uint   `db:"id"`
		Agent          string `db:"agent"`
		AgentSessionID string `db:"agent_session_id"`
	}
	if err := sqlx.SelectContext(ctx, q, &stored, fmt.Sprintf(
		`SELECT id, agent, agent_session_id FROM host_ai_sessions WHERE host_id = ? AND (agent, agent_session_id) IN (%s) FOR UPDATE`,
		sb.String()), args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "lock host ai sessions for events")
	}
	ids := make(map[hostAISessionKey]uint, len(stored))
	for _, s := range stored {
		ids[hostAISessionKey{agent: s.Agent, sessionID: s.AgentSessionID}] = s.ID
	}
	return ids, nil
}

func (ds *Datastore) UpsertHostAISessionEvents(ctx context.Context, hostID uint, requested []*fleet.HostAISessionEventsCursor, events []*fleet.HostAISessionEvent) error {
	// A decision, once recorded, is kept: a later read of the same call may come back without its result.
	const upsertStmt = `
INSERT INTO host_ai_session_events (
	ai_session_id, tool_call_index, action, target, decision, executed_at, payload, content_length, truncated
) VALUES %s
ON DUPLICATE KEY UPDATE
	decision = IF(VALUES(decision) = '', decision, VALUES(decision))`

	// The session's summary of its events, read by ListHostAISessionsNeedingEvents. Reads resume after the stored
	// events, so a read that started after an undecided call may have missed its decision in what it read:
	// events_ended_at then stays, so the call is reread once the session is caught up. Assignments see the values of
	// those before them, so first_undecided_tool_call_index is set before events_ended_at looks at it. actions is a
	// SET, so only its members are gathered: an action a newer fleetd reports stays on the event.
	const updateSessionStmt = `
UPDATE host_ai_sessions SET
	event_count = (SELECT COUNT(*) FROM host_ai_session_events WHERE ai_session_id = ?),
	backfill = backfill AND event_count < tool_call_count,
	actions = (
		SELECT COALESCE(GROUP_CONCAT(DISTINCT action), '') FROM host_ai_session_events
		WHERE ai_session_id = ? AND action IN ('read', 'write', 'execute', 'search', 'network', 'mcp', 'other')
	),
	first_undecided_tool_call_index = (
		SELECT MIN(tool_call_index) FROM host_ai_session_events WHERE ai_session_id = ? AND decision = '' AND
			executed_at >= COALESCE(?, executed_at) - INTERVAL ? SECOND
	),
	events_ended_at = IF(first_undecided_tool_call_index < ?, events_ended_at, ?),
	events_empty_reads = 0
WHERE id = ?`

	// Sessions in the order their events first appear, with the ended_at their events were read at and the index
	// the read started from.
	var keys []hostAISessionKey
	endedAt := make(map[hostAISessionKey]*time.Time)
	readFrom := make(map[hostAISessionKey]uint)
	for _, e := range events {
		key := newHostAISessionKey(e.Agent, e.AgentSessionID)
		if _, ok := endedAt[key]; !ok {
			keys = append(keys, key)
			endedAt[key] = e.SessionEndedAt
			readFrom[key] = e.ToolCallIndex
		} else {
			readFrom[key] = min(readFrom[key], e.ToolCallIndex)
		}
	}
	// Requested sessions with no events in the result.
	var empty []hostAISessionKey
	for _, c := range requested {
		if key := newHostAISessionKey(c.Agent, c.AgentSessionID); !slices.Contains(keys, key) {
			empty = append(empty, key)
		}
	}
	if len(keys) == 0 && len(empty) == 0 {
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		sessionIDs, err := lockHostAISessions(ctx, tx, hostID, slices.Concat(keys, empty))
		if err != nil {
			return err
		}

		// Stored events that are read again only have their decision updated. Inserting them would hit the unique key:
		// InnoDB undoes the row it already put in the primary key, keeping a gap lock there, and transactions given
		// interleaved ids then deadlock inserting into each other's gaps. The sessions are locked, so what's read here
		// can't change before the insert.
		stored := make(map[uint]map[uint]struct{}, len(keys))
		for _, key := range keys {
			id, ok := sessionIDs[key]
			if !ok {
				continue
			}
			var indexes []uint
			if err := sqlx.SelectContext(ctx, tx, &indexes,
				`SELECT tool_call_index FROM host_ai_session_events WHERE ai_session_id = ? AND tool_call_index >= ?`,
				id, readFrom[key]); err != nil {
				return ctxerr.Wrap(ctx, err, "select stored host ai session events")
			}
			stored[id] = make(map[uint]struct{}, len(indexes))
			for _, i := range indexes {
				stored[id][i] = struct{}{}
			}
		}
		type decisionKey struct {
			sessionID uint
			decision  string
		}
		decided := make(map[decisionKey][]uint)
		toInsert := make([]*fleet.HostAISessionEvent, 0, len(events))
		for _, e := range events {
			id, ok := sessionIDs[newHostAISessionKey(e.Agent, e.AgentSessionID)]
			if !ok {
				continue
			}
			if _, ok := stored[id][e.ToolCallIndex]; !ok {
				toInsert = append(toInsert, e)
			} else if e.Decision != "" {
				k := decisionKey{id, truncateRunes(e.Decision, hostAISessionDecisionMaxLen)}
				decided[k] = append(decided[k], e.ToolCallIndex)
			}
		}
		for k, indexes := range decided {
			stmt, args, err := sqlx.In(`UPDATE host_ai_session_events SET decision = ?
				WHERE ai_session_id = ? AND decision <> ? AND tool_call_index IN (?)`, k.decision, k.sessionID, k.decision, indexes)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build update host ai session event decisions")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "update host ai session event decisions")
			}
		}
		generateValueArgs := func(e *fleet.HostAISessionEvent) (string, []any) {
			return "(?, ?, ?, ?, ?, ?, ?, ?, ?),", []any{
				sessionIDs[newHostAISessionKey(e.Agent, e.AgentSessionID)],
				e.ToolCallIndex,
				truncateRunes(string(e.Action), hostAISessionActionMaxLen),
				truncateRunes(e.Target, hostAISessionTargetMaxLen),
				truncateRunes(e.Decision, hostAISessionDecisionMaxLen),
				e.ExecutedAt,
				string(e.Payload),
				e.ContentLength,
				e.Truncated,
			}

		}
		executeUpsertBatch := func(valuePart string, args []any) error {
			_, err := tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, strings.TrimSuffix(valuePart, ",")), args...)
			return ctxerr.Wrap(ctx, err, "upsert host ai session events")
		}
		if err := batchProcessDB(toInsert, hostAISessionEventsBatchSize, generateValueArgs, executeUpsertBatch); err != nil {
			return err
		}

		for _, key := range keys {
			id, ok := sessionIDs[key]
			if !ok {
				continue
			}
			if _, err := tx.ExecContext(ctx, updateSessionStmt,
				id, id, id, endedAt[key], int(hostAISessionDecisionWindow.Seconds()), readFrom[key], endedAt[key], id); err != nil {
				return ctxerr.Wrap(ctx, err, "update host ai session events summary")
			}

		}

		var emptyIDs []uint
		for _, key := range empty {
			if id, ok := sessionIDs[key]; ok {
				emptyIDs = append(emptyIDs, id)
			}
		}
		if len(emptyIDs) > 0 {
			stmt, args, err := sqlx.In(
				`UPDATE host_ai_sessions SET events_empty_reads = events_empty_reads + 1 WHERE id IN (?) AND events_empty_reads < ?`,
				emptyIDs, hostAISessionMaxEmptyEventReads)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build update host ai session empty reads")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "update host ai session empty reads")
			}
		}
		return nil
	})
}

func (ds *Datastore) CleanupHostAISessions(ctx context.Context, before time.Time) (int64, error) {
	// A session can have thousands of events, so they're deleted in batches before their session rather than by one
	// cascading DELETE. The budget is checked between sessions: a session goes whole, so its stored events never stop
	// being a contiguous run should it be reported again. Sessions are kept for as long after they started, as hosts
	// only report those that started that recently. started_at is the host's clock, so a session whose start is
	// unknown or in the future is deleted once Fleet hasn't seen it change for as long instead; the second arm skips
	// those the first covers so each batch is distinct.
	const selectStmt = `
SELECT id FROM (
	(SELECT id FROM host_ai_sessions WHERE started_at < ? LIMIT ?)
	UNION ALL
	(SELECT id FROM host_ai_sessions WHERE updated_at < ? AND (started_at IS NULL OR started_at >= ?) LIMIT ?)
) expired LIMIT ?`
	startedBefore := before.Add(-hostAISessionsCleanupGrace)
	batchSize := hostAISessionsCleanupBatchSize
	// The budget counts rows, not statements: most sessions have fewer events than a batch.
	maxRows := int64(batchSize * hostAISessionsCleanupMaxBatches)

	var deleted, rows int64
	for {
		var ids []uint
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &ids, selectStmt,
			startedBefore, batchSize, before, startedBefore, batchSize, batchSize); err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "select expired host ai sessions")
		}
		for _, id := range ids {
			if rows >= maxRows {
				ds.logger.WarnContext(ctx, "cleanup host ai sessions hit its row cap, any remaining rows are cleaned on the next run",
					"deleted", deleted, "max_rows", maxRows)
				return deleted, nil
			}
			for {
				res, err := ds.writer(ctx).ExecContext(ctx,
					`DELETE FROM host_ai_session_events WHERE ai_session_id = ? LIMIT ?`, id, batchSize)
				if err != nil {
					return deleted, ctxerr.Wrap(ctx, err, "delete expired host ai session events")
				}
				n, _ := res.RowsAffected()
				rows += n
				if n < int64(batchSize) {
					break
				}
			}
			// Deleted by id even if reported again since it was selected: its events are gone, and a session stored
			// anew is collected from a consistent start.
			res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_ai_sessions WHERE id = ?`, id)
			if err != nil {
				return deleted, ctxerr.Wrap(ctx, err, "delete expired host ai session")
			}
			n, _ := res.RowsAffected()
			deleted += n
			rows += n
		}
		if len(ids) < batchSize {
			return deleted, nil
		}
	}
}

// hostAISessionOrderColumns are the columns the order keys sort by. The nullable ones hold timestamps an agent may not
// record, which sort last whichever the direction so a cursor still reaches them.
var hostAISessionOrderColumns = map[string]struct {
	column   string
	nullable bool
}{
	"started_at":         {"started_at", true},
	"ended_at":           {"ended_at", true},
	"agent":              {"agent", false},
	"username":           {"username", false},
	"model":              {"model", false},
	"output_token_count": {"output_token_count", false},
}

const hostAISessionColumns = `
	id, host_id, agent, agent_version, agent_session_id, username, account_identifier, model, project_path,
	git_repository, git_branch, started_at, ended_at, user_message_count, assistant_message_count, tool_call_count,
	actions, backfill, input_token_count, output_token_count, cache_read_token_count, cache_write_token_count,
	transcript_path, transcript_size`

// aiSessionCategories maps actions to their API categories, in the order of the categories, always returning a
// non-nil slice so it's listed as [].
func aiSessionCategories(actions ...fleet.AISessionAction) []fleet.AISessionCategory {
	categories := []fleet.AISessionCategory{}
	for _, c := range []fleet.AISessionCategory{fleet.AISessionCategoryCommand, fleet.AISessionCategoryFileAccess} {
		if slices.ContainsFunc(actions, func(a fleet.AISessionAction) bool { return a.Category() == c }) {
			categories = append(categories, c)
		}
	}
	return categories
}

// aiSessionCategoryActions are the actions of each API category, for filtering on them.
var aiSessionCategoryActions = map[fleet.AISessionCategory][]fleet.AISessionAction{
	fleet.AISessionCategoryCommand:    {fleet.AISessionActionExecute},
	fleet.AISessionCategoryFileAccess: {fleet.AISessionActionRead, fleet.AISessionActionWrite, fleet.AISessionActionSearch},
}

type hostAISessionRow struct {
	fleet.HostAISession
	ActionsSet string `db:"actions"`
}

func (r *hostAISessionRow) toSession() *fleet.HostAISession {
	s := r.HostAISession
	s.Actions = []fleet.AISessionAction{}
	if r.ActionsSet != "" {
		for a := range strings.SplitSeq(r.ActionsSet, ",") {
			s.Actions = append(s.Actions, fleet.AISessionAction(a))
		}
	}
	s.Categories = aiSessionCategories(s.Actions...)
	return &s
}

func (ds *Datastore) ListHostAISessions(ctx context.Context, hostID uint, opts fleet.HostAISessionListOptions) ([]*fleet.HostAISession, *fleet.PaginationMetadata, error) {
	where := ` FROM host_ai_sessions WHERE host_id = ?`
	args := []any{hostID}
	if opts.Agent != "" {
		where += ` AND agent = ?`
		args = append(args, opts.Agent)
	}
	if opts.Category != "" {
		actions, ok := aiSessionCategoryActions[opts.Category]
		if !ok {
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("category", "unsupported category: "+string(opts.Category)))
		}
		var inSet []string
		for _, a := range actions {
			inSet = append(inSet, `FIND_IN_SET(?, actions) > 0`)
			args = append(args, a)
		}
		where += ` AND (` + strings.Join(inSet, ` OR `) + `)`
	}
	where, args = searchLike(where, args, opts.MatchQuery, "username", "project_path", "git_repository", "model")

	orderKey := opts.OrderKey
	if orderKey == "" {
		orderKey = "ended_at"
	}
	order, ok := hostAISessionOrderColumns[orderKey]
	if !ok {
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("order_key", "unsupported order key: "+orderKey))
	}
	direction, comparison := "ASC", ">"
	if opts.OrderDirection == fleet.OrderDescending {
		direction, comparison = "DESC", "<"
	}
	listWhere, listArgs := where, slices.Clone(args)
	// The order and cursor are built here rather than by the list options helper, so unknown timestamps sort last and
	// the cursor includes them. The order keys aren't unique, so ties are broken by id to keep pages stable; the cursor
	// compares the order column alone, like the helper's, so it skips the rows tied with it.
	listOpts := opts.ListOptions
	listOpts.OrderKey = ""
	if opts.After != "" {
		if order.nullable {
			listWhere += fmt.Sprintf(" AND (%s %s ? OR %s IS NULL)", order.column, comparison, order.column)
		} else {
			listWhere += fmt.Sprintf(" AND %s %s ?", order.column, comparison)
		}
		listArgs = append(listArgs, opts.After)
		listOpts.After = ""
		listOpts.Page = 0 // the cursor supersedes the page
	}
	orderBy := " ORDER BY "
	if order.nullable {
		orderBy += order.column + " IS NULL, "
	}
	orderBy += fmt.Sprintf("%s %s, id %s", order.column, direction, direction)
	stmt, stmtArgs, err := appendListOptionsWithCursorToSQLSecure(
		`SELECT `+hostAISessionColumns+listWhere+orderBy, listArgs, &listOpts, common_mysql.OrderKeyAllowlist{})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "apply list options")
	}
	var rows []*hostAISessionRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, stmtArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host ai sessions")
	}

	var meta *fleet.PaginationMetadata
	if listOpts.IncludeMetadata {
		meta = &fleet.PaginationMetadata{HasPreviousResults: listOpts.Page > 0}
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &meta.TotalResults, `SELECT COUNT(*)`+where, args...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "count host ai sessions")
		}
		if len(rows) > int(listOpts.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			rows = rows[:len(rows)-1]
		}
	}

	sessions := make([]*fleet.HostAISession, 0, len(rows))
	for _, r := range rows {
		sessions = append(sessions, r.toSession())
	}
	return sessions, meta, nil
}

func (ds *Datastore) GetHostAISession(ctx context.Context, hostID, id uint) (*fleet.HostAISession, error) {
	var row hostAISessionRow
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row,
		`SELECT `+hostAISessionColumns+` FROM host_ai_sessions WHERE id = ? AND host_id = ?`, id, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostAISession").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host ai session")
	}
	return row.toSession(), nil
}

func (ds *Datastore) ListHostAISessionEvents(ctx context.Context, aiSessionID uint, opts fleet.HostAISessionEventListOptions) ([]*fleet.HostAISessionEvent, *fleet.PaginationMetadata, error) {
	where := ` FROM host_ai_session_events WHERE ai_session_id = ?`
	args := []any{aiSessionID}
	if opts.Category != "" {
		actions, ok := aiSessionCategoryActions[opts.Category]
		if !ok {
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("category", "unsupported category: "+string(opts.Category)))
		}
		where += ` AND action IN (?` + strings.Repeat(`, ?`, len(actions)-1) + `)`
		for _, a := range actions {
			args = append(args, a)
		}
	}

	// Events are always listed in the order the calls ran, so the order isn't a list option.
	listOpts := opts.ListOptions
	listOpts.OrderKey = ""
	listOpts.After = ""
	stmt, stmtArgs, err := appendListOptionsWithCursorToSQLSecure(`
SELECT id, tool_call_index, action, target, decision, executed_at, payload, truncated`+where+`
ORDER BY tool_call_index`, slices.Clone(args), &listOpts, common_mysql.OrderKeyAllowlist{})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "apply list options")
	}
	var events []*fleet.HostAISessionEvent
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &events, stmt, stmtArgs...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host ai session events")
	}

	var meta *fleet.PaginationMetadata
	if listOpts.IncludeMetadata {
		meta = &fleet.PaginationMetadata{HasPreviousResults: listOpts.Page > 0}
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &meta.TotalResults, `SELECT COUNT(*)`+where, args...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "count host ai session events")
		}
		if len(events) > int(listOpts.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			events = events[:len(events)-1]
		}
	}

	if events == nil {
		events = []*fleet.HostAISessionEvent{}
	}
	for _, e := range events {
		e.Categories = aiSessionCategories(e.Action)
	}
	return events, meta, nil
}

package mysql

import (
	"context"
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

	hostAISessionsCleanupBatchSize = 500

	hostAISessionMaxEvents uint = fleet.HostAISessionMaxEvents
)

// hostAISessionsCleanupMargin is how long before the cleanup deletes a session its events stop being read, so they
// aren't being stored while it's deleted.
const hostAISessionsCleanupMargin = 24 * time.Hour

// hostAISessionActions are the members of the host_ai_sessions.actions SET.
var hostAISessionActions = []fleet.AISessionAction{
	fleet.AISessionActionRead, fleet.AISessionActionWrite, fleet.AISessionActionExecute, fleet.AISessionActionSearch,
	fleet.AISessionActionNetwork, fleet.AISessionActionMCP, fleet.AISessionActionOther,
}

// Widths of the VARCHAR columns. Values are cut to fit rather than failing the host's ingestion.
const (
	// host_ai_sessions
	hostAISessionAgentMaxLen   = 32
	hostAISessionIDMaxLen      = 255
	hostAISessionVersionMaxLen = 64
	hostAISessionFieldMaxLen   = 255

	// host_ai_session_events
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
	// Assignments see the values of those before them, so events_empty_reads compares ended_at and tool_call_count before they're assigned: a session given up on is read
	// again once it's active or has new calls, as Cursor's end time comes from its own metadata and may not move with
	// them.
	const upsertStmt = `
INSERT INTO host_ai_sessions (
	host_id, agent, agent_session_id, agent_version, username, account_identifier, model, project_path, git_repository,
	git_branch, started_at, ended_at, user_message_count, assistant_message_count, tool_call_count, input_token_count,
	output_token_count, cache_read_token_count, cache_write_token_count, transcript_path, transcript_size
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

	if len(sessions) == 0 {
		return nil
	}

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
	// Left nil when no reported session has a known end, which matches no stored one.
	var earliestEnd *time.Time
	for _, s := range reported {
		if s.EndedAt != nil && (earliestEnd == nil || s.EndedAt.Before(*earliestEnd)) {
			earliestEnd = s.EndedAt
		}
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Only new and changed sessions are written. The host reports all its recent sessions at every refresh, and
		// rewriting the unchanged ones with INSERT ... ON DUPLICATE KEY UPDATE takes next-key locks at the end of the
		// primary key that deadlock other hosts' refreshes.
		//
		// The host has stored sessions it no longer reports, so only those that ended since the earliest reported
		// one are read. A stored session that ended before then and is reported again is taken for new and upserted.
		var stored []*fleet.HostAISession
		if err := sqlx.SelectContext(ctx, tx, &stored, `
SELECT
	id, agent, agent_session_id, agent_version, username, account_identifier, model, project_path, git_repository,
	git_branch, started_at, ended_at, user_message_count, assistant_message_count, tool_call_count, input_token_count,
	output_token_count, cache_read_token_count, cache_write_token_count, transcript_path, transcript_size
FROM host_ai_sessions WHERE host_id = ? AND (ended_at >= ? OR ended_at IS NULL)`, hostID, earliestEnd); err != nil {
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

		generateValueArgs := func(s *fleet.HostAISession) (string, []any) {
			return "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),", []any{
				hostID, s.Agent, s.AgentSessionID, s.AgentVersion, s.Username, s.AccountIdentifier, s.Model,
				s.ProjectPath, s.GitRepository, s.GitBranch, s.StartedAt, s.EndedAt,
				s.UserMessageCount, s.AssistantMessageCount, s.ToolCallCount,
				s.InputTokenCount, s.OutputTokenCount, s.CacheReadTokenCount, s.CacheWriteTokenCount,
				s.TranscriptPath, s.TranscriptSize,
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
	// reread only once its ended_at moves past the one its events were read at. Sessions the cleanup is about to
	// delete are left out: those that ended in the past go by their end, and the IF's NULL condition sends those whose
	// end is unknown to the updated_at check, as it does those ending in the future. A session with as many events
	// stored as Fleet keeps is caught up.
	const stmt = `
SELECT
	agent, agent_session_id, transcript_path, ended_at, event_count,
	IF(event_count < LEAST(tool_call_count, ?), NULL, first_undecided_tool_call_index) AS reread_from_tool_call_index
FROM host_ai_sessions
WHERE host_id = ? AND transcript_path <> '' AND events_empty_reads < ? AND
	IF(ended_at <= ?, ended_at >= ?, updated_at >= ?) AND (
	event_count < LEAST(tool_call_count, ?) OR
	(agent <> ? AND first_undecided_tool_call_index IS NOT NULL AND ended_at > COALESCE(events_ended_at, '1000-01-01'))
)
ORDER BY ended_at DESC
LIMIT ?`
	now := time.Now().UTC()
	activeAfter := now.Add(-fleet.HostAISessionsMaxAge).Add(hostAISessionsCleanupMargin)
	var cursors []*fleet.HostAISessionEventsCursor
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &cursors, stmt,
		hostAISessionMaxEvents, hostID, hostAISessionMaxEmptyEventReads, now, activeAfter, activeAfter,
		hostAISessionMaxEvents, fleet.AISessionAgentCursor, limit); err != nil {
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

	// The session's summary of its events, read by ListHostAISessionsNeedingEvents. events_ended_at records that the
	// stored events were all read at the session's ended_at, so it stays when they weren't: when the read started
	// after an undecided call, which may have had its decision in what was read, or when fleetd's result budget cut
	// the read before the last stored call. The undecided calls are then reread once the session is caught up.
	// Assignments see the values of those before them, so event_count and first_undecided_tool_call_index are set
	// before events_ended_at looks at them.
	//
	// Only the count looks at all the session's events, in the index. Their rows hold the payloads, so the rest reads
	// only those this read touched: actions gains the new events' (a SET drops repeats), and the first undecided call
	// is looked for from the earlier of the last one found and where the read started. None before that can turn up,
	// as decisions aren't cleared and the window only moves forward with the session's end.
	const updateSessionStmt = `
UPDATE host_ai_sessions SET
	event_count = (SELECT COUNT(*) FROM host_ai_session_events WHERE ai_session_id = ?),
	actions = CONCAT_WS(',', NULLIF(actions, ''), NULLIF(?, '')),
	first_undecided_tool_call_index = (
		SELECT MIN(tool_call_index) FROM host_ai_session_events WHERE ai_session_id = ? AND
			tool_call_index >= LEAST(COALESCE(first_undecided_tool_call_index, ?), ?) AND decision = '' AND
			executed_at >= COALESCE(?, executed_at) - INTERVAL ? SECOND
	),
	events_ended_at = IF(first_undecided_tool_call_index < ? OR ? + 1 < event_count, events_ended_at, ?),
	events_empty_reads = 0
WHERE id = ?`

	// The query doesn't ask for calls past the limit, so these only come from a host that ignored its bound.
	events = slices.DeleteFunc(slices.Clone(events), func(e *fleet.HostAISessionEvent) bool {
		return e.ToolCallIndex >= hostAISessionMaxEvents
	})

	// Sessions in the order their events first appear, with the ended_at their events were read at and the indexes
	// the read started from and got to.
	var keys []hostAISessionKey
	endedAt := make(map[hostAISessionKey]*time.Time)
	readFrom := make(map[hostAISessionKey]uint)
	readTo := make(map[hostAISessionKey]uint)
	for _, e := range events {
		key := newHostAISessionKey(e.Agent, e.AgentSessionID)
		if _, ok := endedAt[key]; !ok {
			keys = append(keys, key)
			endedAt[key] = e.SessionEndedAt
			readFrom[key], readTo[key] = e.ToolCallIndex, e.ToolCallIndex
		} else {
			readFrom[key] = min(readFrom[key], e.ToolCallIndex)
			readTo[key] = max(readTo[key], e.ToolCallIndex)
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
		// Actions of the new events that the SET holds: one a newer fleetd reports stays on its event only.
		newActions := make(map[uint][]string, len(keys))
		for _, e := range events {
			id, ok := sessionIDs[newHostAISessionKey(e.Agent, e.AgentSessionID)]
			if !ok {
				continue
			}
			if _, ok := stored[id][e.ToolCallIndex]; !ok {
				toInsert = append(toInsert, e)
				if slices.Contains(hostAISessionActions, e.Action) && !slices.Contains(newActions[id], string(e.Action)) {
					newActions[id] = append(newActions[id], string(e.Action))
				}
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
			if _, err := tx.ExecContext(ctx, updateSessionStmt, id, strings.Join(newActions[id], ","), id, readFrom[key],
				readFrom[key], endedAt[key], int(hostAISessionDecisionWindow.Seconds()), readFrom[key], readTo[key],
				endedAt[key], id); err != nil {
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

func (ds *Datastore) CleanupHostAISessions(ctx context.Context, before time.Time, maxRunTime time.Duration) (int64, error) {
	// Sessions of deleted hosts go first: deleting a host leaves them, as their events can be too many to delete with
	// it. Hosts are looked up by the distinct host IDs rather than per session, which stays cheap however many
	// sessions there are.
	const deletedHostsStmt = `
SELECT s.host_id FROM (SELECT DISTINCT host_id FROM host_ai_sessions) s
LEFT JOIN hosts h ON h.id = s.host_id
WHERE h.id IS NULL`
	// Sessions are kept for as long after they were last active, well after hosts stop reporting them. ended_at is
	// the host's clock, so a session whose end is unknown or in the future is deleted once Fleet hasn't seen it change
	// for as long instead.
	const expiredStmt = `
SELECT id FROM (
	(SELECT id FROM host_ai_sessions WHERE ended_at < ? LIMIT ?)
	UNION ALL
	(SELECT id FROM host_ai_sessions WHERE updated_at < ? AND (ended_at IS NULL OR ended_at > ?) LIMIT ?)
) expired LIMIT ?`
	now := time.Now().UTC()
	deadline := time.Now().Add(maxRunTime)
	batchSize := hostAISessionsCleanupBatchSize

	// A session can have thousands of events, so they're deleted in batches before their session rather than by one
	// cascading DELETE, each statement on its own so none holds its locks for long.
	deleteSession := func(id uint) (int64, error) {
		for {
			res, err := ds.writer(ctx).ExecContext(ctx,
				`DELETE FROM host_ai_session_events WHERE ai_session_id = ? LIMIT ?`, id, batchSize)
			if err != nil {
				return 0, err
			}
			if n, _ := res.RowsAffected(); n < int64(batchSize) {
				break
			}
		}
		res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_ai_sessions WHERE id = ?`, id)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	// The time is checked between sessions: a session goes whole, so its stored events never stop being a contiguous
	// run should it be reported again. One that hits a lock error is left for the next run rather than ending this one.
	var deleted, skipped int64
	deleteSessions := func(ids []uint) (bool, error) {
		for _, id := range ids {
			if err := ctx.Err(); err != nil {
				return false, ctxerr.Wrap(ctx, err, "cleanup host ai sessions")
			}
			if !time.Now().Before(deadline) {
				ds.logger.InfoContext(ctx, "cleanup host ai sessions reached its run time limit, the rest is cleaned on the next run",
					"deleted", deleted, "max_run_time", maxRunTime)
				return false, nil
			}
			n, err := deleteSession(id)
			switch {
			case err == nil:
				deleted += n
			case common_mysql.RetryableError(err):
				skipped++
			default:
				return false, ctxerr.Wrap(ctx, err, "delete host ai session")
			}
		}
		return true, nil
	}
	defer func() {
		if skipped > 0 {
			ds.logger.WarnContext(ctx, "cleanup host ai sessions left sessions locked by other transactions for the next run",
				"skipped", skipped)
		}
	}()

	var deletedHosts []uint
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &deletedHosts, deletedHostsStmt); err != nil {
		return deleted, ctxerr.Wrap(ctx, err, "select deleted hosts with ai sessions")
	}
	for _, hostID := range deletedHosts {
		var ids []uint
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &ids, `SELECT id FROM host_ai_sessions WHERE host_id = ?`, hostID); err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "select deleted host ai sessions")
		}
		if ok, err := deleteSessions(ids); err != nil || !ok {
			return deleted, err
		}
	}

	for {
		var ids []uint
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &ids, expiredStmt,
			before, batchSize, before, now, batchSize, batchSize); err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "select expired host ai sessions")
		}
		if ok, err := deleteSessions(ids); err != nil || !ok || len(ids) < batchSize {
			return deleted, err
		}
	}
}

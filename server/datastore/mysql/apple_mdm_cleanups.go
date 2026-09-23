package mysql

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// nanoCleanupScanBatchSize bounds one candidate scan of a cleanup sweep. A
// var so tests can shrink it to exercise multi-scan runs.
var nanoCleanupScanBatchSize = 1000

// nanoCleanupMaxScansPerRun caps the candidate scans one sweep performs per
// run, independently of the row budget. A var so tests can stop a run
// mid-scan and check the cursor resumes on the next.
var nanoCleanupMaxScansPerRun = 50

// nanoCleanupAfterQueueDeleteHook, when set, runs between the queue and result
// deletes of a batch. Tests use it to fail the transaction midway and prove
// the pair is rolled back together.
var nanoCleanupAfterQueueDeleteHook func() error

// nanoCommandUnreferencedFilter is true for a nano_commands row c that no
// feature still reads state from. Every probe is an indexed point lookup, so
// the same full set is applied to every candidate rather than tailored per
// request type.
const nanoCommandUnreferencedFilter = `
	NOT EXISTS (SELECT 1 FROM host_mdm_actions hma WHERE hma.lock_ref = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_mdm_actions hma WHERE hma.wipe_ref = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_mdm_actions hma WHERE hma.unlock_ref = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_mdm_apple_profiles hmap WHERE hmap.command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_mdm_apple_bootstrap_packages hmabp WHERE hmabp.command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM nano_cert_auth_associations ncaa WHERE ncaa.renew_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_vpp_software_installs hvsi
		WHERE (hvsi.command_uuid = c.command_uuid OR hvsi.verification_command_uuid = c.command_uuid)
		AND hvsi.verification_at IS NULL AND hvsi.verification_failed_at IS NULL AND hvsi.canceled = 0)
	AND NOT EXISTS (SELECT 1 FROM host_in_house_software_installs hihsi
		WHERE (hihsi.command_uuid = c.command_uuid OR hihsi.verification_command_uuid = c.command_uuid)
		AND hihsi.verification_at IS NULL AND hihsi.verification_failed_at IS NULL AND hihsi.canceled = 0)
	AND NOT EXISTS (SELECT 1 FROM host_recovery_key_passwords rkp WHERE rkp.pending_set_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_recovery_key_passwords rkp WHERE rkp.pending_verify_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_recovery_key_passwords rkp WHERE rkp.set_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_recovery_key_passwords rkp WHERE rkp.verify_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_managed_local_account_passwords hmlap WHERE hmlap.command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM host_managed_local_account_passwords hmlap WHERE hmlap.pending_command_uuid = c.command_uuid)
	AND NOT EXISTS (SELECT 1 FROM setup_experience_status_results sesr
		WHERE sesr.nano_command_uuid = c.command_uuid AND sesr.status IN ('pending', 'running'))`

// nanoQueuePair identifies one nano_enrollment_queue row and its paired
// nano_command_results row.
type nanoQueuePair struct {
	ID          string `db:"id"`
	CommandUUID string `db:"command_uuid"`
}

// nanoQueueCandidate is a scanned queue row with the columns its keyset
// cursor is built from.
type nanoQueueCandidate struct {
	nanoQueuePair
	Priority  int       `db:"priority"`
	CreatedAt time.Time `db:"created_at"`
}

func (ds *Datastore) CleanupNanoCommands(ctx context.Context, opts fleet.MDMAppleCommandCleanupOptions, state *fleet.MDMAppleCommandCleanupState) (*fleet.MDMAppleCommandCleanupState, fleet.MDMAppleCommandCleanupStats, error) {
	var stats fleet.MDMAppleCommandCleanupStats
	if state == nil {
		state = &fleet.MDMAppleCommandCleanupState{}
	}
	if state.Retention == nil {
		state.Retention = make(map[string]fleet.MDMAppleCommandCleanupCursor)
	}

	// The row budget is shared by every pair sweep in order: once one sweep
	// exhausts it the later ones don't run this tick.
	budget := opts.MaxRowDeletions
	var touched []string
	if opts.ShortRetention > 0 && budget > 0 {
		deleted, uuids, exhausted, err := ds.purgeInactiveNanoCommands(ctx, opts.ShortRetention, budget)
		if err != nil {
			return state, stats, err
		}
		stats.InactivePairsDeleted = deleted
		stats.RowBudgetExhausted = exhausted
		budget -= deleted
		touched = append(touched, uuids...)
	}

	shortClasses := fleet.AppleMDMShortRetentionClasses
	if opts.ShortRetention > 0 {
		if budget > 0 && !stats.RowBudgetExhausted {
			deleted, uuids, exhausted, err := ds.sweepCompletedNanoCommands(ctx, state, "short", opts.ShortRetention, budget,
				func(rt, uuid string) bool { return matchesAppleMDMRetentionClass(shortClasses, rt, uuid) })
			if err != nil {
				return state, stats, err
			}
			stats.ShortPairsDeleted = deleted
			stats.RowBudgetExhausted = exhausted
			budget -= deleted
			touched = append(touched, uuids...)
		}
	} else {
		// with the short tier off, its classes are not kept forever: they
		// age out with the standard window instead
		shortClasses = nil
	}
	if opts.StandardRetention > 0 && budget > 0 && !stats.RowBudgetExhausted {
		deleted, uuids, exhausted, err := ds.sweepCompletedNanoCommands(ctx, state, "standard", opts.StandardRetention, budget,
			func(rt, uuid string) bool {
				return slices.Contains(fleet.AppleMDMStandardRetentionRequestTypes, rt) ||
					(shortClasses == nil && matchesAppleMDMRetentionClass(fleet.AppleMDMShortRetentionClasses, rt, uuid))
			})
		if err != nil {
			return state, stats, err
		}
		stats.StandardPairsDeleted = deleted
		stats.RowBudgetExhausted = exhausted
		touched = append(touched, uuids...)
	}

	if opts.MaxCmdDeletions > 0 && len(touched) > 0 {
		deleted, exhausted, err := ds.mopNanoCommands(ctx, uniqueStrings(touched), opts.MaxCmdDeletions)
		if err != nil {
			return state, stats, err
		}
		stats.CommandsDeleted = deleted
		stats.CmdBudgetExhausted = exhausted
	}
	return state, stats, nil
}

// matchesAppleMDMRetentionClass reports whether a command of request type rt
// and UUID uuid belongs to one of classes.
func matchesAppleMDMRetentionClass(classes []fleet.AppleMDMCommandRetentionClass, rt, uuid string) bool {
	for _, c := range classes {
		if c.RequestType == rt && strings.HasPrefix(uuid, c.UUIDPrefix) {
			return true
		}
	}
	return false
}

// nanoTerminalStatuses are the result statuses after which a device will not
// answer a command again. NotNow is deliberately absent: the command is still
// outstanding and nano re-serves it.
var nanoTerminalStatuses = []string{fleet.MDMAppleStatusAcknowledged, fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError}

// nanoResultCandidate is a scanned result row with the columns its keyset
// cursor is built from.
type nanoResultCandidate struct {
	nanoQueuePair
	UpdatedAt time.Time `db:"updated_at"`
}

// sweepCompletedNanoCommands deletes queue/result pairs whose result is
// terminal and older than retention, for commands accepted by inClass, up to
// budget. One keyset scan per terminal status, each resuming from the cursor
// in state (keyed "<tier>:<status>") and clearing it when it reaches rows
// younger than the window, so the next run laps from the oldest rows again.
func (ds *Datastore) sweepCompletedNanoCommands(ctx context.Context, state *fleet.MDMAppleCommandCleanupState, tier string, retention time.Duration, budget int, inClass func(requestType, uuid string) bool) (int, []string, bool, error) {
	// Rides idx_ncr_status_updated_at: with status fixed, InnoDB's appended
	// primary key makes the index order (updated_at, id, command_uuid), so the
	// keyset needs no sort. Classification is done on the page afterwards
	// rather than in the scan so every scan costs one page of index rows even
	// when a long run of never-swept types sits in front of the cursor.
	// updated_at is nullable in the schema but always set by its DEFAULT and
	// ON UPDATE clauses; a NULL would fall outside both comparisons.
	// Nested ORs rather than a row constructor for the same reason as the
	// inactive purge: with status ahead of it in the index, the tuple form is
	// only a filter and MySQL walks the range from the start on every scan.
	const candidatesStmt = `
		SELECT ncr.id, ncr.command_uuid, ncr.updated_at
		FROM nano_command_results ncr
		WHERE ncr.status = ?
		  AND ncr.updated_at < NOW(6) - INTERVAL ? SECOND
		  AND (ncr.updated_at > ?
		    OR (ncr.updated_at = ? AND (ncr.id > ?
		      OR (ncr.id = ? AND ncr.command_uuid > ?))))
		ORDER BY ncr.updated_at, ncr.id, ncr.command_uuid
		LIMIT ?`

	var deleted int
	var touched []string
	var capped bool
	for _, status := range nanoTerminalStatuses {
		key := tier + ":" + status
		// an unset key is the zero cursor, and a zero time.Time sorts below
		// every real TIMESTAMP, so the scan starts at the oldest row
		cursor := state.Retention[key]
		var pageFull bool
		for range nanoCleanupMaxScansPerRun {
			if deleted >= budget {
				state.Retention[key] = cursor
				return deleted, touched, true, nil
			}
			limit := min(nanoCleanupScanBatchSize, budget-deleted)
			var candidates []nanoResultCandidate
			// primary for the same reason as the inactive purge: the guard
			// probe and the delete must agree with this read
			if err := sqlx.SelectContext(ctx, ds.writer(ctx), &candidates, candidatesStmt,
				status, int(retention.Seconds()), cursor.UpdatedAt, cursor.UpdatedAt, cursor.ID, cursor.ID, cursor.CommandUUID, limit); err != nil {
				return deleted, touched, false, ctxerr.Wrap(ctx, err, "select completed nano commands")
			}
			pageFull = len(candidates) == limit
			if !pageFull {
				// reached rows younger than the window (or the end): this
				// status laps from the oldest rows next run
				delete(state.Retention, key)
			}
			if len(candidates) == 0 {
				break
			}
			last := candidates[len(candidates)-1]
			cursor = fleet.MDMAppleCommandCleanupCursor{UpdatedAt: last.UpdatedAt, ID: last.ID, CommandUUID: last.CommandUUID}
			if pageFull {
				state.Retention[key] = cursor
			}

			pairs, err := ds.classifyNanoCandidates(ctx, candidates, inClass)
			if err != nil {
				return deleted, touched, false, err
			}
			if len(pairs) == 0 {
				if !pageFull {
					break
				}
				continue
			}
			n, err := ds.deleteNanoQueuePairs(ctx, pairs)
			if err != nil {
				return deleted, touched, false, err
			}
			deleted += n
			for _, p := range pairs {
				touched = append(touched, p.CommandUUID)
			}
			if !pageFull {
				break
			}
		}
		if pageFull {
			// scan cap reached with a full last page: this status has more
			// candidates behind its cursor
			capped = true
		}
	}
	return deleted, touched, capped, nil
}

// classifyNanoCandidates keeps the candidates whose command is accepted by
// inClass and not referenced by any feature.
func (ds *Datastore) classifyNanoCandidates(ctx context.Context, candidates []nanoResultCandidate, inClass func(requestType, uuid string) bool) ([]nanoQueuePair, error) {
	uuids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		uuids = append(uuids, c.CommandUUID)
	}
	uuids = uniqueStrings(uuids)

	stmt, args, err := sqlx.In(`SELECT command_uuid, request_type FROM nano_commands WHERE command_uuid IN (?)`, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build nano commands request type query")
	}
	var cmds []struct {
		CommandUUID string `db:"command_uuid"`
		RequestType string `db:"request_type"`
	}
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &cmds, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select nano commands request types")
	}
	eligible := make([]string, 0, len(cmds))
	for _, c := range cmds {
		if inClass(c.RequestType, c.CommandUUID) {
			eligible = append(eligible, c.CommandUUID)
		}
	}
	if len(eligible) == 0 {
		return nil, nil
	}
	safe, err := ds.unreferencedNanoCommands(ctx, eligible)
	if err != nil {
		return nil, err
	}
	var pairs []nanoQueuePair
	for _, c := range candidates {
		if _, ok := safe[c.CommandUUID]; ok {
			pairs = append(pairs, c.nanoQueuePair)
		}
	}
	return pairs, nil
}

// purgeInactiveNanoCommands deletes queue rows deactivated more than
// retention ago, with their results, skipping request types whose inactive
// rows are still read back. It returns the pairs deleted, the command UUIDs
// they belonged to, and whether it stopped early (budget or scan cap) with
// candidates left.
func (ds *Datastore) purgeInactiveNanoCommands(ctx context.Context, retention time.Duration, budget int) (int, []string, bool, error) {
	// Ordered along idx_neq_filter (active, priority, created_at) plus the
	// appended primary key, so each scan resumes after the last row seen
	// without a sort. Without the keyset, a window full of guard-pinned rows
	// would be re-read every scan and starve the eligible rows behind it.
	// The keyset is spelled out as nested ORs: a row constructor starting at
	// the index's second column is only a filter to MySQL, which then walks
	// the whole active = 0 range up to the cursor on every scan.
	const candidatesStmt = `
		SELECT neq.id, neq.command_uuid, neq.priority, neq.created_at
		FROM nano_enrollment_queue neq
		JOIN nano_commands nc ON nc.command_uuid = neq.command_uuid
		WHERE neq.active = 0
		  AND (neq.priority > ?
		    OR (neq.priority = ? AND (neq.created_at > ?
		      OR (neq.created_at = ? AND (neq.id > ?
		        OR (neq.id = ? AND neq.command_uuid > ?))))))
		  AND neq.updated_at < NOW(6) - INTERVAL ? SECOND
		  AND nc.request_type NOT IN (?)
		ORDER BY neq.priority, neq.created_at, neq.id, neq.command_uuid
		LIMIT ?`

	var deleted int
	var touched []string
	// below every real row: priority is a signed tinyint and a zero time.Time
	// sorts under any TIMESTAMP
	cursor := nanoQueueCandidate{Priority: -128, CreatedAt: time.Time{}}
	for range nanoCleanupMaxScansPerRun {
		limit := min(nanoCleanupScanBatchSize, budget-deleted)
		stmt, args, err := sqlx.In(candidatesStmt,
			cursor.Priority, cursor.Priority, cursor.CreatedAt, cursor.CreatedAt, cursor.ID, cursor.ID, cursor.CommandUUID,
			int(retention.Seconds()), fleet.AppleMDMInactivePurgeDenylist, limit)
		if err != nil {
			return deleted, touched, false, ctxerr.Wrap(ctx, err, "build inactive nano commands query")
		}
		var candidates []nanoQueueCandidate
		// Read from the primary: the guard probe and the delete must see the
		// same references, and a replica-lag miss here would delete a pair a
		// feature just started relying on.
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &candidates, stmt, args...); err != nil {
			return deleted, touched, false, ctxerr.Wrap(ctx, err, "select inactive nano commands")
		}
		if len(candidates) == 0 {
			return deleted, touched, false, nil
		}
		cursor = candidates[len(candidates)-1]
		pageFull := len(candidates) == limit

		cmdUUIDs := make([]string, 0, len(candidates))
		for _, c := range candidates {
			cmdUUIDs = append(cmdUUIDs, c.CommandUUID)
		}
		safe, err := ds.unreferencedNanoCommands(ctx, uniqueStrings(cmdUUIDs))
		if err != nil {
			return deleted, touched, false, err
		}
		var pairs []nanoQueuePair
		for _, c := range candidates {
			if _, ok := safe[c.CommandUUID]; ok {
				pairs = append(pairs, c.nanoQueuePair)
			}
		}
		if len(pairs) == 0 {
			// every row in this window is pinned by a reference: move past it
			if !pageFull {
				return deleted, touched, false, nil
			}
			continue
		}

		n, err := ds.deleteNanoQueuePairs(ctx, pairs)
		if err != nil {
			return deleted, touched, false, err
		}
		deleted += n
		for _, p := range pairs {
			touched = append(touched, p.CommandUUID)
		}
		if deleted >= budget {
			return deleted, touched, true, nil
		}
		if !pageFull {
			return deleted, touched, false, nil
		}
	}
	// scan cap reached with a full last page: more candidates remain
	return deleted, touched, true, nil
}

// unreferencedNanoCommands returns the subset of cmdUUIDs that no feature still
// references, as a set.
func (ds *Datastore) unreferencedNanoCommands(ctx context.Context, cmdUUIDs []string) (map[string]struct{}, error) {
	stmt, args, err := sqlx.In(`SELECT c.command_uuid FROM nano_commands c WHERE c.command_uuid IN (?) AND `+nanoCommandUnreferencedFilter, cmdUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build unreferenced nano commands query")
	}
	var safe []string
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &safe, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select unreferenced nano commands")
	}
	set := make(map[string]struct{}, len(safe))
	for _, u := range safe {
		set[u] = struct{}{}
	}
	return set, nil
}

// deleteNanoQueuePairs deletes the given queue rows and their results in one
// transaction, queue first: nano serves a command whose queue row exists with
// no result, so deleting the result first would re-serve the command to the
// device. Returns the number of queue rows deleted.
func (ds *Datastore) deleteNanoQueuePairs(ctx context.Context, pairs []nanoQueuePair) (int, error) {
	placeholders := strings.TrimSuffix(strings.Repeat("(?, ?), ", len(pairs)), ", ")
	args := make([]any, 0, len(pairs)*2)
	for _, p := range pairs {
		args = append(args, p.ID, p.CommandUUID)
	}
	var deleted int64
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM nano_enrollment_queue WHERE (id, command_uuid) IN (`+placeholders+`)`, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete nano enrollment queue rows")
		}
		deleted, _ = res.RowsAffected()
		if nanoCleanupAfterQueueDeleteHook != nil {
			if err := nanoCleanupAfterQueueDeleteHook(); err != nil {
				return ctxerr.Wrap(ctx, err, "nano cleanup test hook")
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM nano_command_results WHERE (id, command_uuid) IN (`+placeholders+`)`, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete nano command results rows")
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(deleted), nil
}

// mopNanoCommands deletes, among cmdUUIDs, the nano_commands rows that no queue
// row, result row, bootstrap package or certificate renewal still references,
// up to budget. Those four are the tables a delete would cascade into or
// FK-fail on, so they are rechecked inside the DELETE itself; the soft
// references were checked by the pair guard moments earlier, and a finished
// command does not acquire new ones.
func (ds *Datastore) mopNanoCommands(ctx context.Context, cmdUUIDs []string, budget int) (int, bool, error) {
	const stmt = `
		DELETE FROM nano_commands
		WHERE command_uuid IN (?)
		  AND NOT EXISTS (SELECT 1 FROM nano_enrollment_queue neq WHERE neq.command_uuid = nano_commands.command_uuid)
		  AND NOT EXISTS (SELECT 1 FROM nano_command_results ncr WHERE ncr.command_uuid = nano_commands.command_uuid)
		  AND NOT EXISTS (SELECT 1 FROM host_mdm_apple_bootstrap_packages hmabp WHERE hmabp.command_uuid = nano_commands.command_uuid)
		  AND NOT EXISTS (SELECT 1 FROM nano_cert_auth_associations ncaa WHERE ncaa.renew_command_uuid = nano_commands.command_uuid)
		LIMIT ?`

	var deleted int
	for start := 0; start < len(cmdUUIDs); start += nanoCleanupScanBatchSize {
		remaining := budget - deleted
		if remaining <= 0 {
			return deleted, true, nil
		}
		chunk := cmdUUIDs[start:min(start+nanoCleanupScanBatchSize, len(cmdUUIDs))]
		q, args, err := sqlx.In(stmt, chunk, remaining)
		if err != nil {
			return deleted, false, ctxerr.Wrap(ctx, err, "build nano commands mop query")
		}
		res, err := ds.writer(ctx).ExecContext(ctx, q, args...)
		if err != nil {
			return deleted, false, ctxerr.Wrap(ctx, err, "delete unreferenced nano commands")
		}
		n, _ := res.RowsAffected()
		deleted += int(n)
		if int(n) == remaining {
			// the LIMIT bit: assume more of this chunk qualified
			return deleted, true, nil
		}
	}
	return deleted, false, nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

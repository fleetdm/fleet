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

// Rows per candidate scan; a var so tests can shrink it.
var nanoCleanupScanBatchSize = 1000

// Candidate scans per sweep per run; a var so tests can force a mid-scan stop.
var nanoCleanupMaxScansPerRun = 50

// Test hook run between the queue and result deletes to prove they roll back together.
var nanoCleanupAfterQueueDeleteHook func() error

// True for a nano_commands row c that no feature still references; every probe is an indexed lookup.
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

// One nano_enrollment_queue row and its paired nano_command_results row.
type nanoQueuePair struct {
	ID          string `db:"id"`
	CommandUUID string `db:"command_uuid"`
}

// A scanned queue row with its keyset cursor columns.
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

	// one row budget across the sweeps in order; a sweep's own scan cap does not stop the others
	budget := opts.MaxRowDeletions
	var touched []string
	if opts.ShortRetention > 0 && budget > 0 {
		deleted, cmdUUIDs, exhausted, err := ds.purgeInactiveNanoCommands(ctx, opts.ShortRetention, budget)
		if err != nil {
			return state, stats, err
		}
		stats.InactivePairsDeleted = deleted
		stats.RowBudgetExhausted = stats.RowBudgetExhausted || exhausted
		budget -= deleted
		touched = append(touched, cmdUUIDs...)
	}

	shortTierAgesOutWithStandardTier := false
	if opts.ShortRetention > 0 {
		if budget > 0 {
			deleted, cmdUUIDs, exhausted, err := ds.sweepCompletedNanoCommands(ctx, state, "short", opts.ShortRetention, budget,
				func(rt, uuid string) bool {
					return matchesAppleMDMRetentionClass(fleet.AppleMDMShortRetentionClasses, rt, uuid)
				})
			if err != nil {
				return state, stats, err
			}
			stats.ShortPairsDeleted = deleted
			stats.RowBudgetExhausted = stats.RowBudgetExhausted || exhausted
			budget -= deleted
			touched = append(touched, cmdUUIDs...)
		}
	} else {
		// short tier off: its classes age out with the standard tier instead of being kept forever
		shortTierAgesOutWithStandardTier = true
	}
	if opts.StandardRetention > 0 && budget > 0 {
		deleted, cmdUUIDs, exhausted, err := ds.sweepCompletedNanoCommands(ctx, state, "standard", opts.StandardRetention, budget,
			func(rt, uuid string) bool {
				return slices.Contains(fleet.AppleMDMStandardRetentionRequestTypes, rt) ||
					(shortTierAgesOutWithStandardTier && matchesAppleMDMRetentionClass(fleet.AppleMDMShortRetentionClasses, rt, uuid))
			})
		if err != nil {
			return state, stats, err
		}
		stats.StandardPairsDeleted = deleted
		stats.RowBudgetExhausted = stats.RowBudgetExhausted || exhausted
		touched = append(touched, cmdUUIDs...)
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

// matchesAppleMDMRetentionClass reports whether a command belongs to one of classes.
func matchesAppleMDMRetentionClass(classes []fleet.AppleMDMCommandRetentionClass, rt, uuid string) bool {
	for _, c := range classes {
		if c.RequestType == rt && strings.HasPrefix(uuid, c.UUIDPrefix) {
			return true
		}
	}
	return false
}

// Statuses a device will not answer again; NotNow is still outstanding and gets re-served.
var nanoTerminalStatuses = []string{fleet.MDMAppleStatusAcknowledged, fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError}

// A scanned result row with its keyset cursor columns.
type nanoResultCandidate struct {
	nanoQueuePair
	UpdatedAt time.Time `db:"updated_at"`
}

// sweepCompletedNanoCommands deletes terminal pairs older than retention for commands accepted by
// inClass, one keyset scan per status resuming from state and lapping once it reaches young rows.
func (ds *Datastore) sweepCompletedNanoCommands(ctx context.Context, state *fleet.MDMAppleCommandCleanupState, tier string, retention time.Duration, budget int, inClass func(requestType, uuid string) bool) (int, []string, bool, error) {
	// keyset over idx_ncr_status_updated_at as nested ORs so MySQL seeks to the cursor; classification
	// happens after the scan so every scan costs exactly one index page
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
		// an unset key is the zero cursor, which sorts below every row
		cursor := state.Retention[key]
		var pageFull bool
		for range nanoCleanupMaxScansPerRun {
			if deleted >= budget {
				state.Retention[key] = cursor
				return deleted, touched, true, nil
			}
			limit := min(nanoCleanupScanBatchSize, budget-deleted)
			var candidates []nanoResultCandidate
			// primary: the guard probe and the delete must agree with this read
			if err := sqlx.SelectContext(ctx, ds.writer(ctx), &candidates, candidatesStmt,
				status, int(retention.Seconds()), cursor.UpdatedAt, cursor.UpdatedAt, cursor.ID, cursor.ID, cursor.CommandUUID, limit); err != nil {
				return deleted, touched, false, ctxerr.Wrap(ctx, err, "select completed nano commands")
			}
			pageFull = len(candidates) == limit
			if !pageFull {
				// short page: reached rows younger than the window, lap from the oldest next run
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
			// scan cap with a full page: more candidates remain
			capped = true
		}
	}
	return deleted, touched, capped, nil
}

// classifyNanoCandidates keeps the candidates whose command inClass accepts and nothing references.
func (ds *Datastore) classifyNanoCandidates(ctx context.Context, candidates []nanoResultCandidate, inClass func(requestType, uuid string) bool) ([]nanoQueuePair, error) {
	cmdUUIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		cmdUUIDs = append(cmdUUIDs, c.CommandUUID)
	}
	cmdUUIDs = uniqueStrings(cmdUUIDs)

	stmt, args, err := sqlx.In(`SELECT command_uuid, request_type FROM nano_commands WHERE command_uuid IN (?)`, cmdUUIDs)
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

// purgeInactiveNanoCommands deletes pairs deactivated more than retention ago, skipping denylisted
// types; returns pairs deleted, their command UUIDs, and whether it stopped early.
func (ds *Datastore) purgeInactiveNanoCommands(ctx context.Context, retention time.Duration, budget int) (int, []string, bool, error) {
	// keyset over idx_neq_filter (active, priority, created_at) + PK as nested ORs so MySQL seeks to the cursor
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
	// below every row: the signed tinyint minimum and a zero time.Time
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
		// primary: a replica-lag miss here would delete a pair a feature just started using
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

// unreferencedNanoCommands returns the subset of cmdUUIDs nothing references, as a set.
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

// deleteNanoQueuePairs deletes the pairs in one transaction, queue first: a queue row without a result
// is re-served to the device.
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

// mopNanoCommands deletes the cmdUUIDs that no queue row, result, bootstrap package or cert renewal
// references, rechecked inside the DELETE so a cascade can never take a live row.
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

package mysql

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const policyAutomationBatchSize = 1000

// RecordPolicyTransitions is the single writer of host_policy_runs rows.
//
// Per-policy semantics (one row per policy_id/host_id pair is maintained):
//   - New policy, fails first time: insert (old_status=NULL, new_status=false, consecutive_failures=1).
//   - New policy, passes first time: insert (old_status=NULL, new_status=true, consecutive_failures=0).
//   - Was passing, now failing: update (old_status=true, new_status=false, consecutive_failures=1).
//   - Was failing, now passing: update (old_status=false, new_status=true, consecutive_failures=0).
//   - Was passing, still passing: no-op.
//   - Was failing, still failing: bump consecutive_failures by 1.
func (ds *Datastore) RecordPolicyTransitions(
	ctx context.Context,
	hostID uint,
	policyResults map[uint]*bool,
	newFailing, newPassing []uint,
) (map[uint]uint, error) {
	if len(policyResults) == 0 {
		return nil, nil
	}

	type pendingRow struct {
		policyID            uint
		newStatus           bool
		consecutiveFailures uint
	}
	pending := make([]pendingRow, 0, len(policyResults))
	allCurrentlyPassing := true
	for pid, res := range policyResults {
		if res == nil {
			continue
		}
		if !*res {
			allCurrentlyPassing = false
		}
		row := pendingRow{policyID: pid, newStatus: *res}
		if !*res {
			row.consecutiveFailures = 1
		}
		pending = append(pending, row)
	}
	if len(pending) == 0 {
		return nil, nil
	}

	// Fast path: nothing transitioned (no first-time-failing, no pass→fail,
	// no fail→pass) and every current result is passing. Every row would
	// either be a still-passing no-op or a first-time-passing insert;
	if len(newFailing) == 0 && len(newPassing) == 0 && allCurrentlyPassing {
		return nil, nil
	}

	// No transaction wrapper: each ODKU is a single atomic statement, and
	// when newFailing>0 the follow-up lookup runs on the primary, which
	// always sees the just-committed rows. Wrapping the hot path in a tx
	// holds a connection longer and serializes BEGIN/COMMIT round-trips —
	// noticeable overhead on the per-check-in path.
	writer := ds.writer(ctx)
	for chunkStart := 0; chunkStart < len(pending); chunkStart += policyAutomationBatchSize {
		chunkEnd := min(chunkStart+policyAutomationBatchSize, len(pending))
		chunk := pending[chunkStart:chunkEnd]

		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*4)
		for _, r := range chunk {
			placeholders = append(placeholders, "(?, ?, NULL, ?, ?)")
			args = append(args, r.policyID, hostID, r.newStatus, r.consecutiveFailures)
		}
		query := `INSERT INTO host_policy_runs (policy_id, host_id, old_status, new_status, consecutive_failures)
		           VALUES ` + strings.Join(placeholders, ",") + `
		           ON DUPLICATE KEY UPDATE
		             old_status = CASE
		                 WHEN new_status = VALUES(new_status) THEN old_status
		                 ELSE new_status
		             END,
		             consecutive_failures = CASE
		                 WHEN new_status = 0 AND VALUES(new_status) = 0 THEN consecutive_failures + 1
		                 ELSE VALUES(consecutive_failures)
		             END,
		             new_status = VALUES(new_status)`
		if _, err := writer.ExecContext(ctx, query, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "upsert host_policy_runs")
		}
	}

	failingIDs := make(map[uint]uint, len(newFailing))
	if len(newFailing) > 0 {
		refs, err := lookupFailingPolicyRunRefs(ctx, writer, newFailing, []uint{hostID})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select failing host_policy_runs ids")
		}
		for _, r := range refs {
			failingIDs[r.PolicyID] = r.RunID
		}
	}
	return failingIDs, nil
}

// lookupFailingPolicyRunRefs returns the failing host_policy_runs row
// (new_status=false) for every (policy_id, host_id) pair in the cross-product
// policyIDs × hostIDs. Pairs without a matching row are absent from the result.
//
// Both sides are chunked at policyAutomationBatchSize so the per-statement
// placeholder count stays bounded at ~2×policyAutomationBatchSize regardless
// of how large either input grows. Total query count is
// ceil(len(policyIDs)/B) × ceil(len(hostIDs)/B); production callers pass a
// singleton on one side so the corresponding loop iterates once.
func lookupFailingPolicyRunRefs(
	ctx context.Context,
	exec sqlx.QueryerContext,
	policyIDs, hostIDs []uint,
) ([]fleet.PolicyRunRef, error) {
	if len(policyIDs) == 0 || len(hostIDs) == 0 {
		return nil, nil
	}

	const stmt = `SELECT id, policy_id, host_id FROM host_policy_runs
		WHERE policy_id IN (?) AND host_id IN (?) AND new_status = false`

	var out []fleet.PolicyRunRef
	for pStart := 0; pStart < len(policyIDs); pStart += policyAutomationBatchSize {
		pChunk := policyIDs[pStart:min(pStart+policyAutomationBatchSize, len(policyIDs))]

		for hStart := 0; hStart < len(hostIDs); hStart += policyAutomationBatchSize {
			hChunk := hostIDs[hStart:min(hStart+policyAutomationBatchSize, len(hostIDs))]

			query, args, err := sqlx.In(stmt, pChunk, hChunk)
			if err != nil {
				return nil, err
			}
			var batch []fleet.PolicyRunRef
			if err := sqlx.SelectContext(ctx, exec, &batch, query, args...); err != nil {
				return nil, err
			}
			out = append(out, batch...)
		}
	}
	return out, nil
}

func (ds *Datastore) GetFailingPolicyRuns(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
	// Read-only lookup called from every async dispatch surface (webhook, Jira,
	// Zendesk, calendar, conditional access). Hits the read replica so the
	// dispatch path doesn't compete with osquery check-ins for primary
	// connections; the small staleness window between RecordPolicyTransitions
	// and dispatch is acceptable — pairs that haven't replicated yet just
	// don't get a recording, which the spec models as a best-effort contract.
	out, err := lookupFailingPolicyRunRefs(ctx, ds.reader(ctx), policyIDs, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query failing host_policy_run ids")
	}
	return out, nil
}

func (ds *Datastore) CreatePolicyAutomationExecutions(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
	if len(executions) == 0 {
		return uuid.Nil, nil
	}

	batchID := uuid.New()
	if err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return createPolicyAutomationExecutionsTx(ctx, tx, batchID, typ, executions)
	}); err != nil {
		return uuid.Nil, err
	}
	return batchID, nil
}

func createPolicyAutomationExecutionsTx(ctx context.Context, tx sqlx.ExtContext, batchID uuid.UUID, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) error {
	batchBytes := batchID[:]

	// Link each policy_run to this batch via the join table. policy_id is
	// not stored here — it is reachable through host_policy_runs.policy_id.
	for chunkStart := 0; chunkStart < len(executions); chunkStart += policyAutomationBatchSize {
		chunkEnd := min(chunkStart+policyAutomationBatchSize, len(executions))
		chunk := executions[chunkStart:chunkEnd]

		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*3)
		for _, e := range chunk {
			placeholders = append(placeholders, "(?, ?, ?)")
			args = append(args, e.RunID, typ, batchBytes)
		}
		query := `INSERT INTO host_policy_runs_to_policy_automation_executions (policy_run_id, automation_type, batch_id) VALUES ` + strings.Join(placeholders, ",")

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host_policy_runs_to_policy_automation_executions")
		}
	}

	// Single batch-status row; defaults to 'pending' until the
	// orchestrator finalizes via UpdatePolicyAutomationExecutions.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO policy_automation_executions (batch_id) VALUES (?)`, batchBytes,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "insert policy_automation_executions")
	}
	return nil
}

func (ds *Datastore) UpdatePolicyAutomationExecutions(
	ctx context.Context,
	batchID uuid.UUID,
	outcomeErr error,
) error {
	if batchID == uuid.Nil {
		return nil
	}

	// nil outcomeErr → Success
	// non-nil outcomeErr → Failure with err.Error() as the message.
	status := fleet.PolicyAutomationStatusSuccess
	var errPtr *string
	if outcomeErr != nil {
		status = fleet.PolicyAutomationStatusFailure
		if msg := outcomeErr.Error(); msg != "" {
			errPtr = &msg
		}
	}

	// Worker retry loops (jira/zendesk: up to 5 attempts) call this on every
	// attempt. The monotonic state machine encoded in the WHERE clause:
	//
	//   pending  → success | failure   (first attempt's outcome)
	//   failure  → success             (a later retry succeeded — upgrade)
	//   failure  → failure             SKIP — no UI value rewriting same status
	//   success  → anything            SKIP — terminal state, locked in
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE policy_automation_executions
		 SET status = ?, error_message = ?
		 WHERE batch_id = ?
		   AND (status = 'pending' OR (status = 'failure' AND ? = 'success'))`,
		status, errPtr, batchID[:], status,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update policy_automation_executions status by batch")
	}
	return nil
}

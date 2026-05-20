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

// RecordPolicyTransitions is the single writer of policy_runs rows.
//
// Per-policy semantics (one row per policy_id/host_id pair is maintained):
//   - New policy, fails first time: insert (old_status=NULL, new_status=false, consecutive_failures=1).
//   - New policy, passes first time: insert (old_status=NULL, new_status=true, consecutive_failures=0).
//   - Was passing, now failing: update (old_status=true, new_status=false, consecutive_failures=1).
//   - Was failing, now passing: update (old_status=false, new_status=true, consecutive_failures=0).
//   - Was passing, still passing: no-op.
//   - Was failing, still failing: bump consecutive_failures by 1.
//
// All six cases collapse into a single INSERT … ON DUPLICATE KEY UPDATE per
// chunk. The CASE expressions read old column values because MySQL evaluates
// ODKU SET clauses left-to-right and only the new_status assignment at the
// end commits the new value. For the "still passing" case every SET clause
// resolves to the existing value, so InnoDB detects no change and skips both
// the row write and the ON UPDATE CURRENT_TIMESTAMP trigger.
func (ds *Datastore) RecordPolicyTransitions(
	ctx context.Context,
	hostID uint,
	policyResults map[uint]*bool,
	newFailing []uint,
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
	for pid, res := range policyResults {
		if res == nil {
			continue
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
		query := `INSERT INTO policy_runs (policy_id, host_id, old_status, new_status, consecutive_failures)
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
			return nil, ctxerr.Wrap(ctx, err, "upsert policy_runs")
		}
	}

	failingIDs := make(map[uint]uint, len(newFailing))
	if len(newFailing) > 0 {
		refs, err := lookupFailingPolicyRunRefs(ctx, writer, newFailing, []uint{hostID})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select failing policy_runs ids")
		}
		for _, r := range refs {
			failingIDs[r.PolicyID] = r.RunID
		}
	}
	return failingIDs, nil
}

// lookupFailingPolicyRunRefs returns the failing policy_runs row (new_status=false)
// for every (policy_id, host_id) pair in the cross-product policyIDs × hostIDs.
// Callers that want a 1:N shape (e.g. one policy × N hosts, the webhook batch
// pattern) should pass a singleton for the "1" side; pairs without a matching
// row are simply absent from the result.
//
// The query is chunked on the larger of the two sides — the smaller side is
// inlined verbatim into every per-chunk statement, so the per-statement
// placeholder count is bounded by policyAutomationBatchSize + len(smaller side).
// If both sides exceed policyAutomationBatchSize this could approach 2×batch
// placeholders per statement, which is still well under MySQL's
// max_prepared_stmt_count default.
func lookupFailingPolicyRunRefs(
	ctx context.Context,
	exec sqlx.ExtContext,
	policyIDs, hostIDs []uint,
) ([]fleet.PolicyRunRef, error) {
	if len(policyIDs) == 0 || len(hostIDs) == 0 {
		return nil, nil
	}

	var chunkSide, fixedSide []uint
	var chunkCol, fixedCol string
	if len(policyIDs) >= len(hostIDs) {
		chunkSide, chunkCol = policyIDs, "policy_id"
		fixedSide, fixedCol = hostIDs, "host_id"
	} else {
		chunkSide, chunkCol = hostIDs, "host_id"
		fixedSide, fixedCol = policyIDs, "policy_id"
	}

	fixedPlaceholders := strings.Repeat("?,", len(fixedSide))
	fixedPlaceholders = fixedPlaceholders[:len(fixedPlaceholders)-1]

	var out []fleet.PolicyRunRef
	for chunkStart := 0; chunkStart < len(chunkSide); chunkStart += policyAutomationBatchSize {
		chunkEnd := min(chunkStart+policyAutomationBatchSize, len(chunkSide))
		chunk := chunkSide[chunkStart:chunkEnd]

		chunkPlaceholders := strings.Repeat("?,", len(chunk))
		chunkPlaceholders = chunkPlaceholders[:len(chunkPlaceholders)-1]

		args := make([]any, 0, len(chunk)+len(fixedSide))
		for _, v := range chunk {
			args = append(args, v)
		}
		for _, v := range fixedSide {
			args = append(args, v)
		}

		query := `SELECT id, policy_id, host_id FROM policy_runs WHERE ` +
			chunkCol + ` IN (` + chunkPlaceholders + `) AND ` +
			fixedCol + ` IN (` + fixedPlaceholders + `) AND new_status = false`

		rows, err := exec.QueryxContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		err = func() error {
			defer rows.Close()
			for rows.Next() {
				var r fleet.PolicyRunRef
				if err := rows.Scan(&r.RunID, &r.PolicyID, &r.HostID); err != nil {
					return err
				}
				out = append(out, r)
			}
			return rows.Err()
		}()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (ds *Datastore) GetFailingPolicyRuns(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
	out, err := lookupFailingPolicyRunRefs(ctx, ds.writer(ctx), policyIDs, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query failing policy_run ids")
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
	// not stored here — it is reachable through policy_runs.policy_id.
	for chunkStart := 0; chunkStart < len(executions); chunkStart += policyAutomationBatchSize {
		chunkEnd := min(chunkStart+policyAutomationBatchSize, len(executions))
		chunk := executions[chunkStart:chunkEnd]

		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*3)
		for _, e := range chunk {
			placeholders = append(placeholders, "(?, ?, ?)")
			args = append(args, e.RunID, typ, batchBytes)
		}
		query := `INSERT INTO policy_runs_to_policy_automation_executions (policy_run_id, automation_type, batch_id) VALUES ` + strings.Join(placeholders, ",")

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert policy_runs_to_policy_automation_executions")
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

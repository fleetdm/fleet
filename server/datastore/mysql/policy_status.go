package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ClearPolicyRuns(ctx context.Context, policyID uint) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return clearPolicyRunsTx(ctx, tx, policyID)
	})
}

func clearPolicyRunsTx(ctx context.Context, tx sqlx.ExtContext, policyID uint) error {
	// Confirm at least one run exists; return not-found if there is nothing to reset.
	var count int
	if err := sqlx.GetContext(ctx, tx, &count,
		`SELECT COUNT(*) FROM host_policy_runs WHERE policy_id = ?`, policyID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "count host_policy_runs")
	}
	if count == 0 {
		return ctxerr.Wrap(ctx, notFound("HostPolicyRuns").WithID(policyID))
	}

	// Collect the batch_ids that will be orphaned after we delete host_policy_runs.
	// The join table will cascade-delete, but policy_automation_executions has
	// no FK back to the join table and must be cleaned up manually.
	var batchIDs [][]byte
	if err := sqlx.SelectContext(ctx, tx, &batchIDs, `
		SELECT DISTINCT j.batch_id
		FROM host_policy_runs_to_policy_automation_executions j
		JOIN host_policy_runs r ON r.id = j.policy_run_id
		WHERE r.policy_id = ?`, policyID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "collect batch_ids")
	}

	// Remove the batch execution records before deleting host_policy_runs so the
	// join table rows are still present as references during the select above.
	if len(batchIDs) > 0 {
		query, args, err := sqlx.In(
			`DELETE FROM policy_automation_executions WHERE batch_id IN (?)`, batchIDs,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete policy_automation_executions query")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete policy_automation_executions")
		}
	}

	// Delete host_policy_runs. Two FK cascades fire automatically:
	//   - ON DELETE CASCADE clears host_policy_runs_to_policy_automation_executions.
	//   - ON DELETE SET NULL clears policy_run_id on host_script_results,
	//     host_software_installs, host_vpp_software_installs,
	//     script_upcoming_activities, software_install_upcoming_activities,
	//     and vpp_app_upcoming_activities — disassociating script, software, and
	//     VPP install rows from the run history we just removed.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM host_policy_runs WHERE policy_id = ?`, policyID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_policy_runs")
	}

	// NULL out the direct policy_id column on activity/result rows that were
	// stamped with the old automation system (no policy_run_id linkage).
	nullUpdates := []struct {
		query string
		label string
	}{
		{`UPDATE host_script_results SET policy_id = NULL WHERE policy_id = ?`, "host_script_results"},
		{`UPDATE host_software_installs SET policy_id = NULL WHERE policy_id = ?`, "host_software_installs"},
		{`UPDATE host_vpp_software_installs SET policy_id = NULL WHERE policy_id = ?`, "host_vpp_software_installs"},
		{`UPDATE script_upcoming_activities SET policy_id = NULL WHERE policy_id = ?`, "script_upcoming_activities"},
		{`UPDATE software_install_upcoming_activities SET policy_id = NULL WHERE policy_id = ?`, "software_install_upcoming_activities"},
		{`UPDATE vpp_app_upcoming_activities SET policy_id = NULL WHERE policy_id = ?`, "vpp_app_upcoming_activities"},
	}
	for _, u := range nullUpdates {
		if _, err := tx.ExecContext(ctx, u.query, policyID); err != nil {
			return ctxerr.Wrap(ctx, err, "null policy_id in "+u.label)
		}
	}

	return nil
}

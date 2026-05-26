package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestClearPolicyRuns(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := t.Context()

	user := test.NewUser(t, ds, "Test", "testcpr@example.com", true)
	policy := newTestPolicy(t, ds, user, "clear_pol", "darwin", nil)

	mkHost := func(n int) *fleet.Host {
		name := fmt.Sprintf("cpr-host-%d", n)
		return test.NewHost(t, ds, name, fmt.Sprintf("10.9.0.%d", n), "key-"+name, "uuid-"+name, time.Now())
	}

	t.Run("no runs returns not found", func(t *testing.T) {
		err := ds.ClearPolicyRuns(ctx, policy.ID)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("clears host_policy_runs and policy_automation_executions", func(t *testing.T) {
		h1 := mkHost(1)
		h2 := mkHost(2)

		// Seed two failing host_policy_runs.
		runIDs1, err := ds.RecordPolicyTransitions(ctx, h1.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)
		runIDs2, err := ds.RecordPolicyTransitions(ctx, h2.ID, map[uint]*bool{policy.ID: new(false)}, []uint{policy.ID}, nil)
		require.NoError(t, err)

		run1 := fleet.PolicyRunRef{PolicyID: policy.ID, HostID: h1.ID, RunID: runIDs1[policy.ID]}
		run2 := fleet.PolicyRunRef{PolicyID: policy.ID, HostID: h2.ID, RunID: runIDs2[policy.ID]}

		// Record one automation batch covering both runs.
		batchID, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, []fleet.PolicyRunRef{run1, run2})
		require.NoError(t, err)

		// Confirm rows exist before the call.
		var runCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &runCount, `SELECT COUNT(*) FROM host_policy_runs WHERE policy_id = ?`, policy.ID))
		require.Equal(t, 2, runCount)

		var execCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &execCount, `SELECT COUNT(*) FROM policy_automation_executions WHERE batch_id = ?`, batchID[:]))
		require.Equal(t, 1, execCount)

		// Clear.
		require.NoError(t, ds.ClearPolicyRuns(ctx, policy.ID))

		// host_policy_runs must be gone.
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &runCount, `SELECT COUNT(*) FROM host_policy_runs WHERE policy_id = ?`, policy.ID))
		require.Equal(t, 0, runCount)

		// policy_automation_executions must be gone.
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &execCount, `SELECT COUNT(*) FROM policy_automation_executions WHERE batch_id = ?`, batchID[:]))
		require.Equal(t, 0, execCount)

		// join table rows must also be gone (cascade from host_policy_runs delete).
		var joinCount int
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &joinCount, `SELECT COUNT(*) FROM host_policy_runs_to_policy_automation_executions WHERE policy_run_id IN (?, ?)`, run1.RunID, run2.RunID))
		require.Equal(t, 0, joinCount)

		// Second call with no remaining runs must return not-found.
		err = ds.ClearPolicyRuns(ctx, policy.ID)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("disassociates VPP installs from host_policy_runs and policy_id", func(t *testing.T) {
		pVPP := newTestPolicy(t, ds, user, "clear_pol_vpp", "darwin", nil)
		// Control policy whose VPP rows must NOT be touched by ClearPolicyRuns(pVPP.ID).
		pControl := newTestPolicy(t, ds, user, "clear_pol_vpp_control", "darwin", nil)
		h := mkHost(4)

		// Seed a host_policy_runs row for pVPP (required so ClearPolicyRuns
		// doesn't return not-found) and capture the run id for the FK stamp.
		runIDs, err := ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{pVPP.ID: new(false)}, []uint{pVPP.ID}, nil)
		require.NoError(t, err)
		runID := runIDs[pVPP.ID]
		require.NotZero(t, runID)

		// vpp_apps row backs the (adam_id, platform) FK on host_vpp_software_installs.
		const adamID = "111222333"
		const vppPlatform = "darwin"
		_, err = ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO vpp_apps (adam_id, platform, name) VALUES (?, ?, 'ClearTestApp')`,
			adamID, vppPlatform)
		require.NoError(t, err)

		// Target row: stamped with both policy_id and policy_run_id for pVPP.
		_, err = ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_vpp_software_installs
				(host_id, adam_id, platform, command_uuid, policy_id, policy_run_id)
			VALUES (?, ?, ?, ?, ?, ?)`,
			h.ID, adamID, vppPlatform, "cmd-clear-vpp", pVPP.ID, runID)
		require.NoError(t, err)
		// Control row: policy_id points to pControl; must remain stamped.
		_, err = ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_vpp_software_installs
				(host_id, adam_id, platform, command_uuid, policy_id)
			VALUES (?, ?, ?, ?, ?)`,
			h.ID, adamID, vppPlatform, "cmd-clear-vpp-control", pControl.ID)
		require.NoError(t, err)

		require.NoError(t, ds.ClearPolicyRuns(ctx, pVPP.ID))

		// Target row: policy_id NULLed by the explicit UPDATE, policy_run_id
		// NULLed via the host_policy_runs delete cascade.
		var targetPolicyID, targetPolicyRunID *uint
		require.NoError(t, ds.writer(ctx).QueryRowxContext(ctx,
			`SELECT policy_id, policy_run_id FROM host_vpp_software_installs WHERE command_uuid = ?`,
			"cmd-clear-vpp").Scan(&targetPolicyID, &targetPolicyRunID))
		require.Nil(t, targetPolicyID, "policy_id must be NULLed")
		require.Nil(t, targetPolicyRunID, "policy_run_id must be SET NULL by FK cascade")

		// Control row untouched.
		var ctrlPolicyID *uint
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &ctrlPolicyID,
			`SELECT policy_id FROM host_vpp_software_installs WHERE command_uuid = ?`,
			"cmd-clear-vpp-control"))
		require.NotNil(t, ctrlPolicyID)
		require.Equal(t, pControl.ID, *ctrlPolicyID)
	})

	t.Run("nulls direct policy_id column in host_script_results", func(t *testing.T) {
		p2 := newTestPolicy(t, ds, user, "clear_pol_hsr", "darwin", nil)
		// Control policy whose rows must NOT be touched by ClearPolicyRuns(p2.ID).
		pControl := newTestPolicy(t, ds, user, "clear_pol_hsr_control", "darwin", nil)
		h := mkHost(3)

		// Seed a host_policy_runs row for p2 (required so ClearPolicyRuns doesn't return not-found).
		_, err := ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{p2.ID: new(false)}, []uint{p2.ID}, nil)
		require.NoError(t, err)

		// Insert a host_script_results row for p2 (target) and one for pControl (must survive).
		execID := "exec-clear-test-001"
		controlExecID := "exec-clear-test-control"
		_, err = ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_script_results (host_id, execution_id, output, runtime, policy_id, attempt_number)
			 VALUES (?, ?, '', 0, ?, 1)`, h.ID, execID, p2.ID)
		require.NoError(t, err)
		_, err = ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_script_results (host_id, execution_id, output, runtime, policy_id, attempt_number)
			 VALUES (?, ?, '', 0, ?, 1)`, h.ID, controlExecID, pControl.ID)
		require.NoError(t, err)

		require.NoError(t, ds.ClearPolicyRuns(ctx, p2.ID))

		// The policy_id column on the target row must be NULL after the reset.
		var policyID *uint
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &policyID,
			`SELECT policy_id FROM host_script_results WHERE execution_id = ?`, execID))
		require.Nil(t, policyID)

		// The control row must be untouched.
		var controlPolicyID *uint
		require.NoError(t, ds.writer(ctx).GetContext(ctx, &controlPolicyID,
			`SELECT policy_id FROM host_script_results WHERE execution_id = ?`, controlExecID))
		require.NotNil(t, controlPolicyID)
		require.Equal(t, pControl.ID, *controlPolicyID)
	})
}

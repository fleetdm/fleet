package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetPolicyStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	p1 := newTestPolicy(t, ds, user, "p1", "darwin", nil)

	h1 := test.NewHost(t, ds, "host1", "10.0.0.1", "key1", "uuid1", time.Now())
	h2 := test.NewHost(t, ds, "host2", "10.0.0.2", "key2", "uuid2", time.Now())
	h3 := test.NewHost(t, ds, "host3", "10.0.0.3", "key3", "uuid3", time.Now())

	err := ds.AsyncBatchInsertPolicyMembership(ctx, []fleet.PolicyMembershipResult{
		{HostID: h1.ID, PolicyID: p1.ID, Passes: new(true)},
		{HostID: h2.ID, PolicyID: p1.ID, Passes: new(false)},
		{HostID: h3.ID, PolicyID: p1.ID, Passes: new(false)},
	})
	require.NoError(t, err)

	failingRunIDs, err := ds.RecordPolicyTransitions(ctx, h2.ID, map[uint]*bool{p1.ID: new(false)}, []uint{p1.ID}, nil)
	require.NoError(t, err)
	require.Contains(t, failingRunIDs, p1.ID)
	require.NotZero(t, failingRunIDs[p1.ID])
	h2RunID := failingRunIDs[p1.ID]

	failingRunIDs, err = ds.RecordPolicyTransitions(ctx, h3.ID, map[uint]*bool{p1.ID: new(false)}, []uint{p1.ID}, nil)
	require.NoError(t, err)
	require.Contains(t, failingRunIDs, p1.ID)
	require.NotZero(t, failingRunIDs[p1.ID])
	h3RunID := failingRunIDs[p1.ID]

	batchID, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, []fleet.PolicyRunRef{
		{PolicyID: p1.ID, HostID: h2.ID, RunID: h2RunID},
	})
	require.NoError(t, err)
	err = ds.UpdatePolicyAutomationExecutions(ctx, batchID, nil)
	require.NoError(t, err)

	batchID2, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, []fleet.PolicyRunRef{
		{PolicyID: p1.ID, HostID: h2.ID, RunID: h2RunID},
	})
	require.NoError(t, err)
	err = ds.UpdatePolicyAutomationExecutions(ctx, batchID2, context.DeadlineExceeded)
	require.NoError(t, err)

	batchID3, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationJira, []fleet.PolicyRunRef{
		{PolicyID: p1.ID, HostID: h3.ID, RunID: h3RunID},
	})
	require.NoError(t, err)
	err = ds.UpdatePolicyAutomationExecutions(ctx, batchID3, nil)
	require.NoError(t, err)

	adminFilter := fleet.TeamFilter{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	}

	t.Run("basic fetch", func(t *testing.T) {
		runs, count, meta, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			ListOptions: fleet.ListOptions{PerPage: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Len(t, runs, 3)
		require.False(t, meta.HasNextResults)

		var h1Run, h2Run, h3Run *fleet.GetPolicyStatusPolicyRun
		for i, r := range runs {
			switch r.HostID {
			case h1.ID:
				h1Run = &runs[i]
			case h2.ID:
				h2Run = &runs[i]
			case h3.ID:
				h3Run = &runs[i]
			}
		}

		require.NotNil(t, h1Run)
		require.True(t, h1Run.NewStatus)
		require.Equal(t, uint(0), h1Run.ConsecutiveFailures)
		require.Empty(t, h1Run.AutomationExecutions)

		require.NotNil(t, h2Run)
		require.False(t, h2Run.NewStatus)
		require.Equal(t, uint(1), h2Run.ConsecutiveFailures)
		require.Len(t, h2Run.AutomationExecutions, 2)
		statuses := []string{h2Run.AutomationExecutions[0].Status, h2Run.AutomationExecutions[1].Status}
		require.Contains(t, statuses, "success")
		require.Contains(t, statuses, "failed")

		require.NotNil(t, h3Run)
		require.False(t, h3Run.NewStatus)
		require.Len(t, h3Run.AutomationExecutions, 1)
		require.Equal(t, "success", h3Run.AutomationExecutions[0].Status)
	})

	t.Run("pagination metadata reflects has_next_results", func(t *testing.T) {
		runs, _, meta, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			ListOptions: fleet.ListOptions{PerPage: 2},
		})
		require.NoError(t, err)
		require.Len(t, runs, 2)
		require.True(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)

		runs2, _, meta2, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			ListOptions: fleet.ListOptions{PerPage: 2, Page: 1},
		})
		require.NoError(t, err)
		require.Len(t, runs2, 1)
		require.False(t, meta2.HasNextResults)
		require.True(t, meta2.HasPreviousResults)
	})

	t.Run("filter by run_status = policy_failed", func(t *testing.T) {
		runs, count, _, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			RunStatus:   "policy_failed",
			ListOptions: fleet.ListOptions{PerPage: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Len(t, runs, 2)
	})

	t.Run("filter by run_status = automation_failed", func(t *testing.T) {
		runs, count, _, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			RunStatus:   "automation_failed",
			ListOptions: fleet.ListOptions{PerPage: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, runs, 1)
		require.Equal(t, h2.ID, runs[0].HostID)
	})

	t.Run("filter by hostname", func(t *testing.T) {
		runs, count, _, err := ds.GetPolicyStatus(ctx, p1.ID, adminFilter, fleet.GetPolicyStatusRequest{
			HostNameQuery: "host3",
			ListOptions:   fleet.ListOptions{PerPage: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, runs, 1)
		require.Equal(t, h3.ID, runs[0].HostID)
	})

	t.Run("team filter scopes hosts", func(t *testing.T) {
		// A user with no role and no teams sees nothing.
		emptyFilter := fleet.TeamFilter{User: &fleet.User{}}
		runs, count, _, err := ds.GetPolicyStatus(ctx, p1.ID, emptyFilter, fleet.GetPolicyStatusRequest{
			ListOptions: fleet.ListOptions{PerPage: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, runs)
	})
}

func TestGetPolicyStatusSkippedAutomations(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	p := newTestPolicy(t, ds, user, "p1", "darwin", nil)

	// h1 is darwin (matches policy platform); script will be incompatible (.ps1).
	// h2 is darwin (matches policy platform); installer platform mismatch (windows).
	// h3 is darwin; installer platform matches but host is excluded by label.
	// h4 is darwin; installer platform matches and host is in label scope (no synthetic row).
	h1 := test.NewHost(t, ds, "host1", "10.0.0.1", "key1", "uuid1", time.Now())
	h2 := test.NewHost(t, ds, "host2", "10.0.0.2", "key2", "uuid2", time.Now())
	h3 := test.NewHost(t, ds, "host3", "10.0.0.3", "key3", "uuid3", time.Now())
	h4 := test.NewHost(t, ds, "host4", "10.0.0.4", "key4", "uuid4", time.Now())

	err := ds.AsyncBatchInsertPolicyMembership(ctx, []fleet.PolicyMembershipResult{
		{HostID: h1.ID, PolicyID: p.ID, Passes: new(false)},
		{HostID: h2.ID, PolicyID: p.ID, Passes: new(false)},
		{HostID: h3.ID, PolicyID: p.ID, Passes: new(false)},
		{HostID: h4.ID, PolicyID: p.ID, Passes: new(false)},
	})
	require.NoError(t, err)

	for _, h := range []*fleet.Host{h1, h2, h3, h4} {
		runIDs, err := ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{p.ID: new(false)}, []uint{p.ID}, nil)
		require.NoError(t, err)
		require.NotZero(t, runIDs[p.ID])
	}

	// Configure the policy with a .ps1 script (incompatible with darwin hosts).
	script, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "windows-only.ps1",
		ScriptContents: "Write-Host hi",
	})
	require.NoError(t, err)

	// Create a windows installer (incompatible with darwin hosts on h2, h3, h4).
	var scriptContentID int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx,
			`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('test')), 'test')`)
		if err != nil {
			return err
		}
		scriptContentID, err = res.LastInsertId()
		return err
	})
	var titleIDWindows, titleIDInScope int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx,
			`INSERT INTO software_titles (name, source, extension_for) VALUES ('TestWin', 'programs', '')`)
		if err != nil {
			return err
		}
		titleIDWindows, err = res.LastInsertId()
		if err != nil {
			return err
		}
		res, err = q.ExecContext(ctx,
			`INSERT INTO software_titles (name, source, extension_for) VALUES ('TestMac', 'apps', '')`)
		if err != nil {
			return err
		}
		titleIDInScope, err = res.LastInsertId()
		return err
	})

	var installerWindowsID, installerScopedID int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		// Windows installer (used for the not_compatible case).
		res, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
				 install_script_content_id, uninstall_script_content_id, platform, package_ids, patch_query)
			VALUES (NULL, 0, ?, 'storage1', 'win.msi', 'msi', '1.0', ?, ?, 'windows', '', '')`,
			titleIDWindows, scriptContentID, scriptContentID)
		if err != nil {
			return err
		}
		installerWindowsID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		// Darwin installer (used for the not_in_target case via label exclusion of h3).
		res, err = q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
				 install_script_content_id, uninstall_script_content_id, platform, package_ids, patch_query)
			VALUES (NULL, 0, ?, 'storage2', 'mac.pkg', 'pkg', '1.0', ?, ?, 'darwin', '', '')`,
			titleIDInScope, scriptContentID, scriptContentID)
		if err != nil {
			return err
		}
		installerScopedID, err = res.LastInsertId()
		return err
	})

	t.Run("script not_compatible: .ps1 on darwin", func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE policies SET script_id = ?, software_installer_id = NULL WHERE id = ?`, script.ID, p.ID)
			return err
		})

		runs, _, _, err := ds.GetPolicyStatus(ctx, p.ID, fleet.TeamFilter{
			User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
		}, fleet.GetPolicyStatusRequest{ListOptions: fleet.ListOptions{PerPage: 10}})
		require.NoError(t, err)

		for _, r := range runs {
			if r.HostID != h1.ID {
				continue
			}
			var got *fleet.GetPolicyStatusAutomationExecution
			for i, a := range r.AutomationExecutions {
				if a.Type == "script_run" {
					got = &r.AutomationExecutions[i]
				}
			}
			require.NotNil(t, got, "expected synthetic script_run row on host with incompatible platform")
			require.Equal(t, "not_compatible", got.Status)
			require.Equal(t, "windows-only.ps1", got.Name)
		}
	})

	t.Run("software_installation not_compatible: windows installer on darwin", func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE policies SET script_id = NULL, software_installer_id = ? WHERE id = ?`, installerWindowsID, p.ID)
			return err
		})

		runs, _, _, err := ds.GetPolicyStatus(ctx, p.ID, fleet.TeamFilter{
			User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
		}, fleet.GetPolicyStatusRequest{ListOptions: fleet.ListOptions{PerPage: 10}})
		require.NoError(t, err)

		// Every failing run should have a software_installation row with not_compatible.
		for _, r := range runs {
			var got *fleet.GetPolicyStatusAutomationExecution
			for i, a := range r.AutomationExecutions {
				if a.Type == "software_installation" {
					got = &r.AutomationExecutions[i]
				}
			}
			require.NotNil(t, got, "host %d missing software_installation row", r.HostID)
			require.Equal(t, "not_compatible", got.Status)
			require.Equal(t, "TestWin", got.Name)
		}
	})

	t.Run("software_installation not_in_target: host excluded by label", func(t *testing.T) {
		// Scope the darwin installer to exclude h3. include_any over a label
		// that only h4 is a member of will achieve this.
		labelID, err := ds.NewLabel(ctx, &fleet.Label{Name: "in_scope", Query: "SELECT 1"})
		require.NoError(t, err)
		require.NoError(t, ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{labelID.ID, h4.ID}}))

		// Make h3's label_updated_at recent so the exclude-any predicate does
		// not short-circuit on "host hasn't reported labels yet."
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE hosts SET label_updated_at = NOW() WHERE id IN (?, ?, ?)`, h2.ID, h3.ID, h4.ID)
			return err
		})

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO software_installer_labels (software_installer_id, label_id, exclude, require_all) VALUES (?, ?, 0, 0)`,
				installerScopedID, labelID.ID)
			return err
		})

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE policies SET script_id = NULL, software_installer_id = ? WHERE id = ?`, installerScopedID, p.ID)
			return err
		})

		runs, _, _, err := ds.GetPolicyStatus(ctx, p.ID, fleet.TeamFilter{
			User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
		}, fleet.GetPolicyStatusRequest{ListOptions: fleet.ListOptions{PerPage: 10}})
		require.NoError(t, err)

		var seen3, seen4 bool
		for _, r := range runs {
			for _, a := range r.AutomationExecutions {
				if a.Type != "software_installation" {
					continue
				}
				switch r.HostID {
				case h3.ID:
					require.Equal(t, "not_in_target", a.Status, "h3 should be out of scope")
					require.Equal(t, "TestMac", a.Name)
					seen3 = true
				case h4.ID:
					seen4 = true
				}
			}
		}
		require.True(t, seen3, "expected synthetic not_in_target row for h3")
		require.False(t, seen4, "h4 is in scope; no synthetic row expected")
	})

	t.Run("passing run does not synthesize skipped automations", func(t *testing.T) {
		// Flip h1 to passing in both policy_membership (drives new_status in
		// the response) and host_policy_runs (closes out the failure record).
		require.NoError(t, ds.AsyncBatchInsertPolicyMembership(ctx, []fleet.PolicyMembershipResult{
			{HostID: h1.ID, PolicyID: p.ID, Passes: new(true)},
		}))
		_, err := ds.RecordPolicyTransitions(ctx, h1.ID, map[uint]*bool{p.ID: new(true)}, nil, nil)
		require.NoError(t, err)

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE policies SET script_id = ?, software_installer_id = NULL WHERE id = ?`, script.ID, p.ID)
			return err
		})

		runs, _, _, err := ds.GetPolicyStatus(ctx, p.ID, fleet.TeamFilter{
			User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
		}, fleet.GetPolicyStatusRequest{ListOptions: fleet.ListOptions{PerPage: 10}})
		require.NoError(t, err)
		for _, r := range runs {
			if r.HostID != h1.ID {
				continue
			}
			require.True(t, r.NewStatus, "h1 should be passing now")
			require.Empty(t, r.AutomationExecutions, "passing run must not have synthetic automation rows")
		}
	})
}

// TestGetPolicyStatusVPPInstalls covers the two VPP branches added to
// fetchAutomationsForPolicyRuns and the VPP-failure clause in the
// automation_failed filter. Each lifecycle state of a VPP install is seeded
// directly on host_vpp_software_installs / vpp_app_upcoming_activities /
// nano_command_results so the status derivation can be exercised end-to-end.
func TestGetPolicyStatusVPPInstalls(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()

	user := test.NewUser(t, ds, "Test", "test@example.com", true)
	p := newTestPolicy(t, ds, user, "p_vpp", "darwin", nil)

	// One host per lifecycle state we want to assert on.
	hSuccess := test.NewHost(t, ds, "vpp-success", "10.1.0.1", "k1", "u1", time.Now())
	hVerifFail := test.NewHost(t, ds, "vpp-verifail", "10.1.0.2", "k2", "u2", time.Now())
	hMDMError := test.NewHost(t, ds, "vpp-mdm-err", "10.1.0.3", "k3", "u3", time.Now())
	hQueuedResult := test.NewHost(t, ds, "vpp-queued-r", "10.1.0.4", "k4", "u4", time.Now())
	hQueuedUpcoming := test.NewHost(t, ds, "vpp-queued-u", "10.1.0.5", "k5", "u5", time.Now())
	hDedupe := test.NewHost(t, ds, "vpp-dedupe", "10.1.0.6", "k6", "u6", time.Now())

	allHosts := []*fleet.Host{hSuccess, hVerifFail, hMDMError, hQueuedResult, hQueuedUpcoming, hDedupe}
	var memberships []fleet.PolicyMembershipResult
	for _, h := range allHosts {
		memberships = append(memberships, fleet.PolicyMembershipResult{HostID: h.ID, PolicyID: p.ID, Passes: new(false)})
	}
	require.NoError(t, ds.AsyncBatchInsertPolicyMembership(ctx, memberships))

	runIDs := map[uint]uint{}
	for _, h := range allHosts {
		m, err := ds.RecordPolicyTransitions(ctx, h.ID, map[uint]*bool{p.ID: new(false)}, []uint{p.ID}, nil)
		require.NoError(t, err)
		require.NotZero(t, m[p.ID])
		runIDs[h.ID] = m[p.ID]
	}

	// Seed vpp_apps so host_vpp_software_installs_ibfk_3 (adam_id, platform) holds.
	const adamID = "999000111"
	const vppPlatform = "darwin"
	const appName = "TestVPPApp"
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO vpp_apps (adam_id, platform, name) VALUES (?, ?, ?)`,
			adamID, vppPlatform, appName)
		return err
	})

	insertHVSI := func(t *testing.T, host *fleet.Host, cmdUUID string, setVerifiedAt, setVerifiedFailedAt bool) {
		t.Helper()
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				INSERT INTO host_vpp_software_installs
				  (host_id, adam_id, platform, command_uuid, policy_run_id, verification_at, verification_failed_at)
				VALUES (?, ?, ?, ?, ?, IF(?, NOW(6), NULL), IF(?, NOW(6), NULL))`,
				host.ID, adamID, vppPlatform, cmdUUID, runIDs[host.ID],
				setVerifiedAt, setVerifiedFailedAt)
			return err
		})
	}

	insertUpcomingVPP := func(t *testing.T, host *fleet.Host, execID string) {
		t.Helper()
		var upcomingID int64
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx,
				`INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload) VALUES (?, 'vpp_app_install', ?, '{}')`,
				host.ID, execID)
			if err != nil {
				return err
			}
			upcomingID, err = res.LastInsertId()
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform, policy_run_id) VALUES (?, ?, ?, ?)`,
				upcomingID, adamID, vppPlatform, runIDs[host.ID])
			return err
		})
	}

	// Success: verification_at set.
	insertHVSI(t, hSuccess, "cmd-success", true, false)
	// Failure (verification): verification_failed_at set.
	insertHVSI(t, hVerifFail, "cmd-verifail", false, true)
	// Failure (MDM): result row with status='Error', no verification timestamps.
	// nano_command_results FKs require a nano_enrollments row (host's UUID) and
	// a nano_commands row (command_uuid). The CHECK constraint requires result
	// to start with '<?xml'.
	insertHVSI(t, hMDMError, "cmd-mdm-err", false, false)
	nanoEnroll(t, ds, hMDMError, false)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, 'InstallApplication', '<?xml')`,
			"cmd-mdm-err")
		if err != nil {
			return err
		}
		_, err = q.ExecContext(ctx,
			`INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, 'Error', '<?xml')`,
			hMDMError.UUID, "cmd-mdm-err")
		return err
	})
	// Queued (result row exists, no verification timestamps, no error result).
	insertHVSI(t, hQueuedResult, "cmd-queued-result", false, false)
	// Queued (upcoming-activity row only, no result row yet).
	insertUpcomingVPP(t, hQueuedUpcoming, "exec-upcoming")
	// Dedupe: both a real success row AND an upcoming row exist; the upcoming
	// row must be suppressed by the NOT EXISTS guard.
	insertHVSI(t, hDedupe, "cmd-dedupe", true, false)
	insertUpcomingVPP(t, hDedupe, "exec-dedupe")

	adminFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}}

	t.Run("status mapping per lifecycle state", func(t *testing.T) {
		runs, _, _, err := ds.GetPolicyStatus(ctx, p.ID, adminFilter, fleet.GetPolicyStatusRequest{
			ListOptions: fleet.ListOptions{PerPage: 100},
		})
		require.NoError(t, err)

		byHost := map[uint]fleet.GetPolicyStatusPolicyRun{}
		for _, r := range runs {
			byHost[r.HostID] = r
		}

		// VPP installs surface as 'software_installation' (no dedicated type).
		getOnly := func(t *testing.T, hostID uint) fleet.GetPolicyStatusAutomationExecution {
			t.Helper()
			r, ok := byHost[hostID]
			require.True(t, ok, "host %d not in response", hostID)
			var got []fleet.GetPolicyStatusAutomationExecution
			for _, a := range r.AutomationExecutions {
				if a.Type == "software_installation" {
					got = append(got, a)
				}
			}
			require.Len(t, got, 1, "expected exactly one software_installation row for host %d, got %+v", hostID, got)
			return got[0]
		}

		require.Equal(t, "success", getOnly(t, hSuccess.ID).Status)
		require.Equal(t, appName, getOnly(t, hSuccess.ID).Name)
		require.Equal(t, "failed", getOnly(t, hVerifFail.ID).Status, "verification_failed_at must map to failed")
		require.Equal(t, "failed", getOnly(t, hMDMError.ID).Status, "ncr.status='Error' must map to failed")
		require.Equal(t, "queued", getOnly(t, hQueuedResult.ID).Status, "result row with no verification + no error is queued")
		require.Equal(t, "queued", getOnly(t, hQueuedUpcoming.ID).Status, "upcoming-only row is queued")

		// Dedupe: only the real row should surface, not the upcoming row.
		require.Equal(t, "success", getOnly(t, hDedupe.ID).Status,
			"NOT EXISTS guard on vpp_app_upcoming_activities must suppress the upcoming row when a result row exists")
	})

	t.Run("automation_failed filter includes VPP failures", func(t *testing.T) {
		runs, count, _, err := ds.GetPolicyStatus(ctx, p.ID, adminFilter, fleet.GetPolicyStatusRequest{
			RunStatus:   "automation_failed",
			ListOptions: fleet.ListOptions{PerPage: 100},
		})
		require.NoError(t, err)

		got := map[uint]struct{}{}
		for _, r := range runs {
			got[r.HostID] = struct{}{}
		}
		// Both VPP failure modes must be picked up.
		require.Contains(t, got, hVerifFail.ID, "verification_failed_at must count as automation_failed")
		require.Contains(t, got, hMDMError.ID, "ncr.status='Error' must count as automation_failed")
		// Success / queued hosts must be filtered out.
		require.NotContains(t, got, hSuccess.ID)
		require.NotContains(t, got, hQueuedResult.ID)
		require.NotContains(t, got, hQueuedUpcoming.ID)
		require.NotContains(t, got, hDedupe.ID)
		require.Equal(t, 2, count)
	})
}

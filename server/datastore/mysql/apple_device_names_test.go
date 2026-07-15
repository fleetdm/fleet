package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestHostDeviceNames(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Eligibility", testHostDeviceNamesEligibility},
		{"NoTeam", testHostDeviceNamesNoTeam},
		{"CommandLifecycle", testHostDeviceNamesCommandLifecycle},
		{"DeactivateStaleCommands", testHostDeviceNamesDeactivateStaleCommands},
		{"RequeueClearsStaleCommand", testHostDeviceNamesRequeueClearsStaleCommand},
		{"TeamDeletionRequeuesUnderNoTeam", testHostDeviceNamesTeamDeletionRequeuesUnderNoTeam},
		{"Verify", testHostDeviceNamesVerify},
		{"Resend", testHostDeviceNamesResend},
		{"Reconcile", testHostDeviceNamesReconcile},
		{"TransferViaAddHostsToTeam", testHostDeviceNamesTransferViaAddHostsToTeam},
		{"TransferBatched", testHostDeviceNamesTransferBatched},
		{"SummaryAndFilter", testHostDeviceNamesSummaryAndFilter},
		{"NoTeamSummaryAndFilter", testHostDeviceNamesNoTeamSummaryAndFilter},
		{"SummaryFilterLabel", testHostDeviceNamesSummaryFilterLabel},
		{"TeamDeletionCleanup", testHostDeviceNamesTeamDeletionCleanup},
		{"HostDeletionCleanup", testHostDeviceNamesHostDeletionCleanup},
		{"FullLifecycle", testHostDeviceNamesFullLifecycle},
		{"ResolveResult", testHostDeviceNamesResolveResult},
		{"VerifyGracePeriod", testHostDeviceNamesVerifyGracePeriod},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// enrollAppleHostForDeviceName creates an Apple host and enrolls it in Fleet's
// MDM. When personal is true, the host is marked as a personal (BYOD) enrollment.
func enrollAppleHostForDeviceName(t *testing.T, ds *Datastore, name, platform string, teamID uint, personal bool) *fleet.Host {
	ctx := t.Context()
	host := test.NewHost(t, ds, name, "1.1.1.1", name+"-key", name+"-uuid", time.Now(),
		test.WithPlatform(platform), test.WithTeamID(teamID))

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	serverURL, err := apple_mdm.ResolveAppleEnrollMDMURL(ac.ServerSettings.ServerURL)
	require.NoError(t, err)

	nanoEnroll(t, ds, host, false)
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, host.ID, false, true, serverURL, true, fleet.WellKnownMDMFleet, "", personal))
	return host
}

func getDeviceNameRow(t *testing.T, ds *Datastore, hostUUID string) *fleet.HostDeviceNameEnforcement {
	enforcement, err := ds.GetHostDeviceNameEnforcement(t.Context(), hostUUID)
	require.NoError(t, err)
	return enforcement
}

func testHostDeviceNamesEligibility(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "eligibility-team"})
	require.NoError(t, err)

	macHost := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)
	iosHost := enrollAppleHostForDeviceName(t, ds, "ios", "ios", team.ID, false)
	ipadHost := enrollAppleHostForDeviceName(t, ds, "ipad", "ipados", team.ID, false)
	byodHost := enrollAppleHostForDeviceName(t, ds, "byod", "ios", team.ID, true)
	winHost := enrollAppleHostForDeviceName(t, ds, "win", "windows", team.ID, false)

	// Account-Driven User Enrollment (BYOD): nanomdm records the enrollment type
	// as "User Enrollment (Device)". Apple rejects the DeviceName command on user
	// (BYOD) enrollments, so it must be excluded on the enrollment type. This host
	// carries is_personal_enrollment = 0 (the column default) to represent a device
	// enrolled before that flag existed, proving the type filter — not just the
	// personal flag — is what keeps BYOD out.
	udBYODHost := test.NewHost(t, ds, "ud-byod", "1.1.1.4", "udb-key", "udb-uuid", time.Now(),
		test.WithPlatform("ios"), test.WithTeamID(team.ID))
	nanoEnrollUserDeviceAndSetHostMDMData(t, ds, udBYODHost)

	// linux and non-enrolled darwin hosts are never eligible.
	linuxHost := test.NewHost(t, ds, "linux", "1.1.1.2", "linux-key", "linux-uuid", time.Now(),
		test.WithPlatform("linux"), test.WithTeamID(team.ID))
	notEnrolled := test.NewHost(t, ds, "not-enrolled", "1.1.1.3", "ne-key", "ne-uuid", time.Now(),
		test.WithPlatform("darwin"), test.WithTeamID(team.ID))

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))

	// Only Apple, Fleet-MDM enrolled, non-personal hosts get a row.
	eligible := []*fleet.Host{macHost, iosHost, ipadHost}
	for _, h := range eligible {
		row := getDeviceNameRow(t, ds, h.UUID)
		require.Nil(t, row.Status, "eligible host %s should be queued (NULL status)", h.Hostname)
	}

	ineligible := []*fleet.Host{byodHost, udBYODHost, winHost, linuxHost, notEnrolled}
	for _, h := range ineligible {
		_, err := ds.GetHostDeviceNameEnforcement(ctx, h.UUID)
		require.True(t, fleet.IsNotFound(err), "ineligible host %s should have no row", h.Hostname)
	}

	// A re-save re-queues even hosts that had already been verified: mark one
	// verified, then bulk upsert resets its status back to NULL (ON DUPLICATE KEY
	// UPDATE branch).
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, fleet.MDMDeliveryVerified, macHost.UUID)
	require.NoError(t, err)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.Nil(t, getDeviceNameRow(t, ds, macHost.UUID).Status, "re-save should reset a verified host back to queued")

	// A second eligible host in another team, to prove delete is team-scoped.
	otherTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "eligibility-other-team"})
	require.NoError(t, err)
	otherHost := enrollAppleHostForDeviceName(t, ds, "other-mac", "darwin", otherTeam.ID, false)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &otherTeam.ID))

	// Clearing the team removes every row for that team and leaves other teams' rows.
	require.NoError(t, ds.DeleteHostDeviceNameEnforcementForTeam(ctx, &team.ID))
	for _, h := range eligible {
		_, err := ds.GetHostDeviceNameEnforcement(ctx, h.UUID)
		require.True(t, fleet.IsNotFound(err), "row for %s should be deleted", h.Hostname)
	}
	require.Nil(t, getDeviceNameRow(t, ds, otherHost.UUID).Status, "other team's row must survive the delete")
}

func testHostDeviceNamesCommandLifecycle(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "lifecycle-team"})
	require.NoError(t, err)

	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)
	// Give the host a serial so we can assert it flows through ListHostsPending.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET hardware_serial = ?, computer_name = ? WHERE id = ?`, "SERIAL123", "old-name", host.ID)
	require.NoError(t, err)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))

	// The queued host shows up in the pending list with its host details.
	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, host.ID, pending[0].HostID)
	require.Equal(t, host.UUID, pending[0].HostUUID)
	require.Equal(t, "SERIAL123", pending[0].HardwareSerial)
	require.Equal(t, "darwin", pending[0].Platform)
	require.Equal(t, "old-name", pending[0].ComputerName)
	require.NotNil(t, pending[0].TeamID)
	require.Equal(t, team.ID, *pending[0].TeamID)

	// Marking the command as sent moves the row to pending and records details.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd-1"), "WS-SERIAL123", ""))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.NotNil(t, row.Status)
	require.Equal(t, fleet.MDMDeliveryPending, *row.Status)
	require.NotNil(t, row.CommandUUID)
	require.Equal(t, "DEVNAME-cmd-1", *row.CommandUUID)
	require.NotNil(t, row.ExpectedDeviceName)
	require.Equal(t, "WS-SERIAL123", *row.ExpectedDeviceName)

	// It is no longer pending.
	pending, err = ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, pending)

	// An acknowledgment moves the row to verifying and renames the host in Fleet
	// (computer_name, hostname, display name) to the expected name, atomically.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd-1", true, ""))
	require.Equal(t, fleet.MDMDeliveryVerifying, *getDeviceNameRow(t, ds, host.UUID).Status)
	renamed, err := ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "WS-SERIAL123", renamed.ComputerName)
	require.Equal(t, "WS-SERIAL123", renamed.Hostname)
	require.Equal(t, "WS-SERIAL123", renamed.DisplayName())

	// An error result records the Apple detail and does not rename the host.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd-2"), "WS-SERIAL123", ""))
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd-2", false, "Apple error chain"))
	row = getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryFailed, *row.Status)
	require.Equal(t, "Apple error chain", row.Detail)

	// An unknown command UUID is a not-found error.
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-nope", true, "")
	require.True(t, fleet.IsNotFound(err))

	// Getting an enforcement row for a host with none is a not-found error.
	_, err = ds.GetHostDeviceNameEnforcement(ctx, "missing-uuid")
	require.True(t, fleet.IsNotFound(err))
}

func testHostDeviceNamesDeactivateStaleCommands(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "deactivate-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)
	other := enrollAppleHostForDeviceName(t, ds, "mac2", "darwin", team.ID, false)

	// enqueueCmd inserts a command and its (active) enrollment-queue row for a host.
	enqueueCmd := func(hostUUID, cmdUUID, requestType string) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx,
				`INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, '<?xml')`, cmdUUID, requestType); err != nil {
				return err
			}
			_, err := q.ExecContext(ctx,
				`INSERT INTO nano_enrollment_queue (id, command_uuid, active, priority) VALUES (?, ?, 1, 0)`, hostUUID, cmdUUID)
			return err
		})
	}
	queueActive := func(hostUUID, cmdUUID string) bool {
		var active bool
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &active,
				`SELECT active FROM nano_enrollment_queue WHERE id = ? AND command_uuid = ?`, hostUUID, cmdUUID)
		})
		return active
	}

	// A lingering device-name command from an earlier send, a non-device-name
	// command for the same host, and a device-name command for another host.
	enqueueCmd(host.UUID, fleet.DeviceNameCommandUUIDPrefix+"stale", "Settings")
	enqueueCmd(host.UUID, "INSTALL-profile", "InstallProfile")
	enqueueCmd(other.UUID, fleet.DeviceNameCommandUUIDPrefix+"other", "Settings")

	require.NoError(t, ds.DeactivateHostDeviceNameCommands(ctx, []string{host.UUID}))

	// Only the target host's device-name command is deactivated; its unrelated
	// command and the other host's command are untouched.
	require.False(t, queueActive(host.UUID, fleet.DeviceNameCommandUUIDPrefix+"stale"))
	require.True(t, queueActive(host.UUID, "INSTALL-profile"))
	require.True(t, queueActive(other.UUID, fleet.DeviceNameCommandUUIDPrefix+"other"))

	// Empty input is a no-op.
	require.NoError(t, ds.DeactivateHostDeviceNameCommands(ctx, nil))
}

func testHostDeviceNamesVerify(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "verify-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd"), "WS-1", ""))

	// A NULL/pending row is left untouched by verification.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-1"))
	require.Equal(t, fleet.MDMDeliveryPending, *getDeviceNameRow(t, ds, host.UUID).Status)

	// Move to verifying, then a matching report verifies it.
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd", true, "")
	require.NoError(t, err)
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-1"))
	require.Equal(t, fleet.MDMDeliveryVerified, *getDeviceNameRow(t, ds, host.UUID).Status)

	// Re-verifying an already-verified, still-matching row is a no-op: status
	// stays verified and the row is not rewritten (updated_at unchanged).
	verifiedAt := getDeviceNameRow(t, ds, host.UUID).UpdatedAt
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-1"))
	afterReverify := getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryVerified, *afterReverify.Status)
	require.True(t, afterReverify.UpdatedAt.Equal(verifiedAt), "re-verifying a matching row must not rewrite it")

	// A later mismatching report is drift: verified -> failed with a detail.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "renamed-by-user"))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryFailed, *row.Status)
	require.NotEmpty(t, row.Detail)

	// A failed row is left untouched even by a later matching report: only
	// verifying/verified rows are reconciled, so recovery requires an explicit
	// resend rather than silent self-healing.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-1"))
	require.Equal(t, fleet.MDMDeliveryFailed, *getDeviceNameRow(t, ds, host.UUID).Status)

	// A host with no row is a no-op (no error).
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, "missing-uuid", "anything"))
}

func testHostDeviceNamesResend(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "resend-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd"), "WS-1", ""))
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd", false, "boom")
	require.NoError(t, err)
	require.Equal(t, fleet.MDMDeliveryFailed, *getDeviceNameRow(t, ds, host.UUID).Status)

	// Resend resets the status to NULL so the cron re-enqueues it, and clears the
	// previous command UUID so a late ack for it can't match this row.
	require.NoError(t, ds.ResendHostDeviceName(ctx, host.UUID))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.Nil(t, row.Status)
	require.Nil(t, row.CommandUUID)

	// The previous command's late acknowledgment no longer matches any row.
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd", true, "")
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status, "late ack must not resurrect the row")

	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, host.UUID, pending[0].HostUUID)
}

func testHostDeviceNamesReconcile(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	setTemplate := func(teamID uint, tmpl string) {
		_, err := ds.writer(ctx).ExecContext(ctx,
			`UPDATE teams SET config = JSON_SET(config, '$.mdm.name_template', ?) WHERE id = ?`, tmpl, teamID)
		require.NoError(t, err)
	}

	withTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "with-template"})
	require.NoError(t, err)
	setTemplate(withTemplate.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")

	noTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "no-template"})
	require.NoError(t, err)

	hostWith := enrollAppleHostForDeviceName(t, ds, "with", "darwin", withTemplate.ID, false)
	hostWithout := enrollAppleHostForDeviceName(t, ds, "without", "darwin", noTemplate.ID, false)
	hostByod := enrollAppleHostForDeviceName(t, ds, "byod", "ios", withTemplate.ID, true)

	// Reconcile upserts rows for eligible hosts whose team has a template, and
	// leaves template-less / ineligible hosts without a row.
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{hostWith.ID, hostWithout.ID, hostByod.ID}))
	require.Nil(t, getDeviceNameRow(t, ds, hostWith.UUID).Status)
	for _, h := range []*fleet.Host{hostWithout, hostByod} {
		_, err := ds.GetHostDeviceNameEnforcement(ctx, h.UUID)
		require.True(t, fleet.IsNotFound(err), "host %s should have no row", h.Hostname)
	}

	// Simulate a transfer: the host moves to the template-less team. Reconcile
	// must delete its now-orphaned row.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = ? WHERE id = ?`, noTemplate.ID, hostWith.ID)
	require.NoError(t, err)
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{hostWith.ID}))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, hostWith.UUID)
	require.True(t, fleet.IsNotFound(err), "row should be deleted after transfer to template-less team")

	// Transfer back to the template team: reconcile re-creates the queued row.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = ? WHERE id = ?`, withTemplate.ID, hostWith.ID)
	require.NoError(t, err)
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{hostWith.ID}))
	require.Nil(t, getDeviceNameRow(t, ds, hostWith.UUID).Status)

	// An empty host list is a no-op.
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, nil))
}

func testHostDeviceNamesNoTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// setNoTeamTemplate writes name_template into the global app config JSON, the
	// storage location for the No-team template, mirroring the team helper.
	setNoTeamTemplate := func(tmpl string) {
		_, err := ds.writer(ctx).ExecContext(ctx,
			`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', ?)`, tmpl)
		require.NoError(t, err)
	}

	// A team host proves the No-team writers never touch team-scoped rows.
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "no-team-scope-control"})
	require.NoError(t, err)
	teamHost := enrollAppleHostForDeviceName(t, ds, "team-mac", "darwin", team.ID, false)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))

	// A No-team eligible Apple host and a No-team BYOD host (enrolled into a temp
	// team, then moved to No team so team_id IS NULL).
	noTeamMac := enrollAppleHostForDeviceName(t, ds, "noteam-mac", "darwin", team.ID, false)
	noTeamByod := enrollAppleHostForDeviceName(t, ds, "noteam-byod", "ios", team.ID, true)
	for _, h := range []*fleet.Host{noTeamMac, noTeamByod} {
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = NULL WHERE id = ?`, h.ID)
		require.NoError(t, err)
	}

	// BulkUpsert with a nil team scopes to team_id IS NULL: only the eligible
	// No-team host is queued; BYOD stays rowless and the team row is untouched.
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, nil))
	require.Nil(t, getDeviceNameRow(t, ds, noTeamMac.UUID).Status)
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamByod.UUID)
	require.True(t, fleet.IsNotFound(err), "No-team BYOD host must not get a row")
	require.Nil(t, getDeviceNameRow(t, ds, teamHost.UUID).Status, "team host row must survive a No-team upsert")

	// Delete with a nil team removes only No-team rows.
	require.NoError(t, ds.DeleteHostDeviceNameEnforcementForTeam(ctx, nil))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamMac.UUID)
	require.True(t, fleet.IsNotFound(err), "No-team row should be deleted")
	require.Nil(t, getDeviceNameRow(t, ds, teamHost.UUID).Status, "team host row must survive a No-team delete")

	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', CAST('null' AS JSON))`)
	require.NoError(t, err)
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{noTeamMac.ID}))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamMac.UUID)
	require.True(t, fleet.IsNotFound(err), "enrollment reconcile: JSON-null No-team template must not enforce")
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{noTeamMac.ID})))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamMac.UUID)
	require.True(t, fleet.IsNotFound(err), "team-scoped reconcile: JSON-null No-team template must not enforce")

	// Reconcile resolves the No-team template from app config: with a template set
	// the eligible No-team host is queued and BYOD stays rowless.
	setNoTeamTemplate("WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{noTeamMac.ID, noTeamByod.ID}))
	require.Nil(t, getDeviceNameRow(t, ds, noTeamMac.UUID).Status)
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamByod.UUID)
	require.True(t, fleet.IsNotFound(err), "No-team BYOD host must not be reconciled into a row")

	// Clearing the No-team template makes reconcile delete the orphaned row.
	setNoTeamTemplate("")
	require.NoError(t, ds.ReconcileHostDeviceNamesForHosts(ctx, []uint{noTeamMac.ID}))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, noTeamMac.UUID)
	require.True(t, fleet.IsNotFound(err), "No-team row should be deleted after the template is cleared")
}

// setDeviceNameTemplate writes name_template into a team's config JSON directly,
// so the eligibility/reconcile SQL sees a non-empty template without depending on
// the TeamMDM struct field.
func setDeviceNameTemplate(t *testing.T, ds *Datastore, teamID uint, tmpl string) {
	_, err := ds.writer(t.Context()).ExecContext(t.Context(),
		`UPDATE teams SET config = JSON_SET(config, '$.mdm.name_template', ?) WHERE id = ?`, tmpl, teamID)
	require.NoError(t, err)
}

// testHostDeviceNamesTransferViaAddHostsToTeam exercises the transfer
// reconciliation wired into the AddHostsToTeam datastore transaction, which
// covers every service entry point that moves hosts between teams.
func testHostDeviceNamesTransferViaAddHostsToTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	withTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "transfer-with-template"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, withTemplate.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")

	noTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "transfer-no-template"})
	require.NoError(t, err)

	// A host starts in the template team with a queued row.
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", withTemplate.ID, false)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &withTemplate.ID))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status)

	// Give the host a name so we can assert the transfer never renames it.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET computer_name = ? WHERE id = ?`, "keep-this-name", host.ID)
	require.NoError(t, err)

	// template -> template-less: the row is deleted (enforcement stops) and the
	// host's name is left untouched.
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&noTemplate.ID, []uint{host.ID})))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.True(t, fleet.IsNotFound(err), "transfer to a template-less team should delete the enforcement row")
	var name string
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &name, `SELECT computer_name FROM hosts WHERE id = ?`, host.ID))
	require.Equal(t, "keep-this-name", name, "transfer must not rename the host")

	// template-less -> template: reconcile re-creates the queued row (UI reverts
	// to Enforcing/Pending).
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&withTemplate.ID, []uint{host.ID})))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status, "transfer into a template team should create a queued row")

	// template -> template (a different team that also has a template): the row is
	// reset to NULL so the destination team's template is enforced afresh. Mark it
	// verified first to prove the transfer resets an already-settled row.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, fleet.MDMDeliveryVerified, host.UUID)
	require.NoError(t, err)
	otherTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "transfer-other-template"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, otherTemplate.ID, "Lab-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&otherTemplate.ID, []uint{host.ID})))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status, "transfer between template teams should reset the row to queued")

	// template -> No team (No team has no template): the row is deleted.
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{host.ID})))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.True(t, fleet.IsNotFound(err), "transfer to a template-less No team should delete the enforcement row")

	// No team WITH a template: transferring the host into No team now queues a row,
	// resolving the template from the global app config (the team-scoped reconcile's
	// No-team branch).
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', ?)`, "NoTeam-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{host.ID})))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status, "transfer into No team with a template should queue a row")
}

// testHostDeviceNamesTransferBatched moves several hosts at once with a batch
// size smaller than the host count, so the reconcile runs across multiple batches
// within AddHostsToTeam.
func testHostDeviceNamesTransferBatched(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	noTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "batch-no-template"})
	require.NoError(t, err)
	withTemplate, err := ds.NewTeam(ctx, &fleet.Team{Name: "batch-with-template"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, withTemplate.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")

	// Three eligible hosts start in the template-less team (no rows) plus one BYOD
	// host that must never get a row.
	var hostIDs []uint
	eligible := make([]*fleet.Host, 0, 3)
	for i := range 3 {
		h := enrollAppleHostForDeviceName(t, ds, "batch"+string(rune('a'+i)), "darwin", noTemplate.ID, false)
		eligible = append(eligible, h)
		hostIDs = append(hostIDs, h.ID)
	}
	byod := enrollAppleHostForDeviceName(t, ds, "batch-byod", "ios", noTemplate.ID, true)
	hostIDs = append(hostIDs, byod.ID)

	// Move all four into the template team with a batch size of 1 so the reconcile
	// runs once per batch.
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&withTemplate.ID, hostIDs).WithBatchSize(1)))
	for _, h := range eligible {
		require.Nil(t, getDeviceNameRow(t, ds, h.UUID).Status, "eligible host %s should be queued after batched transfer", h.Hostname)
	}
	_, err = ds.GetHostDeviceNameEnforcement(ctx, byod.UUID)
	require.True(t, fleet.IsNotFound(err), "BYOD host must not get a row")

	// Move them all back to the template-less team, again batched: all rows deleted.
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&noTemplate.ID, hostIDs).WithBatchSize(2)))
	for _, h := range eligible {
		_, err = ds.GetHostDeviceNameEnforcement(ctx, h.UUID)
		require.True(t, fleet.IsNotFound(err), "row for %s should be deleted after batched transfer out", h.Hostname)
	}
}

// testHostDeviceNamesSummaryAndFilter asserts that host-name enforcement rows are
// folded into the OS-settings aggregate counts and the os_settings host-list
// filter, and that ineligible hosts (no row) count in no bucket.
func testHostDeviceNamesSummaryAndFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "summary-team"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, team.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")

	failedHost := enrollAppleHostForDeviceName(t, ds, "failed", "darwin", team.ID, false)
	verifiedHost := enrollAppleHostForDeviceName(t, ds, "verified", "ios", team.ID, false)
	verifyingHost := enrollAppleHostForDeviceName(t, ds, "verifying", "ios", team.ID, false)
	queuedHost := enrollAppleHostForDeviceName(t, ds, "queued", "ipados", team.ID, false)
	comboHost := enrollAppleHostForDeviceName(t, ds, "combo", "darwin", team.ID, false)
	byodHost := enrollAppleHostForDeviceName(t, ds, "byod", "ios", team.ID, true)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	// The BYOD host is ineligible, so it never got a row.
	_, err = ds.GetHostDeviceNameEnforcement(ctx, byodHost.UUID)
	require.True(t, fleet.IsNotFound(err))

	setStatus := func(hostUUID string, status fleet.MDMDeliveryStatus) {
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, status, hostUUID)
		require.NoError(t, err)
	}
	setStatus(failedHost.UUID, fleet.MDMDeliveryFailed)
	setStatus(verifiedHost.UUID, fleet.MDMDeliveryVerified)
	setStatus(verifyingHost.UUID, fleet.MDMDeliveryVerifying)
	setStatus(comboHost.UUID, fleet.MDMDeliveryFailed)
	// queuedHost keeps its NULL status from the bulk upsert, which renders as pending.

	// comboHost has a failed rename (set above) AND a verified (install) config
	// profile: the shared status CASE must combine the profile and device-name
	// buckets and let the failed rename win (failed > verified precedence).
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_mdm_apple_profiles
			(host_uuid, profile_uuid, command_uuid, status, operation_type, detail, profile_name, profile_identifier, checksum)
		VALUES (?, ?, '', ?, ?, '', 'p1', 'com.example.p1', ?)`,
		comboHost.UUID, "a"+comboHost.UUID, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeInstall, []byte("csum"))
	require.NoError(t, err)

	// Aggregate summary folds the rename statuses in (a NULL/queued row counts as
	// pending); the BYOD host is in no bucket, and comboHost lands in failed.
	summary, err := ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.Equal(t, uint(2), summary.Failed, "failedHost + comboHost (failed rename wins over its verified profile)")
	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(1), summary.Verifying)
	require.Equal(t, uint(1), summary.Pending)

	// Each aggregate card's host-list filter returns the matching hosts, including
	// the queued (NULL-status) host under pending and comboHost under failed.
	userFilter := fleet.TeamFilter{User: test.UserAdmin}
	assertFilter := func(status fleet.OSSettingsStatus, want ...*fleet.Host) {
		hosts, err := ds.ListHosts(ctx, userFilter, fleet.HostListOptions{TeamFilter: &team.ID, OSSettingsFilter: status})
		require.NoError(t, err)
		gotIDs := make([]uint, 0, len(hosts))
		for _, h := range hosts {
			gotIDs = append(gotIDs, h.ID)
		}
		wantIDs := make([]uint, 0, len(want))
		for _, h := range want {
			wantIDs = append(wantIDs, h.ID)
		}
		require.ElementsMatch(t, wantIDs, gotIDs)
	}
	assertFilter(fleet.OSSettingsFailed, failedHost, comboHost)
	assertFilter(fleet.OSSettingsVerified, verifiedHost)
	assertFilter(fleet.OSSettingsVerifying, verifyingHost)
	assertFilter(fleet.OSSettingsPending, queuedHost)
}

func testHostDeviceNamesNoTeamSummaryAndFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// The No-team template lives on the global app config.
	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', ?)`, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, err)

	// Hosts are enrolled into a temp team then moved to No team (team_id IS NULL).
	tmpTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "noteam-summary-tmp"})
	require.NoError(t, err)
	failedHost := enrollAppleHostForDeviceName(t, ds, "nt-failed", "darwin", tmpTeam.ID, false)
	verifiedHost := enrollAppleHostForDeviceName(t, ds, "nt-verified", "ios", tmpTeam.ID, false)
	verifyingHost := enrollAppleHostForDeviceName(t, ds, "nt-verifying", "ios", tmpTeam.ID, false)
	queuedHost := enrollAppleHostForDeviceName(t, ds, "nt-queued", "ipados", tmpTeam.ID, false)
	byodHost := enrollAppleHostForDeviceName(t, ds, "nt-byod", "ios", tmpTeam.ID, true)
	for _, h := range []*fleet.Host{failedHost, verifiedHost, verifyingHost, queuedHost, byodHost} {
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = NULL WHERE id = ?`, h.ID)
		require.NoError(t, err)
	}

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, nil))
	// The BYOD host is ineligible, so it never got a row (and host-detail omits it).
	_, err = ds.GetHostDeviceNameEnforcement(ctx, byodHost.UUID)
	require.True(t, fleet.IsNotFound(err))

	setStatus := func(hostUUID string, status fleet.MDMDeliveryStatus) {
		_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, status, hostUUID)
		require.NoError(t, err)
	}
	setStatus(failedHost.UUID, fleet.MDMDeliveryFailed)
	setStatus(verifiedHost.UUID, fleet.MDMDeliveryVerified)
	setStatus(verifyingHost.UUID, fleet.MDMDeliveryVerifying)
	// queuedHost keeps its NULL status from the bulk upsert, which renders as pending.

	// The host-detail row lookup returns the queued/failed rows (host-keyed).
	require.Equal(t, fleet.MDMDeliveryFailed, *getDeviceNameRow(t, ds, failedHost.UUID).Status)

	// Aggregate summary for No team (nil team) folds the rename statuses in.
	summary, err := ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(1), summary.Failed)
	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(1), summary.Verifying)
	require.Equal(t, uint(1), summary.Pending)

	// The os_settings host-list filter scoped to No team (TeamFilter == 0) returns
	// the matching No-team hosts per bucket.
	noTeam := uint(0)
	userFilter := fleet.TeamFilter{User: test.UserAdmin}
	assertFilter := func(status fleet.OSSettingsStatus, want ...*fleet.Host) {
		hosts, err := ds.ListHosts(ctx, userFilter, fleet.HostListOptions{TeamFilter: &noTeam, OSSettingsFilter: status})
		require.NoError(t, err)
		gotIDs := make([]uint, 0, len(hosts))
		for _, h := range hosts {
			gotIDs = append(gotIDs, h.ID)
		}
		wantIDs := make([]uint, 0, len(want))
		for _, h := range want {
			wantIDs = append(wantIDs, h.ID)
		}
		require.ElementsMatch(t, wantIDs, gotIDs)
	}
	assertFilter(fleet.OSSettingsFailed, failedHost)
	assertFilter(fleet.OSSettingsVerified, verifiedHost)
	assertFilter(fleet.OSSettingsVerifying, verifyingHost)
	assertFilter(fleet.OSSettingsPending, queuedHost)
}

// testHostDeviceNamesSummaryFilterLabel covers the os_settings host-list filter
// on the label-hosts path (ListHostsInLabel), which folds in the same
// device-name status join as the main list path.
func testHostDeviceNamesSummaryFilterLabel(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "label-team"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, team.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")

	failedHost := enrollAppleHostForDeviceName(t, ds, "label-failed", "darwin", team.ID, false)
	verifiedHost := enrollAppleHostForDeviceName(t, ds, "label-verified", "ios", team.ID, false)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, fleet.MDMDeliveryFailed, failedHost.UUID)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_mdm_apple_device_names SET status = ? WHERE host_uuid = ?`, fleet.MDMDeliveryVerified, verifiedHost.UUID)
	require.NoError(t, err)

	label, err := ds.NewLabel(ctx, &fleet.Label{Name: "label-dn", Query: "select 1"})
	require.NoError(t, err)
	for _, h := range []*fleet.Host{failedHost, verifiedHost} {
		require.NoError(t, ds.RecordLabelQueryExecutions(ctx, h, map[uint]*bool{label.ID: new(true)}, time.Now(), false))
	}

	userFilter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.ListHostsInLabel(ctx, userFilter, label.ID, fleet.HostListOptions{TeamFilter: &team.ID, OSSettingsFilter: fleet.OSSettingsFailed})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, failedHost.ID, hosts[0].ID)
}

// testHostDeviceNamesTeamDeletionCleanup asserts that deleting a team removes the
// host-name enforcement rows for its hosts. Team deletion moves the hosts to "No
// team" via ON DELETE SET NULL (not AddHostsToTeam), so the enforcement rows must
// be cleaned up explicitly in DeleteTeam.
func testHostDeviceNamesTeamDeletionCleanup(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// This case asserts the "No team has no template" behavior. app_config_json is
	// not truncated between subtests, so establish that precondition explicitly
	// rather than rely on leftover global state (otherwise deleting the team would
	// re-queue its hosts under a leaked No-team template).
	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', CAST('null' AS JSON))`)
	require.NoError(t, err)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "delete-me"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, team.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	host := enrollAppleHostForDeviceName(t, ds, "del", "darwin", team.ID, false)

	otherTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "keep-me"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, otherTeam.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	otherHost := enrollAppleHostForDeviceName(t, ds, "keep", "darwin", otherTeam.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &otherTeam.ID))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status)
	require.Nil(t, getDeviceNameRow(t, ds, otherHost.UUID).Status)

	require.NoError(t, ds.DeleteTeam(ctx, team.ID))

	// The deleted team's host keeps its record but moves to No team; its
	// enforcement row is gone.
	_, err = ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.True(t, fleet.IsNotFound(err), "enforcement row must be deleted with the team")
	movedHost, err := ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.Nil(t, movedHost.TeamID, "host should have moved to No team, not been deleted")

	// The other team's row is untouched.
	require.Nil(t, getDeviceNameRow(t, ds, otherHost.UUID).Status, "other team's row must survive")
}

// testHostDeviceNamesRequeueClearsStaleCommand locks in that re-queuing a row
// (template change → BulkUpsert) clears the previously-sent command tracking, so
// a late acknowledgment of the superseded command can't match the re-queued row
// and rename the host to the old name.
func testHostDeviceNamesRequeueClearsStaleCommand(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "requeue-team"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, team.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET computer_name = ? WHERE id = ?`, "current-name", host.ID)
	require.NoError(t, err)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	// Simulate a command in flight: pending row carrying a command UUID + the
	// resolved name it expects the device to apply.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-stale"), "OLD-NAME", ""))

	// A template change re-queues the row; the stale command tracking must be cleared.
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.Nil(t, row.Status, "re-queued row must be NULL status")
	require.Nil(t, row.CommandUUID, "re-queue must clear the stale command_uuid")
	require.Nil(t, row.ExpectedDeviceName, "re-queue must clear the stale expected name")

	// A late ACK for the superseded command must not match the re-queued row and
	// must not rename the host.
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-stale", true, "")
	require.True(t, fleet.IsNotFound(err), "late ACK for a superseded command must not match the re-queued row")
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status, "row must remain queued")
	h, err := ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "current-name", h.ComputerName, "a stale ACK must not rename the host")
}

// testHostDeviceNamesTeamDeletionRequeuesUnderNoTeam covers deleting a team whose
// hosts fall to a "No team" that has its own template: the hosts must be
// re-queued under the No-team template, not left permanently unenforced.
func testHostDeviceNamesTeamDeletionRequeuesUnderNoTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// app_config_json is not truncated between subtests, so restore the No-team
	// template afterwards to avoid leaking it into later cases.
	defer func() {
		_, err := ds.writer(ctx).ExecContext(ctx,
			`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', CAST('null' AS JSON))`)
		require.NoError(t, err)
	}()

	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.name_template', ?)`,
		"NT-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	require.NoError(t, err)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "delete-into-noteam"})
	require.NoError(t, err)
	setDeviceNameTemplate(t, ds, team.ID, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL")
	eligible := enrollAppleHostForDeviceName(t, ds, "elig", "darwin", team.ID, false)
	byod := enrollAppleHostForDeviceName(t, ds, "byod", "ios", team.ID, true)
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	// Put the eligible host mid-flight to also prove the reconcile resets it.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, eligible.UUID, fleet.MDMDeliveryPending, new("DEVNAME-x"), "WS-OLD", ""))

	require.NoError(t, ds.DeleteTeam(ctx, team.ID))

	// The eligible host moved to No team (which has a template) and is re-queued
	// with its stale command cleared; the BYOD host stays rowless.
	row := getDeviceNameRow(t, ds, eligible.UUID)
	require.Nil(t, row.Status, "host must be re-queued under the No-team template")
	require.Nil(t, row.CommandUUID, "re-queue must clear the stale command_uuid")
	movedHost, err := ds.Host(ctx, eligible.ID)
	require.NoError(t, err)
	require.Nil(t, movedHost.TeamID, "host should have moved to No team")
	_, err = ds.GetHostDeviceNameEnforcement(ctx, byod.UUID)
	require.True(t, fleet.IsNotFound(err), "BYOD host must not be enforced under No team")
}

func testHostDeviceNamesHostDeletionCleanup(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "deletion-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status)

	// Deleting the host must remove its enforcement row (no FK cascades it).
	require.NoError(t, ds.DeleteHost(ctx, host.ID))
	_, err = ds.GetHostDeviceNameEnforcement(ctx, host.UUID)
	require.True(t, fleet.IsNotFound(err), "enforcement row must be deleted with the host")
}

func testHostDeviceNamesResolveResult(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "resolve-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))

	// A too-long resolution fails the row without sending a command; the row
	// leaves the pending list so the cron does not retry it.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryFailed, nil, "", "Resolved name exceeds 63 bytes."))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.NotNil(t, row.Status)
	require.Equal(t, fleet.MDMDeliveryFailed, *row.Status)
	require.Equal(t, "Resolved name exceeds 63 bytes.", row.Detail)
	require.Nil(t, row.CommandUUID)
	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, pending)

	// An already-matching host goes straight to verified with the resolved name
	// recorded, so later reports can still detect drift from that name.
	require.NoError(t, ds.ResendHostDeviceName(ctx, host.UUID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryVerified, nil, "WS-1", ""))
	row = getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryVerified, *row.Status)
	require.NotNil(t, row.ExpectedDeviceName)
	require.Equal(t, "WS-1", *row.ExpectedDeviceName)
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "renamed-on-device"))
	require.Equal(t, fleet.MDMDeliveryFailed, *getDeviceNameRow(t, ds, host.UUID).Status)

	// Recording a resolve result clears any previously sent command UUID, so a
	// stale result for that command can't overwrite the outcome.
	require.NoError(t, ds.ResendHostDeviceName(ctx, host.UUID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-stale"), "WS-1", ""))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryVerified, nil, "WS-1", ""))
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-stale", false, "boom")
	require.True(t, fleet.IsNotFound(err))
	require.Equal(t, fleet.MDMDeliveryVerified, *getDeviceNameRow(t, ds, host.UUID).Status)

	// A host with no row is a no-op (no error).
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, "missing-uuid", fleet.MDMDeliveryVerified, nil, "WS-1", ""))
}

func testHostDeviceNamesVerifyGracePeriod(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "grace-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)

	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd"), "WS-1", ""))
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd", true, "")
	require.NoError(t, err)

	// A mismatching report arriving shortly after the acknowledgment is a report
	// that was generated before the device applied the rename, not drift: the row
	// stays verifying and waits for a fresh report.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "stale-pre-rename-name"))
	require.Equal(t, fleet.MDMDeliveryVerifying, *getDeviceNameRow(t, ds, host.UUID).Status)

	// A matching report is trusted at any time.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-1"))
	require.Equal(t, fleet.MDMDeliveryVerified, *getDeviceNameRow(t, ds, host.UUID).Status)

	// A mismatch on a verified row is genuine drift, grace period or not: the
	// verified state was reached by a fresh post-rename report, so a later
	// mismatch means the device was renamed off-template.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "renamed-by-user"))
	require.Equal(t, fleet.MDMDeliveryFailed, *getDeviceNameRow(t, ds, host.UUID).Status)

	// Once the grace period has elapsed, a mismatch on a still-verifying row is
	// no longer explainable as an in-flight stale report and fails the row.
	require.NoError(t, ds.ResendHostDeviceName(ctx, host.UUID))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-cmd-2"), "WS-1", ""))
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-cmd-2", true, "")
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_mdm_apple_device_names SET updated_at = DATE_SUB(NOW(6), INTERVAL 1 HOUR) WHERE host_uuid = ?`, host.UUID)
	require.NoError(t, err)
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "still-the-old-name"))
	row := getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryFailed, *row.Status)
	require.NotEmpty(t, row.Detail)
}

// testHostDeviceNamesFullLifecycle walks a single host through the entire
// enforcement state machine in the order the real actors drive it: admin saves a
// template (bulk upsert), the cron picks up the queued row and sends a command,
// the MDM result handler acks it, name ingestion verifies it, the device drifts,
// the admin resends, and finally a second command supersedes an in-flight one so
// the stale ack is dropped.
func testHostDeviceNamesFullLifecycle(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "lifecycle-team"})
	require.NoError(t, err)
	host := enrollAppleHostForDeviceName(t, ds, "mac", "darwin", team.ID, false)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET hardware_serial = ?, computer_name = ? WHERE id = ?`, "SERIAL1", "old-name", host.ID)
	require.NoError(t, err)

	// 1. Admin saves a template -> the host is queued (status NULL).
	require.NoError(t, ds.BulkUpsertHostDeviceNameEnforcement(ctx, &team.ID))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status)

	// 2. Cron picks up the queued row and enqueues a command.
	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, host.UUID, pending[0].HostUUID)
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-1"), "WS-SERIAL1", ""))
	require.Equal(t, fleet.MDMDeliveryPending, *getDeviceNameRow(t, ds, host.UUID).Status)

	// 3. MDM acks the command -> row goes verifying and the host is renamed in Fleet.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-1", true, ""))
	require.Equal(t, fleet.MDMDeliveryVerifying, *getDeviceNameRow(t, ds, host.UUID).Status)

	// 4. Name ingestion reports the matching name -> verified.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-SERIAL1"))
	require.Equal(t, fleet.MDMDeliveryVerified, *getDeviceNameRow(t, ds, host.UUID).Status)

	// 5. The device drifts (renamed on-device) -> failed with a detail.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "renamed-on-device"))
	failed := getDeviceNameRow(t, ds, host.UUID)
	require.Equal(t, fleet.MDMDeliveryFailed, *failed.Status)
	require.NotEmpty(t, failed.Detail)

	// 6. Admin clicks Resend -> back to queued, and the cron sees it again.
	require.NoError(t, ds.ResendHostDeviceName(ctx, host.UUID))
	require.Nil(t, getDeviceNameRow(t, ds, host.UUID).Status)
	pending, err = ds.ListHostsPendingDeviceNameCommand(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	// 7. A second command supersedes an in-flight one: the cron sends command 2
	// while command 1 is still outstanding, then command 1's stale ack arrives.
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-2a"), "WS-SERIAL1", ""))
	require.NoError(t, ds.SetHostDeviceNameStatus(ctx, host.UUID, fleet.MDMDeliveryPending, new("DEVNAME-2b"), "WS-SERIAL1", ""))

	// The superseded command's ack no longer matches the row -> not found, and
	// the row is untouched (still pending on the newest command).
	err = ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-2a", true, "")
	require.True(t, fleet.IsNotFound(err))
	require.Equal(t, fleet.MDMDeliveryPending, *getDeviceNameRow(t, ds, host.UUID).Status)

	// The newest command's ack is applied and renames the host.
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromCommand(ctx, "DEVNAME-2b", true, ""))
	require.NoError(t, ds.UpdateHostDeviceNameStatusFromReport(ctx, host.UUID, "WS-SERIAL1"))
	require.Equal(t, fleet.MDMDeliveryVerified, *getDeviceNameRow(t, ds, host.UUID).Status)
}

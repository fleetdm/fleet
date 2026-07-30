package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enrollWindowsHostInMDMForTest inserts a Windows MDM enrollment for the host and mirrors what osquery's
// directIngestMDMWindows does (host_mdm.enrolled = 1 once the device's registry confirms MDM enrollment), making the
// host eligible for the Windows profile reconcilers. Returns the enrolled device row, reloaded so its ID is set.
func enrollWindowsHostInMDMForTest(t *testing.T, ds fleet.Datastore, host *fleet.Host) *fleet.MDMWindowsEnrolledDevice {
	t.Helper()
	ctx := t.Context()

	dev := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "TestDeviceName",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               host.UUID,
	}
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, dev))
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, host.ID, false, true,
		"https://example.com", false, fleet.WellKnownMDMFleet, "", false))

	dev, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, dev.MDMDeviceID)
	require.NoError(t, err)
	return dev
}

// TestReconcileWindowsProfilesAfterTeamAddDeferred is the real-DB seam test
// for the production async path: BulkSetPendingMDMHostProfiles defers Windows
// reconciliation, and the install row only appears after
// ReconcileWindowsProfiles (the cron) runs.
func TestReconcileWindowsProfilesAfterTeamAddDeferred(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()
	logger := testutils.TestLogger(t)

	// ReconcileWindowsProfiles short-circuits to no-op when Windows MDM is
	// not enabled in app config.
	appCfg, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.WindowsEnabledAndConfigured = true
	require.NoError(t, ds.SaveAppConfig(ctx, appCfg))

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "deferred-recon-after-team-add"})
	require.NoError(t, err)
	profileUUID := mysqltest.InsertWindowsProfileForTest(t, ds, team.ID)

	host := test.NewHost(t, ds, "deferred-recon-host", "1.1.1.1", "deferred-recon-key", "deferred-recon-host-uuid", time.Now(),
		test.WithPlatform("windows"), test.WithTeamID(team.ID))

	enrollWindowsHostInMDMForTest(t, ds, host)

	// Step 1: BulkSet defers Windows reconciliation. Production leaves
	// updates.WindowsConfigProfile false (the consumer in
	// service/mdm.go's BatchSetMDMProfiles ORs it with the transactional
	// signal from batchSetMDMWindowsProfilesDB).
	updates, err := ds.BulkSetPendingMDMHostProfiles(ctx, []uint{host.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.WindowsConfigProfile)

	rowsBefore, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, rowsBefore, "host_mdm_windows_profiles must be untouched until the cron runs")

	// Sanity: the snapshot+compute path must surface the host+profile pair, otherwise the cron has nothing to dispatch.
	hosts, allProfiles, hostLabels, currentByHost, err := ds.GetWindowsProfileReconcileSnapshot(ctx, "", 10_000)
	require.NoError(t, err)
	profilesByTeam := make(map[uint][]*fleet.WindowsProfileForReconcile)
	profilesWithBrokenLabels := make(map[string]struct{})
	for _, p := range allProfiles {
		profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)
		if p.HasBrokenLabel() {
			profilesWithBrokenLabels[p.ProfileUUID] = struct{}{}
		}
	}
	toInstall, _ := microsoft_mdm.ComputeWindowsReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabels)
	var matched bool
	for _, p := range toInstall {
		if p.HostUUID == host.UUID && p.ProfileUUID == profileUUID {
			matched = true
			break
		}
	}
	require.True(t, matched, "desired-state listing did not surface our host+profile pair; got %d entries", len(toInstall))

	// Step 2: drive the cron.
	require.NoError(t, ReconcileWindowsProfiles(ctx, ds, logger))

	// Step 3: the install row should now exist.
	rowsAfter, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, rowsAfter, 1, "cron must enqueue the team's Windows profile for the new host")
	assert.Equal(t, profileUUID, rowsAfter[0].ProfileUUID)
	assert.Equal(t, fleet.MDMOperationTypeInstall, rowsAfter[0].OperationType)
}

// TestReconcileWindowsProfilesForEnrollingHost is the real-DB seam test for the per-host path: a freshly enrolled
// host gets its team's profile queued by ReconcileWindowsProfilesForEnrollingHost directly, without waiting for the
// cron's walk.
func TestReconcileWindowsProfilesForEnrollingHost(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()
	logger := testutils.TestLogger(t)

	appCfg, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.WindowsEnabledAndConfigured = true
	require.NoError(t, ds.SaveAppConfig(ctx, appCfg))

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "per-host-recon-enroll"})
	require.NoError(t, err)
	profileUUID := mysqltest.InsertWindowsProfileForTest(t, ds, team.ID)

	host := test.NewHost(t, ds, "per-host-recon-host", "1.1.1.2", "per-host-recon-key", "per-host-recon-host-uuid", time.Now(),
		test.WithPlatform("windows"), test.WithTeamID(team.ID))

	// Before the host is an eligible MDM-enrolled host, the per-host reconcile is a no-op.
	require.NoError(t, ReconcileWindowsProfilesForEnrollingHost(ctx, ds, logger, host.UUID))
	rows, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, rows, "ineligible host must not get profiles queued")

	dev := enrollWindowsHostInMDMForTest(t, ds, host)

	// Now the per-host reconcile queues the team's profile immediately: a pending install row AND a command the
	// device will receive on its next management session.
	require.NoError(t, ReconcileWindowsProfilesForEnrollingHost(ctx, ds, logger, host.UUID))
	rows, err = ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "per-host reconcile must queue the team's Windows profile at enrollment")
	assert.Equal(t, profileUUID, rows[0].ProfileUUID)
	assert.Equal(t, fleet.MDMOperationTypeInstall, rows[0].OperationType)
	require.NotNil(t, rows[0].Status)
	assert.Equal(t, fleet.MDMDeliveryPending, *rows[0].Status)

	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, dev.ID)
	require.NoError(t, err)
	require.Len(t, cmds, 1, "the install command must be queued for the host's enrollment")

	// Idempotent: a second run queues nothing new, neither rows nor commands.
	require.NoError(t, ReconcileWindowsProfilesForEnrollingHost(ctx, ds, logger, host.UUID))
	rowsAgain, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, rowsAgain, 1, "second per-host reconcile run must be a no-op")
	cmdsAgain, err := ds.MDMWindowsGetPendingCommands(ctx, dev.ID)
	require.NoError(t, err)
	require.Len(t, cmdsAgain, 1, "second per-host reconcile run must not enqueue another command")
}

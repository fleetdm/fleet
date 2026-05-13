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

	// Sanity: the listing must surface the host+profile pair, otherwise the
	// cron has nothing to dispatch.
	toInstall, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
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

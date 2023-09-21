package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestMDMWindows(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestMDMWindowsEnrolledDevices", testMDMWindowsEnrolledDevice},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testMDMWindowsEnrolledDevice(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         uuid.New().String(),
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
	}

	err := ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	var ae fleet.AlreadyExistsError
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.ErrorAs(t, err, &ae)

	gotEnrolledDevice, err := ds.MDMWindowsGetEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.NoError(t, err)
	require.NotZero(t, gotEnrolledDevice.CreatedAt)
	require.Equal(t, enrolledDevice.MDMDeviceID, gotEnrolledDevice.MDMDeviceID)
	require.Equal(t, enrolledDevice.MDMHardwareID, gotEnrolledDevice.MDMHardwareID)

	err = ds.MDMWindowsDeleteEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.NoError(t, err)

	var nfe fleet.NotFoundError
	_, err = ds.MDMWindowsGetEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.ErrorAs(t, err, &nfe)

	err = ds.MDMWindowsDeleteEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.ErrorAs(t, err, &nfe)
}

func TestMDMWindowsDiskEncryption(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	checkBitLockerSummary := func(t *testing.T, teamID *uint, expected fleet.MDMWindowsBitLockerSummary) {
		bls, err := ds.GetMDMWindowsBitLockerSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, bls)
		require.Equal(t, expected, *bls)
	}

	checkListHostsFilterOSSettings := func(t *testing.T, teamID *uint, status fleet.OSSettingsStatus, expectedIDs []uint) {
		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status})
		require.NoError(t, err)
		require.Len(t, gotHosts, len(expectedIDs))
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}
	}

	checkListHostsFilterDiskEncryption := func(t *testing.T, teamID *uint, status fleet.DiskEncryptionStatus, expectedIDs []uint) {
		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsDiskEncryptionFilter: status})
		require.NoError(t, err)
		require.Len(t, gotHosts, len(expectedIDs))
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}
	}

	checkHostBitLockerStatus := func(t *testing.T, expected fleet.DiskEncryptionStatus, hostIDs []uint) {
		for _, id := range hostIDs {
			h, err := ds.Host(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, h)
			bls, err := ds.GetMDMWindowsBitLockerStatus(ctx, h)
			require.NoError(t, err)
			require.NotNil(t, bls)
			require.Equal(t, expected, *bls)
		}
	}

	type hostIDsByStatus map[fleet.DiskEncryptionStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, expected hostIDsByStatus) {
		for status, hostIDs := range expected {
			checkListHostsFilterDiskEncryption(t, teamID, status, hostIDs)
			checkHostBitLockerStatus(t, status, hostIDs)
		}

		checkBitLockerSummary(t, teamID, fleet.MDMWindowsBitLockerSummary{
			Verified:            uint(len(expected[fleet.DiskEncryptionVerified])),
			Verifying:           uint(len(expected[fleet.DiskEncryptionVerifying])),
			Failed:              uint(len(expected[fleet.DiskEncryptionFailed])),
			Enforcing:           uint(len(expected[fleet.DiskEncryptionEnforcing])),
			RemovingEnforcement: uint(len(expected[fleet.DiskEncryptionRemovingEnforcement])),
			ActionRequired:      uint(len(expected[fleet.DiskEncryptionActionRequired])),
		})

		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerified, expected[fleet.DiskEncryptionVerified])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerifying, expected[fleet.DiskEncryptionVerifying])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsFailed, expected[fleet.DiskEncryptionFailed])
		var expectedPending []uint
		expectedPending = append(expectedPending, expected[fleet.DiskEncryptionEnforcing]...)
		expectedPending = append(expectedPending, expected[fleet.DiskEncryptionRemovingEnforcement]...)
		expectedPending = append(expectedPending, expected[fleet.DiskEncryptionActionRequired]...)
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsPending, expectedPending)
	}

	// Create some hosts
	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		p := "windows"
		if i >= 5 {
			p = "darwin"
		}
		u := uuid.New().String()
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         &u,
			UUID:            u,
			Hostname:        u,
			Platform:        p,
		})
		require.NoError(t, err)
		require.NotNil(t, h)
		hosts = append(hosts, h)

		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet))
	}

	t.Run("Disk encryption disabled", func(t *testing.T) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		require.False(t, ac.MDM.EnableDiskEncryption.Value)

		checkExpected(t, nil, hostIDsByStatus{}) // no hosts are counted because disk encryption is not enabled
	})

	t.Run("Disk encryption enabled", func(t *testing.T) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
		ac, err = ds.AppConfig(ctx)
		require.NoError(t, err)
		require.True(t, ac.MDM.EnableDiskEncryption.Value)

		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}, // all windows hosts are counted
		})
	})

	t.Run("BitLocker verified status", func(t *testing.T) {
		// TODO: Update test to use methods to set windows disk encryption when they are implemented
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO host_disk_encryption_keys (host_id, decryptable, client_error) VALUES (?, ?, ?)`,
				hosts[0].ID,
				true,
				"")
			return err
		})
		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
		})
	})

	t.Run("BitLocker failed status", func(t *testing.T) {
		// TODO: Update test to use methods to set windows disk encryption when they are implemented
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO host_disk_encryption_keys (host_id, decryptable, client_error) VALUES (?, ?, ?)`,
				hosts[1].ID,
				false,
				"test-error")
			return err
		})

		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[2].ID, hosts[3].ID, hosts[4].ID},
		})
	})

	t.Run("BitLocker team filtering", func(t *testing.T) {
		// Test team filtering
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
		require.NoError(t, err)

		tm, err := ds.Team(ctx, team.ID)
		require.NoError(t, err)
		require.NotNil(t, tm)
		require.False(t, tm.Config.MDM.EnableDiskEncryption) // disk encryption is not enabled for team

		// Transfer hosts[2] to the team
		require.NoError(t, ds.AddHostsToTeam(ctx, &team.ID, []uint{hosts[2].ID}))

		// Check the summary for the team
		checkExpected(t, &team.ID, hostIDsByStatus{}) // disk encryption is not enabled for team so hosts[2] is not counted

		// Check the summary for no team
		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[3].ID, hosts[4].ID}, // hosts[2] is no longer included in the no team summary
		})

		// Enable disk encryption for the team
		tm.Config.MDM.EnableDiskEncryption = true
		tm, err = ds.SaveTeam(ctx, tm)
		require.NoError(t, err)
		require.NotNil(t, tm)
		require.True(t, tm.Config.MDM.EnableDiskEncryption)

		// Check the summary for the team
		checkExpected(t, &team.ID, hostIDsByStatus{
			fleet.DiskEncryptionEnforcing: []uint{hosts[2].ID}, // disk encryption is enabled for team so hosts[2] is counted
		})

		// Check the summary for no team (should be unchanged)
		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[3].ID, hosts[4].ID},
		})
	})

	t.Run("BitLocker Windows server excluded", func(t *testing.T) {
		require.NoError(t, ds.SetOrUpdateMDMData(ctx,
			hosts[3].ID,
			true, // set is_server to true for hosts[3]
			true, "https://example.com", false, fleet.WellKnownMDMFleet))

		// Check Windows servers not counted
		checkExpected(t, nil, hostIDsByStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[4].ID}, // hosts[3] is not counted
		})
	})

	t.Run("OS settings filters include Windows and macOS hosts", func(t *testing.T) {
		// Make macOS host fail disk encryption
		require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				HostUUID:          hosts[5].UUID,
				ProfileIdentifier: mobileconfig.FleetFileVaultPayloadIdentifier,
				ProfileName:       "Disk encryption",
				ProfileID:         1,
				CommandUUID:       uuid.New().String(),
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				Status:            &fleet.MDMAppleDeliveryFailed,
				Checksum:          []byte("checksum"),
			},
		}))

		// Check that BitLocker summary does not include macOS hosts
		checkBitLockerSummary(t, nil, fleet.MDMWindowsBitLockerSummary{
			Verified:            1,
			Verifying:           0,
			Failed:              1,
			Enforcing:           1,
			RemovingEnforcement: 0,
			ActionRequired:      0,
		})

		// Check that filtered lists do include macOS hosts
		checkListHostsFilterDiskEncryption(t, nil, fleet.DiskEncryptionFailed, []uint{hosts[1].ID, hosts[5].ID})
		checkListHostsFilterOSSettings(t, nil, fleet.OSSettingsFailed, []uint{hosts[1].ID, hosts[5].ID})
	})
}

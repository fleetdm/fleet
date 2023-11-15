package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
		{"TestMDMWindowsInsertCommandForHosts", testMDMWindowsInsertCommandForHosts},
		{"TestMDMWindowsGetPendingCommands", testMDMWindowsGetPendingCommands},
		{"TestMDMWindowsCommandResults", testMDMWindowsCommandResults},
		{"TestMDMWindowsConfigProfiles", testMDMWindowsConfigProfiles},
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

	// inserting a device again doesn't trow an error
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	gotEnrolledDevice, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)
	require.NotZero(t, gotEnrolledDevice.CreatedAt)
	require.Equal(t, enrolledDevice.MDMDeviceID, gotEnrolledDevice.MDMDeviceID)
	require.Equal(t, enrolledDevice.MDMHardwareID, gotEnrolledDevice.MDMHardwareID)

	err = ds.MDMWindowsDeleteEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.NoError(t, err)

	var nfe fleet.NotFoundError
	_, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.ErrorAs(t, err, &nfe)

	err = ds.MDMWindowsDeleteEnrolledDevice(ctx, enrolledDevice.MDMHardwareID)
	require.ErrorAs(t, err, &nfe)

	// Test using device ID instead of hardware ID
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	// inserting a device again doesn't trow an error
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	gotEnrolledDevice, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)
	require.NotZero(t, gotEnrolledDevice.CreatedAt)
	require.Equal(t, enrolledDevice.MDMDeviceID, gotEnrolledDevice.MDMDeviceID)
	require.Equal(t, enrolledDevice.MDMHardwareID, gotEnrolledDevice.MDMHardwareID)
	require.Empty(t, gotEnrolledDevice.HostUUID)

	err = ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)

	_, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
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

	checkMDMProfilesSummary := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMWindowsProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
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
		require.Len(t, gotHosts, len(expectedIDs), "status: %s", status)
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
			require.NotNil(t, bls.Status)
			require.Equal(t, expected, *bls.Status)
		}
	}

	type hostIDsByDEStatus map[fleet.DiskEncryptionStatus][]uint
	type hostIDsByProfileStatus map[fleet.MDMDeliveryStatus][]uint

	expectedProfilesFromDE := func(expectedDE hostIDsByDEStatus) hostIDsByProfileStatus {
		expectedProfiles := make(hostIDsByProfileStatus)
		expectedProfiles[fleet.MDMDeliveryPending] = []uint{}
		for status, hostIDs := range expectedDE {
			switch status {
			case fleet.DiskEncryptionVerified:
				expectedProfiles[fleet.MDMDeliveryVerified] = hostIDs
			case fleet.DiskEncryptionVerifying:
				expectedProfiles[fleet.MDMDeliveryVerifying] = hostIDs
			case fleet.DiskEncryptionFailed:
				expectedProfiles[fleet.MDMDeliveryFailed] = hostIDs
			case fleet.DiskEncryptionEnforcing, fleet.DiskEncryptionRemovingEnforcement, fleet.DiskEncryptionActionRequired:
				expectedProfiles[fleet.MDMDeliveryPending] = append(expectedProfiles[fleet.MDMDeliveryPending], hostIDs...)
			}
		}
		return expectedProfiles
	}

	checkExpected := func(t *testing.T, teamID *uint, expectedDE hostIDsByDEStatus, expectedProfiles ...hostIDsByProfileStatus) {
		var ep hostIDsByProfileStatus
		switch len(expectedProfiles) {
		case 1:
			ep = expectedProfiles[0]
		case 0:
			ep = expectedProfilesFromDE(expectedDE)
		default:
			require.FailNow(t, "expectedProfiles must have length 0 or 1")
		}

		for _, status := range []fleet.DiskEncryptionStatus{
			fleet.DiskEncryptionVerified,
			fleet.DiskEncryptionVerifying,
			fleet.DiskEncryptionFailed,
			fleet.DiskEncryptionEnforcing,
			fleet.DiskEncryptionRemovingEnforcement,
			fleet.DiskEncryptionActionRequired,
		} {
			hostIDs, ok := expectedDE[status]
			if !ok {
				hostIDs = []uint{}
			}
			checkListHostsFilterDiskEncryption(t, teamID, status, hostIDs)
			checkHostBitLockerStatus(t, status, hostIDs)
		}

		checkBitLockerSummary(t, teamID, fleet.MDMWindowsBitLockerSummary{
			Verified:            uint(len(expectedDE[fleet.DiskEncryptionVerified])),
			Verifying:           uint(len(expectedDE[fleet.DiskEncryptionVerifying])),
			Failed:              uint(len(expectedDE[fleet.DiskEncryptionFailed])),
			Enforcing:           uint(len(expectedDE[fleet.DiskEncryptionEnforcing])),
			RemovingEnforcement: uint(len(expectedDE[fleet.DiskEncryptionRemovingEnforcement])),
			ActionRequired:      uint(len(expectedDE[fleet.DiskEncryptionActionRequired])),
		})

		checkMDMProfilesSummary(t, teamID, fleet.MDMProfilesSummary{
			Pending:   uint(len(ep[fleet.MDMDeliveryPending])),
			Failed:    uint(len(ep[fleet.MDMDeliveryFailed])),
			Verifying: uint(len(ep[fleet.MDMDeliveryVerifying])),
			Verified:  uint(len(ep[fleet.MDMDeliveryVerified])),
		})

		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerified, expectedDE[fleet.DiskEncryptionVerified])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerifying, expectedDE[fleet.DiskEncryptionVerifying])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsFailed, expectedDE[fleet.DiskEncryptionFailed])
		var expectedPending []uint
		expectedPending = append(expectedPending, expectedDE[fleet.DiskEncryptionEnforcing]...)
		expectedPending = append(expectedPending, expectedDE[fleet.DiskEncryptionRemovingEnforcement]...)
		expectedPending = append(expectedPending, expectedDE[fleet.DiskEncryptionActionRequired]...)
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsPending, expectedPending)
	}

	updateHostDisks := func(t *testing.T, hostID uint, encrypted bool, updated_at time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `UPDATE host_disks SET encrypted = ?, updated_at = ? where host_id = ?`
			_, err := q.ExecContext(ctx, stmt, encrypted, updated_at, hostID)
			return err
		})
	}

	setKeyUpdatedAt := func(t *testing.T, hostID uint, keyUpdatedAt time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `UPDATE host_disk_encryption_keys SET updated_at = ? where host_id = ?`
			_, err := q.ExecContext(ctx, stmt, keyUpdatedAt, hostID)
			return err
		})
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, status) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, status)
			return err
		})
	}

	cleanupHostProfiles := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_windows_profiles`)
			return err
		})
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

		checkExpected(t, nil, hostIDsByDEStatus{}) // no hosts are counted because disk encryption is not enabled
	})

	t.Run("Disk encryption enabled", func(t *testing.T) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
		ac, err = ds.AppConfig(ctx)
		require.NoError(t, err)
		require.True(t, ac.MDM.EnableDiskEncryption.Value)

		t.Run("Bitlocker enforcing status", func(t *testing.T) {
			// all windows hosts are counted as enforcing because they have not reported any disk encryption status yet
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			require.NoError(t, ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "test-key", "", ptr.Bool(true)))
			checkExpected(t, nil, hostIDsByDEStatus{
				// status is still pending because hosts_disks hasn't been updated yet
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true))
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
				fleet.DiskEncryptionEnforcing: []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			cases := []struct {
				name                       string
				hostDisksEncrypted         bool
				reportedAfterKey           bool
				expectedWithinGracePeriod  fleet.DiskEncryptionStatus
				expectedOutsideGracePeriod fleet.DiskEncryptionStatus
			}{
				{
					name:                       "encrypted reported after key",
					hostDisksEncrypted:         true,
					reportedAfterKey:           true,
					expectedWithinGracePeriod:  fleet.DiskEncryptionVerified,
					expectedOutsideGracePeriod: fleet.DiskEncryptionVerified,
				},
				{
					name:                       "encrypted reported before key",
					hostDisksEncrypted:         true,
					reportedAfterKey:           false,
					expectedWithinGracePeriod:  fleet.DiskEncryptionVerifying,
					expectedOutsideGracePeriod: fleet.DiskEncryptionVerifying,
				},
				{
					name:                       "not encrypted reported before key",
					hostDisksEncrypted:         false,
					reportedAfterKey:           false,
					expectedWithinGracePeriod:  fleet.DiskEncryptionEnforcing,
					expectedOutsideGracePeriod: fleet.DiskEncryptionEnforcing,
				},
				{
					name:                       "not encrypted reported after key",
					hostDisksEncrypted:         false,
					reportedAfterKey:           true,
					expectedWithinGracePeriod:  fleet.DiskEncryptionVerifying,
					expectedOutsideGracePeriod: fleet.DiskEncryptionEnforcing,
				},
			}

			testHostID := hosts[0].ID
			otherWindowsHostIDs := []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}

			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					var keyUpdatedAt, hostDisksUpdatedAt time.Time

					t.Run("within grace period", func(t *testing.T) {
						expected := make(hostIDsByDEStatus)
						if c.expectedWithinGracePeriod == fleet.DiskEncryptionEnforcing {
							expected[fleet.DiskEncryptionEnforcing] = append([]uint{testHostID}, otherWindowsHostIDs...)
						} else {
							expected[c.expectedWithinGracePeriod] = []uint{testHostID}
							expected[fleet.DiskEncryptionEnforcing] = otherWindowsHostIDs
						}

						keyUpdatedAt = time.Now().Add(-10 * time.Minute)
						setKeyUpdatedAt(t, testHostID, keyUpdatedAt)

						if c.reportedAfterKey {
							hostDisksUpdatedAt = keyUpdatedAt.Add(5 * time.Minute)
						} else {
							hostDisksUpdatedAt = keyUpdatedAt.Add(-5 * time.Minute)
						}
						updateHostDisks(t, testHostID, c.hostDisksEncrypted, hostDisksUpdatedAt)

						checkExpected(t, nil, expected)
					})

					t.Run("outside grace period", func(t *testing.T) {
						expected := make(hostIDsByDEStatus)
						if c.expectedOutsideGracePeriod == fleet.DiskEncryptionEnforcing {
							expected[fleet.DiskEncryptionEnforcing] = append([]uint{testHostID}, otherWindowsHostIDs...)
						} else {
							expected[c.expectedOutsideGracePeriod] = []uint{testHostID}
							expected[fleet.DiskEncryptionEnforcing] = otherWindowsHostIDs
						}

						keyUpdatedAt = time.Now().Add(-2 * time.Hour)
						setKeyUpdatedAt(t, testHostID, keyUpdatedAt)

						if c.reportedAfterKey {
							hostDisksUpdatedAt = keyUpdatedAt.Add(5 * time.Minute)
						} else {
							hostDisksUpdatedAt = keyUpdatedAt.Add(-5 * time.Minute)
						}
						updateHostDisks(t, testHostID, c.hostDisksEncrypted, hostDisksUpdatedAt)

						checkExpected(t, nil, expected)
					})
				})
			}
		})

		// ensure hosts[0] is set to verified for the rest of the tests
		require.NoError(t, ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "test-key", "", ptr.Bool(true)))
		require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true))
		checkExpected(t, nil, hostIDsByDEStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
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

			expected := hostIDsByDEStatus{
				fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
				fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
				fleet.DiskEncryptionEnforcing: []uint{hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}

			checkExpected(t, nil, expected)

			// bitlocker failed status determines MDM aggregate status (profiles status is ignored)
			upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", fleet.MDMDeliveryFailed)
			checkExpected(t, nil, expected)
			upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", fleet.MDMDeliveryPending)
			checkExpected(t, nil, expected)
			upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", fleet.MDMDeliveryVerifying)
			checkExpected(t, nil, expected)
			upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", fleet.MDMDeliveryVerified)
			checkExpected(t, nil, expected)

			// profiles failed status determines MDM aggregate status (bitlocker status is ignored)
			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", fleet.MDMDeliveryFailed)
			expectedProfiles := expectedProfilesFromDE(expected)
			expectedProfiles[fleet.MDMDeliveryFailed] = append(expectedProfiles[fleet.MDMDeliveryFailed], hosts[0].ID)
			expectedProfiles[fleet.MDMDeliveryVerified] = []uint{}
			checkExpected(t, nil, expected, expectedProfiles)

			cleanupHostProfiles(t)
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
			checkExpected(t, &team.ID, hostIDsByDEStatus{}) // disk encryption is not enabled for team so hosts[2] is not counted

			// Check the summary for no team
			checkExpected(t, nil, hostIDsByDEStatus{
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
			checkExpected(t, &team.ID, hostIDsByDEStatus{
				fleet.DiskEncryptionEnforcing: []uint{hosts[2].ID}, // disk encryption is enabled for team so hosts[2] is counted
			})

			// Check the summary for no team (should be unchanged)
			checkExpected(t, nil, hostIDsByDEStatus{
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
			checkExpected(t, nil, hostIDsByDEStatus{
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
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            &fleet.MDMDeliveryFailed,
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
	})
}

func TestMDMWindowsProfilesSummary(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	checkMDMProfilesSummary := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMWindowsProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
	}

	type hostIDsByProfileStatus map[fleet.MDMDeliveryStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, ep hostIDsByProfileStatus) {
		checkMDMProfilesSummary(t, teamID, fleet.MDMProfilesSummary{
			Pending:   uint(len(ep[fleet.MDMDeliveryPending])),
			Failed:    uint(len(ep[fleet.MDMDeliveryFailed])),
			Verifying: uint(len(ep[fleet.MDMDeliveryVerifying])),
			Verified:  uint(len(ep[fleet.MDMDeliveryVerified])),
		})
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status *fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, status) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, status)
			if err != nil {
				return err
			}
			stmt = `UPDATE host_mdm_windows_profiles SET operation_type = ? WHERE host_uuid = ? AND profile_uuid = ?`
			_, err = q.ExecContext(ctx, stmt, fleet.MDMOperationTypeInstall, hostUUID, profUUID)
			return err
		})
	}

	cleanupHostProfiles := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_windows_profiles`)
			return err
		})
	}

	// updateHostDisks := func(t *testing.T, hostID uint, encrypted bool, updated_at time.Time) {
	// 	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
	// 		stmt := `UPDATE host_disks SET encrypted = ?, updated_at = ? where host_id = ?`
	// 		_, err := q.ExecContext(ctx, stmt, encrypted, updated_at, hostID)
	// 		return err
	// 	})
	// }

	// setKeyUpdatedAt := func(t *testing.T, hostID uint, keyUpdatedAt time.Time) {
	// 	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
	// 		stmt := `UPDATE host_disk_encryption_keys SET updated_at = ? where host_id = ?`
	// 		_, err := q.ExecContext(ctx, stmt, keyUpdatedAt, hostID)
	// 		return err
	// 	})
	// }

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

		expected := hostIDsByProfileStatus{}
		// no hosts are counted because no profiles and disk encryption is not enabled
		checkExpected(t, nil, expected)

		upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryPending)
		expected[fleet.MDMDeliveryPending] = []uint{hosts[0].ID}
		checkExpected(t, nil, expected)

		upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", &fleet.MDMDeliveryFailed)
		expected[fleet.MDMDeliveryFailed] = []uint{hosts[1].ID}
		checkExpected(t, nil, expected)

		upsertHostProfileStatus(t, hosts[2].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
		expected[fleet.MDMDeliveryVerifying] = []uint{hosts[2].ID}
		checkExpected(t, nil, expected)

		upsertHostProfileStatus(t, hosts[3].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
		expected[fleet.MDMDeliveryVerified] = []uint{hosts[3].ID}
		checkExpected(t, nil, expected)

		upsertHostProfileStatus(t, hosts[4].UUID, "some-windows-profile", nil)
		// nil status is treated as pending
		expected[fleet.MDMDeliveryPending] = append(expected[fleet.MDMDeliveryPending], hosts[4].ID)
		checkExpected(t, nil, expected)

		cleanupHostProfiles(t)
	})

	t.Run("Disk encryption enabled", func(t *testing.T) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
		ac, err = ds.AppConfig(ctx)
		require.NoError(t, err)
		require.True(t, ac.MDM.EnableDiskEncryption.Value)

		t.Run("bitlocker pending", func(t *testing.T) {
			expected := hostIDsByProfileStatus{
				fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}
			// all hosts are pending because no profiles and disk encryption is enabled
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryPending)
			// hosts[0] status pending because both profiles status and bitlocker status are pending
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[1].UUID, "some-windows-profile", &fleet.MDMDeliveryFailed)
			// status for hosts[1] now failed because any failed status determines MDM aggregate status
			expected[fleet.MDMDeliveryFailed] = []uint{hosts[1].ID}
			expected[fleet.MDMDeliveryPending] = []uint{hosts[0].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[2].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
			// status for hosts[2] still pending because bitlocker pending status takes precedence over
			// profiles verifying status
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[3].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
			// status for hosts[3] still pending because bitlocker pending status takes precedence over
			// profiles verified status
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[4].UUID, "some-windows-profile", nil)
			// hosts[0] status pending because bitlocker status is pending and nil profile status is
			// also treated as pending
			checkExpected(t, nil, expected)

			cleanupHostProfiles(t)
		})

		t.Run("bitlocker verified", func(t *testing.T) {
			expected := hostIDsByProfileStatus{
				fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}
			// all hosts are pending because no profiles and disk encryption is enabled
			checkExpected(t, nil, expected)

			require.NoError(t, ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "test-key", "", ptr.Bool(true)))
			// status is still pending because hosts_disks hasn't been updated yet
			checkExpected(t, nil, expected)

			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true))
			// status for hosts[0] now verified because bitlocker status is verified and host[0] has
			// no profiles
			checkExpected(t, nil, hostIDsByProfileStatus{
				fleet.MDMDeliveryVerified: []uint{hosts[0].ID},
				fleet.MDMDeliveryPending:  []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryFailed)
			// status for hosts[0] now failed because any failed status takes precedence
			expected = hostIDsByProfileStatus{
				fleet.MDMDeliveryFailed:   []uint{hosts[0].ID},
				fleet.MDMDeliveryVerified: []uint{},
				fleet.MDMDeliveryPending:  []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}
			checkExpected(t, nil, expected)

			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryPending)
			// status for hosts[0] now pending because profiles status pendiing takes precedence over bitlocker status verified
			expected[fleet.MDMDeliveryFailed] = []uint{}
			expected[fleet.MDMDeliveryPending] = []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}

			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
			// status for hosts[0] now verifying because profiles status verifying takes precedence over
			// bitlocker status verified
			expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
			expected[fleet.MDMDeliveryVerifying] = []uint{hosts[0].ID}

			upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
			// status for hosts[0] now verified because both profiles status and bitlocker status are verified
			expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
			expected[fleet.MDMDeliveryVerified] = []uint{hosts[0].ID}
			expected[fleet.MDMDeliveryVerifying] = []uint{}
			checkExpected(t, nil, expected)

			cleanupHostProfiles(t)
		})

		// updateHostDisks(t, hosts[0].ID, true, time.Now())
		// setKeyUpdatedAt(t, hosts[0].ID, time.Now().Add(-2*time.Hour))

		// turn off disk encryption
		ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
	})
}

func testMDMWindowsInsertCommandForHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	d1 := &fleet.MDMWindowsEnrolledDevice{
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
		HostUUID:               uuid.NewString(),
	}

	d2 := &fleet.MDMWindowsEnrolledDevice{
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
		HostUUID:               uuid.NewString(),
	}

	err := ds.MDMWindowsInsertEnrolledDevice(ctx, d1)
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, d1.HostUUID, d1.MDMDeviceID)
	require.NoError(t, err)

	err = ds.MDMWindowsInsertEnrolledDevice(ctx, d2)
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, d2.HostUUID, d2.MDMDeviceID)
	require.NoError(t, err)

	cmd := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri",
	}

	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{}, cmd)
	require.NoError(t, err)
	// no commands are enqueued nor created
	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d1.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, cmds)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, cmds)

	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d1.HostUUID, d2.HostUUID}, cmd)
	require.NoError(t, err)
	// command enqueued and created
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d1.MDMDeviceID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2.MDMDeviceID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	// commands can be added by device id as well
	cmd.CommandUUID = uuid.NewString()
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d1.MDMDeviceID, d2.MDMDeviceID}, cmd)
	require.NoError(t, err)
	// command enqueued and created
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d1.MDMDeviceID)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2.MDMDeviceID)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
}

func testMDMWindowsGetPendingCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	d := &fleet.MDMWindowsEnrolledDevice{
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
		HostUUID:               uuid.NewString(),
	}
	err := ds.MDMWindowsInsertEnrolledDevice(ctx, d)
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, d.HostUUID, d.MDMDeviceID)
	require.NoError(t, err)

	// device without commands
	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, cmds)

	// device with commands
	cmd := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri",
	}
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d.HostUUID}, cmd)
	require.NoError(t, err)

	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	// non-existent device
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, "fail")
	require.NoError(t, err)
	require.Empty(t, cmds)
}

func testMDMWindowsCommandResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	insertDB := func(t *testing.T, query string, args ...interface{}) (int64, error) {
		t.Helper()
		res, err := ds.writer(ctx).Exec(query, args...)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}

	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-win-host-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-win-host-uuid",
		Platform:      "windows",
	})
	require.NoError(t, err)

	dev := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "test-device-id",
		MDMHardwareID:          "test-hardware-id",
		MDMDeviceState:         "ds",
		MDMDeviceType:          "dt",
		MDMDeviceName:          "dn",
		MDMEnrollType:          "et",
		MDMEnrollUserID:        "euid",
		MDMEnrollProtoVersion:  "epv",
		MDMEnrollClientVersion: "ecv",
		MDMNotInOOBE:           false,
		HostUUID:               h.UUID,
	}

	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, dev))
	var enrollmentID uint
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &enrollmentID, `SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev.MDMDeviceID))
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE id = ?`, dev.HostUUID, enrollmentID)
	require.NoError(t, err)

	rawCmd := "some-command"
	cmdUUID := "some-uuid"
	cmdTarget := "some-target-loc-uri"
	_, err = insertDB(t, `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`, cmdUUID, rawCmd, cmdTarget)
	require.NoError(t, err)

	rawResponse := []byte("some-response")
	responseID, err := insertDB(t, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`, enrollmentID, rawResponse)
	require.NoError(t, err)

	rawResult := []byte("some-result")
	statusCode := "200"
	_, err = insertDB(t, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code) VALUES (?, ?, ?, ?, ?)`, enrollmentID, cmdUUID, rawResult, responseID, statusCode)
	require.NoError(t, err)

	p, err := ds.GetMDMCommandPlatform(ctx, cmdUUID)
	require.NoError(t, err)
	require.Equal(t, "windows", p)

	results, err := ds.GetMDMWindowsCommandResults(ctx, cmdUUID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, dev.HostUUID, results[0].HostUUID)
	require.Equal(t, cmdUUID, results[0].CommandUUID)
	require.Equal(t, rawResponse, results[0].Result)
	require.Equal(t, cmdTarget, results[0].RequestType)
	require.Equal(t, statusCode, results[0].Status)
	require.Empty(t, results[0].Hostname) // populated only at the service layer

	p, err = ds.GetMDMCommandPlatform(ctx, "unknown-cmd-uuid")
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, p)

	results, err = ds.GetMDMWindowsCommandResults(ctx, "unknown-cmd-uuid")
	require.NoError(t, err) // expect no error here, just no results
	require.Empty(t, results)
}

func testMDMWindowsConfigProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, name, team_id, syncml) VALUES (uuid(), 'abc', ?, ?)`, 0, "<SyncML></SyncML>")
		return err
	})

	var profUUID string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &profUUID,
			`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = 'abc'`)
	})

	_, err := ds.GetMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	prof, err := ds.GetMDMWindowsConfigProfile(ctx, profUUID)
	require.NoError(t, err)
	require.Equal(t, profUUID, prof.ProfileUUID)
	require.NotNil(t, prof.TeamID)
	require.Zero(t, *prof.TeamID)
	require.Equal(t, "abc", prof.Name)
	require.Equal(t, "<SyncML></SyncML>", string(prof.SyncML))
	require.NotZero(t, prof.CreatedAt)
	require.NotZero(t, prof.UpdatedAt)

	err = ds.DeleteMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	err = ds.DeleteMDMWindowsConfigProfile(ctx, profUUID)
	require.NoError(t, err)
}

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
		{"TestMDMWindowsProfileManagement", testMDMWindowsProfileManagement},
		{"TestBulkOperationsMDMWindowsHostProfiles", testBulkOperationsMDMWindowsHostProfiles},
		{"TestBulkOperationsMDMWindowsHostProfilesBatch2", testBulkOperationsMDMWindowsHostProfilesBatch2},
		{"TestBulkOperationsMDMWindowsHostProfilesBatch3", testBulkOperationsMDMWindowsHostProfilesBatch3},
		{"TestGetMDMWindowsProfilesContents", testGetMDMWindowsProfilesContents},
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

	type hostIDsByStatus map[fleet.DiskEncryptionStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, expected hostIDsByStatus) {
		for _, status := range []fleet.DiskEncryptionStatus{
			fleet.DiskEncryptionVerified,
			fleet.DiskEncryptionVerifying,
			fleet.DiskEncryptionFailed,
			fleet.DiskEncryptionEnforcing,
			fleet.DiskEncryptionRemovingEnforcement,
			fleet.DiskEncryptionActionRequired,
		} {
			hostIDs, ok := expected[status]
			if !ok {
				hostIDs = []uint{}
			}
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

		t.Run("Bitlocker enforcing status", func(t *testing.T) {
			// all windows hosts are counted as enforcing because they have not reported any disk encryption status yet
			checkExpected(t, nil, hostIDsByStatus{
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			require.NoError(t, ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "test-key", "", ptr.Bool(true)))
			checkExpected(t, nil, hostIDsByStatus{
				// status is still pending because hosts_disks hasn't been updated yet
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true))
			checkExpected(t, nil, hostIDsByStatus{
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
						expected := make(hostIDsByStatus)
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
						expected := make(hostIDsByStatus)
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
		checkExpected(t, nil, hostIDsByStatus{
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

func windowsEnroll(t *testing.T, ds fleet.Datastore, h *fleet.Host) {
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
		HostUUID:               h.UUID,
	}
	err := ds.MDMWindowsInsertEnrolledDevice(ctx, d1)
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, d1.HostUUID, d1.MDMDeviceID)
	require.NoError(t, err)
}
func testMDMWindowsProfileManagement(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	globalProfiles := []string{
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
	}

	// if there are no hosts, then no profiles need to be installed
	profiles, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profiles)

	host1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, host1)

	// non-Windows hosts shouldn't modify any of the results below
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-macos-host",
		OsqueryHostID: ptr.String("4824"),
		NodeKey:       ptr.String("4824"),
		UUID:          "test-macos-host",
		TeamID:        nil,
		Platform:      "macos",
	})
	require.NoError(t, err)

	// a windows host that's not MDM enrolled into Fleet shouldn't
	// modify any of the results below
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-non-mdm-host",
		OsqueryHostID: ptr.String("4825"),
		NodeKey:       ptr.String("4825"),
		UUID:          "test-non-mdm-host",
		TeamID:        nil,
		Platform:      "windows",
	})
	require.NoError(t, err)

	profilesMatch := func(t *testing.T, want []string, profs []*fleet.MDMWindowsProfilePayload) {
		got := []string{}
		for _, prof := range profs {
			got = append(got, prof.ProfileUUID)
		}
		require.ElementsMatch(t, want, got)
	}

	// global profiles to install on the newly added host
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	profilesMatch(t, globalProfiles, profiles)

	// add another host, it belongs to a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host2-name",
		OsqueryHostID: ptr.String("1338"),
		NodeKey:       ptr.String("1338"),
		UUID:          "test-uuid-2",
		TeamID:        &team.ID,
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, host2)

	// still the same profiles to assign as there are no profiles for team 1
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	profilesMatch(t, globalProfiles, profiles)

	// assign profiles to team 1
	teamProfiles := []string{
		InsertWindowsProfileForTest(t, ds, team.ID),
		InsertWindowsProfileForTest(t, ds, team.ID),
	}

	// new profiles, this time for the new host belonging to team 1
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	profilesMatch(t, append(globalProfiles, teamProfiles...), profiles)

	// add another global host
	host3, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host3-name",
		OsqueryHostID: ptr.String("1339"),
		NodeKey:       ptr.String("1339"),
		UUID:          "test-uuid-3",
		TeamID:        nil,
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, host3)

	// more profiles, this time for both global hosts and the team
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	profilesMatch(t, append(globalProfiles, append(globalProfiles, teamProfiles...)...), profiles)

	// cron runs and updates the status
	err = ds.BulkUpsertMDMWindowsHostProfiles(
		ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   globalProfiles[0],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   globalProfiles[0],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   globalProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   globalProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   globalProfiles[2],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   globalProfiles[2],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   teamProfiles[0],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-2",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   teamProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-2",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
		},
	)
	require.NoError(t, err)

	// no profiles left to install
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profiles)

	// no profiles to remove yet
	toRemove, err := ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// add host1 to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host1.ID})
	require.NoError(t, err)

	// profiles to be added for host1 are now related to the team
	profiles, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	profilesMatch(t, teamProfiles, profiles)

	// profiles to be removed includes host1's old profiles
	toRemove, err = ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	profilesMatch(t, globalProfiles, toRemove)
}

func testBulkOperationsMDMWindowsHostProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	profiles := []string{
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
	}

	getAllHostProfiles := func() []*fleet.MDMWindowsProfilePayload {
		var hostProfiles []*fleet.MDMWindowsProfilePayload
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid, status, operation_type FROM host_mdm_windows_profiles ORDER BY profile_name ASC`
			return sqlx.SelectContext(ctx, q, &hostProfiles, stmt)
		})
		return hostProfiles
	}

	// empty payloads is a noop
	err := ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{})
	require.NoError(t, err)
	require.Empty(t, getAllHostProfiles())

	// valid payload inserts new records
	err = ds.BulkUpsertMDMWindowsHostProfiles(
		ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profiles[0],
				ProfileName:   "A",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[1],
				ProfileName:   "B",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[2],
				ProfileName:   "C",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[3],
				ProfileName:   "D",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[4],
				ProfileName:   "E",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
		},
	)
	require.NoError(t, err)
	hostsProfs := getAllHostProfiles()
	require.Len(t, hostsProfs, 5)
	for i, p := range hostsProfs {
		require.Equal(t, profiles[i], p.ProfileUUID)
		require.Equal(t, fleet.MDMOperationTypeInstall, p.OperationType)
		require.Equal(t, &fleet.MDMDeliveryVerifying, p.Status)
	}

	// valid payload updates existing records
	err = ds.BulkUpsertMDMWindowsHostProfiles(
		ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profiles[0],
				ProfileName:   "A",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[1],
				ProfileName:   "B",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[2],
				ProfileName:   "C",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[3],
				ProfileName:   "D",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
			{
				ProfileUUID:   profiles[4],
				ProfileName:   "E",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
			},
		},
	)
	require.NoError(t, err)
	hostsProfs = getAllHostProfiles()
	require.Len(t, hostsProfs, 5)
	for i, p := range hostsProfs {
		require.Equal(t, profiles[i], p.ProfileUUID)
		require.Equal(t, fleet.MDMOperationTypeInstall, p.OperationType)
		require.Equal(t, &fleet.MDMDeliveryVerified, p.Status)
	}

	// empty payload
	err = ds.BulkDeleteMDMWindowsHostsConfigProfiles(ctx, []*fleet.MDMWindowsProfilePayload{})
	require.NoError(t, err)
	hostsProfs = getAllHostProfiles()
	require.Len(t, hostsProfs, 5)

	// partial deletes
	err = ds.BulkDeleteMDMWindowsHostsConfigProfiles(ctx, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: profiles[0],
			HostUUID:    "test-uuid-1",
		},
		{
			ProfileUUID: profiles[1],
			HostUUID:    "test-uuid-3",
		},
		{
			ProfileUUID: profiles[2],
			HostUUID:    "test-uuid-1",
		},
	})
	require.NoError(t, err)
	hostsProfs = getAllHostProfiles()
	require.Len(t, hostsProfs, 2)

	// full deletes
	err = ds.BulkDeleteMDMWindowsHostsConfigProfiles(ctx, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: profiles[0],
			HostUUID:    "test-uuid-1",
		},
		{
			ProfileUUID: profiles[1],
			HostUUID:    "test-uuid-3",
		},
		{
			ProfileUUID: profiles[2],
			HostUUID:    "test-uuid-1",
		},
		{
			ProfileUUID: profiles[3],
			HostUUID:    "test-uuid-1",
		},
		{
			ProfileUUID: profiles[4],
			HostUUID:    "test-uuid-1",
		},
	})
	require.NoError(t, err)
	hostsProfs = getAllHostProfiles()
	require.Len(t, hostsProfs, 0)
}

func testBulkOperationsMDMWindowsHostProfilesBatch2(t *testing.T, ds *Datastore) {
	testUpsertMDMDesiredProfilesBatchSize = 2
	testDeleteMDMProfilesBatchSize = 2
	t.Cleanup(func() {
		testUpsertMDMDesiredProfilesBatchSize = 0
		testDeleteMDMProfilesBatchSize = 0
	})
	testBulkOperationsMDMWindowsHostProfiles(t, ds)
}

func testBulkOperationsMDMWindowsHostProfilesBatch3(t *testing.T, ds *Datastore) {
	testUpsertMDMDesiredProfilesBatchSize = 3
	testDeleteMDMProfilesBatchSize = 3
	t.Cleanup(func() {
		testUpsertMDMDesiredProfilesBatchSize = 0
		testDeleteMDMProfilesBatchSize = 0
	})
	testBulkOperationsMDMWindowsHostProfiles(t, ds)
}

func testGetMDMWindowsProfilesContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	profileUUIDs := []string{
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
	}

	cases := []struct {
		ids  []string
		want map[string][]byte
	}{
		{[]string{}, nil},
		{nil, nil},
		{[]string{profileUUIDs[0]}, map[string][]byte{profileUUIDs[0]: generateDummyWindowsProfile(profileUUIDs[0])}},
		{
			[]string{profileUUIDs[0], profileUUIDs[1], profileUUIDs[2]},
			map[string][]byte{
				profileUUIDs[0]: generateDummyWindowsProfile(profileUUIDs[0]),
				profileUUIDs[1]: generateDummyWindowsProfile(profileUUIDs[1]),
				profileUUIDs[2]: generateDummyWindowsProfile(profileUUIDs[2]),
			},
		},
	}

	for _, c := range cases {
		out, err := ds.GetMDMWindowsProfilesContents(ctx, c.ids)
		require.NoError(t, err)
		require.Equal(t, c.want, out)
	}
}

func testMDMWindowsConfigProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a couple Windows profiles for no-team (nil and 0 means no team)
	profA, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: nil, SyncML: []byte("<Replace></Replace>")})
	require.NoError(t, err)
	require.NotEmpty(t, profA.ProfileUUID)
	profB, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: ptr.Uint(0), SyncML: []byte("<Replace></Replace>")})
	require.NoError(t, err)
	require.NotEmpty(t, profB.ProfileUUID)
	// create an Apple profile for no-team
	profC, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("c", "c", 0))
	require.NoError(t, err)
	require.NotZero(t, profC.ProfileID)

	// create the same name for team 1 as Windows profile
	profATm, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")})
	require.NoError(t, err)
	require.NotEmpty(t, profATm.ProfileUUID)
	require.NotNil(t, profATm.TeamID)
	require.Equal(t, uint(1), *profATm.TeamID)
	// create the same B profile for team 1 as Apple profile
	profBTm, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 1))
	require.NoError(t, err)
	require.NotZero(t, profBTm.ProfileID)

	var existsErr *existsError
	// create a duplicate of Windows for no-team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: nil, SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Apple for no-team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "c", TeamID: nil, SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Windows for team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Apple for team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with an Apple profile for no-team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 0))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with an Apple profile for team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 1))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	_, err = ds.GetMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	prof, err := ds.GetMDMWindowsConfigProfile(ctx, profA.ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, profA.ProfileUUID, prof.ProfileUUID)
	require.NotNil(t, prof.TeamID)
	require.Zero(t, *prof.TeamID)
	require.Equal(t, "a", prof.Name)
	require.Equal(t, "<Replace></Replace>", string(prof.SyncML))
	require.NotZero(t, prof.CreatedAt)
	require.NotZero(t, prof.UpdatedAt)

	err = ds.DeleteMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	err = ds.DeleteMDMWindowsConfigProfile(ctx, profA.ProfileUUID)
	require.NoError(t, err)
}

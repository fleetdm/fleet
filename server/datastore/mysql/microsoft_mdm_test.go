package mysql

import (
	"context"
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"database/sql"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
		{"TestMDMWindowsInsertCommandAndUpsertHostProfilesForHosts", testMDMWindowsInsertCommandAndUpsertHostProfilesForHosts},
		{"TestMDMWindowsGetPendingCommands", testMDMWindowsGetPendingCommands},
		{"TestMDMWindowsCommandResults", testMDMWindowsCommandResults},
		{"TestMDMWindowsCommandResultsWithPendingResult", testMDMWindowsCommandResultsWithPendingResult},
		{"TestMDMWindowsProfileManagement", testMDMWindowsProfileManagement},
		{"TestBulkOperationsMDMWindowsHostProfiles", testBulkOperationsMDMWindowsHostProfiles},
		{"TestBulkOperationsMDMWindowsHostProfilesBatch2", testBulkOperationsMDMWindowsHostProfilesBatch2},
		{"TestBulkOperationsMDMWindowsHostProfilesBatch3", testBulkOperationsMDMWindowsHostProfilesBatch3},
		{"TestGetMDMWindowsProfilesContents", testGetMDMWindowsProfilesContents},
		{"TestMDMWindowsConfigProfiles", testMDMWindowsConfigProfiles},
		{"TestMDMWindowsConfigProfilesWithFleetVars", testMDMWindowsConfigProfilesWithFleetVars},
		{"TestSetOrReplaceMDMWindowsConfigProfile", testSetOrReplaceMDMWindowsConfigProfile},
		{"TestMDMWindowsDiskEncryption", testMDMWindowsDiskEncryption},
		{"TestMDMWindowsProfilesSummary", testMDMWindowsProfilesSummary},
		{"TestMDMWindowsProfilesSummaryEnumeration", testMDMWindowsProfilesSummaryEnumeration},
		{"TestBatchSetMDMWindowsProfiles", testBatchSetMDMWindowsProfiles},
		{"TestMDMWindowsProfileLabels", testMDMWindowsProfileLabels},
		{"TestMDMWindowsSaveResponse", testSaveResponse},
		{"TestSetMDMWindowsProfilesWithVariables", testSetMDMWindowsProfilesWithVariables},
		{"TestWindowsMDMManagedSCEPCertificates", testWindowsMDMManagedSCEPCertificates},
		{"TestGetWindowsMDMCommandsForResending", testGetWindowsMDMCommandsForResending},
		{"TestResendWindowsMDMCommand", testResendWindowsMDMCommand},
		{"TestDeleteProfileLocURIProtection", testDeleteProfileLocURIProtection},
		{"TestEditProfileDeletesRemovedLocURIs", testEditProfileDeletesRemovedLocURIs},
		{"TestBatchDeleteMultipleWindowsProfiles", testBatchDeleteMultipleWindowsProfiles},
		{"TestMDMWindowsUnenrollCleansUpProfiles", testMDMWindowsUnenrollCleansUpProfiles},
		{"TestMDMWindowsProfilesToRemoveSkipsOrphanedHosts", testMDMWindowsProfilesToRemoveSkipsOrphanedHosts},
		{"TestMDMWindowsInsertCommandSkipsUnenrolledHosts", testMDMWindowsInsertCommandSkipsUnenrolledHosts},
		{"TestCleanupWindowsMDMCommandQueue", testCleanupWindowsMDMCommandQueue},
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
	require.Equal(t, fleet.WindowsMDMAwaitingConfigurationNone, gotEnrolledDevice.AwaitingConfiguration)
	require.Nil(t, gotEnrolledDevice.AwaitingConfigurationAt)

	err = ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, enrolledDevice.MDMHardwareID)
	require.NoError(t, err)

	var nfe fleet.NotFoundError
	_, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.ErrorAs(t, err, &nfe)

	err = ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, enrolledDevice.MDMHardwareID)
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

	err = ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(ctx, enrolledDevice.MDMHardwareID)
	require.ErrorAs(t, err, &nfe)

	// Test that awaiting configuration is persisted and updated on upsert.
	now := time.Now().UTC()
	enrolledDevice.AwaitingConfiguration = fleet.WindowsMDMAwaitingConfigurationPending
	enrolledDevice.AwaitingConfigurationAt = &now
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	gotEnrolledDevice, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)
	require.Equal(t, fleet.WindowsMDMAwaitingConfigurationPending, gotEnrolledDevice.AwaitingConfiguration)
	require.NotNil(t, gotEnrolledDevice.AwaitingConfigurationAt)

	// Re-enroll clears awaiting configuration via upsert.
	enrolledDevice.AwaitingConfiguration = fleet.WindowsMDMAwaitingConfigurationNone
	enrolledDevice.AwaitingConfigurationAt = nil
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	gotEnrolledDevice, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)
	require.Equal(t, fleet.WindowsMDMAwaitingConfigurationNone, gotEnrolledDevice.AwaitingConfiguration)
	require.Nil(t, gotEnrolledDevice.AwaitingConfigurationAt)
}

func testMDMWindowsDiskEncryption(t *testing.T, ds *Datastore) {
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

		count, err := ds.CountHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsDiskEncryptionFilter: status})
		require.NoError(t, err)
		require.Equal(t, len(expectedIDs), count, fmt.Sprintf("status: %s", status))
	}

	checkHostBitLockerStatus := func(t *testing.T, expected fleet.DiskEncryptionStatus, hostIDs []uint) {
		for _, id := range hostIDs {
			h, err := ds.Host(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, h)
			mdmInfo, err := ds.GetHostMDM(ctx, id)
			require.NoError(t, err)
			require.NotNil(t, mdmInfo)
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

		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerified, ep[fleet.MDMDeliveryVerified])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerifying, ep[fleet.MDMDeliveryVerifying])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsFailed, ep[fleet.MDMDeliveryFailed])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsPending, ep[fleet.MDMDeliveryPending])
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

	setProtectionStatus := func(t *testing.T, hostID uint, protectionStatus *int) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `UPDATE host_disks SET bitlocker_protection_status = ? where host_id = ?`
			_, err := q.ExecContext(ctx, stmt, protectionStatus, hostID)
			return err
		})
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			// Generate a command UUID for the profile
			commandUUID := "cmd-" + profUUID
			stmt := `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, status, command_uuid) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, commandUUID, status)
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

		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))

		if p == "darwin" {
			nanoEnroll(t, ds, h, false)
		} else {
			windowsEnroll(t, ds, h)
		}
	}

	t.Run("Disk encryption disabled", func(t *testing.T) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
		ac, err = ds.AppConfig(ctx)
		require.NoError(t, err)
		require.False(t, ac.MDM.EnableDiskEncryption.Value)

		cleanupHostProfiles(t)

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

		t.Run("Bitlocker enforcing-verifying-verified", func(t *testing.T) {
			// all windows hosts are counted as enforcing because they have not reported any disk encryption status yet
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", ptr.Bool(true))
			require.NoError(t, err)
			checkExpected(t, nil, hostIDsByDEStatus{
				// status is still pending because hosts_disks hasn't been updated yet
				fleet.DiskEncryptionEnforcing: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
			})

			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
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
		_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", ptr.Bool(true))
		require.NoError(t, err)
		require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
		checkExpected(t, nil, hostIDsByDEStatus{
			fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
			fleet.DiskEncryptionEnforcing: []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
		})

		t.Run("BitLocker failed status", func(t *testing.T) {
			// set hosts[1] to failed
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[1], "", "test-error", ptr.Bool(false))
			require.NoError(t, err)

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

		t.Run("BitLocker profile status with PIN required", func(t *testing.T) {
			// Turn on Bitlocker requirement
			ac.MDM.RequireBitLockerPIN = optjson.SetBool(true)
			require.NoError(t, ds.SaveAppConfig(ctx, ac))
			ac, err = ds.AppConfig(ctx)
			require.NoError(t, err)
			require.True(t, ac.MDM.RequireBitLockerPIN.Value)

			// Expect that the host that would be "verified"
			// is now in "action required" status.
			// This will also verify that when filtering by profile status,
			// the "verified" host is now counted as "pending".
			expected := hostIDsByDEStatus{
				fleet.DiskEncryptionActionRequired: []uint{hosts[0].ID},
				fleet.DiskEncryptionFailed:         []uint{hosts[1].ID},
				fleet.DiskEncryptionEnforcing:      []uint{hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}

			checkExpected(t, nil, expected)

			// Set the "tpm_pin_set" to true for the host that would be "verified"
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `UPDATE host_disks SET tpm_pin_set = true WHERE host_id = ?`, hosts[0].ID)
				return err
			})

			expected = hostIDsByDEStatus{
				fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
				fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
				fleet.DiskEncryptionEnforcing: []uint{hosts[2].ID, hosts[3].ID, hosts[4].ID},
			}

			checkExpected(t, nil, expected)

			// Reset the "tpm_pin_set" to false for the host.
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `UPDATE host_disks SET tpm_pin_set = false WHERE host_id = ?`, hosts[0].ID)
				return err
			})

			// Reset "RequireBitLockerPIN" to false
			ac.MDM.RequireBitLockerPIN = optjson.SetBool(false)
			require.NoError(t, ds.SaveAppConfig(ctx, ac))
			ac, err = ds.AppConfig(ctx)
			require.NoError(t, err)
			require.False(t, ac.MDM.RequireBitLockerPIN.Value)
		})

		t.Run("BitLocker team filtering", func(t *testing.T) {
			// Test team filtering
			team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
			require.NoError(t, err)

			tm, err := ds.TeamWithExtras(ctx, team.ID)
			require.NoError(t, err)
			require.NotNil(t, tm)
			require.False(t, tm.Config.MDM.EnableDiskEncryption) // disk encryption is not enabled for team

			// Transfer hosts[2] to the team
			require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{hosts[2].ID})))

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
				true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))

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
					ProfileUUID:       "a" + uuid.NewString(),
					CommandUUID:       uuid.New().String(),
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            &fleet.MDMDeliveryFailed,
					Checksum:          []byte("checksum"),
					Scope:             fleet.PayloadScopeSystem,
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

			// delete the macOS host profile
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_identifier = ?`, hosts[5].UUID, mobileconfig.FleetFileVaultPayloadIdentifier)
				return err
			})
		})

		t.Run("BitLocker host disks must update to transition from Verifying to Verified", func(t *testing.T) {
			// we'll use hosts[4] as the target for this test
			targetHost := hosts[4]

			// confirm our initial state is as expected from previous tests
			// hosts[2] is was transferred to a team and is not counted
			// hosts[3] is a Windows server and is not counted
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
				fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
				fleet.DiskEncryptionEnforcing: []uint{targetHost.ID}, // targetHost is initially enforcing
			})

			// simulate targetHost previously reported encrypted for disk encryption detail query
			// results
			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, targetHost.ID, true, nil))
			// manualy update host_disks for targetHost to encrypted and ensure updated_at
			// timestamp is in the past
			updateHostDisks(t, targetHost.ID, true, time.Now().Add(-3*time.Hour))

			// simulate targetHost reporting disk encryption key
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, targetHost, "test-key", "", ptr.Bool(true))
			require.NoError(t, err)

			// check that targetHost is now counted as verifying (not verified because host_disks still needs to be updated)
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
				fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
				fleet.DiskEncryptionVerifying: []uint{targetHost.ID},
			})

			// simulate targetHost reporting detail query results for disk encryption
			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, targetHost.ID, true, nil))
			// status for targetHost now verified because SetOrUpdateHostDisksEncryption always sets host_disks.updated_at
			// to the current timestamp even if the `encrypted` value hasn't changed
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionVerified: []uint{hosts[0].ID, targetHost.ID},
				fleet.DiskEncryptionFailed:   []uint{hosts[1].ID},
			})
		})

		t.Run("BitLocker protection status", func(t *testing.T) {
			targetHost := hosts[4]

			// Explicitly set up state so this subtest can run standalone.
			// hosts[0]: encrypted, key escrowed, verified
			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", new(true))
			require.NoError(t, err)
			// targetHost: encrypted, key escrowed, verified
			require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, targetHost.ID, true, nil))
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, targetHost, "test-key", "", new(true))
			require.NoError(t, err)
			setProtectionStatus(t, targetHost.ID, nil)
			// hosts[1]: failed (has client error)
			_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[1], "", "test-error", new(false))
			require.NoError(t, err)

			// baseline check
			checkExpected(t, nil, hostIDsByDEStatus{
				fleet.DiskEncryptionVerified: []uint{hosts[0].ID, targetHost.ID},
				fleet.DiskEncryptionFailed:   []uint{hosts[1].ID},
			})

			t.Run("protection_status=1 stays verified", func(t *testing.T) {
				setProtectionStatus(t, targetHost.ID, new(fleet.BitLockerProtectionStatusOn))
				checkExpected(t, nil, hostIDsByDEStatus{
					fleet.DiskEncryptionVerified: []uint{hosts[0].ID, targetHost.ID},
					fleet.DiskEncryptionFailed:   []uint{hosts[1].ID},
				})
			})

			t.Run("protection_status=0 becomes action_required", func(t *testing.T) {
				setProtectionStatus(t, targetHost.ID, new(fleet.BitLockerProtectionStatusOff))
				checkExpected(t, nil, hostIDsByDEStatus{
					fleet.DiskEncryptionVerified:       []uint{hosts[0].ID},
					fleet.DiskEncryptionActionRequired: []uint{targetHost.ID},
					fleet.DiskEncryptionFailed:         []uint{hosts[1].ID},
				})
			})

			t.Run("protection_status=NULL treated as on (backward compat)", func(t *testing.T) {
				setProtectionStatus(t, targetHost.ID, nil)
				checkExpected(t, nil, hostIDsByDEStatus{
					fleet.DiskEncryptionVerified: []uint{hosts[0].ID, targetHost.ID},
					fleet.DiskEncryptionFailed:   []uint{hosts[1].ID},
				})
			})

			t.Run("protection off during encryption in progress is verifying not action_required", func(t *testing.T) {
				// Simulate: orbit escrowed key, encryption in progress, osquery reports
				// encrypted=false and protection_status=0. This is normal during encryption.
				_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, targetHost, "test-key", "", new(true))
				require.NoError(t, err)
				updateHostDisks(t, targetHost.ID, false, time.Now())
				setProtectionStatus(t, targetHost.ID, new(fleet.BitLockerProtectionStatusOff))

				checkExpected(t, nil, hostIDsByDEStatus{
					fleet.DiskEncryptionVerified:  []uint{hosts[0].ID},
					fleet.DiskEncryptionVerifying: []uint{targetHost.ID},
					fleet.DiskEncryptionFailed:    []uint{hosts[1].ID},
				})
			})

			t.Run("action_required detail message for protection off", func(t *testing.T) {
				// Restore targetHost to encrypted + protection off
				require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, targetHost.ID, true, new(fleet.BitLockerProtectionStatusOff)))
				h, err := ds.Host(ctx, targetHost.ID)
				require.NoError(t, err)
				bls, err := ds.GetMDMWindowsBitLockerStatus(ctx, h)
				require.NoError(t, err)
				require.NotNil(t, bls)
				require.NotNil(t, bls.Status)
				require.Equal(t, fleet.DiskEncryptionActionRequired, *bls.Status)
				require.Contains(t, bls.Detail, "BitLocker protection is off")
			})
		})
	})
}

func testMDMWindowsProfilesSummary(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	checkMDMProfilesSummary := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMWindowsProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
	}

	checkListHostsFilterOSSettings := func(t *testing.T, teamID *uint, status fleet.OSSettingsStatus, expectedIDs []uint) {
		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status})
		require.NoError(t, err)
		if len(expectedIDs) != len(gotHosts) {
			gotIDs := make([]uint, len(gotHosts))
			for i, h := range gotHosts {
				gotIDs[i] = h.ID
			}
			require.Len(t, gotHosts, len(expectedIDs), fmt.Sprintf("status: %s expected: %v got: %v", status, expectedIDs, gotIDs))

		}
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}

		count, err := ds.CountHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status})
		require.NoError(t, err)
		require.Equal(t, len(expectedIDs), count, "status: %s", status)
	}

	type hostIDsByProfileStatus map[fleet.MDMDeliveryStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, ep hostIDsByProfileStatus) {
		checkMDMProfilesSummary(t, teamID, fleet.MDMProfilesSummary{
			Pending:   uint(len(ep[fleet.MDMDeliveryPending])),
			Failed:    uint(len(ep[fleet.MDMDeliveryFailed])),
			Verifying: uint(len(ep[fleet.MDMDeliveryVerifying])),
			Verified:  uint(len(ep[fleet.MDMDeliveryVerified])),
		})

		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerified, ep[fleet.MDMDeliveryVerified])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsVerifying, ep[fleet.MDMDeliveryVerifying])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsFailed, ep[fleet.MDMDeliveryFailed])
		checkListHostsFilterOSSettings(t, teamID, fleet.OSSettingsPending, ep[fleet.MDMDeliveryPending])
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status *fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			// Generate a command UUID for the profile
			commandUUID := "cmd-" + profUUID
			stmt := `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, status, command_uuid) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, commandUUID, status)
			if err != nil {
				return err
			}
			stmt = `UPDATE host_mdm_windows_profiles SET operation_type = ? WHERE host_uuid = ? AND profile_uuid = ?`
			_, err = q.ExecContext(ctx, stmt, fleet.MDMOperationTypeInstall, hostUUID, profUUID)
			return err
		})
	}

	cleanupTables := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_windows_profiles`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_disk_encryption_keys`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_disks`)
			return err
		})
	}

	updateHostDisks := func(t *testing.T, hostID uint, encrypted bool, updated_at time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `UPDATE host_disks SET encrypted = ?, updated_at = ? where host_id = ?`
			_, err := q.ExecContext(ctx, stmt, encrypted, updated_at, hostID)
			return err
		})
	}

	// Create some hosts
	var hosts []*fleet.Host
	uuidToDeviceID := map[string]string{}
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

		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
		if p == "windows" {
			uuidToDeviceID[h.UUID] = windowsEnroll(t, ds, h)
		}
	}

	t.Run("profiles summary accounts for bitlocker status", func(t *testing.T) {
		t.Run("bitlocker disabled", func(t *testing.T) {
			ac, err := ds.AppConfig(ctx)
			require.NoError(t, err)
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			require.NoError(t, ds.SaveAppConfig(ctx, ac))
			ac, err = ds.AppConfig(ctx)
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

			cleanupTables(t)
		})

		t.Run("bitlocker enabled", func(t *testing.T) {
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

				cleanupTables(t)
			})

			t.Run("bitlocker verifying", func(t *testing.T) {
				expected := hostIDsByProfileStatus{
					fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				}
				// all hosts are pending because no profiles and disk encryption is enabled
				checkExpected(t, nil, expected)

				require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
				_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", ptr.Bool(true))
				require.NoError(t, err)
				// simulate bitlocker verifying status by ensuring host_disks updated at timestamp is before host_disk_encryption_key
				updateHostDisks(t, hosts[0].ID, true, time.Now().Add(-10*time.Minute))
				// status for hosts[0] now verifying because bitlocker status is verifying and host[0] has
				// no profiles
				expected[fleet.MDMDeliveryVerifying] = []uint{hosts[0].ID}
				expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryFailed)
				// status for hosts[0] now failed because any failed status takes precedence
				expected = hostIDsByProfileStatus{
					fleet.MDMDeliveryFailed:    []uint{hosts[0].ID},
					fleet.MDMDeliveryVerifying: []uint{},
					fleet.MDMDeliveryPending:   []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryPending)
				// status for hosts[0] now pending because profiles status pendiing takes precedence over bitlocker status verifying
				expected[fleet.MDMDeliveryFailed] = []uint{}
				expected[fleet.MDMDeliveryPending] = []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
				// status for hosts[0] now verifying because both profiles status and bitlocker status are verifying
				expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				expected[fleet.MDMDeliveryVerifying] = []uint{hosts[0].ID}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
				// status for hosts[0] still verifying because bitlocker status verifying takes
				// precedence over profile status verified
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", nil)
				// status for hosts[0] now pending because nil profile status is treated as pending and
				// pending status takes precedence over bitlocker status verifying
				expected[fleet.MDMDeliveryVerifying] = []uint{}
				expected[fleet.MDMDeliveryPending] = []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				checkExpected(t, nil, expected)

				cleanupTables(t)
			})

			t.Run("bitlocker verified", func(t *testing.T) {
				expected := hostIDsByProfileStatus{
					fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				}
				// all hosts are pending because no profiles and disk encryption is enabled
				checkExpected(t, nil, expected)

				_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", ptr.Bool(true))
				require.NoError(t, err)
				// status is still pending because hosts_disks hasn't been updated yet
				checkExpected(t, nil, expected)

				require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
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
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
				// status for hosts[0] now verifying because profiles status verifying takes precedence over
				// bitlocker status verified
				expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				expected[fleet.MDMDeliveryVerifying] = []uint{hosts[0].ID}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
				// status for hosts[0] now verified because both profiles status and bitlocker status are verified
				expected[fleet.MDMDeliveryPending] = []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID}
				expected[fleet.MDMDeliveryVerified] = []uint{hosts[0].ID}
				expected[fleet.MDMDeliveryVerifying] = []uint{}
				checkExpected(t, nil, expected)

				cleanupTables(t)
			})

			t.Run("BitLocker host disks must update to transition from Verifying to Verified", func(t *testing.T) {
				// all hosts are pending because no profiles and disk encryption is enabled
				checkExpected(t, nil, hostIDsByProfileStatus{
					fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				})

				// simulate host already has encrypted disks
				require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
				// manualy update host_disks for hosts[0] to encrypted and ensure updated_at
				// timestamp is in the past
				updateHostDisks(t, hosts[0].ID, true, time.Now().Add(-2*time.Hour))

				_, err := ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "test-key", "", ptr.Bool(true))
				require.NoError(t, err)
				// status is verifying because hosts_disks hasn't been updated again
				checkExpected(t, nil, hostIDsByProfileStatus{
					fleet.MDMDeliveryVerifying: []uint{hosts[0].ID},
					fleet.MDMDeliveryPending:   []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				})

				require.NoError(t, ds.SetOrUpdateHostDisksEncryption(ctx, hosts[0].ID, true, nil))
				// status for hosts[0] now verified because SetOrUpdateHostDisksEncryption always sets host_disks.updated_at
				// to the current timestamp even if the `encrypted` value hasn't changed
				checkExpected(t, nil, hostIDsByProfileStatus{
					fleet.MDMDeliveryVerified: []uint{hosts[0].ID},
					fleet.MDMDeliveryPending:  []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				})

				cleanupTables(t)
			})

			t.Run("bitlocker failed", func(t *testing.T) {
				expected := hostIDsByProfileStatus{
					fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				}
				// all hosts are pending because no profiles and disk encryption is enabled
				checkExpected(t, nil, expected)

				_, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0], "", "some-bitlocker-error", nil)
				require.NoError(t, err)
				// status for hosts[0] now failed because any failed status takes precedence
				expected = hostIDsByProfileStatus{
					fleet.MDMDeliveryFailed:  []uint{hosts[0].ID},
					fleet.MDMDeliveryPending: []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
				}
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryFailed)
				// status for hosts[0] still failed because any failed status takes precedence
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryPending)
				// status for hosts[0] still failed because bitlocker status failed takes precedence
				// over profiles status pending
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerifying)
				// status for hosts[0] still failed because bitlocker status failed takes precedence
				// over profiles status verifying
				checkExpected(t, nil, expected)

				upsertHostProfileStatus(t, hosts[0].UUID, "some-windows-profile", &fleet.MDMDeliveryVerified)
				// status for hosts[0] still failed because bitlocker status failed takes precedence
				// over profiles status verified
				checkExpected(t, nil, expected)

				cleanupTables(t)
			})

			// turn off disk encryption so that the rest of the tests can focus on profiles status
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			require.NoError(t, ds.SaveAppConfig(ctx, ac))
		})
	})

	t.Run("profiles summary accounts for host profiles with mixed statuses", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			// upsert five profiles for hosts[0] with nil statuses
			upsertHostProfileStatus(t, hosts[0].UUID, fmt.Sprintf("some-windows-profile-%d", i), nil)
			// upsert five profiles for hosts[1] with pending statuses
			upsertHostProfileStatus(t, hosts[1].UUID, fmt.Sprintf("some-windows-profile-%d", i), &fleet.MDMDeliveryPending)
			// upsert five profiles for hosts[2] with verifying statuses
			upsertHostProfileStatus(t, hosts[2].UUID, fmt.Sprintf("some-windows-profile-%d", i), &fleet.MDMDeliveryVerifying)
			// upsert five profiles for hosts[3] with verified statuses
			upsertHostProfileStatus(t, hosts[3].UUID, fmt.Sprintf("some-windows-profile-%d", i), &fleet.MDMDeliveryVerified)
			// upsert five profiles for hosts[4] with failed statuses
			upsertHostProfileStatus(t, hosts[4].UUID, fmt.Sprintf("some-windows-profile-%d", i), &fleet.MDMDeliveryFailed)
		}

		expected := hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
			fleet.MDMDeliveryVerified:  []uint{hosts[3].ID},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// add some other windows hosts that won't be be assigned any profiles
		otherHosts := make([]*fleet.Host, 0, 5)
		for i := 0; i < 5; i++ {
			u := uuid.New().String()
			h, err := ds.NewHost(ctx, &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				NodeKey:         &u,
				UUID:            u,
				Hostname:        u,
				Platform:        "windows",
			})
			require.NoError(t, err)
			require.NotNil(t, h)
			otherHosts = append(otherHosts, h)

			require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
			windowsEnroll(t, ds, h)
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to failed status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-windows-profile-0", &fleet.MDMDeliveryFailed)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to pending status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-windows-profile-0", &fleet.MDMDeliveryPending)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to verifying status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-windows-profile-0", &fleet.MDMDeliveryVerifying)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID, hosts[3].ID},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// upsert some-profile-0 to verified status for hosts[0:4]
		for i := 0; i < 5; i++ {
			upsertHostProfileStatus(t, hosts[i].UUID, "some-windows-profile-0", &fleet.MDMDeliveryVerified)
		}
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
			fleet.MDMDeliveryVerified:  []uint{hosts[3].ID},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// turn on disk encryption
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))

		// hosts[0:3] are now pending because disk encryption is enabled, hosts[4] is still failed,
		// and other hosts are now counted as pending
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, otherHosts[0].ID, otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
			fleet.MDMDeliveryVerifying: []uint{},
			fleet.MDMDeliveryVerified:  []uint{},
			fleet.MDMDeliveryFailed:    []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// create a new team
		t1, err := ds.NewTeam(ctx, &fleet.Team{Name: uuid.NewString()})
		require.NoError(t, err)
		require.NotNil(t, t1)

		// transfer hosts[1:2] to the team
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&t1.ID, []uint{hosts[1].ID, hosts[2].ID})))

		// hosts[1:2] now counted for the team, hosts[2] is counted as verifying again because
		// disk encryption is not enabled for the team
		expectedTeam1 := hostIDsByProfileStatus{
			fleet.MDMDeliveryPending:   []uint{hosts[1].ID},
			fleet.MDMDeliveryVerifying: []uint{hosts[2].ID},
		}
		checkExpected(t, &t1.ID, expectedTeam1)
		// hosts[1:2] are not counted for no team
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[3].ID, otherHosts[0].ID, otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
			fleet.MDMDeliveryFailed:  []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// report otherHosts[0] as a server
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, otherHosts[0].ID, true, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
		// otherHosts[0] is no longer counted
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[3].ID, otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
			fleet.MDMDeliveryFailed:  []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// report hosts[0] as a server
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, hosts[0].ID, true, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
		// hosts[0] is no longer counted
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending: []uint{hosts[3].ID, otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
			fleet.MDMDeliveryFailed:  []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// unenroll hosts[3]
		require.NoError(t, ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, uuidToDeviceID[hosts[3].UUID]))
		// hosts[3] is no longer counted
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending: []uint{otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
			fleet.MDMDeliveryFailed:  []uint{hosts[4].ID},
		}
		checkExpected(t, nil, expected)

		// report hosts[4] as enrolled to a different MDM
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, hosts[4].ID, false, true, "https://some-other-mdm.example.com", false, "some-other-mdm", "", false))
		require.NoError(t, ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, uuidToDeviceID[hosts[4].UUID]))
		// hosts[4] is no longer counted
		expected = hostIDsByProfileStatus{
			fleet.MDMDeliveryPending: []uint{otherHosts[1].ID, otherHosts[2].ID, otherHosts[3].ID, otherHosts[4].ID},
		}
		checkExpected(t, nil, expected)

		cleanupTables(t)

		// turn off disk encryption for future tests
		ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
		require.NoError(t, ds.SaveAppConfig(ctx, ac))
	})
}

func testMDMWindowsInsertCommandForHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	d1 := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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

	err = ds.MDMWindowsInsertEnrolledDevice(ctx, d2)
	require.NoError(t, err)

	d1ID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d1.MDMHardwareID)
	d2ID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d2.MDMHardwareID)

	cmd := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri",
	}

	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{}, cmd)
	require.NoError(t, err)
	// no commands are enqueued nor created
	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d1ID)
	require.NoError(t, err)
	require.Empty(t, cmds)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
	require.NoError(t, err)
	require.Empty(t, cmds)

	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d1.HostUUID, d2.HostUUID}, cmd)
	require.NoError(t, err)
	// command enqueued and created
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d1ID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	// commands can be added by device id as well
	cmd.CommandUUID = uuid.NewString()
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d1.MDMDeviceID, d2.MDMDeviceID}, cmd)
	require.NoError(t, err)
	// command enqueued and created
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d1ID)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
	require.NoError(t, err)
	require.Len(t, cmds, 2)

	// create a device that enrolls with the same device id and uuid as d1
	// but a different hardware id (simulates the issue in #20764).
	d3 := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            d1.MDMDeviceID,
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               d1.HostUUID,
	}

	time.Sleep(time.Second) // ensure it gets a latest created_at
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, d3)
	require.NoError(t, err)
	d3ID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d3.MDMHardwareID)

	// commands can still be enqueued, will be enqueued for the latest enrolled device
	// when a duplicate host uuid/device id exists (i.e. for d3 even if d2 is passed -
	// they have the same ids).
	cmd.CommandUUID = uuid.NewString()
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{d1.MDMDeviceID, d2.MDMDeviceID}, cmd)
	require.NoError(t, err)
	// Commands are queried per-enrollment: the new command was enqueued against
	// d3 (the latest enrollment sharing d1's MDMDeviceID) and d2, so d1's own
	// enrollment still sees only its two prior commands.
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d1ID)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
	require.NoError(t, err)
	require.Len(t, cmds, 3)
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d3ID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
}

func testMDMWindowsInsertCommandAndUpsertHostProfilesForHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	newEnrolledDevice := func() *fleet.MDMWindowsEnrolledDevice {
		return &fleet.MDMWindowsEnrolledDevice{
			MDMDeviceID:            uuid.New().String(),
			MDMHardwareID:          uuid.New().String() + uuid.New().String(),
			MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
			MDMDeviceType:          "CIMClient_Windows",
			MDMDeviceName:          "DESKTOP-1C3ARC1",
			MDMEnrollType:          "ProgrammaticEnrollment",
			MDMEnrollUserID:        "",
			MDMEnrollProtoVersion:  "5.0",
			MDMEnrollClientVersion: "10.0.19045.2965",
			MDMNotInOOBE:           false,
			HostUUID:               uuid.NewString(),
		}
	}

	d1 := newEnrolledDevice()
	d2 := newEnrolledDevice()
	err := ds.MDMWindowsInsertEnrolledDevice(ctx, d1)
	require.NoError(t, err)
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, d2)
	require.NoError(t, err)

	d1ID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d1.MDMHardwareID)
	d2ID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d2.MDMHardwareID)

	getAllHostProfiles := func() []*fleet.MDMWindowsProfilePayload {
		var hostProfiles []*fleet.MDMWindowsProfilePayload
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid, host_uuid, status, operation_type, command_uuid, profile_name FROM host_mdm_windows_profiles ORDER BY host_uuid, profile_name`
			return sqlx.SelectContext(ctx, q, &hostProfiles, stmt)
		})
		return hostProfiles
	}

	t.Run("empty host list is a noop", func(t *testing.T) {
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID:  uuid.NewString(),
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}
		err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{}, cmd, nil)
		require.NoError(t, err)
	})

	t.Run("inserts command and profiles for multiple hosts", func(t *testing.T) {
		profUUID1 := InsertWindowsProfileForTest(t, ds, 0)
		profUUID2 := InsertWindowsProfileForTest(t, ds, 0)

		cmdUUID := uuid.NewString()
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID:  cmdUUID,
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}

		payload := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profUUID1,
				ProfileName:   "prof1",
				HostUUID:      d1.HostUUID,
				CommandUUID:   cmdUUID,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum1"),
			},
			{
				ProfileUUID:   profUUID2,
				ProfileName:   "prof2",
				HostUUID:      d2.HostUUID,
				CommandUUID:   cmdUUID,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum2"),
			},
		}

		err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID, d2.HostUUID}, cmd, payload)
		require.NoError(t, err)

		// Verify commands were enqueued
		cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d1ID)
		require.NoError(t, err)
		require.Len(t, cmds, 1)
		cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
		require.NoError(t, err)
		require.Len(t, cmds, 1)

		// Verify host profiles were upserted
		hostProfs := getAllHostProfiles()
		require.Len(t, hostProfs, 2)
		// Verify both profiles are present with expected values
		for _, hp := range hostProfs {
			require.Equal(t, fleet.MDMOperationTypeInstall, hp.OperationType)
			require.Equal(t, &fleet.MDMDeliveryPending, hp.Status)
			require.Equal(t, cmdUUID, hp.CommandUUID)
		}
	})

	t.Run("duplicate command uuid returns already exists", func(t *testing.T) {
		profUUID := InsertWindowsProfileForTest(t, ds, 0)
		cmdUUID := uuid.NewString()
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID:  cmdUUID,
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}
		payload := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profUUID,
				ProfileName:   "prof-dup",
				HostUUID:      d1.HostUUID,
				CommandUUID:   cmdUUID,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum-dup"),
			},
		}

		err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID}, cmd, payload)
		require.NoError(t, err)

		// Same command UUID again should fail with already exists
		err = ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID}, cmd, payload)
		require.Error(t, err)
		require.ErrorContains(t, err, "already exists")
	})

	t.Run("upserts update existing host profiles", func(t *testing.T) {
		profUUID := InsertWindowsProfileForTest(t, ds, 0)
		cmdUUID1 := uuid.NewString()
		cmd1 := &fleet.MDMWindowsCommand{
			CommandUUID:  cmdUUID1,
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}
		payload1 := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profUUID,
				ProfileName:   "prof-upsert",
				HostUUID:      d1.HostUUID,
				CommandUUID:   cmdUUID1,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum-v1"),
			},
		}

		err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID}, cmd1, payload1)
		require.NoError(t, err)

		// Now upsert with a new command and updated status
		cmdUUID2 := uuid.NewString()
		cmd2 := &fleet.MDMWindowsCommand{
			CommandUUID:  cmdUUID2,
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}
		payload2 := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profUUID,
				ProfileName:   "prof-upsert-updated",
				HostUUID:      d1.HostUUID,
				CommandUUID:   cmdUUID2,
				OperationType: fleet.MDMOperationTypeRemove,
				Status:        &fleet.MDMDeliveryVerifying,
				Checksum:      []byte("checksum-v2"),
			},
		}

		err = ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID}, cmd2, payload2)
		require.NoError(t, err)

		// Verify the profile was updated (not duplicated)
		var profiles []*fleet.MDMWindowsProfilePayload
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid, host_uuid, status, operation_type, command_uuid, profile_name FROM host_mdm_windows_profiles WHERE profile_uuid = ? AND host_uuid = ?`
			return sqlx.SelectContext(ctx, q, &profiles, stmt, profUUID, d1.HostUUID)
		})
		require.Len(t, profiles, 1)
		require.Equal(t, cmdUUID2, profiles[0].CommandUUID)
		require.Equal(t, fleet.MDMOperationTypeRemove, profiles[0].OperationType)
		require.Equal(t, &fleet.MDMDeliveryVerifying, profiles[0].Status)
		require.Equal(t, "prof-upsert-updated", profiles[0].ProfileName)
	})

	t.Run("batching works correctly", func(t *testing.T) {
		// Set a small batch size to force multiple batches
		ds.testUpsertMDMDesiredProfilesBatchSize = 1

		profUUID1 := InsertWindowsProfileForTest(t, ds, 0)
		profUUID2 := InsertWindowsProfileForTest(t, ds, 0)
		cmdUUID := uuid.NewString()
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID:  cmdUUID,
			RawCommand:   []byte("<Exec></Exec>"),
			TargetLocURI: "./test/uri",
		}
		payload := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{
				ProfileUUID:   profUUID1,
				ProfileName:   "prof-batch1",
				HostUUID:      d1.HostUUID,
				CommandUUID:   cmdUUID,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum-batch1"),
			},
			{
				ProfileUUID:   profUUID2,
				ProfileName:   "prof-batch2",
				HostUUID:      d2.HostUUID,
				CommandUUID:   cmdUUID,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &fleet.MDMDeliveryPending,
				Checksum:      []byte("checksum-batch2"),
			},
		}

		err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID, d2.HostUUID}, cmd, payload)
		require.NoError(t, err)

		// Verify both hosts got their commands despite batching
		cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d1ID)
		require.NoError(t, err)
		found := false
		for _, c := range cmds {
			if c.CommandUUID == cmdUUID {
				found = true
				break
			}
		}
		require.True(t, found, "expected command for d1")

		cmds, err = ds.MDMWindowsGetPendingCommands(ctx, d2ID)
		require.NoError(t, err)
		found = false
		for _, c := range cmds {
			if c.CommandUUID == cmdUUID {
				found = true
				break
			}
		}
		require.True(t, found, "expected command for d2")

		// Reset batch size
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
	})
}

func testMDMWindowsGetPendingCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	d := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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

	dID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d.MDMHardwareID)

	// device without commands
	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, dID)
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

	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, dID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	// non-existent enrollment
	cmds, err = ds.MDMWindowsGetPendingCommands(ctx, 0)
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

	dev := createMDMWindowsEnrollment(ctx, t, ds)
	var enrollmentID uint
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &enrollmentID, `SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev.MDMDeviceID))
	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE id = ?`, dev.HostUUID, enrollmentID)
	require.NoError(t, err)

	rawCmd := "some-command"
	cmdUUID := "some-uuid"
	cmdTarget := "some-target-loc-uri"
	_, err = insertDB(t, `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`, cmdUUID, rawCmd, cmdTarget)
	require.NoError(t, err)

	rawResult := []byte("some-result")
	statusCode := "200"
	_, err = insertDB(t, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code) VALUES (?, ?, ?, NULL, ?)`, enrollmentID, cmdUUID, rawResult, statusCode)
	require.NoError(t, err)

	// Create multiple command queue entries to ensure no duplicated rows.
	dev2 := createEnrolledDevice(t, ds)
	dev3 := createEnrolledDevice(t, ds)
	var enrollmentID2 uint
	var enrollmentID3 uint
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &enrollmentID2,
		`SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev2.MDMDeviceID))
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &enrollmentID3,
		`SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev3.MDMDeviceID))

	// Insert queue entry for BOTH enrollments
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid)
         VALUES (?, ?), (?, ?)`,
		enrollmentID2, cmdUUID,
		enrollmentID3, cmdUUID,
	)
	require.NoError(t, err)

	p, err := ds.GetMDMCommandPlatform(ctx, cmdUUID)
	require.NoError(t, err)
	require.Equal(t, "windows", p)

	results, err := ds.GetMDMWindowsCommandResults(ctx, cmdUUID, "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, dev.HostUUID, results[0].HostUUID)
	require.Equal(t, cmdUUID, results[0].CommandUUID)
	require.Equal(t, rawResult, results[0].Result)
	require.Equal(t, cmdTarget, results[0].RequestType)
	require.Equal(t, statusCode, results[0].Status)
	require.Empty(t, results[0].Hostname) // populated only at the service layer
	require.Equal(t, rawCmd, string(results[0].Payload))

	p, err = ds.GetMDMCommandPlatform(ctx, "unknown-cmd-uuid")
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, p)

	results, err = ds.GetMDMWindowsCommandResults(ctx, "unknown-cmd-uuid", "")
	require.NoError(t, err) // expect no error here, just no results
	require.Empty(t, results)
}

// mdmWindowsEnrollmentIDByHardwareID resolves an enrollment row's auto-increment id
// from its hardware id. MDMWindowsInsertEnrolledDevice does not populate the ID field
// on the inserted struct, and hardware_id is the only identifier guaranteed unique in
// tests that exercise the duplicate device_id / host_uuid case.
func mdmWindowsEnrollmentIDByHardwareID(ctx context.Context, t *testing.T, ds *Datastore, hwID string) uint {
	t.Helper()
	var id uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &id, `SELECT id FROM mdm_windows_enrollments WHERE mdm_hardware_id = ?`, hwID)
	})
	return id
}

func createMDMWindowsEnrollment(ctx context.Context, t *testing.T, ds *Datastore) *fleet.MDMWindowsEnrolledDevice {
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
	return dev
}

func testMDMWindowsCommandResultsWithPendingResult(t *testing.T, ds *Datastore) {
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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

	_, err = insertDB(t, `INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid) VALUES (?, ? )`, enrollmentID, cmdUUID)
	require.NoError(t, err)

	p, err := ds.GetMDMCommandPlatform(ctx, cmdUUID)
	require.NoError(t, err)
	require.Equal(t, "windows", p)

	results, err := ds.GetMDMWindowsCommandResults(ctx, cmdUUID, "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, dev.HostUUID, results[0].HostUUID)
	require.Equal(t, cmdUUID, results[0].CommandUUID)
	require.Equal(t, []byte{}, results[0].Result)
	require.Equal(t, cmdTarget, results[0].RequestType)
	require.Equal(t, "101", results[0].Status)
	require.Empty(t, results[0].Hostname) // populated only at the service layer
	require.Equal(t, rawCmd, string(results[0].Payload))

	p, err = ds.GetMDMCommandPlatform(ctx, "unknown-cmd-uuid")
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, p)

	results, err = ds.GetMDMWindowsCommandResults(ctx, "unknown-cmd-uuid", "")
	require.NoError(t, err) // expect no error here, just no results
	require.Empty(t, results)
}

// enrolls the host in Windows MDM and returns the device's enrollment ID.
func windowsEnroll(t *testing.T, ds fleet.Datastore, h *fleet.Host) string {
	ctx := context.Background()
	d1 := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
	return d1.MDMDeviceID
}

// windowsProfileUUIDByName returns the profile_uuid of a Windows config profile
// identified by its (unique) name.
func windowsProfileUUIDByName(t *testing.T, ds *Datastore, name string) string {
	t.Helper()
	var u string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &u,
			`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`, name)
	})
	return u
}

// installWindowsProfilesAsVerified simulates a successful prior install of each
// profile on each host by inserting operation_type=install, status=verified rows
// via BulkUpsertMDMWindowsHostProfiles.
func installWindowsProfilesAsVerified(t *testing.T, ds *Datastore, hostUUIDs, profileUUIDs []string) {
	t.Helper()
	verified := fleet.MDMDeliveryVerified
	var payloads []*fleet.MDMWindowsBulkUpsertHostProfilePayload
	for _, hUUID := range hostUUIDs {
		for _, pUUID := range profileUUIDs {
			payloads = append(payloads, &fleet.MDMWindowsBulkUpsertHostProfilePayload{
				ProfileUUID:   pUUID,
				ProfileName:   "test",
				HostUUID:      hUUID,
				CommandUUID:   uuid.NewString(),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &verified,
				Checksum:      []byte{0},
			})
		}
	}
	require.NoError(t, ds.BulkUpsertMDMWindowsHostProfiles(t.Context(), payloads))
}

// rawWindowsDeleteCommandForHostProfile returns the raw SyncML of the queued
// <Delete> command for a (host, profile) pair by joining host_mdm_windows_profiles
// (operation_type=remove) to windows_mdm_commands via command_uuid. Returns nil
// if no such command is queued, so callers can assert with a descriptive message.
func rawWindowsDeleteCommandForHostProfile(t *testing.T, ds *Datastore, hostUUID, profileUUID string) []byte {
	t.Helper()
	var raws [][]byte
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(t.Context(), q, &raws,
			`SELECT wc.raw_command FROM windows_mdm_commands wc
			 JOIN host_mdm_windows_profiles hwp ON hwp.command_uuid = wc.command_uuid
			 WHERE hwp.host_uuid = ? AND hwp.profile_uuid = ? AND hwp.operation_type = ?`,
			hostUUID, profileUUID, fleet.MDMOperationTypeRemove)
	})
	if len(raws) == 0 {
		return nil
	}
	return raws[0]
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
	profileByUUID := make(map[string]*fleet.MDMWindowsProfilePayload, len(profiles))
	for _, prof := range profiles {
		profileByUUID[prof.ProfileUUID] = prof
	}

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
				Checksum:      profileByUUID[globalProfiles[0]].Checksum,
			},
			{
				ProfileUUID:   globalProfiles[0],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[globalProfiles[0]].Checksum,
			},
			{
				ProfileUUID:   globalProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[globalProfiles[1]].Checksum,
			},
			{
				ProfileUUID:   globalProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[globalProfiles[1]].Checksum,
			},
			{
				ProfileUUID:   globalProfiles[2],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[globalProfiles[2]].Checksum,
			},
			{
				ProfileUUID:   globalProfiles[2],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[globalProfiles[2]].Checksum,
			},
			{
				ProfileUUID:   teamProfiles[0],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-2",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[teamProfiles[0]].Checksum,
			},
			{
				ProfileUUID:   teamProfiles[1],
				ProfileName:   "foo",
				HostUUID:      "test-uuid-2",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      profileByUUID[teamProfiles[1]].Checksum,
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
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID}))
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
				Checksum:      []byte{0},
			},
			{
				ProfileUUID:   profiles[1],
				ProfileName:   "B",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{1},
			},
			{
				ProfileUUID:   profiles[2],
				ProfileName:   "C",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{2},
			},
			{
				ProfileUUID:   profiles[3],
				ProfileName:   "D",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{3},
			},
			{
				ProfileUUID:   profiles[4],
				ProfileName:   "E",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerifying,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{4},
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
				Checksum:      []byte{0},
			},
			{
				ProfileUUID:   profiles[1],
				ProfileName:   "B",
				HostUUID:      "test-uuid-3",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{1},
			},
			{
				ProfileUUID:   profiles[2],
				ProfileName:   "C",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{2},
			},
			{
				ProfileUUID:   profiles[3],
				ProfileName:   "D",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{3},
			},
			{
				ProfileUUID:   profiles[4],
				ProfileName:   "E",
				HostUUID:      "test-uuid-1",
				Status:        &fleet.MDMDeliveryVerified,
				OperationType: fleet.MDMOperationTypeInstall,
				CommandUUID:   "command-uuid",
				Checksum:      []byte{4},
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
	ds.testUpsertMDMDesiredProfilesBatchSize = 2
	ds.testDeleteMDMProfilesBatchSize = 2
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
		ds.testDeleteMDMProfilesBatchSize = 0
	})
	testBulkOperationsMDMWindowsHostProfiles(t, ds)
}

func testBulkOperationsMDMWindowsHostProfilesBatch3(t *testing.T, ds *Datastore) {
	ds.testUpsertMDMDesiredProfilesBatchSize = 3
	ds.testDeleteMDMProfilesBatchSize = 3
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
		ds.testDeleteMDMProfilesBatchSize = 0
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
		want map[string]fleet.MDMWindowsProfileContents
	}{
		{[]string{}, nil},
		{nil, nil},
		{
			[]string{profileUUIDs[0]},
			map[string]fleet.MDMWindowsProfileContents{profileUUIDs[0]: generateDummyWindowsProfileContents(profileUUIDs[0])},
		},
		{
			[]string{profileUUIDs[0], profileUUIDs[1], profileUUIDs[2]},
			map[string]fleet.MDMWindowsProfileContents{
				profileUUIDs[0]: generateDummyWindowsProfileContents(profileUUIDs[0]),
				profileUUIDs[1]: generateDummyWindowsProfileContents(profileUUIDs[1]),
				profileUUIDs[2]: generateDummyWindowsProfileContents(profileUUIDs[2]),
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
	profA, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: nil, SyncML: []byte("<Replace></Replace>")}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, profA.ProfileUUID)
	profB, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: ptr.Uint(0), SyncML: []byte("<Replace></Replace>")}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, profB.ProfileUUID)
	// create an Apple profile for no-team
	profC, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("c", "c", 0), nil)
	require.NoError(t, err)
	require.NotZero(t, profC.ProfileID)
	require.NotEmpty(t, profC.ProfileUUID)

	// create the same name for team 1 as Windows profile
	profATm, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, profATm.ProfileUUID)
	require.NotNil(t, profATm.TeamID)
	require.Equal(t, uint(1), *profATm.TeamID)
	// create the same B profile for team 1 as Apple profile
	profBTm, err := ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("b", "b", 1), nil)
	require.NoError(t, err)
	require.NotZero(t, profBTm.ProfileID)
	require.NotEmpty(t, profBTm.ProfileUUID)

	var existsErr *existsError
	// create a duplicate of Windows for no-team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: nil, SyncML: []byte("<Replace></Replace>")}, nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Apple for no-team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "c", TeamID: nil, SyncML: []byte("<Replace></Replace>")}, nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Windows for team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")}, nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Apple for team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")}, nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with an Apple profile for no-team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("a", "a", 0), nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with an Apple profile for team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("a", "a", 1), nil)
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// create a profile with labels that don't exist
	_, err = ds.NewMDMWindowsConfigProfile(
		ctx,
		fleet.MDMWindowsConfigProfile{
			Name:             "fake-labels",
			TeamID:           nil,
			SyncML:           []byte("<Replace></Replace>"),
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{LabelName: "foo", LabelID: 1}},
		}, nil)
	require.NotNil(t, err)
	require.True(t, fleet.IsForeignKey(err))

	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
	}
	label, err = ds.NewLabel(ctx, label)
	require.NoError(t, err)

	// create a profile with a label that exists
	profWithLabel, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		fleet.MDMWindowsConfigProfile{
			Name:             "with-labels",
			TeamID:           nil,
			SyncML:           []byte("<Replace></Replace>"),
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{LabelName: label.Name, LabelID: label.ID}},
		}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, profWithLabel.ProfileUUID)

	// get that profile with label
	prof, err := ds.GetMDMWindowsConfigProfile(ctx, profWithLabel.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.LabelsIncludeAll, 1)
	require.Equal(t, label.Name, prof.LabelsIncludeAll[0].LabelName)
	require.Equal(t, label.ID, prof.LabelsIncludeAll[0].LabelID)
	require.False(t, prof.LabelsIncludeAll[0].Broken)

	// break that profile by deleting the label
	require.NoError(t, ds.DeleteLabel(ctx, label.Name, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}))

	prof, err = ds.GetMDMWindowsConfigProfile(ctx, profWithLabel.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.LabelsIncludeAll, 1)
	require.Equal(t, label.Name, prof.LabelsIncludeAll[0].LabelName)
	require.Zero(t, prof.LabelsIncludeAll[0].LabelID)
	require.True(t, prof.LabelsIncludeAll[0].Broken)

	_, err = ds.GetMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	prof, err = ds.GetMDMWindowsConfigProfile(ctx, profA.ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, profA.ProfileUUID, prof.ProfileUUID)
	require.NotNil(t, prof.TeamID)
	require.Zero(t, *prof.TeamID)
	require.Equal(t, "a", prof.Name)
	require.Equal(t, "<Replace></Replace>", string(prof.SyncML))
	require.NotZero(t, prof.CreatedAt)
	require.NotZero(t, prof.UploadedAt)
	require.Nil(t, prof.LabelsIncludeAll)

	err = ds.DeleteMDMWindowsConfigProfile(ctx, "not-valid")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	err = ds.DeleteMDMWindowsConfigProfile(ctx, profA.ProfileUUID)
	require.NoError(t, err)
}

func testMDMWindowsConfigProfilesWithFleetVars(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Test that usesFleetVars parameter correctly persists variables in the database
	// Create a profile with Fleet variables
	profWithVars, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "profile_with_vars",
		TeamID: nil,
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/$FLEET_VAR_HOST_UUID</LocURI></Target></Item></Replace>"),
	}, []fleet.FleetVarName{fleet.FleetVarHostUUID})
	require.NoError(t, err)
	require.NotEmpty(t, profWithVars.ProfileUUID)

	// Query the mdm_configuration_profile_variables table to verify the variables were persisted
	var varNames []string
	stmt := `
		SELECT fv.name
		FROM mdm_configuration_profile_variables mcpv
		JOIN fleet_variables fv ON mcpv.fleet_variable_id = fv.id
		WHERE mcpv.windows_profile_uuid = ?
		ORDER BY fv.name
	`
	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, profWithVars.ProfileUUID)
	require.NoError(t, err)

	// Assert that the returned variable names exactly match the provided slice
	// Note: the database stores the full name with FLEET_VAR_ prefix
	expectedVarNames := []string{"FLEET_VAR_" + string(fleet.FleetVarHostUUID)}
	require.Equal(t, expectedVarNames, varNames, "Variable names in database should match the provided usesFleetVars slice")

	// Test with empty usesFleetVars slice
	profNoVars, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "profile_no_vars",
		TeamID: nil,
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/NoVars</LocURI></Target></Item></Replace>"),
	}, []fleet.FleetVarName{})
	require.NoError(t, err)
	require.NotEmpty(t, profNoVars.ProfileUUID)

	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, profNoVars.ProfileUUID)
	require.NoError(t, err)
	require.Empty(t, varNames, "No variables should be persisted when usesFleetVars is empty")

	// Test with nil usesFleetVars
	profNilVars, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "profile_nil_vars",
		TeamID: nil,
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/NilVars</LocURI></Target></Item></Replace>"),
	}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, profNilVars.ProfileUUID)

	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, profNilVars.ProfileUUID)
	require.NoError(t, err)
	require.Empty(t, varNames, "No variables should be persisted when usesFleetVars is nil")

	// Test that BatchSetMDMProfiles properly clears stale variable associations
	// Create a team profile with variables
	teamProf1, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "team_profile_1",
		TeamID: ptr.Uint(1),
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/$FLEET_VAR_HOST_UUID</LocURI></Target></Item></Replace>"),
	}, []fleet.FleetVarName{fleet.FleetVarHostUUID})
	require.NoError(t, err)

	// Verify the variable was persisted
	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, teamProf1.ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, expectedVarNames, varNames, "Team profile should have HOST_UUID variable")

	// Now update the profile via BatchSetMDMProfiles to remove the variable
	teamProf1Updated := &fleet.MDMWindowsConfigProfile{
		Name:   "team_profile_1",
		TeamID: ptr.Uint(1),
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/NoVarsAnymore</LocURI></Target></Item></Replace>"),
	}

	// BatchSetMDMProfiles should process this profile and clear its variable associations
	// since the content no longer contains variables
	_, err = ds.BatchSetMDMProfiles(ctx, ptr.Uint(1), nil, []*fleet.MDMWindowsConfigProfile{teamProf1Updated}, nil, nil, nil)
	require.NoError(t, err)

	// Verify the variable associations were cleared
	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, teamProf1.ProfileUUID)
	require.NoError(t, err)
	require.Empty(t, varNames, "Variables should be cleared when profile is updated without variables")

	// Create another team profile to test multiple profiles with mixed variables
	teamProf2, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "team_profile_2",
		TeamID: ptr.Uint(1),
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/Profile2</LocURI></Target></Item></Replace>"),
	}, nil)
	require.NoError(t, err)

	// Update both profiles - one adds variables, one keeps no variables
	teamProf1WithVarsAgain := &fleet.MDMWindowsConfigProfile{
		Name:   "team_profile_1",
		TeamID: ptr.Uint(1),
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/WithVarsAgain/$FLEET_VAR_HOST_UUID</LocURI></Target></Item></Replace>"),
	}
	teamProf2NoChange := &fleet.MDMWindowsConfigProfile{
		Name:   "team_profile_2",
		TeamID: ptr.Uint(1),
		SyncML: []byte("<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Test/Profile2Updated</LocURI></Target></Item></Replace>"),
	}

	// Mock the profilesVariablesByIdentifier that would be passed from service layer
	profilesVars := []fleet.MDMProfileIdentifierFleetVariables{
		{Identifier: fleet.MDMWindowsProfileUUIDPrefix + "team_profile_1", FleetVariables: []fleet.FleetVarName{fleet.FleetVarHostUUID}},
	}

	_, err = ds.BatchSetMDMProfiles(ctx, ptr.Uint(1), nil, []*fleet.MDMWindowsConfigProfile{teamProf1WithVarsAgain, teamProf2NoChange}, nil, nil, profilesVars)
	require.NoError(t, err)

	// Verify profile 1 has variables again
	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, teamProf1.ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, expectedVarNames, varNames, "Profile 1 should have variables again")

	// Verify profile 2 still has no variables
	err = ds.writer(ctx).SelectContext(ctx, &varNames, stmt, teamProf2.ProfileUUID)
	require.NoError(t, err)
	require.Empty(t, varNames, "Profile 2 should still have no variables")
}

func testSetOrReplaceMDMWindowsConfigProfile(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	getProfileByTeamAndName := func(tmID *uint, name string) *fleet.MDMWindowsConfigProfile {
		var prof fleet.MDMWindowsConfigProfile
		var teamID uint
		if tmID != nil {
			teamID = *tmID
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &prof,
				`SELECT profile_uuid, team_id, name, syncml, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`,
				teamID, name)
		})
		return &prof
	}

	// nothing for no-team, nothing for team 1
	expectWindowsProfiles(t, ds, nil, nil)
	expectWindowsProfiles(t, ds, ptr.Uint(1), nil)

	// create a profile for no-team
	cp1 := *windowsConfigProfileForTest(t, "N1", "N1")
	err := ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp1)
	require.NoError(t, err)
	profNoTmN1 := getProfileByTeamAndName(nil, "N1")

	// creating the same profile for Apple / no-team fails
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("N1", "I1", 0), nil)
	require.Error(t, err)

	cp1.UploadedAt = profNoTmN1.UploadedAt
	profs1 := expectWindowsProfiles(t, ds, nil, []*fleet.MDMWindowsConfigProfile{&cp1})

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// update the profile content for no-team
	cp2 := *windowsConfigProfileForTest(t, "N1", "N1.modified")
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp2)
	require.NoError(t, err)

	profNoTmN1b := getProfileByTeamAndName(nil, "N1")
	profs2 := expectWindowsProfiles(t, ds, nil, []*fleet.MDMWindowsConfigProfile{&cp2})

	// profile UUIDs are the same
	require.Equal(t, profs1["N1"], profs2["N1"])
	// uploaded_at is not the same
	require.False(t, profNoTmN1.UploadedAt.Equal(profNoTmN1b.UploadedAt))

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// update the profile for no-team without change
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp2)
	require.NoError(t, err)
	cp2.UploadedAt = profNoTmN1b.UploadedAt
	expectWindowsProfiles(t, ds, nil, []*fleet.MDMWindowsConfigProfile{&cp2})

	// create a profile for Apple and team 1 with that name works
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateAppleCP("N1", "I1", 1), nil)
	require.NoError(t, err)

	// try to create that profile for Windows and team 1 fails
	cp3 := *windowsConfigProfileForTest(t, "N1", "N1")
	cp3.TeamID = ptr.Uint(1)
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp3)
	require.Error(t, err)

	expectWindowsProfiles(t, ds, ptr.Uint(1), nil)

	// create a profile with the same name for team 2 works
	cp4 := *windowsConfigProfileForTest(t, "N1", "N1")
	cp4.TeamID = ptr.Uint(2)
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp4)
	require.NoError(t, err)

	profs3 := expectWindowsProfiles(t, ds, ptr.Uint(2), []*fleet.MDMWindowsConfigProfile{&cp4})
	// profile UUIDs are not the same as for no-team
	require.NotEqual(t, profs3["N1"], profs2["N1"])

	// create a different profile for no-team
	cp5 := *windowsConfigProfileForTest(t, "N2", "N2")
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp5)
	require.NoError(t, err)

	expectWindowsProfiles(t, ds, nil, []*fleet.MDMWindowsConfigProfile{&cp2, &cp5})

	// update that profile for no-team
	cp6 := *windowsConfigProfileForTest(t, "N2", "N2.modified")
	err = ds.SetOrUpdateMDMWindowsConfigProfile(ctx, cp6)
	require.NoError(t, err)

	expectWindowsProfiles(t, ds, nil, []*fleet.MDMWindowsConfigProfile{&cp2, &cp6})
}

func testMDMWindowsProfileLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// Create a windows host
	u := uuid.New().String()
	host, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		// Set this slightly in the past to test dynamic label exclusion
		LabelUpdatedAt:  time.Now().Add(-5 * time.Second),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         &u,
		UUID:            u,
		Hostname:        u,
		Platform:        "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, host)

	// "include-any" labels
	l1, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "include-any-label1",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	l2, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "include-any-label2",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	l3, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "include-any-label3",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	// include-all labels
	l4, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "include-all-label4",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	l5, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "include-all-label5",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	// exclude-any labels
	l6, err := ds.NewLabel(ctx, &fleet.Label{
		Name:        "exclude-any-label6",
		Description: "desc",
		Query:       "select 1;",
	})
	require.NoError(t, err)

	l7, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "exclude-any-label7",
		Description:         "desc",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	// Create a profile with "include-any" with l1
	includeAnyProf, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		*windowsConfigProfileForTest(t, "prof-include-any", "./Foo/Bar", l1, l2, l3),
		nil,
	)
	require.NoError(t, err)
	require.NotEmpty(t, includeAnyProf.ProfileUUID)
	profileChecksums := make(map[string][]byte)
	checksum := md5.Sum(includeAnyProf.SyncML) // nolint:gosec // used only to hash for efficient comparisons
	profileChecksums[includeAnyProf.ProfileUUID] = checksum[:]

	// Create a profile with "include-all" with l4 and l5
	includeAllProf, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		*windowsConfigProfileForTest(t, "prof-include-all", "./Foo/Bar", l4, l5),
		nil,
	)
	require.NoError(t, err)
	require.NotEmpty(t, includeAllProf.ProfileUUID)
	checksum = md5.Sum(includeAllProf.SyncML) // nolint:gosec // used only to hash for efficient comparisons
	profileChecksums[includeAllProf.ProfileUUID] = checksum[:]

	// Create a profile with "exclude-any" with l6 and l7
	excludeAnyProf, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		*windowsConfigProfileForTest(t, "prof-exclude-any", "./Foo/Bar", l6, l7),
		nil,
	)
	require.NoError(t, err)
	checksum = md5.Sum(excludeAnyProf.SyncML) // nolint:gosec // used only to hash for efficient comparisons
	profileChecksums[excludeAnyProf.ProfileUUID] = checksum[:]

	// Create a profile with "exclude-any" with l7 only since it is a manual label
	excludeAnyManualProf, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		*windowsConfigProfileForTest(t, "prof-exclude-any-manual", "./Foo/Bar", l7),
		nil,
	)
	require.NoError(t, err)
	checksum = md5.Sum(excludeAnyManualProf.SyncML) // nolint:gosec // used only to hash for efficient comparisons
	profileChecksums[excludeAnyManualProf.ProfileUUID] = checksum[:]

	// Connect the host and l1, l4, l5
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l1.ID, host.ID}, {l4.ID, host.ID}, {l5.ID, host.ID}})
	require.NoError(t, err)

	// We should see 3 profiles in the "to install" list
	profilesToInstall, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: includeAllProf.ProfileUUID, ProfileName: includeAllProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAllProf.ProfileUUID],
		},
		{
			ProfileUUID: includeAnyProf.ProfileUUID, ProfileName: includeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)

	host.LabelUpdatedAt = time.Now().Add(1 * time.Second)
	err = ds.UpdateHost(ctx, host)
	require.NoError(t, err)

	// We should see all 4  profiles in the "to install" list
	profilesToInstall, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: includeAllProf.ProfileUUID, ProfileName: includeAllProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAllProf.ProfileUUID],
		},
		{
			ProfileUUID: includeAnyProf.ProfileUUID, ProfileName: includeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyProf.ProfileUUID, ProfileName: excludeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)

	// Remove the l1<->host relationship, but add l2<->labelHost. The profile should still show
	// up since it's "include any"
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l1.ID, host.ID}})
	require.NoError(t, err)

	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l2.ID, host.ID}})
	require.NoError(t, err)

	profilesToInstall, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: includeAllProf.ProfileUUID, ProfileName: includeAllProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAllProf.ProfileUUID],
		},
		{
			ProfileUUID: includeAnyProf.ProfileUUID, ProfileName: includeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyProf.ProfileUUID, ProfileName: excludeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)

	// Remove the l2<->host relationship. Since the profile is "include-any", it should no longer
	// show up
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l2.ID, host.ID}})
	require.NoError(t, err)

	profilesToInstall, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: includeAllProf.ProfileUUID, ProfileName: includeAllProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[includeAllProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyProf.ProfileUUID, ProfileName: excludeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)

	// Remove the l4<->host relationship. Since the profile is "include-all", it should no longer show
	// up even though the l5<->host connection is still there.
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l4.ID, host.ID}})
	require.NoError(t, err)

	profilesToInstall, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: excludeAnyProf.ProfileUUID, ProfileName: excludeAnyProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyProf.ProfileUUID],
		},
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)

	// Add a l6<->host relationship. The exclude-any profile with l6 and l7 should be gone now with only
	// the exclude-any-manual profile remaining.
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l6.ID, host.ID}})
	require.NoError(t, err)

	profilesToInstall, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMWindowsProfilePayload{
		{
			ProfileUUID: excludeAnyManualProf.ProfileUUID, ProfileName: excludeAnyManualProf.Name, HostUUID: host.UUID,
			Checksum: profileChecksums[excludeAnyManualProf.ProfileUUID],
		},
	}, profilesToInstall)
}

func expectWindowsProfiles(
	t *testing.T,
	ds *Datastore,
	tmID *uint,
	want []*fleet.MDMWindowsConfigProfile,
) map[string]string {
	if tmID == nil {
		tmID = ptr.Uint(0)
	}

	var got []*fleet.MDMWindowsConfigProfile
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		ctx := context.Background()
		return sqlx.SelectContext(ctx, q, &got,
			`SELECT profile_uuid, team_id, name, syncml, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE team_id = ?`,
			tmID)
	})

	// create map of expected profiles keyed by name
	wantMap := make(map[string]*fleet.MDMWindowsConfigProfile, len(want))
	for _, cp := range want {
		wantMap[cp.Name] = cp
	}

	// compare only the fields we care about, and build the resulting map of
	// profile name as key to profile UUID as value
	m := make(map[string]string)
	for _, gotp := range got {
		m[gotp.Name] = gotp.ProfileUUID
		if gotp.TeamID != nil && *gotp.TeamID == 0 {
			gotp.TeamID = nil
		}

		// ProfileUUID is non-empty and starts with "w", but otherwise we don't
		// care about it for test assertions.
		require.NotEmpty(t, gotp.ProfileUUID)
		require.True(t, strings.HasPrefix(gotp.ProfileUUID, "w"))
		gotp.ProfileUUID = ""

		gotp.CreatedAt = time.Time{}

		// if an expected uploaded_at timestamp is provided for this profile, keep
		// its value, otherwise clear it as we don't care about asserting its
		// value.
		if wantp := wantMap[gotp.Name]; wantp == nil || wantp.UploadedAt.IsZero() {
			gotp.UploadedAt = time.Time{}
		}
	}
	// order is not guaranteed
	require.ElementsMatch(t, want, got)

	return m
}

func testBatchSetMDMWindowsProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	applyAndExpect := func(newSet []*fleet.MDMWindowsConfigProfile, tmID *uint, want []*fleet.MDMWindowsConfigProfile,
		wantUpdated bool,
	) map[string]string {
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			updatedDB, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, tmID, newSet, nil)
			require.NoError(t, err)
			assert.Equal(t, wantUpdated, updatedDB)
			return err
		})
		require.NoError(t, err)
		return expectWindowsProfiles(t, ds, tmID, want)
	}

	getProfileByTeamAndName := func(tmID *uint, name string) *fleet.MDMWindowsConfigProfile {
		var prof fleet.MDMWindowsConfigProfile
		var teamID uint
		if tmID != nil {
			teamID = *tmID
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &prof,
				`SELECT profile_uuid, team_id, name, syncml, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`,
				teamID, name)
		})
		return &prof
	}

	withTeamID := func(p *fleet.MDMWindowsConfigProfile, tmID uint) *fleet.MDMWindowsConfigProfile {
		p.TeamID = &tmID
		return p
	}
	withUploadedAt := func(p *fleet.MDMWindowsConfigProfile, ua time.Time) *fleet.MDMWindowsConfigProfile {
		p.UploadedAt = ua
		return p
	}

	// apply empty set for no-team
	applyAndExpect(nil, nil, nil, false)

	// apply single profile set for tm1
	mTm1 := applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N1", "l1"),
	}, ptr.Uint(1), []*fleet.MDMWindowsConfigProfile{
		withTeamID(windowsConfigProfileForTest(t, "N1", "l1"), 1),
	}, true)
	profTm1N1 := getProfileByTeamAndName(ptr.Uint(1), "N1")

	// apply single profile set for no-team
	applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N1", "l1"),
	}, nil, []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N1", "l1"),
	}, true)

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// apply new profile set for tm1
	mTm1b := applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N1", "l1"), // unchanged
		windowsConfigProfileForTest(t, "N2", "l2"),
	}, ptr.Uint(1), []*fleet.MDMWindowsConfigProfile{
		withUploadedAt(withTeamID(windowsConfigProfileForTest(t, "N1", "l1"), 1), profTm1N1.UploadedAt),
		withTeamID(windowsConfigProfileForTest(t, "N2", "l2"), 1),
	}, true)
	// uuid for N1-I1 is unchanged
	require.Equal(t, mTm1["I1"], mTm1b["I1"])
	profTm1N2 := getProfileByTeamAndName(ptr.Uint(1), "N2")

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// apply edited profile (by content only), unchanged profile and new profile
	// for tm1
	mTm1c := applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N1", "l1b"), // content updated
		windowsConfigProfileForTest(t, "N2", "l2"),  // unchanged
		windowsConfigProfileForTest(t, "N3", "l3"),  // new
	}, ptr.Uint(1), []*fleet.MDMWindowsConfigProfile{
		withTeamID(windowsConfigProfileForTest(t, "N1", "l1b"), 1),
		withUploadedAt(withTeamID(windowsConfigProfileForTest(t, "N2", "l2"), 1), profTm1N2.UploadedAt),
		withTeamID(windowsConfigProfileForTest(t, "N3", "l3"), 1),
	}, true)
	// uuid for N1-I1 is unchanged
	require.Equal(t, mTm1b["I1"], mTm1c["I1"])
	// uuid for N2-I2 is unchanged
	require.Equal(t, mTm1b["I2"], mTm1c["I2"])

	profTm1N1c := getProfileByTeamAndName(ptr.Uint(1), "N1")
	// uploaded-at was modified because the content changed
	require.False(t, profTm1N1.UploadedAt.Equal(profTm1N1c.UploadedAt))

	// apply only new profiles to no-team
	applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, nil, []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, true)

	// apply the same thing again -- nothing updated
	applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, nil, []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, false)

	// Change the content of one profile -- update expected
	applyAndExpect([]*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4b"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, nil, []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "N4", "l4b"),
		windowsConfigProfileForTest(t, "N5", "l5"),
	}, true)

	// clear profiles for tm1
	applyAndExpect(nil, ptr.Uint(1), nil, true)
}

// if the label name starts with "exclude-", the label is considered an "exclude-any", otherwise
// it is an "include-all".
func windowsConfigProfileForTest(t *testing.T, name, locURI string, labels ...*fleet.Label) *fleet.MDMWindowsConfigProfile {
	prof := &fleet.MDMWindowsConfigProfile{
		Name: name,
		SyncML: []byte(fmt.Sprintf(`
			<Replace>
				<Item>
				  <Target>
					  <LocURI>%s</LocURI>
				  </Target>
				</Item>
			</Replace>
		`, locURI)),
	}

	for _, lbl := range labels {
		switch {
		case strings.HasPrefix(lbl.Name, "exclude-"):
			prof.LabelsExcludeAny = append(prof.LabelsExcludeAny, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		case strings.HasPrefix(lbl.Name, "include-any-"):
			prof.LabelsIncludeAny = append(prof.LabelsIncludeAny, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		default:
			prof.LabelsIncludeAll = append(prof.LabelsIncludeAll, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		}
	}

	return prof
}

func testSaveResponse(t *testing.T, ds *Datastore) {
	// Set up: 3 devices, 1 command, 1 response for 1 device
	enrolledDevice1 := createEnrolledDevice(t, ds)
	enrolledDevice2 := createEnrolledDevice(t, ds)
	enrolledDevice3 := createEnrolledDevice(t, ds)

	atomicCommandUUID := uuid.NewString()
	replaceCommandUUID := uuid.NewString()
	cmd := &fleet.MDMWindowsCommand{
		CommandUUID: atomicCommandUUID,
		RawCommand: []byte(fmt.Sprintf(`
<Atomic>
	<!-- CmdID generated by Fleet -->
	<CmdID>%s</CmdID>
	<Replace>
		<!-- CmdID generated by Fleet -->
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
			</Target>
			<Meta>
				<Format
					xmlns="syncml:metinf">int
				</Format>
			</Meta>
			<Data>1</Data>
		</Item>
	</Replace>
</Atomic>
`, atomicCommandUUID, replaceCommandUUID)),
		TargetLocURI: "",
	}
	cmdEntries := []enrichResponseEntry{
		{Type: "Atomic", StatusCode: 200, UUID: atomicCommandUUID},
		{Type: "Replace", StatusCode: 200, UUID: replaceCommandUUID},
	}
	err := ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
		[]string{enrolledDevice1.MDMDeviceID, enrolledDevice2.MDMDeviceID, enrolledDevice3.MDMDeviceID}, cmd)
	require.NoError(t, err)

	// We only found a batch update method, so we are using a single statement here to insert host profile, for simplicity.
	ExecAdhocSQL(t, ds, func(t sqlx.ExtContext) error {
		_, err := t.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive', ?)`, enrolledDevice1.HostUUID, atomicCommandUUID, uuid.NewString())
		return err
	})

	enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, cmdEntries)

	// Do test
	_, err = ds.MDMWindowsSaveResponse(context.Background(), enrolledDevice1, enrichedSyncML, []string{}, false)
	require.NoError(t, err)

	// Verify results
	results, err := ds.GetMDMWindowsCommandResults(context.Background(), cmd.CommandUUID, "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
	assert.Equal(t, cmd.CommandUUID, results[0].CommandUUID)
	assert.Equal(t, "200", results[0].Status)

	var count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ?",
			atomicCommandUUID)
	})
	assert.Equal(t, 3, count, "Queue rows are no longer deleted on ACK; all three devices should still be in the queue")

	// Finish setting up the second device for testing
	ExecAdhocSQL(t, ds, func(t sqlx.ExtContext) error {
		_, err := t.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive', ?)`, enrolledDevice2.HostUUID, atomicCommandUUID, uuid.NewString())
		return err
	})
	enrichedSyncML2 := createResponseAsEnrichedSyncML(t, enrolledDevice2, cmdEntries)

	// Do test on the second device
	_, err = ds.MDMWindowsSaveResponse(context.Background(), enrolledDevice2, enrichedSyncML2, []string{}, false)
	require.NoError(t, err)

	// Verify results for the second device
	results, err = ds.GetMDMWindowsCommandResults(context.Background(), cmd.CommandUUID, "")
	require.NoError(t, err)
	require.Len(t, results, 2)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ?",
			atomicCommandUUID)
	})
	assert.Equal(t, 3, count, "Queue rows are no longer deleted on ACK; all three devices should still be in the queue")

	// Third device, which in our test case failed and will have it's command resent
	ExecAdhocSQL(t, ds, func(t sqlx.ExtContext) error {
		_, err := t.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive', ?)`, enrolledDevice3.HostUUID, atomicCommandUUID, uuid.NewString())
		return err
	})
	enrichedSyncML3 := createResponseAsEnrichedSyncML(t, enrolledDevice3, cmdEntries)

	// Do test on the third device
	_, err = ds.MDMWindowsSaveResponse(context.Background(), enrolledDevice3, enrichedSyncML3, []string{atomicCommandUUID}, false)
	require.NoError(t, err)

	// Verify results does not exist for the third device
	results, err = ds.GetMDMWindowsCommandResults(context.Background(), cmd.CommandUUID, "")
	require.NoError(t, err)
	require.Len(t, results, 2) // still two
	for _, res := range results {
		assert.NotEqual(t, enrolledDevice3.HostUUID, res.HostUUID, "Host 3 should not have a result recorded")
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ?",
			atomicCommandUUID)
	})
	// Queue rows persist after ACK; device 3's command was excluded from processing (being resent),
	// so all three queue rows remain.
	assert.Equal(t, 3, count, "Queue rows are no longer deleted on ACK; all three devices should still be in the queue")

	t.Run("non-atomic command saves and verifies correctly", func(t *testing.T) {
		replaceCommandUUID := uuid.NewString()
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID: replaceCommandUUID,
			RawCommand: fmt.Appendf([]byte{}, `

	<Replace>
		<!-- CmdID generated by Fleet -->
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
			</Target>
			<Meta>
				<Format
					xmlns="syncml:metinf">int
				</Format>
			</Meta>
			<Data>1</Data>
		</Item>
	</Replace>
`, replaceCommandUUID),
			TargetLocURI: "",
		}
		cmdEntries := []enrichResponseEntry{
			{Type: "Replace", StatusCode: 200, UUID: replaceCommandUUID},
		}
		err := ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, cmd)
		require.NoError(t, err)

		// We only found a batch update method, so we are using a single statement here to insert host profile, for simplicity.
		ExecAdhocSQL(t, ds, func(t sqlx.ExtContext) error {
			_, err := t.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive', ?)`, enrolledDevice1.HostUUID, replaceCommandUUID, uuid.NewString())
			return err
		})

		enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, cmdEntries)
		// Do test
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
		require.NoError(t, err)

		// Verify results
		results, err := ds.GetMDMWindowsCommandResults(t.Context(), cmd.CommandUUID, "")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
		assert.Equal(t, cmd.CommandUUID, results[0].CommandUUID)
		assert.Equal(t, "200", results[0].Status)
	})

	t.Run("combined atomic and non-atomic profiles", func(t *testing.T) {
		getAtomicAndReplaceCommands := func() (string, string, *fleet.MDMWindowsCommand, *fleet.MDMWindowsCommand) {
			atomicCommandUUID := uuid.NewString()
			replaceCommandUUID := uuid.NewString()
			atomicCmd := &fleet.MDMWindowsCommand{
				CommandUUID: atomicCommandUUID,
				RawCommand: fmt.Appendf([]byte{}, `
	<Atomic>
		<!-- CmdID generated by Fleet -->
		<CmdID>%s</CmdID>
		<Replace>
			<!-- CmdID generated by Fleet -->
			<CmdID>%s</CmdID>
			<Item>
				<Target>
					<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
				</Target>
				<Meta>
					<Format
						xmlns="syncml:metinf">int
					</Format>
				</Meta>
				<Data>1</Data>
			</Item>
		</Replace>
	</Atomic>
`, atomicCommandUUID, uuid.NewString()),
				TargetLocURI: "",
			}
			replaceCommand := &fleet.MDMWindowsCommand{
				CommandUUID: replaceCommandUUID,
				RawCommand: fmt.Appendf([]byte{}, `
	<Replace>
		<!-- CmdID generated by Fleet -->
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
			</Target>
			<Meta>
				<Format
					xmlns="syncml:metinf">int
				</Format>
			</Meta>
			<Data>1</Data>
		</Item>
	</Replace>
`, replaceCommandUUID),
				TargetLocURI: "",
			}
			return atomicCommandUUID, replaceCommandUUID, atomicCmd, replaceCommand
		}

		t.Run("saves correctly", func(t *testing.T) {
			atomicCommandUUID, replaceCommandUUID, atomicCmd, replaceCmd := getAtomicAndReplaceCommands()
			cmdEntries := []enrichResponseEntry{
				{Type: "Atomic", StatusCode: 200, UUID: atomicCommandUUID},
				{Type: "Replace", StatusCode: 200, UUID: replaceCommandUUID},
			}
			err := ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
				[]string{enrolledDevice1.MDMDeviceID}, atomicCmd)
			require.NoError(t, err)
			err = ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
				[]string{enrolledDevice1.MDMDeviceID}, replaceCmd)
			require.NoError(t, err)

			// We only found a batch update method, so we are using a single statement here to insert host profile, for simplicity.
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive-atomic-success', ?)`, enrolledDevice1.HostUUID, atomicCommandUUID, uuid.NewString())
				require.NoError(t, err)
				_, err = q.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive-replace-success', ?)`, enrolledDevice1.HostUUID, replaceCommandUUID, uuid.NewString())
				return err
			})

			enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, cmdEntries)
			// Do test
			_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
			require.NoError(t, err)

			// Verify results
			results, err := ds.GetMDMWindowsCommandResults(t.Context(), atomicCmd.CommandUUID, "")
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
			assert.Equal(t, atomicCmd.CommandUUID, results[0].CommandUUID)
			assert.Equal(t, "200", results[0].Status)

			results, err = ds.GetMDMWindowsCommandResults(t.Context(), replaceCmd.CommandUUID, "")
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
			assert.Equal(t, replaceCmd.CommandUUID, results[0].CommandUUID)
			assert.Equal(t, "200", results[0].Status)

			// Query the profile status for the host profiles where the atomic and replace commands were assigned
			var atomicProfileStatus, replaceProfileStatus string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(t.Context(), q, &atomicProfileStatus, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, atomicCmd.CommandUUID)
			})
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(t.Context(), q, &replaceProfileStatus, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, replaceCmd.CommandUUID)
			})
			assert.Equal(t, "verified", atomicProfileStatus)
			assert.Equal(t, "verified", replaceProfileStatus)
		})

		t.Run("fails only for the failed profile", func(t *testing.T) {
			atomicCommandUUID, replaceCommandUUID, atomicCmd, replaceCmd := getAtomicAndReplaceCommands()
			cmdEntries := []enrichResponseEntry{
				{Type: "Atomic", StatusCode: 200, UUID: atomicCommandUUID},
				{Type: "Replace", StatusCode: 405, UUID: replaceCommandUUID},
			}
			err := ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
				[]string{enrolledDevice1.MDMDeviceID}, atomicCmd)
			require.NoError(t, err)
			err = ds.mdmWindowsInsertCommandForHostsDB(context.Background(), ds.primary,
				[]string{enrolledDevice1.MDMDeviceID}, replaceCmd)
			require.NoError(t, err)

			// We only found a batch update method, so we are using a single statement here to insert host profile, for simplicity.
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive-atomic-failure', ?)`, enrolledDevice1.HostUUID, atomicCommandUUID, uuid.NewString())
				require.NoError(t, err)
				_, err = q.ExecContext(context.Background(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'disable-onedrive-replace-failure', ?)`, enrolledDevice1.HostUUID, replaceCommandUUID, uuid.NewString())
				return err
			})

			enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, cmdEntries)
			// Do test
			_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
			require.NoError(t, err)

			// Verify results
			results, err := ds.GetMDMWindowsCommandResults(t.Context(), atomicCmd.CommandUUID, "")
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
			assert.Equal(t, atomicCmd.CommandUUID, results[0].CommandUUID)
			assert.Equal(t, "200", results[0].Status)

			results, err = ds.GetMDMWindowsCommandResults(t.Context(), replaceCmd.CommandUUID, "")
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, enrolledDevice1.HostUUID, results[0].HostUUID)
			assert.Equal(t, replaceCmd.CommandUUID, results[0].CommandUUID)
			assert.Equal(t, "405", results[0].Status)

			// Query the profile status for the host profiles where the atomic and replace commands were assigned
			var atomicProfileStatus, replaceProfileStatus *string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(t.Context(), q, &atomicProfileStatus, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, atomicCmd.CommandUUID)
			})
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(t.Context(), q, &replaceProfileStatus, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, replaceCmd.CommandUUID)
			})
			assert.Equal(t, "verified", *atomicProfileStatus)
			assert.Nil(t, replaceProfileStatus) // We want nil here, as the retry kicks in on failures
		})
	})

	t.Run("remove status outcomes", func(t *testing.T) {
		// Both verified and failed removes are terminal (best-effort removal)
		// and should be deleted from host_mdm_windows_profiles.
		cases := []struct {
			name       string
			statusCode int
		}{
			{"verified remove deletes row", 200},
			{"failed remove deletes row", 418}, // not in the "treated as success" list, maps to Failed
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				deleteCommandUUID := uuid.NewString()
				cmd := &fleet.MDMWindowsCommand{
					CommandUUID: deleteCommandUUID,
					RawCommand: fmt.Appendf([]byte{}, `
	<Delete>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
			</Target>
		</Item>
	</Delete>
`, deleteCommandUUID),
				}
				err := ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
					[]string{enrolledDevice1.MDMDeviceID}, cmd)
				require.NoError(t, err)

				profileUUID := uuid.NewString()
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(t.Context(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'verifying', 'remove', ?, 'disable-onedrive', ?)`, enrolledDevice1.HostUUID, deleteCommandUUID, profileUUID)
					return err
				})

				enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, []enrichResponseEntry{
					{Type: "Delete", StatusCode: tc.statusCode, UUID: deleteCommandUUID},
				})
				_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
				require.NoError(t, err)

				var count int
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(t.Context(), q, &count, `
SELECT COUNT(*) FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, deleteCommandUUID)
				})
				assert.Equal(t, 0, count, "terminal remove row should be deleted")
			})
		}
	})

	t.Run("mixed install and remove in same batch", func(t *testing.T) {
		// Install profile (Replace command, status 200) + Remove profile (Delete command, status 200).
		replaceCommandUUID := uuid.NewString()
		replaceCmd := &fleet.MDMWindowsCommand{
			CommandUUID: replaceCommandUUID,
			RawCommand: fmt.Appendf([]byte{}, `
	<Replace>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
			</Target>
			<Meta><Format xmlns="syncml:metinf">int</Format></Meta>
			<Data>1</Data>
		</Item>
	</Replace>
`, replaceCommandUUID),
			TargetLocURI: "",
		}
		deleteCommandUUID := uuid.NewString()
		deleteCmd := &fleet.MDMWindowsCommand{
			CommandUUID: deleteCommandUUID,
			RawCommand: fmt.Appendf([]byte{}, `
	<Delete>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/SomeOtherSetting</LocURI>
			</Target>
		</Item>
	</Delete>
`, deleteCommandUUID),
			TargetLocURI: "",
		}

		err := ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, replaceCmd)
		require.NoError(t, err)
		err = ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, deleteCmd)
		require.NoError(t, err)

		installProfileUUID := uuid.NewString()
		removeProfileUUID := uuid.NewString()
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'install-profile', ?)`, enrolledDevice1.HostUUID, replaceCommandUUID, installProfileUUID)
			require.NoError(t, err)
			_, err = q.ExecContext(t.Context(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'verifying', 'remove', ?, 'remove-profile', ?)`, enrolledDevice1.HostUUID, deleteCommandUUID, removeProfileUUID)
			return err
		})

		cmdEntries := []enrichResponseEntry{
			{Type: "Replace", StatusCode: 200, UUID: replaceCommandUUID},
			{Type: "Delete", StatusCode: 200, UUID: deleteCommandUUID},
		}
		enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, cmdEntries)
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
		require.NoError(t, err)

		// Install profile should be upserted with verified status.
		var installStatus string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &installStatus, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, replaceCommandUUID)
		})
		assert.Equal(t, "verified", installStatus, "install profile should be verified")

		// Remove profile should be deleted.
		var removeCount int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &removeCount, `
SELECT COUNT(*) FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND command_uuid = ?`, enrolledDevice1.HostUUID, deleteCommandUUID)
		})
		assert.Equal(t, 0, removeCount, "verified remove row should be deleted")
	})

	t.Run("remove then reinstall same profile", func(t *testing.T) {
		// Validates that deleting a verified-remove row by command_uuid does not
		// interfere with a fresh install of the same (host_uuid, profile_uuid) pair.
		profileUUID := uuid.NewString()

		// Step 1: create a remove row and ACK it as verified → row deleted.
		removeCommandUUID := uuid.NewString()
		removeCmd := &fleet.MDMWindowsCommand{
			CommandUUID: removeCommandUUID,
			RawCommand: fmt.Appendf([]byte{}, `
	<Delete>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/ReinstallTest</LocURI>
			</Target>
		</Item>
	</Delete>
`, removeCommandUUID),
		}
		err := ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, removeCmd)
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'verifying', 'remove', ?, 'reinstall-test', ?)`, enrolledDevice1.HostUUID, removeCommandUUID, profileUUID)
			return err
		})
		enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, []enrichResponseEntry{
			{Type: "Delete", StatusCode: 200, UUID: removeCommandUUID},
		})
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
		require.NoError(t, err)

		var count int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &count, `
SELECT COUNT(*) FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND profile_uuid = ?`, enrolledDevice1.HostUUID, profileUUID)
		})
		require.Equal(t, 0, count, "remove row should be deleted after verified ACK")

		// Step 2: fresh install of the same profile with a new command UUID.
		installCommandUUID := uuid.NewString()
		installCmd := &fleet.MDMWindowsCommand{
			CommandUUID: installCommandUUID,
			RawCommand: fmt.Appendf([]byte{}, `
	<Replace>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Policy/Config/System/ReinstallTest</LocURI>
			</Target>
			<Meta><Format xmlns="syncml:metinf">int</Format></Meta>
			<Data>1</Data>
		</Item>
	</Replace>
`, installCommandUUID),
		}
		err = ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, installCmd)
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(), `
INSERT INTO host_mdm_windows_profiles (host_uuid, status, operation_type, command_uuid, profile_name, profile_uuid)
VALUES (?, 'pending', 'install', ?, 'reinstall-test', ?)`, enrolledDevice1.HostUUID, installCommandUUID, profileUUID)
			return err
		})
		enrichedSyncML = createResponseAsEnrichedSyncML(t, enrolledDevice1, []enrichResponseEntry{
			{Type: "Replace", StatusCode: 200, UUID: installCommandUUID},
		})
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, false)
		require.NoError(t, err)

		// The reinstalled profile should land as verified.
		var status string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &status, `
SELECT status FROM host_mdm_windows_profiles
WHERE host_uuid = ? AND profile_uuid = ?`, enrolledDevice1.HostUUID, profileUUID)
		})
		assert.Equal(t, "verified", status, "reinstalled profile should be verified")
	})

	t.Run("wipe failure returns WipeFailed result", func(t *testing.T) {
		ctx := t.Context()

		// Create a real host and enroll it in Windows MDM.
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        "test-wipe-host",
			OsqueryHostID:   ptr.String("osquery-wipe-" + uuid.NewString()),
			NodeKey:         ptr.String("nodekey-wipe-" + uuid.NewString()),
			UUID:            uuid.NewString(),
			Platform:        "windows",
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		deviceID := windowsEnroll(t, ds, host)
		enrolled, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
		require.NoError(t, err)

		// buildWipeResponse constructs an EnrichedSyncML that contains a status
		// entry for the given wipe command UUID. It sets Cmd to nil so that the
		// profile-payload builder is not triggered (wipe commands are not profiles).
		buildWipeResponse := func(t *testing.T, cmdUUID string, statusCode string) fleet.EnrichedSyncML {
			t.Helper()

			// Start from a minimal valid response (the helper gives us the SyncHdr ack).
			resp := createResponseAsEnrichedSyncML(t, enrolled, []enrichResponseEntry{})
			// Add the wipe command UUID with the desired status but without Cmd
			// so BuildMDMWindowsProfilePayloadFromMDMResponse is not invoked.
			resp.CmdRefUUIDs = append(resp.CmdRefUUIDs, cmdUUID)
			if statusCode != "" {
				resp.CmdRefUUIDToStatus[cmdUUID] = fleet.SyncMLCmd{
					Data: &statusCode,
					// Cmd intentionally nil — wipe is not a profile command.
				}
			}
			return resp
		}

		t.Run("status 500 signals wipe failure", func(t *testing.T) {
			wipeCmdUUID := uuid.NewString()
			err := ds.WipeHostViaWindowsMDM(ctx, host, &fleet.MDMWindowsCommand{
				CommandUUID:  wipeCmdUUID,
				RawCommand:   []byte(`<Exec></Exec>`),
				TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
			})
			require.NoError(t, err)

			result, err := ds.MDMWindowsSaveResponse(ctx, enrolled, buildWipeResponse(t, wipeCmdUUID, "500"), []string{}, false)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.WipeFailed)
			assert.Equal(t, host.UUID, result.WipeFailed.HostUUID)
		})

		t.Run("duplicate response does not signal wipe failure again", func(t *testing.T) {
			wipeCmdUUID := uuid.NewString()
			err := ds.WipeHostViaWindowsMDM(ctx, host, &fleet.MDMWindowsCommand{
				CommandUUID:  wipeCmdUUID,
				RawCommand:   []byte(`<Exec></Exec>`),
				TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
			})
			require.NoError(t, err)

			// First response — failure, clears wipe_ref.
			resp := buildWipeResponse(t, wipeCmdUUID, "500")
			result, err := ds.MDMWindowsSaveResponse(ctx, enrolled, resp, []string{}, false)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.WipeFailed)

			// Queue row persists after ACK (no longer deleted), so no
			// re-insert is needed. Do NOT restore wipe_ref — it was
			// cleared by the first failure.

			// Second response — wipe_ref is already NULL, so 0 rows affected.
			resp2 := buildWipeResponse(t, wipeCmdUUID, "500")
			result, err = ds.MDMWindowsSaveResponse(ctx, enrolled, resp2, []string{}, false)
			require.NoError(t, err)
			if result != nil {
				assert.Nil(t, result.WipeFailed)
			}
		})

		t.Run("status 200 does not signal wipe failure", func(t *testing.T) {
			wipeCmdUUID := uuid.NewString()
			err := ds.WipeHostViaWindowsMDM(ctx, host, &fleet.MDMWindowsCommand{
				CommandUUID:  wipeCmdUUID,
				RawCommand:   []byte(`<Exec></Exec>`),
				TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
			})
			require.NoError(t, err)

			result, err := ds.MDMWindowsSaveResponse(ctx, enrolled, buildWipeResponse(t, wipeCmdUUID, "200"), []string{}, false)
			require.NoError(t, err)
			if result != nil {
				assert.Nil(t, result.WipeFailed)
			}
		})

		t.Run("empty status does not signal wipe failure", func(t *testing.T) {
			wipeCmdUUID := uuid.NewString()
			err := ds.WipeHostViaWindowsMDM(ctx, host, &fleet.MDMWindowsCommand{
				CommandUUID:  wipeCmdUUID,
				RawCommand:   []byte(`<Exec></Exec>`),
				TargetLocURI: "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected",
			})
			require.NoError(t, err)

			// Pass empty status — the command UUID is in CmdRefUUIDs but has
			// no entry in CmdRefUUIDToStatus, so statusCode stays "".
			result, err := ds.MDMWindowsSaveResponse(ctx, enrolled, buildWipeResponse(t, wipeCmdUUID, ""), []string{}, false)
			require.NoError(t, err)
			if result != nil {
				assert.Nil(t, result.WipeFailed)
			}
		})
	})

	t.Run("saveFullResponse stores envelope and links response_id", func(t *testing.T) {
		cmdUUID := uuid.NewString()
		cmd := &fleet.MDMWindowsCommand{
			CommandUUID: cmdUUID,
			RawCommand: fmt.Appendf([]byte{}, `
	<Exec>
		<CmdID>%s</CmdID>
		<Item>
			<Target>
				<LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
			</Target>
		</Item>
	</Exec>
`, cmdUUID),
			TargetLocURI: "./Device/Vendor/MSFT/Reboot/RebootNow",
		}
		err := ds.mdmWindowsInsertCommandForHostsDB(t.Context(), ds.primary,
			[]string{enrolledDevice1.MDMDeviceID}, cmd)
		require.NoError(t, err)

		enrichedSyncML := createResponseAsEnrichedSyncML(t, enrolledDevice1, []enrichResponseEntry{
			{Type: "Exec", StatusCode: 200, UUID: cmdUUID},
		})

		// Count responses before.
		var beforeCount int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &beforeCount,
				"SELECT COUNT(*) FROM windows_mdm_responses")
		})

		// Save with saveFullResponse=true.
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML, []string{}, true)
		require.NoError(t, err)

		// A new windows_mdm_responses row should exist.
		var afterCount int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &afterCount,
				"SELECT COUNT(*) FROM windows_mdm_responses")
		})
		assert.Equal(t, beforeCount+1, afterCount, "saveFullResponse=true should insert a response row")

		// The command_results row should have a non-NULL response_id.
		var responseID *int64
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &responseID,
				"SELECT response_id FROM windows_mdm_command_results WHERE enrollment_id = ? AND command_uuid = ?", enrolledDevice1.ID, cmdUUID)
		})
		require.NotNil(t, responseID, "response_id should be set when saveFullResponse=true")

		// Re-ack the same command with saveFullResponse=true — response_id should update.
		// Re-enqueue the command first.
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(),
				`INSERT INTO windows_mdm_command_queue (enrollment_id, command_uuid) VALUES (?, ?)`,
				enrolledDevice1.ID, cmdUUID)
			return err
		})

		enrichedSyncML2 := createResponseAsEnrichedSyncML(t, enrolledDevice1, []enrichResponseEntry{
			{Type: "Exec", StatusCode: 200, UUID: cmdUUID},
		})
		_, err = ds.MDMWindowsSaveResponse(t.Context(), enrolledDevice1, enrichedSyncML2, []string{}, true)
		require.NoError(t, err)

		var newResponseID *int64
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(t.Context(), q, &newResponseID,
				"SELECT response_id FROM windows_mdm_command_results WHERE enrollment_id = ? AND command_uuid = ?", enrolledDevice1.ID, cmdUUID)
		})
		require.NotNil(t, newResponseID)
		assert.NotEqual(t, *responseID, *newResponseID, "response_id should update on duplicate when saveFullResponse=true")
	})
}

type enrichResponseEntry struct {
	Type       string
	StatusCode int
	UUID       string
}

func createResponseAsEnrichedSyncML(t *testing.T, enrolledDevice *fleet.MDMWindowsEnrolledDevice, entries []enrichResponseEntry) fleet.EnrichedSyncML {
	rawResponse := fmt.Sprintf(`
<SyncML
    xmlns="SYNCML:SYNCML1.2">
    <SyncHdr>
        <VerDTD>1.2</VerDTD>
        <VerProto>DM/1.2</VerProto>
        <SessionID>81</SessionID>
        <MsgID>2</MsgID>
        <Target>
            <LocURI>https://example.com/api/mdm/microsoft/management</LocURI>
        </Target>
        <Source>
            <LocURI>%s</LocURI>
        </Source>
    </SyncHdr>
    <SyncBody>
        <Status>
            <CmdID>1</CmdID>
            <MsgRef>1</MsgRef>
            <CmdRef>0</CmdRef>
            <Cmd>SyncHdr</Cmd>
            <Data>200</Data>
        </Status>`, enrolledDevice.MDMDeviceID)

	for i, entry := range entries {
		rawResponse += fmt.Sprintf(`
		<Status>
			<CmdID>%d</CmdID>
			<MsgRef>1</MsgRef>
			<CmdRef>%s</CmdRef>
			<Cmd>%s</Cmd>
			<Data>%d</Data>
		</Status>`, i+2, entry.UUID, entry.Type, entry.StatusCode)
	}
	rawResponse += `
        <Final/>
    </SyncBody>
</SyncML>
`
	syncML := &fleet.SyncML{}
	err := xml.Unmarshal([]byte(rawResponse), syncML)
	require.NoError(t, err)
	syncML.Raw = []byte(rawResponse)
	enrichedSyncML := fleet.NewEnrichedSyncML(syncML)
	return enrichedSyncML
}

func createEnrolledDevice(t *testing.T, ds *Datastore) *fleet.MDMWindowsEnrolledDevice {
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               uuid.NewString(),
	}
	err := ds.MDMWindowsInsertEnrolledDevice(context.Background(), enrolledDevice)
	require.NoError(t, err)
	enrolledDevice.ID = mdmWindowsEnrollmentIDByHardwareID(context.Background(), t, ds, enrolledDevice.MDMHardwareID)
	return enrolledDevice
}

func testSetMDMWindowsProfilesWithVariables(t *testing.T, ds *Datastore) {
	// NOTE: as of this code being written, Fleet variables are not yet supported
	// in Windows profiles, but the profile-variable batch-association function
	// is already implemented as platform-independent (as it was not
	// harder/longer to do this way). This just sanity-checks that the function
	// works as expected for Windows.

	ctx := context.Background()

	checkProfileVariables := func(profUUID string, teamID uint, wantVars []fleet.FleetVarName) {
		var gotVars []string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &gotVars, `
				SELECT
					fv.name
				FROM
					mdm_windows_configuration_profiles mwcp
					INNER JOIN mdm_configuration_profile_variables mcpv ON mwcp.profile_uuid = mcpv.windows_profile_uuid
					INNER JOIN fleet_variables fv ON mcpv.fleet_variable_id = fv.id
				WHERE
					mwcp.name = ? AND
					mwcp.team_id = ?`, "name-"+profUUID, teamID) // test profiles are created with a name = "name-" + uuid
		})
		wantVarStrings := make([]string, len(wantVars))
		for i := range wantVars {
			wantVarStrings[i] = "FLEET_VAR_" + string(wantVars[i])
		}
		require.ElementsMatch(t, wantVarStrings, gotVars)
	}

	globalProfiles := []string{
		InsertWindowsProfileForTest(t, ds, 0),
		InsertWindowsProfileForTest(t, ds, 0),
	}

	// both profiles have no variable
	_, err := batchSetProfileVariableAssociationsDB(ctx, ds.writer(ctx), []fleet.MDMProfileUUIDFleetVariables{
		{ProfileUUID: globalProfiles[0], FleetVariables: nil},
		{ProfileUUID: globalProfiles[1], FleetVariables: nil},
	}, "windows", false)
	require.NoError(t, err)

	checkProfileVariables(globalProfiles[0], 0, nil)
	checkProfileVariables(globalProfiles[1], 0, nil)

	// add some variables
	_, err = batchSetProfileVariableAssociationsDB(ctx, ds.writer(ctx), []fleet.MDMProfileUUIDFleetVariables{
		{ProfileUUID: globalProfiles[0], FleetVariables: []fleet.FleetVarName{fleet.FleetVarHostEndUserIDPUsername, fleet.FleetVarName(string(fleet.FleetVarDigiCertDataPrefix) + "ZZZ")}},
		{ProfileUUID: globalProfiles[1], FleetVariables: []fleet.FleetVarName{fleet.FleetVarHostEndUserIDPGroups}},
	}, "windows", false)
	require.NoError(t, err)

	checkProfileVariables(globalProfiles[0], 0, []fleet.FleetVarName{fleet.FleetVarHostEndUserIDPUsername, fleet.FleetVarDigiCertDataPrefix})
	checkProfileVariables(globalProfiles[1], 0, []fleet.FleetVarName{fleet.FleetVarHostEndUserIDPGroups})
}

func testWindowsMDMManagedSCEPCertificates(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	testCases := []struct {
		name                 string
		caName               string
		caType               fleet.CAConfigAssetType
		challengeRetrievedAt *time.Time
	}{
		/* 		{
			name:                 "NDES",
			caName:               "ndes",
			caType:               fleet.CAConfigNDES,
			challengeRetrievedAt: ptr.Time(time.Now().Add(-time.Hour).UTC().Round(time.Microsecond)),
		}, */
		{
			name:                 "Custom SCEP",
			caName:               "test-ca",
			caType:               fleet.CAConfigCustomSCEPProxy,
			challengeRetrievedAt: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			caName := tc.caName
			caType := tc.caType
			challengeRetrievedAt := tc.challengeRetrievedAt

			profileUUID := uuid.NewString()
			dummySyncML := generateDummyWindowsProfile(profileUUID)
			dummyCP := fleet.MDMWindowsConfigProfile{
				Name:   tc.caName,
				SyncML: dummySyncML,
			}
			initialCP, err := ds.NewMDMWindowsConfigProfile(ctx, dummyCP, nil)
			require.NoError(t, err)

			host, err := ds.NewHost(ctx, &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				OsqueryHostID:   ptr.String("host0-osquery-id" + tc.caName),
				NodeKey:         ptr.String("host0-node-key" + tc.caName),
				UUID:            "host0-test-mdm-profiles" + tc.caName,
				Hostname:        "hostname0",
			})
			require.NoError(t, err)

			// Host and profile are not linked
			profile, err := ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
			require.NoError(t, err)
			assert.Nil(t, profile)

			err = ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
				{
					ProfileUUID:   initialCP.ProfileUUID,
					ProfileName:   initialCP.Name,
					HostUUID:      host.UUID,
					Status:        &fleet.MDMDeliveryPending,
					OperationType: fleet.MDMOperationTypeInstall,
					CommandUUID:   "command-uuid",
					Checksum:      []byte("checksum"),
				},
			},
			)
			require.NoError(t, err)

			// Host and profile do not have certificate metadata
			profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
			require.NoError(t, err)
			assert.Nil(t, profile)

			// Initial certificate state where a host has been requested to install but we have no metadata
			err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
				{
					HostUUID:             host.UUID,
					ProfileUUID:          initialCP.ProfileUUID,
					ChallengeRetrievedAt: challengeRetrievedAt,
					Type:                 caType,
					CAName:               caName,
				},
			})
			require.NoError(t, err)

			// Check that the managed certificate was inserted correctly
			profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
			require.NoError(t, err)
			require.NotNil(t, profile)
			assert.Equal(t, host.UUID, profile.HostUUID)
			assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
			assert.Equal(t, challengeRetrievedAt, profile.ChallengeRetrievedAt)
			assert.Equal(t, caType, profile.Type)
			assert.Nil(t, profile.Serial)
			assert.Nil(t, profile.NotValidBefore)
			assert.Nil(t, profile.NotValidAfter)
			assert.Equal(t, caName, profile.CAName)

			// Renew should not do anything yet
			err = ds.RenewMDMManagedCertificates(ctx)
			require.NoError(t, err)
			profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
			require.NoError(t, err)
			require.NotNil(t, profile.Status)
			assert.Equal(t, fleet.MDMDeliveryPending, *profile.Status)

			// Cleanup should do nothing
			err = ds.CleanUpMDMManagedCertificates(ctx)
			require.NoError(t, err)
			profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
			require.NoError(t, err)
			require.NotNil(t, profile)

			serial := "8ABADCAFEF684D6348F5EC95AEFF468F237A9D75"

			t.Run("Non renewal scenario 1 - validity window > 30 days but not yet time to renew", func(t *testing.T) {
				// Set not_valid_before to 1 day in the past and not_valid_after to 31 days in the future so
				// the validity window is 32 days of which there are 31 left which should not trigger renewal
				notValidAfter := time.Now().Add(31 * 24 * time.Hour).UTC().Round(time.Microsecond)
				notValidBefore := time.Now().Add(-1 * 24 * time.Hour).UTC().Round(time.Microsecond)
				err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
					{
						HostUUID:             host.UUID,
						ProfileUUID:          initialCP.ProfileUUID,
						ChallengeRetrievedAt: challengeRetrievedAt,
						NotValidBefore:       &notValidBefore,
						NotValidAfter:        &notValidAfter,
						Type:                 caType,
						CAName:               caName,
						Serial:               &serial,
					},
				})
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `
						UPDATE host_mdm_windows_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?
					`, fleet.MDMDeliveryVerified, host.UUID, initialCP.ProfileUUID)
					if err != nil {
						return err
					}
					return nil
				})

				// Verify the policy is not currently marked for resend and that the upsert executed correctly
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				assert.Equal(t, host.UUID, profile.HostUUID)
				assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
				assert.Equal(t, challengeRetrievedAt, profile.ChallengeRetrievedAt)
				assert.Equal(t, &notValidBefore, profile.NotValidBefore)
				assert.Equal(t, &notValidAfter, profile.NotValidAfter)
				assert.Equal(t, caType, profile.Type)
				require.NotNil(t, profile.Serial)
				assert.Equal(t, serial, *profile.Serial)
				assert.Equal(t, caName, profile.CAName)

				// Renew should not change the MDM delivery status
				err = ds.RenewMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				// Cleanup should do nothing
				err = ds.CleanUpMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile)
			})

			t.Run("Non renewal scenario 2 - validity window < 30 days but not yet time to renew", func(t *testing.T) {
				// Set not_valid_before to 13 days in the past and not_valid_after to 15 days in the future so
				// the validity window is 28 days of which there are 15 left which should not trigger renewal
				notValidAfter := time.Now().Add(15 * 24 * time.Hour).UTC().Round(time.Microsecond)
				notValidBefore := time.Now().Add(-13 * 24 * time.Hour).UTC().Round(time.Microsecond)
				err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
					{
						HostUUID:             host.UUID,
						ProfileUUID:          initialCP.ProfileUUID,
						ChallengeRetrievedAt: challengeRetrievedAt,
						NotValidBefore:       &notValidBefore,
						NotValidAfter:        &notValidAfter,
						Type:                 caType,
						CAName:               caName,
						Serial:               &serial,
					},
				})
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `
						UPDATE host_mdm_windows_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?
					`, fleet.MDMDeliveryVerified, host.UUID, initialCP.ProfileUUID)
					if err != nil {
						return err
					}
					return nil
				})

				// Verify the policy is not currently marked for resend and that the upsert executed correctly
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				assert.Equal(t, host.UUID, profile.HostUUID)
				assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
				assert.Equal(t, challengeRetrievedAt, profile.ChallengeRetrievedAt)
				assert.Equal(t, &notValidBefore, profile.NotValidBefore)
				assert.Equal(t, &notValidAfter, profile.NotValidAfter)
				assert.Equal(t, caType, profile.Type)
				require.NotNil(t, profile.Serial)
				assert.Equal(t, serial, *profile.Serial)
				assert.Equal(t, caName, profile.CAName)

				// Renew should not change the MDM delivery status
				err = ds.RenewMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				// Cleanup should do nothing
				err = ds.CleanUpMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile)
			})

			t.Run("Renew scenario 1 - validity window > 30 days", func(t *testing.T) {
				// Set not_valid_before to 31 days in the past the validity window becomes 60 days, of which there are
				// 29 left which should trigger the first renewal scenario(window > 30 days, renew when < 30
				// days left)
				notValidAfter := time.Now().Add(29 * 24 * time.Hour).UTC().Round(time.Microsecond)
				notValidBefore := time.Now().Add(-31 * 24 * time.Hour).UTC().Round(time.Microsecond)
				err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
					{
						HostUUID:             host.UUID,
						ProfileUUID:          initialCP.ProfileUUID,
						ChallengeRetrievedAt: challengeRetrievedAt,
						NotValidBefore:       &notValidBefore,
						NotValidAfter:        &notValidAfter,
						Type:                 caType,
						CAName:               caName,
						Serial:               &serial,
					},
				})
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `
						UPDATE host_mdm_windows_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?
					`, fleet.MDMDeliveryVerified, host.UUID, initialCP.ProfileUUID)
					if err != nil {
						return err
					}
					return nil
				})

				// Verify the policy is not currently marked for resend and that the upsert executed correctly
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				assert.Equal(t, host.UUID, profile.HostUUID)
				assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
				assert.Equal(t, challengeRetrievedAt, profile.ChallengeRetrievedAt)
				assert.Equal(t, &notValidBefore, profile.NotValidBefore)
				assert.Equal(t, &notValidAfter, profile.NotValidAfter)
				assert.Equal(t, caType, profile.Type)
				require.NotNil(t, profile.Serial)
				assert.Equal(t, serial, *profile.Serial)
				assert.Equal(t, caName, profile.CAName)

				// Renew should set the MDM delivery status to "null" so the profile gets resent and the certificate renewed
				err = ds.RenewMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.Nil(t, profile.Status)

				// Cleanup should do nothing
				err = ds.CleanUpMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile)
			})

			t.Run("Renew scenario 2 - validity window < 30 days", func(t *testing.T) {
				// Set not_valid_before to 15 days in the past and not_valid_after to 14 days in the future so the
				// validity window becomes 29 days, of which there are 14 left which should trigger the second
				// renewal scenario(window < 30 days, renew when there is half that time left)
				notValidBefore := time.Now().Add(-15 * 24 * time.Hour).UTC().Round(time.Microsecond)
				notValidAfter := time.Now().Add(14 * 24 * time.Hour).UTC().Round(time.Microsecond)
				err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
					{
						HostUUID:             host.UUID,
						ProfileUUID:          initialCP.ProfileUUID,
						ChallengeRetrievedAt: challengeRetrievedAt,
						NotValidBefore:       &notValidBefore,
						NotValidAfter:        &notValidAfter,
						Type:                 caType,
						CAName:               caName,
						Serial:               &serial,
					},
				})
				require.NoError(t, err)

				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `
					UPDATE host_mdm_windows_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?
				`, fleet.MDMDeliveryVerified, host.UUID, initialCP.ProfileUUID)
					if err != nil {
						return err
					}
					return nil
				})
				require.NoError(t, err)

				// Verify the policy is not currently marked for resend and that the upsert executed correctly
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile.Status)
				assert.Equal(t, fleet.MDMDeliveryVerified, *profile.Status)

				assert.Equal(t, host.UUID, profile.HostUUID)
				assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
				assert.Equal(t, challengeRetrievedAt, profile.ChallengeRetrievedAt)
				assert.Equal(t, &notValidBefore, profile.NotValidBefore)
				assert.Equal(t, &notValidAfter, profile.NotValidAfter)
				assert.Equal(t, caType, profile.Type)
				require.NotNil(t, profile.Serial)
				assert.Equal(t, serial, *profile.Serial)
				assert.Equal(t, caName, profile.CAName)

				// Renew should set the MDM delivery status to "null" so the profile gets resent and the certificate renewed
				err = ds.RenewMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.Nil(t, profile.Status)

				// Cleanup should do nothing
				err = ds.CleanUpMDMManagedCertificates(ctx)
				require.NoError(t, err)
				profile, err = ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID, caName)
				require.NoError(t, err)
				require.NotNil(t, profile)
			})
		})
	}
}

func testGetWindowsMDMCommandsForResending(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	topLevelCmdUUID := uuid.NewString()
	cmdUUID := uuid.NewString()

	// Create entry in mdm_windows_enrollments table for command queue
	dev := createMDMWindowsEnrollment(ctx, t, ds)

	// No commands in windows_mdm_commands so doesn't matter what we put in
	commands, err := ds.GetWindowsMDMCommandsForResending(ctx, dev.MDMDeviceID, []string{cmdUUID})
	require.NoError(t, err)
	require.Empty(t, commands)

	// Insert a command
	rawCommand := fmt.Appendf(nil, "<CmdID>%s</CmdID>", cmdUUID)
	err = ds.mdmWindowsInsertCommandForHostsDB(ctx, ds.writer(ctx), []string{dev.HostUUID}, &fleet.MDMWindowsCommand{
		CommandUUID: topLevelCmdUUID,
		RawCommand:  rawCommand,
	})
	require.NoError(t, err)

	//

	// Fetch command for resending
	commands, err = ds.GetWindowsMDMCommandsForResending(ctx, dev.MDMDeviceID, []string{cmdUUID})
	require.NoError(t, err)
	require.Len(t, commands, 1)
	assert.Equal(t, topLevelCmdUUID, commands[0].CommandUUID)
	assert.Equal(t, rawCommand, commands[0].RawCommand)

	// Check that we search raw body and not match on command_uuid
	commands, err = ds.GetWindowsMDMCommandsForResending(ctx, dev.MDMDeviceID, []string{topLevelCmdUUID})
	require.NoError(t, err)
	require.Empty(t, commands)
}

func testResendWindowsMDMCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	dev := createMDMWindowsEnrollment(ctx, t, ds)
	cmdUUID := uuid.NewString()

	// Query enrollment id from mdm_windows_enrollments
	var enrollmentID int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &enrollmentID, "SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?", dev.MDMDeviceID)
	})
	require.Greater(t, enrollmentID, int64(0), "Enrollment ID should be greater than 0")

	// Insert host profile entry
	err := ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
		{
			HostUUID:      dev.HostUUID,
			ProfileUUID:   uuid.NewString(),
			ProfileName:   "test-profile",
			Status:        &fleet.MDMDeliveryFailed,
			OperationType: fleet.MDMOperationTypeInstall,
			CommandUUID:   cmdUUID,
			Checksum:      []byte("checksum"),
			Detail:        "fake detail we expect to be cleared on resend",
		},
	})
	require.NoError(t, err)

	// Insert a command for the original profile
	cmdBody := []byte(`<Add></Add>`)
	cmd := &fleet.MDMWindowsCommand{
		CommandUUID: cmdUUID,
		RawCommand:  cmdBody,
	}
	err = ds.mdmWindowsInsertCommandForHostsDB(ctx, ds.writer(ctx), []string{dev.HostUUID}, cmd)
	require.NoError(t, err)

	// Verify we have a windows_mdm_command_queue entry
	var count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ? AND enrollment_id = ?",
			cmd.CommandUUID, enrollmentID)
	})
	assert.Equal(t, 1, count, "Command queue entry should exist before resend")

	// Resend command
	// We manually do replacement here
	newCmdUUID := uuid.NewString()
	newBody := []byte(`<Replace></Replace>`)
	newCmd := &fleet.MDMWindowsCommand{
		CommandUUID: newCmdUUID,
		RawCommand:  newBody,
	}
	err = ds.ResendWindowsMDMCommand(ctx, dev.MDMDeviceID, newCmd, cmd)
	require.NoError(t, err)

	// Verify we have a windows_mdm_command_queue entry for the new command
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ? AND enrollment_id = ?",
			newCmd.CommandUUID, enrollmentID)
	})
	assert.Equal(t, 1, count, "New command queue entry should exist after resend")

	// verify we don't have a windows_mdm_command_queue entry for the old command
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ? AND enrollment_id = ?",
			cmd.CommandUUID, enrollmentID)
	})
	assert.Equal(t, 0, count, "Old command queue entry should not exist after resend")

	// Verify host profile status is reset and detail cleared
	var status string
	var detail sql.NullString
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &status, "SELECT status FROM host_mdm_windows_profiles WHERE command_uuid = ? AND host_uuid = ?",
			newCmd.CommandUUID, dev.HostUUID)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &detail, "SELECT detail FROM host_mdm_windows_profiles WHERE command_uuid = ? AND host_uuid = ?",
			newCmd.CommandUUID, dev.HostUUID)
	})
	assert.Equal(t, string(fleet.MDMDeliveryPending), status, "Host profile status should be reset to pending on resend")
	require.True(t, detail.Valid, "Host profile detail should be cleared on resend")
	assert.Empty(t, detail.String, "Host profile detail should be cleared on resend")
}

func testDeleteProfileLocURIProtection(t *testing.T, ds *Datastore) {
	// One subtest below replicates the post-team-move reconciliation by
	// calling BulkSetPendingMDMHostProfiles and then asserts on
	// host_mdm_windows_profiles. Production defers that to the cron, so
	// this test opts into the eager hook.
	ds.EnableTestWindowsEagerHook(t)
	ctx := t.Context()

	h1 := test.NewHost(t, ds, "host1", "10.0.0.1", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform("windows"))
	windowsEnroll(t, ds, h1)

	h2 := test.NewHost(t, ds, "host2", "10.0.0.2", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform("windows"))
	windowsEnroll(t, ds, h2)

	// Profile A: LocURIs X, Y. Profile B: LocURI Y (shared with A).
	profA := &fleet.MDMWindowsConfigProfile{Name: "profA", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/X</LocURI></Target><Data>1</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Y</LocURI></Target><Data>1</Data></Item></Replace>`)}
	profB := &fleet.MDMWindowsConfigProfile{Name: "profB", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Y</LocURI></Target><Data>2</Data></Item></Replace>`)}

	// Insert both profiles.
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profA, profB}, nil)
		return err
	})
	require.NoError(t, err)
	profAUUID := windowsProfileUUIDByName(t, ds, "profA")
	profBUUID := windowsProfileUUIDByName(t, ds, "profB")

	// Simulate both profiles installed on both hosts.
	installWindowsProfilesAsVerified(t, ds, []string{h1.UUID, h2.UUID}, []string{profAUUID, profBUUID})

	t.Run("shared LocURI not deleted when other profile uses it", func(t *testing.T) {
		t.Cleanup(func() {
			TruncateTables(t, ds, "host_mdm_windows_profiles", "windows_mdm_command_queue", "windows_mdm_commands", "mdm_windows_configuration_profiles", "mdm_configuration_profile_labels", "label_membership")
		})

		// Delete profile A. LocURI Y is shared with B, so only X should be deleted.
		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profB}, nil)
			return err
		})
		require.NoError(t, err)

		// Verify the delete command on BOTH hosts.
		for _, h := range []*fleet.Host{h1, h2} {
			rawCmd := rawWindowsDeleteCommandForHostProfile(t, ds, h.UUID, profAUUID)
			require.NotEmpty(t, rawCmd, "host %s should have a delete command", h.Hostname)
			rawStr := string(rawCmd)
			assert.Contains(t, rawStr, "./Device/X", "host %s: should delete LocURI X", h.Hostname)
			assert.NotContains(t, rawStr, "./Device/Y", "host %s: should NOT delete LocURI Y (protected by profB)", h.Hostname)
		}
	})

	t.Run("label-scoped protector only protects hosts in scope", func(t *testing.T) {
		t.Cleanup(func() {
			TruncateTables(t, ds, "host_mdm_windows_profiles", "windows_mdm_command_queue", "windows_mdm_commands", "mdm_windows_configuration_profiles", "mdm_configuration_profile_labels", "label_membership")
		})

		// Profile A: LocURIs X, Y. Profile B: LocURI Y (shared), label-scoped to h1 only.
		// When A is deleted:
		//   - h1: Y is protected (B applies), only X is deleted
		//   - h2: Y is NOT protected (B doesn't apply), both X and Y are deleted
		profA2 := &fleet.MDMWindowsConfigProfile{Name: "ls-profA", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/X</LocURI></Target><Data>1</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Y</LocURI></Target><Data>1</Data></Item></Replace>`)}
		profB2 := &fleet.MDMWindowsConfigProfile{Name: "ls-profB", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Y</LocURI></Target><Data>2</Data></Item></Replace>`)}

		// Insert both profiles.
		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profA2, profB2}, nil)
			return err
		})
		require.NoError(t, err)

		profA2UUID := windowsProfileUUIDByName(t, ds, "ls-profA")
		profB2UUID := windowsProfileUUIDByName(t, ds, "ls-profB")

		// Create a label and make profB label-scoped to it.
		scopeLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "locuri-scope-label", Query: ""})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`,
				profB2UUID, scopeLabel.Name, scopeLabel.ID)
			return err
		})

		// Only h1 is a member of the label.
		err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{scopeLabel.ID, h1.ID}})
		require.NoError(t, err)

		// Simulate both profiles installed on both hosts. For profB, also add
		// an install row for h1 (simulating the reconciler assigned it based on
		// label membership).
		verified := fleet.MDMDeliveryVerified
		err = ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{ProfileUUID: profA2UUID, ProfileName: "ls-profA", HostUUID: h1.UUID, CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Status: &verified, Checksum: []byte{0}},
			{ProfileUUID: profA2UUID, ProfileName: "ls-profA", HostUUID: h2.UUID, CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Status: &verified, Checksum: []byte{0}},
			// profB only installed on h1 (label-scoped, reconciler only assigned it to h1).
			{ProfileUUID: profB2UUID, ProfileName: "ls-profB", HostUUID: h1.UUID, CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Status: &verified, Checksum: []byte{0}},
		})
		require.NoError(t, err)

		// Delete profA (keep profB).
		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profB2}, nil)
			return err
		})
		require.NoError(t, err)

		// h1: profB applies (label-scoped, h1 is in the label).
		// Y is protected, only X should be deleted.
		var h1Cmds [][]byte
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &h1Cmds,
				`SELECT wc.raw_command FROM windows_mdm_commands wc
				JOIN windows_mdm_command_queue cq ON cq.command_uuid = wc.command_uuid
				JOIN mdm_windows_enrollments mwe ON mwe.id = cq.enrollment_id
				WHERE mwe.host_uuid = ?`, h1.UUID)
		})
		require.NotEmpty(t, h1Cmds, "h1 should have delete commands")
		for _, cmd := range h1Cmds {
			s := string(cmd)
			if strings.Contains(s, "<Delete") {
				assert.Contains(t, s, "./Device/X", "h1: should delete X")
				assert.NotContains(t, s, "./Device/Y", "h1: should NOT delete Y (protected by label-scoped profB)")
			}
		}

		// h2: profB does NOT apply (h2 not in label).
		// Y is NOT protected, both X and Y should be deleted.
		var h2Cmds [][]byte
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &h2Cmds,
				`SELECT wc.raw_command FROM windows_mdm_commands wc
				JOIN windows_mdm_command_queue cq ON cq.command_uuid = wc.command_uuid
				JOIN mdm_windows_enrollments mwe ON mwe.id = cq.enrollment_id
				WHERE mwe.host_uuid = ?`, h2.UUID)
		})
		require.NotEmpty(t, h2Cmds, "h2 should have delete commands")
		h2HasX, h2HasY := false, false
		for _, cmd := range h2Cmds {
			s := string(cmd)
			if strings.Contains(s, "<Delete") {
				if strings.Contains(s, "./Device/X") {
					h2HasX = true
				}
				if strings.Contains(s, "./Device/Y") {
					h2HasY = true
				}
			}
		}
		assert.True(t, h2HasX, "h2: should delete X")
		assert.True(t, h2HasY, "h2: should delete Y (profB doesn't apply, not in label scope)")
	})

	// Reproduces the regression where deleting a no-team profile failed with
	// "expected profiles from 1 team, got 2" because some hosts that once had
	// the profile had since been moved to a different team.
	t.Run("stale rows from moved host do not block delete", func(t *testing.T) {
		t.Cleanup(func() {
			TruncateTables(t, ds, "host_mdm_windows_profiles", "windows_mdm_command_queue", "windows_mdm_commands", "mdm_windows_configuration_profiles")
		})

		prof := &fleet.MDMWindowsConfigProfile{Name: "no-team-prof", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Z</LocURI></Target><Data>1</Data></Item></Replace>`)}
		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{prof}, nil)
			return err
		})
		require.NoError(t, err)
		profUUID := windowsProfileUUIDByName(t, ds, "no-team-prof")

		// Both hosts start with the profile installed and verified.
		installWindowsProfilesAsVerified(t, ds, []string{h1.UUID, h2.UUID}, []string{profUUID})

		// Move h2 out of no-team; replicate the post-move reconciliation the
		// service layer does so h2's row flips to operation=remove, status=NULL.
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "moved-host-destination"})
		require.NoError(t, err)
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{h2.ID})))
		_, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{h2.ID}, nil, nil, nil)
		require.NoError(t, err)
		// Restore h2 to no-team at the end so the next subtest starts clean.
		t.Cleanup(func() {
			_ = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{h2.ID}))
		})

		// Insert a destination-team profile with the same LocURI. If LocURI
		// protection ever leaks across teams, this would be treated as a
		// protector for the no-team delete and the <Delete> would be empty.
		destTeamProf := &fleet.MDMWindowsConfigProfile{Name: "dest-team-prof", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Z</LocURI></Target><Data>2</Data></Item></Replace>`)}
		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, &team.ID, []*fleet.MDMWindowsConfigProfile{destTeamProf}, nil)
			return err
		})
		require.NoError(t, err)

		// GitOps path: submit an empty profile set for no-team, which deletes
		// the remaining profile.
		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{}, nil)
			return err
		})
		require.NoError(t, err)

		// Both hosts should now have a queued <Delete>: h1 as the direct
		// consequence of the deletion, h2 because phase 2 upgrades the
		// already remove+NULL row to a concrete command. This also proves
		// offline moved hosts still receive the removal on next check-in,
		// and that the destination-team profile with the same LocURI did
		// not spuriously protect ./Device/Z from the no-team deletion.
		for _, h := range []*fleet.Host{h1, h2} {
			rawCmd := rawWindowsDeleteCommandForHostProfile(t, ds, h.UUID, profUUID)
			require.NotEmpty(t, rawCmd, "host %s should have a queued <Delete>", h.Hostname)
			assert.Contains(t, string(rawCmd), "./Device/Z")
		}
	})
}

func testEditProfileDeletesRemovedLocURIs(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	h1 := test.NewHost(t, ds, "host-edit-1", "10.0.0.3", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform("windows"))
	windowsEnroll(t, ds, h1)

	t.Run("removed LocURI generates delete", func(t *testing.T) {
		t.Cleanup(func() {
			TruncateTables(t, ds, "host_mdm_windows_profiles", "windows_mdm_command_queue", "windows_mdm_commands", "mdm_windows_configuration_profiles")
		})

		// Profile with two LocURIs.
		prof := &fleet.MDMWindowsConfigProfile{Name: "edit-test", SyncML: []byte(`<Atomic><Replace><Item><Target><LocURI>./Device/Keep</LocURI></Target><Data>1</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Remove</LocURI></Target><Data>1</Data></Item></Replace></Atomic>`)}

		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{prof}, nil)
			return err
		})
		require.NoError(t, err)

		var profUUID string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &profUUID, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = 'edit-test'`)
		})

		// Simulate profile installed on the host.
		verified := fleet.MDMDeliveryVerified
		err = ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{ProfileUUID: profUUID, ProfileName: "edit-test", HostUUID: h1.UUID, CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Status: &verified, Checksum: []byte{0}},
		})
		require.NoError(t, err)

		// Edit profile: remove ./Device/Remove.
		profEdited := &fleet.MDMWindowsConfigProfile{Name: "edit-test", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Keep</LocURI></Target><Data>1</Data></Item></Replace>`)}

		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profEdited}, nil)
			return err
		})
		require.NoError(t, err)

		// A delete command should have been generated for ./Device/Remove.
		var deleteCommands [][]byte
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &deleteCommands,
				`SELECT wc.raw_command FROM windows_mdm_commands wc
				JOIN windows_mdm_command_queue cq ON cq.command_uuid = wc.command_uuid
				JOIN mdm_windows_enrollments mwe ON mwe.id = cq.enrollment_id
				WHERE mwe.host_uuid = ?
				ORDER BY wc.created_at DESC`, h1.UUID)
		})

		foundDelete := false
		for _, cmd := range deleteCommands {
			s := string(cmd)
			if strings.Contains(s, "<Delete") && strings.Contains(s, "./Device/Remove") {
				foundDelete = true
				assert.NotContains(t, s, "./Device/Keep", "should not delete the kept LocURI")
			}
		}
		assert.True(t, foundDelete, "expected a <Delete> command for ./Device/Remove")
	})

	t.Run("shared LocURI not deleted when editing", func(t *testing.T) {
		t.Cleanup(func() {
			TruncateTables(t, ds, "host_mdm_windows_profiles", "windows_mdm_command_queue", "windows_mdm_commands", "mdm_windows_configuration_profiles")
		})

		// Profile A has LocURIs P, Q. Profile B has LocURI Q (shared).
		profA := &fleet.MDMWindowsConfigProfile{Name: "edit-shared-A", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/P</LocURI></Target><Data>1</Data></Item></Replace><Replace><Item><Target><LocURI>./Device/Q</LocURI></Target><Data>1</Data></Item></Replace>`)}
		profBShared := &fleet.MDMWindowsConfigProfile{Name: "edit-shared-B", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/Q</LocURI></Target><Data>2</Data></Item></Replace>`)}

		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profA, profBShared}, nil)
			return err
		})
		require.NoError(t, err)

		var profAUUID string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &profAUUID, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = 'edit-shared-A'`)
		})

		// Simulate profile A installed on the host.
		verified := fleet.MDMDeliveryVerified
		err = ds.BulkUpsertMDMWindowsHostProfiles(ctx, []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
			{ProfileUUID: profAUUID, ProfileName: "edit-shared-A", HostUUID: h1.UUID, CommandUUID: uuid.NewString(), OperationType: fleet.MDMOperationTypeInstall, Status: &verified, Checksum: []byte{0}},
		})
		require.NoError(t, err)

		// Edit A to remove Q (shared with B), keep only P.
		profAEdited := &fleet.MDMWindowsConfigProfile{Name: "edit-shared-A", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/P</LocURI></Target><Data>1</Data></Item></Replace>`)}

		err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profAEdited, profBShared}, nil)
			return err
		})
		require.NoError(t, err)

		// Check that no delete was generated for Q (protected by B),
		// and no delete was generated for P (still in the edited profile).
		var deleteCommands [][]byte
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &deleteCommands,
				`SELECT wc.raw_command FROM windows_mdm_commands wc
				JOIN windows_mdm_command_queue cq ON cq.command_uuid = wc.command_uuid
				JOIN mdm_windows_enrollments mwe ON mwe.id = cq.enrollment_id
				WHERE mwe.host_uuid = ?
				ORDER BY wc.created_at DESC`, h1.UUID)
		})

		for _, cmd := range deleteCommands {
			s := string(cmd)
			if strings.Contains(s, "<Delete") {
				assert.NotContains(t, s, "./Device/Q", "should NOT delete Q (protected by edit-shared-B)")
				assert.NotContains(t, s, "./Device/P", "should NOT delete P (still in edited profile)")
			}
		}
	})
}

// testBatchDeleteMultipleWindowsProfiles exercises the multi-profile delete path in
// cancelWindowsHostInstallsForDeletedMDMProfiles via a single batchSetMDMWindowsProfilesDB
// call that removes several profiles at once. This hits the CASE-mapped multi-profile
// UPDATE on host_mdm_windows_profiles (one statement, many profiles, one command_uuid
// per profile) and verifies that each profile's rows receive the correct, distinct
// command_uuid.
func testBatchDeleteMultipleWindowsProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	h1 := test.NewHost(t, ds, "host-multi-del-1", "10.0.0.10", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform("windows"))
	windowsEnroll(t, ds, h1)
	h2 := test.NewHost(t, ds, "host-multi-del-2", "10.0.0.11", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform("windows"))
	windowsEnroll(t, ds, h2)

	profA := &fleet.MDMWindowsConfigProfile{Name: "multi-del-A", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/A</LocURI></Target><Data>1</Data></Item></Replace>`)}
	profB := &fleet.MDMWindowsConfigProfile{Name: "multi-del-B", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/B</LocURI></Target><Data>1</Data></Item></Replace>`)}
	profC := &fleet.MDMWindowsConfigProfile{Name: "multi-del-C", SyncML: []byte(`<Replace><Item><Target><LocURI>./Device/C</LocURI></Target><Data>1</Data></Item></Replace>`)}

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{profA, profB, profC}, nil)
		return err
	})
	require.NoError(t, err)

	profAUUID := windowsProfileUUIDByName(t, ds, "multi-del-A")
	profBUUID := windowsProfileUUIDByName(t, ds, "multi-del-B")
	profCUUID := windowsProfileUUIDByName(t, ds, "multi-del-C")

	// Mark all three profiles as verified-installed on both hosts, so phase-2
	// cleanup has to generate <Delete> commands and flip the rows to remove+pending.
	installWindowsProfilesAsVerified(t, ds,
		[]string{h1.UUID, h2.UUID},
		[]string{profAUUID, profBUUID, profCUUID})

	// Delete ALL THREE profiles in a single batch-set call by passing an empty new set.
	// This drives cancelWindowsHostInstallsForDeletedMDMProfiles with >1 profile, which
	// is precisely the code path the new multi-profile batched UPDATE targets.
	err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, nil, []*fleet.MDMWindowsConfigProfile{}, nil)
		return err
	})
	require.NoError(t, err)

	// 2 hosts × 3 profiles = 6 rows, each flipped to remove+pending with a
	// non-empty command_uuid and empty detail.
	type row struct {
		OperationType string `db:"operation_type"`
		Status        string `db:"status"`
		Detail        string `db:"detail"`
		CommandUUID   string `db:"command_uuid"`
	}
	var rows []row
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &rows,
			`SELECT operation_type, status, detail, command_uuid
			 FROM host_mdm_windows_profiles
			 WHERE host_uuid IN (?, ?)
			 ORDER BY host_uuid, profile_uuid`, h1.UUID, h2.UUID)
	})
	require.Len(t, rows, 6)
	for _, r := range rows {
		assert.Equal(t, string(fleet.MDMOperationTypeRemove), r.OperationType)
		assert.Equal(t, string(fleet.MDMDeliveryPending), r.Status)
		assert.Empty(t, r.Detail)
		assert.NotEmpty(t, r.CommandUUID)
	}

	// For every (host, profile), follow the command_uuid on the flipped row
	// through to windows_mdm_commands and assert the raw SyncML targets THAT
	// profile's specific LocURI. This catches CASE mis-assignment (profA's
	// command_uuid cross-wired to profB's Delete command) and an unintended
	// ELSE-clause fire (command_uuid unchanged, not in windows_mdm_commands).
	profLocURIs := map[string]string{
		profAUUID: "./Device/A",
		profBUUID: "./Device/B",
		profCUUID: "./Device/C",
	}
	for _, hUUID := range []string{h1.UUID, h2.UUID} {
		for profUUID, wantLocURI := range profLocURIs {
			raw := rawWindowsDeleteCommandForHostProfile(t, ds, hUUID, profUUID)
			require.NotEmpty(t, raw, "host %s profile %s: no Delete command queued", hUUID, profUUID)
			assert.Contains(t, string(raw), wantLocURI,
				"host %s profile %s: command should target %s, not another profile's LocURI",
				hUUID, profUUID, wantLocURI)
		}
	}
}

// testMDMWindowsUnenrollCleansUpProfiles verifies that pending profile rows are cleaned up when a
// Windows host unenrolls from MDM (issue #42427, scenario B).
func testMDMWindowsUnenrollCleansUpProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// create a Windows host and enroll it
	host := test.NewHost(t, ds, "win-unenroll", "10.0.0.1", "win-key", "win-unenroll-uuid", time.Now())
	host.Platform = "windows"
	require.NoError(t, ds.UpdateHost(ctx, host))
	deviceID := windowsEnroll(t, ds, host)

	// create a Windows profile and set it as pending install on the host
	profUUID := InsertWindowsProfileForTest(t, ds, 0)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_windows_profiles
				(host_uuid, status, operation_type, command_uuid, profile_name, checksum, profile_uuid)
			VALUES (?, ?, ?, ?, ?, UNHEX(MD5('test')), ?)`,
			host.UUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, uuid.NewString(), "TestProfile", profUUID)
		return err
	})

	// verify the profile row exists
	winProfs, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, winProfs, 1)
	require.Equal(t, profUUID, winProfs[0].ProfileUUID)

	// unenroll the device
	err = ds.MDMWindowsDeleteEnrolledDeviceWithDeviceID(ctx, deviceID)
	require.NoError(t, err)

	// verify the profile row has been cleaned up
	winProfs, err = ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, winProfs)
}

// testMDMWindowsProfilesToRemoveSkipsOrphanedHosts verifies that
// ListMDMWindowsProfilesToRemoveForHosts does not return profiles for hosts
// whose mdm_windows_enrollments row has been deleted (issue #44369).
func testMDMWindowsProfilesToRemoveSkipsOrphanedHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create two Windows hosts and enroll both.
	host1 := test.NewHost(t, ds, "win-orphan1", "10.0.0.1", "k1", "uuid-orphan1", time.Now())
	host1.Platform = "windows"
	require.NoError(t, ds.UpdateHost(ctx, host1))
	windowsEnroll(t, ds, host1)

	host2 := test.NewHost(t, ds, "win-orphan2", "10.0.0.2", "k2", "uuid-orphan2", time.Now())
	host2.Platform = "windows"
	require.NoError(t, ds.UpdateHost(ctx, host2))
	windowsEnroll(t, ds, host2)

	// Create a global profile and mark it as installed+verified on both hosts.
	profUUID := InsertWindowsProfileForTest(t, ds, 0)
	installWindowsProfilesAsVerified(t, ds, []string{host1.UUID, host2.UUID}, []string{profUUID})

	// Delete the profile definition via raw SQL so that host_mdm_windows_profiles
	// rows become orphaned removal candidates (bypassing the cascade cleanup in
	// DeleteMDMWindowsConfigProfile).
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, profUUID)
		return err
	})

	// Sanity check: both hosts should be removal candidates while enrolled.
	toRemove, err := ds.ListMDMWindowsProfilesToRemoveForHosts(ctx, []string{host1.UUID, host2.UUID})
	require.NoError(t, err)
	require.Len(t, toRemove, 2)

	// Now delete host1's enrollment to simulate the orphan condition.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM mdm_windows_enrollments WHERE host_uuid = ?`, host1.UUID)
		return err
	})

	// Only host2 (still enrolled) should be returned; host1 is orphaned.
	toRemove, err = ds.ListMDMWindowsProfilesToRemoveForHosts(ctx, []string{host1.UUID, host2.UUID})
	require.NoError(t, err)
	require.Len(t, toRemove, 1)
	require.Equal(t, host2.UUID, toRemove[0].HostUUID)
}

// testMDMWindowsInsertCommandSkipsUnenrolledHosts verifies that
// MDMWindowsInsertCommandAndUpsertHostProfilesForHosts gracefully skips hosts
// without an enrollment instead of failing the batch (issue #44369).
func testMDMWindowsInsertCommandSkipsUnenrolledHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create and enroll two devices.
	d1 := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-2C3ARC2",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               uuid.NewString(),
	}
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, d1))
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, d2))

	d1EnrollID := mdmWindowsEnrollmentIDByHardwareID(ctx, t, ds, d1.MDMHardwareID)

	// Delete d2's enrollment to simulate an orphan.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM mdm_windows_enrollments WHERE host_uuid = ?`, d2.HostUUID)
		return err
	})

	profUUID := InsertWindowsProfileForTest(t, ds, 0)
	cmdUUID := uuid.NewString()
	cmd := &fleet.MDMWindowsCommand{
		CommandUUID:  cmdUUID,
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri",
	}
	payload := []*fleet.MDMWindowsBulkUpsertHostProfilePayload{
		{
			ProfileUUID:   profUUID,
			ProfileName:   "prof1",
			HostUUID:      d1.HostUUID,
			CommandUUID:   cmdUUID,
			OperationType: fleet.MDMOperationTypeInstall,
			Status:        &fleet.MDMDeliveryPending,
			Checksum:      []byte("checksum1"),
		},
		{
			ProfileUUID:   profUUID,
			ProfileName:   "prof1",
			HostUUID:      d2.HostUUID,
			CommandUUID:   cmdUUID,
			OperationType: fleet.MDMOperationTypeInstall,
			Status:        &fleet.MDMDeliveryPending,
			Checksum:      []byte("checksum2"),
		},
	}

	// Should succeed — d2 is skipped, d1 is processed.
	err := ds.MDMWindowsInsertCommandAndUpsertHostProfilesForHosts(ctx, []string{d1.HostUUID, d2.HostUUID}, cmd, payload)
	require.NoError(t, err)

	// d1 (enrolled) should have a queued command.
	cmds, err := ds.MDMWindowsGetPendingCommands(ctx, d1EnrollID)
	require.NoError(t, err)
	require.Len(t, cmds, 1)

	// There should be exactly 1 command queue entry (d1 only); d2 was
	// silently skipped by the INSERT ... SELECT because its enrollment
	// no longer exists.
	var queueCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queueCount,
			`SELECT COUNT(*) FROM windows_mdm_command_queue WHERE command_uuid = ?`, cmdUUID)
	})
	require.Equal(t, 1, queueCount)

	// Both hosts get profile rows upserted — d2's profile row is
	// intentionally kept. It's harmless dead data: the EXISTS check in
	// windowsProfilesToRemoveQuery prevents it from being selected for
	// removal on the next cron cycle.
	var profileCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &profileCount,
			`SELECT COUNT(*) FROM host_mdm_windows_profiles WHERE command_uuid = ?`, cmdUUID)
	})
	require.Equal(t, 2, profileCount)
}

func testCleanupWindowsMDMCommandQueue(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	dev := createEnrolledDevice(t, ds)

	// Insert two commands queued for the device.
	cmd1 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte(`<Atomic><CmdID>` + uuid.NewString() + `</CmdID></Atomic>`),
		TargetLocURI: "./Device/Test1",
	}
	cmd2 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte(`<Atomic><CmdID>` + uuid.NewString() + `</CmdID></Atomic>`),
		TargetLocURI: "./Device/Test2",
	}
	cmd3 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte(`<Atomic><CmdID>` + uuid.NewString() + `</CmdID></Atomic>`),
		TargetLocURI: "./Device/Test3",
	}
	err := ds.mdmWindowsInsertCommandForHostsDB(ctx, ds.primary, []string{dev.MDMDeviceID}, cmd1)
	require.NoError(t, err)
	err = ds.mdmWindowsInsertCommandForHostsDB(ctx, ds.primary, []string{dev.MDMDeviceID}, cmd2)
	require.NoError(t, err)
	err = ds.mdmWindowsInsertCommandForHostsDB(ctx, ds.primary, []string{dev.MDMDeviceID}, cmd3)
	require.NoError(t, err)

	// All three should be in the queue.
	var count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE enrollment_id = ?", dev.ID)
	})
	require.Equal(t, 3, count)

	// Insert a response row (required FK for command results).
	var responseID int64
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, '<SyncML/>')`, dev.ID)
		if err != nil {
			return err
		}
		responseID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		return nil
	})

	// Insert a result for cmd1 with a timestamp >1 hour ago (eligible for GC).
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, status_code, response_id, created_at)
			VALUES (?, ?, '<Status/>', '200', ?, NOW() - INTERVAL 2 HOUR)`,
			dev.ID, cmd1.CommandUUID, responseID)
		return err
	})

	// Insert a result for cmd2 with a recent timestamp (not yet eligible for GC).
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, status_code, response_id, created_at)
			VALUES (?, ?, '<Status/>', '200', ?, NOW())`,
			dev.ID, cmd2.CommandUUID, responseID)
		return err
	})

	// Run cleanup.
	err = ds.CleanupWindowsMDMCommandQueue(ctx)
	require.NoError(t, err)

	// cmd1's queue row should be deleted (result is >1 hour old).
	var cmd1Count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &cmd1Count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE enrollment_id = ? AND command_uuid = ?",
			dev.ID, cmd1.CommandUUID)
	})
	assert.Equal(t, 0, cmd1Count, "Queue row for cmd1 should be cleaned up (result >1 hour old)")

	// cmd2's queue row should still exist (result is recent).
	var cmd2Count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &cmd2Count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE enrollment_id = ? AND command_uuid = ?",
			dev.ID, cmd2.CommandUUID)
	})
	assert.Equal(t, 1, cmd2Count, "Queue row for cmd2 should remain (result <1 hour old)")

	// cmd3's queue row should still exist (no result at all — still pending).
	var cmd3Count int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &cmd3Count, "SELECT COUNT(*) FROM windows_mdm_command_queue WHERE enrollment_id = ? AND command_uuid = ?",
			dev.ID, cmd3.CommandUUID)
	})
	assert.Equal(t, 1, cmd3Count, "Queue row for cmd3 should remain (pending, no result)")
}

// testMDMWindowsProfilesSummaryEnumeration exhaustively enumerates every
// possible (status, operation_type, reserved) shape a host_mdm_windows_profiles
// row can take and exercises every 0-, 1-, and 2-profile host configuration
// through GetMDMWindowsProfilesSummary.
//
// The input universe is finite and small: status is FK-constrained to
// {NULL, failed, pending, verifying, verified} (5 values) and operation_type
// is FK-constrained to {NULL, install, remove} (3 values); a profile is either
// reserved or non-reserved (2). Per row that is 30 shapes; for 2-profile hosts
// we enumerate all 30x30 = 900 ordered pairs, plus 30 single-profile cases,
// plus one zero-profile case.
func testMDMWindowsProfilesSummaryEnumeration(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Keep us on the profiles-only summary path (no BitLocker branches).
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
	require.NoError(t, ds.SaveAppConfig(ctx, ac))

	type profileShape struct {
		status        *fleet.MDMDeliveryStatus
		operationType *fleet.MDMOperationType
		reserved      bool
	}

	var shapes []profileShape
	statuses := []*fleet.MDMDeliveryStatus{
		nil,
		&fleet.MDMDeliveryFailed,
		&fleet.MDMDeliveryPending,
		&fleet.MDMDeliveryVerifying,
		&fleet.MDMDeliveryVerified,
	}
	// MDMOperationType constants are untyped until bound to a local var, so we
	// materialize pointers here.
	opInstall := fleet.MDMOperationTypeInstall
	opRemove := fleet.MDMOperationTypeRemove
	opTypes := []*fleet.MDMOperationType{
		nil,
		&opInstall,
		&opRemove,
	}
	for _, s := range statuses {
		for _, o := range opTypes {
			for _, r := range []bool{false, true} {
				shapes = append(shapes, profileShape{status: s, operationType: o, reserved: r})
			}
		}
	}
	require.Len(t, shapes, len(statuses)*len(opTypes)*2)

	// expectedFinalStatus implements the pre-refactor EXISTS/NOT-EXISTS logic
	// literally so we can validate the new aggregation-based query against it.
	expectedFinalStatus := func(profiles []profileShape) string {
		isNonReserved := func(p profileShape) bool { return !p.reserved }
		isNonReservedInstall := func(p profileShape) bool {
			return !p.reserved && p.operationType != nil && *p.operationType == fleet.MDMOperationTypeInstall
		}

		// EXISTS(non-reserved row with status='failed')
		if slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReserved(p) && p.status != nil && *p.status == fleet.MDMDeliveryFailed
		}) {
			return "failed"
		}
		// EXISTS(non-reserved row with status IS NULL OR status='pending')
		if slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReserved(p) && (p.status == nil || *p.status == fleet.MDMDeliveryPending)
		}) {
			return "pending"
		}
		// EXISTS(non-reserved install row with status='verifying')
		// AND NOT EXISTS(non-reserved install row with status NULL or status NOT IN (verifying, verified))
		hasVerifying := slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReservedInstall(p) && p.status != nil && *p.status == fleet.MDMDeliveryVerifying
		})
		verifyingBlocker := slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReservedInstall(p) && (p.status == nil ||
				(*p.status != fleet.MDMDeliveryVerifying && *p.status != fleet.MDMDeliveryVerified))
		})
		if hasVerifying && !verifyingBlocker {
			return "verifying"
		}
		// EXISTS(non-reserved install row with status='verified')
		// AND NOT EXISTS(non-reserved install row with status NULL or status != 'verified')
		hasVerified := slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReservedInstall(p) && p.status != nil && *p.status == fleet.MDMDeliveryVerified
		})
		verifiedBlocker := slices.ContainsFunc(profiles, func(p profileShape) bool {
			return isNonReservedInstall(p) && (p.status == nil || *p.status != fleet.MDMDeliveryVerified)
		})
		if hasVerified && !verifiedBlocker {
			return "verified"
		}
		return ""
	}

	// Build every configuration: 0-, 1-, and 2-profile.
	var cases [][]profileShape
	cases = append(cases, nil)
	for _, a := range shapes {
		cases = append(cases, []profileShape{a})
	}
	for _, a := range shapes {
		for _, b := range shapes {
			cases = append(cases, []profileShape{a, b})
		}
	}
	require.Len(t, cases, 1+len(shapes)+len(shapes)*len(shapes))

	// Map "<bucket>" -> OSSettingsFilter for the per-host membership check.
	bucketFilter := map[string]fleet.OSSettingsStatus{
		"failed":    fleet.OSSettingsFailed,
		"pending":   fleet.OSSettingsPending,
		"verifying": fleet.OSSettingsVerifying,
		"verified":  fleet.OSSettingsVerified,
	}

	// Tally expected bucket counts (for the summary assertion) and expected
	// host membership per bucket (for the filter assertion). The membership
	// check defends against regressions that swap two configurations between
	// buckets while preserving aggregate counts.
	var expected fleet.MDMProfilesSummary
	expectedHostsByBucket := map[fleet.OSSettingsStatus][]uint{}
	expectedBucketByCase := make([]string, len(cases))
	for i, c := range cases {
		bucket := expectedFinalStatus(c)
		expectedBucketByCase[i] = bucket
		switch bucket {
		case "failed":
			expected.Failed++
		case "pending":
			expected.Pending++
		case "verifying":
			expected.Verifying++
		case "verified":
			expected.Verified++
		}
	}

	// Insert one Windows host per case (all on the global "no team" scope so
	// the summary query's `h.team_id IS NULL` branch covers them), then insert
	// its profile rows. Track each host's expected bucket by ID for the
	// per-host membership assertion below.
	const nonReservedName = "enum-test-profile"
	reservedName := mdm.FleetWindowsOSUpdatesProfileName
	now := time.Now()
	for caseIdx, c := range cases {
		hostUUID := fmt.Sprintf("enum-host-%04d", caseIdx)
		h := test.NewHost(t, ds, hostUUID, "1.1.1.1", hostUUID, hostUUID, now, test.WithPlatform("windows"))
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", false))
		windowsEnroll(t, ds, h)

		if filter, ok := bucketFilter[expectedBucketByCase[caseIdx]]; ok {
			expectedHostsByBucket[filter] = append(expectedHostsByBucket[filter], h.ID)
		}

		for i, p := range c {
			profUUID := fmt.Sprintf("%s-p%d", hostUUID, i)
			commandUUID := fmt.Sprintf("cmd-%s", profUUID)
			name := nonReservedName
			if p.reserved {
				name = reservedName
			}
			var statusArg any
			if p.status != nil {
				statusArg = string(*p.status)
			}
			var opTypeArg any
			if p.operationType != nil {
				opTypeArg = string(*p.operationType)
			}
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx,
					`INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, profile_name, status, operation_type, command_uuid) VALUES (?, ?, ?, ?, ?, ?)`,
					hostUUID, profUUID, name, statusArg, opTypeArg, commandUUID)
				return err
			})
		}
	}

	// 1) Aggregate bucket counts via GetMDMWindowsProfilesSummary.
	got, err := ds.GetMDMWindowsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equalf(t, expected, *got,
		"aggregate bucket counts diverged from reference implementation over %d enumerated configurations", len(cases))

	// 2) Per-host membership via ListHosts with OSSettingsFilter. This
	//    catches regressions that preserve aggregate counts but swap two
	//    hosts between buckets, and also exercises filterHostsByOSSettingsStatus
	//    (the host-list path uses the same windowsHostProfileStatusSubquery
	//    helper but a different outer query).
	teamFilter := fleet.TeamFilter{User: test.UserAdmin}
	for _, filter := range []fleet.OSSettingsStatus{
		fleet.OSSettingsFailed,
		fleet.OSSettingsPending,
		fleet.OSSettingsVerifying,
		fleet.OSSettingsVerified,
	} {
		gotHosts, err := ds.ListHosts(ctx, teamFilter, fleet.HostListOptions{OSSettingsFilter: filter})
		require.NoError(t, err)
		gotIDs := make([]uint, 0, len(gotHosts))
		for _, h := range gotHosts {
			gotIDs = append(gotIDs, h.ID)
		}
		require.ElementsMatchf(t, expectedHostsByBucket[filter], gotIDs,
			"per-host membership mismatch for OSSettingsFilter=%s", filter)
	}
}

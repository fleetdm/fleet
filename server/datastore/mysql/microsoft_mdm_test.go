package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func TestGetMDMWindowsBitLockerSummary(t *testing.T) {
	ds := CreateMySQLDS(t)

	ctx := context.Background()

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

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	// TODO: Update test to rely on MDM.EnableDiskEncryption when it is implemented
	require.False(t, ac.MDM.MacOSSettings.EnableDiskEncryption)

	bls, err := ds.GetMDMWindowsBitLockerSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(0), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(0), bls.Failed)
	require.Equal(t, uint(0), bls.Enforcing)
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	ac.MDM.MacOSSettings.EnableDiskEncryption = true
	require.NoError(t, ds.SaveAppConfig(ctx, ac))
	ac, err = ds.AppConfig(ctx)
	require.NoError(t, err)
	require.True(t, ac.MDM.MacOSSettings.EnableDiskEncryption)

	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(0), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(0), bls.Failed)
	require.Equal(t, uint(5), bls.Enforcing)
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	// TODO: Update test to use methods to set windows disk encryption when they are implemented
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_disk_encryption_keys (host_id, decryptable, client_error) VALUES (?, ?, ?)`,
			hosts[0].ID,
			true,
			"")
		return err
	})
	// TODO: Update test to use methods to set windows disk encryption when they are implemented
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_disk_encryption_keys (host_id, decryptable, client_error) VALUES (?, ?, ?)`,
			hosts[1].ID,
			false,
			"test-error")
		return err
	})

	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(1), bls.Verified) // hosts[0]
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(1), bls.Failed) // hosts[1]
	require.Equal(t, uint(3), bls.Enforcing)
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	// Test team filtering
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team"})
	require.NoError(t, err)

	tm, err := ds.Team(ctx, team.ID)
	require.NoError(t, err)
	require.NotNil(t, tm)
	require.False(t, tm.Config.MDM.MacOSSettings.EnableDiskEncryption) // disk encryption is not enabled for team

	// Transfer hosts[2] to the team
	require.NoError(t, ds.AddHostsToTeam(ctx, &team.ID, []uint{hosts[2].ID}))

	// Check the summary for the team
	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(0), bls.Failed)
	require.Equal(t, uint(0), bls.Enforcing) // disk encryption is not enabled for team so hosts[2] is not counted
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	// Check the summary for no team
	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(1), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(1), bls.Failed)
	require.Equal(t, uint(2), bls.Enforcing) // hosts[2] is no longer included in the no team summary
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	// Enable disk encryption for the team
	tm.Config.MDM.MacOSSettings.EnableDiskEncryption = true
	tm, err = ds.SaveTeam(ctx, tm)
	require.NoError(t, err)
	require.NotNil(t, tm)
	require.True(t, tm.Config.MDM.MacOSSettings.EnableDiskEncryption)

	// Check the summary for the team
	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(0), bls.Failed)
	require.Equal(t, uint(1), bls.Enforcing) // hosts[2] is now counted
	require.Equal(t, uint(0), bls.RemovingEnforcement)

	// Check the summary for no team (should be unchanged)
	bls, err = ds.GetMDMWindowsBitLockerSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint(1), bls.Verified)
	require.Equal(t, uint(0), bls.Verifying)
	require.Equal(t, uint(1), bls.Failed)
	require.Equal(t, uint(2), bls.Enforcing)
	require.Equal(t, uint(0), bls.RemovingEnforcement)
}

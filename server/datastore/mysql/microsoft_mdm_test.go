package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMDMWindows(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestMDMWindowsEnrolledDevices", testMDMWindowsEnrolledDevice},
		{"TestMDMWindowsPendingCommand", testMDMWindowsPendingCommand},
		{"TestMDMWindowsCommand", testMDMWindowCommand},
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

func testMDMWindowsPendingCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	deviceID := uuid.New().String()

	pendingCmd := &fleet.MDMWindowsPendingCommand{
		CommandUUID:  uuid.New().String(),
		DeviceID:     deviceID,
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri",
		SettingValue: "testdata",
		DataType:     2,
		SystemOrigin: false,
	}

	err := ds.MDMWindowsInsertPendingCommand(ctx, pendingCmd)
	require.NoError(t, err)

	var ae fleet.AlreadyExistsError
	err = ds.MDMWindowsInsertPendingCommand(ctx, pendingCmd)
	require.ErrorAs(t, err, &ae)

	gotPendingCmds, err := ds.MDMWindowsListPendingCommands(ctx, deviceID)
	require.NoError(t, err)
	require.NotZero(t, gotPendingCmds)
	require.NotZero(t, gotPendingCmds[0].CreatedAt)
	require.Equal(t, gotPendingCmds[0].DeviceID, deviceID)
}

func testMDMWindowCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	deviceID := uuid.New().String()
	sessionID := "1"
	messageID := "2"
	commandID := "3"
	newCmd := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.New().String(),
		DeviceID:     deviceID,
		SessionID:    sessionID,
		MessageID:    messageID,
		CommandID:    commandID,
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri",
		SettingValue: "testdata",
		DataType:     2,
		SystemOrigin: false,
	}

	err := ds.MDMWindowsInsertCommand(ctx, newCmd)
	require.NoError(t, err)

	var ae fleet.AlreadyExistsError
	err = ds.MDMWindowsInsertCommand(ctx, newCmd)
	require.ErrorAs(t, err, &ae)

	gotCmds, err := ds.MDMWindowsListCommands(ctx, deviceID)
	require.NoError(t, err)
	require.NotZero(t, gotCmds)
	require.NotZero(t, gotCmds[0].CreatedAt)
	require.Equal(t, gotCmds[0].DeviceID, deviceID)
	require.Equal(t, gotCmds[0].SessionID, sessionID)
	require.Equal(t, gotCmds[0].MessageID, messageID)
	require.Equal(t, gotCmds[0].CommandID, commandID)
}

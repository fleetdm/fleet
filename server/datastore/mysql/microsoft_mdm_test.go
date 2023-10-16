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

	// Inserting two pending commands
	notTrackedDeviceID := uuid.New().String()
	trackedDeviceID := uuid.New().String()
	trackedCommandUUID := uuid.New().String()
	pendingCmd1 := &fleet.MDMWindowsPendingCommand{
		CommandUUID:  trackedCommandUUID,
		DeviceID:     trackedDeviceID,
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri1",
		SettingValue: "testdata1",
		DataType:     2,
		SystemOrigin: false,
	}

	pendingCmd2 := &fleet.MDMWindowsPendingCommand{
		CommandUUID:  uuid.New().String(),
		DeviceID:     notTrackedDeviceID,
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri",
		SettingValue: "testdata",
		DataType:     2,
		SystemOrigin: false,
	}

	err := ds.MDMWindowsInsertPendingCommand(ctx, pendingCmd1)
	require.NoError(t, err)

	err = ds.MDMWindowsInsertPendingCommand(ctx, pendingCmd2)
	require.NoError(t, err)

	// Checking that pending command cannot be inserted if already exists
	var ae fleet.AlreadyExistsError
	err = ds.MDMWindowsInsertPendingCommand(ctx, pendingCmd2)
	require.ErrorAs(t, err, &ae)

	// Now checking if pending command for a given DeviceID can be retrieved
	gotPendingCmds, err := ds.MDMWindowsGetPendingCommands(ctx, trackedDeviceID)
	require.NoError(t, err)
	require.NotZero(t, gotPendingCmds)
	require.NotZero(t, gotPendingCmds[0].CreatedAt)
	require.Equal(t, gotPendingCmds[0].DeviceID, trackedDeviceID)

	// Now inserting commands in the tracking table
	// One of these commands should be for work DeviceID
	newCmd1 := &fleet.MDMWindowsCommand{
		CommandUUID:  trackedCommandUUID,
		DeviceID:     trackedDeviceID,
		SessionID:    "2",
		MessageID:    "3",
		CommandID:    "4",
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri1",
		SettingValue: "testdata1",
		SystemOrigin: false,
	}

	newCmd2 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.New().String(),
		DeviceID:     uuid.New().String(),
		SessionID:    "6",
		MessageID:    "7",
		CommandID:    "8",
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri2",
		SettingValue: "testdata2",
		SystemOrigin: false,
	}

	err = ds.MDMWindowsInsertCommand(ctx, newCmd1)
	require.NoError(t, err)

	err = ds.MDMWindowsInsertCommand(ctx, newCmd2)
	require.NoError(t, err)

	gotCmds, err := ds.MDMWindowsListCommands(ctx, trackedDeviceID)
	require.NoError(t, err)
	require.NotZero(t, gotCmds)

	// Checking if pendings table returns nothing for the command already being tracked
	gotPendingCmds, err = ds.MDMWindowsGetPendingCommands(ctx, trackedDeviceID)
	require.NoError(t, err)
	require.Zero(t, gotPendingCmds)

	// Checking if pendings table returns an entry for device not yet tracked
	gotPendingCmds, err = ds.MDMWindowsGetPendingCommands(ctx, notTrackedDeviceID)
	require.NoError(t, err)
	require.Equal(t, len(gotPendingCmds), 1)
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

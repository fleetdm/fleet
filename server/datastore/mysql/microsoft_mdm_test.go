package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

	// Test using device ID instead of hardware ID
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.NoError(t, err)

	err = ds.MDMWindowsInsertEnrolledDevice(ctx, enrolledDevice)
	require.ErrorAs(t, err, &ae)

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

	// insert a command for multiple devices
	pendingCmd3 := &fleet.MDMWindowsPendingCommand{
		CommandUUID:  uuid.New().String(),
		CmdVerb:      fleet.CmdGet,
		SettingURI:   "./test/uri3",
		SettingValue: "testdata3",
		DataType:     3,
		SystemOrigin: false,
	}
	err = ds.MDMWindowsInsertPendingCommandForDevices(ctx, []string{trackedDeviceID, notTrackedDeviceID}, pendingCmd3)
	require.NoError(t, err)

	// the command can be retrieved as pending for both devices
	gotPendingCmds, err = ds.MDMWindowsGetPendingCommands(ctx, trackedDeviceID)
	require.NoError(t, err)
	require.Len(t, gotPendingCmds, 1)
	require.Equal(t, pendingCmd3.CommandUUID, gotPendingCmds[0].CommandUUID)

	gotPendingCmds, err = ds.MDMWindowsGetPendingCommands(ctx, notTrackedDeviceID)
	require.NoError(t, err)
	require.Len(t, gotPendingCmds, 2)
	gotCmdUUIDs := make([]string, len(gotPendingCmds))
	for i, cmd := range gotPendingCmds {
		gotCmdUUIDs[i] = cmd.CommandUUID
	}
	require.ElementsMatch(t, []string{pendingCmd2.CommandUUID, pendingCmd3.CommandUUID}, gotCmdUUIDs)
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

	// Checking if command can be updated with error code
	err = ds.MDMWindowsUpdateCommandErrorCode(ctx, deviceID, sessionID, messageID, commandID, mdm.CmdStatusOK)
	require.NoError(t, err)

	gotCmds, err = ds.MDMWindowsListCommands(ctx, deviceID)
	require.NoError(t, err)
	require.NotZero(t, gotCmds)
	require.Equal(t, gotCmds[0].ErrorCode, mdm.CmdStatusOK)

	// Checking if command can be updated with result value
	resultData := "2023-10-18T06:16:24.0000756-07:00"
	err = ds.MDMWindowsUpdateCommandReceivedResult(ctx, deviceID, sessionID, messageID, commandID, resultData)
	require.NoError(t, err)

	gotCmds, err = ds.MDMWindowsListCommands(ctx, deviceID)
	require.NoError(t, err)
	require.NotZero(t, gotCmds)
	require.Equal(t, gotCmds[0].CmdResult, resultData)
}

func TestMDMWindowsCommandResults(t *testing.T) {
	ds := CreateMySQLDS(t)
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

	responseID, err := insertDB(t, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`, enrollmentID, "some-response")
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
	require.Equal(t, rawResult, results[0].Result)
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

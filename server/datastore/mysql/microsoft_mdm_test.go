package mysql

import (
	"context" // nolint:gosec // used only to hash for efficient comparisons
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		{"TestMDMWindowsInsertCommandForHosts", testMDMWindowsInsertCommandForHosts},
		{"TestMDMWindowsGetPendingCommands", testMDMWindowsGetPendingCommands},
		{"TestMDMWindowsCommandResults", testMDMWindowsCommandResults},
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
	AddHostUUIDToWinEnrollmentInTest(t, ds, d1.HostUUID, d1.MDMDeviceID)

	err = ds.MDMWindowsInsertEnrolledDevice(ctx, d2)
	require.NoError(t, err)
	AddHostUUIDToWinEnrollmentInTest(t, ds, d2.HostUUID, d2.MDMDeviceID)

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
	AddHostUUIDToWinEnrollmentInTest(t, ds, d.HostUUID, d.MDMDeviceID)

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

package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/stretchr/testify/require"
)

func TestMDMShared(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestMDMCommands", testMDMCommands},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testMDMCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// no commands or devices enrolled => no results
	cmds, err := ds.ListMDMCommands(ctx, fleet.TeamFilter{}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, cmds)

	// enroll a windows device
	windowsH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "windows-test",
		OsqueryHostID: ptr.String("osquery-windows"),
		NodeKey:       ptr.String("node-key-windows"),
		UUID:          uuid.NewString(),
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnrollment := &fleet.MDMWindowsEnrolledDevice{
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
		HostUUID:               windowsH.UUID,
	}
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, windowsEnrollment)
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, windowsEnrollment.HostUUID, windowsEnrollment.MDMDeviceID)
	require.NoError(t, err)
	windowsEnrollment, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, windowsEnrollment.MDMDeviceID)
	require.NoError(t, err)

	// enroll a macOS device
	macH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test",
		OsqueryHostID: ptr.String("osquery-macos"),
		NodeKey:       ptr.String("node-key-macos"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, macH, false)

	// no commands => no results
	cmds, err = ds.ListMDMCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, cmds)

	// insert a windows command
	winCmd := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri",
	}
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{windowsEnrollment.MDMDeviceID}, winCmd)
	require.NoError(t, err)

	// we get one result
	cmds, err = ds.ListMDMCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	require.Equal(t, winCmd.CommandUUID, cmds[0].CommandUUID)
	require.Equal(t, winCmd.TargetLocURI, cmds[0].RequestType)
	require.Equal(t, "Pending", cmds[0].Status)

	appleCmdUUID := uuid.New().String()
	appleCmd := createRawAppleCmd("ProfileList", appleCmdUUID)
	commander, appleCommanderStorage := createMDMAppleCommanderAndStorage(t, ds)
	err = commander.EnqueueCommand(ctx, []string{macH.UUID}, appleCmd)
	require.NoError(t, err)

	// we get both commands
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			ListOptions: fleet.ListOptions{OrderKey: "hostname"},
		})
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	require.Equal(t, appleCmdUUID, cmds[0].CommandUUID)
	require.Equal(t, "ProfileList", cmds[0].RequestType)
	require.Equal(t, "Pending", cmds[0].Status)
	require.Equal(t, winCmd.CommandUUID, cmds[1].CommandUUID)
	require.Equal(t, winCmd.TargetLocURI, cmds[1].RequestType)
	require.Equal(t, "Pending", cmds[1].Status)

	// store results for both commands
	err = appleCommanderStorage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: macH.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: appleCmdUUID,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Raw:         []byte(appleCmd),
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`, windowsEnrollment.ID, "")
		if err != nil {
			return err
		}
		resID, _ := res.LastInsertId()
		_, err = tx.ExecContext(ctx, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, status_code, response_id) VALUES (?, ?, ?, ?, ?)`, windowsEnrollment.ID, winCmd.CommandUUID, "", "200", resID)
		return err
	})

	// we get both commands
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			ListOptions: fleet.ListOptions{OrderKey: "hostname"},
		})
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	require.Equal(t, appleCmdUUID, cmds[0].CommandUUID)
	require.Equal(t, "ProfileList", cmds[0].RequestType)
	require.Equal(t, "Acknowledged", cmds[0].Status)
	require.Equal(t, winCmd.CommandUUID, cmds[1].CommandUUID)
	require.Equal(t, winCmd.TargetLocURI, cmds[1].RequestType)
	require.Equal(t, "200", cmds[1].Status)
}

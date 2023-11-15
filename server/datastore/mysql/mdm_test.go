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
		{"TestBatchSetMDMProfiles", testBatchSetMDMProfiles},
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

func testBatchSetMDMProfiles(t *testing.T, ds *Datastore) {
	applyAndExpect := func(
		newAppleSet []*fleet.MDMAppleConfigProfile,
		newWindowsSet []*fleet.MDMWindowsConfigProfile,
		tmID *uint,
		wantApple []*fleet.MDMAppleConfigProfile,
		wantWindows []*fleet.MDMWindowsConfigProfile,
	) {
		ctx := context.Background()
		err := ds.BatchSetMDMProfiles(ctx, tmID, newAppleSet, newWindowsSet)
		require.NoError(t, err)
		expectAppleProfiles(t, ds, newAppleSet, tmID, wantApple)
		expectWindowsProfiles(t, ds, newWindowsSet, tmID, wantWindows)
	}

	withTeamIDApple := func(p *fleet.MDMAppleConfigProfile, tmID uint) *fleet.MDMAppleConfigProfile {
		p.TeamID = &tmID
		return p
	}

	withTeamIDWindows := func(p *fleet.MDMWindowsConfigProfile, tmID uint) *fleet.MDMWindowsConfigProfile {
		p.TeamID = &tmID
		return p
	}

	// empty set for no team (both Apple and Windows)
	applyAndExpect(nil, nil, nil, nil, nil)

	// single Apple and Windows profile set for a specific team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
		ptr.Uint(1),
		[]*fleet.MDMAppleConfigProfile{withTeamIDApple(configProfileForTest(t, "N1", "I1", "a"), 1)},
		[]*fleet.MDMWindowsConfigProfile{withTeamIDWindows(windowsConfigProfileForTest(t, "W1", "l1"), 1)},
	)

	// single Apple and Windows profile set for no team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
		nil,
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
	)

	// new Apple and Windows profile sets for a specific team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "N1", "I1", "a"), // unchanged
			configProfileForTest(t, "N2", "I2", "b"),
		},
		[]*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "W1", "l1"), // unchanged
			windowsConfigProfileForTest(t, "W2", "l2"),
		},
		ptr.Uint(1),
		[]*fleet.MDMAppleConfigProfile{
			withTeamIDApple(configProfileForTest(t, "N1", "I1", "a"), 1),
			withTeamIDApple(configProfileForTest(t, "N2", "I2", "b"), 1),
		},
		[]*fleet.MDMWindowsConfigProfile{
			withTeamIDWindows(windowsConfigProfileForTest(t, "W1", "l1"), 1),
			withTeamIDWindows(windowsConfigProfileForTest(t, "W2", "l2"), 1),
		},
	)

	// edited profiles, unchanged profiles, and new profiles for a specific team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "N1", "I1", "a-updated"), // content updated
			configProfileForTest(t, "N2", "I2", "b"),         // unchanged
			configProfileForTest(t, "N3", "I3", "c"),         // new
		},
		[]*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "W1", "l1-updated"), // content updated
			windowsConfigProfileForTest(t, "W2", "l2"),         // unchanged
			windowsConfigProfileForTest(t, "W3", "l3"),         // new
		},
		ptr.Uint(1),
		[]*fleet.MDMAppleConfigProfile{
			withTeamIDApple(configProfileForTest(t, "N1", "I1", "a-updated"), 1),
			withTeamIDApple(configProfileForTest(t, "N2", "I2", "b"), 1),
			withTeamIDApple(configProfileForTest(t, "N3", "I3", "c"), 1),
		},
		[]*fleet.MDMWindowsConfigProfile{
			withTeamIDWindows(windowsConfigProfileForTest(t, "W1", "l1-updated"), 1),
			withTeamIDWindows(windowsConfigProfileForTest(t, "W2", "l2"), 1),
			withTeamIDWindows(windowsConfigProfileForTest(t, "W3", "l3"), 1),
		},
	)

	// new Apple and Windows profiles to no team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "N4", "I4", "d"),
			configProfileForTest(t, "N5", "I5", "e"),
		},
		[]*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "W4", "l4"),
			windowsConfigProfileForTest(t, "W5", "l5"),
		},
		nil,
		[]*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "N4", "I4", "d"),
			configProfileForTest(t, "N5", "I5", "e"),
		},
		[]*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "W4", "l4"),
			windowsConfigProfileForTest(t, "W5", "l5"),
		},
	)

	// Test Case 8: Clear profiles for a specific team
	applyAndExpect(nil, nil, ptr.Uint(1), nil, nil)
}

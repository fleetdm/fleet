package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
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
		{"TestListMDMConfigProfiles", testListMDMConfigProfiles},
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

func testListMDMConfigProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	opts := fleet.ListOptions{OrderKey: "name", IncludeMetadata: true}
	winProf := []byte("<Replace></Replace>")

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)

	// both profile tables are empty
	profs, meta, err := ds.ListMDMConfigProfiles(ctx, nil, opts)
	require.NoError(t, err)
	require.Len(t, profs, 0)
	require.Equal(t, *meta, fleet.PaginationMetadata{})

	// add fleet-managed profiles for the team and globally
	for idf := range mobileconfig.FleetPayloadIdentifiers() {
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, team.ID))
		require.NoError(t, err)
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, 0))
		require.NoError(t, err)
	}

	// still returns no result
	profs, meta, err = ds.ListMDMConfigProfiles(ctx, nil, opts)
	require.NoError(t, err)
	require.Len(t, profs, 0)
	require.Equal(t, *meta, fleet.PaginationMetadata{})

	profs, meta, err = ds.ListMDMConfigProfiles(ctx, &team.ID, opts)
	require.NoError(t, err)
	require.Len(t, profs, 0)
	require.Equal(t, *meta, fleet.PaginationMetadata{})

	// create a mac profile for global and a Windows profile for team
	profA, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("A", "A", 0))
	require.NoError(t, err)
	profB, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "B", TeamID: &team.ID, SyncML: winProf})
	require.NoError(t, err)

	// get global profiles returns the mac one
	profs, meta, err = ds.ListMDMConfigProfiles(ctx, nil, opts)
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, profA.Name, profs[0].Name)
	require.Equal(t, *meta, fleet.PaginationMetadata{})

	// get team profiles returns the Windows one
	profs, meta, err = ds.ListMDMConfigProfiles(ctx, &team.ID, opts)
	require.NoError(t, err)
	require.Len(t, profs, 1)
	require.Equal(t, profB.Name, profs[0].Name)
	require.Equal(t, *meta, fleet.PaginationMetadata{})

	// create more profiles and test the pagination with a table-driven test so that
	// global and team both have 9 profiles (including A and B already created above).
	for i := 0; i < 3; i++ {
		inc := i * 4 // e.g. C, D, E, F on first loop, G, H, I, J on second loop, etc.

		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP(string(rune('C'+inc)), string(rune('C'+inc)), 0))
		require.NoError(t, err)
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP(string(rune('C'+inc+1)), string(rune('C'+inc+1)), team.ID))
		require.NoError(t, err)

		_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: string(rune('C' + inc + 2)), TeamID: nil, SyncML: winProf})
		require.NoError(t, err)
		_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: string(rune('C' + inc + 3)), TeamID: &team.ID, SyncML: winProf})
		require.NoError(t, err)
	}

	cases := []struct {
		desc      string
		tmID      *uint
		opts      fleet.ListOptions
		wantNames []string
		wantMeta  fleet.PaginationMetadata
	}{
		{"all global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true}, []string{"A", "C", "E", "G", "I", "K", "M"}, fleet.PaginationMetadata{}},
		{"all team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true}, []string{"B", "D", "F", "H", "J", "L", "N"}, fleet.PaginationMetadata{}},

		{"page 0 per page 2, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2}, []string{"A", "C"}, fleet.PaginationMetadata{HasNextResults: true}},
		{"page 1 per page 2, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 1}, []string{"E", "G"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 2 per page 2, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 2}, []string{"I", "K"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 3 per page 2, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 3}, []string{"M"}, fleet.PaginationMetadata{HasPreviousResults: true}},
		{"page 4 per page 2, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 4}, []string{}, fleet.PaginationMetadata{HasPreviousResults: true}},

		{"page 0 per page 2, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2}, []string{"B", "D"}, fleet.PaginationMetadata{HasNextResults: true}},
		{"page 1 per page 2, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 1}, []string{"F", "H"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 2 per page 2, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 2}, []string{"J", "L"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 3 per page 2, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 3}, []string{"N"}, fleet.PaginationMetadata{HasPreviousResults: true}},
		{"page 4 per page 2, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 4}, []string{}, fleet.PaginationMetadata{HasPreviousResults: true}},

		{"page 0 per page 3, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3}, []string{"A", "C", "E"}, fleet.PaginationMetadata{HasNextResults: true}},
		{"page 1 per page 3, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 1}, []string{"G", "I", "K"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 2 per page 3, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 2}, []string{"M"}, fleet.PaginationMetadata{HasPreviousResults: true}},
		{"page 3 per page 3, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 3}, []string{}, fleet.PaginationMetadata{HasPreviousResults: true}},

		{"page 0 per page 3, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3}, []string{"B", "D", "F"}, fleet.PaginationMetadata{HasNextResults: true}},
		{"page 1 per page 3, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 1}, []string{"H", "J", "L"}, fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true}},
		{"page 2 per page 3, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 2}, []string{"N"}, fleet.PaginationMetadata{HasPreviousResults: true}},
		{"page 3 per page 3, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 3}, []string{}, fleet.PaginationMetadata{HasPreviousResults: true}},

		{"no metadata, global", nil, fleet.ListOptions{OrderKey: "name", IncludeMetadata: false, PerPage: 2, Page: 1}, []string{"E", "G"}, fleet.PaginationMetadata{}},
		{"no metadata, team", &team.ID, fleet.ListOptions{OrderKey: "name", IncludeMetadata: false, PerPage: 2, Page: 1}, []string{"F", "H"}, fleet.PaginationMetadata{}},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			profs, meta, err := ds.ListMDMConfigProfiles(ctx, c.tmID, c.opts)
			require.NoError(t, err)
			require.Len(t, profs, len(c.wantNames))

			got := make([]string, len(profs))
			for i, p := range profs {
				got[i] = p.Name
			}
			require.Equal(t, got, c.wantNames)

			var gotMeta fleet.PaginationMetadata
			if meta != nil {
				gotMeta = *meta
			}
			require.Equal(t, c.wantMeta, gotMeta)
		})
	}
}

package mysql

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
		{"TestBulkSetPendingMDMHostProfiles", testBulkSetPendingMDMHostProfiles},
		{"TestBulkSetPendingMDMHostProfilesBatch2", testBulkSetPendingMDMHostProfilesBatch2},
		{"TestBulkSetPendingMDMHostProfilesBatch3", testBulkSetPendingMDMHostProfilesBatch3},
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

func testBulkSetPendingMDMHostProfilesBatch2(t *testing.T, ds *Datastore) {
	testUpsertMDMDesiredProfilesBatchSize = 2
	testDeleteMDMProfilesBatchSize = 2
	t.Cleanup(func() {
		testUpsertMDMDesiredProfilesBatchSize = 0
		testDeleteMDMProfilesBatchSize = 0
	})
	testBulkSetPendingMDMHostProfiles(t, ds)
}

func testBulkSetPendingMDMHostProfilesBatch3(t *testing.T, ds *Datastore) {
	testUpsertMDMDesiredProfilesBatchSize = 3
	testDeleteMDMProfilesBatchSize = 3
	t.Cleanup(func() {
		testUpsertMDMDesiredProfilesBatchSize = 0
		testDeleteMDMProfilesBatchSize = 0
	})
	testBulkSetPendingMDMHostProfiles(t, ds)
}

func testBulkSetPendingMDMHostProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hostIDsFromHosts := func(hosts ...*fleet.Host) []uint {
		ids := make([]uint, len(hosts))
		for i, h := range hosts {
			ids[i] = h.ID
		}
		return ids
	}

	type anyProfile struct {
		ProfileID        string                   `db:"profile_uuid"`
		Status           *fleet.MDMDeliveryStatus `db:"status"`
		OperationType    fleet.MDMOperationType   `db:"operation_type"`
		IdentifierOrName string                   `db:"profile_name"`
	}

	// only asserts the profile ID, status and operation
	assertHostProfiles := func(want map[*fleet.Host][]anyProfile) {
		for h, wantProfs := range want {
			var gotProfs []anyProfile

			switch h.Platform {
			case "windows":
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					return sqlx.SelectContext(
						ctx, q, &gotProfs,
						`SELECT
						   profile_uuid,
						   COALESCE(status, 'pending') as status,
						   COALESCE(operation_type, '') as operation_type,
						   profile_name
					  	 FROM host_mdm_windows_profiles
					  	 WHERE host_uuid = ?`, h.UUID)
				})
				require.Equal(t, len(wantProfs), len(gotProfs), "host uuid: %s", h.UUID)
			default:
				profs, err := ds.GetHostMDMProfiles(ctx, h.UUID)
				require.NoError(t, err)
				require.Equal(t, len(wantProfs), len(profs), "host uuid: %s", h.UUID)
				for _, p := range profs {
					gotProfs = append(gotProfs, anyProfile{
						ProfileID:        fmt.Sprint(p.ProfileID),
						Status:           p.Status,
						OperationType:    p.OperationType,
						IdentifierOrName: p.Identifier,
					})
				}
			}

			sortProfs := func(profs []anyProfile) []anyProfile {
				sort.Slice(profs, func(i, j int) bool {
					l, r := profs[i], profs[j]
					if l.ProfileID == r.ProfileID {
						return l.OperationType < r.OperationType
					}

					if len(l.ProfileID) <= 2 && len(r.ProfileID) <= 2 {
						a, aErr := strconv.Atoi(l.ProfileID)
						b, bErr := strconv.Atoi(r.ProfileID)

						if aErr == nil && bErr == nil {
							// both are numeric, compare as numbers
							return a < b
						}
					}

					// default alphabetical comparison
					return l.ProfileID < r.ProfileID
				})
				return profs
			}
			gotProfs = sortProfs(gotProfs)
			wantProfs = sortProfs(wantProfs)
			for i, wp := range wantProfs {
				gp := gotProfs[i]
				require.Equal(t, wp.ProfileID, gp.ProfileID, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
				require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
				require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
			}
		}
	}

	getProfs := func(teamID *uint) []*fleet.MDMConfigProfilePayload {
		// TODO(roberto): the docs says that you can pass a comma separated
		// list of columns to OrderKey, but that doesn't seem to work
		profs, _, err := ds.ListMDMConfigProfiles(ctx, teamID, fleet.ListOptions{OrderKey: "platform"})
		require.NoError(t, err)
		sort.Slice(profs, func(i, j int) bool {
			l, r := profs[i], profs[j]

			if l.Platform != r.Platform {
				return l.Platform < r.Platform
			}

			return l.ProfileID < r.ProfileID
		})
		return profs
	}

	// create some darwin hosts, all enrolled
	darwinHosts := make([]*fleet.Host, 3)
	for i := 0; i < 3; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:          fmt.Sprintf("test-uuid-%d", i),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)
		darwinHosts[i] = h
		t.Logf("enrolled darwin host [%d]: %s", i, h.UUID)
	}

	// create a non-enrolled host
	i := 3
	unenrolledHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("test-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("test-uuid-%d", i),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a non-darwin host
	i = 4
	linuxHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("test-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("test-uuid-%d", i),
		Platform:      "linux",
	})
	require.NoError(t, err)

	// create some windows hosts, all enrolled
	i = 5
	windowsHosts := make([]*fleet.Host, 3)
	for j := 0; j < 3; j++ {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("test-host%d-name", i+j),
			OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i+j)),
			NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i+j)),
			UUID:          fmt.Sprintf("test-uuid-%d", i+j),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, h)
		windowsHosts[j] = h
		t.Logf("enrolled windows host [%d]: %s", j, h.UUID)
	}

	// bulk set for no target ids, does nothing
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	// bulk set for combination of target ids, not allowed
	err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{1}, []uint{2}, nil, nil, nil)
	require.Error(t, err)

	// bulk set for all created hosts, no profiles yet so nothing changed
	allHosts := append(darwinHosts, unenrolledHost, linuxHost)
	allHosts = append(allHosts, windowsHosts...)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]:  {},
		darwinHosts[1]:  {},
		darwinHosts[2]:  {},
		unenrolledHost:  {},
		linuxHost:       {},
		windowsHosts[0]: {},
		windowsHosts[1]: {},
		windowsHosts[2]: {},
	})

	// create some global (no-team) profiles
	macGlobalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G1", "G1", "a"),
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
	}
	winGlobalProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G1", "L1"),
		windowsConfigProfileForTest(t, "G2", "L2"),
		windowsConfigProfileForTest(t, "G3", "L3"),
	}
	err = ds.BatchSetMDMProfiles(ctx, nil, macGlobalProfiles, winGlobalProfiles)
	require.NoError(t, err)
	//macGlobalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	//require.NoError(t, err)
	//require.Len(t, macGlobalProfiles, 3)
	globalProfiles := getProfs(nil)
	require.Len(t, globalProfiles, 6)

	// list profiles to install, should result in the global profiles for all
	// enrolled hosts
	toInstallDarwin, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallDarwin, len(macGlobalProfiles)*len(darwinHosts))
	toInstallWindows, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallWindows, len(winGlobalProfiles)*len(windowsHosts))

	// none are listed as "to remove"
	toRemoveDarwin, err := ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveDarwin, 0)
	toRemoveWindows, err := ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveWindows, 0)

	// bulk set for all created hosts, enrolled hosts get the no-team profiles
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// move darwinHosts[0] and windowsHosts[0] to that team
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{darwinHosts[0].ID, windowsHosts[0].ID})
	require.NoError(t, err)

	// 6 are still reported as "to install" because op=install and status=nil
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallDarwin, 6)
	toInstallWindows, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallWindows, 6)

	// those installed to enrolledHosts[0] are listed as "to remove"
	toRemoveDarwin, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveDarwin, 3)
	toRemoveWindows, err = ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveWindows, 3)

	// update status of the moved host (team has no profiles)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(darwinHosts[0], windowsHosts[0]), nil, nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		// windows profiles are directly deleted without a pending state
		windowsHosts[0]: {},
		windowsHosts[1]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create another team
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	// move enrolledHosts[1] to that team
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{darwinHosts[1].ID, windowsHosts[1].ID})
	require.NoError(t, err)

	// 3 are still reported as "to install" because op=install and status=nil
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallDarwin, 3)
	toInstallWindows, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallWindows, 3)

	// 6 are now "to remove" for darwin
	toRemoveDarwin, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveDarwin, 6)
	// 3 are now "to remove" for windows
	toRemoveWindows, err = ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveWindows, 3)

	// update status of the moved host via its uuid (team has no profiles)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, nil, nil, []string{darwinHosts[1].UUID, windowsHosts[1].UUID})
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost:  {},
		linuxHost:       {},
		windowsHosts[0]: {},
		// windows profiles are directly deleted without a pending state
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create profiles for team 1
	tm1DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.1", "T1.1", "d"),
		configProfileForTest(t, "T1.2", "T1.2", "e"),
	}
	tm1WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T1.1", "T1.1"),
		windowsConfigProfileForTest(t, "T1.2", "T1.2"),
	}
	err = ds.BatchSetMDMProfiles(ctx, &team1.ID, tm1DarwinProfiles, tm1WindowsProfiles)
	require.NoError(t, err)

	tm1Profiles := getProfs(&team1.ID)
	require.Len(t, tm1Profiles, 4)

	// 5 are now reported as "to install" (3 global + 2 team1)
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallDarwin, 5)
	toInstallWindows, err = ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstallWindows, 5)

	// 6 are still "to remove"
	toRemoveDarwin, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveDarwin, 6)
	// no profiles to remove in windows
	toRemoveWindows, err = ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemoveWindows, 0)

	// update status of the affected team
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: tm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: tm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: tm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	darwinGlobalProfiles, err := ds.ListMDMAppleConfigProfiles(ctx, nil)
	sort.Slice(darwinGlobalProfiles, func(i, j int) bool {
		l, r := darwinGlobalProfiles[i], darwinGlobalProfiles[j]
		return l.ProfileID < r.ProfileID
	})
	require.NoError(t, err)

	// successfully remove globalProfiles[0, 1] for darwinHosts[0], and remove as failed globalProfiles[2]
	// Do *not* use UpdateOrDeleteHostMDMAppleProfile here, as it deletes/updates based on command uuid
	// (meant to be called from the MDMDirector in response from MDM commands), it would delete/update
	// all rows in this test since we don't have command uuids.
	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			HostUUID: darwinHosts[0].UUID, ProfileID: darwinGlobalProfiles[0].ProfileID,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileID: darwinGlobalProfiles[1].ProfileID,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileID: darwinGlobalProfiles[2].ProfileID,
			Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
	})
	require.NoError(t, err)

	// add a profile to team1, and remove profile T1.1
	newTm1DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.2", "T1.2", "e"),
		configProfileForTest(t, "T1.3", "T1.3", "f"),
	}
	newTm1WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T1.1", "T1.1"),
		windowsConfigProfileForTest(t, "T1.3", "T1.3"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles)
	require.NoError(t, err)

	newTm1Profiles := getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 4)

	// update status of the affected team
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// re-add tm1Profiles[0] to list of team1 profiles
	// NOTE: even though it is the same profile, it's unique DB ID is different because
	// it got deleted and re-inserted from the team's profiles, so this is reflected in
	// the host's profiles list.
	newTm1DarwinProfiles = []*fleet.MDMAppleConfigProfile{
		tm1DarwinProfiles[0],
		configProfileForTest(t, "T1.2", "T1.2", "e"),
		configProfileForTest(t, "T1.3", "T1.3", "f"),
	}
	newTm1WindowsProfiles = []*fleet.MDMWindowsConfigProfile{
		tm1WindowsProfiles[0],
		windowsConfigProfileForTest(t, "T1.2", "T1.2"),
		windowsConfigProfileForTest(t, "T1.3", "T1.3"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles)
	require.NoError(t, err)
	newTm1Profiles = getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 6)

	// update status of the affected team
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: globalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: globalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// remove a global profile and add a new one

	newDarwinGlobalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
		configProfileForTest(t, "G4", "G4", "d"),
	}
	newWindowsGlobalProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G2", "G2"),
		windowsConfigProfileForTest(t, "G3", "G3"),
		windowsConfigProfileForTest(t, "G4", "G4"),
	}

	err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles)
	require.NoError(t, err)

	newGlobalProfiles := getProfs(nil)
	require.Len(t, newGlobalProfiles, 6)

	// update status of the affected "no-team"
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{0}, nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// add another global profile

	newDarwinGlobalProfiles = []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
		configProfileForTest(t, "G4", "G4", "d"),
		configProfileForTest(t, "G5", "G5", "e"),
	}

	newWindowsGlobalProfiles = []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G2", "G2"),
		windowsConfigProfileForTest(t, "G3", "G3"),
		windowsConfigProfileForTest(t, "G4", "G4"),
		windowsConfigProfileForTest(t, "G5", "G5"),
	}

	err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles)
	require.NoError(t, err)
	newGlobalProfiles = getProfs(nil)
	require.Len(t, newGlobalProfiles, 8)

	newDarwinProfileID, err := strconv.ParseUint(newGlobalProfiles[3].ProfileID, 10, 64)
	require.NoError(t, err)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []uint{uint(newDarwinProfileID)}, []string{}, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: newGlobalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[6].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []uint{}, []string{newGlobalProfiles[7].ProfileID}, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileID: newGlobalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[6].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[7].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// add a profile to team2

	tm2DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T2.1", "T2.1", "a"),
	}

	tm2WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T2.1", "T2.1"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team2.ID, tm2DarwinProfiles, tm2WindowsProfiles)
	require.NoError(t, err)
	tm2Profiles := getProfs(&team2.ID)
	require.Len(t, tm2Profiles, 2)

	// update status via tm2 id and the global 0 id to test that custom sql statement
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID, 0}, nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileID: tm2Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileID: newGlobalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[6].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[7].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// simulate an entry with some values set to NULL
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET detail = NULL WHERE profile_id = ?`, globalProfiles[2].ProfileID)
		if err != nil {
			return err
		}
		return nil
	})

	// do a final sync of all hosts, should not change anything
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(append(darwinHosts, unenrolledHost, linuxHost)...), nil, nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileID: newTm1Profiles[3].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newTm1Profiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileID: tm2Profiles[1].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileID: newGlobalProfiles[4].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[5].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[6].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileID: newGlobalProfiles[7].ProfileID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})
}

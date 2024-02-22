package mysql

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/tokenpki"
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
		{"TestGetHostMDMProfilesExpectedForVerification", testGetHostMDMProfilesExpectedForVerification},
		{"TestBatchSetProfileLabelAssociations", testBatchSetProfileLabelAssociations},
		{"TestBatchSetProfilesTransactionError", testBatchSetMDMProfilesTransactionError},
		{"TestMDMEULA", testMDMEULA},
		{"TestGetHostCertAssociationsToExpire", testSCEPRenewalHelpers},
		{"TestSCEPRenewalHelpers", testSCEPRenewalHelpers},
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
		expectAppleProfiles(t, ds, tmID, wantApple)
		expectWindowsProfiles(t, ds, tmID, wantWindows)
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

	// add fleet-managed Apple profiles for the team and globally
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

	// add fleet-managed Windows profiles for the team and globally
	for name := range mdm_types.FleetReservedProfileNames() {
		_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: name, TeamID: &team.ID, SyncML: winProf})
		require.NoError(t, err)
		_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: name, TeamID: nil, SyncML: winProf})
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

	// create 8 labels for label-based profiles
	var labels []*fleet.Label
	for i := 0; i < 8; i++ {
		lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "l" + strconv.Itoa(i), Query: "select 1"})
		require.NoError(t, err)
		labels = append(labels, lbl)
	}

	// create more profiles and test the pagination with a table-driven test so that
	// global and team both have 9 profiles (including A and B already created above).
	for i := 0; i < 3; i++ {
		inc := i * 4 // e.g. C, D, E, F on first loop, G, H, I, J on second loop, etc.

		// create label-based profiles for i==0, meaning CDEF will be label-based
		acp := *generateCP(string(rune('C'+inc)), string(rune('C'+inc)), 0)
		if i == 0 {
			acp.Labels = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[0].Name, LabelID: labels[0].ID},
				{LabelName: labels[1].Name, LabelID: labels[1].ID},
			}
		}
		_, err = ds.NewMDMAppleConfigProfile(ctx, acp)
		require.NoError(t, err)

		acp = *generateCP(string(rune('C'+inc+1)), string(rune('C'+inc+1)), team.ID)
		if i == 0 {
			acp.Labels = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[2].Name, LabelID: labels[2].ID},
				{LabelName: labels[3].Name, LabelID: labels[3].ID},
			}
		}
		_, err = ds.NewMDMAppleConfigProfile(ctx, acp)
		require.NoError(t, err)

		wcp := fleet.MDMWindowsConfigProfile{Name: string(rune('C' + inc + 2)), TeamID: nil, SyncML: winProf}
		if i == 0 {
			wcp.Labels = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[4].Name, LabelID: labels[4].ID},
				{LabelName: labels[5].Name, LabelID: labels[5].ID},
			}
		}
		_, err = ds.NewMDMWindowsConfigProfile(ctx, wcp)
		require.NoError(t, err)

		wcp = fleet.MDMWindowsConfigProfile{Name: string(rune('C' + inc + 3)), TeamID: &team.ID, SyncML: winProf}
		if i == 0 {
			wcp.Labels = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[6].Name, LabelID: labels[6].ID},
				{LabelName: labels[7].Name, LabelID: labels[7].ID},
			}
		}
		_, err = ds.NewMDMWindowsConfigProfile(ctx, wcp)
		require.NoError(t, err)
	}

	// delete label 3 and 4 so that profiles D and E are broken
	require.NoError(t, ds.DeleteLabel(ctx, labels[3].Name))
	require.NoError(t, ds.DeleteLabel(ctx, labels[4].Name))
	profLabels := map[string][]fleet.ConfigurationProfileLabel{
		"C": {
			{LabelName: labels[0].Name, LabelID: labels[0].ID},
			{LabelName: labels[1].Name, LabelID: labels[1].ID},
		},
		"D": {
			{LabelName: labels[2].Name, LabelID: labels[2].ID},
			{LabelName: labels[3].Name, LabelID: 0, Broken: true},
		},
		"E": {
			{LabelName: labels[4].Name, LabelID: 0, Broken: true},
			{LabelName: labels[5].Name, LabelID: labels[5].ID},
		},
		"F": {
			{LabelName: labels[6].Name, LabelID: labels[6].ID},
			{LabelName: labels[7].Name, LabelID: labels[7].ID},
		},
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

				wantProfs := profLabels[p.Name]
				require.Equal(t, len(wantProfs), len(p.Labels), "profile name: %s", p.Name)
				if len(wantProfs) > 0 {
					// clear the profile uuids from the labels list
					for i, l := range p.Labels {
						l.ProfileUUID = ""
						p.Labels[i] = l
					}
					require.ElementsMatch(t, wantProfs, p.Labels, "profile name: %s", p.Name)
				}
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
	ds.testUpsertMDMDesiredProfilesBatchSize = 2
	ds.testDeleteMDMProfilesBatchSize = 2
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
		ds.testDeleteMDMProfilesBatchSize = 0
	})
	testBulkSetPendingMDMHostProfiles(t, ds)
}

func testBulkSetPendingMDMHostProfilesBatch3(t *testing.T, ds *Datastore) {
	ds.testUpsertMDMDesiredProfilesBatchSize = 3
	ds.testDeleteMDMProfilesBatchSize = 3
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
		ds.testDeleteMDMProfilesBatchSize = 0
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
		ProfileUUID      string
		Status           *fleet.MDMDeliveryStatus
		OperationType    fleet.MDMOperationType
		IdentifierOrName string
	}

	// only asserts the profile ID, status and operation
	assertHostProfiles := func(want map[*fleet.Host][]anyProfile) {
		// TODO(mna): it would help readability of this test to capture the "last
		// state" of this call and accept the diff as the expected result, merging
		// them together before the assertions. Would need some hackery to clear a
		// profile from the list.

		for h, wantProfs := range want {
			var gotProfs []anyProfile

			switch h.Platform {
			case "windows":
				profs, err := ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
				require.NoError(t, err)
				require.Equal(t, len(wantProfs), len(profs), "host uuid: %s", h.UUID)
				for _, p := range profs {
					gotProfs = append(gotProfs, anyProfile{
						ProfileUUID:      p.ProfileUUID,
						Status:           p.Status,
						OperationType:    p.OperationType,
						IdentifierOrName: p.Name,
					})
				}
			default:
				profs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
				require.NoError(t, err)
				require.Equal(t, len(wantProfs), len(profs), "host uuid: %s", h.UUID)
				for _, p := range profs {
					gotProfs = append(gotProfs, anyProfile{
						ProfileUUID:      p.ProfileUUID,
						Status:           p.Status,
						OperationType:    p.OperationType,
						IdentifierOrName: p.Identifier,
					})
				}
			}

			sortProfs := func(profs []anyProfile) []anyProfile {
				sort.Slice(profs, func(i, j int) bool {
					l, r := profs[i], profs[j]
					if l.ProfileUUID == r.ProfileUUID {
						return l.OperationType < r.OperationType
					}

					// default alphabetical comparison
					return l.IdentifierOrName < r.IdentifierOrName
				})
				return profs
			}

			gotProfs = sortProfs(gotProfs)
			wantProfs = sortProfs(wantProfs)
			for i, wp := range wantProfs {
				gp := gotProfs[i]
				require.Equal(t, wp.ProfileUUID, gp.ProfileUUID, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
				require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
				require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id or name: %s", h.UUID, gp.IdentifierOrName)
			}
		}
	}

	getProfs := func(teamID *uint) []*fleet.MDMConfigProfilePayload {
		// TODO(roberto): the docs says that you can pass a comma separated
		// list of columns to OrderKey, but that doesn't seem to work
		profs, _, err := ds.ListMDMConfigProfiles(ctx, teamID, fleet.ListOptions{})
		require.NoError(t, err)
		sort.Slice(profs, func(i, j int) bool {
			l, r := profs[i], profs[j]
			if l.Platform != r.Platform {
				return l.Platform < r.Platform
			}

			return l.Name < r.Name
		})
		return profs
	}

	// create some darwin hosts, all enrolled
	var darwinHosts []*fleet.Host // not pre-allocating, causes gosec false positive
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
		darwinHosts = append(darwinHosts, h)
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
	var windowsHosts []*fleet.Host // not preallocating, causes gosec false positive
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
		windowsHosts = append(windowsHosts, h)
		t.Logf("enrolled windows host [%d]: %s", j, h.UUID)
	}

	// bulk set for no target ids, does nothing
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, nil, nil)
	require.NoError(t, err)
	// bulk set for combination of target ids, not allowed
	err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{1}, []uint{2}, nil, nil)
	require.Error(t, err)

	// bulk set for all created hosts, no profiles yet so nothing changed
	allHosts := append(darwinHosts, unenrolledHost, linuxHost)
	allHosts = append(allHosts, windowsHosts...)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil)
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
		configProfileForTest(t, "G1a", "G1a", "a"),
		configProfileForTest(t, "G2a", "G2a", "b"),
		configProfileForTest(t, "G3a", "G3a", "c"),
	}
	winGlobalProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G1w", "L1"),
		windowsConfigProfileForTest(t, "G2w", "L2"),
		windowsConfigProfileForTest(t, "G3w", "L3"),
	}
	err = ds.BatchSetMDMProfiles(ctx, nil, macGlobalProfiles, winGlobalProfiles)
	require.NoError(t, err)
	macGlobalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, macGlobalProfiles, 3)
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
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
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
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(darwinHosts[0], windowsHosts[0]), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		// windows profiles are directly deleted without a pending state (there's no on-host removal of profiles)
		windowsHosts[0]: {},
		windowsHosts[1]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
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
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, nil, []string{darwinHosts[1].UUID, windowsHosts[1].UUID})
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost:  {},
		linuxHost:       {},
		windowsHosts[0]: {},
		// windows profiles are directly deleted without a pending state
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create profiles for team 1
	tm1DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.1a", "T1.1a", "d"),
		configProfileForTest(t, "T1.2a", "T1.2a", "e"),
	}
	tm1WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T1.1w", "T1.1"),
		windowsConfigProfileForTest(t, "T1.2w", "T1.2"),
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
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: tm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	darwinGlobalProfiles, err := ds.ListMDMAppleConfigProfiles(ctx, nil)
	sort.Slice(darwinGlobalProfiles, func(i, j int) bool {
		l, r := darwinGlobalProfiles[i], darwinGlobalProfiles[j]
		return l.Name < r.Name
	})
	require.NoError(t, err)

	// successfully remove globalProfiles[0, 1] for darwinHosts[0], and remove as
	// failed globalProfiles[2] Do *not* use UpdateOrDeleteHostMDMAppleProfile
	// here, as it deletes/updates based on command uuid (meant to be called from
	// the MDMDirector in response from MDM commands), it would delete/update all
	// rows in this test since we don't have command uuids.
	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[0].ProfileUUID,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[1].ProfileUUID,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[2].ProfileUUID,
			Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"),
		},
	})
	require.NoError(t, err)

	// add a profile to team1, and remove profile T1.1 on Apple, T1.2 on Windows
	newTm1DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.2a", "T1.2a", "e"),
		configProfileForTest(t, "T1.3a", "T1.3a", "f"),
	}
	newTm1WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T1.1w", "T1.1"),
		windowsConfigProfileForTest(t, "T1.3w", "T1.3"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles)
	require.NoError(t, err)

	newTm1Profiles := getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 4)

	// update status of the affected team
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// re-add tm1Profiles[0] to list of team1 profiles (T1.1 on Apple, T1.2 on Windows)
	// NOTE: even though it is the same profile, it's unique DB ID is different because
	// it got deleted and re-inserted from the team's profiles, so this is reflected in
	// the host's profiles list.
	newTm1DarwinProfiles = []*fleet.MDMAppleConfigProfile{
		tm1DarwinProfiles[0],
		configProfileForTest(t, "T1.2a", "T1.2a", "e"),
		configProfileForTest(t, "T1.3a", "T1.3a", "f"),
	}
	newTm1WindowsProfiles = []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T1.1w", "T1.1"),
		tm1WindowsProfiles[1],
		windowsConfigProfileForTest(t, "T1.3w", "T1.3"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles)
	require.NoError(t, err)
	newTm1Profiles = getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 6)

	// update status of the affected team
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: globalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: globalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// remove a global profile and add a new one

	newDarwinGlobalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2a", "G2a", "b"),
		configProfileForTest(t, "G3a", "G3a", "c"),
		configProfileForTest(t, "G4a", "G4a", "d"),
	}
	newWindowsGlobalProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G2w", "G2"),
		windowsConfigProfileForTest(t, "G3w", "G3"),
		windowsConfigProfileForTest(t, "G4w", "G4"),
	}

	err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles)
	require.NoError(t, err)

	newGlobalProfiles := getProfs(nil)
	require.Len(t, newGlobalProfiles, 6)

	// update status of the affected "no-team"
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{0}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// add another global profile

	newDarwinGlobalProfiles = []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2a", "G2a", "b"),
		configProfileForTest(t, "G3a", "G3a", "c"),
		configProfileForTest(t, "G4a", "G4a", "d"),
		configProfileForTest(t, "G5a", "G5a", "e"),
	}

	newWindowsGlobalProfiles = []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G2w", "G2"),
		windowsConfigProfileForTest(t, "G3w", "G3"),
		windowsConfigProfileForTest(t, "G4w", "G4"),
		windowsConfigProfileForTest(t, "G5w", "G5"),
	}

	err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles)
	require.NoError(t, err)
	newGlobalProfiles = getProfs(nil)
	require.Len(t, newGlobalProfiles, 8)

	// bulk-set only those affected by the new Apple global profile
	newDarwinProfileUUID := newGlobalProfiles[3].ProfileUUID
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newDarwinProfileUUID}, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// bulk-set only those affected by the new Apple global profile
	newWindowsProfileUUID := newGlobalProfiles[7].ProfileUUID
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newWindowsProfileUUID}, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// add a profile to team2

	tm2DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T2.1a", "T2.1a", "a"),
	}

	tm2WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T2.1w", "T2.1"),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team2.ID, tm2DarwinProfiles, tm2WindowsProfiles)
	require.NoError(t, err)
	tm2Profiles := getProfs(&team2.ID)
	require.Len(t, tm2Profiles, 2)

	// update status via tm2 id and the global 0 id to test that custom sql statement
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID, 0}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create some labels for label-based profiles
	var labels []*fleet.Label
	for i := 0; i < 6; i++ {
		lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "l" + strconv.Itoa(i), Query: "select 1"})
		require.NoError(t, err)
		labels = append(labels, lbl)
	}

	// TODO(mna): temporary, until BatchSetMDMProfiles supports labels
	setProfileLabels := func(t *testing.T, p *fleet.MDMConfigProfilePayload, labels ...*fleet.Label) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx, `DELETE FROM mdm_configuration_profile_labels WHERE apple_profile_uuid = ? OR windows_profile_uuid = ?`, p.ProfileUUID, p.ProfileUUID); err != nil {
				return err
			}

			var auuid, wuuid *string
			if p.Platform == "windows" {
				wuuid = &p.ProfileUUID
			} else {
				auuid = &p.ProfileUUID
			}
			for _, lbl := range labels {
				if _, err := q.ExecContext(ctx, `INSERT INTO mdm_configuration_profile_labels
					(apple_profile_uuid, windows_profile_uuid, label_name, label_id)
					VALUES
					(?, ?, ?, ?)`, auuid, wuuid, lbl.Name, lbl.ID); err != nil {
					return err
				}
			}
			return err
		})
	}

	// create two global label-based profiles for each OS, and two team-based
	newDarwinGlobalProfiles = []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2a", "G2a", "b"),
		configProfileForTest(t, "G3a", "G3a", "c"),
		configProfileForTest(t, "G4a", "G4a", "d"),
		configProfileForTest(t, "G5a", "G5a", "e"),
		configProfileForTest(t, "G6a", "G6a", "f", labels[0], labels[1]),
		configProfileForTest(t, "G7a", "G7a", "g", labels[2]),
	}

	newWindowsGlobalProfiles = []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G2w", "G2"),
		windowsConfigProfileForTest(t, "G3w", "G3"),
		windowsConfigProfileForTest(t, "G4w", "G4"),
		windowsConfigProfileForTest(t, "G5w", "G5"),
		windowsConfigProfileForTest(t, "G6w", "G6", labels[3], labels[4]),
		windowsConfigProfileForTest(t, "G7w", "G7", labels[5]),
	}

	err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles)
	require.NoError(t, err)
	newGlobalProfiles = getProfs(nil)
	require.Len(t, newGlobalProfiles, 12)
	// TODO(mna): temporary until BatchSetMDMProfiles supports labels
	setProfileLabels(t, newGlobalProfiles[4], labels[0], labels[1])
	setProfileLabels(t, newGlobalProfiles[5], labels[2])
	setProfileLabels(t, newGlobalProfiles[10], labels[3], labels[4])
	setProfileLabels(t, newGlobalProfiles[11], labels[5])

	// simulate an entry with some values set to NULL
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET detail = NULL WHERE profile_uuid = ?`, globalProfiles[2].ProfileUUID)
		return err
	})

	// do a sync of all hosts, should not change anything as no host is a member
	// of the new label-based profiles (indices change due to new Apple and
	// Windows profiles)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...), nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// create a new Apple and Windows hosts, global (no team)
	i = 8
	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("test-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("test-uuid-%d", i),
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, h)
	windowsHosts = append(windowsHosts, h)
	t.Logf("enrolled windows host [%d]: %s", len(windowsHosts)-1, h.UUID)

	i = 9
	h, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("test-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("test-uuid-%d", i),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, h, false)
	darwinHosts = append(darwinHosts, h)
	t.Logf("enrolled darwin host [%d]: %s", len(darwinHosts)-1, h.UUID)

	// make the new Apple host a member of labels[0] and [1]
	// make the new Windows host a member of labels[3] and [4]
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[0].ID, darwinHosts[3].ID}, {labels[1].ID, darwinHosts[3].ID},
		{labels[3].ID, windowsHosts[3].ID}, {labels[4].ID, windowsHosts[3].ID},
	})
	require.NoError(t, err)

	// do a full sync, the new global hosts get the standard global profiles and
	// also the label-based profile that they are a member of
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...), nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// make the darwinHosts[2] host a member of all labels
	// make the windowsHosts[2] host a member of all labels
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[0].ID, darwinHosts[2].ID}, {labels[1].ID, darwinHosts[2].ID}, {labels[2].ID, darwinHosts[2].ID},
		{labels[3].ID, darwinHosts[2].ID}, {labels[4].ID, darwinHosts[2].ID}, {labels[5].ID, darwinHosts[2].ID},
		{labels[0].ID, windowsHosts[2].ID}, {labels[1].ID, windowsHosts[2].ID}, {labels[2].ID, windowsHosts[2].ID},
		{labels[3].ID, windowsHosts[2].ID}, {labels[4].ID, windowsHosts[2].ID}, {labels[5].ID, windowsHosts[2].ID},
	})
	require.NoError(t, err)

	// do a sync of those hosts, they will get the two label-based profiles of their platform
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(darwinHosts[2], windowsHosts[2]), nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[11].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// "break" the two G6 label-based profile by deleting labels[0] and [3]
	require.NoError(t, ds.DeleteLabel(ctx, labels[0].Name))
	require.NoError(t, ds.DeleteLabel(ctx, labels[3].Name))

	// sync the affected profiles
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newGlobalProfiles[4].ProfileUUID}, nil)
	require.NoError(t, err)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newGlobalProfiles[10].ProfileUUID}, nil)
	require.NoError(t, err)

	// nothing changes - broken label-based profiles are simply ignored
	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[11].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// update darwin/windows[2] so they are not members of labels[1][2] and [4][5], which
	// should remove the G7 label-based profile, but not G6 as it is broken.
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[2].ID}, {labels[2].ID, darwinHosts[2].ID},
		{labels[4].ID, windowsHosts[2].ID}, {labels[5].ID, windowsHosts[2].ID},
	})
	require.NoError(t, err)

	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(darwinHosts[2], windowsHosts[2]), nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// "unbreak" the two G6 label-based profiles by removing the deleted labels
	// from their requirements
	setProfileLabels(t, newGlobalProfiles[4], labels[1])
	setProfileLabels(t, newGlobalProfiles[10], labels[4])

	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newGlobalProfiles[4].ProfileUUID}, nil)
	require.NoError(t, err)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newGlobalProfiles[10].ProfileUUID}, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// add a label-based profile to team 2
	tm2DarwinProfiles = []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T2.1a", "T2.1a", "a"),
		configProfileForTest(t, "T2.2a", "T2.2a", "b", labels[1], labels[2]),
	}
	tm2WindowsProfiles = []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T2.1w", "T2.1"),
		windowsConfigProfileForTest(t, "T2.2w", "T2.2", labels[4], labels[5]),
	}

	err = ds.BatchSetMDMProfiles(ctx, &team2.ID, tm2DarwinProfiles, tm2WindowsProfiles)
	require.NoError(t, err)
	tm2Profiles = getProfs(&team2.ID)
	require.Len(t, tm2Profiles, 4)
	// TODO(mna): temporary until BatchSetMDMProfiles supports labels
	setProfileLabels(t, tm2Profiles[1], labels[1], labels[2])
	setProfileLabels(t, tm2Profiles[3], labels[4], labels[5])

	// sync team 2, no changes because no host is a member of the labels (except
	// index change due to new profiles)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// make darwinHosts[1] and windowsHosts[1] members of the required labels
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[1].ID}, {labels[2].ID, darwinHosts[1].ID},
		{labels[4].ID, windowsHosts[1].ID}, {labels[5].ID, windowsHosts[1].ID},
	})
	require.NoError(t, err)

	// sync team 2, the label-based profile of team2 is now pending install
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// "break" the team 2 label-based profile by deleting a label
	require.NoError(t, ds.DeleteLabel(ctx, labels[1].Name))
	require.NoError(t, ds.DeleteLabel(ctx, labels[4].Name))

	// sync team 2, the label-based profile of team2 is left untouched (broken
	// profiles are ignored)
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// remove team 2 hosts membership from labels
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[1].ID}, {labels[2].ID, darwinHosts[1].ID},
		{labels[4].ID, windowsHosts[1].ID}, {labels[5].ID, windowsHosts[1].ID},
	})
	require.NoError(t, err)

	// sync team 2, the label-based profile of team2 is still left untouched
	// because even if the hosts are not members anymore, the profile is broken
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// "unbreak" the profile by removing the deleted label from its requirements
	setProfileLabels(t, tm2Profiles[1], labels[2])
	setProfileLabels(t, tm2Profiles[3], labels[5])

	// sync team 2, now it sees that the hosts are not members and the profile
	// gets removed
	err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})

	// sanity-check, a full sync does not change anything
	err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...), nil, nil, nil)
	require.NoError(t, err)

	assertHostProfiles(map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newTm1Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		darwinHosts[1]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: globalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: tm2Profiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: tm2Profiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[2]: {
			{ProfileUUID: globalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
			{ProfileUUID: newGlobalProfiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove},
		},
		darwinHosts[3]: {
			{ProfileUUID: newGlobalProfiles[0].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[1].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{ProfileUUID: newTm1Profiles[3].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[4].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newTm1Profiles[5].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[1]: {
			{ProfileUUID: tm2Profiles[2].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[2]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
		windowsHosts[3]: {
			{ProfileUUID: newGlobalProfiles[6].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[7].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[8].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[9].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
			{ProfileUUID: newGlobalProfiles[10].ProfileUUID, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall},
		},
	})
}

func testGetHostMDMProfilesExpectedForVerification(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Setup funcs

	// ===================================================
	// MacOS
	// ===================================================
	setup1 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test",
			OsqueryHostID: ptr.String("osquery-macos"),
			NodeKey:       ptr.String("node-key-macos"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T1.1", "T1.1", "d"),
			configProfileForTest(t, "T1.2", "T1.2", "e"),
		}

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil)
		require.NoError(t, err)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 2)

		return team.ID, host.ID
	}

	setup2 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-2",
			OsqueryHostID: ptr.String("osquery-macos-2"),
			NodeKey:       ptr.String("node-key-macos-2"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T2.1", "T2.1", "d"),
			configProfileForTest(t, "T2.2", "T2.2", "e"),
			configProfileForTest(t, "labeled_prof", "labeled_prof", "labeled_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_1"})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`, "labeled_prof")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					label.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					label.Name,
					label.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		return team.ID, host.ID
	}

	setup3 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-3",
			OsqueryHostID: ptr.String("osquery-macos-3"),
			NodeKey:       ptr.String("node-key-macos-3"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 3"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T3.1", "T3.1", "d"),
			configProfileForTest(t, "T3.2", "T3.2", "e"),
			configProfileForTest(t, "labeled_prof_2", "labeled_prof_2", "labeled_prof_2"),
		}

		testLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_2"})
		require.NoError(t, err)

		testLabel3, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_3"})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`, "labeled_prof_2")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					testLabel2.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					testLabel2.Name,
					testLabel2.ID,
				)
				return err
			},
		)

		// Also add mapping to test label 3
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					testLabel3.Name,
					testLabel3.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		return team.ID, host.ID
	}

	setup4 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-4",
			OsqueryHostID: ptr.String("osquery-macos-4"),
			NodeKey:       ptr.String("node-key-macos-4"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 4"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T4.1", "T4.1", "d"),
			configProfileForTest(t, "T4.2", "T4.2", "e"),
			configProfileForTest(t, "broken_label_prof", "broken_label_prof", "broken_label_prof"),
		}

		testLabel4, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_4"})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`, "broken_label_prof")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					testLabel4.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					testLabel4.Name,
					testLabel4.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		// Now delete label, we shouldn't see the related profile
		err = ds.DeleteLabel(ctx, testLabel4.Name)
		require.NoError(t, err)

		return team.ID, host.ID
	}

	// ===================================================
	// Windows
	// ===================================================

	setup5 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "windows-test",
			OsqueryHostID: ptr.String("osquery-windows"),
			NodeKey:       ptr.String("node-key-windows"),
			UUID:          uuid.NewString(),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, host)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 5"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T5.1", "T5.1"),
			windowsConfigProfileForTest(t, "T5.2", "T5.2"),
		}

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles)
		require.NoError(t, err)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 2)

		return team.ID, host.ID
	}

	setup6 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "windows-test-2",
			OsqueryHostID: ptr.String("osquery-windows-2"),
			NodeKey:       ptr.String("node-key-windows-2"),
			UUID:          uuid.NewString(),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, host)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 6"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T6.1", "T6.1"),
			windowsConfigProfileForTest(t, "T6.2", "T6.2"),
			windowsConfigProfileForTest(t, "labeled_prof", "labeled_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_6"})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`, "labeled_prof")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					label.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					label.Name,
					label.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		return team.ID, host.ID
	}

	setup7 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "windows-test-3",
			OsqueryHostID: ptr.String("osquery-windows-3"),
			NodeKey:       ptr.String("node-key-windows-3"),
			UUID:          uuid.NewString(),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, host)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 7"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T7.1", "T7.1"),
			windowsConfigProfileForTest(t, "T7.2", "T3.7"),
			windowsConfigProfileForTest(t, "labeled_prof_2", "labeled_prof_2"),
		}

		testLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		testLabel3, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`, "labeled_prof_2")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					testLabel2.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					testLabel2.Name,
					testLabel2.ID,
				)
				return err
			},
		)

		// Also add mapping to test label 3
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					testLabel3.Name,
					testLabel3.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		return team.ID, host.ID
	}

	setup8 := func() (uint, uint) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "windows-test-4",
			OsqueryHostID: ptr.String("osquery-windows-4"),
			NodeKey:       ptr.String("node-key-windows-4"),
			UUID:          uuid.NewString(),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, host)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 8"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
		require.NoError(t, err)

		// create profiles for team
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T8.1", "T8.1"),
			windowsConfigProfileForTest(t, "T8.2", "T8.2"),
			windowsConfigProfileForTest(t, "broken_label_prof", "broken_label_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		err = ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles)
		require.NoError(t, err)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`, "broken_label_prof")
		})

		// Update label with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
					host.ID,
					label.ID,
				)
				return err
			},
		)

		// Update profile <-> label mapping
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
					uid,
					label.Name,
					label.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		// Now delete label, we shouldn't see the related profile
		err = ds.DeleteLabel(ctx, label.Name)
		require.NoError(t, err)

		return team.ID, host.ID
	}

	tests := []struct {
		name        string
		setupFunc   func() (uint, uint)
		wantMac     map[string]*fleet.ExpectedMDMProfile
		wantWindows map[string]*fleet.ExpectedMDMProfile
		os          string
	}{
		{
			name:      "macos basic team profiles no labels",
			setupFunc: setup1,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T1.1": {Identifier: "T1.1"},
				"T1.2": {Identifier: "T1.2"},
			},
		},
		{
			name:      "macos labeled team profile",
			setupFunc: setup2,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T2.1":         {Identifier: "T2.1"},
				"T2.2":         {Identifier: "T2.2"},
				"labeled_prof": {Identifier: "labeled_prof"},
			},
		},
		{
			name:      "macos labeled team profile with additional labeled profile",
			setupFunc: setup3,
			// Our expected profiles should not include the labeled profile, because it
			// maps to a label that is not applied to the host.
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T3.1": {Identifier: "T3.1"},
				"T3.2": {Identifier: "T3.2"},
			},
		},
		{
			name:      "macos profile with broken label",
			setupFunc: setup4,
			// Our expected profiles should not include the labeled profile, because it is broken
			// (the label was deleted)
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T4.1": {Identifier: "T4.1"},
				"T4.2": {Identifier: "T4.2"},
			},
		},
		{
			name:      "windows basic team profiles no labels",
			setupFunc: setup5,
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T5.1": {Name: "T5.1"},
				"T5.2": {Name: "T5.2"},
			},
		},
		{
			name:      "windows labeled team profile",
			setupFunc: setup6,
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T6.1":         {Name: "T6.1"},
				"T6.2":         {Name: "T6.2"},
				"labeled_prof": {Name: "labeled_prof"},
			},
		},
		{
			name:      "windows labeled team profile with additional labeled profile",
			setupFunc: setup7,
			// Our expected profiles should not include the labeled profile, because it
			// maps to a label that is not applied to the host.
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T7.1": {Name: "T7.1"},
				"T7.2": {Name: "T7.2"},
			},
		},
		{
			name:      "windows profile with broken label",
			setupFunc: setup8,
			// Our expected profiles should not include the labeled profile, because it is broken
			// (the label was deleted)
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T8.1": {Name: "T8.1"},
				"T8.2": {Name: "T8.2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, hostID := tt.setupFunc()
			if len(tt.wantMac) > 0 {
				got, err := ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID, hostID)
				require.NoError(t, err)
				for k, v := range tt.wantMac {
					require.Contains(t, got, k)
					require.Equal(t, v.Identifier, got[k].Identifier)
				}
			}

			if len(tt.wantWindows) > 0 {
				got, err := ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID, hostID)
				require.NoError(t, err)
				for k, v := range tt.wantWindows {
					require.Contains(t, got, k)
					require.Equal(t, v.Name, got[k].Name)
				}
			}
		})
	}
}

func testBatchSetProfileLabelAssociations(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a label
	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
	}
	label, err := ds.NewLabel(ctx, label)
	require.NoError(t, err)

	// create a macOS config profile
	macOSProfile, err := ds.NewMDMAppleConfigProfile(
		ctx,
		fleet.MDMAppleConfigProfile{
			Name:         "DummyTestName",
			Identifier:   "DummyTestIdentifier",
			Mobileconfig: mobileconfig.Mobileconfig([]byte("DummyTestMobileconfigBytes")),
			TeamID:       nil,
		},
	)
	require.NoError(t, err)
	otherMacProfile, err := ds.NewMDMAppleConfigProfile(
		ctx,
		fleet.MDMAppleConfigProfile{
			Name:         "OtherDummyTestName",
			Identifier:   "OtherDummyTestIdentifier",
			Mobileconfig: mobileconfig.Mobileconfig([]byte("OtherDummyTestMobileconfigBytes")),
			TeamID:       nil,
		},
	)
	require.NoError(t, err)

	// create a Windows config profile
	windowsProfile, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		fleet.MDMWindowsConfigProfile{
			Name:   "with-labels",
			TeamID: nil,
			SyncML: []byte("<Replace></Replace>"),
		},
	)
	require.NoError(t, err)
	otherWinProfile, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		fleet.MDMWindowsConfigProfile{
			Name:   "other-with-labels",
			TeamID: nil,
			SyncML: []byte("<Replace></Replace>"),
		},
	)
	require.NoError(t, err)

	// assign the label to the "other" profiles, should not change throughout the test
	wantOtherWin := []fleet.ConfigurationProfileLabel{
		{ProfileUUID: otherWinProfile.ProfileUUID, LabelName: label.Name, LabelID: label.ID},
	}
	require.NoError(t, batchSetProfileLabelAssociationsDB(ctx, ds.writer(ctx), wantOtherWin, "windows"))
	wantOtherMac := []fleet.ConfigurationProfileLabel{
		{ProfileUUID: otherMacProfile.ProfileUUID, LabelName: label.Name, LabelID: label.ID},
	}
	require.NoError(t, batchSetProfileLabelAssociationsDB(ctx, ds.writer(ctx), wantOtherMac, "darwin"))

	platforms := map[string]string{
		"darwin":  macOSProfile.ProfileUUID,
		"windows": windowsProfile.ProfileUUID,
	}

	for platform, uuid := range platforms {
		expectLabels := func(t *testing.T, profUUID, platform string, want []fleet.ConfigurationProfileLabel) {
			if len(want) == 0 {
				return
			}

			p := platform
			if p == "darwin" {
				p = "apple"
			}

			query := fmt.Sprintf("SELECT %s_profile_uuid as profile_uuid, label_id, label_name FROM mdm_configuration_profile_labels WHERE %s_profile_uuid = ?", p, p)

			var got []fleet.ConfigurationProfileLabel
			ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
				err := sqlx.SelectContext(ctx, tx, &got, query, profUUID)
				require.NoError(t, err)
				require.Len(t, got, len(want))
				return nil
			})
			require.ElementsMatch(t, want, got)
		}

		t.Run("empty input "+platform, func(t *testing.T) {
			want := []fleet.ConfigurationProfileLabel{}
			err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, want, platform)
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, want)
			// does not change other profiles
			expectLabels(t, otherWinProfile.ProfileUUID, "windows", wantOtherWin)
			expectLabels(t, otherMacProfile.ProfileUUID, "darwin", wantOtherMac)
		})

		t.Run("valid input "+platform, func(t *testing.T) {
			profileLabels := []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID},
			}
			err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, platform)
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, profileLabels)
			// does not change other profiles
			expectLabels(t, otherWinProfile.ProfileUUID, "windows", wantOtherWin)
			expectLabels(t, otherMacProfile.ProfileUUID, "darwin", wantOtherMac)
		})

		t.Run("invalid profile UUID "+platform, func(t *testing.T) {
			invalidProfileLabels := []fleet.ConfigurationProfileLabel{
				{ProfileUUID: "invalid-uuid", LabelName: label.Name, LabelID: label.ID},
			}

			err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, platform)
			})
			require.Error(t, err)
		})

		t.Run("invalid label data "+platform, func(t *testing.T) {
			// invalid id
			invalidProfileLabels := []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: 12345},
			}
			err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, platform)
			})
			require.Error(t, err)

			// both invalid
			invalidProfileLabels = []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: "xyz", LabelID: 1235},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, platform)
			})
			require.Error(t, err)
		})

		t.Run("labels are removed "+platform, func(t *testing.T) {
			// create a new label
			newLabel := &fleet.Label{
				Name:        "new label" + platform,
				Description: "a label",
				Query:       "select 1 from orbit_info;",
			}
			newLabel, err := ds.NewLabel(ctx, newLabel)
			require.NoError(t, err)

			// apply a batch set with the new label
			profileLabels := []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID},
				{ProfileUUID: uuid, LabelName: newLabel.Name, LabelID: newLabel.ID},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, platform)
			})
			require.NoError(t, err)
			// both are stored in the DB
			expectLabels(t, uuid, platform, profileLabels)

			// batch apply again without the newLabel
			profileLabels = []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				return batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, platform)
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, profileLabels)

			// does not change other profiles
			expectLabels(t, otherWinProfile.ProfileUUID, "windows", wantOtherWin)
			expectLabels(t, otherMacProfile.ProfileUUID, "darwin", wantOtherMac)
		})
	}

	t.Run("unsupported platform", func(t *testing.T) {
		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			return batchSetProfileLabelAssociationsDB(ctx, tx, []fleet.ConfigurationProfileLabel{{}}, "unsupported")
		})
		require.Error(t, err)
	})
}

func testBatchSetMDMProfilesTransactionError(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "label", Query: "select 1"})
	require.NoError(t, err)

	cases := []struct {
		windowsErr string
		appleErr   string
		wantErr    string
	}{
		{"select:a", "", "batch set windows profiles: load existing profiles: select:a"},
		{"insert:b", "", ": insert:b"},
		{"delete:c", "", "batch set windows profiles: delete obsolete profiles: delete:c"},
		{"reselect:d", "", "batch set windows profiles: load newly inserted profiles: reselect:d"},
		{"labels:e", "", "batch set windows profiles: inserting windows profile label associations: labels:e"},
		{"inselect:k", "", "batch set windows profiles: build query to load existing profiles: inselect:k"},
		{"indelete:l", "", "batch set windows profiles: build statement to delete obsolete profiles: indelete:l"},
		{"inreselect:m", "", "batch set windows profiles: build query to load newly inserted profiles: inreselect:m"},
		{"", "select:f", "batch set apple profiles: load existing profiles: select:f"},
		{"", "insert:g", ": insert:g"},
		{"", "delete:h", "batch set apple profiles: delete obsolete profiles: delete:h"},
		{"", "reselect:i", "batch set apple profiles: load newly inserted profiles: reselect:i"},
		{"", "labels:j", "batch set apple profiles: inserting apple profile label associations: labels:j"},
		{"", "inselect:n", "batch set apple profiles: build query to load existing profiles: inselect:n"},
		{"", "indelete:o", "batch set apple profiles: build statement to delete obsolete profiles: indelete:o"},
		{"", "inreselect:p", "batch set apple profiles: build query to load newly inserted profiles: inreselect:p"},
	}
	for _, c := range cases {
		t.Run(c.windowsErr+" "+c.appleErr, func(t *testing.T) {
			t.Cleanup(func() {
				testBatchSetMDMAppleProfilesErr = ""
				testBatchSetMDMWindowsProfilesErr = ""
			})

			appleProfs := []*fleet.MDMAppleConfigProfile{
				configProfileForTest(t, "N1", "I1", "a"),
				configProfileForTest(t, "N2", "I2", "b"),
			}
			winProfs := []*fleet.MDMWindowsConfigProfile{
				windowsConfigProfileForTest(t, "W1", "l1"),
				windowsConfigProfileForTest(t, "W2", "l2"),
			}
			// set the initial profiles without error
			err := ds.BatchSetMDMProfiles(ctx, nil, appleProfs, winProfs)
			require.NoError(t, err)

			// now ensure all steps are required (add a profile, delete a profile, set labels)
			appleProfs = []*fleet.MDMAppleConfigProfile{
				configProfileForTest(t, "N1", "I1", "aa"),
				configProfileForTest(t, "N3", "I3", "c", lbl),
			}
			winProfs = []*fleet.MDMWindowsConfigProfile{
				windowsConfigProfileForTest(t, "W1", "l11"),
				windowsConfigProfileForTest(t, "W3", "l3", lbl),
			}
			// setup the expected errors
			testBatchSetMDMAppleProfilesErr = c.appleErr
			testBatchSetMDMWindowsProfilesErr = c.windowsErr

			err = ds.BatchSetMDMProfiles(ctx, nil, appleProfs, winProfs)
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func testMDMEULA(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	eula := &fleet.MDMEULA{
		Token: uuid.New().String(),
		Name:  "eula.pdf",
		Bytes: []byte("contents"),
	}

	err := ds.MDMInsertEULA(ctx, eula)
	require.NoError(t, err)

	var ae fleet.AlreadyExistsError
	err = ds.MDMInsertEULA(ctx, eula)
	require.ErrorAs(t, err, &ae)

	gotEULA, err := ds.MDMGetEULAMetadata(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gotEULA.CreatedAt)
	require.Equal(t, eula.Token, gotEULA.Token)
	require.Equal(t, eula.Name, gotEULA.Name)

	gotEULABytes, err := ds.MDMGetEULABytes(ctx, eula.Token)
	require.NoError(t, err)
	require.EqualValues(t, eula.Bytes, gotEULABytes.Bytes)
	require.Equal(t, eula.Name, gotEULABytes.Name)

	err = ds.MDMDeleteEULA(ctx, eula.Token)
	require.NoError(t, err)

	var nfe fleet.NotFoundError
	_, err = ds.MDMGetEULAMetadata(ctx)
	require.ErrorAs(t, err, &nfe)
	_, err = ds.MDMGetEULABytes(ctx, eula.Token)
	require.ErrorAs(t, err, &nfe)
	err = ds.MDMDeleteEULA(ctx, eula.Token)
	require.ErrorAs(t, err, &nfe)

	err = ds.MDMInsertEULA(ctx, eula)
	require.NoError(t, err)
}

func testSCEPRenewalHelpers(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	scepDepot, err := ds.NewSCEPDepot(testCertPEM, testKeyPEM)
	require.NoError(t, err)

	nanoStorage, err := ds.NewMDMAppleMDMStorage(testCertPEM, testKeyPEM)
	require.NoError(t, err)

	var i int
	setHost := func(notAfter time.Time) *fleet.Host {
		i++
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:          fmt.Sprintf("test-uuid-%d", i),
			Platform:      "darwin",
		})
		require.NoError(t, err)

		// create a cert + association
		serial, err := scepDepot.Serial()
		require.NoError(t, err)
		cert := &x509.Certificate{
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName: "FleetDM Identity",
			},
			NotAfter: notAfter,
			// use the host UUID, just to make sure they're
			// different from each other, we don't care about the
			// DER contents here
			Raw: []byte(h.UUID)}
		err = scepDepot.Put(cert.Subject.CommonName, cert)
		require.NoError(t, err)
		req := mdm.Request{
			EnrollID: &mdm.EnrollID{ID: h.UUID},
			Context:  ctx,
		}
		certHash := certauth.HashCert(cert)
		err = nanoStorage.AssociateCertHash(&req, certHash, notAfter)
		require.NoError(t, err)
		nanoEnroll(t, ds, h, false)
		return h
	}

	// certs expired at lest 1 year ago
	h1 := setHost(time.Now().AddDate(-1, -1, 0))
	h2 := setHost(time.Now().AddDate(-1, 0, 0))
	// cert that expires in 1 month
	h3 := setHost(time.Now().AddDate(0, 1, 0))
	// cert that expires in 1 year
	h4 := setHost(time.Now().AddDate(1, 0, 0))

	// list assocs that expire in the next 10 days
	assocs, err := ds.GetHostCertAssociationsToExpire(ctx, 10, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 2)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)

	// list certs that expire in the next 1000 days with limit = 1
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 1)
	require.NoError(t, err)
	require.Len(t, assocs, 1)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)

	// list certs that expire in the next 50 days
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 50, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 3)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	require.Equal(t, h3.UUID, assocs[2].HostUUID)

	// list certs that expire in the next 1000 days
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 4)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	require.Equal(t, h3.UUID, assocs[2].HostUUID)
	require.Equal(t, h4.UUID, assocs[3].HostUUID)

	checkSCEPRenew := func(assoc fleet.SCEPIdentityAssociation, want *string) {
		var got *string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &got, `SELECT renew_command_uuid FROM nano_cert_auth_associations WHERE id = ?`, assoc.HostUUID)
		})
		require.EqualValues(t, want, got)
	}

	// insert dummy nano commands
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, `
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('foo', 'foo', '<?xml'), ('bar', 'bar', '<?xml')
	`)
		return err
	})

	err = ds.SetCommandForPendingSCEPRenewal(ctx, []fleet.SCEPIdentityAssociation{}, "foo")
	checkSCEPRenew(assocs[0], nil)
	checkSCEPRenew(assocs[1], nil)
	checkSCEPRenew(assocs[2], nil)
	checkSCEPRenew(assocs[3], nil)
	require.NoError(t, err)

	err = ds.SetCommandForPendingSCEPRenewal(ctx, []fleet.SCEPIdentityAssociation{assocs[0]}, "foo")
	require.NoError(t, err)
	checkSCEPRenew(assocs[0], ptr.String("foo"))
	checkSCEPRenew(assocs[1], nil)
	checkSCEPRenew(assocs[2], nil)
	checkSCEPRenew(assocs[3], nil)

	err = ds.SetCommandForPendingSCEPRenewal(ctx, assocs, "bar")
	require.NoError(t, err)
	checkSCEPRenew(assocs[0], ptr.String("bar"))
	checkSCEPRenew(assocs[1], ptr.String("bar"))
	checkSCEPRenew(assocs[2], ptr.String("bar"))
	checkSCEPRenew(assocs[3], ptr.String("bar"))

	err = ds.SetCommandForPendingSCEPRenewal(ctx, []fleet.SCEPIdentityAssociation{{HostUUID: "foo", SHA256: "bar"}}, "bar")
	require.ErrorContains(t, err, "this function can only be used to update existing associations")
}

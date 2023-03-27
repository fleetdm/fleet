package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleConfigProfile(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestNewMDMAppleConfigProfileDuplicateName", testNewMDMAppleConfigProfileDuplicateName},
		{"TestNewMDMAppleConfigProfileDuplicateIdentifier", testNewMDMAppleConfigProfileDuplicateIdentifier},
		{"TestDeleteMDMAppleConfigProfile", testDeleteMDMAppleConfigProfile},
		{"TestDeleteMDMAppleConfigProfileByTeamAndIdentifier", testDeleteMDMAppleConfigProfileByTeamAndIdentifier},
		{"TestListMDMAppleConfigProfiles", testListMDMAppleConfigProfiles},
		{"TestHostDetailsMDMProfiles", testHostDetailsMDMProfiles},
		{"TestBatchSetMDMAppleProfiles", testBatchSetMDMAppleProfiles},
		{"TestMDMAppleProfileManagement", testMDMAppleProfileManagement},
		{"TestGetMDMAppleProfilesContents", testGetMDMAppleProfilesContents},
		{"TestMDMAppleHostsProfilesStatus", testMDMAppleHostsProfilesStatus},
		{"TestMDMAppleInsertIdPAccount", testMDMAppleInsertIdPAccount},
		{"TestIgnoreMDMClientError", testIgnoreMDMClientError},
		{"TestDeleteMDMAppleProfilesForHost", testDeleteMDMAppleProfilesForHost},
		{"TestBulkSetPendingMDMAppleHostProfiles", testBulkSetPendingMDMAppleHostProfiles},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testNewMDMAppleConfigProfileDuplicateName(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)

	// cannot create another profile with the same name if it is on the same team
	duplicateCP := fleet.MDMAppleConfigProfile{
		Name:         initialCP.Name,
		Identifier:   "DifferentIdentifierDoesNotMatter",
		TeamID:       initialCP.TeamID,
		Mobileconfig: initialCP.Mobileconfig,
	}
	_, err := ds.NewMDMAppleConfigProfile(ctx, duplicateCP)
	expectedErr := &existsError{ResourceType: "MDMAppleConfigProfile.PayloadDisplayName", Identifier: initialCP.Name, TeamID: initialCP.TeamID}
	require.ErrorContains(t, err, expectedErr.Error())

	// can create another profile with the same name if it is on a different team
	duplicateCP.TeamID = ptr.Uint(*duplicateCP.TeamID + 1)
	newCP, err := ds.NewMDMAppleConfigProfile(ctx, duplicateCP)
	require.NoError(t, err)
	checkConfigProfile(t, duplicateCP, *newCP)
	storedCP, err := ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)
}

func testNewMDMAppleConfigProfileDuplicateIdentifier(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)

	// cannot create another profile with the same identifier if it is on the same team
	duplicateCP := fleet.MDMAppleConfigProfile{
		Name:         "DifferentNameDoesNotMatter",
		Identifier:   initialCP.Identifier,
		TeamID:       initialCP.TeamID,
		Mobileconfig: initialCP.Mobileconfig,
	}
	_, err := ds.NewMDMAppleConfigProfile(ctx, duplicateCP)
	expectedErr := &existsError{ResourceType: "MDMAppleConfigProfile.PayloadIdentifier", Identifier: initialCP.Identifier, TeamID: initialCP.TeamID}
	require.ErrorContains(t, err, expectedErr.Error())

	// can create another profile with the same name if it is on a different team
	duplicateCP.TeamID = ptr.Uint(*duplicateCP.TeamID + 1)
	newCP, err := ds.NewMDMAppleConfigProfile(ctx, duplicateCP)
	require.NoError(t, err)
	checkConfigProfile(t, duplicateCP, *newCP)
	storedCP, err := ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)
}

func testListMDMAppleConfigProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	generateCP := func(name string, identifier string, teamID uint) *fleet.MDMAppleConfigProfile {
		mc := mobileconfig.Mobileconfig([]byte(name + identifier))
		return &fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   identifier,
			TeamID:       &teamID,
			Mobileconfig: mc,
		}
	}

	expectedTeam0 := []*fleet.MDMAppleConfigProfile{}
	expectedTeam1 := []*fleet.MDMAppleConfigProfile{}

	// add profile with team id zero (i.e. profile is not associated with any team)
	cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("name0", "identifier0", 0))
	require.NoError(t, err)
	expectedTeam0 = append(expectedTeam0, cp)
	cps, err := ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, cps, 1)
	checkConfigProfile(t, *expectedTeam0[0], *cps[0])

	// add fleet-managed profiles for the team and globally
	for idf := range mobileconfig.FleetPayloadIdentifiers() {
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, 1))
		require.NoError(t, err)
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, 0))
		require.NoError(t, err)
	}

	// add profile with team id 1
	cp, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name1", "identifier1", 1))
	require.NoError(t, err)
	expectedTeam1 = append(expectedTeam1, cp)
	// list profiles for team id 1
	cps, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(1))
	require.NoError(t, err)
	require.Len(t, cps, 1)
	checkConfigProfile(t, *expectedTeam1[0], *cps[0])

	// add another profile with team id 1
	cp, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("another_name1", "another_identifier1", 1))
	require.NoError(t, err)
	expectedTeam1 = append(expectedTeam1, cp)
	// list profiles for team id 1
	cps, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(1))
	require.NoError(t, err)
	require.Len(t, cps, 2)
	for _, cp := range cps {
		switch cp.Name {
		case "name1":
			checkConfigProfile(t, *expectedTeam1[0], *cp)
		case "another_name1":
			checkConfigProfile(t, *expectedTeam1[1], *cp)
		default:
			t.FailNow()
		}
	}

	// try to list profiles for non-existent team id
	cps, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(42))
	require.NoError(t, err)
	require.Len(t, cps, 0)
}

func testDeleteMDMAppleConfigProfile(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)

	err := ds.DeleteMDMAppleConfigProfile(ctx, initialCP.ProfileID)
	require.NoError(t, err)

	_, err = ds.GetMDMAppleConfigProfile(ctx, initialCP.ProfileID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.DeleteMDMAppleConfigProfile(ctx, initialCP.ProfileID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testDeleteMDMAppleConfigProfileByTeamAndIdentifier(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)

	err := ds.DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx, initialCP.TeamID, initialCP.Identifier)
	require.NoError(t, err)

	_, err = ds.GetMDMAppleConfigProfile(ctx, initialCP.ProfileID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx, initialCP.TeamID, initialCP.Identifier)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func storeDummyConfigProfileForTest(t *testing.T, ds *Datastore) *fleet.MDMAppleConfigProfile {
	dummyMC := mobileconfig.Mobileconfig([]byte("DummyTestMobileconfigBytes"))
	dummyCP := fleet.MDMAppleConfigProfile{
		Name:         "DummyTestName",
		Identifier:   "DummyTestIdentifier",
		Mobileconfig: dummyMC,
		TeamID:       nil,
	}

	ctx := context.Background()

	newCP, err := ds.NewMDMAppleConfigProfile(ctx, dummyCP)
	require.NoError(t, err)
	checkConfigProfile(t, dummyCP, *newCP)
	storedCP, err := ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)

	return storedCP
}

func checkConfigProfile(t *testing.T, expected fleet.MDMAppleConfigProfile, actual fleet.MDMAppleConfigProfile) {
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Identifier, actual.Identifier)
	require.Equal(t, expected.Mobileconfig, actual.Mobileconfig)
}

func testHostDetailsMDMProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	p0, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{Name: "Name0", Identifier: "Identifier0", Mobileconfig: []byte("profile0-bytes")})
	require.NoError(t, err)

	p1, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{Name: "Name1", Identifier: "Identifier1", Mobileconfig: []byte("profile1-bytes")})
	require.NoError(t, err)

	p2, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{Name: "Name2", Identifier: "Identifier2", Mobileconfig: []byte("profile2-bytes")})
	require.NoError(t, err)

	profiles, err := ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, profiles, 3)

	h0, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host0-osquery-id"),
		NodeKey:         ptr.String("host0-node-key"),
		UUID:            "host0-test-mdm-profiles",
		Hostname:        "hostname0",
	})
	require.NoError(t, err)

	gotHost, err := ds.Host(ctx, h0.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles)
	gotProfs, err := ds.GetHostMDMProfiles(ctx, h0.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)

	h1, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host1-osquery-id"),
		NodeKey:         ptr.String("host1-node-key"),
		UUID:            "host1-test-mdm-profiles",
		Hostname:        "hostname1",
	})
	require.NoError(t, err)

	gotHost, err = ds.Host(ctx, h1.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles)
	gotProfs, err = ds.GetHostMDMProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)

	expectedProfiles0 := map[uint]fleet.HostMDMAppleProfile{
		p0.ProfileID: {HostUUID: h0.UUID, Name: p0.Name, ProfileID: p0.ProfileID, CommandUUID: "cmd0-uuid", Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: ""},
		p1.ProfileID: {HostUUID: h0.UUID, Name: p1.Name, ProfileID: p1.ProfileID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMAppleDeliveryApplied, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: ""},
		p2.ProfileID: {HostUUID: h0.UUID, Name: p2.Name, ProfileID: p2.ProfileID, CommandUUID: "cmd2-uuid", Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove, Detail: "Error removing profile"},
	}

	expectedProfiles1 := map[uint]fleet.HostMDMAppleProfile{
		p0.ProfileID: {HostUUID: h1.UUID, Name: p0.Name, ProfileID: p0.ProfileID, CommandUUID: "cmd0-uuid", Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: "Error installing profile"},
		p1.ProfileID: {HostUUID: h1.UUID, Name: p1.Name, ProfileID: p1.ProfileID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMAppleDeliveryApplied, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: ""},
		p2.ProfileID: {HostUUID: h1.UUID, Name: p2.Name, ProfileID: p2.ProfileID, CommandUUID: "cmd2-uuid", Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove, Detail: "Error removing profile"},
	}

	var args []interface{}
	for _, p := range expectedProfiles0 {
		args = append(args, p.HostUUID, p.ProfileID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}
	for _, p := range expectedProfiles1 {
		args = append(args, p.HostUUID, p.ProfileID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
	INSERT INTO host_mdm_apple_profiles (
		host_uuid, profile_id, command_uuid, status, operation_type, detail, profile_name)
	VALUES (?,?,?,?,?,?,?),(?,?,?,?,?,?,?),(?,?,?,?,?,?,?),(?,?,?,?,?,?,?),(?,?,?,?,?,?,?),(?,?,?,?,?,?,?)
		`, args...,
		)
		if err != nil {
			return err
		}
		return nil
	})

	gotHost, err = ds.Host(ctx, h0.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles) // ds.Host never returns MDM profiles

	gotProfs, err = ds.GetHostMDMProfiles(ctx, h0.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 3)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles0[gp.ProfileID]
		require.True(t, ok)
		require.Equal(t, ep.Name, gp.Name)
		require.Equal(t, *ep.Status, *gp.Status)
		require.Equal(t, ep.OperationType, gp.OperationType)
		require.Equal(t, ep.Detail, gp.Detail)
	}

	gotHost, err = ds.Host(ctx, h1.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles) // ds.Host never returns MDM profiles

	gotProfs, err = ds.GetHostMDMProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 3)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles1[gp.ProfileID]
		require.True(t, ok)
		require.Equal(t, ep.Name, gp.Name)
		require.Equal(t, *ep.Status, *gp.Status)
		require.Equal(t, ep.OperationType, gp.OperationType)
		require.Equal(t, ep.Detail, gp.Detail)
	}

	// mark h1's install+failed profile as install+pending
	h1InstallFailed := expectedProfiles1[p0.ProfileID]
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      h1InstallFailed.HostUUID,
		CommandUUID:   h1InstallFailed.CommandUUID,
		ProfileID:     h1InstallFailed.ProfileID,
		Name:          h1InstallFailed.Name,
		Status:        &fleet.MDMAppleDeliveryPending,
		OperationType: fleet.MDMAppleOperationTypeInstall,
		Detail:        "",
	})
	require.NoError(t, err)

	// mark h1's remove+failed profile as remove+applied, deletes the host profile row
	h1RemoveFailed := expectedProfiles1[p2.ProfileID]
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      h1RemoveFailed.HostUUID,
		CommandUUID:   h1RemoveFailed.CommandUUID,
		ProfileID:     h1RemoveFailed.ProfileID,
		Name:          h1RemoveFailed.Name,
		Status:        &fleet.MDMAppleDeliveryApplied,
		OperationType: fleet.MDMAppleOperationTypeRemove,
		Detail:        "",
	})
	require.NoError(t, err)

	gotProfs, err = ds.GetHostMDMProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 2) // remove+applied is not there anymore

	h1InstallPending := h1InstallFailed
	h1InstallPending.Status = &fleet.MDMAppleDeliveryPending
	h1InstallPending.Detail = ""
	expectedProfiles1[p0.ProfileID] = h1InstallPending
	delete(expectedProfiles1, p2.ProfileID)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles1[gp.ProfileID]
		require.True(t, ok)
		require.Equal(t, ep.Name, gp.Name)
		require.Equal(t, *ep.Status, *gp.Status)
		require.Equal(t, ep.OperationType, gp.OperationType)
		require.Equal(t, ep.Detail, gp.Detail)
	}
}

func TestIngestMDMAppleDevicesFromDEPSync(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()
	createBuiltinLabels(t, ds)

	for i := 0; i < 10; i++ {
		_, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        fmt.Sprintf("hostname_%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("osquery-host-id_%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("node-key_%d", i)),
			UUID:            fmt.Sprintf("uuid_%d", i),
			HardwareSerial:  fmt.Sprintf("serial_%d", i),
		})
		require.NoError(t, err)
	}

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 10)
	wantSerials := []string{}
	for _, h := range hosts {
		wantSerials = append(wantSerials, h.HardwareSerial)
	}

	// mock results incoming from depsync.Syncer
	depDevices := []godep.Device{
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // ingested; new serial, macOS, "added" op type
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // not ingested; duplicate serial
		{SerialNumber: hosts[0].HardwareSerial, Model: "MacBook Pro", OS: "OSX", OpType: "added"}, // not ingested; existing serial
		{SerialNumber: "ijk", Model: "MacBook Pro", OS: "", OpType: "added"},                      // ingested; empty OS
		{SerialNumber: "tuv", Model: "MacBook Pro", OS: "OSX", OpType: "modified"},                // not ingested; op type "modified"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "updated"},                 // not ingested; op type "updated"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "deleted"},                 // not ingested; op type "deleted"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                   // ingested; new serial, macOS, "added" op type
	}
	wantSerials = append(wantSerials, "abc", "xyz", "ijk")

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(3), n) // 3 new hosts ("abc", "xyz", "ijk")

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, len(wantSerials))
	gotSerials := []string{}
	for _, h := range hosts {
		gotSerials = append(gotSerials, h.HardwareSerial)
		if hs := h.HardwareSerial; hs == "abc" || hs == "xyz" {
			checkMDMHostRelatedTables(t, ds, h.ID, hs, "MacBook Pro")
		}
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
}

func TestDEPSyncTeamAssignment(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()
	createBuiltinLabels(t, ds)

	depDevices := []godep.Device{
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
		{SerialNumber: "def", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 2)
	for _, h := range hosts {
		require.Nil(t, h.TeamID)
	}

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)

	// assign the team as the default team for DEP devices
	ac, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.MDM.AppleBMDefaultTeam = team.Name
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	depDevices = []godep.Device{
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}

	n, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 3)
	for _, h := range hosts {
		if h.HardwareSerial == "xyz" {
			require.EqualValues(t, team.ID, *h.TeamID)
		} else {
			require.Nil(t, h.TeamID)
		}
	}

	ac.MDM.AppleBMDefaultTeam = "non-existent"
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	depDevices = []godep.Device{
		{SerialNumber: "jqk", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}

	n, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.EqualValues(t, n, 1)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 4)
	for _, h := range hosts {
		if h.HardwareSerial == "jqk" {
			require.Nil(t, h.TeamID)
		}
	}
}

func TestMDMEnrollment(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestHostAlreadyExistsInFleet", testIngestMDMAppleHostAlreadyExistsInFleet},
		{"TestIngestAfterDEPSync", testIngestMDMAppleIngestAfterDEPSync},
		{"TestBeforeDEPSync", testIngestMDMAppleCheckinBeforeDEPSync},
		{"TestMultipleIngest", testIngestMDMAppleCheckinMultipleIngest},
		{"TestCheckOut", testUpdateHostTablesOnMDMUnenroll},
		{"TestNonDarwinHostAlreadyExistsInFleet", testIngestMDMNonDarwinHostAlreadyExistsInFleet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			createBuiltinLabels(t, ds)

			c.fn(t, ds)
		})
	}
}

func testIngestMDMAppleHostAlreadyExistsInFleet(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-name",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1337"),
		NodeKey:         ptr.String("1337"),
		UUID:            testUUID,
		HardwareSerial:  testSerial,
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet)
	require.NoError(t, err)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	err = ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func testIngestMDMNonDarwinHostAlreadyExistsInFleet(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"

	// this cannot happen for real, but it tests the host-matching logic in that
	// even if the host does match on serial number, it is not used as matching
	// host because it is not a macOS (darwin) platform host.
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-name",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("1337"),
		NodeKey:         ptr.String("1337"),
		UUID:            testUUID,
		HardwareSerial:  testSerial,
		Platform:        "linux",
	})
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "https://fleetdm.com", true, "Fleet MDM")
	require.NoError(t, err)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	err = ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 2)
	// a new host was created with the provided uuid/serial and darwin as platform
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
	require.Equal(t, testSerial, hosts[1].HardwareSerial)
	require.Equal(t, testUUID, hosts[1].UUID)
	id0, id1 := hosts[0].ID, hosts[1].ID
	platform0, platform1 := hosts[0].Platform, hosts[1].Platform
	require.NotEqual(t, id0, id1)
	require.NotEqual(t, platform0, platform1)
	require.ElementsMatch(t, []string{"darwin", "linux"}, []string{platform0, platform1})
}

func testIngestMDMAppleIngestAfterDEPSync(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"
	testModel := "MacBook Pro"

	// simulate a host that is first ingested via DEP (e.g., the device was added via Apple Business Manager)
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	// hosts that are first ingested via DEP will have a serial number but not a UUID because UUID
	// is not available from the DEP sync endpoint
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, "", hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)

	// now simulate the initial MDM checkin by that same host
	err = ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)
}

func testIngestMDMAppleCheckinBeforeDEPSync(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"
	testModel := "MacBook Pro"

	// ingest host on initial mdm checkin
	err := ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
		Model:        testModel,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)

	// no effect if same host appears in DEP sync
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), n)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)
}

func testIngestMDMAppleCheckinMultipleIngest(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"

	err := ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	// duplicate Authenticate request has no effect
	err = ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
}

func testUpdateHostTablesOnMDMUnenroll(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	testSerial := "test-serial"
	testUUID := "test-uuid"
	err := ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{
		UDID:         testUUID,
		SerialNumber: testSerial,
	})
	require.NoError(t, err)

	// check that an entry in host_mdm exists
	var count int
	err = sqlx.GetContext(context.Background(), ds.reader, &count, `SELECT COUNT(*) FROM host_mdm WHERE host_id = (SELECT id FROM hosts WHERE uuid = ?)`, testUUID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = ds.UpdateHostTablesOnMDMUnenroll(ctx, testUUID)
	require.NoError(t, err)

	err = sqlx.GetContext(context.Background(), ds.reader, &count, `SELECT COUNT(*) FROM host_mdm WHERE host_id = ?`, testUUID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testBatchSetMDMAppleProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	applyAndExpect := func(newSet []*fleet.MDMAppleConfigProfile, tmID *uint, want []*fleet.MDMAppleConfigProfile) map[string]uint {
		err := ds.BatchSetMDMAppleProfiles(ctx, tmID, newSet)
		require.NoError(t, err)

		if tmID == nil {
			tmID = ptr.Uint(0)
		}
		// don't use ds.ListMDMAppleConfigProfiles as it leaves out
		// fleet-managed profiles.
		var got []*fleet.MDMAppleConfigProfile
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &got, `SELECT * FROM mdm_apple_configuration_profiles WHERE team_id = ?`, tmID)
		})

		// compare only the fields we care about, and build the resulting map of
		// profile identifier as key to profile ID as value
		m := make(map[string]uint)
		for _, gotp := range got {
			m[gotp.Identifier] = gotp.ProfileID
			if gotp.TeamID != nil && *gotp.TeamID == 0 {
				gotp.TeamID = nil
			}
			gotp.ProfileID = 0
			gotp.CreatedAt = time.Time{}
			gotp.UpdatedAt = time.Time{}
		}
		// order is not guaranteed
		require.ElementsMatch(t, want, got)

		return m
	}

	withTeamID := func(p *fleet.MDMAppleConfigProfile, tmID uint) *fleet.MDMAppleConfigProfile {
		p.TeamID = &tmID
		return p
	}

	// apply empty set for no-team
	applyAndExpect(nil, nil, nil)

	// apply single profile set for tm1
	mTm1 := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "a"),
	}, ptr.Uint(1), []*fleet.MDMAppleConfigProfile{
		withTeamID(configProfileForTest(t, "N1", "I1", "a"), 1),
	})

	// apply single profile set for no-team
	mNoTm := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	}, nil, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	})

	// apply new profile set for tm1
	mTm1b := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "a"), // unchanged
		configProfileForTest(t, "N2", "I2", "b"),
	}, ptr.Uint(1), []*fleet.MDMAppleConfigProfile{
		withTeamID(configProfileForTest(t, "N1", "I1", "a"), 1),
		withTeamID(configProfileForTest(t, "N2", "I2", "b"), 1),
	})
	// identifier for N1-I1 is unchanged
	require.Equal(t, mTm1["I1"], mTm1b["I1"])

	// apply edited (by name only) profile set for no-team
	mNoTmb := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N2", "I1", "b"),
	}, nil, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N2", "I1", "b"),
	})
	require.NotEqual(t, mNoTm["I1"], mNoTmb["I1"])

	// apply edited profile (by content only), unchanged profile and new profile
	// for tm1
	mTm1c := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "z"), // content updated
		configProfileForTest(t, "N2", "I2", "b"), // unchanged
		configProfileForTest(t, "N3", "I3", "c"), // new
	}, ptr.Uint(1), []*fleet.MDMAppleConfigProfile{
		withTeamID(configProfileForTest(t, "N1", "I1", "z"), 1),
		withTeamID(configProfileForTest(t, "N2", "I2", "b"), 1),
		withTeamID(configProfileForTest(t, "N3", "I3", "c"), 1),
	})
	// identifier for N1-I1 is changed
	require.NotEqual(t, mTm1b["I1"], mTm1c["I1"])
	// identifier for N2-I2 is unchanged
	require.Equal(t, mTm1b["I2"], mTm1c["I2"])

	// apply only new profiles to no-team
	applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N4", "I4", "d"),
		configProfileForTest(t, "N5", "I5", "e"),
	}, nil, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N4", "I4", "d"),
		configProfileForTest(t, "N5", "I5", "e"),
	})

	// clear profiles for tm1
	applyAndExpect(nil, ptr.Uint(1), nil)

	// simulate profiles being added by fleet
	fleetProfiles := []*fleet.MDMAppleConfigProfile{}
	expectFleetProfiles := []*fleet.MDMAppleConfigProfile{}
	for fp := range mobileconfig.FleetPayloadIdentifiers() {
		fleetProfiles = append(fleetProfiles, configProfileForTest(t, fp, fp, fp))
		expectFleetProfiles = append(expectFleetProfiles, withTeamID(configProfileForTest(t, fp, fp, fp), 1))
	}

	applyAndExpect(fleetProfiles, nil, fleetProfiles)
	applyAndExpect(fleetProfiles, ptr.Uint(1), expectFleetProfiles)

	// add no-team profiles
	applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	}, nil, append([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	}, fleetProfiles...))

	// add team profiles
	applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "a"),
		configProfileForTest(t, "N2", "I2", "b"),
	}, ptr.Uint(1), append([]*fleet.MDMAppleConfigProfile{
		withTeamID(configProfileForTest(t, "N1", "I1", "a"), 1),
		withTeamID(configProfileForTest(t, "N2", "I2", "b"), 1),
	}, expectFleetProfiles...))

	// cleaning profiles still leaves the profile managed by Fleet
	applyAndExpect(nil, nil, fleetProfiles)
	applyAndExpect(nil, ptr.Uint(1), expectFleetProfiles)
}

func configProfileForTest(t *testing.T, name, identifier, uuid string) *fleet.MDMAppleConfigProfile {
	prof := []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, name, identifier, uuid))
	cp, err := fleet.NewMDMAppleConfigProfile(prof, nil)
	require.NoError(t, err)
	return cp
}

func testMDMAppleProfileManagement(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	globalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "z"),
		configProfileForTest(t, "N2", "I2", "b"),
		configProfileForTest(t, "N3", "I3", "c"),
	}
	err := ds.BatchSetMDMAppleProfiles(ctx, nil, globalProfiles)
	require.NoError(t, err)

	globalPfs, err := ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, globalPfs, len(globalProfiles))

	_, err = ds.writer.Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('command-uuid', 'foo', '<?xml')
	`)
	require.NoError(t, err)

	// if there are no hosts, then no profiles need to be applied
	profiles, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profiles)

	host1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	// add a user enrollment for this device, nothing else should be modified
	nanoEnroll(t, ds, host1, true)

	// non-macOS hosts shouldn't modify any of the results below
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-windows-host",
		OsqueryHostID: ptr.String("4824"),
		NodeKey:       ptr.String("4824"),
		UUID:          "test-windows-host",
		TeamID:        nil,
		Platform:      "windows",
	})
	require.NoError(t, err)

	// a macOS host that's not MDM enrolled into Fleet shouldn't
	// modify any of the results below
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-non-mdm-host",
		OsqueryHostID: ptr.String("4825"),
		NodeKey:       ptr.String("4825"),
		UUID:          "test-non-mdm-host",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// global profiles to install on the newly added host
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// add another host, it belongs to a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test team"})
	require.NoError(t, err)
	host2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host2-name",
		OsqueryHostID: ptr.String("1338"),
		NodeKey:       ptr.String("1338"),
		UUID:          "test-uuid-2",
		TeamID:        &team.ID,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host2, false)

	// still the same profiles to assign as there are no profiles for team 1
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// assign profiles to team 1
	err = ds.BatchSetMDMAppleProfiles(ctx, &team.ID, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N4", "I4", "x"),
		configProfileForTest(t, "N5", "I5", "y"),
	})
	require.NoError(t, err)

	globalPfs, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, globalPfs, 3)
	teamPfs, err := ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(1))
	require.NoError(t, err)
	require.Len(t, teamPfs, 2)

	// new profiles, this time for the new host belonging to team 1
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2"},
	}, profiles)

	// add another global host
	host3, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host3-name",
		OsqueryHostID: ptr.String("1339"),
		NodeKey:       ptr.String("1339"),
		UUID:          "test-uuid-3",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host3, false)

	// more profiles, this time for both global hosts and the team
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2"},
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-3"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-3"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-3"},
	}, profiles)

	// cron runs and updates the status
	err = ds.BulkUpsertMDMAppleHostProfiles(
		ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileID:         globalPfs[0].ProfileID,
				ProfileIdentifier: globalPfs[0].Identifier,
				ProfileName:       globalPfs[0].Name,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[0].ProfileID,
				ProfileIdentifier: globalPfs[0].Identifier,
				ProfileName:       globalPfs[0].Name,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[0].ProfileID,
				ProfileIdentifier: teamPfs[0].Identifier,
				ProfileName:       teamPfs[0].Name,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[1].ProfileID,
				ProfileIdentifier: teamPfs[1].Identifier,
				ProfileName:       teamPfs[1].Name,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
		},
	)
	require.NoError(t, err)

	// no profiles left to install
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profiles)

	// no profiles to remove yet
	toRemove, err := ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// add host1 to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host1.ID})
	require.NoError(t, err)

	// profiles to be added for host1 are now related to the team
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// profiles to be removed includes host1's old profiles
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
	}, toRemove)
}

// checkMDMHostRelatedTables checks that rows are inserted for new MDM hosts in
// each of host_display_names, host_seen_times, and label_membership. Note that
// related tables records for pre-existing hosts are created outside of the MDM
// enrollment flows so they are not checked in some tests above (e.g.,
// testIngestMDMAppleHostAlreadyExistsInFleet)
func checkMDMHostRelatedTables(t *testing.T, ds *Datastore, hostID uint, expectedSerial string, expectedModel string) {
	var displayName string
	err := sqlx.GetContext(context.Background(), ds.reader, &displayName, `SELECT display_name FROM host_display_names WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s (%s)", expectedModel, expectedSerial), displayName)

	var labelsOK []bool
	err = sqlx.SelectContext(context.Background(), ds.reader, &labelsOK, `SELECT 1 FROM label_membership WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Len(t, labelsOK, 2)
	require.True(t, labelsOK[0])
	require.True(t, labelsOK[1])

	appCfg, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	var hmdm fleet.HostMDM
	err = sqlx.GetContext(context.Background(), ds.reader, &hmdm, `SELECT host_id, server_url, mdm_id FROM host_mdm WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Equal(t, hostID, hmdm.HostID)
	serverURL, err := apple_mdm.ResolveAppleMDMURL(appCfg.ServerSettings.ServerURL)
	require.NoError(t, err)
	require.Equal(t, serverURL, hmdm.ServerURL)
	require.NotEmpty(t, hmdm.MDMID)

	var mdmSolution fleet.MDMSolution
	err = sqlx.GetContext(context.Background(), ds.reader, &mdmSolution, `SELECT name, server_url FROM mobile_device_management_solutions WHERE id = ?`, hmdm.MDMID)
	require.NoError(t, err)
	require.Equal(t, fleet.WellKnownMDMFleet, mdmSolution.Name)
	require.Equal(t, serverURL, mdmSolution.ServerURL)
}

func testGetMDMAppleProfilesContents(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	profiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "z"),
		configProfileForTest(t, "N2", "I2", "b"),
		configProfileForTest(t, "N3", "I3", "c"),
	}
	err := ds.BatchSetMDMAppleProfiles(ctx, nil, profiles)
	require.NoError(t, err)

	profiles, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)

	cases := []struct {
		ids  []uint
		want map[uint]mobileconfig.Mobileconfig
	}{
		{[]uint{}, nil},
		{nil, nil},
		{[]uint{profiles[0].ProfileID}, map[uint]mobileconfig.Mobileconfig{profiles[0].ProfileID: profiles[0].Mobileconfig}},
		{
			[]uint{profiles[0].ProfileID, profiles[1].ProfileID, profiles[2].ProfileID},
			map[uint]mobileconfig.Mobileconfig{
				profiles[0].ProfileID: profiles[0].Mobileconfig,
				profiles[1].ProfileID: profiles[1].Mobileconfig,
				profiles[2].ProfileID: profiles[2].Mobileconfig,
			},
		},
	}

	for _, c := range cases {
		out, err := ds.GetMDMAppleProfilesContents(ctx, c.ids)
		require.NoError(t, err)
		require.Equal(t, c.want, out)
	}
}

// createBuiltinLabels creates entries for "All Hosts" and "macOS" labels, which are assumed to be
// extant for MDM flows
func createBuiltinLabels(t *testing.T, ds *Datastore) {
	_, err := ds.writer.Exec(`
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		"All Hosts",
		"",
		"",
		"",
		fleet.LabelTypeBuiltIn,
		"macOS",
		"",
		"",
		"",
		fleet.LabelTypeBuiltIn,
	)
	require.NoError(t, err)
}

func nanoEnroll(t *testing.T, ds *Datastore, host *fleet.Host, withUser bool) {
	_, err := ds.writer.Exec(`INSERT INTO nano_devices (id, authenticate) VALUES (?, 'test')`, host.UUID)
	require.NoError(t, err)

	_, err = ds.writer.Exec(`
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex)
VALUES
	(?, ?, ?, ?, ?, ?, ?)`,
		host.UUID,
		host.UUID,
		nil,
		"Device",
		host.UUID+".topic",
		host.UUID+".magic",
		host.UUID,
	)
	require.NoError(t, err)

	if withUser {
		_, err = ds.writer.Exec(`
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex)
VALUES
	(?, ?, ?, ?, ?, ?, ?)`,
			host.UUID+":Device",
			host.UUID,
			nil,
			"User",
			host.UUID+".topic",
			host.UUID+".magic",
			host.UUID,
		)
		require.NoError(t, err)
	}
}

func testMDMAppleHostsProfilesStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	generateCP := func(name string, identifier string, teamID uint) *fleet.MDMAppleConfigProfile {
		mc := mobileconfig.Mobileconfig([]byte(name + identifier))
		return &fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   identifier,
			TeamID:       &teamID,
			Mobileconfig: mc,
		}
	}

	upsertHostCPs := func(hosts []*fleet.Host, profiles []*fleet.MDMAppleConfigProfile, opType fleet.MDMAppleOperationType, status *fleet.MDMAppleDeliveryStatus) {
		upserts := []*fleet.MDMAppleBulkUpsertHostProfilePayload{}
		for _, h := range hosts {
			for _, cp := range profiles {
				payload := fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileID:         cp.ProfileID,
					ProfileIdentifier: cp.Identifier,
					ProfileName:       cp.Name,
					HostUUID:          h.UUID,
					CommandUUID:       "",
					OperationType:     opType,
					Status:            status,
				}
				upserts = append(upserts, &payload)
			}
		}
		err := ds.BulkUpsertMDMAppleHostProfiles(ctx, upserts)
		require.NoError(t, err)
	}

	checkListHosts := func(status fleet.MacOSSettingsStatus, teamID *uint, expected []*fleet.Host) bool {
		expectedIDs := []uint{}
		for _, h := range expected {
			expectedIDs = append(expectedIDs, h.ID)
		}

		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}}, fleet.HostListOptions{MacOSSettingsFilter: status, TeamFilter: teamID})
		gotIDs := []uint{}
		for _, h := range gotHosts {
			gotIDs = append(gotIDs, h.ID)
		}

		return assert.NoError(t, err) && assert.Len(t, gotHosts, len(expected)) && assert.ElementsMatch(t, expectedIDs, gotIDs)
	}

	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now())
		hosts = append(hosts, h)
	}

	// create somes config profiles for no team
	var noTeamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 10; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), 0))
		require.NoError(t, err)
		noTeamCPs = append(noTeamCPs, cp)
	}

	// all hosts nil status (pending install) for all profiles
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, nil)
	res, err := ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// all hosts pending install of all profiles
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryPending)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0] and hosts[1] failed one profile
	upsertHostCPs(hosts[0:2], noTeamCPs[0:1], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed)
	// hosts[0] and hosts[1] have one profile pending as nil
	upsertHostCPs(hosts[0:2], noTeamCPs[3:4], fleet.MDMAppleOperationTypeInstall, nil)
	// hosts[0] also failed another profile
	upsertHostCPs(hosts[0:1], noTeamCPs[1:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed)
	// hosts[4] has all profiles reported as nil (pending)
	upsertHostCPs(hosts[4:5], noTeamCPs, fleet.MDMAppleOperationTypeInstall, nil)
	// hosts[5] has one profile reported as nil (pending)
	upsertHostCPs(hosts[5:6], noTeamCPs[0:1], fleet.MDMAppleOperationTypeInstall, nil)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // two hosts are failing at least one profile (hosts[0] and hosts[1])
	require.Equal(t, uint(2), res.Failed)             // only count one failure per host (hosts[0] failed two profiles but only counts once)
	require.Equal(t, uint(0), res.Latest)
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0:3] applied a third profile
	upsertHostCPs(hosts[0:3], noTeamCPs[2:3], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // no change
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// hosts[6] deletes all its profiles
	require.NoError(t, ds.DeleteMDMAppleProfilesForHost(ctx, hosts[6].UUID))
	pendingHosts := append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] applied all profiles but one is with status nil (pending)
	upsertHostCPs(hosts[9:10], noTeamCPs[:9], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied)
	upsertHostCPs(hosts[9:10], noTeamCPs[9:10], fleet.MDMAppleOperationTypeInstall, nil)
	pendingHosts = append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore, hosts[9] still pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] applied all profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract hosts[6 and 9] from pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(1), res.Latest)             // add one host that has applied all profiles
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), hosts[9:10]))

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "rocket"})
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending) // no profiles yet
	require.Equal(t, uint(0), res.Failed)  // no profiles yet
	require.Equal(t, uint(0), res.Latest)  // no profiles yet
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, &tm.ID, []*fleet.Host{}))

	// transfer hosts[9] to new team
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{hosts[9].ID})
	require.NoError(t, err)
	// remove all no team profiles from hosts[9]
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryPending)

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // hosts[9] is still not pending, transferred to team
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // hosts[9] was transferred so this is now zero
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary for new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, &tm.ID, []*fleet.Host{}))

	// create somes config profiles for the new team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 10; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), tm.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}

	// install all team profiles on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is still pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, &tm.ID, []*fleet.Host{}))

	// hosts[9] successfully removed old profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Latest) // hosts[9] is all good
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, &tm.ID, hosts[9:10]))

	// confirm no changes in summary for profiles with no team
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, ptr.Uint(0)) // team id zero represents no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract two failed hosts, one without profiles and hosts[9] transferred
	require.Equal(t, uint(2), res.Failed)             // two failed hosts
	require.Equal(t, uint(0), res.Latest)             // hosts[9] transferred to new team so is not counted under no team
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusFailing, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsStatusLatest, ptr.Uint(0), []*fleet.Host{}))
}

func testMDMAppleInsertIdPAccount(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	acc := &fleet.MDMIdPAccount{
		UUID:     "ABC-DEF",
		Username: "email@example.com",
		SaltedSHA512PBKDF2Dictionary: fleet.SaltedSHA512PBKDF2Dictionary{
			Iterations: 50000,
			Salt:       []byte("salt"),
			Entropy:    []byte("entropy"),
		},
	}

	err := ds.InsertMDMIdPAccount(ctx, acc)
	require.NoError(t, err)

	// try to instert the same account
	err = ds.InsertMDMIdPAccount(ctx, acc)
	require.NoError(t, err)

	// try to insert an empty account
	err = ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{})
	require.Error(t, err)

	// duplicated values get updated
	acc.SaltedSHA512PBKDF2Dictionary.Iterations = 3000
	acc.SaltedSHA512PBKDF2Dictionary.Salt = []byte("tlas")
	acc.SaltedSHA512PBKDF2Dictionary.Entropy = []byte("yportne")
	err = ds.InsertMDMIdPAccount(ctx, acc)
	require.NoError(t, err)
	var out fleet.MDMIdPAccount
	err = sqlx.GetContext(ctx, ds.reader, &out, "SELECT * FROM mdm_idp_accounts WHERE uuid = 'ABC-DEF'")
	require.NoError(t, err)
	require.Equal(t, acc, &out)
}

func testIgnoreMDMClientError(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create new record for remove pending
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileID:         uint(1),
		ProfileIdentifier: "p1",
		ProfileName:       "name1",
		HostUUID:          "h1",
		CommandUUID:       "c1",
		OperationType:     fleet.MDMAppleOperationTypeRemove,
		Status:            &fleet.MDMAppleDeliveryPending,
	}}))
	cps, err := ds.GetHostMDMProfiles(ctx, "h1")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name1", cps[0].Name)
	require.Equal(t, fleet.MDMAppleOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *cps[0].Status)

	// simulate remove failed with client error message
	require.NoError(t, ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   "c1",
		HostUUID:      "h1",
		Status:        &fleet.MDMAppleDeliveryFailed,
		Detail:        "MDMClientError (89): Profile with identifier 'p1' not found.",
		OperationType: fleet.MDMAppleOperationTypeRemove,
	}))
	cps, err = ds.GetHostMDMProfiles(ctx, "h1")
	require.NoError(t, err)
	require.Len(t, cps, 0) // we ignore error code 89 and delete the pending record as well

	// create another new record
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileID:         uint(2),
		ProfileIdentifier: "p2",
		ProfileName:       "name2",
		HostUUID:          "h2",
		CommandUUID:       "c2",
		OperationType:     fleet.MDMAppleOperationTypeRemove,
		Status:            &fleet.MDMAppleDeliveryPending,
	}}))
	cps, err = ds.GetHostMDMProfiles(ctx, "h2")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name2", cps[0].Name)
	require.Equal(t, fleet.MDMAppleOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *cps[0].Status)

	// simulate remove failed with another client error message that we don't want to ignore
	require.NoError(t, ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   "c2",
		HostUUID:      "h2",
		Status:        &fleet.MDMAppleDeliveryFailed,
		Detail:        "MDMClientError (96): Cannot replace profile 'p2' because it was not installed by the MDM server.",
		OperationType: fleet.MDMAppleOperationTypeRemove,
	}))
	cps, err = ds.GetHostMDMProfiles(ctx, "h2")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name2", cps[0].Name)
	require.Equal(t, fleet.MDMAppleOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMAppleDeliveryFailed, *cps[0].Status)
	require.Equal(t, "MDMClientError (96): Cannot replace profile 'p2' because it was not installed by the MDM server.", cps[0].Detail)
}

func testDeleteMDMAppleProfilesForHost(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	h, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host0-osquery-id"),
		NodeKey:         ptr.String("host0-node-key"),
		UUID:            "host0-test-mdm-profiles",
		Hostname:        "hostname0",
	})
	require.NoError(t, err)

	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileID:         uint(1),
		ProfileIdentifier: "p1",
		ProfileName:       "name1",
		HostUUID:          h.UUID,
		CommandUUID:       "c1",
		OperationType:     fleet.MDMAppleOperationTypeRemove,
		Status:            &fleet.MDMAppleDeliveryPending,
	}}))

	gotProfs, err := ds.GetHostMDMProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 1)

	err = ds.DeleteMDMAppleProfilesForHost(ctx, h.UUID)
	require.NoError(t, err)
	gotProfs, err = ds.GetHostMDMProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)
}

func testBulkSetPendingMDMAppleHostProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	hostIDsFromHosts := func(hosts ...*fleet.Host) []uint {
		ids := make([]uint, len(hosts))
		for i, h := range hosts {
			ids[i] = h.ID
		}
		return ids
	}

	// only asserts the profile ID, status and operation
	assertHostProfiles := func(want map[*fleet.Host][]fleet.HostMDMAppleProfile) {
		for h, wantProfs := range want {
			gotProfs, err := ds.GetHostMDMProfiles(ctx, h.UUID)
			require.NoError(t, err)
			require.Equal(t, len(wantProfs), len(gotProfs), "host uuid: %s", h.UUID)

			sort.Slice(gotProfs, func(i, j int) bool {
				l, r := gotProfs[i], gotProfs[j]
				return l.ProfileID < r.ProfileID
			})
			sort.Slice(wantProfs, func(i, j int) bool {
				l, r := wantProfs[i], wantProfs[j]
				return l.ProfileID < r.ProfileID
			})
			for i, wp := range wantProfs {
				gp := gotProfs[i]
				require.Equal(t, wp.ProfileID, gp.ProfileID, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
				require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
				require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
			}
		}
	}

	// create some hosts, all enrolled
	enrolledHosts := make([]*fleet.Host, 3)
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
		enrolledHosts[i] = h
		t.Logf("enrolled host [%d]: %s", i, h.UUID)
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

	// bulk set for no target ids, does nothing
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, nil, nil, nil)
	require.NoError(t, err)
	// bulk set for combination of target ids, not allowed
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, []uint{1}, []uint{2}, nil, nil)
	require.Error(t, err)

	// bulk set for all created hosts, no profiles yet so nothing changed
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDsFromHosts(append(enrolledHosts, unenrolledHost, linuxHost)...), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {},
		enrolledHosts[1]: {},
		enrolledHosts[2]: {},
		unenrolledHost:   {},
		linuxHost:        {},
	})

	// create some global (no-team) profiles
	globalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G1", "G1", "a"),
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, nil, globalProfiles)
	require.NoError(t, err)
	globalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, globalProfiles, 3)

	// list profiles to install, should result in the global profiles for all 3
	// enrolled hosts
	toInstall, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstall, len(globalProfiles)*len(enrolledHosts))

	// none are listed as "to remove"
	toRemove, err := ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemove, 0)

	// bulk set for all created hosts, enrolled hosts get the no-team profiles
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDsFromHosts(append(enrolledHosts, unenrolledHost, linuxHost)...), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// move enrolledHosts[0] to that team
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{enrolledHosts[0].ID})
	require.NoError(t, err)

	// 6 are still reported as "to install" because op=install and status=nil
	toInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstall, 6)

	// those applied to enrolledHosts[0] are listed as "to remove"
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemove, 3)

	// update status of the moved host (team has no profiles)
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDsFromHosts(enrolledHosts[0]), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// create another team
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	// move enrolledHosts[1] to that team
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{enrolledHosts[1].ID})
	require.NoError(t, err)

	// 3 are still reported as "to install" because op=install and status=nil
	toInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstall, 3)

	// 6 are now "to remove"
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemove, 6)

	// update status of the moved host via its uuid (team has no profiles)
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, nil, nil, []string{enrolledHosts[1].UUID})
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// create profiles for team 1
	tm1Profiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.1", "T1.1", "d"),
		configProfileForTest(t, "T1.2", "T1.2", "e"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, &team1.ID, tm1Profiles)
	require.NoError(t, err)
	tm1Profiles, err = ds.ListMDMAppleConfigProfiles(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, tm1Profiles, 2)

	// 5 are now reported as "to install" (3 global + 2 team1)
	toInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Len(t, toInstall, 5)

	// 6 are still "to remove"
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemove, 6)

	// update status of the affected team
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: tm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// successfully remove globalProfiles[0, 1] for enrolledHosts[0], and remove as failed globalProfiles[2]
	// Do *not* use UpdateOrDeleteHostMDMAppleProfile here, as it deletes/updates based on command uuid
	// (meant to be called from the MDMDirector in response from MDM commands), it would delete/update
	// all rows in this test since we don't have command uuids.
	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[0].ProfileID,
			Status: &fleet.MDMAppleDeliveryApplied, OperationType: fleet.MDMAppleOperationTypeRemove},
		{HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[1].ProfileID,
			Status: &fleet.MDMAppleDeliveryApplied, OperationType: fleet.MDMAppleOperationTypeRemove},
		{HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[2].ProfileID,
			Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
	})
	require.NoError(t, err)

	// add a profile to team1, and remove profile T1.1
	newTm1Profiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T1.2", "T1.2", "e"),
		configProfileForTest(t, "T1.3", "T1.3", "f"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, &team1.ID, newTm1Profiles)
	require.NoError(t, err)
	newTm1Profiles, err = ds.ListMDMAppleConfigProfiles(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, newTm1Profiles, 2)

	// update status of the affected team
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// re-add tm1Profiles[0] to list of team1 profiles
	// NOTE: even though it is the same profile, it's unique DB ID is different because
	// it got deleted and re-inserted from the team's profiles, so this is reflected in
	// the host's profiles list.
	newTm1Profiles = []*fleet.MDMAppleConfigProfile{
		tm1Profiles[0],
		configProfileForTest(t, "T1.2", "T1.2", "e"),
		configProfileForTest(t, "T1.3", "T1.3", "f"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, &team1.ID, newTm1Profiles)
	require.NoError(t, err)
	newTm1Profiles, err = ds.ListMDMAppleConfigProfiles(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, newTm1Profiles, 3)

	// update status of the affected team
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// remove a global profile and add a new one
	newGlobalProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
		configProfileForTest(t, "G4", "G4", "d"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, nil, newGlobalProfiles)
	require.NoError(t, err)
	newGlobalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, newGlobalProfiles, 3)

	// update status of the affected "no-team"
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{0}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// add another global profile
	newGlobalProfiles = []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "G2", "G2", "b"),
		configProfileForTest(t, "G3", "G3", "c"),
		configProfileForTest(t, "G4", "G4", "d"),
		configProfileForTest(t, "G5", "G5", "e"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, nil, newGlobalProfiles)
	require.NoError(t, err)
	newGlobalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, newGlobalProfiles, 4)

	// update status via the new profile's ID
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, nil, []uint{newGlobalProfiles[3].ProfileID}, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// add a profile to team2
	tm2Profiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T2.1", "T2.1", "a"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, &team2.ID, tm2Profiles)
	require.NoError(t, err)
	tm2Profiles, err = ds.ListMDMAppleConfigProfiles(ctx, &team2.ID)
	require.NoError(t, err)
	require.Len(t, tm2Profiles, 1)

	// update status via tm2 id and the global 0 id to test that custom sql statement
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, nil, []uint{team2.ID, 0}, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// do a final sync of all hosts, should not change anything
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDsFromHosts(append(enrolledHosts, unenrolledHost, linuxHost)...), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: nil, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})
}

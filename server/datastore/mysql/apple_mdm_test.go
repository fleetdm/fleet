package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
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
		{"TestListMDMAppleConfigProfiles", testListMDMAppleConfigProfiles},
		{"TestHostDetailsMDMProfiles", testHostDetailsMDMProfiles},
		{"TestBatchSetMDMAppleProfiles", testBatchSetMDMAppleProfiles},
		{"TestMDMAppleProfileManagement", testMDMAppleProfileManagement},
		{"TestUpdateHostMDMAppleProfile", testGetMDMAppleProfilesContents},
		{"TestMDMAppleHostsProfilesSummary", testMDMAppleHostsProfilesSummary},
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
		mc := fleet.Mobileconfig([]byte(name + identifier))
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

func storeDummyConfigProfileForTest(t *testing.T, ds *Datastore) *fleet.MDMAppleConfigProfile {
	dummyMC := fleet.Mobileconfig([]byte("DummyTestMobileconfigBytes"))
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
		args = append(args, p.HostUUID, p.ProfileID, p.CommandUUID, *p.Status, p.OperationType, p.Detail)
	}
	for _, p := range expectedProfiles1 {
		args = append(args, p.HostUUID, p.ProfileID, p.CommandUUID, *p.Status, p.OperationType, p.Detail)
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
	INSERT INTO host_mdm_apple_profiles (
		host_uuid, profile_id, command_uuid, status, operation_type, detail)
	VALUES (?,?,?,?,?,?),(?,?,?,?,?,?),(?,?,?,?,?,?),(?,?,?,?,?,?),(?,?,?,?,?,?),(?,?,?,?,?,?)
		`, args...,
		)
		if err != nil {
			return err
		}
		return nil
	})

	gotHost, err = ds.Host(ctx, h1.ID)
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
		got, err := ds.ListMDMAppleConfigProfiles(ctx, tmID)
		require.NoError(t, err)

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
}

func configProfileForTest(t *testing.T, name, identifier, uuid string) *fleet.MDMAppleConfigProfile {
	prof := fleet.Mobileconfig(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
	cp, err := prof.ParseConfigProfile()
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
	nanoEnroll(t, ds, host1)

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
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-1"},
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
	nanoEnroll(t, ds, host2)

	// still the same profiles to assign as there are no profiles for team 1
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-1"},
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
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, HostUUID: "test-uuid-2"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, HostUUID: "test-uuid-2"},
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
	nanoEnroll(t, ds, host3)

	// more profiles, this time for both global hosts and the team
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, HostUUID: "test-uuid-2"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, HostUUID: "test-uuid-2"},
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-3"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-3"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-3"},
	}, profiles)

	// cron runs and updates the status
	err = ds.BulkUpsertMDMAppleHostProfiles(
		ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileID:         globalPfs[0].ProfileID,
				ProfileIdentifier: globalPfs[0].Identifier,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[0].ProfileID,
				ProfileIdentifier: globalPfs[0].Identifier,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[0].ProfileID,
				ProfileIdentifier: teamPfs[0].Identifier,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMAppleDeliveryApplied,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[1].ProfileID,
				ProfileIdentifier: teamPfs[1].Identifier,
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
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, HostUUID: "test-uuid-1"},
	}, profiles)

	// profiles to be removed includes host1's old profiles
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, HostUUID: "test-uuid-1"},
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
		want map[uint]fleet.Mobileconfig
	}{
		{[]uint{}, nil},
		{nil, nil},
		{[]uint{profiles[0].ProfileID}, map[uint]fleet.Mobileconfig{profiles[0].ProfileID: profiles[0].Mobileconfig}},
		{
			[]uint{profiles[0].ProfileID, profiles[1].ProfileID, profiles[2].ProfileID},
			map[uint]fleet.Mobileconfig{
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

func nanoEnroll(t *testing.T, ds *Datastore, host *fleet.Host) {
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
}

func testMDMAppleHostsProfilesSummary(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	generateCP := func(name string, identifier string, teamID uint) *fleet.MDMAppleConfigProfile {
		mc := fleet.Mobileconfig([]byte(name + identifier))
		return &fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   identifier,
			TeamID:       &teamID,
			Mobileconfig: mc,
		}
	}

	upsertHostCPs := func(hosts []*fleet.Host, profiles []*fleet.MDMAppleConfigProfile, opType fleet.MDMAppleOperationType, status fleet.MDMAppleDeliveryStatus) {
		upserts := []*fleet.MDMAppleBulkUpsertHostProfilePayload{}
		for _, h := range hosts {
			for _, cp := range profiles {
				payload := fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileID:         cp.ProfileID,
					ProfileIdentifier: cp.Identifier,
					HostUUID:          h.UUID,
					CommandUUID:       "",
					OperationType:     opType,
					Status:            &status,
				}
				upserts = append(upserts, &payload)
			}
		}
		err := ds.BulkUpsertMDMAppleHostProfiles(ctx, upserts)
		require.NoError(t, err)
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

	// all hosts pending install of all profiles
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryPending)
	res, err := ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)

	// hosts[0] and hosts[1] failed one profile
	upsertHostCPs(hosts[0:2], noTeamCPs[0:1], fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryFailed)
	// hosts[0] also failed another profile
	upsertHostCPs(hosts[0:1], noTeamCPs[1:2], fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryFailed)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // two hosts are failing at least one profile (hosts[0] and hosts[1])
	require.Equal(t, uint(2), res.Failed)             // only count one failure per host (hosts[0] failed two profiles but only counts once)
	require.Equal(t, uint(0), res.Latest)

	// hosts[0:3] applied a third profile
	upsertHostCPs(hosts[0:3], noTeamCPs[2:3], fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // no change
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // no change, host must apply all profiles count as latest

	// hosts[9] applied all profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // subtract third host from pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(1), res.Latest)             // add one host that has applied all profiles

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "rocket"})
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending) // no profiles yet
	require.Equal(t, uint(0), res.Failed)  // no profiles yet
	require.Equal(t, uint(0), res.Latest)  // no profiles yet

	// transfer hosts[9] to new team
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{hosts[9].ID})
	require.NoError(t, err)
	// remove all no team profiles from hosts[9]
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, fleet.MDMAppleDeliveryPending)

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // subtract two failed hosts and one transferred host
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Latest)             // hosts[9] was transferred so this is now zero

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary for new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)

	// create somes config profiles for the new team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 10; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), tm.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}

	// install all team profiles on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs, fleet.MDMAppleOperationTypeInstall, fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is still pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Latest)

	// hosts[9] successfully removed old profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, fleet.MDMAppleDeliveryApplied)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Latest) // hosts[9] is all good

	// confirm no changes in summary for profiles with no team
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, ptr.Uint(0)) // team id zero represents no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // subtract two failed hosts and one transferred host
	require.Equal(t, uint(2), res.Failed)             // two failed hosts
	require.Equal(t, uint(0), res.Latest)             // hosts[9] transferred to new team so is not counted under no team
}

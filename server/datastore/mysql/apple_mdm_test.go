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
		{"TestBatchSetMDMAppleProfiles", testBatchSetMDMAppleProfiles},
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

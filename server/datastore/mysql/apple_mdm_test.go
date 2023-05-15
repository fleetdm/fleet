package mysql

import (
	"context"
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMApple(t *testing.T) {
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
		{"TestAggregateMacOSSettingsStatusWithFileVault", testAggregateMacOSSettingsStatusWithFileVault},
		{"TestMDMAppleHostsProfilesStatus", testMDMAppleHostsProfilesStatus},
		{"TestMDMAppleInsertIdPAccount", testMDMAppleInsertIdPAccount},
		{"TestIgnoreMDMClientError", testIgnoreMDMClientError},
		{"TestDeleteMDMAppleProfilesForHost", testDeleteMDMAppleProfilesForHost},
		{"TestBulkSetPendingMDMAppleHostProfiles", testBulkSetPendingMDMAppleHostProfiles},
		{"TestGetMDMAppleCommandResults", testGetMDMAppleCommandResults},
		{"TestBulkUpsertMDMAppleConfigProfiles", testBulkUpsertMDMAppleConfigProfile},
		{"TestMDMAppleBootstrapPackageCRUD", testMDMAppleBootstrapPackageCRUD},
		{"TestListMDMAppleCommands", testListMDMAppleCommands},
		{"TestMDMAppleEULA", testMDMAppleEULA},
		{"TestMDMAppleSetupAssistant", testMDMAppleSetupAssistant},
		{"TestMDMAppleEnrollmentProfile", testMDMAppleEnrollmentProfile},
		{"TestListMDMAppleSerials", testListMDMAppleSerials},
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

func generateCP(name string, identifier string, teamID uint) *fleet.MDMAppleConfigProfile {
	mc := mobileconfig.Mobileconfig([]byte(name + identifier))
	return &fleet.MDMAppleConfigProfile{
		Name:         name,
		Identifier:   identifier,
		TeamID:       &teamID,
		Mobileconfig: mc,
	}
}

func testListMDMAppleConfigProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

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
		p1.ProfileID: {HostUUID: h0.UUID, Name: p1.Name, ProfileID: p1.ProfileID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMAppleDeliveryVerifying, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: ""},
		p2.ProfileID: {HostUUID: h0.UUID, Name: p2.Name, ProfileID: p2.ProfileID, CommandUUID: "cmd2-uuid", Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove, Detail: "Error removing profile"},
	}

	expectedProfiles1 := map[uint]fleet.HostMDMAppleProfile{
		p0.ProfileID: {HostUUID: h1.UUID, Name: p0.Name, ProfileID: p0.ProfileID, CommandUUID: "cmd0-uuid", Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: "Error installing profile"},
		p1.ProfileID: {HostUUID: h1.UUID, Name: p1.Name, ProfileID: p1.ProfileID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMAppleDeliveryVerifying, OperationType: fleet.MDMAppleOperationTypeInstall, Detail: ""},
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

	// mark h1's remove+failed profile as remove+verifying, deletes the host profile row
	h1RemoveFailed := expectedProfiles1[p2.ProfileID]
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      h1RemoveFailed.HostUUID,
		CommandUUID:   h1RemoveFailed.CommandUUID,
		ProfileID:     h1RemoveFailed.ProfileID,
		Name:          h1RemoveFailed.Name,
		Status:        &fleet.MDMAppleDeliveryVerifying,
		OperationType: fleet.MDMAppleOperationTypeRemove,
		Detail:        "",
	})
	require.NoError(t, err)

	gotProfs, err = ds.GetHostMDMProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 2) // remove+verifying is not there anymore

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

	n, tmID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(3), n) // 3 new hosts ("abc", "xyz", "ijk")
	require.Nil(t, tmID)

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

	n, tmID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Nil(t, tmID)
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

	n, tmID, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.NotNil(t, tmID)
	require.Equal(t, team.ID, *tmID)

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

	n, tmID, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.EqualValues(t, n, 1)
	require.Nil(t, tmID)

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
	n, tmID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.Nil(t, tmID)

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
	n, tmID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
	require.Nil(t, tmID)

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

	profiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "z"),
	}

	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileID:         profiles[0].ProfileID,
			ProfileIdentifier: profiles[0].Identifier,
			ProfileName:       profiles[0].Name,
			HostUUID:          testUUID,
			Status:            &fleet.MDMAppleDeliveryVerifying,
			OperationType:     fleet.MDMAppleOperationTypeInstall,
			CommandUUID:       "command-uuid",
			Checksum:          []byte("csum"),
		},
	},
	)
	require.NoError(t, err)

	hostProfs, err := ds.GetHostMDMProfiles(ctx, testUUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, len(profiles))

	var hostID uint
	err = sqlx.GetContext(context.Background(), ds.reader, &hostID, `SELECT id  FROM hosts WHERE uuid = ?`, testUUID)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hostID, "asdf")
	require.NoError(t, err)

	key, err := ds.GetHostDiskEncryptionKey(ctx, hostID)
	require.NoError(t, err)
	require.NotNil(t, key)

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

	hostProfs, err = ds.GetHostMDMProfiles(ctx, testUUID)
	require.NoError(t, err)
	require.Empty(t, hostProfs)
	key, err = ds.GetHostDiskEncryptionKey(ctx, hostID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, key)
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
	require.Equal(t, mNoTm["I1"], mNoTmb["I1"])

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
	// identifier for N1-I1 is unchanged
	require.Equal(t, mTm1b["I1"], mTm1c["I1"])
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
	sum := md5.Sum(prof) // nolint:gosec // used only to hash for efficient comparisons
	cp.Checksum = sum[:]
	require.NoError(t, err)
	return cp
}

func testMDMAppleProfileManagement(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	matchProfiles := func(want, got []*fleet.MDMAppleProfilePayload) {
		// match only the fields we care about
		for _, p := range got {
			require.NotEmpty(t, p.Checksum)
			p.Checksum = nil
		}
		require.ElementsMatch(t, want, got)
	}

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

	// if there are no hosts, then no profiles need to be installed
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
	matchProfiles([]*fleet.MDMAppleProfilePayload{
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
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileID: globalPfs[0].ProfileID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[1].ProfileID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileID: globalPfs[2].ProfileID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// assign profiles to team 1
	teamProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N4", "I4", "x"),
		configProfileForTest(t, "N5", "I5", "y"),
	}
	err = ds.BatchSetMDMAppleProfiles(ctx, &team.ID, teamProfiles)
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
	matchProfiles([]*fleet.MDMAppleProfilePayload{
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
	matchProfiles([]*fleet.MDMAppleProfilePayload{
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
				Checksum:          globalProfiles[0].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[0].ProfileID,
				ProfileIdentifier: globalPfs[0].Identifier,
				ProfileName:       globalPfs[0].Name,
				Checksum:          globalProfiles[0].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				Checksum:          globalProfiles[1].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[1].ProfileID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				Checksum:          globalProfiles[1].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				Checksum:          globalProfiles[2].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         globalPfs[2].ProfileID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				Checksum:          globalProfiles[2].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[0].ProfileID,
				ProfileIdentifier: teamPfs[0].Identifier,
				ProfileName:       teamPfs[0].Name,
				Checksum:          teamProfiles[0].Checksum,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMAppleDeliveryVerifying,
				OperationType:     fleet.MDMAppleOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileID:         teamPfs[1].ProfileID,
				ProfileIdentifier: teamPfs[1].Identifier,
				ProfileName:       teamPfs[1].Name,
				Checksum:          teamProfiles[1].Checksum,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMAppleDeliveryVerifying,
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
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileID: teamPfs[0].ProfileID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileID: teamPfs[1].ProfileID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// profiles to be removed includes host1's old profiles
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
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

func upsertHostCPs(
	hosts []*fleet.Host,
	profiles []*fleet.MDMAppleConfigProfile,
	opType fleet.MDMAppleOperationType,
	status *fleet.MDMAppleDeliveryStatus,
	ctx context.Context,
	ds *Datastore,
	t *testing.T,
) {
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
				Checksum:          []byte("csum"),
			}
			upserts = append(upserts, &payload)
		}
	}
	err := ds.BulkUpsertMDMAppleHostProfiles(ctx, upserts)
	require.NoError(t, err)
}

func testAggregateMacOSSettingsStatusWithFileVault(t *testing.T, ds *Datastore) {
	ctx := context.Background()

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
	// add filevault profile for no team
	fvNoTeam, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault", "com.fleetdm.fleet.mdm.filevault", 0))
	require.NoError(t, err)

	upsertHostCPs(hosts, append(noTeamCPs, fvNoTeam), fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	res, err := ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because filevault not installed
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryPending, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because filevault pending
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because no disk encryption key
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "foo")
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because disk encryption key decryptable is not set
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-1), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[0] now has filevault fully enforced

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[1].ID, "bar")
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[1].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-1), res.Pending) // hosts[1] still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[1].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(2), res.Verifying) // hosts[0] and hosts[1] now have filevault fully enforced

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, hosts[:2]))

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test"})
	require.NoError(t, err)

	// add hosts[9] to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{hosts[9].ID})
	require.NoError(t, err)

	// remove profiles from hosts[9]
	upsertHostCPs(hosts[9:10], append(noTeamCPs, fvNoTeam), fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying)

	// create somes config profiles for team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 2; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), team.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}
	// add filevault profile for team
	fvTeam, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault", mobileconfig.FleetFileVaultPayloadIdentifier, team.ID))
	require.NoError(t, err)

	upsertHostCPs(hosts[9:10], append(teamCPs, fvTeam), fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending because it has no disk encryption key
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[9].ID, "baz")
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[9].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[9] now has filevault fully enforced

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, &team.ID, hosts[9:10]))
}

func testMDMAppleHostsProfilesStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

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
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	res, err := ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// all hosts pending install of all profiles
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryPending, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0] and hosts[1] failed one profile
	upsertHostCPs(hosts[0:2], noTeamCPs[0:1], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed, ctx, ds, t)
	// hosts[0] and hosts[1] have one profile pending as nil
	upsertHostCPs(hosts[0:2], noTeamCPs[3:4], fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	// hosts[0] also failed another profile
	upsertHostCPs(hosts[0:1], noTeamCPs[1:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed, ctx, ds, t)
	// hosts[4] has all profiles reported as nil (pending)
	upsertHostCPs(hosts[4:5], noTeamCPs, fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	// hosts[5] has one profile reported as nil (pending)
	upsertHostCPs(hosts[5:6], noTeamCPs[0:1], fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // two hosts are failing at least one profile (hosts[0] and hosts[1])
	require.Equal(t, uint(2), res.Failed)             // only count one failure per host (hosts[0] failed two profiles but only counts once)
	require.Equal(t, uint(0), res.Verifying)
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0:3] installed a third profile
	upsertHostCPs(hosts[0:3], noTeamCPs[2:3], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // no change
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// hosts[6] deletes all its profiles
	tx, err := ds.writer.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, ds.deleteMDMAppleProfilesForHost(ctx, tx, hosts[6].UUID))
	require.NoError(t, tx.Commit())
	pendingHosts := append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] installed all profiles but one is with status nil (pending)
	upsertHostCPs(hosts[9:10], noTeamCPs[:9], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	upsertHostCPs(hosts[9:10], noTeamCPs[9:10], fleet.MDMAppleOperationTypeInstall, nil, ctx, ds, t)
	pendingHosts = append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore, hosts[9] still pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] installed all profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract hosts[6 and 9] from pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(1), res.Verifying)          // add one host that has installed all profiles
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), hosts[9:10]))

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "rocket"})
	require.NoError(t, err)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)   // no profiles yet
	require.Equal(t, uint(0), res.Failed)    // no profiles yet
	require.Equal(t, uint(0), res.Verifying) // no profiles yet
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, &tm.ID, []*fleet.Host{}))

	// transfer hosts[9] to new team
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{hosts[9].ID})
	require.NoError(t, err)
	// remove all no team profiles from hosts[9]
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryPending, ctx, ds, t)

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // hosts[9] is still not pending, transferred to team
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // hosts[9] was transferred so this is now zero
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))

	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID) // get summary for new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, &tm.ID, []*fleet.Host{}))

	// create somes config profiles for the new team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 10; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), tm.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}

	// install all team profiles on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is still pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, &tm.ID, []*fleet.Host{}))

	// hosts[9] successfully removed old profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[9] is all good
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, &tm.ID, hosts[9:10]))

	// confirm no changes in summary for profiles with no team
	res, err = ds.GetMDMAppleHostsProfilesSummary(ctx, ptr.Uint(0)) // team id zero represents no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract two failed hosts, one without profiles and hosts[9] transferred
	require.Equal(t, uint(2), res.Failed)             // two failed hosts
	require.Equal(t, uint(0), res.Verifying)          // hosts[9] transferred to new team so is not counted under no team
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.MacOSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.MacOSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.MacOSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
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
		Checksum:          []byte("csum"),
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
		Checksum:          []byte("csum"),
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
		Checksum:          []byte("csum"),
	}}))

	gotProfs, err := ds.GetHostMDMProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 1)

	tx, err := ds.writer.BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, ds.deleteMDMAppleProfilesForHost(ctx, tx, h.UUID))
	require.NoError(t, tx.Commit())
	require.NoError(t, err)
	gotProfs, err = ds.GetHostMDMProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)
}

func createDiskEncryptionRecord(ctx context.Context, ds *Datastore, t *testing.T, hostId uint, key string, decryptable bool, threshold time.Time) {
	err := ds.SetOrUpdateHostDiskEncryptionKey(ctx, hostId, key)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hostId}, decryptable, threshold)
	require.NoError(t, err)
}

func TestMDMAppleFileVaultSummary(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	// 10 new hosts
	var hosts []*fleet.Host
	for i := 0; i < 7; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now())
		hosts = append(hosts, h)
	}

	// no teams tests =====
	noTeamFVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault-1", "com.fleetdm.fleet.mdm.filevault", 0))
	require.NoError(t, err)

	// verifying status
	verifyingHost := hosts[0]
	upsertHostCPs([]*fleet.Host{verifyingHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
	createDiskEncryptionRecord(ctx, ds, t, verifyingHost.ID, "key-1", true, oneMinuteAfterThreshold)

	fvProfileSummary, err := ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err := ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// action required status
	requiredActionHost := hosts[1]
	upsertHostCPs(
		[]*fleet.Host{requiredActionHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMAppleOperationTypeInstall,
		&fleet.MDMAppleDeliveryVerifying, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{requiredActionHost.ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// enforcing status
	enforcingHost := hosts[2]

	// host profile status is `pending`
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMAppleOperationTypeInstall,
		&fleet.MDMAppleDeliveryPending, ctx, ds, t,
	)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// host profile status does not exist
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMAppleOperationTypeInstall,
		nil, ctx, ds, t,
	)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// host profile status is verifying but decryptable key field does not exist
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMAppleOperationTypeInstall,
		&fleet.MDMAppleDeliveryPending, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{enforcingHost.ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// failed status
	failedHost := hosts[3]
	upsertHostCPs([]*fleet.Host{failedHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed, ctx, ds, t)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(1), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(1), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// removing enforcement status
	removingEnforcementHost := hosts[4]
	upsertHostCPs([]*fleet.Host{removingEnforcementHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryPending, ctx, ds, t)
	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)

	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(1), fvProfileSummary.Failed)
	require.Equal(t, uint(1), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(3), allProfilesSummary.Pending)
	require.Equal(t, uint(1), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)

	// teams filter tests =====
	verifyingTeam1Host := hosts[6]
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-1"})
	require.NoError(t, err)
	team1FVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("filevault-team-1", "com.fleetdm.fleet.mdm.filevault", tm.ID))
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{verifyingTeam1Host.ID})
	require.NoError(t, err)

	upsertHostCPs([]*fleet.Host{verifyingTeam1Host}, []*fleet.MDMAppleConfigProfile{team1FVProfile}, fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, ctx, ds, t)
	createDiskEncryptionRecord(ctx, ds, t, verifyingTeam1Host.ID, "key-2", true, oneMinuteAfterThreshold)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleHostsProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
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
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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

	// those installed to enrolledHosts[0] are listed as "to remove"
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Len(t, toRemove, 3)

	// update status of the moved host (team has no profiles)
	err = ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDsFromHosts(enrolledHosts[0]), nil, nil, nil)
	require.NoError(t, err)
	assertHostProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		enrolledHosts[0]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: tm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})

	// successfully remove globalProfiles[0, 1] for enrolledHosts[0], and remove as failed globalProfiles[2]
	// Do *not* use UpdateOrDeleteHostMDMAppleProfile here, as it deletes/updates based on command uuid
	// (meant to be called from the MDMDirector in response from MDM commands), it would delete/update
	// all rows in this test since we don't have command uuids.
	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[0].ProfileID,
			Status: &fleet.MDMAppleDeliveryVerifying, OperationType: fleet.MDMAppleOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[1].ProfileID,
			Status: &fleet.MDMAppleDeliveryVerifying, OperationType: fleet.MDMAppleOperationTypeRemove, Checksum: []byte("csum"),
		},
		{
			HostUUID: enrolledHosts[0].UUID, ProfileID: globalProfiles[2].ProfileID,
			Status: &fleet.MDMAppleDeliveryFailed, OperationType: fleet.MDMAppleOperationTypeRemove, Checksum: []byte("csum"),
		},
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
			{ProfileID: tm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
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
			{ProfileID: newTm1Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newTm1Profiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[1]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: globalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: tm2Profiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		enrolledHosts[2]: {
			{ProfileID: globalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeRemove},
			{ProfileID: newGlobalProfiles[0].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[1].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[2].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
			{ProfileID: newGlobalProfiles[3].ProfileID, Status: &fleet.MDMAppleDeliveryPending, OperationType: fleet.MDMAppleOperationTypeInstall},
		},
		unenrolledHost: {},
		linuxHost:      {},
	})
}

func testGetMDMAppleCommandResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	createRawCmd := func(cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>ProfileList</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)
	}

	// no enrolled host, unknown command
	res, err := ds.GetMDMAppleCommandResults(ctx, uuid.New().String())
	require.NoError(t, err)
	require.Empty(t, res)

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

	commander, storage := createMDMAppleCommanderAndStorage(t, ds)

	// enqueue a command for an unenrolled host fails with a foreign key error (no enrollment)
	uuid1 := uuid.New().String()
	err = commander.EnqueueCommand(ctx, []string{unenrolledHost.UUID}, createRawCmd(uuid1))
	require.Error(t, err)
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.Equal(t, uint16(mysqlerr.ER_NO_REFERENCED_ROW_2), mysqlErr.Number)

	// command has no results
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid1)
	require.NoError(t, err)
	require.Empty(t, res)

	// enqueue a command for a couple of enrolled hosts
	uuid2 := uuid.New().String()
	rawCmd2 := createRawCmd(uuid2)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[0].UUID, enrolledHosts[1].UUID}, rawCmd2)
	require.NoError(t, err)

	// command has no results yet
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid2)
	require.NoError(t, err)
	require.Empty(t, res)

	// simulate a result for enrolledHosts[0]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[0].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Raw:         []byte(rawCmd2),
	})
	require.NoError(t, err)

	// command has a result for [0]
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid2)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.Equal(t, res[0], &fleet.MDMAppleCommandResult{
		DeviceID:    enrolledHosts[0].UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd2),
	})

	// simulate a result for enrolledHosts[1]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[1].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Error",
		RequestType: "ProfileList",
		Raw:         []byte(rawCmd2),
	})
	require.NoError(t, err)

	// command has both results
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid2)
	require.NoError(t, err)
	require.Len(t, res, 2)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.NotZero(t, res[1].UpdatedAt)
	res[1].UpdatedAt = time.Time{}

	require.ElementsMatch(t, res, []*fleet.MDMAppleCommandResult{
		{
			DeviceID:    enrolledHosts[0].UUID,
			CommandUUID: uuid2,
			Status:      "Acknowledged",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
		},
		{
			DeviceID:    enrolledHosts[1].UUID,
			CommandUUID: uuid2,
			Status:      "Error",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
		},
	})

	// delete host [0] and verify that it didn't delete its command results
	err = ds.DeleteHost(ctx, enrolledHosts[0].ID)
	require.NoError(t, err)

	res, err = ds.GetMDMAppleCommandResults(ctx, uuid2)
	require.NoError(t, err)
	require.Len(t, res, 2)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.NotZero(t, res[1].UpdatedAt)
	res[1].UpdatedAt = time.Time{}
	require.ElementsMatch(t, res, []*fleet.MDMAppleCommandResult{
		{
			DeviceID:    enrolledHosts[0].UUID,
			CommandUUID: uuid2,
			Status:      "Acknowledged",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
		},
		{
			DeviceID:    enrolledHosts[1].UUID,
			CommandUUID: uuid2,
			Status:      "Error",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
		},
	})
}

func createMDMAppleCommanderAndStorage(t *testing.T, ds *Datastore) (*apple_mdm.MDMAppleCommander, *NanoMDMStorage) {
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	mdmStorage, err := ds.NewMDMAppleMDMStorage(testCertPEM, testKeyPEM)
	require.NoError(t, err)

	return apple_mdm.NewMDMAppleCommander(mdmStorage, pusherFunc(okPusherFunc)), mdmStorage
}

func okPusherFunc(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	m := make(map[string]*push.Response, len(ids))
	for _, id := range ids {
		m[id] = &push.Response{Id: id}
	}
	return m, nil
}

type pusherFunc func(context.Context, []string) (map[string]*push.Response, error)

func (f pusherFunc) Push(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	return f(ctx, ids)
}

func testBulkUpsertMDMAppleConfigProfile(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	mc := mobileconfig.Mobileconfig([]byte("TestConfigProfile"))
	globalCP := &fleet.MDMAppleConfigProfile{
		Name:         "DummyTestName",
		Identifier:   "DummyTestIdentifier",
		Mobileconfig: mc,
		TeamID:       nil,
	}
	teamCP := &fleet.MDMAppleConfigProfile{
		Name:         "DummyTestName",
		Identifier:   "DummyTestIdentifier",
		Mobileconfig: mc,
		TeamID:       ptr.Uint(1),
	}
	allProfiles := []*fleet.MDMAppleConfigProfile{globalCP, teamCP}

	checkProfiles := func() {
		for _, p := range allProfiles {
			profiles, err := ds.ListMDMAppleConfigProfiles(ctx, p.TeamID)
			require.NoError(t, err)
			require.Len(t, profiles, 1)
			checkConfigProfile(t, *p, *profiles[0])
		}
	}

	err := ds.BulkUpsertMDMAppleConfigProfiles(ctx, allProfiles)
	require.NoError(t, err)
	checkProfiles()

	newMc := mobileconfig.Mobileconfig([]byte("TestUpdatedConfigProfile"))
	globalCP.Mobileconfig = newMc
	teamCP.Mobileconfig = newMc
	err = ds.BulkUpsertMDMAppleConfigProfiles(ctx, allProfiles)
	require.NoError(t, err)
	checkProfiles()
}

func testMDMAppleBootstrapPackageCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	var nfe fleet.NotFoundError
	var aerr fleet.AlreadyExistsError

	err := ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{})
	require.Error(t, err)

	bp1 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(0),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1)
	require.NoError(t, err)

	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1)
	require.ErrorAs(t, err, &aerr)

	bp2 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(2),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp2)
	require.NoError(t, err)

	meta, err := ds.GetMDMAppleBootstrapPackageMeta(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, bp1.TeamID, meta.TeamID)
	require.Equal(t, bp1.Name, meta.Name)
	require.Equal(t, bp1.Sha256, meta.Sha256)
	require.Equal(t, bp1.Token, meta.Token)

	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 3)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	bytes, err := ds.GetMDMAppleBootstrapPackageBytes(ctx, bp1.Token)
	require.NoError(t, err)
	require.Equal(t, bp1.Bytes, bytes.Bytes)

	bytes, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, "fake")
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, bytes)

	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 0)
	require.NoError(t, err)

	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 0)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 0)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)
}

func testListMDMAppleCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	createRawCmd := func(reqType, cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>%s</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, reqType, cmdUUID)
	}

	// create some enrolled hosts
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
	// create a team
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	// assign enrolledHosts[2] to tm1
	err = ds.AddHostsToTeam(ctx, &tm1.ID, []uint{enrolledHosts[2].ID})
	require.NoError(t, err)

	commander, storage := createMDMAppleCommanderAndStorage(t, ds)

	// no commands yet
	res, err := ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, res)

	// enqueue a command for enrolled hosts [0] and [1]
	uuid1 := uuid.New().String()
	rawCmd1 := createRawCmd("ListApps", uuid1)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[0].UUID, enrolledHosts[1].UUID}, rawCmd1)
	require.NoError(t, err)

	// command has no results yet, so the status is empty
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 2)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.NotZero(t, res[1].UpdatedAt)
	res[1].UpdatedAt = time.Time{}

	require.ElementsMatch(t, res, []*fleet.MDMAppleCommand{
		{
			DeviceID:    enrolledHosts[0].UUID,
			CommandUUID: uuid1,
			Status:      "Pending",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[0].Hostname,
			TeamID:      nil,
		},
		{
			DeviceID:    enrolledHosts[1].UUID,
			CommandUUID: uuid1,
			Status:      "Pending",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[1].Hostname,
			TeamID:      nil,
		},
	})

	// simulate a result for enrolledHosts[0]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[0].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid1,
		Status:      "Acknowledged",
		RequestType: "ListApps",
		Raw:         []byte(rawCmd1),
	})
	require.NoError(t, err)

	// command is now listed with a status for this result
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 2)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.NotZero(t, res[1].UpdatedAt)
	res[1].UpdatedAt = time.Time{}

	require.ElementsMatch(t, res, []*fleet.MDMAppleCommand{
		{
			DeviceID:    enrolledHosts[0].UUID,
			CommandUUID: uuid1,
			Status:      "Acknowledged",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[0].Hostname,
			TeamID:      nil,
		},
		{
			DeviceID:    enrolledHosts[1].UUID,
			CommandUUID: uuid1,
			Status:      "Pending",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[1].Hostname,
			TeamID:      nil,
		},
	})

	// simulate a result for enrolledHosts[1]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[1].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid1,
		Status:      "Error",
		RequestType: "ListApps",
		Raw:         []byte(rawCmd1),
	})
	require.NoError(t, err)

	// both results are now listed
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 2)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.NotZero(t, res[1].UpdatedAt)
	res[1].UpdatedAt = time.Time{}

	require.ElementsMatch(t, res, []*fleet.MDMAppleCommand{
		{
			DeviceID:    enrolledHosts[0].UUID,
			CommandUUID: uuid1,
			Status:      "Acknowledged",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[0].Hostname,
			TeamID:      nil,
		},
		{
			DeviceID:    enrolledHosts[1].UUID,
			CommandUUID: uuid1,
			Status:      "Error",
			RequestType: "ListApps",
			Hostname:    enrolledHosts[1].Hostname,
			TeamID:      nil,
		},
	})

	// enqueue another command for enrolled hosts [1] and [2]
	uuid2 := uuid.New().String()
	rawCmd2 := createRawCmd("InstallApp", uuid2)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[1].UUID, enrolledHosts[2].UUID}, rawCmd2)
	require.NoError(t, err)

	// simulate a result for enrolledHosts[1] and [2]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[1].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "InstallApp",
		Raw:         []byte(rawCmd2),
	})
	require.NoError(t, err)
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[2].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "InstallApp",
		Raw:         []byte(rawCmd2),
	})
	require.NoError(t, err)

	// results are listed
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 4)

	// page-by-page: first page
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{
		ListOptions: fleet.ListOptions{Page: 0, PerPage: 3, OrderKey: "device_id", OrderDirection: fleet.OrderDescending},
	})
	require.NoError(t, err)
	require.Len(t, res, 3)

	// page-by-page: second page
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMAppleCommandListOptions{
		ListOptions: fleet.ListOptions{Page: 1, PerPage: 3, OrderKey: "device_id", OrderDirection: fleet.OrderDescending},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)

	// filter by a user from team tm1, can only see that team's host
	u1, err := ds.NewUser(ctx, &fleet.User{
		Password:   []byte("garbage"),
		Salt:       "garbage",
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{Team: *tm1, Role: fleet.RoleObserver},
		},
	})
	require.NoError(t, err)
	u1, err = ds.UserByID(ctx, u1.ID)
	require.NoError(t, err)

	// u1 is an observer, so if IncludeObserver is not set, returns nothing
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: u1}, &fleet.MDMAppleCommandListOptions{
		ListOptions: fleet.ListOptions{PerPage: 3},
	})
	require.NoError(t, err)
	require.Len(t, res, 0)

	// now with IncludeObserver set to true
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: u1, IncludeObserver: true}, &fleet.MDMAppleCommandListOptions{
		ListOptions: fleet.ListOptions{PerPage: 3, OrderKey: "updated_at", OrderDirection: fleet.OrderDescending},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}
	require.ElementsMatch(t, res, []*fleet.MDMAppleCommand{
		{
			DeviceID:    enrolledHosts[2].UUID,
			CommandUUID: uuid2,
			Status:      "Acknowledged",
			RequestType: "InstallApp",
			Hostname:    enrolledHosts[2].Hostname,
			TeamID:      &tm1.ID,
		},
	})
}

func testMDMAppleEULA(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	eula := &fleet.MDMAppleEULA{
		Token: uuid.New().String(),
		Name:  "eula.pdf",
		Bytes: []byte("contents"),
	}

	err := ds.MDMAppleInsertEULA(ctx, eula)
	require.NoError(t, err)

	var ae fleet.AlreadyExistsError
	err = ds.MDMAppleInsertEULA(ctx, eula)
	require.ErrorAs(t, err, &ae)

	gotEULA, err := ds.MDMAppleGetEULAMetadata(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gotEULA.CreatedAt)
	require.Equal(t, eula.Token, gotEULA.Token)
	require.Equal(t, eula.Name, gotEULA.Name)

	gotEULABytes, err := ds.MDMAppleGetEULABytes(ctx, eula.Token)
	require.NoError(t, err)
	require.EqualValues(t, eula.Bytes, gotEULABytes.Bytes)
	require.Equal(t, eula.Name, gotEULABytes.Name)

	err = ds.MDMAppleDeleteEULA(ctx, eula.Token)
	require.NoError(t, err)

	var nfe fleet.NotFoundError
	_, err = ds.MDMAppleGetEULAMetadata(ctx)
	require.ErrorAs(t, err, &nfe)
	_, err = ds.MDMAppleGetEULABytes(ctx, eula.Token)
	require.ErrorAs(t, err, &nfe)
	err = ds.MDMAppleDeleteEULA(ctx, eula.Token)
	require.ErrorAs(t, err, &nfe)

	err = ds.MDMAppleInsertEULA(ctx, eula)
	require.NoError(t, err)
}

func testMDMAppleSetupAssistant(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get non-existing
	_, err := ds.GetMDMAppleSetupAssistant(ctx, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// create for no team
	noTeamAsst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{Name: "test", Profile: json.RawMessage("{}")})
	require.NoError(t, err)
	require.NotZero(t, noTeamAsst.ID)
	require.NotZero(t, noTeamAsst.UploadedAt)
	require.Nil(t, noTeamAsst.TeamID)
	require.Equal(t, "test", noTeamAsst.Name)
	require.Equal(t, "{}", string(noTeamAsst.Profile))

	// get for no team returns the same data
	getAsst, err := ds.GetMDMAppleSetupAssistant(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, noTeamAsst, getAsst)

	// create for non-existing team fails
	_, err = ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: ptr.Uint(123), Name: "test", Profile: json.RawMessage("{}")})
	require.Error(t, err)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm"})
	require.NoError(t, err)

	// create for existing team
	tmAsst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test", Profile: json.RawMessage("{}")})
	require.NoError(t, err)
	require.NotZero(t, tmAsst.ID)
	require.NotZero(t, tmAsst.UploadedAt)
	require.NotNil(t, tmAsst.TeamID)
	require.Equal(t, tm.ID, *tmAsst.TeamID)
	require.Equal(t, "test", tmAsst.Name)
	require.Equal(t, "{}", string(tmAsst.Profile))

	// get for team returns the same data
	getAsst, err = ds.GetMDMAppleSetupAssistant(ctx, &tm.ID)
	require.NoError(t, err)
	require.Equal(t, tmAsst, getAsst)

	// upsert team
	tmAsst2, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":2}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst2.ID, tmAsst.ID)
	require.False(t, tmAsst2.UploadedAt.Before(tmAsst.UploadedAt)) // after or equal
	require.Equal(t, tmAsst.TeamID, tmAsst2.TeamID)
	require.Equal(t, "test2", tmAsst2.Name)
	require.JSONEq(t, `{"x": 2}`, string(tmAsst2.Profile))

	// upsert no team
	noTeamAsst2, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{Name: "test3", Profile: json.RawMessage(`{"x": 3}`)})
	require.NoError(t, err)
	require.Equal(t, noTeamAsst2.ID, noTeamAsst.ID)
	require.False(t, noTeamAsst2.UploadedAt.Before(noTeamAsst.UploadedAt)) // after or equal
	require.Nil(t, noTeamAsst2.TeamID)
	require.Equal(t, "test3", noTeamAsst2.Name)
	require.JSONEq(t, `{"x": 3}`, string(noTeamAsst2.Profile))

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert team no change, uploaded at timestamp does not change
	tmAsst3, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":2}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst2, tmAsst3)

	// set a profile uuid for the team assistant
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, "abcd")
	require.NoError(t, err)

	// get for team returns the same data, but now with a profile uuid
	getAsst, err = ds.GetMDMAppleSetupAssistant(ctx, &tm.ID)
	require.NoError(t, err)
	require.Equal(t, "abcd", getAsst.ProfileUUID)
	getAsst.ProfileUUID = ""
	require.Equal(t, tmAsst3, getAsst)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert again the team with no change, uploaded at timestamp does not change nor does the profile uuid
	tmAsst4, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":2}`)})
	require.NoError(t, err)
	require.Equal(t, "abcd", tmAsst4.ProfileUUID)
	tmAsst4.ProfileUUID = ""
	require.Equal(t, tmAsst3, tmAsst4)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert team with a change, clears the profile uuid and updates the uploaded at timestamp
	tmAsst5, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":3}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst4.ID, tmAsst5.ID)
	require.True(t, tmAsst5.UploadedAt.After(tmAsst4.UploadedAt))
	require.Equal(t, tmAsst4.TeamID, tmAsst5.TeamID)
	require.Equal(t, "test2", tmAsst5.Name)
	require.Empty(t, tmAsst5.ProfileUUID)
	require.JSONEq(t, `{"x": 3}`, string(tmAsst5.Profile))

	// set a profile uuid for the team assistant
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, "efgh")
	require.NoError(t, err)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert again the team with no change
	tmAsst6, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":3}`)})
	require.NoError(t, err)
	require.Equal(t, "efgh", tmAsst6.ProfileUUID)
	tmAsst6.ProfileUUID = ""
	require.Equal(t, tmAsst5, tmAsst6)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert team with a name change
	tmAsst7, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test3", Profile: json.RawMessage(`{"x":3}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst6.ID, tmAsst7.ID)
	require.True(t, tmAsst7.UploadedAt.After(tmAsst6.UploadedAt))
	require.Equal(t, tmAsst6.TeamID, tmAsst7.TeamID)
	require.Equal(t, "test3", tmAsst7.Name)
	require.Empty(t, tmAsst7.ProfileUUID)
	require.JSONEq(t, `{"x": 3}`, string(tmAsst7.Profile))

	// delete no team
	err = ds.DeleteMDMAppleSetupAssistant(ctx, nil)
	require.NoError(t, err)

	// delete the team, which will cascade delete the setup assistant
	err = ds.DeleteTeam(ctx, tm.ID)
	require.NoError(t, err)

	// get the team assistant
	_, err = ds.GetMDMAppleSetupAssistant(ctx, &tm.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// delete the team assistant, no error if it doesn't exist
	err = ds.DeleteMDMAppleSetupAssistant(ctx, &tm.ID)
	require.NoError(t, err)
}

func testMDMAppleEnrollmentProfile(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	_, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	_, err = ds.GetMDMAppleEnrollmentProfileByToken(ctx, "abcd")
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// add a new automatic enrollment profile
	rawMsg := json.RawMessage(`{"allow_pairing": true}`)
	profAuto, err := ds.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       "automatic",
		DEPProfile: &rawMsg,
		Token:      "abcd",
	})
	require.NoError(t, err)
	require.NotZero(t, profAuto.ID)

	// add a new manual enrollment profile
	profMan, err := ds.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       "manual",
		DEPProfile: &rawMsg,
		Token:      "efgh",
	})
	require.NoError(t, err)
	require.NotZero(t, profMan.ID)

	profs, err := ds.ListMDMAppleEnrollmentProfiles(ctx)
	require.NoError(t, err)
	require.Len(t, profs, 2)

	tokens := make([]string, 2)
	for i, p := range profs {
		tokens[i] = p.Token
	}
	require.ElementsMatch(t, []string{"abcd", "efgh"}, tokens)

	// get the automatic profile by type
	getProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	require.NoError(t, err)
	getProf.UpdateCreateTimestamps = fleet.UpdateCreateTimestamps{}
	require.Equal(t, profAuto, getProf)

	// get the manual profile by token
	getProf, err = ds.GetMDMAppleEnrollmentProfileByToken(ctx, "efgh")
	require.NoError(t, err)
	getProf.UpdateCreateTimestamps = fleet.UpdateCreateTimestamps{}
	require.Equal(t, profMan, getProf)
}

func testListMDMAppleSerials(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a mix of DEP-enrolled hosts, non-Fleet-MDM, pending DEP-enrollment
	hosts := make([]*fleet.Host, 10)
	for i := 0; i < len(hosts); i++ {
		serial := fmt.Sprintf("serial-%d", i)
		if i == 9 {
			serial = ""
		}
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:       fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID:  ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:        ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			Platform:       "darwin",
			HardwareSerial: serial,
		})
		require.NoError(t, err)
		switch {
		case i <= 3:
			// dep-enrolled in fleet
			err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet)
		case i == 4:
			// pending dep enrollment in fleet
			err = ds.SetOrUpdateMDMData(ctx, h.ID, false, false, "https://example.com", true, fleet.WellKnownMDMFleet)
		case i == 5:
			// manually enrolled in fleet
			err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet)
		case i == 6:
			// dep enrolled in fleet but is a server
			err = ds.SetOrUpdateMDMData(ctx, h.ID, true, true, "https://example.com", true, fleet.WellKnownMDMFleet)
		case i == 7:
			// dep enrolled in non-Fleet
			err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM)
		case i == 8:
			// not mdm-enrolled at all
			err = nil
		case i == 9:
			// dep-enrolled in fleet, but empty serial so not returned
			err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet)
		}
		require.NoError(t, err)
		if i <= 3 {
			nanoEnroll(t, ds, h, false)
		}
		hosts[i] = h
		t.Logf("host [%d]: %s - %s", i, h.UUID, h.HardwareSerial)
	}

	// create teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// assign hosts[2,7,8] to tm1
	err = ds.AddHostsToTeam(ctx, &tm1.ID, []uint{hosts[2].ID, hosts[7].ID, hosts[8].ID})
	require.NoError(t, err)

	// list serials in team 2, has none
	serials, err := ds.ListMDMAppleDEPSerialsInTeam(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in team 1, has one (hosts[2])
	serials, err = ds.ListMDMAppleDEPSerialsInTeam(ctx, &tm1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-2"}, serials)

	// list serials in no-team, has 4 (hosts[0,1,3,4]), hosts[4] is pending, the others enrolled
	serials, err = ds.ListMDMAppleDEPSerialsInTeam(ctx, nil)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-3", "serial-4"}, serials)

	// list serials with no host IDs returns empty
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in hosts[0,1,2,3,4] returns all of them
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-2", "serial-3", "serial-4"}, serials)

	// list serials in hosts[5,6,7,8,9] returns none
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{hosts[5].ID, hosts[6].ID, hosts[7].ID, hosts[8].ID, hosts[9].ID})
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in all hosts returns [0-4]
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{
		hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID,
		hosts[5].ID, hosts[6].ID, hosts[7].ID, hosts[8].ID, hosts[9].ID,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-2", "serial-3", "serial-4"}, serials)
}

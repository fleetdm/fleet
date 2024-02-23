package mysql

import (
	"context"
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanodep/tokenpki"
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
		{"TestNewMDMAppleConfigProfileLabels", testNewMDMAppleConfigProfileLabels},
		{"TestNewMDMAppleConfigProfileDuplicateIdentifier", testNewMDMAppleConfigProfileDuplicateIdentifier},
		{"TestDeleteMDMAppleConfigProfile", testDeleteMDMAppleConfigProfile},
		{"TestDeleteMDMAppleConfigProfileByTeamAndIdentifier", testDeleteMDMAppleConfigProfileByTeamAndIdentifier},
		{"TestListMDMAppleConfigProfiles", testListMDMAppleConfigProfiles},
		{"TestHostDetailsMDMProfiles", testHostDetailsMDMProfiles},
		{"TestBatchSetMDMAppleProfiles", testBatchSetMDMAppleProfiles},
		{"TestMDMAppleProfileManagement", testMDMAppleProfileManagement},
		{"TestMDMAppleProfileManagementBatch2", testMDMAppleProfileManagementBatch2},
		{"TestMDMAppleProfileManagementBatch3", testMDMAppleProfileManagementBatch3},
		{"TestGetMDMAppleProfilesContents", testGetMDMAppleProfilesContents},
		{"TestAggregateMacOSSettingsStatusWithFileVault", testAggregateMacOSSettingsStatusWithFileVault},
		{"TestMDMAppleHostsProfilesStatus", testMDMAppleHostsProfilesStatus},
		{"TestMDMAppleIdPAccount", testMDMAppleIdPAccount},
		{"TestIgnoreMDMClientError", testDoNotIgnoreMDMClientError},
		{"TestDeleteMDMAppleProfilesForHost", testDeleteMDMAppleProfilesForHost},
		{"TestGetMDMAppleCommandResults", testGetMDMAppleCommandResults},
		{"TestBulkUpsertMDMAppleConfigProfiles", testBulkUpsertMDMAppleConfigProfile},
		{"TestMDMAppleBootstrapPackageCRUD", testMDMAppleBootstrapPackageCRUD},
		{"TestListMDMAppleCommands", testListMDMAppleCommands},
		{"TestMDMAppleSetupAssistant", testMDMAppleSetupAssistant},
		{"TestMDMAppleEnrollmentProfile", testMDMAppleEnrollmentProfile},
		{"TestListMDMAppleSerials", testListMDMAppleSerials},
		{"TestMDMAppleDefaultSetupAssistant", testMDMAppleDefaultSetupAssistant},
		{"TestSetVerifiedMacOSProfiles", testSetVerifiedMacOSProfiles},
		{"TestMDMAppleConfigProfileHash", testMDMAppleConfigProfileHash},
		{"TestResetMDMAppleEnrollment", testResetMDMAppleEnrollment},
		{"TestMDMAppleDeleteHostDEPAssignments", testMDMAppleDeleteHostDEPAssignments},
		{"CleanMacOSMDMLock", testCleanMacOSMDMLock},
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

	// create a couple Apple profiles for no-team
	profA, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 0))
	require.NoError(t, err)
	require.NotZero(t, profA.ProfileID)
	require.NotEmpty(t, profA.ProfileUUID)
	require.Equal(t, "a", string(profA.ProfileUUID[0]))
	profB, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 0))
	require.NoError(t, err)
	require.NotZero(t, profB.ProfileID)
	require.NotEmpty(t, profB.ProfileUUID)
	require.Equal(t, "a", string(profB.ProfileUUID[0]))
	// create a Windows profile for no-team
	profC, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "c", TeamID: nil, SyncML: []byte("<Replace></Replace>")})
	require.NoError(t, err)
	require.NotEmpty(t, profC.ProfileUUID)
	require.Equal(t, "w", string(profC.ProfileUUID[0]))

	// create the same name for team 1 as Apple profile
	profATm, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 1))
	require.NoError(t, err)
	require.NotZero(t, profATm.ProfileID)
	require.NotEmpty(t, profATm.ProfileUUID)
	require.Equal(t, "a", string(profATm.ProfileUUID[0]))
	require.NotNil(t, profATm.TeamID)
	require.Equal(t, uint(1), *profATm.TeamID)
	// create the same B profile for team 1 as Windows profile
	profBTm, err := ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "b", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")})
	require.NoError(t, err)
	require.NotEmpty(t, profBTm.ProfileUUID)

	var existsErr *existsError
	// create a duplicate of Apple for no-team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 0))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Windows for no-team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("c", "c", 0))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Apple for team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 1))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate of Windows for team
	_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 1))
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with a Windows profile for no-team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: nil, SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
	// create a duplicate name with a Windows profile for team
	_, err = ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "a", TeamID: ptr.Uint(1), SyncML: []byte("<Replace></Replace>")})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)
}

func testNewMDMAppleConfigProfileLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	dummyMC := mobileconfig.Mobileconfig([]byte("DummyTestMobileconfigBytes"))
	cp := fleet.MDMAppleConfigProfile{
		Name:         "DummyTestName",
		Identifier:   "DummyTestIdentifier",
		Mobileconfig: dummyMC,
		TeamID:       nil,
		Labels: []fleet.ConfigurationProfileLabel{
			{LabelName: "foo", LabelID: 1},
		},
	}
	_, err := ds.NewMDMAppleConfigProfile(ctx, cp)
	require.NotNil(t, err)
	require.True(t, fleet.IsForeignKey(err))

	label := &fleet.Label{
		Name:        "my label",
		Description: "a label",
		Query:       "select 1 from processes;",
		Platform:    "darwin",
	}
	label, err = ds.NewLabel(ctx, label)
	require.NoError(t, err)
	cp.Labels = []fleet.ConfigurationProfileLabel{
		{LabelName: label.Name, LabelID: label.ID},
	}
	prof, err := ds.NewMDMAppleConfigProfile(ctx, cp)
	require.NoError(t, err)
	require.NotEmpty(t, prof.ProfileUUID)
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

	// get it back from both the deprecated ID and the uuid methods
	storedCP, err := ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, newCP.ProfileID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)
	require.Nil(t, storedCP.Labels)
	storedCP, err = ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileUUID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)
	require.Nil(t, storedCP.Labels)

	// create a label-based profile
	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "lbl", Query: "select 1"})
	require.NoError(t, err)

	labelCP := fleet.MDMAppleConfigProfile{
		Name:         "label-based",
		Identifier:   "label-based",
		Mobileconfig: mobileconfig.Mobileconfig([]byte("LabelTestMobileconfigBytes")),
		Labels: []fleet.ConfigurationProfileLabel{
			{LabelName: lbl.Name, LabelID: lbl.ID},
		},
	}
	labelProf, err := ds.NewMDMAppleConfigProfile(ctx, labelCP)
	require.NoError(t, err)

	// get it back from both the deprecated ID and the uuid methods, labels are
	// only included in the uuid one
	prof, err := ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, labelProf.ProfileID)
	require.NoError(t, err)
	require.Nil(t, prof.Labels)
	prof, err = ds.GetMDMAppleConfigProfile(ctx, labelProf.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.Labels, 1)
	require.Equal(t, lbl.Name, prof.Labels[0].LabelName)
	require.False(t, prof.Labels[0].Broken)

	// break the profile by deleting the label
	require.NoError(t, ds.DeleteLabel(ctx, lbl.Name))

	prof, err = ds.GetMDMAppleConfigProfile(ctx, labelProf.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.Labels, 1)
	require.Equal(t, lbl.Name, prof.Labels[0].LabelName)
	require.True(t, prof.Labels[0].Broken)
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
	checkConfigProfileWithChecksum(t, *expectedTeam0[0], *cps[0])

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
	checkConfigProfileWithChecksum(t, *expectedTeam1[0], *cps[0])

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
			checkConfigProfileWithChecksum(t, *expectedTeam1[0], *cp)
		case "another_name1":
			checkConfigProfileWithChecksum(t, *expectedTeam1[1], *cp)
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

	// first via the deprecated ID
	initialCP := storeDummyConfigProfileForTest(t, ds)
	err := ds.DeleteMDMAppleConfigProfileByDeprecatedID(ctx, initialCP.ProfileID)
	require.NoError(t, err)
	_, err = ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, initialCP.ProfileID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.DeleteMDMAppleConfigProfileByDeprecatedID(ctx, initialCP.ProfileID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// next via the uuid
	initialCP = storeDummyConfigProfileForTest(t, ds)
	err = ds.DeleteMDMAppleConfigProfile(ctx, initialCP.ProfileUUID)
	require.NoError(t, err)
	_, err = ds.GetMDMAppleConfigProfile(ctx, initialCP.ProfileUUID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.DeleteMDMAppleConfigProfile(ctx, initialCP.ProfileUUID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testDeleteMDMAppleConfigProfileByTeamAndIdentifier(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)

	err := ds.DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx, initialCP.TeamID, initialCP.Identifier)
	require.NoError(t, err)

	_, err = ds.GetMDMAppleConfigProfile(ctx, initialCP.ProfileUUID)
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
	storedCP, err := ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileUUID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)

	return storedCP
}

func checkConfigProfile(t *testing.T, expected, actual fleet.MDMAppleConfigProfile) {
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Identifier, actual.Identifier)
	require.Equal(t, expected.Mobileconfig, actual.Mobileconfig)
	if !expected.UploadedAt.IsZero() {
		require.True(t, expected.UploadedAt.Equal(actual.UploadedAt))
	}
}

func checkConfigProfileWithChecksum(t *testing.T, expected, actual fleet.MDMAppleConfigProfile) {
	checkConfigProfile(t, expected, actual)
	require.ElementsMatch(t, md5.Sum(expected.Mobileconfig), actual.Checksum) // nolint:gosec // used only to hash for efficient comparisons
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
	gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h0.UUID)
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
	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)

	expectedProfiles0 := map[string]fleet.HostMDMAppleProfile{
		p0.ProfileUUID: {HostUUID: h0.UUID, Name: p0.Name, ProfileUUID: p0.ProfileUUID, CommandUUID: "cmd0-uuid", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, Detail: ""},
		p1.ProfileUUID: {HostUUID: h0.UUID, Name: p1.Name, ProfileUUID: p1.ProfileUUID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall, Detail: ""},
		p2.ProfileUUID: {HostUUID: h0.UUID, Name: p2.Name, ProfileUUID: p2.ProfileUUID, CommandUUID: "cmd2-uuid", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Detail: "Error removing profile"},
	}

	expectedProfiles1 := map[string]fleet.HostMDMAppleProfile{
		p0.ProfileUUID: {HostUUID: h1.UUID, Name: p0.Name, ProfileUUID: p0.ProfileUUID, CommandUUID: "cmd0-uuid", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, Detail: "Error installing profile"},
		p1.ProfileUUID: {HostUUID: h1.UUID, Name: p1.Name, ProfileUUID: p1.ProfileUUID, CommandUUID: "cmd1-uuid", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall, Detail: ""},
		p2.ProfileUUID: {HostUUID: h1.UUID, Name: p2.Name, ProfileUUID: p2.ProfileUUID, CommandUUID: "cmd2-uuid", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Detail: "Error removing profile"},
	}

	var args []interface{}
	for _, p := range expectedProfiles0 {
		args = append(args, p.HostUUID, p.ProfileUUID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}
	for _, p := range expectedProfiles1 {
		args = append(args, p.HostUUID, p.ProfileUUID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
	INSERT INTO host_mdm_apple_profiles (
		host_uuid, profile_uuid, command_uuid, status, operation_type, detail, profile_name)
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

	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h0.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 3)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles0[gp.ProfileUUID]
		require.True(t, ok)
		require.Equal(t, ep.Name, gp.Name)
		require.Equal(t, *ep.Status, *gp.Status)
		require.Equal(t, ep.OperationType, gp.OperationType)
		require.Equal(t, ep.Detail, gp.Detail)
	}

	gotHost, err = ds.Host(ctx, h1.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles) // ds.Host never returns MDM profiles

	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 3)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles1[gp.ProfileUUID]
		require.True(t, ok)
		require.Equal(t, ep.Name, gp.Name)
		require.Equal(t, *ep.Status, *gp.Status)
		require.Equal(t, ep.OperationType, gp.OperationType)
		require.Equal(t, ep.Detail, gp.Detail)
	}

	// mark h1's install+failed profile as install+pending
	h1InstallFailed := expectedProfiles1[p0.ProfileUUID]
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      h1InstallFailed.HostUUID,
		CommandUUID:   h1InstallFailed.CommandUUID,
		ProfileUUID:   h1InstallFailed.ProfileUUID,
		Name:          h1InstallFailed.Name,
		Status:        &fleet.MDMDeliveryPending,
		OperationType: fleet.MDMOperationTypeInstall,
		Detail:        "",
	})
	require.NoError(t, err)

	// mark h1's remove+failed profile as remove+verifying, deletes the host profile row
	h1RemoveFailed := expectedProfiles1[p2.ProfileUUID]
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      h1RemoveFailed.HostUUID,
		CommandUUID:   h1RemoveFailed.CommandUUID,
		ProfileUUID:   h1RemoveFailed.ProfileUUID,
		Name:          h1RemoveFailed.Name,
		Status:        &fleet.MDMDeliveryVerifying,
		OperationType: fleet.MDMOperationTypeRemove,
		Detail:        "",
	})
	require.NoError(t, err)

	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 2) // remove+verifying is not there anymore

	h1InstallPending := h1InstallFailed
	h1InstallPending.Status = &fleet.MDMDeliveryPending
	h1InstallPending.Detail = ""
	expectedProfiles1[p0.ProfileUUID] = h1InstallPending
	delete(expectedProfiles1, p2.ProfileUUID)
	for _, gp := range gotProfs {
		ep, ok := expectedProfiles1[gp.ProfileUUID]
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
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                      // ingested; new serial, macOS, "added" op type
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                      // not ingested; duplicate serial
		{SerialNumber: hosts[0].HardwareSerial, Model: "MacBook Pro", OS: "OSX", OpType: "added"},    // not ingested; existing serial
		{SerialNumber: "ijk", Model: "MacBook Pro", OS: "", OpType: "added"},                         // ingested; empty OS
		{SerialNumber: "tuv", Model: "MacBook Pro", OS: "OSX", OpType: "modified"},                   // ingested; op type "modified", but new serial
		{SerialNumber: hosts[1].HardwareSerial, Model: "MacBook Pro", OS: "OSX", OpType: "modified"}, // not ingested; op type "modified", existing serial
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "updated"},                    // not ingested; op type "updated"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "deleted"},                    // not ingested; op type "deleted"
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},                      // ingested; new serial, macOS, "added" op type
	}
	wantSerials = append(wantSerials, "abc", "xyz", "ijk", "tuv")

	n, tmID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices)
	require.NoError(t, err)
	require.EqualValues(t, 4, n) // 4 new hosts ("abc", "xyz", "ijk", "tuv")
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
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
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
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "https://fleetdm.com", true, "Fleet MDM", "")
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
			ProfileUUID:       profiles[0].ProfileUUID,
			ProfileIdentifier: profiles[0].Identifier,
			ProfileName:       profiles[0].Name,
			HostUUID:          testUUID,
			Status:            &fleet.MDMDeliveryVerifying,
			OperationType:     fleet.MDMOperationTypeInstall,
			CommandUUID:       "command-uuid",
			Checksum:          []byte("csum"),
		},
	},
	)
	require.NoError(t, err)

	hostProfs, err := ds.GetHostMDMAppleProfiles(ctx, testUUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, len(profiles))

	var hostID uint
	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &hostID, `SELECT id  FROM hosts WHERE uuid = ?`, testUUID)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hostID, "asdf", "", nil)
	require.NoError(t, err)

	key, err := ds.GetHostDiskEncryptionKey(ctx, hostID)
	require.NoError(t, err)
	require.NotNil(t, key)

	// check that an entry in host_mdm exists
	var count int
	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &count, `SELECT COUNT(*) FROM host_mdm WHERE host_id = (SELECT id FROM hosts WHERE uuid = ?)`, testUUID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = ds.UpdateHostTablesOnMDMUnenroll(ctx, testUUID)
	require.NoError(t, err)

	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &count, `SELECT COUNT(*) FROM host_mdm WHERE host_id = ?`, testUUID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	hostProfs, err = ds.GetHostMDMAppleProfiles(ctx, testUUID)
	require.NoError(t, err)
	require.Empty(t, hostProfs)
	key, err = ds.GetHostDiskEncryptionKey(ctx, hostID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Nil(t, key)
}

func expectAppleProfiles(
	t *testing.T,
	ds *Datastore,
	tmID *uint,
	want []*fleet.MDMAppleConfigProfile,
) map[string]string {
	if tmID == nil {
		tmID = ptr.Uint(0)
	}
	// don't use ds.ListMDMAppleConfigProfiles as it leaves out
	// fleet-managed profiles.
	var got []*fleet.MDMAppleConfigProfile
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		ctx := context.Background()
		return sqlx.SelectContext(ctx, q, &got, `SELECT * FROM mdm_apple_configuration_profiles WHERE team_id = ?`, tmID)
	})

	// create map of expected profiles keyed by identifier
	wantMap := make(map[string]*fleet.MDMAppleConfigProfile, len(want))
	for _, cp := range want {
		wantMap[cp.Identifier] = cp
	}

	// compare only the fields we care about, and build the resulting map of
	// profile identifier as key to profile UUID as value
	m := make(map[string]string)
	for _, gotp := range got {
		m[gotp.Identifier] = gotp.ProfileUUID
		if gotp.TeamID != nil && *gotp.TeamID == 0 {
			gotp.TeamID = nil
		}

		// ProfileID is non-zero (auto-increment), but otherwise we don't care
		// about it for test assertions.
		require.NotZero(t, gotp.ProfileID)
		gotp.ProfileID = 0

		// ProfileUUID is non-empty and starts with "a", but otherwise we don't
		// care about it for test assertions.
		require.NotEmpty(t, gotp.ProfileUUID)
		require.True(t, strings.HasPrefix(gotp.ProfileUUID, "a"))
		gotp.ProfileUUID = ""

		gotp.CreatedAt = time.Time{}

		// if an expected uploaded_at timestamp is provided for this profile, keep
		// its value, otherwise clear it as we don't care about asserting its
		// value.
		if wantp := wantMap[gotp.Identifier]; wantp == nil || wantp.UploadedAt.IsZero() {
			gotp.UploadedAt = time.Time{}
		}
	}
	// order is not guaranteed
	require.ElementsMatch(t, want, got)
	return m
}

func testBatchSetMDMAppleProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	applyAndExpect := func(newSet []*fleet.MDMAppleConfigProfile, tmID *uint, want []*fleet.MDMAppleConfigProfile) map[string]string {
		err := ds.BatchSetMDMAppleProfiles(ctx, tmID, newSet)
		require.NoError(t, err)
		return expectAppleProfiles(t, ds, tmID, want)
	}
	getProfileByTeamAndIdentifier := func(tmID *uint, identifier string) *fleet.MDMAppleConfigProfile {
		var prof fleet.MDMAppleConfigProfile
		var teamID uint
		if tmID != nil {
			teamID = *tmID
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &prof,
				`SELECT * FROM mdm_apple_configuration_profiles WHERE team_id = ? AND identifier = ?`,
				teamID, identifier)
		})
		return &prof
	}

	withTeamID := func(p *fleet.MDMAppleConfigProfile, tmID uint) *fleet.MDMAppleConfigProfile {
		p.TeamID = &tmID
		return p
	}
	withUploadedAt := func(p *fleet.MDMAppleConfigProfile, ua time.Time) *fleet.MDMAppleConfigProfile {
		p.UploadedAt = ua
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
	profTm1I1 := getProfileByTeamAndIdentifier(ptr.Uint(1), "I1")

	// apply single profile set for no-team
	mNoTm := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	}, nil, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "b"),
	})
	profNoTmI1 := getProfileByTeamAndIdentifier(nil, "I1")

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// apply new profile set for tm1
	mTm1b := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "a"), // unchanged
		configProfileForTest(t, "N2", "I2", "b"),
	}, ptr.Uint(1), []*fleet.MDMAppleConfigProfile{
		withUploadedAt(withTeamID(configProfileForTest(t, "N1", "I1", "a"), 1), profTm1I1.UploadedAt),
		withTeamID(configProfileForTest(t, "N2", "I2", "b"), 1),
	})
	// identifier for N1-I1 is unchanged
	require.Equal(t, mTm1["I1"], mTm1b["I1"])
	profTm1I2 := getProfileByTeamAndIdentifier(ptr.Uint(1), "I2")

	// apply edited (by name only) profile set for no-team
	mNoTmb := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N2", "I1", "b"),
	}, nil, []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N2", "I1", "b"), // name change implies uploaded_at change
	})
	require.Equal(t, mNoTm["I1"], mNoTmb["I1"])

	profNoTmI1b := getProfileByTeamAndIdentifier(nil, "I1")
	require.False(t, profNoTmI1.UploadedAt.Equal(profNoTmI1b.UploadedAt))

	// wait a second to ensure timestamps in the DB change
	time.Sleep(time.Second)

	// apply edited profile (by content only), unchanged profile and new profile
	// for tm1
	mTm1c := applyAndExpect([]*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "N1", "I1", "z"), // content updated
		configProfileForTest(t, "N2", "I2", "b"), // unchanged
		configProfileForTest(t, "N3", "I3", "c"), // new
	}, ptr.Uint(1), []*fleet.MDMAppleConfigProfile{
		withTeamID(configProfileForTest(t, "N1", "I1", "z"), 1),
		withUploadedAt(withTeamID(configProfileForTest(t, "N2", "I2", "b"), 1), profTm1I2.UploadedAt),
		withTeamID(configProfileForTest(t, "N3", "I3", "c"), 1),
	})
	// identifier for N1-I1 is unchanged
	require.Equal(t, mTm1b["I1"], mTm1c["I1"])
	// identifier for N2-I2 is unchanged
	require.Equal(t, mTm1b["I2"], mTm1c["I2"])

	profTm1I1c := getProfileByTeamAndIdentifier(ptr.Uint(1), "I1")
	// uploaded-at was modified because the content changed
	require.False(t, profTm1I1.UploadedAt.Equal(profTm1I1c.UploadedAt))

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

func configProfileBytesForTest(name, identifier, uuid string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
}

func configProfileForTest(t *testing.T, name, identifier, uuid string, labels ...*fleet.Label) *fleet.MDMAppleConfigProfile {
	prof := configProfileBytesForTest(name, identifier, uuid)
	cp, err := fleet.NewMDMAppleConfigProfile(prof, nil)
	require.NoError(t, err)
	sum := md5.Sum(prof) // nolint:gosec // used only to hash for efficient comparisons
	cp.Checksum = sum[:]

	for _, lbl := range labels {
		cp.Labels = append(cp.Labels, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
	}

	return cp
}

func teamConfigProfileForTest(t *testing.T, name, identifier, uuid string, teamID uint) *fleet.MDMAppleConfigProfile {
	prof := configProfileBytesForTest(name, identifier, uuid)
	cp, err := fleet.NewMDMAppleConfigProfile(configProfileBytesForTest(name, identifier, uuid), &teamID)
	require.NoError(t, err)
	sum := md5.Sum(prof) // nolint:gosec // used only to hash for efficient comparisons
	cp.Checksum = sum[:]
	return cp
}

func testMDMAppleProfileManagementBatch2(t *testing.T, ds *Datastore) {
	ds.testUpsertMDMDesiredProfilesBatchSize = 2
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
	})
	testMDMAppleProfileManagement(t, ds)
}

func testMDMAppleProfileManagementBatch3(t *testing.T, ds *Datastore) {
	ds.testUpsertMDMDesiredProfilesBatchSize = 3
	t.Cleanup(func() {
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
	})
	testMDMAppleProfileManagement(t, ds)
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

	_, err = ds.writer(ctx).Exec(`
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
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
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
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
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
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2"},
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
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2"},
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-3"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-3"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-3"},
	}, profiles)

	// cron runs and updates the status
	err = ds.BulkUpsertMDMAppleHostProfiles(
		ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       globalPfs[0].ProfileUUID,
				ProfileIdentifier: globalPfs[0].Identifier,
				ProfileName:       globalPfs[0].Name,
				Checksum:          globalProfiles[0].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       globalPfs[0].ProfileUUID,
				ProfileIdentifier: globalPfs[0].Identifier,
				ProfileName:       globalPfs[0].Name,
				Checksum:          globalProfiles[0].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       globalPfs[1].ProfileUUID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				Checksum:          globalProfiles[1].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       globalPfs[1].ProfileUUID,
				ProfileIdentifier: globalPfs[1].Identifier,
				ProfileName:       globalPfs[1].Name,
				Checksum:          globalProfiles[1].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       globalPfs[2].ProfileUUID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				Checksum:          globalProfiles[2].Checksum,
				HostUUID:          "test-uuid-1",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       globalPfs[2].ProfileUUID,
				ProfileIdentifier: globalPfs[2].Identifier,
				ProfileName:       globalPfs[2].Name,
				Checksum:          globalProfiles[2].Checksum,
				HostUUID:          "test-uuid-3",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       teamPfs[0].ProfileUUID,
				ProfileIdentifier: teamPfs[0].Identifier,
				ProfileName:       teamPfs[0].Name,
				Checksum:          teamProfiles[0].Checksum,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
				CommandUUID:       "command-uuid",
			},
			{
				ProfileUUID:       teamPfs[1].ProfileUUID,
				ProfileIdentifier: teamPfs[1].Identifier,
				ProfileName:       teamPfs[1].Name,
				Checksum:          teamProfiles[1].Checksum,
				HostUUID:          "test-uuid-2",
				Status:            &fleet.MDMDeliveryVerifying,
				OperationType:     fleet.MDMOperationTypeInstall,
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

	// set host1 and host 3 to verified status, leave host2 as verifying
	verified := []*fleet.HostMacOSProfile{
		{Identifier: globalPfs[0].Identifier, DisplayName: globalPfs[0].Name, InstallDate: time.Now()},
		{Identifier: globalPfs[1].Identifier, DisplayName: globalPfs[1].Name, InstallDate: time.Now()},
		{Identifier: globalPfs[2].Identifier, DisplayName: globalPfs[2].Name, InstallDate: time.Now()},
	}
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, host1, profilesByIdentifier(verified)))
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, host3, profilesByIdentifier(verified)))

	// still no profiles to install
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profiles)

	// still no profiles to remove
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// add host1 to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host1.ID})
	require.NoError(t, err)

	// profiles to be added for host1 are now related to the team
	profiles, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-1"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-1"},
	}, profiles)

	// profiles to be removed includes host1's old profiles
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{
			ProfileUUID:       globalPfs[0].ProfileUUID,
			ProfileIdentifier: globalPfs[0].Identifier,
			ProfileName:       globalPfs[0].Name,
			Status:            &fleet.MDMDeliveryVerified,
			OperationType:     fleet.MDMOperationTypeInstall,
			HostUUID:          "test-uuid-1",
			CommandUUID:       "command-uuid",
		},
		{
			ProfileUUID:       globalPfs[1].ProfileUUID,
			ProfileIdentifier: globalPfs[1].Identifier,
			ProfileName:       globalPfs[1].Name,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerified,
			HostUUID:          "test-uuid-1",
			CommandUUID:       "command-uuid",
		},
		{
			ProfileUUID:       globalPfs[2].ProfileUUID,
			ProfileIdentifier: globalPfs[2].Identifier,
			ProfileName:       globalPfs[2].Name,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryVerified,
			HostUUID:          "test-uuid-1",
			CommandUUID:       "command-uuid",
		},
	}, toRemove)
}

// checkMDMHostRelatedTables checks that rows are inserted for new MDM hosts in
// each of host_display_names, host_seen_times, and label_membership. Note that
// related tables records for pre-existing hosts are created outside of the MDM
// enrollment flows so they are not checked in some tests above (e.g.,
// testIngestMDMAppleHostAlreadyExistsInFleet)
func checkMDMHostRelatedTables(t *testing.T, ds *Datastore, hostID uint, expectedSerial string, expectedModel string) {
	var displayName string
	err := sqlx.GetContext(context.Background(), ds.reader(context.Background()), &displayName, `SELECT display_name FROM host_display_names WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s (%s)", expectedModel, expectedSerial), displayName)

	var labelsOK []bool
	err = sqlx.SelectContext(context.Background(), ds.reader(context.Background()), &labelsOK, `SELECT 1 FROM label_membership WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Len(t, labelsOK, 2)
	require.True(t, labelsOK[0])
	require.True(t, labelsOK[1])

	appCfg, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	var hmdm fleet.HostMDM
	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &hmdm, `SELECT host_id, server_url, mdm_id FROM host_mdm WHERE host_id = ?`, hostID)
	require.NoError(t, err)
	require.Equal(t, hostID, hmdm.HostID)
	serverURL, err := apple_mdm.ResolveAppleMDMURL(appCfg.ServerSettings.ServerURL)
	require.NoError(t, err)
	require.Equal(t, serverURL, hmdm.ServerURL)
	require.NotEmpty(t, hmdm.MDMID)

	var mdmSolution fleet.MDMSolution
	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &mdmSolution, `SELECT name, server_url FROM mobile_device_management_solutions WHERE id = ?`, hmdm.MDMID)
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
		uuids []string
		want  map[string]mobileconfig.Mobileconfig
	}{
		{[]string{}, nil},
		{nil, nil},
		{[]string{profiles[0].ProfileUUID}, map[string]mobileconfig.Mobileconfig{profiles[0].ProfileUUID: profiles[0].Mobileconfig}},
		{
			[]string{profiles[0].ProfileUUID, profiles[1].ProfileUUID, profiles[2].ProfileUUID},
			map[string]mobileconfig.Mobileconfig{
				profiles[0].ProfileUUID: profiles[0].Mobileconfig,
				profiles[1].ProfileUUID: profiles[1].Mobileconfig,
				profiles[2].ProfileUUID: profiles[2].Mobileconfig,
			},
		},
	}

	for _, c := range cases {
		out, err := ds.GetMDMAppleProfilesContents(ctx, c.uuids)
		require.NoError(t, err)
		require.Equal(t, c.want, out)
	}
}

// createBuiltinLabels creates entries for "All Hosts" and "macOS" labels, which are assumed to be
// extant for MDM flows
func createBuiltinLabels(t *testing.T, ds *Datastore) {
	_, err := ds.writer(context.Background()).Exec(`
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
	_, err := ds.writer(context.Background()).Exec(`INSERT INTO nano_devices (id, authenticate) VALUES (?, 'test')`, host.UUID)
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, token_update_tally)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?)`,
		host.UUID,
		host.UUID,
		nil,
		"Device",
		host.UUID+".topic",
		host.UUID+".magic",
		host.UUID,
		1,
	)
	require.NoError(t, err)

	if withUser {
		_, err = ds.writer(context.Background()).Exec(`
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
	opType fleet.MDMOperationType,
	status *fleet.MDMDeliveryStatus,
	ctx context.Context,
	ds *Datastore,
	t *testing.T,
) {
	upserts := []*fleet.MDMAppleBulkUpsertHostProfilePayload{}
	for _, h := range hosts {
		for _, cp := range profiles {
			csum := []byte("csum")
			if cp.Checksum != nil {
				csum = cp.Checksum
			}
			payload := fleet.MDMAppleBulkUpsertHostProfilePayload{
				ProfileUUID:       cp.ProfileUUID,
				ProfileIdentifier: cp.Identifier,
				ProfileName:       cp.Name,
				HostUUID:          h.UUID,
				CommandUUID:       "",
				OperationType:     opType,
				Status:            status,
				Checksum:          csum,
			}
			upserts = append(upserts, &payload)
		}
	}
	err := ds.BulkUpsertMDMAppleHostProfiles(ctx, upserts)
	require.NoError(t, err)
}

func testAggregateMacOSSettingsStatusWithFileVault(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	checkListHosts := func(status fleet.OSSettingsStatus, teamID *uint, expected []*fleet.Host) bool {
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

	// upsert all host profiles with nil status, counts all as pending
	upsertHostCPs(hosts, append(noTeamCPs, fvNoTeam), fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	res, err := ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert all but filevault to verifying
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because filevault not installed
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert all but filevault to verified
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because filevault not installed
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert filevault to pending
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because filevault pending
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because no disk encryption key
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "foo", "", nil)
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because disk encryption key decryptable is not set
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-1), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[0] now has filevault fully enforced but not verified
	require.Equal(t, uint(0), res.Verified)

	// upsert hosts[0] filevault to verified
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[0], profilesByIdentifier([]*fleet.HostMacOSProfile{{Identifier: fvNoTeam.Identifier, DisplayName: fvNoTeam.Name, InstallDate: time.Now()}})))
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-1), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified) // hosts[0] now has filevault fully enforced and verified

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[1].ID, "bar", "", nil)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[1].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-1), res.Pending) // hosts[1] still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[1].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[1] now has filevault fully enforced
	require.Equal(t, uint(1), res.Verified)

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, hosts[1:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, hosts[0:1]))

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test"})
	require.NoError(t, err)

	// add hosts[9] to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{hosts[9].ID})
	require.NoError(t, err)

	// remove profiles from hosts[9]
	upsertHostCPs(hosts[9:10], append(noTeamCPs, fvNoTeam), fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying) // remove operations aren't currently subject to verification and only pending/failed removals are counted in summary
	require.Equal(t, uint(0), res.Verified)

	// create somes config profiles for team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 2; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), team.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}
	// add filevault profile for team
	fvTeam, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fleetmdm.FleetFileVaultProfileName, mobileconfig.FleetFileVaultPayloadIdentifier, team.ID))
	require.NoError(t, err)

	upsertHostCPs(hosts[9:10], append(teamCPs, fvTeam), fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending because it has no disk encryption key
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[9].ID, "baz", "", nil)
	require.NoError(t, err)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[9].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[9] now has filevault fully enforced but still verifying
	require.Equal(t, uint(0), res.Verified)

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.OSSettingsPending, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &team.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &team.ID, []*fleet.Host{}))

	upsertHostCPs(hosts[9:10], append(teamCPs, fvTeam), fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified) // hosts[9] now has filevault fully enforced and verified

	// set decryptable to false for hosts[9]
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[9].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending because it has no disk encryption key even though it was previously verified
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.OSSettingsPending, &team.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &team.ID, []*fleet.Host{}))

	// set decryptable back to true for hosts[9]
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[9].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &team.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified) // hosts[9] goes back to verified

	// check that list hosts by status matches summary
	require.True(t, checkListHosts(fleet.OSSettingsPending, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &team.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &team.ID, hosts[9:10]))
}

func testMDMAppleHostsProfilesStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	checkFilterHostsByMacOSSettings := func(status fleet.OSSettingsStatus, teamID *uint, expected []*fleet.Host) bool {
		expectedIDs := []uint{}
		for _, h := range expected {
			expectedIDs = append(expectedIDs, h.ID)
		}

		// check that list hosts by macos settings status matches summary
		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}}, fleet.HostListOptions{MacOSSettingsFilter: status, TeamFilter: teamID})
		gotIDs := []uint{}
		for _, h := range gotHosts {
			gotIDs = append(gotIDs, h.ID)
		}

		return assert.NoError(t, err) && assert.Len(t, gotHosts, len(expected)) && assert.ElementsMatch(t, expectedIDs, gotIDs)
	}

	// check that list hosts by os settings status matches summary
	checkFilterHostsByOSSettings := func(status fleet.OSSettingsStatus, teamID *uint, expected []*fleet.Host) bool {
		expectedIDs := []uint{}
		for _, h := range expected {
			expectedIDs = append(expectedIDs, h.ID)
		}

		gotHosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}}, fleet.HostListOptions{OSSettingsFilter: status, TeamFilter: teamID})
		gotIDs := []uint{}
		for _, h := range gotHosts {
			gotIDs = append(gotIDs, h.ID)
		}

		return assert.NoError(t, err) && assert.Len(t, gotHosts, len(expected)) && assert.ElementsMatch(t, expectedIDs, gotIDs)
	}

	checkListHosts := func(status fleet.OSSettingsStatus, teamID *uint, expected []*fleet.Host) bool {
		return checkFilterHostsByMacOSSettings(status, teamID, expected) && checkFilterHostsByOSSettings(status, teamID, expected)
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
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	res, err := ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, hosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// all hosts pending install of all profiles
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)), res.Pending) // each host only counts once
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, hosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), hosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0] and hosts[1] failed one profile
	upsertHostCPs(hosts[0:2], noTeamCPs[0:1], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, ctx, ds, t)
	// hosts[0] and hosts[1] have one profile pending as nil
	upsertHostCPs(hosts[0:2], noTeamCPs[3:4], fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	// hosts[0] also failed another profile
	upsertHostCPs(hosts[0:1], noTeamCPs[1:2], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, ctx, ds, t)
	// hosts[4] has all profiles reported as nil (pending)
	upsertHostCPs(hosts[4:5], noTeamCPs, fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	// hosts[5] has one profile reported as nil (pending)
	upsertHostCPs(hosts[5:6], noTeamCPs[0:1], fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // two hosts are failing at least one profile (hosts[0] and hosts[1])
	require.Equal(t, uint(2), res.Failed)             // only count one failure per host (hosts[0] failed two profiles but only counts once)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// hosts[0:3] installed a third profile
	upsertHostCPs(hosts[0:3], noTeamCPs[2:3], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-2), res.Pending) // no change
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, hosts[2:]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), hosts[2:]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// hosts[6] deletes all its profiles
	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, ds.deleteMDMAppleProfilesForHost(ctx, tx, hosts[6].UUID))
	require.NoError(t, tx.Commit())
	pendingHosts := append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] installed all profiles but one is with status nil (pending)
	upsertHostCPs(hosts[9:10], noTeamCPs[:9], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	upsertHostCPs(hosts[9:10], noTeamCPs[9:10], fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	pendingHosts = append(hosts[2:6:6], hosts[7:]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-3), res.Pending) // hosts[6] not reported here anymore, hosts[9] still pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // no change, host must apply all profiles count as latest
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// hosts[9] installed all profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract hosts[6 and 9] from pending
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(1), res.Verifying)          // add one host that has installed all profiles
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "rocket"})
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID) // get summary new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)   // no profiles yet
	require.Equal(t, uint(0), res.Failed)    // no profiles yet
	require.Equal(t, uint(0), res.Verifying) // no profiles yet
	require.Equal(t, uint(0), res.Verified)  // no profiles yet
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, []*fleet.Host{}))

	// transfer hosts[9] to new team
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{hosts[9].ID})
	require.NoError(t, err)
	// remove all no team profiles from hosts[9]
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, ctx, ds, t)

	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // hosts[9] is still not pending, transferred to team
	require.Equal(t, uint(2), res.Failed)             // no change
	require.Equal(t, uint(0), res.Verifying)          // hosts[9] was transferred so this is now zero
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))

	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID) // get summary for new team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, []*fleet.Host{}))

	// create somes config profiles for the new team
	var teamCPs []*fleet.MDMAppleConfigProfile
	for i := 0; i < 10; i++ {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fmt.Sprintf("name%d", i), fmt.Sprintf("identifier%d", i), tm.ID))
		require.NoError(t, err)
		teamCPs = append(teamCPs, cp)
	}

	// install all team profiles on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(1), res.Pending) // hosts[9] is still pending removal of old profiles
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, []*fleet.Host{}))

	// hosts[9] successfully removed old profiles
	upsertHostCPs(hosts[9:10], noTeamCPs, fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[9] is verifying all new profiles
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, []*fleet.Host{}))

	// verify one profile on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs[0:1], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[9] is still verifying other profiles
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, hosts[9:10]))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, []*fleet.Host{}))

	// verify the other profiles on hosts[9]
	upsertHostCPs(hosts[9:10], teamCPs[1:], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, uint(0), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified) // hosts[9] is all verified
	require.True(t, checkListHosts(fleet.OSSettingsPending, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, &tm.ID, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, &tm.ID, hosts[9:10]))

	// confirm no changes in summary for profiles with no team
	res, err = ds.GetMDMAppleProfilesSummary(ctx, ptr.Uint(0)) // team id zero represents no team
	require.NoError(t, err)
	require.NotNil(t, res)
	pendingHosts = append(hosts[2:6:6], hosts[7:9]...)
	require.Equal(t, uint(len(hosts)-4), res.Pending) // subtract two failed hosts, one without profiles and hosts[9] transferred
	require.Equal(t, uint(2), res.Failed)             // two failed hosts
	require.Equal(t, uint(0), res.Verifying)          // hosts[9] transferred to new team so is not counted under no team
	require.Equal(t, uint(0), res.Verified)
	require.True(t, checkListHosts(fleet.OSSettingsPending, nil, pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, nil, hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, nil, []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsPending, ptr.Uint(0), pendingHosts))
	require.True(t, checkListHosts(fleet.OSSettingsFailed, ptr.Uint(0), hosts[0:2]))
	require.True(t, checkListHosts(fleet.OSSettingsVerifying, ptr.Uint(0), []*fleet.Host{}))
	require.True(t, checkListHosts(fleet.OSSettingsVerified, ptr.Uint(0), []*fleet.Host{}))
}

func testMDMAppleIdPAccount(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	acc := &fleet.MDMIdPAccount{
		Username: "email@example.com",
		Email:    "email@example.com",
		Fullname: "John Doe",
	}

	err := ds.InsertMDMIdPAccount(ctx, acc)
	require.NoError(t, err)

	// try to instert the same account
	err = ds.InsertMDMIdPAccount(ctx, acc)
	require.NoError(t, err)

	out, err := ds.GetMDMIdPAccountByEmail(ctx, acc.Email)
	require.NoError(t, err)
	// update the acc UUID
	acc.UUID = out.UUID
	require.Equal(t, acc, out)

	var nfe fleet.NotFoundError
	out, err = ds.GetMDMIdPAccountByEmail(ctx, "bad@email.com")
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, out)

	out, err = ds.GetMDMIdPAccountByUUID(ctx, acc.UUID)
	require.NoError(t, err)
	require.Equal(t, acc, out)

	out, err = ds.GetMDMIdPAccountByUUID(ctx, "BAD-TOKEN")
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, out)
}

func testDoNotIgnoreMDMClientError(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create new record for remove pending
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       "a" + uuid.NewString(),
		ProfileIdentifier: "p1",
		ProfileName:       "name1",
		HostUUID:          "h1",
		CommandUUID:       "c1",
		OperationType:     fleet.MDMOperationTypeRemove,
		Status:            &fleet.MDMDeliveryPending,
		Checksum:          []byte("csum"),
	}}))
	cps, err := ds.GetHostMDMAppleProfiles(ctx, "h1")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name1", cps[0].Name)
	require.Equal(t, fleet.MDMOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *cps[0].Status)

	// simulate remove failed with client error message
	require.NoError(t, ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   "c1",
		HostUUID:      "h1",
		Status:        &fleet.MDMDeliveryFailed,
		Detail:        "MDMClientError (89): Profile with identifier 'p1' not found.",
		OperationType: fleet.MDMOperationTypeRemove,
	}))
	cps, err = ds.GetHostMDMAppleProfiles(ctx, "h1")
	require.NoError(t, err)
	require.Len(t, cps, 1) // we no longer ignore error code 89
	require.Equal(t, "name1", cps[0].Name)
	require.Equal(t, fleet.MDMOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMDeliveryFailed, *cps[0].Status)
	require.Equal(t, "Failed to remove: MDMClientError (89): Profile with identifier 'p1' not found.", cps[0].Detail)

	// create another new record
	require.NoError(t, ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       "a" + uuid.NewString(),
		ProfileIdentifier: "p2",
		ProfileName:       "name2",
		HostUUID:          "h2",
		CommandUUID:       "c2",
		OperationType:     fleet.MDMOperationTypeRemove,
		Status:            &fleet.MDMDeliveryPending,
		Checksum:          []byte("csum"),
	}}))
	cps, err = ds.GetHostMDMAppleProfiles(ctx, "h2")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name2", cps[0].Name)
	require.Equal(t, fleet.MDMOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *cps[0].Status)

	// simulate remove failed with another client error message that we don't want to ignore
	require.NoError(t, ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   "c2",
		HostUUID:      "h2",
		Status:        &fleet.MDMDeliveryFailed,
		Detail:        "MDMClientError (96): Cannot replace profile 'p2' because it was not installed by the MDM server.",
		OperationType: fleet.MDMOperationTypeRemove,
	}))
	cps, err = ds.GetHostMDMAppleProfiles(ctx, "h2")
	require.NoError(t, err)
	require.Len(t, cps, 1)
	require.Equal(t, "name2", cps[0].Name)
	require.Equal(t, fleet.MDMOperationTypeRemove, cps[0].OperationType)
	require.NotNil(t, cps[0].Status)
	require.Equal(t, fleet.MDMDeliveryFailed, *cps[0].Status)
	require.Equal(t, "Failed to remove: MDMClientError (96): Cannot replace profile 'p2' because it was not installed by the MDM server.", cps[0].Detail)
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
		ProfileUUID:       "a" + uuid.NewString(),
		ProfileIdentifier: "p1",
		ProfileName:       "name1",
		HostUUID:          h.UUID,
		CommandUUID:       "c1",
		OperationType:     fleet.MDMOperationTypeRemove,
		Status:            &fleet.MDMDeliveryPending,
		Checksum:          []byte("csum"),
	}}))

	gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 1)

	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, ds.deleteMDMAppleProfilesForHost(ctx, tx, h.UUID))
	require.NoError(t, tx.Commit())
	require.NoError(t, err)
	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)
}

func createDiskEncryptionRecord(ctx context.Context, ds *Datastore, t *testing.T, hostId uint, key string, decryptable bool, threshold time.Time) {
	err := ds.SetOrUpdateHostDiskEncryptionKey(ctx, hostId, key, "", nil)
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
	noTeamFVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fleetmdm.FleetFileVaultProfileName, mobileconfig.FleetFileVaultPayloadIdentifier, 0))
	require.NoError(t, err)

	// verifying status
	verifyingHost := hosts[0]
	upsertHostCPs([]*fleet.Host{verifyingHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
	createDiskEncryptionRecord(ctx, ds, t, verifyingHost.ID, "key-1", true, oneMinuteAfterThreshold)

	fvProfileSummary, err := ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(0), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err := ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// action required status
	requiredActionHost := hosts[1]
	upsertHostCPs(
		[]*fleet.Host{requiredActionHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryVerifying, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{requiredActionHost.ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// enforcing status
	enforcingHost := hosts[2]

	// host profile status is `pending`
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// host profile status does not exist
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		nil, ctx, ds, t,
	)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// host profile status is verifying but decryptable key field does not exist
	upsertHostCPs(
		[]*fleet.Host{enforcingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryPending, ctx, ds, t,
	)
	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{enforcingHost.ID}, false, oneMinuteAfterThreshold)
	require.NoError(t, err)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// failed status
	failedHost := hosts[3]
	upsertHostCPs([]*fleet.Host{failedHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, ctx, ds, t)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(1), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(2), allProfilesSummary.Pending)
	require.Equal(t, uint(1), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// removing enforcement status
	removingEnforcementHost := hosts[4]
	upsertHostCPs([]*fleet.Host{removingEnforcementHost}, []*fleet.MDMAppleConfigProfile{noTeamFVProfile}, fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, ctx, ds, t)
	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, nil)

	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(1), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(1), fvProfileSummary.Enforcing)
	require.Equal(t, uint(1), fvProfileSummary.Failed)
	require.Equal(t, uint(1), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(3), allProfilesSummary.Pending)
	require.Equal(t, uint(1), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// teams filter tests =====
	verifyingTeam1Host := hosts[6]
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-1"})
	require.NoError(t, err)
	team1FVProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP(fleetmdm.FleetFileVaultProfileName, mobileconfig.FleetFileVaultPayloadIdentifier, tm.ID))
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &tm.ID, []uint{verifyingTeam1Host.ID})
	require.NoError(t, err)

	upsertHostCPs([]*fleet.Host{verifyingTeam1Host}, []*fleet.MDMAppleConfigProfile{team1FVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	createDiskEncryptionRecord(ctx, ds, t, verifyingTeam1Host.ID, "key-2", true, oneMinuteAfterThreshold)

	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(1), fvProfileSummary.Verifying)
	require.Equal(t, uint(0), fvProfileSummary.Verified)
	require.Equal(t, uint(0), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// verified status
	upsertHostCPs([]*fleet.Host{verifyingTeam1Host}, []*fleet.MDMAppleConfigProfile{team1FVProfile}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	fvProfileSummary, err = ds.GetMDMAppleFileVaultSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), fvProfileSummary.Verifying)
	require.Equal(t, uint(1), fvProfileSummary.Verified)
	require.Equal(t, uint(0), fvProfileSummary.ActionRequired)
	require.Equal(t, uint(0), fvProfileSummary.Enforcing)
	require.Equal(t, uint(0), fvProfileSummary.Failed)
	require.Equal(t, uint(0), fvProfileSummary.RemovingEnforcement)

	allProfilesSummary, err = ds.GetMDMAppleProfilesSummary(ctx, &tm.ID)
	require.NoError(t, err)
	require.NotNil(t, fvProfileSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(0), allProfilesSummary.Verifying)
	require.Equal(t, uint(1), allProfilesSummary.Verified)
}

func testGetMDMAppleCommandResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// no enrolled host, unknown command
	res, err := ds.GetMDMAppleCommandResults(ctx, uuid.New().String())
	require.NoError(t, err)
	require.Empty(t, res)

	p, err := ds.GetMDMCommandPlatform(ctx, uuid.New().String())
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, p)

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
	err = commander.EnqueueCommand(ctx, []string{unenrolledHost.UUID}, createRawAppleCmd("ProfileList", uuid1))
	require.Error(t, err)
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.Equal(t, uint16(mysqlerr.ER_NO_REFERENCED_ROW_2), mysqlErr.Number)

	// command has no results
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid1)
	require.NoError(t, err)
	require.Empty(t, res)

	p, err = ds.GetMDMCommandPlatform(ctx, uuid1)
	require.True(t, fleet.IsNotFound(err))
	require.Empty(t, p)

	// enqueue a command for a couple of enrolled hosts
	uuid2 := uuid.New().String()
	rawCmd2 := createRawAppleCmd("ProfileList", uuid2)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[0].UUID, enrolledHosts[1].UUID}, rawCmd2)
	require.NoError(t, err)

	// command has no results yet
	res, err = ds.GetMDMAppleCommandResults(ctx, uuid2)
	require.NoError(t, err)
	require.Empty(t, res)
	// but it's already enqueued
	p, err = ds.GetMDMCommandPlatform(ctx, uuid2)
	require.NoError(t, err)
	require.Equal(t, "darwin", p)

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
	require.Equal(t, res[0], &fleet.MDMCommandResult{
		HostUUID:    enrolledHosts[0].UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd2),
		Payload:     []byte(rawCmd2),
	})
	p, err = ds.GetMDMCommandPlatform(ctx, uuid2)
	require.NoError(t, err)
	require.Equal(t, "darwin", p)

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

	require.ElementsMatch(t, res, []*fleet.MDMCommandResult{
		{
			HostUUID:    enrolledHosts[0].UUID,
			CommandUUID: uuid2,
			Status:      "Acknowledged",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
			Payload:     []byte(rawCmd2),
		},
		{
			HostUUID:    enrolledHosts[1].UUID,
			CommandUUID: uuid2,
			Status:      "Error",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
			Payload:     []byte(rawCmd2),
		},
	})

	p, err = ds.GetMDMCommandPlatform(ctx, uuid2)
	require.NoError(t, err)
	require.Equal(t, "darwin", p)

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
	require.ElementsMatch(t, res, []*fleet.MDMCommandResult{
		{
			HostUUID:    enrolledHosts[0].UUID,
			CommandUUID: uuid2,
			Status:      "Acknowledged",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
			Payload:     []byte(rawCmd2),
		},
		{
			HostUUID:    enrolledHosts[1].UUID,
			CommandUUID: uuid2,
			Status:      "Error",
			RequestType: "ProfileList",
			Result:      []byte(rawCmd2),
			Payload:     []byte(rawCmd2),
		},
	})

	p, err = ds.GetMDMCommandPlatform(ctx, uuid2)
	require.NoError(t, err)
	require.Equal(t, "darwin", p)
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

	checkProfiles := func(uploadedAtMatch bool) {
		for _, p := range allProfiles {
			profiles, err := ds.ListMDMAppleConfigProfiles(ctx, p.TeamID)
			require.NoError(t, err)
			require.Len(t, profiles, 1)

			wantProf := *p
			if !uploadedAtMatch {
				require.True(t, profiles[0].UploadedAt.After(wantProf.UploadedAt))
				wantProf.UploadedAt = time.Time{}
			}
			checkConfigProfile(t, wantProf, *profiles[0])
		}
	}

	err := ds.BulkUpsertMDMAppleConfigProfiles(ctx, allProfiles)
	require.NoError(t, err)

	reloadUploadedAt := func() {
		// reload to get the uploaded_at timestamps
		profiles, err := ds.ListMDMAppleConfigProfiles(ctx, nil)
		require.NoError(t, err)
		require.Len(t, profiles, 1)
		globalCP.UploadedAt = profiles[0].UploadedAt
		profiles, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(1))
		require.NoError(t, err)
		require.Len(t, profiles, 1)
		teamCP.UploadedAt = profiles[0].UploadedAt
	}
	reloadUploadedAt()

	checkProfiles(true)

	time.Sleep(time.Second) // ensure DB timestamps change

	newMc := mobileconfig.Mobileconfig([]byte("TestUpdatedConfigProfile"))
	globalCP.Mobileconfig = newMc
	teamCP.Mobileconfig = newMc
	err = ds.BulkUpsertMDMAppleConfigProfiles(ctx, allProfiles)
	require.NoError(t, err)

	// uploaded_at should be after the previously loaded timestamps
	checkProfiles(false)

	time.Sleep(time.Second) // ensure DB timestamps change

	// call it again with no changes, should not update timestamps
	reloadUploadedAt()
	err = ds.BulkUpsertMDMAppleConfigProfiles(ctx, allProfiles)
	require.NoError(t, err)
	checkProfiles(true)
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
	res, err := ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, res)

	// enqueue a command for enrolled hosts [0] and [1]
	uuid1 := uuid.New().String()
	rawCmd1 := createRawAppleCmd("ListApps", uuid1)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[0].UUID, enrolledHosts[1].UUID}, rawCmd1)
	require.NoError(t, err)

	// command has no results yet, so the status is empty
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
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
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
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
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
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
	rawCmd2 := createRawAppleCmd("InstallApp", uuid2)
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
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 4)

	// page-by-page: first page
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{
		ListOptions: fleet.ListOptions{Page: 0, PerPage: 3, OrderKey: "device_id", OrderDirection: fleet.OrderDescending},
	})
	require.NoError(t, err)
	require.Len(t, res, 3)

	// page-by-page: second page
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{
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
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: u1}, &fleet.MDMCommandListOptions{
		ListOptions: fleet.ListOptions{PerPage: 3},
	})
	require.NoError(t, err)
	require.Len(t, res, 0)

	// now with IncludeObserver set to true
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: u1, IncludeObserver: true}, &fleet.MDMCommandListOptions{
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

	// randomly set two commadns as inactive
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `UPDATE nano_enrollment_queue SET active = 0 LIMIT 2`)
		return err
	})
	// only two results are listed
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 2)
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
	hosts := make([]*fleet.Host, 7)
	for i := 0; i < len(hosts); i++ {
		serial := fmt.Sprintf("serial-%d", i)
		if i == 6 {
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
			// assigned in ABM to Fleet
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h})
			require.NoError(t, err)
		case i == 4:
			// not ABM assigned
		case i == 5:
			// ABM assignment was deleted
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h})
			require.NoError(t, err)
			err = ds.DeleteHostDEPAssignments(ctx, []string{h.HardwareSerial})
			require.NoError(t, err)
		case i == 6:
			// assigned in ABM, but we don't have a serial
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h})
			require.NoError(t, err)
		}
		hosts[i] = h
		t.Logf("host [%d]: %s - %s", i, h.UUID, h.HardwareSerial)
	}

	// create teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// assign hosts[2,4,5] to tm1
	err = ds.AddHostsToTeam(ctx, &tm1.ID, []uint{hosts[2].ID, hosts[4].ID, hosts[5].ID})
	require.NoError(t, err)

	// list serials in team 2, has none
	serials, err := ds.ListMDMAppleDEPSerialsInTeam(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in team 1, has one (hosts[2])
	serials, err = ds.ListMDMAppleDEPSerialsInTeam(ctx, &tm1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-2"}, serials)

	// list serials in no-team, has 3 (hosts[0,1,3]), hosts[6] doesn't have a serial number
	serials, err = ds.ListMDMAppleDEPSerialsInTeam(ctx, nil)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-3"}, serials)

	// list serials with no host IDs returns empty
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in hosts[0,1,2,3] returns all of them
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-2", "serial-3"}, serials)

	// list serials in hosts[4,5,6] returns none
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{hosts[4].ID, hosts[5].ID, hosts[6].ID})
	require.NoError(t, err)
	require.Empty(t, serials)

	// list serials in all hosts returns [0-3]
	serials, err = ds.ListMDMAppleDEPSerialsInHostIDs(ctx, []uint{
		hosts[0].ID, hosts[1].ID, hosts[2].ID, hosts[3].ID, hosts[4].ID,
		hosts[5].ID, hosts[6].ID,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"serial-0", "serial-1", "serial-2", "serial-3"}, serials)
}

func testMDMAppleDefaultSetupAssistant(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get non-existing
	_, _, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, nil)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// set for no team
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, nil, "no-team")
	require.NoError(t, err)

	// get for no team returns the same data
	uuid, ts, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, "no-team", uuid)
	require.NotZero(t, ts)

	// set for non-existing team fails
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, ptr.Uint(123), "xyz")
	require.Error(t, err)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// get for non-existing team fails
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, ptr.Uint(123))
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm"})
	require.NoError(t, err)

	// set for existing team
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, &tm.ID, "tm")
	require.NoError(t, err)

	// get for existing team
	uuid, ts, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID)
	require.NoError(t, err)
	require.Equal(t, "tm", uuid)
	require.NotZero(t, ts)
}

func testSetVerifiedMacOSProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// map of host IDs to map of profile identifiers to delivery status
	expectedHostMDMStatus := make(map[uint]map[string]fleet.MDMDeliveryStatus)

	// create some config profiles for no team
	cp1, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "name1", "cp1", "uuid1"))
	require.NoError(t, err)
	cp2, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "name2", "cp2", "uuid2"))
	require.NoError(t, err)
	cp3, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "name3", "cp3", "uuid3"))
	require.NoError(t, err)
	cp4, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "name4", "cp4", "uuid4"))
	require.NoError(t, err)

	// list config profiles for no team
	cps, err := ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, cps, 4)
	storedByIdentifier := make(map[string]*fleet.MDMAppleConfigProfile)
	for _, cp := range cps {
		storedByIdentifier[cp.Identifier] = cp
	}

	// create test hosts
	var hosts []*fleet.Host
	for i := 0; i < 3; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now().Add(-1*time.Hour))
		hosts = append(hosts, h)
		expectedHostMDMStatus[h.ID] = map[string]fleet.MDMDeliveryStatus{
			cp1.Identifier: fleet.MDMDeliveryPending,
			cp2.Identifier: fleet.MDMDeliveryVerifying,
			cp3.Identifier: fleet.MDMDeliveryVerified,
			cp4.Identifier: fleet.MDMDeliveryPending,
		}
	}

	// add a team config profile with the same name and identifer as one of the no-team profiles
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm"})
	require.NoError(t, err)
	_, err = ds.NewMDMAppleConfigProfile(ctx, *teamConfigProfileForTest(t, cp2.Name, cp2.Identifier, "uuid2", tm.ID))
	require.NoError(t, err)

	checkHostMDMProfileStatuses := func() {
		for _, h := range hosts {
			gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
			require.NoError(t, err)
			require.Len(t, gotProfs, 4)
			for _, p := range gotProfs {
				s, ok := expectedHostMDMStatus[h.ID][p.Identifier]
				require.True(t, ok)
				require.NotNil(t, p.Status)
				require.Equalf(t, s, *p.Status, "profile identifier %s", p.Identifier)
			}
		}
	}

	adHocSetVerifying := func(hostUUID, profileIndentifier string) {
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx,
				`UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ? AND profile_identifier = ?`,
				fleet.MDMDeliveryVerifying, hostUUID, profileIndentifier)
			return err
		})
	}

	// initialize the host MDM profile statuses
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{storedByIdentifier[cp1.Identifier]}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, ctx, ds, t)
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{storedByIdentifier[cp2.Identifier]}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{storedByIdentifier[cp3.Identifier]}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{storedByIdentifier[cp4.Identifier]}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, ctx, ds, t)
	checkHostMDMProfileStatuses()

	// statuses don't change during the grace period if profiles are missing (i.e. not installed)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[0], map[string]*fleet.HostMacOSProfile{}))
	checkHostMDMProfileStatuses()

	// if install date is before the updated at timestamp of the profile, statuses don't change
	// during the grace period
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[1], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp1.Identifier,
			DisplayName: cp1.Name,
			InstallDate: storedByIdentifier[cp1.Identifier].UploadedAt.Add(-1 * time.Hour),
		},
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: storedByIdentifier[cp2.Identifier].UploadedAt.Add(-1 * time.Hour),
		},
		{
			Identifier:  cp3.Identifier,
			DisplayName: cp3.Name,
			InstallDate: storedByIdentifier[cp3.Identifier].UploadedAt.Add(-1 * time.Hour),
		},
		{
			Identifier:  cp4.Identifier,
			DisplayName: cp4.Name,
			InstallDate: storedByIdentifier[cp4.Identifier].UploadedAt.Add(-1 * time.Hour),
		},
	})))
	checkHostMDMProfileStatuses()

	// if install date is on or after the updated at timestamp of the profile, "verifying" or "pending" status
	// changes to "verified". Any "pending" profiles not reported are not changed
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: storedByIdentifier[cp2.Identifier].UploadedAt,
		},
		{
			Identifier:  cp3.Identifier,
			DisplayName: cp3.Name,
			InstallDate: storedByIdentifier[cp3.Identifier].UploadedAt,
		},
		{
			Identifier:  cp4.Identifier,
			DisplayName: cp4.Name,
			InstallDate: storedByIdentifier[cp4.Identifier].UploadedAt,
		},
	})))
	expectedHostMDMStatus[hosts[2].ID][cp2.Identifier] = fleet.MDMDeliveryVerified
	expectedHostMDMStatus[hosts[2].ID][cp4.Identifier] = fleet.MDMDeliveryVerified
	checkHostMDMProfileStatuses()

	// repeated call doesn't change statuses
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: storedByIdentifier[cp2.Identifier].UploadedAt,
		},
		{
			Identifier:  cp3.Identifier,
			DisplayName: cp3.Name,
			InstallDate: storedByIdentifier[cp3.Identifier].UploadedAt,
		},
		{
			Identifier:  cp4.Identifier,
			DisplayName: cp4.Name,
			InstallDate: storedByIdentifier[cp4.Identifier].UploadedAt,
		},
	})))
	checkHostMDMProfileStatuses()

	// simulate expired grace period by setting uploaded_at timestamp of profiles back by 24 hours
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx,
			`UPDATE mdm_apple_configuration_profiles SET uploaded_at = ? WHERE profile_uuid IN(?, ?, ?, ?)`,
			time.Now().Add(-24*time.Hour),
			cp1.ProfileUUID, cp2.ProfileUUID, cp3.ProfileUUID, cp4.ProfileUUID,
		)
		return err
	})

	// after the grace period and one retry attempt, status changes to "failed" if a profile is missing (i.e. not installed)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp1.Identifier,
			DisplayName: cp1.Name,
			InstallDate: time.Now(),
		},
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: time.Now(),
		},
	})))
	expectedHostMDMStatus[hosts[2].ID][cp1.Identifier] = fleet.MDMDeliveryVerified // cp1 can go from pending to verified
	expectedHostMDMStatus[hosts[2].ID][cp3.Identifier] = fleet.MDMDeliveryPending  // first retry for cp3
	expectedHostMDMStatus[hosts[2].ID][cp4.Identifier] = fleet.MDMDeliveryPending  // first retry for cp4
	checkHostMDMProfileStatuses()
	// simulate retry command acknowledged by setting status to "verifying"
	adHocSetVerifying(hosts[2].UUID, cp3.Identifier)
	adHocSetVerifying(hosts[2].UUID, cp4.Identifier)
	// report osquery results again with cp3 and cp4 still missing
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp1.Identifier,
			DisplayName: cp1.Name,
			InstallDate: time.Now(),
		},
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: time.Now(),
		},
	})))
	expectedHostMDMStatus[hosts[2].ID][cp3.Identifier] = fleet.MDMDeliveryFailed // still missing after retry so expect cp3 to fail
	expectedHostMDMStatus[hosts[2].ID][cp4.Identifier] = fleet.MDMDeliveryFailed // still missing after retry so expect cp4 to fail
	checkHostMDMProfileStatuses()

	// after the grace period and one retry attempt, status changes to "failed" if a profile is outdated (i.e. installed
	// before the updated at timestamp of the profile)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp1.Identifier,
			DisplayName: cp1.Name,
			InstallDate: time.Now(),
		},
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: time.Now().Add(-48 * time.Hour),
		},
	})))
	expectedHostMDMStatus[hosts[2].ID][cp2.Identifier] = fleet.MDMDeliveryPending // first retry for cp2
	checkHostMDMProfileStatuses()
	// simulate retry command acknowledged by setting status to "verifying"
	adHocSetVerifying(hosts[2].UUID, cp2.Identifier)
	// report osquery results again with cp2 still outdated
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[2], profilesByIdentifier([]*fleet.HostMacOSProfile{
		{
			Identifier:  cp1.Identifier,
			DisplayName: cp1.Name,
			InstallDate: time.Now(),
		},
		{
			Identifier:  cp2.Identifier,
			DisplayName: cp2.Name,
			InstallDate: time.Now().Add(-48 * time.Hour),
		},
	})))
	expectedHostMDMStatus[hosts[2].ID][cp2.Identifier] = fleet.MDMDeliveryFailed // still outdated after retry so expect cp2 to fail
	checkHostMDMProfileStatuses()
}

func TestCopyDefaultMDMAppleBootstrapPackage(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()

	checkStoredBP := func(teamID uint, wantErr error, wantNewToken bool, wantBP *fleet.MDMAppleBootstrapPackage) {
		var gotBP fleet.MDMAppleBootstrapPackage
		err := sqlx.GetContext(ctx, ds.primary, &gotBP, "SELECT * FROM mdm_apple_bootstrap_packages WHERE team_id = ?", teamID)
		if wantErr != nil {
			require.EqualError(t, err, wantErr.Error())
			return
		}
		require.NoError(t, err)
		if wantNewToken {
			require.NotEqual(t, wantBP.Token, gotBP.Token)
		} else {
			require.Equal(t, wantBP.Token, gotBP.Token)
		}
		require.Equal(t, wantBP.Name, gotBP.Name)
		require.Equal(t, wantBP.Sha256[:32], gotBP.Sha256)
		require.Equal(t, wantBP.Bytes, gotBP.Bytes)
	}

	checkAppConfig := func(wantURL string) {
		ac, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, wantURL, ac.MDM.MacOSSetup.BootstrapPackage.Value)
	}

	checkTeamConfig := func(teamID uint, wantURL string) {
		tm, err := ds.Team(ctx, teamID)
		require.NoError(t, err)
		require.Equal(t, wantURL, tm.Config.MDM.MacOSSetup.BootstrapPackage.Value)
	}

	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "test"})
	require.NoError(t, err)
	teamID := tm.ID
	noTeamID := uint(0)

	// confirm bootstrap package url is empty by default
	checkAppConfig("")
	checkTeamConfig(teamID, "")

	// create a default bootstrap package
	defaultBP := &fleet.MDMAppleBootstrapPackage{
		TeamID: noTeamID,
		Name:   "name",
		Sha256: sha256.New().Sum([]byte("content")),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, defaultBP)
	require.NoError(t, err)
	checkStoredBP(noTeamID, nil, false, defaultBP)   // default bootstrap package is stored
	checkStoredBP(teamID, sql.ErrNoRows, false, nil) // no bootstrap package yet for team

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	require.Empty(t, ac.MDM.MacOSSetup.BootstrapPackage.Value)
	err = ds.CopyDefaultMDMAppleBootstrapPackage(ctx, ac, teamID)
	require.NoError(t, err)

	checkAppConfig("")                             // no bootstrap package url set in app config
	checkTeamConfig(teamID, "")                    // no bootstrap package url set in team config
	checkStoredBP(noTeamID, nil, false, defaultBP) // no change to default bootstrap package
	checkStoredBP(teamID, nil, true, defaultBP)    // copied default bootstrap package

	// delete and update the default bootstrap package
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, noTeamID)
	require.NoError(t, err)
	checkStoredBP(noTeamID, sql.ErrNoRows, false, nil) // deleted
	checkStoredBP(teamID, nil, true, defaultBP)        // still exists

	// update the default bootstrap package
	defaultBP2 := &fleet.MDMAppleBootstrapPackage{
		TeamID: noTeamID,
		Name:   "new name",
		Sha256: sha256.New().Sum([]byte("new content")),
		Bytes:  []byte("new content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, defaultBP2)
	require.NoError(t, err)
	checkStoredBP(noTeamID, nil, false, defaultBP2)
	// set bootstrap package url in app config
	ac.MDM.MacOSSetup.BootstrapPackage = optjson.SetString("https://example.com/bootstrap.pkg")
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	checkAppConfig("https://example.com/bootstrap.pkg")

	// copy default bootstrap package fails when there is already a team bootstrap package
	var wantErr error = &existsError{ResourceType: "BootstrapPackage", TeamID: &teamID}
	err = ds.CopyDefaultMDMAppleBootstrapPackage(ctx, ac, teamID)
	require.ErrorContains(t, err, wantErr.Error())
	// confirm team bootstrap package is unchanged
	checkStoredBP(teamID, nil, true, defaultBP)
	checkTeamConfig(teamID, "")

	// delete the team bootstrap package
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, teamID)
	require.NoError(t, err)
	checkStoredBP(teamID, sql.ErrNoRows, false, nil)
	checkTeamConfig(teamID, "")

	// confirm no change to default bootstrap package
	checkStoredBP(noTeamID, nil, false, defaultBP2)
	checkAppConfig("https://example.com/bootstrap.pkg")

	// copy default bootstrap package succeeds when there is no team bootstrap package
	err = ds.CopyDefaultMDMAppleBootstrapPackage(ctx, ac, teamID)
	require.NoError(t, err)
	// confirm team bootstrap package gets new token and otherwise matches default bootstrap package
	checkStoredBP(teamID, nil, true, defaultBP2)
	// confirm bootstrap package url was set in team config to match app config
	checkTeamConfig(teamID, "https://example.com/bootstrap.pkg")

	// test some edge cases

	// delete the team bootstrap package doesn't affect the team config
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, teamID)
	require.NoError(t, err)
	checkStoredBP(teamID, sql.ErrNoRows, false, nil)
	checkTeamConfig(teamID, "https://example.com/bootstrap.pkg")

	// set other team config values so we can confirm they are not affected by bootstrap package changes
	tc, err := ds.Team(ctx, teamID)
	require.NoError(t, err)
	tc.Config.MDM.MacOSSetup.MacOSSetupAssistant = optjson.SetString("/path/to/setupassistant")
	tc.Config.MDM.MacOSUpdates.Deadline = optjson.SetString("2024-01-01")
	tc.Config.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("10.15.4")
	tc.Config.WebhookSettings.FailingPoliciesWebhook = fleet.FailingPoliciesWebhookSettings{
		Enable:         true,
		DestinationURL: "https://example.com/webhook",
	}
	tc.Config.Features.EnableHostUsers = false
	savedTeam, err := ds.SaveTeam(ctx, tc)
	require.NoError(t, err)
	require.Equal(t, tc.Config, savedTeam.Config)

	// change the default bootstrap package url
	ac.MDM.MacOSSetup.BootstrapPackage = optjson.SetString("https://example.com/bs.pkg")
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)
	checkAppConfig("https://example.com/bs.pkg")
	checkTeamConfig(teamID, "https://example.com/bootstrap.pkg") // team config is unchanged

	// copy default bootstrap package succeeds when there is no team bootstrap package
	err = ds.CopyDefaultMDMAppleBootstrapPackage(ctx, ac, teamID)
	require.NoError(t, err)
	// confirm team bootstrap package gets new token and otherwise matches default bootstrap package
	checkStoredBP(teamID, nil, true, defaultBP2)
	// confirm bootstrap package url was set in team config to match app config
	checkTeamConfig(teamID, "https://example.com/bs.pkg")

	// confirm other team config values are unchanged
	tc, err = ds.Team(ctx, teamID)
	require.NoError(t, err)
	require.Equal(t, tc.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value, "/path/to/setupassistant")
	require.Equal(t, tc.Config.MDM.MacOSUpdates.Deadline.Value, "2024-01-01")
	require.Equal(t, tc.Config.MDM.MacOSUpdates.MinimumVersion.Value, "10.15.4")
	require.Equal(t, tc.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL, "https://example.com/webhook")
	require.Equal(t, tc.Config.WebhookSettings.FailingPoliciesWebhook.Enable, true)
	require.Equal(t, tc.Config.Features.EnableHostUsers, false)
}

func TestHostDEPAssignments(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	expectedMDMServerURL, err := apple_mdm.ResolveAppleEnrollMDMURL(ac.ServerSettings.ServerURL)
	require.NoError(t, err)

	t.Run("DEP enrollment", func(t *testing.T) {
		depSerial := "dep-serial"
		depUUID := "dep-uuid"
		depOrbitNodeKey := "dep-orbit-node-key"
		depDeviceTok := "dep-device-token"

		n, _, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{{SerialNumber: depSerial}})
		require.NoError(t, err)
		require.Equal(t, int64(1), n)

		var depHostID uint
		err = sqlx.GetContext(ctx, ds.reader(ctx), &depHostID, "SELECT id FROM hosts WHERE hardware_serial = ?", depSerial)
		require.NoError(t, err)

		// host MDM row is created when DEP device is ingested
		getHostResp, err := ds.Host(ctx, depHostID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, depHostID, getHostResp.ID)
		require.Equal(t, "Pending", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// host DEP assignment is created when DEP device is ingested
		depAssignment, err := ds.GetHostDEPAssignment(ctx, depHostID)
		require.NoError(t, err)
		require.Equal(t, depHostID, depAssignment.HostID)
		require.Nil(t, depAssignment.DeletedAt)
		require.WithinDuration(t, time.Now(), depAssignment.AddedAt, 5*time.Second)

		// simulate initial osquery enrollment via Orbit
		testHost, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{HardwareSerial: depSerial, Platform: "darwin", HardwareUUID: depUUID, Hostname: "dep-host"}, depOrbitNodeKey, nil)
		require.NoError(t, err)
		require.NotNil(t, testHost)

		// create device auth token for host
		err = ds.SetOrUpdateDeviceAuthToken(context.Background(), depHostID, depDeviceTok)
		require.NoError(t, err)

		// host MDM doesn't change upon Orbit enrollment
		getHostResp, err = ds.Host(ctx, testHost.ID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, testHost.ID, getHostResp.ID)
		require.Equal(t, "Pending", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// host DEP assignment is reported for load host by Orbit node key and by device token
		h, err := ds.LoadHostByOrbitNodeKey(ctx, depOrbitNodeKey)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, depDeviceTok, 1*time.Hour)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)

		// simulate osquery report of MDM detail query
		err = ds.SetOrUpdateMDMData(ctx, testHost.ID, false, true, expectedMDMServerURL, true, fleet.WellKnownMDMFleet, "")
		require.NoError(t, err)

		// enrollment status changes to "On (automatic)"
		getHostResp, err = ds.Host(ctx, testHost.ID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, testHost.ID, getHostResp.ID)
		require.Equal(t, "On (automatic)", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// host DEP assignment doesn't change
		h, err = ds.LoadHostByOrbitNodeKey(ctx, depOrbitNodeKey)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, depDeviceTok, 1*time.Hour)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)

		// simulate MDM unenroll
		require.NoError(t, ds.UpdateHostTablesOnMDMUnenroll(ctx, depUUID))

		// host MDM row is set to defaults on unenrollment
		getHostResp, err = ds.Host(ctx, testHost.ID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, testHost.ID, getHostResp.ID)
		require.NotNil(t, getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, "Off", *getHostResp.MDM.EnrollmentStatus)
		require.Empty(t, getHostResp.MDM.ServerURL)
		require.Empty(t, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// host DEP assignment doesn't change
		h, err = ds.LoadHostByOrbitNodeKey(ctx, depOrbitNodeKey)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, depDeviceTok, 1*time.Hour)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)

		// simulate osquery report of MDM detail query reflecting re-enrollment to MDM
		err = ds.SetOrUpdateMDMData(ctx, testHost.ID, false, true, expectedMDMServerURL, true, fleet.WellKnownMDMFleet, "")
		require.NoError(t, err)

		// host MDM row is re-created when osquery reports MDM detail query
		getHostResp, err = ds.Host(ctx, testHost.ID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, testHost.ID, getHostResp.ID)
		require.Equal(t, "On (automatic)", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// DEP assignment doesn't change
		h, err = ds.LoadHostByOrbitNodeKey(ctx, depOrbitNodeKey)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, depDeviceTok, 1*time.Hour)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)

		// simulate osquery report of MDM detail query with empty server URL (signals unenrollment
		// from MDM)
		err = ds.SetOrUpdateMDMData(ctx, testHost.ID, false, false, "", false, "", "")
		require.NoError(t, err)

		// host MDM row is reset to defaults when osquery reports MDM detail query with empty server URL
		getHostResp, err = ds.Host(ctx, testHost.ID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, testHost.ID, getHostResp.ID)
		require.NotNil(t, getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, "Off", *getHostResp.MDM.EnrollmentStatus)
		require.Empty(t, getHostResp.MDM.ServerURL)
		require.Empty(t, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// DEP assignment doesn't change
		h, err = ds.LoadHostByOrbitNodeKey(ctx, depOrbitNodeKey)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, depDeviceTok, 1*time.Hour)
		require.NoError(t, err)
		require.True(t, *h.DEPAssignedToFleet)

		hdepa, err := ds.GetHostDEPAssignment(ctx, depHostID)
		require.NoError(t, err)
		require.Equal(t, depHostID, hdepa.HostID)
		require.Nil(t, hdepa.DeletedAt)
		require.Equal(t, depAssignment.AddedAt, hdepa.AddedAt)
	})

	t.Run("manual enrollment", func(t *testing.T) {
		// create a non-DEP host
		manualSerial := "manual-serial"
		manualUUID := "manual-uuid"
		manualOrbitNodeKey := "manual-orbit-node-key"
		manualDeviceToken := "manual-device-token"

		err = ds.IngestMDMAppleDeviceFromCheckin(ctx, fleet.MDMAppleHostDetails{SerialNumber: manualSerial, UDID: manualUUID})
		require.NoError(t, err)

		var manualHostID uint
		err = sqlx.GetContext(ctx, ds.reader(ctx), &manualHostID, "SELECT id FROM hosts WHERE hardware_serial = ?", manualSerial)
		require.NoError(t, err)

		// host MDM is "On (manual)"
		getHostResp, err := ds.Host(ctx, manualHostID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, manualHostID, getHostResp.ID)
		require.Equal(t, "On (manual)", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		// check host DEP assignment not created for non-DEP host
		hdepa, err := ds.GetHostDEPAssignment(ctx, manualHostID)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Nil(t, hdepa)

		// simulate initial osquery enrollment via Orbit
		manualHost, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{HardwareSerial: manualSerial, Platform: "darwin", HardwareUUID: manualUUID, Hostname: "maunual-host"}, manualOrbitNodeKey, nil)
		require.NoError(t, err)
		require.Equal(t, manualHostID, manualHost.ID)

		// create device auth token for host
		err = ds.SetOrUpdateDeviceAuthToken(context.Background(), manualHostID, manualDeviceToken)
		require.NoError(t, err)

		// host MDM doesn't change upon Orbit enrollment
		getHostResp, err = ds.Host(ctx, manualHostID)
		require.NoError(t, err)
		require.NotNil(t, getHostResp)
		require.Equal(t, manualHostID, getHostResp.ID)
		require.Equal(t, "On (manual)", *getHostResp.MDM.EnrollmentStatus)
		require.Equal(t, fleet.WellKnownMDMFleet, getHostResp.MDM.Name)
		require.Nil(t, getHostResp.DEPAssignedToFleet) // always nil for get host

		h, err := ds.LoadHostByOrbitNodeKey(ctx, manualOrbitNodeKey)
		require.NoError(t, err)
		require.False(t, *h.DEPAssignedToFleet)
		h, err = ds.LoadHostByDeviceAuthToken(ctx, manualDeviceToken, 1*time.Hour)
		require.NoError(t, err)
		require.False(t, *h.DEPAssignedToFleet)
	})
}

func testMDMAppleConfigProfileHash(t *testing.T, ds *Datastore) {
	// test that the mysql md5 hash exactly matches the hash produced by Go in
	// the preassign profiles logic (no corner cases with extra whitespace, etc.)
	ctx := context.Background()

	// sprintf placeholders for prefix, content and suffix
	const base = `%s<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
%s
</plist>%s`

	cases := []struct {
		prefix, content, suffix string
	}{
		{"", "", ""},
		{" ", "", ""},
		{"", "", " "},
		{"\t\n ", "", "\t\n "},
		{"", `<dict>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>PayloadUUID</key>
      <string>Ignored</string>
      <key>PayloadType</key>
      <string>Configuration</string>
      <key>PayloadIdentifier</key>
      <string>Ignored</string>
</dict>`, ""},
		{" ", `<dict>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>PayloadUUID</key>
      <string>Ignored</string>
      <key>PayloadType</key>
      <string>Configuration</string>
      <key>PayloadIdentifier</key>
      <string>Ignored</string>
</dict>`, "\r\n"},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%q %q %q", c.prefix, c.content, c.suffix), func(t *testing.T) {
			mc := mobileconfig.Mobileconfig(fmt.Sprintf(base, c.prefix, c.content, c.suffix))

			prof, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{
				Name:         fmt.Sprintf("profile-%d", i),
				Identifier:   fmt.Sprintf("profile-%d", i),
				TeamID:       nil,
				Mobileconfig: mc,
			})
			require.NoError(t, err)

			t.Cleanup(func() {
				err := ds.DeleteMDMAppleConfigProfile(ctx, prof.ProfileUUID)
				require.NoError(t, err)
			})

			goProf := fleet.MDMApplePreassignProfilePayload{Profile: mc}
			goHash := goProf.HexMD5Hash()
			require.NotEmpty(t, goHash)

			var uid string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE checksum = UNHEX(?)`, goHash)
			})
			require.Equal(t, prof.ProfileUUID, uid)
		})
	}
}

func testResetMDMAppleEnrollment(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// try with a host that doesn't have a matching entry
	// in nano_enrollments
	err = ds.ResetMDMAppleEnrollment(ctx, host.UUID)
	require.NoError(t, err)

	// add a matching entry in the nano table
	nanoEnroll(t, ds, host, false)

	enrollment, err := ds.GetNanoMDMEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.Equal(t, enrollment.TokenUpdateTally, 1)

	// add configuration profiles
	cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("name0", "identifier0", 0))
	require.NoError(t, err)
	upsertHostCPs([]*fleet.Host{host}, []*fleet.MDMAppleConfigProfile{cp}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)

	gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 1)

	// add a record of the bootstrap package being installed
	_, err = ds.writer(ctx).Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('command-uuid', 'foo', '<?xml')
	`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).Exec(`
          INSERT INTO nano_command_results (id, command_uuid, status, result)
          VALUES (?, 'command-uuid', 'Acknowledged', '<?xml')
	`, host.UUID)
	require.NoError(t, err)
	err = ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(0),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	})
	require.NoError(t, err)
	err = ds.RecordHostBootstrapPackage(ctx, "command-uuid", host.UUID)
	require.NoError(t, err)
	// add a record of the host DEP assignment
	_, err = ds.writer(ctx).Exec(`
		INSERT INTO host_dep_assignments (host_id)
		VALUES (?)
		ON DUPLICATE KEY UPDATE added_at = CURRENT_TIMESTAMP, deleted_at = NULL
	`, host.ID)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "foo.mdm.example.com", true, "", "")
	require.NoError(t, err)

	sum, err := ds.GetMDMAppleBootstrapPackageSummary(ctx, uint(0))
	require.NoError(t, err)
	require.Zero(t, sum.Failed)
	require.Zero(t, sum.Pending)
	require.EqualValues(t, 1, sum.Installed)

	// reset the enrollment
	err = ds.ResetMDMAppleEnrollment(ctx, host.UUID)
	require.NoError(t, err)

	enrollment, err = ds.GetNanoMDMEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.Zero(t, enrollment.TokenUpdateTally)

	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, gotProfs)

	sum, err = ds.GetMDMAppleBootstrapPackageSummary(ctx, uint(0))
	require.NoError(t, err)
	require.Zero(t, sum.Failed)
	require.Zero(t, sum.Installed)
	require.EqualValues(t, 1, sum.Pending)
}

func testMDMAppleDeleteHostDEPAssignments(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	cases := []struct {
		name string
		in   []string
		want []string
		err  string
	}{
		{"no serials provided", []string{}, []string{"foo", "bar", "baz"}, ""},
		{"no matching serials", []string{"oof", "rab"}, []string{"foo", "bar", "baz"}, ""},
		{"partial matches", []string{"foo", "rab"}, []string{"bar", "baz"}, ""},
		{"all matching", []string{"foo", "bar", "baz"}, []string{}, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			devices := []godep.Device{
				{SerialNumber: "foo"},
				{SerialNumber: "bar"},
				{SerialNumber: "baz"},
			}
			_, _, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, devices)
			require.NoError(t, err)

			err = ds.DeleteHostDEPAssignments(ctx, tt.in)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
			var got []string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.SelectContext(
					ctx, q, &got,
					`SELECT hardware_serial FROM hosts h
                                         JOIN host_dep_assignments hda ON hda.host_id = h.id
                                         WHERE hda.deleted_at IS NULL`,
				)
			})
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

func testCleanMacOSMDMLock(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	checkState := func(t *testing.T, status *fleet.HostLockWipeStatus, unlocked, locked, wiped, pendingUnlock, pendingLock, pendingWipe bool) {
		require.Equal(t, unlocked, status.IsUnlocked())
		require.Equal(t, locked, status.IsLocked())
		require.Equal(t, wiped, status.IsWiped())
		require.Equal(t, pendingLock, status.IsPendingLock())
		require.Equal(t, pendingUnlock, status.IsPendingUnlock())
		require.Equal(t, pendingWipe, status.IsPendingWipe())
	}

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	status, err := ds.GetHostLockWipeStatus(ctx, host.ID, "macos")
	require.NoError(t, err)

	// default state
	checkState(t, status, true, false, false, false, false, false)

	appleStore, err := ds.NewMDMAppleMDMStorage(nil, nil)
	require.NoError(t, err)

	// record a request to lock the host
	cmd := &mdm.Command{
		CommandUUID: "command-uuid",
		Raw:         []byte("<?xml"),
	}
	cmd.Command.RequestType = "DeviceLock"
	err = appleStore.EnqueueDeviceLockCommand(ctx, host, cmd, "123456")
	require.NoError(t, err)

	status, err = ds.GetHostLockWipeStatus(ctx, host.ID, host.FleetPlatform())
	require.NoError(t, err)
	checkState(t, status, true, false, false, false, true, false)

	// execute CleanMacOSMDMLock to simulate successful unlock
	err = ds.CleanMacOSMDMLock(ctx, host.UUID)
	require.NoError(t, err)

	status, err = ds.GetHostLockWipeStatus(ctx, host.ID, "macos")
	require.NoError(t, err)
	checkState(t, status, true, false, false, false, false, false)
	require.Empty(t, status.UnlockPIN)
}

func TestMDMAppleProfileVerification(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	now := time.Now()
	twoMinutesAgo := now.Add(-2 * time.Minute)
	twoHoursAgo := now.Add(-2 * time.Hour)
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)

	type testCase struct {
		name           string
		initialStatus  fleet.MDMDeliveryStatus
		expectedStatus fleet.MDMDeliveryStatus
		expectedDetail string
	}

	setupTestProfile := func(t *testing.T, suffix string) *fleet.MDMAppleConfigProfile {
		cp, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t,
			fmt.Sprintf("name-test-profile-%s", suffix),
			fmt.Sprintf("identifier-test-profile-%s", suffix),
			fmt.Sprintf("uuid-test-profile-%s", suffix)))
		require.NoError(t, err)
		return cp
	}

	setProfileUploadedAt := func(t *testing.T, cp *fleet.MDMAppleConfigProfile, ua time.Time) {
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, `UPDATE mdm_apple_configuration_profiles SET uploaded_at = ? WHERE profile_uuid = ?`, ua, cp.ProfileUUID)
			return err
		})
	}

	setRetries := func(t *testing.T, hostUUID string, retries uint) {
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET retries = ? WHERE host_uuid = ?`, retries, hostUUID)
			return err
		})
	}

	checkHostStatus := func(t *testing.T, h *fleet.Host, expectedStatus fleet.MDMDeliveryStatus, expectedDetail string) error {
		gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		if err != nil {
			return err
		}
		if len(gotProfs) != 1 {
			return errors.New("expected exactly one profile")
		}
		if gotProfs[0].Status == nil {
			return errors.New("expected status to be non-nil")
		}
		if *gotProfs[0].Status != expectedStatus {
			return fmt.Errorf("expected status %s, got %s", expectedStatus, *gotProfs[0].Status)
		}
		if gotProfs[0].Detail != expectedDetail {
			return fmt.Errorf("expected detail %s, got %s", expectedDetail, gotProfs[0].Detail)
		}
		return nil
	}

	initializeProfile := func(t *testing.T, h *fleet.Host, cp *fleet.MDMAppleConfigProfile, status fleet.MDMDeliveryStatus, prevRetries uint) {
		upsertHostCPs([]*fleet.Host{h}, []*fleet.MDMAppleConfigProfile{cp}, fleet.MDMOperationTypeInstall, &status, ctx, ds, t)
		require.NoError(t, checkHostStatus(t, h, status, ""))
		setRetries(t, h.UUID, prevRetries)
	}

	cleanupProfiles := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, `DELETE FROM mdm_apple_configuration_profiles; DELETE FROM host_mdm_apple_profiles`)
			return err
		})
	}

	t.Run("MissingProfileWithRetry", func(t *testing.T) {
		defer cleanupProfiles(t)
		// missing profile, verifying and verified statuses should change to failed after the grace
		// period and one retry
		cases := []testCase{
			{
				name:           "PendingThenMissing",
				initialStatus:  fleet.MDMDeliveryPending,
				expectedStatus: fleet.MDMDeliveryPending, // no change
			},
			{
				name:           "VerifyingThenMissing",
				initialStatus:  fleet.MDMDeliveryVerifying,
				expectedStatus: fleet.MDMDeliveryFailed, // change to failed
			},
			{
				name:           "VerifiedThenMissing",
				initialStatus:  fleet.MDMDeliveryVerified,
				expectedStatus: fleet.MDMDeliveryFailed, // change to failed
			},
			{
				name:           "FailedThenMissing",
				initialStatus:  fleet.MDMDeliveryFailed,
				expectedStatus: fleet.MDMDeliveryFailed, // no change
			},
		}

		for i, tc := range cases {
			// setup
			h := test.NewHost(t, ds, tc.name, tc.name, tc.name, tc.name, twoMinutesAgo)
			cp := setupTestProfile(t, fmt.Sprintf("%s-%d", tc.name, i))
			var reportedProfiles []*fleet.HostMacOSProfile // no profiles reported for this test

			// initialize
			initializeProfile(t, h, cp, tc.initialStatus, 0)

			// within grace period
			setProfileUploadedAt(t, cp, twoMinutesAgo)
			require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
			require.NoError(t, checkHostStatus(t, h, tc.initialStatus, "")) // if missing within grace period, no change

			// reinitialize
			initializeProfile(t, h, cp, tc.initialStatus, 0)

			// outside grace period
			setProfileUploadedAt(t, cp, twoHoursAgo)
			require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
			if tc.expectedStatus == fleet.MDMDeliveryFailed {
				// grace period expired, first failure gets retried so status should be pending and empty detail
				require.NoError(t, checkHostStatus(t, h, fleet.MDMDeliveryPending, ""), tc.name)
			}

			if tc.initialStatus != fleet.MDMDeliveryPending {
				// after retry, assume successful install profile command so status should be verifying
				upsertHostCPs([]*fleet.Host{h}, []*fleet.MDMAppleConfigProfile{cp}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
				// report osquery results
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				// now we see the expected status
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, string(fleet.HostMDMProfileDetailFailedWasVerifying)), tc.name) // grace period expired, max retries so check expected status
			}
		}
	})

	t.Run("OutdatedProfile", func(t *testing.T) {
		// found profile with the expected identifier, but it's outdated (i.e. the install date is
		// before the last update date) so treat it as missing the expected profile verifying and
		// verified statuses should change to failed after the grace period)
		cases := []testCase{
			{
				name:           "PendingThenFoundOutdated",
				initialStatus:  fleet.MDMDeliveryPending,
				expectedStatus: fleet.MDMDeliveryPending, // no change
				expectedDetail: "",
			},
			{
				name:           "VerifyingThenFoundOutdated",
				initialStatus:  fleet.MDMDeliveryVerifying,
				expectedStatus: fleet.MDMDeliveryFailed, // change to failed
				expectedDetail: string(fleet.HostMDMProfileDetailFailedWasVerifying),
			},
			{
				name:           "VerifiedThenFoundOutdated",
				initialStatus:  fleet.MDMDeliveryVerified,
				expectedStatus: fleet.MDMDeliveryFailed, // change to failed
				expectedDetail: string(fleet.HostMDMProfileDetailFailedWasVerified),
			},
			{
				name:           "FailedThenFoundOutdated",
				initialStatus:  fleet.MDMDeliveryFailed,
				expectedStatus: fleet.MDMDeliveryFailed, // no change
				expectedDetail: "",
			},
		}

		for i, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				defer cleanupProfiles(t)

				// setup
				h := test.NewHost(t, ds, tc.name, tc.name, tc.name, tc.name, twoMinutesAgo)
				cp := setupTestProfile(t, fmt.Sprintf("%s-%d", tc.name, i))
				reportedProfiles := []*fleet.HostMacOSProfile{
					{
						DisplayName: cp.Name,
						Identifier:  cp.Identifier,
						InstallDate: twoDaysAgo,
					},
				}

				// initialize with no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// within grace period
				setProfileUploadedAt(t, cp, twoMinutesAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.initialStatus, "")) // outdated profiles are treated similar to missing profiles so status doesn't change if within grace period

				// reinitalize with no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// outside grace period
				setProfileUploadedAt(t, cp, twoHoursAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, tc.expectedDetail)) // grace period expired, check expected status
			})
		}
	})

	t.Run("ExpectedProfile", func(t *testing.T) {
		// happy path, expected profile found so verifying should change to verified
		cases := []testCase{
			{
				name:           "PendingThenFoundExpected",
				initialStatus:  fleet.MDMDeliveryPending,
				expectedStatus: fleet.MDMDeliveryVerified, // pending can go to verified if found
				expectedDetail: "",
			},
			{
				name:           "VerifyingThenFoundExpected",
				initialStatus:  fleet.MDMDeliveryVerifying,
				expectedStatus: fleet.MDMDeliveryVerified, // change to verified
				expectedDetail: "",
			},
			{
				name:           "VerifiedThenFoundExpected",
				initialStatus:  fleet.MDMDeliveryVerified,
				expectedStatus: fleet.MDMDeliveryVerified, // no change
				expectedDetail: "",
			},
			{
				name:           "FailedThenFoundExpected",
				initialStatus:  fleet.MDMDeliveryFailed,
				expectedStatus: fleet.MDMDeliveryVerified, // failed can become verified if found later
				expectedDetail: "",
			},
		}

		for i, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				defer cleanupProfiles(t)

				// setup
				h := test.NewHost(t, ds, tc.name, tc.name, tc.name, tc.name, twoMinutesAgo)
				cp := setupTestProfile(t, fmt.Sprintf("%s-%d", tc.name, i))
				reportedProfiles := []*fleet.HostMacOSProfile{
					{
						DisplayName: cp.Name,
						Identifier:  cp.Identifier,
						InstallDate: now,
					},
				}

				// initialize with no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// within grace period
				setProfileUploadedAt(t, cp, twoMinutesAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, tc.expectedDetail)) // if found within grace period, verifying status can become verified so check expected status

				// reinitializewith no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// outside grace period
				setProfileUploadedAt(t, cp, twoHoursAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, tc.expectedDetail)) // grace period expired, check expected status
			})
		}
	})

	t.Run("UnexpectedProfile", func(t *testing.T) {
		// unexpected profile is ignored and doesn't change status of existing profile
		cases := []testCase{
			{
				name:           "PendingThenFoundExpectedAndUnexpected",
				initialStatus:  fleet.MDMDeliveryPending,
				expectedStatus: fleet.MDMDeliveryVerified, // profile can go from pending to verified
				expectedDetail: "",
			},
			{
				name:           "VerifyingThenFoundExpectedAndUnexpected",
				initialStatus:  fleet.MDMDeliveryVerifying,
				expectedStatus: fleet.MDMDeliveryVerified, // no change
				expectedDetail: "",
			},
			{
				name:           "VerifiedThenFounExpectedAnddUnexpected",
				initialStatus:  fleet.MDMDeliveryVerified,
				expectedStatus: fleet.MDMDeliveryVerified, // no change
				expectedDetail: "",
			},
			{
				name:           "FailedThenFoundExpectedAndUnexpected",
				initialStatus:  fleet.MDMDeliveryFailed,
				expectedStatus: fleet.MDMDeliveryVerified, // failed can become verified if found later
				expectedDetail: "",
			},
		}

		for i, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				defer cleanupProfiles(t)

				// setup
				h := test.NewHost(t, ds, tc.name, tc.name, tc.name, tc.name, twoMinutesAgo)
				cp := setupTestProfile(t, fmt.Sprintf("%s-%d", tc.name, i))
				reportedProfiles := []*fleet.HostMacOSProfile{
					{
						DisplayName: "unexpected-name",
						Identifier:  "unexpected-identifier",
						InstallDate: now,
					},
					{
						DisplayName: cp.Name,
						Identifier:  cp.Identifier,
						InstallDate: now,
					},
				}

				// initialize with no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// within grace period
				setProfileUploadedAt(t, cp, twoMinutesAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, tc.expectedDetail)) // if found within grace period, verifying status can become verified so check expected status

				// reinitialize with no remaining retries
				initializeProfile(t, h, cp, tc.initialStatus, 1)

				// outside grace period
				setProfileUploadedAt(t, cp, twoHoursAgo)
				require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
				require.NoError(t, checkHostStatus(t, h, tc.expectedStatus, tc.expectedDetail)) // grace period expired, check expected status
			})
		}
	})

	t.Run("EarliestInstallDate", func(t *testing.T) {
		defer cleanupProfiles(t)

		hostString := "host-earliest-install-date"
		h := test.NewHost(t, ds, hostString, hostString, hostString, hostString, twoMinutesAgo)

		cp := configProfileForTest(t,
			fmt.Sprintf("name-test-profile-%s", hostString),
			fmt.Sprintf("identifier-test-profile-%s", hostString),
			fmt.Sprintf("uuid-test-profile-%s", hostString))

		// save the config profile to no team
		stored0, err := ds.NewMDMAppleConfigProfile(ctx, *cp)
		require.NoError(t, err)

		reportedProfiles := []*fleet.HostMacOSProfile{
			{
				DisplayName: cp.Name,
				Identifier:  cp.Identifier,
				InstallDate: twoDaysAgo,
			},
		}
		initialStatus := fleet.MDMDeliveryVerifying

		// initialize with no remaining retries
		initializeProfile(t, h, stored0, initialStatus, 1)

		// within grace period
		setProfileUploadedAt(t, stored0, twoMinutesAgo) // host is out of date but still within grace period
		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
		require.NoError(t, checkHostStatus(t, h, fleet.MDMDeliveryVerifying, "")) // no change

		// reinitialize with no remaining retries
		initializeProfile(t, h, stored0, initialStatus, 1)

		// outside grace period
		setProfileUploadedAt(t, stored0, twoHoursAgo) // host is out of date and grace period has passed
		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
		require.NoError(t, checkHostStatus(t, h, fleet.MDMDeliveryFailed, string(fleet.HostMDMProfileDetailFailedWasVerifying))) // set to failed

		// reinitialize with no remaining retries
		initializeProfile(t, h, stored0, initialStatus, 1)

		// save a copy of the config profile to team 1
		cp.TeamID = ptr.Uint(1)
		stored1, err := ds.NewMDMAppleConfigProfile(ctx, *cp)
		require.NoError(t, err)

		setProfileUploadedAt(t, stored0, twoHoursAgo)                  // host would be out of date based on this copy of the profile record
		setProfileUploadedAt(t, stored1, twoDaysAgo.Add(-1*time.Hour)) // BUT this record now establishes the earliest install date

		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, h, profilesByIdentifier(reportedProfiles)))
		require.NoError(t, checkHostStatus(t, h, fleet.MDMDeliveryVerified, "")) // set to verified based on earliest install date
	})
}

func profilesByIdentifier(profiles []*fleet.HostMacOSProfile) map[string]*fleet.HostMacOSProfile {
	byIdentifier := map[string]*fleet.HostMacOSProfile{}
	for _, p := range profiles {
		byIdentifier[p.Identifier] = p
	}
	return byIdentifier
}

func TestRestorePendingDEPHost(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	ctx := context.Background()
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	expectedMDMServerURL, err := apple_mdm.ResolveAppleEnrollMDMURL(ac.ServerSettings.ServerURL)
	require.NoError(t, err)

	t.Run("DEP enrollment", func(t *testing.T) {
		checkHostExistsInTable := func(t *testing.T, tableName string, hostID uint, expected bool, where ...string) {
			stmt := "SELECT 1 FROM " + tableName + " WHERE host_id = ?"
			if len(where) != 0 {
				stmt += " AND " + strings.Join(where, " AND ")
			}
			var exists bool
			err := sqlx.GetContext(ctx, ds.primary, &exists, stmt, hostID)
			if expected {
				require.NoError(t, err, tableName)
				require.True(t, exists, tableName)
			} else {
				require.ErrorIs(t, err, sql.ErrNoRows, tableName)
				require.False(t, exists, tableName)
			}
		}

		checkStoredHost := func(t *testing.T, hostID uint, expectedHost *fleet.Host) {
			h, err := ds.Host(ctx, hostID)
			if expectedHost != nil {
				require.NoError(t, err)
				require.NotNil(t, h)
				require.Equal(t, expectedHost.ID, h.ID)
				require.Equal(t, expectedHost.OrbitNodeKey, h.OrbitNodeKey)
				require.Equal(t, expectedHost.HardwareModel, h.HardwareModel)
				require.Equal(t, expectedHost.HardwareSerial, h.HardwareSerial)
				require.Equal(t, expectedHost.UUID, h.UUID)
				require.Equal(t, expectedHost.Platform, h.Platform)
				require.Equal(t, expectedHost.TeamID, h.TeamID)
			} else {
				nfe := &notFoundError{}
				require.ErrorAs(t, err, &nfe)
			}

			for _, table := range []string{
				"host_mdm",
				"host_display_names",
				// "label_membership", // TODO: uncomment this if/when we add the builtin labels to the mysql test setup
			} {
				checkHostExistsInTable(t, table, hostID, expectedHost != nil)
			}

			// host DEP assignment row is NEVER deleted
			checkHostExistsInTable(t, "host_dep_assignments", hostID, true, "deleted_at IS NULL")
		}

		setupTestHost := func(t *testing.T) (pendingHost, mdmEnrolledHost *fleet.Host) {
			depSerial := "dep-serial"
			depUUID := "dep-uuid"
			depOrbitNodeKey := "dep-orbit-node-key"

			n, _, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{{SerialNumber: depSerial}})
			require.NoError(t, err)
			require.Equal(t, int64(1), n)

			var depHostID uint
			err = sqlx.GetContext(ctx, ds.reader(ctx), &depHostID, "SELECT id FROM hosts WHERE hardware_serial = ?", depSerial)
			require.NoError(t, err)

			// host MDM row is created when DEP device is ingested
			pendingHost, err = ds.Host(ctx, depHostID)
			require.NoError(t, err)
			require.NotNil(t, pendingHost)
			require.Equal(t, depHostID, pendingHost.ID)
			require.Equal(t, "Pending", *pendingHost.MDM.EnrollmentStatus)
			require.Equal(t, fleet.WellKnownMDMFleet, pendingHost.MDM.Name)
			require.Nil(t, pendingHost.OsqueryHostID)

			// host DEP assignment is created when DEP device is ingested
			depAssignment, err := ds.GetHostDEPAssignment(ctx, depHostID)
			require.NoError(t, err)
			require.Equal(t, depHostID, depAssignment.HostID)
			require.Nil(t, depAssignment.DeletedAt)
			require.WithinDuration(t, time.Now(), depAssignment.AddedAt, 5*time.Second)

			// simulate initial osquery enrollment via Orbit
			h, err := ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
				HardwareSerial: depSerial,
				Platform:       "darwin",
				HardwareUUID:   depUUID,
				Hostname:       "dep-host",
			}, depOrbitNodeKey, nil)
			require.NoError(t, err)
			require.NotNil(t, h)
			require.Equal(t, depHostID, h.ID)

			// simulate osquery report of MDM detail query
			err = ds.SetOrUpdateMDMData(ctx, depHostID, false, true, expectedMDMServerURL, true, fleet.WellKnownMDMFleet, "")
			require.NoError(t, err)

			// enrollment status changes to "On (automatic)"
			mdmEnrolledHost, err = ds.Host(ctx, depHostID)
			require.NoError(t, err)
			require.Equal(t, "On (automatic)", *mdmEnrolledHost.MDM.EnrollmentStatus)
			require.Equal(t, fleet.WellKnownMDMFleet, mdmEnrolledHost.MDM.Name)
			require.Equal(t, depUUID, *mdmEnrolledHost.OsqueryHostID)

			return pendingHost, mdmEnrolledHost
		}

		pendingHost, mdmEnrolledHost := setupTestHost(t)
		require.Equal(t, pendingHost.ID, mdmEnrolledHost.ID)
		checkStoredHost(t, mdmEnrolledHost.ID, mdmEnrolledHost)

		// delete the host from Fleet
		err = ds.DeleteHost(ctx, mdmEnrolledHost.ID)
		require.NoError(t, err)
		checkStoredHost(t, mdmEnrolledHost.ID, nil)

		// host is restored
		err = ds.RestoreMDMApplePendingDEPHost(ctx, mdmEnrolledHost)
		require.NoError(t, err)
		expectedHost := *pendingHost
		// host uuid is preserved for restored hosts. It isn't available via DEP so the original
		// pending host record did not include it so we add it to our expected host here.
		expectedHost.UUID = mdmEnrolledHost.UUID
		checkStoredHost(t, mdmEnrolledHost.ID, &expectedHost)
	})
}

func createRawAppleCmd(reqType, cmdUUID string) string {
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

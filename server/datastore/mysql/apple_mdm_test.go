package mysql

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		{"TestHostDetailsMDMProfilesIOSIPadOS", testHostDetailsMDMProfilesIOSIPadOS},
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
		{"TestMDMAppleResetEnrollment", testMDMAppleResetEnrollment},
		{"TestMDMAppleDeleteHostDEPAssignments", testMDMAppleDeleteHostDEPAssignments},
		{"LockUnlockWipeMacOS", testLockUnlockWipeMacOS},
		{"ScreenDEPAssignProfileSerialsForCooldown", testScreenDEPAssignProfileSerialsForCooldown},
		{"MDMAppleDDMDeclarationsToken", testMDMAppleDDMDeclarationsToken},
		{"MDMAppleSetPendingDeclarationsAs", testMDMAppleSetPendingDeclarationsAs},
		{"SetOrUpdateMDMAppleDeclaration", testSetOrUpdateMDMAppleDDMDeclaration},
		{"DEPAssignmentUpdates", testMDMAppleDEPAssignmentUpdates},
		{"TestMDMConfigAsset", testMDMConfigAsset},
		{"ListIOSAndIPadOSToRefetch", testListIOSAndIPadOSToRefetch},
		{"MDMAppleUpsertHostIOSiPadOS", testMDMAppleUpsertHostIOSIPadOS},
		{"IngestMDMAppleDevicesFromDEPSyncIOSIPadOS", testIngestMDMAppleDevicesFromDEPSyncIOSIPadOS},
		{"MDMAppleProfilesOnIOSIPadOS", testMDMAppleProfilesOnIOSIPadOS},
		{"GetHostUUIDsWithPendingMDMAppleCommands", testGetHostUUIDsWithPendingMDMAppleCommands},
		{"MDMAppleBootstrapPackageWithS3", testMDMAppleBootstrapPackageWithS3},
		{"GetAndUpdateABMToken", testMDMAppleGetAndUpdateABMToken},
		{"ABMTokensTermsExpired", testMDMAppleABMTokensTermsExpired},
		{"TestMDMGetABMTokenOrgNamesAssociatedWithTeam", testMDMGetABMTokenOrgNamesAssociatedWithTeam},
		{"HostMDMCommands", testHostMDMCommands},
		{"IngestMDMAppleDeviceFromOTAEnrollment", testIngestMDMAppleDeviceFromOTAEnrollment},
		{"MDMManagedCertificates", testMDMManagedCertificates},
		{"AppleMDMSetBatchAsyncLastSeenAt", testAppleMDMSetBatchAsyncLastSeenAt},
		{"TestMDMAppleProfileLabels", testMDMAppleProfileLabels},
		{"AggregateMacOSSettingsAllPlatforms", testAggregateMacOSSettingsAllPlatforms},
	}

	for _, c := range cases {
		t.Helper()
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
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
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
	cp.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
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
	require.Nil(t, storedCP.LabelsIncludeAll)
	storedCP, err = ds.GetMDMAppleConfigProfile(ctx, newCP.ProfileUUID)
	require.NoError(t, err)
	checkConfigProfile(t, *newCP, *storedCP)
	require.Nil(t, storedCP.LabelsIncludeAll)

	// create a label-based profile
	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "lbl", Query: "select 1"})
	require.NoError(t, err)

	labelCP := fleet.MDMAppleConfigProfile{
		Name:         "label-based",
		Identifier:   "label-based",
		Mobileconfig: mobileconfig.Mobileconfig([]byte("LabelTestMobileconfigBytes")),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelName: lbl.Name, LabelID: lbl.ID},
		},
	}
	labelProf, err := ds.NewMDMAppleConfigProfile(ctx, labelCP)
	require.NoError(t, err)

	// get it back from both the deprecated ID and the uuid methods, labels are
	// only included in the uuid one
	prof, err := ds.GetMDMAppleConfigProfileByDeprecatedID(ctx, labelProf.ProfileID)
	require.NoError(t, err)
	require.Nil(t, prof.LabelsIncludeAll)
	prof, err = ds.GetMDMAppleConfigProfile(ctx, labelProf.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.LabelsIncludeAll, 1)
	require.Equal(t, lbl.Name, prof.LabelsIncludeAll[0].LabelName)
	require.False(t, prof.LabelsIncludeAll[0].Broken)

	// break the profile by deleting the label
	require.NoError(t, ds.DeleteLabel(ctx, lbl.Name))

	prof, err = ds.GetMDMAppleConfigProfile(ctx, labelProf.ProfileUUID)
	require.NoError(t, err)
	require.Len(t, prof.LabelsIncludeAll, 1)
	require.Equal(t, lbl.Name, prof.LabelsIncludeAll[0].LabelName)
	require.True(t, prof.LabelsIncludeAll[0].Broken)
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

	// delete by name via a non-existing name is not an error
	err = ds.DeleteMDMAppleDeclarationByName(ctx, nil, "test")
	require.NoError(t, err)

	testDecl := declForTest("D1", "D1", "{}")
	dbDecl, err := ds.NewMDMAppleDeclaration(ctx, testDecl)
	require.NoError(t, err)

	// delete for a non-existing team does nothing
	err = ds.DeleteMDMAppleDeclarationByName(ctx, ptr.Uint(1), dbDecl.Name)
	require.NoError(t, err)
	// ddm still exists
	_, err = ds.GetMDMAppleDeclaration(ctx, dbDecl.DeclarationUUID)
	require.NoError(t, err)

	// properly delete
	err = ds.DeleteMDMAppleDeclarationByName(ctx, nil, dbDecl.Name)
	require.NoError(t, err)
	_, err = ds.GetMDMAppleDeclaration(ctx, dbDecl.DeclarationUUID)
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

	// The pending profile will be NOT be cleaned up because it was updated too recently.
	err = ds.CleanupHostMDMAppleProfiles(ctx)
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

	// Update the timestamps of the profiles
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET updated_at = updated_at - INTERVAL 2 HOUR`)
		return err
	})

	// The pending profile will be cleaned up because we did not populate the corresponding nano table in this test.
	err = ds.CleanupHostMDMAppleProfiles(ctx)
	require.NoError(t, err)
	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, h1.UUID)
	require.NoError(t, err)
	require.Len(t, gotProfs, 1)
	assert.Equal(t, &fleet.MDMDeliveryVerifying, gotProfs[0].Status)
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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 4, n) // 4 new hosts ("abc", "xyz", "ijk", "tuv")

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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, nil, nil, nil)
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
	ac.MDM.DeprecatedAppleBMDefaultTeam = team.Name
	err = ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	depDevices = []godep.Device{
		{SerialNumber: "abc", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
		{SerialNumber: "xyz", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}

	n, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, team, team, team)
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

	nonExistentTeam := &fleet.Team{ID: 8888}
	depDevices = []godep.Device{
		{SerialNumber: "jqk", Model: "MacBook Pro", OS: "OSX", OpType: "added"},
	}

	n, err = ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, nonExistentTeam, nonExistentTeam, nonExistentTeam)
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
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)
	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
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

	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
		Platform:       "darwin",
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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	// simulate a host that is first ingested via DEP (e.g., the device was added via Apple Business Manager)
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	}, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	// hosts that are first ingested via DEP will have a serial number but not a UUID because UUID
	// is not available from the DEP sync endpoint
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, "", hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)

	// now simulate the initial MDM checkin by that same host
	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
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
	err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
		HardwareModel:  testModel,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)
	checkMDMHostRelatedTables(t, ds, hosts[0].ID, testSerial, testModel)

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	// no effect if same host appears in DEP sync
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{
		{SerialNumber: testSerial, Model: testModel, OS: "OSX", OpType: "added"},
	}, abmToken.ID, nil, nil, nil)
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

	err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
	})
	require.NoError(t, err)

	hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
	require.Equal(t, testSerial, hosts[0].HardwareSerial)
	require.Equal(t, testUUID, hosts[0].UUID)

	// duplicate Authenticate request has no effect
	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
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
	err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           testUUID,
		HardwareSerial: testSerial,
		Platform:       "darwin",
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

	err = ds.MDMTurnOff(ctx, testUUID)
	require.NoError(t, err)

	err = sqlx.GetContext(context.Background(), ds.reader(context.Background()), &count, `SELECT COUNT(*) FROM host_mdm WHERE host_id = ?`, testUUID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	hostProfs, err = ds.GetHostMDMAppleProfiles(ctx, testUUID)
	require.NoError(t, err)
	require.Empty(t, hostProfs)
	key, err = ds.GetHostDiskEncryptionKey(ctx, hostID)
	require.NoError(t, err)
	require.NotNil(t, key)
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
		gotp.SecretsUpdatedAt = nil

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

func expectAppleDeclarations(
	t *testing.T,
	ds *Datastore,
	tmID *uint,
	want []*fleet.MDMAppleDeclaration,
) map[string]string {
	if tmID == nil {
		tmID = ptr.Uint(0)
	}

	var got []*fleet.MDMAppleDeclaration
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		ctx := context.Background()
		return sqlx.SelectContext(ctx, q, &got,
			`SELECT declaration_uuid, team_id, identifier, name, raw_json, token, created_at, uploaded_at FROM mdm_apple_declarations WHERE team_id = ?`,
			tmID)
	})

	// create map of expected declarations keyed by identifier
	wantMap := make(map[string]*fleet.MDMAppleDeclaration, len(want))
	for _, cp := range want {
		wantMap[cp.Identifier] = cp
	}

	JSONRemarshal := func(bytes []byte) ([]byte, error) {
		var ifce interface{}
		err := json.Unmarshal(bytes, &ifce)
		if err != nil {
			return nil, err
		}
		return json.Marshal(ifce)
	}

	// compare only the fields we care about, and build the resulting map of
	// declaration identifier as key to declaration UUID as value
	m := make(map[string]string)
	for _, gotD := range got {

		wantD := wantMap[gotD.Identifier]

		m[gotD.Identifier] = gotD.DeclarationUUID
		if gotD.TeamID != nil && *gotD.TeamID == 0 {
			gotD.TeamID = nil
		}

		// DeclarationUUID is non-empty and starts with "d", but otherwise we don't
		// care about it for test assertions.
		require.NotEmpty(t, gotD.DeclarationUUID)
		require.True(t, strings.HasPrefix(gotD.DeclarationUUID, fleet.MDMAppleDeclarationUUIDPrefix))
		gotD.DeclarationUUID = ""
		gotD.Token = "" // don't care about md5checksum here

		gotD.CreatedAt = time.Time{}

		gotBytes, err := JSONRemarshal(gotD.RawJSON)
		require.NoError(t, err)

		wantBytes, err := JSONRemarshal(wantD.RawJSON)
		require.NoError(t, err)

		require.Equal(t, wantBytes, gotBytes)

		// if an expected uploaded_at timestamp is provided for this declaration, keep
		// its value, otherwise clear it as we don't care about asserting its
		// value.
		if wantD.UploadedAt.IsZero() {
			gotD.UploadedAt = time.Time{}
		}

		require.Equal(t, wantD.Name, gotD.Name)
		require.Equal(t, wantD.Identifier, gotD.Identifier)
		require.Equal(t, wantD.LabelsIncludeAll, gotD.LabelsIncludeAll)
	}
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

// If the label name starts with "exclude-", the label is considered an "exclude-any". If it starts
// with "include-any", it is considered an "include-any". Otherwise it is an "include-all".
func configProfileForTest(t *testing.T, name, identifier, uuid string, labels ...*fleet.Label) *fleet.MDMAppleConfigProfile {
	prof := configProfileBytesForTest(name, identifier, uuid)
	cp, err := fleet.NewMDMAppleConfigProfile(prof, nil)
	require.NoError(t, err)
	sum := md5.Sum(prof) // nolint:gosec // used only to hash for efficient comparisons
	cp.Checksum = sum[:]

	for _, lbl := range labels {
		switch {
		case strings.HasPrefix(lbl.Name, "exclude-"):
			cp.LabelsExcludeAny = append(cp.LabelsExcludeAny, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		case strings.HasPrefix(lbl.Name, "include-any-"):
			cp.LabelsIncludeAny = append(cp.LabelsIncludeAny, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		default:
			cp.LabelsIncludeAll = append(cp.LabelsIncludeAll, fleet.ConfigurationProfileLabel{LabelName: lbl.Name, LabelID: lbl.ID})
		}
	}

	return cp
}

// if the label name starts with "exclude-", the label is considered an "exclude-any", otherwise
// it is an "include-all".
func declForTest(name, identifier, payloadContent string, labels ...*fleet.Label) *fleet.MDMAppleDeclaration {
	tmpl := `{
		"Type": "com.apple.configuration.decl%s",
		"Identifier": "com.fleet.config%s",
		"Payload": {
			"ServiceType": "com.apple.service%s"
		}
	}`

	declBytes := []byte(fmt.Sprintf(tmpl, identifier, identifier, payloadContent))

	decl := &fleet.MDMAppleDeclaration{
		RawJSON:    declBytes,
		Identifier: fmt.Sprintf("com.fleet.config%s", identifier),
		Name:       name,
	}

	for _, l := range labels {
		if strings.HasPrefix(l.Name, "exclude-") {
			decl.LabelsExcludeAny = append(decl.LabelsExcludeAny, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		} else {
			decl.LabelsIncludeAll = append(decl.LabelsIncludeAll, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		}
	}

	return decl
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
	ds.testSelectMDMProfilesBatchSize = 2
	ds.testUpsertMDMDesiredProfilesBatchSize = 2
	t.Cleanup(func() {
		ds.testSelectMDMProfilesBatchSize = 0
		ds.testUpsertMDMDesiredProfilesBatchSize = 0
	})
	testMDMAppleProfileManagement(t, ds)
}

func testMDMAppleProfileManagementBatch3(t *testing.T, ds *Datastore) {
	ds.testSelectMDMProfilesBatchSize = 3
	ds.testUpsertMDMDesiredProfilesBatchSize = 3
	t.Cleanup(func() {
		ds.testSelectMDMProfilesBatchSize = 0
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
			p.SecretsUpdatedAt = nil
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

	// if there are no hosts, then no profilesToInstall need to be installed
	profilesToInstall, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profilesToInstall)

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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
	}, profilesToInstall)

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

	profiles, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)

	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2", HostPlatform: "darwin"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2", HostPlatform: "darwin"},
	}, profilesToInstall)

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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-2", HostPlatform: "darwin"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-2", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[0].ProfileUUID, ProfileIdentifier: globalPfs[0].Identifier, ProfileName: globalPfs[0].Name, HostUUID: "test-uuid-3", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[1].ProfileUUID, ProfileIdentifier: globalPfs[1].Identifier, ProfileName: globalPfs[1].Name, HostUUID: "test-uuid-3", HostPlatform: "darwin"},
		{ProfileUUID: globalPfs[2].ProfileUUID, ProfileIdentifier: globalPfs[2].Identifier, ProfileName: globalPfs[2].Name, HostUUID: "test-uuid-3", HostPlatform: "darwin"},
	}, profilesToInstall)

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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profilesToInstall)

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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profilesToInstall)

	// still no profiles to remove
	toRemove, err = ds.ListMDMAppleProfilesToRemove(ctx)
	require.NoError(t, err)
	require.Empty(t, toRemove)

	// add host1 to team
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host1.ID})
	require.NoError(t, err)

	// profiles to be added for host1 are now related to the team
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: teamPfs[0].ProfileUUID, ProfileIdentifier: teamPfs[0].Identifier, ProfileName: teamPfs[0].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: teamPfs[1].ProfileUUID, ProfileIdentifier: teamPfs[1].Identifier, ProfileName: teamPfs[1].Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
	}, profilesToInstall)

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
	// Labels are deleted when truncating tables in between tests.
	// We need to delete the iOS/iPadOS labels because these two are created on a table migration,
	// and also we want to keep their indexes higher than "All Hosts" and "macOS" (to not break existing tests).
	_, err := ds.writer(context.Background()).Exec(`
		DELETE FROM labels WHERE name = 'iOS' OR name = 'iPadOS'`,
	)
	require.NoError(t, err)

	_, err = ds.writer(context.Background()).Exec(`
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
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
		"iOS",
		"",
		"",
		"",
		fleet.LabelTypeBuiltIn,
		"iPadOS",
		"",
		"",
		"",
		fleet.LabelTypeBuiltIn,
	)
	require.NoError(t, err)
}

func nanoEnrollAndSetHostMDMData(t *testing.T, ds *Datastore, host *fleet.Host, withUser bool) {
	ctx := context.Background()
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	expectedMDMServerURL, err := apple_mdm.ResolveAppleEnrollMDMURL(ac.ServerSettings.ServerURL)
	require.NoError(t, err)
	nanoEnroll(t, ds, host, withUser)
	err = ds.SetOrUpdateMDMData(ctx, host.ID, false, true, expectedMDMServerURL, true, fleet.WellKnownMDMFleet, "")
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

		gotHosts, err := ds.ListHosts(
			ctx,
			fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String("admin")}},
			fleet.HostListOptions{MacOSSettingsFilter: status, TeamFilter: teamID},
		)
		gotIDs := []uint{}
		for _, h := range gotHosts {
			gotIDs = append(gotIDs, h.ID)
		}

		return assert.NoError(t, err) &&
			assert.Len(t, gotHosts, len(expected)) &&
			assert.ElementsMatch(t, expectedIDs, gotIDs)
	}

	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now())
		hosts = append(hosts, h)
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
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
	require.EqualValues(t, len(hosts), res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert all but filevault to verifying
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending) // still pending because filevault not installed
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert all but filevault to verified
	upsertHostCPs(hosts, noTeamCPs, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerified, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending) // still pending because filevault not installed
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	// upsert filevault to pending
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending) // still pending because filevault pending
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{fvNoTeam}, fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, ctx, ds, t)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending) // still pending because no disk encryption key
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, hosts[0].ID, "foo", "", nil)
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	// hosts still pending because disk encryption key decryptable is not set
	require.EqualValues(t, len(hosts)-1, res.Pending)
	require.Equal(t, uint(0), res.Failed)
	// one host is verifying because the disk is encrypted and we're verifying the key
	require.Equal(t, uint(1), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, false, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending) // still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(0), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[0].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-1, res.Pending)
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(1), res.Verifying) // hosts[0] now has filevault fully enforced but not verified
	require.Equal(t, uint(0), res.Verified)

	// upsert hosts[0] filevault to verified
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, ds, hosts[0], profilesByIdentifier([]*fleet.HostMacOSProfile{{Identifier: fvNoTeam.Identifier, DisplayName: fvNoTeam.Name, InstallDate: time.Now()}})))
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-1, res.Pending)
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
	require.EqualValues(t, len(hosts)-1, res.Pending) // hosts[1] still pending because disk encryption key decryptable is false
	require.Equal(t, uint(0), res.Failed)
	require.Equal(t, uint(0), res.Verifying)
	require.Equal(t, uint(1), res.Verified)

	err = ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{hosts[1].ID}, true, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-2, res.Pending)
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
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
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
	require.EqualValues(t, len(hosts), res.Pending) // each host only counts once
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
	require.EqualValues(t, len(hosts), res.Pending) // each host only counts once
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
	require.EqualValues(t, len(hosts)-2, res.Pending) // two hosts are failing at least one profile (hosts[0] and hosts[1])
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
	require.EqualValues(t, len(hosts)-2, res.Pending) // no change
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
	require.NoError(t, ds.deleteMDMOSCustomSettingsForHost(ctx, tx, hosts[6].UUID, "darwin"))
	require.NoError(t, tx.Commit())
	pendingHosts := hosts[2:6:6]
	pendingHosts = append(pendingHosts, hosts[7:]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-3, res.Pending) // hosts[6] not reported here anymore
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
	pendingHosts = hosts[2:6:6]
	pendingHosts = append(pendingHosts, hosts[7:]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-3, res.Pending) // hosts[6] not reported here anymore, hosts[9] still pending
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
	pendingHosts = hosts[2:6:6]
	pendingHosts = append(pendingHosts, hosts[7:9]...)
	res, err = ds.GetMDMAppleProfilesSummary(ctx, nil) // get summary for profiles with no team
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts)-4, res.Pending) // subtract hosts[6 and 9] from pending
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
	pendingHosts = hosts[2:6:6]
	pendingHosts = append(pendingHosts, hosts[7:9]...)
	require.EqualValues(t, len(hosts)-4, res.Pending) // hosts[9] is still not pending, transferred to team
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
	pendingHosts = hosts[2:6:6]
	pendingHosts = append(pendingHosts, hosts[7:9]...)
	require.EqualValues(t, len(hosts)-4, res.Pending) // subtract two failed hosts, one without profiles and hosts[9] transferred
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
	require.NoError(t, ds.deleteMDMOSCustomSettingsForHost(ctx, tx, h.UUID, "darwin"))
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
	upsertHostCPs(
		[]*fleet.Host{verifyingHost},
		[]*fleet.MDMAppleConfigProfile{noTeamFVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryVerifying,
		ctx, ds, t,
	)
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
	require.NotNil(t, allProfilesSummary)
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
	require.NotNil(t, allProfilesSummary)
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
	require.NotNil(t, allProfilesSummary)
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
	require.NotNil(t, allProfilesSummary)
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
	require.NotNil(t, allProfilesSummary)
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
	require.NotNil(t, allProfilesSummary)
	require.Equal(t, uint(0), allProfilesSummary.Pending)
	require.Equal(t, uint(0), allProfilesSummary.Failed)
	require.Equal(t, uint(1), allProfilesSummary.Verifying)
	require.Equal(t, uint(0), allProfilesSummary.Verified)

	// verified status
	upsertHostCPs(
		[]*fleet.Host{verifyingTeam1Host},
		[]*fleet.MDMAppleConfigProfile{team1FVProfile},
		fleet.MDMOperationTypeInstall,
		&fleet.MDMDeliveryVerified,
		ctx, ds, t,
	)
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
	require.NotNil(t, allProfilesSummary)
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
	mdmStorage, err := ds.NewMDMAppleMDMStorage()
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

	err := ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{}, nil)
	require.Error(t, err)

	bp1 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(0),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1, nil)
	require.NoError(t, err)

	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1, nil)
	require.ErrorAs(t, err, &aerr)

	bp2 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(2),
		Name:   t.Name(),
		Sha256: sha256.New().Sum(nil),
		Bytes:  []byte("content"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp2, nil)
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

	bytes, err := ds.GetMDMAppleBootstrapPackageBytes(ctx, bp1.Token, nil)
	require.NoError(t, err)
	require.Equal(t, bp1.Bytes, bytes.Bytes)

	bytes, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, "fake", nil)
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
		Raw:         []byte(rawCmd2),
	})
	require.NoError(t, err)
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[2].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
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
	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, nil, "no-such-token")
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

	// create an ABM token and set a profile uuid for no team
	tok1, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "o1", EncryptedToken: []byte(uuid.NewString())})
	require.NoError(t, err)
	require.NotZero(t, tok1.ID)
	profUUID1 := uuid.NewString()
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, nil, profUUID1, "o1")
	require.NoError(t, err)
	gotProf, gotTs, err := ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, nil, "o1")
	require.NoError(t, err)
	require.Equal(t, profUUID1, gotProf)
	require.NotZero(t, gotTs)

	// set a profile uuid for an unknown token, no error but nothing inserted
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, nil, profUUID1, "no-such-token")
	require.NoError(t, err)
	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, nil, "no-such-token")
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// create another ABM token and set a profile uuid for the team assistant
	// with both tokens
	tok2, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "o2", EncryptedToken: []byte(uuid.NewString())})
	require.NoError(t, err)
	require.NotZero(t, tok2.ID)
	profUUID2 := uuid.NewString()
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, profUUID1, "o1")
	require.NoError(t, err)
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, profUUID2, "o2")
	require.NoError(t, err)
	gotProf, gotTs, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o1")
	require.NoError(t, err)
	require.Equal(t, profUUID1, gotProf)
	require.NotZero(t, gotTs)
	gotProf, gotTs, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o2")
	require.NoError(t, err)
	require.Equal(t, profUUID2, gotProf)
	require.NotZero(t, gotTs)

	// update the profile uuid for o2 only
	profUUID3 := uuid.NewString()
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, profUUID3, "o2")
	require.NoError(t, err)
	gotProf, gotTs, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o1")
	require.NoError(t, err)
	require.Equal(t, profUUID1, gotProf)
	require.NotZero(t, gotTs)
	gotProf, gotTs, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o2")
	require.NoError(t, err)
	require.Equal(t, profUUID3, gotProf)
	require.NotZero(t, gotTs)

	// upsert team assistant
	tmAsst2, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":2}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst2.ID, tmAsst.ID)
	require.False(t, tmAsst2.UploadedAt.Before(tmAsst.UploadedAt)) // after or equal
	require.Equal(t, tmAsst.TeamID, tmAsst2.TeamID)
	require.Equal(t, "test2", tmAsst2.Name)
	require.JSONEq(t, `{"x": 2}`, string(tmAsst2.Profile))

	// profile uuids have been cleared
	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o1")
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o2")
	require.ErrorIs(t, err, sql.ErrNoRows)

	// upsert no team assistant
	noTeamAsst2, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{Name: "test3", Profile: json.RawMessage(`{"x": 3}`)})
	require.NoError(t, err)
	require.Equal(t, noTeamAsst2.ID, noTeamAsst.ID)
	require.False(t, noTeamAsst2.UploadedAt.Before(noTeamAsst.UploadedAt)) // after or equal
	require.Nil(t, noTeamAsst2.TeamID)
	require.Equal(t, "test3", noTeamAsst2.Name)
	require.JSONEq(t, `{"x": 3}`, string(noTeamAsst2.Profile))

	// profile uuid has been cleared
	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, nil, "o1")
	require.ErrorIs(t, err, sql.ErrNoRows)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// set profile uuids for team and no team (one each)
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, nil, profUUID1, "o1")
	require.NoError(t, err)
	err = ds.SetMDMAppleSetupAssistantProfileUUID(ctx, &tm.ID, profUUID2, "o2")
	require.NoError(t, err)

	// upsert team no change, uploaded at timestamp does not change
	tmAsst3, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":2}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst2, tmAsst3)

	// TODO(mna): ideally the profiles would not be cleared when the profile
	// stayed the same, but does not work at the moment and we're pressed by
	// time.
	// gotProf, gotTs, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o2")
	// require.NoError(t, err)
	// require.Equal(t, profUUID2, gotProf)
	// require.Equal(t, tmAsst3.UploadedAt, gotTs)

	time.Sleep(time.Second) // ensures the timestamp checks are not by chance

	// upsert team with a change, clears the profile uuid and updates the uploaded at timestamp
	tmAsst4, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm.ID, Name: "test2", Profile: json.RawMessage(`{"x":3}`)})
	require.NoError(t, err)
	require.Equal(t, tmAsst3.ID, tmAsst4.ID)
	require.True(t, tmAsst4.UploadedAt.After(tmAsst3.UploadedAt))
	require.Equal(t, tmAsst3.TeamID, tmAsst4.TeamID)
	require.Equal(t, "test2", tmAsst4.Name)
	require.JSONEq(t, `{"x": 3}`, string(tmAsst4.Profile))

	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, "o2")
	require.ErrorIs(t, err, sql.ErrNoRows)

	// delete no team
	err = ds.DeleteMDMAppleSetupAssistant(ctx, nil)
	require.NoError(t, err)

	_, _, err = ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, nil, "o1")
	require.ErrorIs(t, err, sql.ErrNoRows)

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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

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
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID)
			require.NoError(t, err)
		case i == 4:
			// not ABM assigned
		case i == 5:
			// ABM assignment was deleted
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID)
			require.NoError(t, err)
			err = ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{h.HardwareSerial})
			require.NoError(t, err)
		case i == 6:
			// assigned in ABM, but we don't have a serial
			err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID)
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

	// create a couple ABM tokens
	tok1, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "o1", EncryptedToken: []byte(uuid.NewString())})
	require.NoError(t, err)
	require.NotEmpty(t, tok1.ID)
	tok2, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "o2", EncryptedToken: []byte(uuid.NewString())})
	require.NoError(t, err)
	require.NotEmpty(t, tok2.ID)

	// get non-existing
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, nil, "no-such-token")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// set for no team
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, nil, "no-team", "o1")
	require.NoError(t, err)

	// get for no team returns the same data
	uuid, ts, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, nil, "o1")
	require.NoError(t, err)
	require.Equal(t, "no-team", uuid)
	require.NotZero(t, ts)

	// set for non-existing team fails
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, ptr.Uint(123), "xyz", "o2")
	require.Error(t, err)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// get for non-existing team fails
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, ptr.Uint(123), "o2")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// create a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm"})
	require.NoError(t, err)

	// set a couple profiles for existing team
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, &tm.ID, "tm1", "o1")
	require.NoError(t, err)
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, &tm.ID, "tm2", "o2")
	require.NoError(t, err)

	// get for existing team
	uuid, ts, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, "o1")
	require.NoError(t, err)
	require.Equal(t, "tm1", uuid)
	require.NotZero(t, ts)
	uuid, ts, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, "o2")
	require.NoError(t, err)
	require.Equal(t, "tm2", uuid)
	require.NotZero(t, ts)
	// get for unknown abm token
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, "no-such-token")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))

	// clear all profiles for team
	err = ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, &tm.ID, "", "")
	require.NoError(t, err)
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, "o1")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, "o2")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.True(t, fleet.IsNotFound(err))
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
	err = ds.InsertMDMAppleBootstrapPackage(ctx, defaultBP, nil)
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
	err = ds.InsertMDMAppleBootstrapPackage(ctx, defaultBP2, nil)
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
	tc.Config.WebhookSettings.FailingPoliciesWebhook = &fleet.FailingPoliciesWebhookSettings{
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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	t.Run("DEP enrollment", func(t *testing.T) {
		depSerial := "dep-serial"
		depUUID := "dep-uuid"
		depOrbitNodeKey := "dep-orbit-node-key"
		depDeviceTok := "dep-device-token"

		n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{{SerialNumber: depSerial}}, abmToken.ID, nil, nil, nil)
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
		require.NotNil(t, depAssignment.ABMTokenID)
		require.Equal(t, *depAssignment.ABMTokenID, abmToken.ID)

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
		require.NoError(t, ds.MDMTurnOff(ctx, depUUID))

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

		err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{HardwareSerial: manualSerial, UUID: manualUUID})
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

func testMDMAppleResetEnrollment(t *testing.T, ds *Datastore) {
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
	err = ds.MDMResetEnrollment(ctx, host.UUID)
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
	}, nil)
	require.NoError(t, err)

	// host has no boostrap package command yet
	_, err = ds.GetHostBootstrapPackageCommand(ctx, host.UUID)
	require.Error(t, err)
	nfe := &notFoundError{}
	require.ErrorAs(t, err, &nfe)

	err = ds.RecordHostBootstrapPackage(ctx, "command-uuid", host.UUID)
	require.NoError(t, err)
	// add a record of the host DEP assignment
	_, err = ds.writer(ctx).Exec(`
		INSERT INTO host_dep_assignments (host_id)
		VALUES (?)
		ON DUPLICATE KEY UPDATE added_at = CURRENT_TIMESTAMP, deleted_at = NULL
	`, host.ID)
	require.NoError(t, err)
	cmd, err := ds.GetHostBootstrapPackageCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Equal(t, "command-uuid", cmd)
	err = ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "foo.mdm.example.com", true, "", "")
	require.NoError(t, err)

	sum, err := ds.GetMDMAppleBootstrapPackageSummary(ctx, uint(0))
	require.NoError(t, err)
	require.Zero(t, sum.Failed)
	require.Zero(t, sum.Pending)
	require.EqualValues(t, 1, sum.Installed)

	// reset the enrollment
	err = ds.MDMResetEnrollment(ctx, host.UUID)
	require.NoError(t, err)

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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

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
			_, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, devices, abmToken.ID, nil, nil, nil)
			require.NoError(t, err)

			err = ds.DeleteHostDEPAssignments(ctx, abmToken.ID, tt.in)
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

func testLockUnlockWipeMacOS(t *testing.T, ds *Datastore) {
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
	nanoEnroll(t, ds, host, false)

	status, err := ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)

	// default state
	checkLockWipeState(t, status, true, false, false, false, false, false)

	appleStore, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	// record a request to lock the host
	cmd := &mdm.Command{
		CommandUUID: "command-uuid",
		Raw:         []byte("<?xml"),
	}
	cmd.Command.RequestType = "DeviceLock"
	err = appleStore.EnqueueDeviceLockCommand(ctx, host, cmd, "123456")
	require.NoError(t, err)

	// it is now pending lock
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, true, false, false, false, true, false)

	// record a command result to simulate locked state
	err = appleStore.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: host.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: cmd.CommandUUID,
		Status:      "Acknowledged",
		Raw:         cmd.Raw,
	})
	require.NoError(t, err)

	err = ds.UpdateHostLockWipeStatusFromAppleMDMResult(ctx, host.UUID, cmd.CommandUUID, "DeviceLock", true)
	require.NoError(t, err)

	// it is now locked
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, false, true, false, false, false, false)

	// request an unlock. This is a NOOP for Apple MDM.
	err = ds.UnlockHostManually(ctx, host.ID, host.FleetPlatform(), time.Now().UTC())
	require.NoError(t, err)

	// it is still locked
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, false, true, false, false, false, false)

	// execute CleanMacOSMDMLock to simulate successful unlock
	err = ds.CleanMacOSMDMLock(ctx, host.UUID)
	require.NoError(t, err)

	// it is back to unlocked state
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, true, false, false, false, false, false)
	require.Empty(t, status.UnlockPIN)

	// record a request to wipe the host
	cmd = &mdm.Command{
		CommandUUID: uuid.NewString(),
		Raw:         []byte("<?xml"),
	}
	cmd.Command.RequestType = "EraseDevice"
	err = appleStore.EnqueueDeviceWipeCommand(ctx, host, cmd)
	require.NoError(t, err)

	// it is now pending wipe
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, true, false, false, false, false, true)

	// record a command result failure to simulate failed wipe (back to unlocked)
	err = appleStore.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: host.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: cmd.CommandUUID,
		Status:      "Error",
		Raw:         cmd.Raw,
	})
	require.NoError(t, err)

	err = ds.UpdateHostLockWipeStatusFromAppleMDMResult(ctx, host.UUID, cmd.CommandUUID, cmd.Command.RequestType, false)
	require.NoError(t, err)

	// it is back to unlocked
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, true, false, false, false, false, false)

	// record a new request to wipe the host
	cmd = &mdm.Command{
		CommandUUID: uuid.NewString(),
		Raw:         []byte("<?xml"),
	}
	cmd.Command.RequestType = "EraseDevice"
	err = appleStore.EnqueueDeviceWipeCommand(ctx, host, cmd)
	require.NoError(t, err)

	// it is back to pending wipe
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, true, false, false, false, false, true)

	// record a command result success to simulate wipe
	err = appleStore.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: host.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: cmd.CommandUUID,
		Status:      "Acknowledged",
		Raw:         cmd.Raw,
	})
	require.NoError(t, err)

	err = ds.UpdateHostLockWipeStatusFromAppleMDMResult(ctx, host.UUID, cmd.CommandUUID, cmd.Command.RequestType, true)
	require.NoError(t, err)

	// it is wiped
	status, err = ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	checkLockWipeState(t, status, false, false, true, false, false, false)
}

func testScreenDEPAssignProfileSerialsForCooldown(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	skip, assign, err := ds.ScreenDEPAssignProfileSerialsForCooldown(ctx, []string{})
	require.NoError(t, err)
	require.Empty(t, skip)
	require.Empty(t, assign)
}

func testMDMAppleDDMDeclarationsToken(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	toks, err := ds.MDMAppleDDMDeclarationsToken(ctx, "not-exists")
	require.NoError(t, err)
	require.Empty(t, toks.DeclarationsToken)

	decl, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-1",
		Name:       "decl-1",
	})
	require.NoError(t, err)
	updates, err := ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl.DeclarationUUID}, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)

	toks, err = ds.MDMAppleDDMDeclarationsToken(ctx, "not-exists")
	require.NoError(t, err)
	require.Empty(t, toks.DeclarationsToken)
	require.NotZero(t, toks.Timestamp)

	host1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host1, true)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl.DeclarationUUID}, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)

	toks, err = ds.MDMAppleDDMDeclarationsToken(ctx, host1.UUID)
	require.NoError(t, err)
	require.NotEmpty(t, toks.DeclarationsToken)
	require.NotZero(t, toks.Timestamp)
	oldTok := toks.DeclarationsToken

	decl2, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-2",
		Name:       "decl-2",
	})
	require.NoError(t, err)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl2.DeclarationUUID}, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)

	toks, err = ds.MDMAppleDDMDeclarationsToken(ctx, host1.UUID)
	require.NoError(t, err)
	require.NotEmpty(t, toks.DeclarationsToken)
	require.NotZero(t, toks.Timestamp)
	require.NotEqual(t, oldTok, toks.DeclarationsToken)
	oldTok = toks.DeclarationsToken

	err = ds.DeleteMDMAppleConfigProfile(ctx, decl.DeclarationUUID)
	require.NoError(t, err)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl2.DeclarationUUID}, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration) // This is false because we delete references in `host_mdm_apple_declarations` for declarations that aren't sent to the host
	assert.False(t, updates.WindowsConfigProfile)

	toks, err = ds.MDMAppleDDMDeclarationsToken(ctx, host1.UUID)
	require.NoError(t, err)
	require.NotEmpty(t, toks.DeclarationsToken)
	require.NotZero(t, toks.Timestamp)
	require.NotEqual(t, oldTok, toks.DeclarationsToken)
}

func testMDMAppleSetPendingDeclarationsAs(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
			Identifier: fmt.Sprintf("decl-%d", i),
			Name:       fmt.Sprintf("decl-%d", i),
		})
		require.NoError(t, err)
	}

	checkStatus := func(declarations []fleet.HostMDMAppleProfile, wantStatus fleet.MDMDeliveryStatus, wantDetail string) {
		for _, d := range declarations {
			require.Equal(t, &wantStatus, d.Status)
			require.Equal(t, wantDetail, d.Detail)
		}
	}

	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, h, true)

	uuids, err := ds.MDMAppleBatchSetHostDeclarationState(ctx)
	require.NoError(t, err)
	require.Equal(t, h.UUID, uuids[0])

	profs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
	require.NoError(t, err)
	require.Len(t, profs, 10)
	checkStatus(profs, fleet.MDMDeliveryPending, "")

	err = ds.MDMAppleSetPendingDeclarationsAs(ctx, h.UUID, &fleet.MDMDeliveryFailed, "mock error")
	require.NoError(t, err)
	profs, err = ds.GetHostMDMAppleProfiles(ctx, h.UUID)
	require.NoError(t, err)
	checkStatus(profs, fleet.MDMDeliveryFailed, "mock error")
}

func testSetOrUpdateMDMAppleDDMDeclaration(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	l1, err := ds.NewLabel(ctx, &fleet.Label{Name: "l1", Query: "select 1"})
	require.NoError(t, err)
	l2, err := ds.NewLabel(ctx, &fleet.Label{Name: "l2", Query: "select 2"})
	require.NoError(t, err)
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm1"})
	require.NoError(t, err)

	d1, err := ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "i1",
		Name:       "d1",
		RawJSON:    json.RawMessage(`{"Identifier": "i1"}`),
	})
	require.NoError(t, err)

	// try to create same name, different identifier fails
	_, err = ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "i1b",
		Name:       "d1",
		RawJSON:    json.RawMessage(`{"Identifier": "i1b"}`),
	})
	require.Error(t, err)
	var existsErr *existsError
	require.ErrorAs(t, err, &existsErr)

	// try to create different name, same identifier fails
	_, err = ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "i1",
		Name:       "d1b",
		RawJSON:    json.RawMessage(`{"Identifier": "i1"}`),
	})
	require.Error(t, err)
	require.ErrorAs(t, err, &existsErr)

	// create same declaration for a different team works
	d1tm1, err := ds.SetOrUpdateMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "i1",
		Name:       "d1",
		TeamID:     &tm1.ID,
		RawJSON:    json.RawMessage(`{"Identifier": "i1"}`),
	})
	require.NoError(t, err)
	require.NotEqual(t, d1.DeclarationUUID, d1tm1.DeclarationUUID)

	d1Ori, err := ds.GetMDMAppleDeclaration(ctx, d1.DeclarationUUID)
	require.NoError(t, err)
	require.Empty(t, d1Ori.LabelsIncludeAll)

	// update d1 with different identifier and labels
	d1, err = ds.SetOrUpdateMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier:       "i1b",
		Name:             "d1",
		RawJSON:          json.RawMessage(`{"Identifier": "i1b"}`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{LabelName: l1.Name, LabelID: l1.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, d1.DeclarationUUID, d1Ori.DeclarationUUID)
	require.NotEqual(t, d1.DeclarationUUID, d1tm1.DeclarationUUID)

	d1B, err := ds.GetMDMAppleDeclaration(ctx, d1.DeclarationUUID)
	require.NoError(t, err)
	require.Len(t, d1B.LabelsIncludeAll, 1)
	require.Equal(t, l1.ID, d1B.LabelsIncludeAll[0].LabelID)

	// update d1 with different label
	d1, err = ds.SetOrUpdateMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier:       "i1b",
		Name:             "d1",
		RawJSON:          json.RawMessage(`{"Identifier": "i1b"}`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{LabelName: l2.Name, LabelID: l2.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, d1.DeclarationUUID, d1Ori.DeclarationUUID)

	d1C, err := ds.GetMDMAppleDeclaration(ctx, d1.DeclarationUUID)
	require.NoError(t, err)
	require.Len(t, d1C.LabelsIncludeAll, 1)
	require.Equal(t, l2.ID, d1C.LabelsIncludeAll[0].LabelID)

	// update d1tm1 with different identifier and label
	d1tm1B, err := ds.SetOrUpdateMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier:       "i1b",
		Name:             "d1",
		TeamID:           &tm1.ID,
		RawJSON:          json.RawMessage(`{"Identifier": "i1b"}`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{{LabelName: l1.Name, LabelID: l1.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, d1tm1B.DeclarationUUID, d1tm1.DeclarationUUID)

	d1tm1B, err = ds.GetMDMAppleDeclaration(ctx, d1tm1B.DeclarationUUID)
	require.NoError(t, err)
	require.Len(t, d1tm1B.LabelsIncludeAll, 1)
	require.Equal(t, l1.ID, d1tm1B.LabelsIncludeAll[0].LabelID)

	// delete no-team d1
	err = ds.DeleteMDMAppleDeclarationByName(ctx, nil, "d1")
	require.NoError(t, err)

	// it does not exist anymore, but the tm1 one still does
	_, err = ds.GetMDMAppleDeclaration(ctx, d1.DeclarationUUID)
	require.Error(t, err)

	d1tm1B, err = ds.GetMDMAppleDeclaration(ctx, d1tm1B.DeclarationUUID)
	require.NoError(t, err)
	require.Equal(t, d1tm1B.DeclarationUUID, d1tm1.DeclarationUUID)
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

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

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

			n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, []godep.Device{{SerialNumber: depSerial}}, abmToken.ID, nil, nil, nil)
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

func testMDMAppleDEPAssignmentUpdates(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	n := t.Name()
	h, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       fmt.Sprintf("test-host%s-name", n),
		OsqueryHostID:  ptr.String(fmt.Sprintf("osquery-%s", n)),
		NodeKey:        ptr.String(fmt.Sprintf("nodekey-%s", n)),
		UUID:           fmt.Sprintf("test-uuid-%s", n),
		Platform:       "darwin",
		HardwareSerial: n,
	})
	require.NoError(t, err)

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	_, err = ds.GetHostDEPAssignment(ctx, h.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID)
	require.NoError(t, err)

	assignment, err := ds.GetHostDEPAssignment(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, h.ID, assignment.HostID)
	require.Nil(t, assignment.DeletedAt)

	err = ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{h.HardwareSerial})
	require.NoError(t, err)

	assignment, err = ds.GetHostDEPAssignment(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, h.ID, assignment.HostID)
	require.NotNil(t, assignment.DeletedAt)

	err = ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h}, abmToken.ID)
	require.NoError(t, err)
	assignment, err = ds.GetHostDEPAssignment(ctx, h.ID)
	require.NoError(t, err)
	require.Equal(t, h.ID, assignment.HostID)
	require.Nil(t, assignment.DeletedAt)
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

func testMDMConfigAsset(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	assets := []fleet.MDMConfigAsset{
		{
			Name:  fleet.MDMAssetCACert,
			Value: []byte("a"),
		},
		{
			Name:  fleet.MDMAssetCAKey,
			Value: []byte("b"),
		},
	}
	wantAssets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
	for _, a := range assets {
		wantAssets[a.Name] = a
	}
	err := ds.InsertMDMConfigAssets(ctx, assets, nil)
	require.NoError(t, err)

	a, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, nil)
	require.NoError(t, err)
	require.Equal(t, wantAssets, a)

	h, err := ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	require.NoError(t, err)
	require.Len(t, h, 2)
	require.NotEmpty(t, h[fleet.MDMAssetCACert])
	require.NotEmpty(t, h[fleet.MDMAssetCAKey])

	// try to fetch an asset that doesn't exist
	var nfe fleet.NotFoundError
	a, err = ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetABMCert}, ds.writer(ctx))
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, a)

	h, err = ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetABMCert})
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, h)

	// try to fetch a mix of assets that exist and doesn't exist
	a, err = ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetABMCert}, nil)
	require.ErrorIs(t, err, ErrPartialResult)
	require.Len(t, a, 1)

	h, err = ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetABMCert})
	require.ErrorIs(t, err, ErrPartialResult)
	require.Len(t, h, 1)
	require.NotEmpty(t, h[fleet.MDMAssetCACert])

	// Replace the assets

	newAssets := []fleet.MDMConfigAsset{
		{
			Name:  fleet.MDMAssetCACert,
			Value: []byte("c"),
		},
		{
			Name:  fleet.MDMAssetCAKey,
			Value: []byte("d"),
		},
	}

	wantNewAssets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{}
	for _, a := range newAssets {
		wantNewAssets[a.Name] = a
	}

	err = ds.ReplaceMDMConfigAssets(ctx, newAssets, nil)
	require.NoError(t, err)

	a, err = ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, ds.reader(ctx))
	require.NoError(t, err)
	require.Equal(t, wantNewAssets, a)

	h, err = ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	require.NoError(t, err)
	require.Len(t, h, 2)
	require.NotEmpty(t, h[fleet.MDMAssetCACert])
	require.NotEmpty(t, h[fleet.MDMAssetCAKey])

	// Soft delete the assets

	err = ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	require.NoError(t, err)

	a, err = ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, nil)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, a)

	h, err = ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, h)

	// Verify that they're still in the DB. Values should be encrypted.

	type assetRow struct {
		Name         string    `db:"name"`
		Value        []byte    `db:"value"`
		DeletionUUID string    `db:"deletion_uuid"`
		DeletedAt    time.Time `db:"deleted_at"`
	}

	var ar []assetRow

	err = sqlx.SelectContext(ctx, ds.reader(ctx), &ar, "SELECT name, value, deletion_uuid, deleted_at FROM mdm_config_assets")
	require.NoError(t, err)

	require.Len(t, ar, 4)

	expected := make(map[string]fleet.MDMConfigAsset)

	for _, a := range append(assets, newAssets...) {
		expected[string(a.Value)] = a
	}

	for _, got := range ar {
		d, err := decrypt(got.Value, ds.serverPrivateKey)
		require.NoError(t, err)
		require.Equal(t, expected[string(d)].Name, fleet.MDMAssetName(got.Name))
		require.NotEmpty(t, got.Value)
		require.Equal(t, expected[string(d)].Value, d)
		require.NotEmpty(t, got.DeletionUUID)
		require.NotEmpty(t, got.DeletedAt)
	}

	// Hard delete
	err = ds.HardDeleteMDMConfigAsset(ctx, fleet.MDMAssetCACert)
	require.NoError(t, err)
	a, err = ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, nil)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, a)

	var result bool
	err = sqlx.GetContext(ctx, ds.reader(ctx), &result, "SELECT 1 FROM mdm_config_assets WHERE name = ?", fleet.MDMAssetCACert)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	// other (non-hard deleted asset still present)
	err = sqlx.GetContext(ctx, ds.reader(ctx), &result, "SELECT 1 FROM mdm_config_assets WHERE name = ?", fleet.MDMAssetCAKey)
	assert.NoError(t, err)
	assert.True(t, result)
}

func testListIOSAndIPadOSToRefetch(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	refetchInterval := 1 * time.Hour
	hostCount := 0
	newHost := func(platform string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:       fmt.Sprintf("foobar%d", hostCount),
			OsqueryHostID:  ptr.String(fmt.Sprintf("foobar-%d", hostCount)),
			NodeKey:        ptr.String(fmt.Sprintf("foobar-%d", hostCount)),
			UUID:           fmt.Sprintf("foobar-%d", hostCount),
			Platform:       platform,
			HardwareSerial: fmt.Sprintf("foobar-%d", hostCount),
		})
		require.NoError(t, err)
		hostCount++
		return h
	}

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	// Test with no hosts.
	devices, err := ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Empty(t, devices)

	// Create a placeholder macOS host.
	_ = newHost("darwin")

	// Mock results incoming from depsync.Syncer
	depDevices := []godep.Device{
		{SerialNumber: "iOS0_SERIAL", DeviceFamily: "iPhone", OpType: "added"},
		{SerialNumber: "iPadOS0_SERIAL", DeviceFamily: "iPad", OpType: "added"},
	}
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	// Hosts are not enrolled yet (e.g. DEP enrolled)
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Empty(t, devices)

	// Now simulate the initial MDM checkin of the devices.
	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           "iOS0_UUID",
		HardwareSerial: "iOS0_SERIAL",
		HardwareModel:  "iPhone14,6",
		Platform:       "ios",
		OsqueryHostID:  ptr.String("iOS0_OSQUERY_HOST_ID"),
	})
	require.NoError(t, err)
	iOS0, err := ds.HostByIdentifier(ctx, "iOS0_SERIAL")
	require.NoError(t, err)
	nanoEnroll(t, ds, iOS0, false)
	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           "iPadOS0_UUID",
		HardwareSerial: "iPadOS0_SERIAL",
		HardwareModel:  "iPad13,18",
		Platform:       "ipados",
		OsqueryHostID:  ptr.String("iPadOS0_OSQUERY_HOST_ID"),
	})
	require.NoError(t, err)
	iPadOS0, err := ds.HostByIdentifier(ctx, "iPadOS0_SERIAL")
	require.NoError(t, err)
	nanoEnroll(t, ds, iPadOS0, false)

	// Test with hosts but empty state in nanomdm command tables.
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Len(t, devices, 2)
	uuids := []string{devices[0].UUID, devices[1].UUID}
	sort.Slice(uuids, func(i, j int) bool {
		return uuids[i] < uuids[j]
	})
	assert.Equal(t, uuids, []string{"iOS0_UUID", "iPadOS0_UUID"})
	assert.Empty(t, devices[0].CommandsAlreadySent)
	assert.Empty(t, devices[1].CommandsAlreadySent)

	// Set iOS detail_updated_at as 30 minutes in the past.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 30 MINUTE) WHERE id = ?`, iOS0.ID)
		return err
	})

	// iOS device should not be returned because it was refetched recently
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, devices[0].UUID, "iPadOS0_UUID")

	// Set iPadOS detail_updated_at as 30 minutes in the past.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 30 MINUTE) WHERE id = ?`, iPadOS0.ID)
		return err
	})

	// Both devices are up-to-date thus none should be returned.
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Empty(t, devices)

	// Set iOS detail_updated_at as 2 hours in the past.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE id = ?`, iOS0.ID)
		return err
	})

	// iOS device be returned because it is out of date.
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, devices[0].UUID, "iOS0_UUID")
	assert.Empty(t, devices[0].CommandsAlreadySent)

	// Update commands already sent to the devices and check that they are returned.
	require.NoError(t, ds.AddHostMDMCommands(ctx, []fleet.HostMDMCommand{{
		HostID:      iOS0.ID,
		CommandType: "my-command",
	}}))
	devices, err = ds.ListIOSAndIPadOSToRefetch(ctx, refetchInterval)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	require.Equal(t, devices[0].UUID, "iOS0_UUID")
	require.Len(t, devices[0].CommandsAlreadySent, 1)
	assert.Equal(t, "my-command", devices[0].CommandsAlreadySent[0])
}

func testMDMAppleUpsertHostIOSIPadOS(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	createBuiltinLabels(t, ds)

	for i, platform := range []string{"ios", "ipados"} {
		// Upsert first to test insertMDMAppleHostDB.
		err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			HardwareSerial: fmt.Sprintf("test-serial-%d", i),
			HardwareModel:  "test-hw-model",
			Platform:       platform,
		})
		require.NoError(t, err)
		h, err := ds.HostByIdentifier(ctx, fmt.Sprintf("test-uuid-%d", i))
		require.NoError(t, err)
		require.Equal(t, false, h.RefetchRequested)
		require.Less(t, time.Since(h.LastEnrolledAt), 1*time.Hour) // check it's not in the date in the 2000 we use as "Never".
		require.Equal(t, "test-hw-model", h.HardwareModel)

		labels, err := ds.ListLabelsForHost(ctx, h.ID)
		require.NoError(t, err)
		require.Len(t, labels, 2)
		sort.Slice(labels, func(i, j int) bool {
			return labels[i].ID < labels[j].ID
		})
		require.Equal(t, "All Hosts", labels[0].Name)
		if i == 0 {
			require.Equal(t, "iOS", labels[1].Name)
		} else {
			require.Equal(t, "iPadOS", labels[1].Name)
		}

		// Insert again to test updateMDMAppleHostDB.
		err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			HardwareSerial: fmt.Sprintf("test-serial-%d", i),
			HardwareModel:  "test-hw-model-2",
			Platform:       platform,
		})
		require.NoError(t, err)
		h, err = ds.HostByIdentifier(ctx, fmt.Sprintf("test-uuid-%d", i))
		require.NoError(t, err)
		require.Equal(t, false, h.RefetchRequested)
		require.Less(t, time.Since(h.LastEnrolledAt), 1*time.Hour) // check it's not in the date in the 2000 we use as "Never".
		require.Equal(t, "test-hw-model-2", h.HardwareModel)

		labels, err = ds.ListLabelsForHost(ctx, h.ID)
		require.NoError(t, err)
		require.Len(t, labels, 2)
		sort.Slice(labels, func(i, j int) bool {
			return labels[i].ID < labels[j].ID
		})
		require.Equal(t, "All Hosts", labels[0].Name)
		if i == 0 {
			require.Equal(t, "iOS", labels[1].Name)
		} else {
			require.Equal(t, "iPadOS", labels[1].Name)
		}
	}

	err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           "test-uuid-2",
		HardwareSerial: "test-serial-2",
		HardwareModel:  "test-hw-model",
		Platform:       "darwin",
	})
	require.NoError(t, err)
	h, err := ds.HostByIdentifier(ctx, "test-uuid-2")
	require.NoError(t, err)
	require.Equal(t, true, h.RefetchRequested)
	require.Less(t, 1*time.Hour, time.Since(h.LastEnrolledAt)) // check it's in the date in the 2000 we use as "Never".
	labels, err := ds.ListLabelsForHost(ctx, h.ID)
	require.NoError(t, err)
	require.Len(t, labels, 2)
	require.Equal(t, "All Hosts", labels[0].Name)
	require.Equal(t, "macOS", labels[1].Name)
}

func testIngestMDMAppleDevicesFromDEPSyncIOSIPadOS(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Mock results incoming from depsync.Syncer
	depDevices := []godep.Device{
		{SerialNumber: "iOS0_SERIAL", DeviceFamily: "iPhone", OpType: "added"},
		{SerialNumber: "iPadOS0_SERIAL", DeviceFamily: "iPad", OpType: "added"},
	}

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, depDevices, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{
		User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	}, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	require.Equal(t, "ios", hosts[0].Platform)
	require.Equal(t, false, hosts[0].RefetchRequested)
	require.Equal(t, "ipados", hosts[1].Platform)
	require.Equal(t, false, hosts[1].RefetchRequested)
}

func testMDMAppleProfilesOnIOSIPadOS(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Add the Fleetd configuration and  profile that are only for macOS.
	params := mobileconfig.FleetdProfileOptions{
		EnrollSecret: t.Name(),
		ServerURL:    "https://example.com",
		PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
		PayloadName:  fleetmdm.FleetdConfigProfileName,
	}
	var contents bytes.Buffer
	err := mobileconfig.FleetdProfileTemplate.Execute(&contents, params)
	require.NoError(t, err)
	fleetdConfigProfile, err := fleet.NewMDMAppleConfigProfile(contents.Bytes(), nil)
	require.NoError(t, err)
	_, err = ds.NewMDMAppleConfigProfile(ctx, *fleetdConfigProfile)
	require.NoError(t, err)

	// For the FileVault profile we re-use the FleetdProfileTemplate
	// (because fileVaultProfileTemplate is not exported)
	var contents2 bytes.Buffer
	params.PayloadName = fleetmdm.FleetFileVaultProfileName
	params.PayloadType = mobileconfig.FleetFileVaultPayloadIdentifier
	err = mobileconfig.FleetdProfileTemplate.Execute(&contents2, params)
	require.NoError(t, err)
	fileVaultProfile, err := fleet.NewMDMAppleConfigProfile(contents2.Bytes(), nil)
	require.NoError(t, err)
	_, err = ds.NewMDMAppleConfigProfile(ctx, *fileVaultProfile)
	require.NoError(t, err)

	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           "iOS0_UUID",
		HardwareSerial: "iOS0_SERIAL",
		HardwareModel:  "iPhone14,6",
		Platform:       "ios",
		OsqueryHostID:  ptr.String("iOS0_OSQUERY_HOST_ID"),
	})
	require.NoError(t, err)
	iOS0, err := ds.HostByIdentifier(ctx, "iOS0_UUID")
	require.NoError(t, err)
	nanoEnroll(t, ds, iOS0, false)
	err = ds.MDMAppleUpsertHost(ctx, &fleet.Host{
		UUID:           "iPadOS0_UUID",
		HardwareSerial: "iPadOS0_SERIAL",
		HardwareModel:  "iPad13,18",
		Platform:       "ipados",
		OsqueryHostID:  ptr.String("iPadOS0_OSQUERY_HOST_ID"),
	})
	require.NoError(t, err)
	iPadOS0, err := ds.HostByIdentifier(ctx, "iPadOS0_UUID")
	require.NoError(t, err)
	nanoEnroll(t, ds, iPadOS0, false)

	someProfile, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 0))
	require.NoError(t, err)

	updates, err := ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{0}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)

	profiles, err := ds.GetHostMDMAppleProfiles(ctx, "iOS0_UUID")
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, someProfile.Name, profiles[0].Name)
	profiles, err = ds.GetHostMDMAppleProfiles(ctx, "iPadOS0_UUID")
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, someProfile.Name, profiles[0].Name)
}

func testGetHostUUIDsWithPendingMDMAppleCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	uuids, err := ds.GetHostUUIDsWithPendingMDMAppleCommands(ctx)
	require.NoError(t, err)
	require.Empty(t, uuids)

	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now())
		hosts = append(hosts, h)
		nanoEnroll(t, ds, h, false)
	}

	uuids, err = ds.GetHostUUIDsWithPendingMDMAppleCommands(ctx)
	require.NoError(t, err)
	require.Empty(t, uuids)

	commander, storage := createMDMAppleCommanderAndStorage(t, ds)
	// insert a command for three hosts
	uuid1 := uuid.New().String()
	rawCmd1 := createRawAppleCmd("ListApps", uuid1)
	err = commander.EnqueueCommand(ctx, []string{hosts[0].UUID, hosts[1].UUID, hosts[2].UUID}, rawCmd1)
	require.NoError(t, err)

	uuids, err = ds.GetHostUUIDsWithPendingMDMAppleCommands(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{hosts[0].UUID, hosts[1].UUID, hosts[2].UUID}, uuids)

	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: hosts[0].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid1,
		Status:      "Acknowledged",
		Raw:         []byte(rawCmd1),
	})
	require.NoError(t, err)

	// only hosts[1] and hosts[2] are returned now
	uuids, err = ds.GetHostUUIDsWithPendingMDMAppleCommands(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{hosts[1].UUID, hosts[2].UUID}, uuids)
}

func testHostDetailsMDMProfilesIOSIPadOS(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	p0, err := ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{
		Name:         "Name0",
		Identifier:   "Identifier0",
		Mobileconfig: []byte("profile0-bytes"),
	})
	require.NoError(t, err)

	profiles, err := ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	iOS, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host0-osquery-id"),
		NodeKey:         ptr.String("host0-node-key"),
		UUID:            "host0-test-mdm-profiles",
		Hostname:        "hostname0",
		Platform:        "ios",
	})
	require.NoError(t, err)
	iPadOS, err := ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   ptr.String("host0-osquery-id-2"),
		NodeKey:         ptr.String("host0-node-key-2"),
		UUID:            "host0-test-mdm-profiles-2",
		Hostname:        "hostname0-2",
		Platform:        "ipados",
	})
	require.NoError(t, err)

	gotHost, err := ds.Host(ctx, iOS.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles)
	gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, iOS.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)
	gotHost, err = ds.Host(ctx, iPadOS.ID)
	require.NoError(t, err)
	require.Nil(t, gotHost.MDM.Profiles)
	gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, iPadOS.UUID)
	require.NoError(t, err)
	require.Nil(t, gotProfs)

	expectedProfilesIOS := map[string]fleet.HostMDMAppleProfile{
		p0.ProfileUUID: {
			HostUUID:      iOS.UUID,
			Name:          p0.Name,
			ProfileUUID:   p0.ProfileUUID,
			CommandUUID:   "cmd0-uuid",
			Status:        &fleet.MDMDeliveryPending,
			OperationType: fleet.MDMOperationTypeInstall,
			Detail:        "",
		},
	}
	expectedProfilesIPadOS := map[string]fleet.HostMDMAppleProfile{
		p0.ProfileUUID: {
			HostUUID:      iPadOS.UUID,
			Name:          p0.Name,
			ProfileUUID:   p0.ProfileUUID,
			CommandUUID:   "cmd0-uuid",
			Status:        &fleet.MDMDeliveryPending,
			OperationType: fleet.MDMOperationTypeInstall,
			Detail:        "",
		},
	}

	var args []interface{}
	for _, p := range expectedProfilesIOS {
		args = append(args, p.HostUUID, p.ProfileUUID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}
	for _, p := range expectedProfilesIPadOS {
		args = append(args, p.HostUUID, p.ProfileUUID, p.CommandUUID, *p.Status, p.OperationType, p.Detail, p.Name)
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
	INSERT INTO host_mdm_apple_profiles (
		host_uuid, profile_uuid, command_uuid, status, operation_type, detail, profile_name)
	VALUES (?,?,?,?,?,?,?),(?,?,?,?,?,?,?)
		`, args...,
		)
		if err != nil {
			return err
		}
		return nil
	})

	for _, tc := range []struct {
		host             *fleet.Host
		expectedProfiles map[string]fleet.HostMDMAppleProfile
	}{
		{
			host:             iOS,
			expectedProfiles: expectedProfilesIOS,
		},
		{
			host:             iPadOS,
			expectedProfiles: expectedProfilesIPadOS,
		},
	} {
		gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, tc.host.UUID)
		require.NoError(t, err)
		require.Len(t, gotProfs, 1)
		for _, gp := range gotProfs {
			ep, ok := expectedProfilesIOS[gp.ProfileUUID]
			require.True(t, ok)
			require.Equal(t, ep.Name, gp.Name)
			require.Equal(t, *ep.Status, *gp.Status)
			require.Equal(t, ep.OperationType, gp.OperationType)
			require.Equal(t, ep.Detail, gp.Detail)
		}

		// mark pending profile to 'verifying', which should instead set it as 'verified'.
		installPendingProfile := expectedProfilesIOS[p0.ProfileUUID]
		err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			HostUUID:      installPendingProfile.HostUUID,
			CommandUUID:   installPendingProfile.CommandUUID,
			ProfileUUID:   installPendingProfile.ProfileUUID,
			Name:          installPendingProfile.Name,
			Status:        &fleet.MDMDeliveryVerifying,
			OperationType: fleet.MDMOperationTypeInstall,
			Detail:        "",
		})
		require.NoError(t, err)

		// Check that the profile is the 'verified' state.
		gotProfs, err = ds.GetHostMDMAppleProfiles(ctx, iOS.UUID)
		require.NoError(t, err)
		require.Len(t, gotProfs, 1)
		require.NotNil(t, gotProfs[0].Status)
		require.Equal(t, fleet.MDMDeliveryVerified, *gotProfs[0].Status)
	}
}

func testMDMAppleBootstrapPackageWithS3(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	var nfe fleet.NotFoundError
	var aerr fleet.AlreadyExistsError

	hashContent := func(content string) []byte {
		h := sha256.New()
		_, err := h.Write([]byte(content))
		require.NoError(t, err)
		return h.Sum(nil)
	}

	bpMatchesWithoutContent := func(want, got *fleet.MDMAppleBootstrapPackage) {
		// make local copies so we don't alter the caller's structs
		w, g := *want, *got
		w.Bytes, g.Bytes = nil, nil
		w.CreatedAt, g.CreatedAt = time.Time{}, time.Time{}
		w.UpdatedAt, g.UpdatedAt = time.Time{}, time.Time{}
		require.Equal(t, w, g)
	}

	pkgStore := s3.SetupTestBootstrapPackageStore(t, "mdm-apple-bootstrap-package-test", "")

	err := ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{}, pkgStore)
	require.Error(t, err)

	// associate bp1 with no team
	bp1 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(0),
		Name:   "bp1",
		Sha256: hashContent("bp1"),
		Bytes:  []byte("bp1"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1, pkgStore)
	require.NoError(t, err)

	// try to store bp1 again, fails as it already exists
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp1, pkgStore)
	require.ErrorAs(t, err, &aerr)

	// associate bp2 with team id 2
	bp2 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(2),
		Name:   "bp2",
		Sha256: hashContent("bp2"),
		Bytes:  []byte("bp2"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp2, pkgStore)
	require.NoError(t, err)

	// associate the same content as bp1 with team id 1, via a copy
	err = ds.CopyDefaultMDMAppleBootstrapPackage(ctx, &fleet.AppConfig{}, 1)
	require.NoError(t, err)

	// get bp for no team
	meta, err := ds.GetMDMAppleBootstrapPackageMeta(ctx, 0)
	require.NoError(t, err)
	bpMatchesWithoutContent(bp1, meta)

	// get for team 1, token differs due to the copy, rest is the same
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 1)
	require.NoError(t, err)
	require.NotEqual(t, bp1.Token, meta.Token)
	bp1b := *bp1
	bp1b.Token = meta.Token
	bp1b.TeamID = 1
	bpMatchesWithoutContent(&bp1b, meta)

	// get for team 2
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 2)
	require.NoError(t, err)
	bpMatchesWithoutContent(bp2, meta)

	// get for team 3, does not exist
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 3)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// get content for no team
	bpContent, err := ds.GetMDMAppleBootstrapPackageBytes(ctx, bp1.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp1.Bytes, bpContent.Bytes)

	// get content for team 1 (copy of no team)
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp1b.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp1b.Bytes, bpContent.Bytes)
	require.Equal(t, bp1.Bytes, bpContent.Bytes)

	// get content for team 2
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp2.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp2.Bytes, bpContent.Bytes)

	// get content with invalid token
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, "no-such-token", pkgStore)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, bpContent)

	// delete bp for no team and team 2
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 0)
	require.NoError(t, err)
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 2)
	require.NoError(t, err)

	// run the cleanup job
	err = ds.CleanupUnusedBootstrapPackages(ctx, pkgStore, time.Now())
	require.NoError(t, err)

	// team 1 can still be retrieved (it shares the same contents)
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp1b.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp1b.Bytes, bpContent.Bytes)

	// team 0 and 2 don't exist anymore
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 0)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 2)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	ok, err := pkgStore.Exists(ctx, hex.EncodeToString(bp1.Sha256))
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = pkgStore.Exists(ctx, hex.EncodeToString(bp2.Sha256))
	require.NoError(t, err)
	require.False(t, ok)

	// delete team 1
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 1)
	require.NoError(t, err)

	// force a team 3 bp to be saved in the DB (simulates upgrading to the new
	// S3-based storage with already-saved bps in the DB)
	bp3 := &fleet.MDMAppleBootstrapPackage{
		TeamID: uint(3),
		Name:   "bp3",
		Sha256: hashContent("bp3"),
		Bytes:  []byte("bp3"),
		Token:  uuid.New().String(),
	}
	err = ds.InsertMDMAppleBootstrapPackage(ctx, bp3, nil) // passing a nil pkgStore to force save in the DB
	require.NoError(t, err)

	// metadata can be read
	meta, err = ds.GetMDMAppleBootstrapPackageMeta(ctx, 3)
	require.NoError(t, err)
	bpMatchesWithoutContent(bp3, meta)

	// content will be retrieved correctly from the DB even if we pass a pkgStore
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp3.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp3.Bytes, bpContent.Bytes)

	// run the cleanup job
	err = ds.CleanupUnusedBootstrapPackages(ctx, pkgStore, time.Now())
	require.NoError(t, err)

	ok, err = pkgStore.Exists(ctx, hex.EncodeToString(bp1.Sha256))
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = pkgStore.Exists(ctx, hex.EncodeToString(bp2.Sha256))
	require.NoError(t, err)
	require.False(t, ok)
	// bp3 does not exist in the S3 store
	ok, err = pkgStore.Exists(ctx, hex.EncodeToString(bp3.Sha256))
	require.NoError(t, err)
	require.False(t, ok)

	// so it can still be retrieved from the DB
	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp3.Token, pkgStore)
	require.NoError(t, err)
	require.Equal(t, bp3.Bytes, bpContent.Bytes)

	// it can be deleted without problem
	err = ds.DeleteMDMAppleBootstrapPackage(ctx, 3)
	require.NoError(t, err)

	bpContent, err = ds.GetMDMAppleBootstrapPackageBytes(ctx, bp3.Token, pkgStore)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, bpContent)
}

func testMDMAppleGetAndUpdateABMToken(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get a non-existing token
	tok, err := ds.GetABMTokenByOrgName(ctx, "no-such-token")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, tok)

	// create some teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	toks, err := ds.ListABMTokens(ctx)
	require.NoError(t, err)
	require.Empty(t, toks)

	tokCount, err := ds.GetABMTokenCount(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 0, tokCount)

	// create a token with an empty name and no team set, and another that will be unused
	encTok := uuid.NewString()

	t1, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, t1.ID)
	t2, err := ds.InsertABMToken(ctx, &fleet.ABMToken{EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, t2.ID)

	toks, err = ds.ListABMTokens(ctx)
	require.NoError(t, err)
	require.Len(t, toks, 2)

	tokCount, err = ds.GetABMTokenCount(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, tokCount)

	// get that token
	tok, err = ds.GetABMTokenByOrgName(ctx, "")
	require.NoError(t, err)
	require.NotZero(t, tok.ID)
	require.Equal(t, encTok, string(tok.EncryptedToken))
	require.Empty(t, tok.OrganizationName)
	require.Empty(t, tok.AppleID)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeamName)

	// update the token with a name and teams
	tok.OrganizationName = "org-name"
	tok.AppleID = "name@example.com"
	tok.MacOSDefaultTeamID = &tm1.ID
	tok.IOSDefaultTeamID = &tm2.ID
	err = ds.SaveABMToken(ctx, tok)
	require.NoError(t, err)

	// reload that token
	tokReload, err := ds.GetABMTokenByOrgName(ctx, "org-name")
	require.NoError(t, err)
	require.Equal(t, tok.ID, tokReload.ID)
	require.Equal(t, encTok, string(tokReload.EncryptedToken))
	require.Equal(t, "org-name", tokReload.OrganizationName)
	require.Equal(t, "name@example.com", tokReload.AppleID)
	require.Equal(t, tm1.Name, tokReload.MacOSTeamName)
	require.Equal(t, tm1.Name, tokReload.MacOSTeam.Name)
	require.Equal(t, tm1.ID, tokReload.MacOSTeam.ID)
	require.Equal(t, tm2.Name, tokReload.IOSTeamName)
	require.Equal(t, tm2.Name, tokReload.IOSTeam.Name)
	require.Equal(t, tm2.ID, tokReload.IOSTeam.ID)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.IPadOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.IPadOSTeam.Name)
	require.Equal(t, uint(0), tokReload.IPadOSTeam.ID)

	// empty name token now doesn't exist
	_, err = ds.GetABMTokenByOrgName(ctx, "")
	require.ErrorAs(t, err, &nfe)

	// update some teams
	tok.MacOSDefaultTeamID = nil
	tok.IPadOSDefaultTeamID = &tm3.ID
	err = ds.SaveABMToken(ctx, tok)
	require.NoError(t, err)

	// reload that token
	tokReload, err = ds.GetABMTokenByOrgName(ctx, "org-name")
	require.NoError(t, err)
	require.Equal(t, tok.ID, tokReload.ID)
	require.Equal(t, encTok, string(tokReload.EncryptedToken))
	require.Equal(t, "org-name", tokReload.OrganizationName)
	require.Equal(t, "name@example.com", tokReload.AppleID)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.MacOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.MacOSTeam.Name)
	require.Equal(t, uint(0), tokReload.MacOSTeam.ID)
	require.Equal(t, tm2.Name, tokReload.IOSTeamName)
	require.Equal(t, tm3.Name, tokReload.IPadOSTeamName)

	// change just the encrypted token
	encTok2 := uuid.NewString()
	tok.EncryptedToken = []byte(encTok2)
	err = ds.SaveABMToken(ctx, tok)
	require.NoError(t, err)

	tokReload, err = ds.GetABMTokenByOrgName(ctx, "org-name")
	require.NoError(t, err)
	require.Equal(t, tok.ID, tokReload.ID)
	require.Equal(t, encTok2, string(tokReload.EncryptedToken))
	require.Equal(t, "org-name", tokReload.OrganizationName)
	require.Equal(t, "name@example.com", tokReload.AppleID)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.MacOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, tokReload.MacOSTeam.Name)
	require.Equal(t, uint(0), tokReload.MacOSTeam.ID)
	require.Equal(t, tm2.Name, tokReload.IOSTeamName)
	require.Equal(t, tm2.Name, tokReload.IOSTeam.Name)
	require.Equal(t, tm2.ID, tokReload.IOSTeam.ID)
	require.Equal(t, tm3.Name, tokReload.IPadOSTeamName)
	require.Equal(t, tm3.Name, tokReload.IPadOSTeam.Name)
	require.Equal(t, tm3.ID, tokReload.IPadOSTeam.ID)

	// Remove unused token
	require.NoError(t, ds.DeleteABMToken(ctx, t1.ID))

	toks, err = ds.ListABMTokens(ctx)
	require.NoError(t, err)
	require.Len(t, toks, 1)
	expTok := toks[0]
	require.Equal(t, "org-name", expTok.OrganizationName)
	require.Equal(t, "name@example.com", expTok.AppleID)
	require.Equal(t, fleet.TeamNameNoTeam, expTok.MacOSTeamName)
	require.Equal(t, fleet.TeamNameNoTeam, expTok.MacOSTeam.Name)
	require.Equal(t, uint(0), expTok.MacOSTeam.ID)
	require.Equal(t, tm2.Name, expTok.IOSTeamName)
	require.Equal(t, tm3.Name, expTok.IPadOSTeamName)

	tokCount, err = ds.GetABMTokenCount(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, tokCount)
}

func testMDMAppleABMTokensTermsExpired(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// count works with no token
	count, err := ds.CountABMTokensWithTermsExpired(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	// create a few tokens
	encTok1 := uuid.NewString()
	t1, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "abm1", EncryptedToken: []byte(encTok1)})
	require.NoError(t, err)
	require.NotEmpty(t, t1.ID)
	encTok2 := uuid.NewString()
	t2, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "abm2", EncryptedToken: []byte(encTok2)})
	require.NoError(t, err)
	require.NotEmpty(t, t2.ID)
	// this one simulates a mirated token - empty name
	encTok3 := uuid.NewString()
	t3, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "", EncryptedToken: []byte(encTok3)})
	require.NoError(t, err)
	require.NotEmpty(t, t3.ID)

	// none have terms expired yet
	count, err = ds.CountABMTokensWithTermsExpired(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	// set t1 terms expired
	was, err := ds.SetABMTokenTermsExpiredForOrgName(ctx, t1.OrganizationName, true)
	require.NoError(t, err)
	require.False(t, was)

	// set t2 terms not expired, no-op
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, t2.OrganizationName, false)
	require.NoError(t, err)
	require.False(t, was)

	// set t3 terms expired
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, t3.OrganizationName, true)
	require.NoError(t, err)
	require.False(t, was)

	// count is now 2
	count, err = ds.CountABMTokensWithTermsExpired(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// set t1 terms not expired
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, t1.OrganizationName, false)
	require.NoError(t, err)
	require.True(t, was)

	// set t3 terms still expired
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, t3.OrganizationName, true)
	require.NoError(t, err)
	require.True(t, was)

	// count is now 1
	count, err = ds.CountABMTokensWithTermsExpired(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	// setting the expired flag of a non-existing token always returns as if it
	// did not update (which is fine, it will only be called after a DEP API call
	// that used this token, so if the token does not exist it would fail the
	// call).
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, "no-such-token", false)
	require.NoError(t, err)
	require.False(t, was)
	was, err = ds.SetABMTokenTermsExpiredForOrgName(ctx, "no-such-token", true)
	require.NoError(t, err)
	require.True(t, was)

	// count is unaffected
	count, err = ds.CountABMTokensWithTermsExpired(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func testMDMGetABMTokenOrgNamesAssociatedWithTeam(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create some teams
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	encTok := uuid.NewString()

	tok1, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "org1", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, tok1.ID)

	tok2, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "org2", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, tok1.ID)

	tok3, err := ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "org3", EncryptedToken: []byte(encTok), MacOSDefaultTeamID: &tm2.ID})
	require.NoError(t, err)
	require.NotEmpty(t, tok1.ID)

	// Create some hosts and add to teams (and one for no team)
	h1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1"),
		NodeKey:       ptr.String("1"),
		UUID:          "test-uuid-1",
		TeamID:        &tm1.ID,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	h2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host2-name",
		OsqueryHostID: ptr.String("2"),
		NodeKey:       ptr.String("2"),
		UUID:          "test-uuid-2",
		TeamID:        &tm1.ID,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	h3, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host3-name",
		OsqueryHostID: ptr.String("3"),
		NodeKey:       ptr.String("3"),
		UUID:          "test-uuid-3",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	h4, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host4-name",
		OsqueryHostID: ptr.String("4"),
		NodeKey:       ptr.String("4"),
		UUID:          "test-uuid-4",
		TeamID:        &tm1.ID,
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// Insert host DEP assignment
	require.NoError(t, ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h1, *h4}, tok1.ID))
	require.NoError(t, ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h2}, tok3.ID))
	require.NoError(t, ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*h3}, tok2.ID))

	// Should return the 2 unique org names [org1, org3]
	orgNames, err := ds.GetABMTokenOrgNamesAssociatedWithTeam(ctx, &tm1.ID)
	require.NoError(t, err)
	sort.Strings(orgNames)
	require.Len(t, orgNames, 2)
	require.Equal(t, orgNames[0], "org1")
	require.Equal(t, orgNames[1], "org3")

	// all tokens default to no team in one way or another
	orgNames, err = ds.GetABMTokenOrgNamesAssociatedWithTeam(ctx, nil)
	require.NoError(t, err)
	sort.Strings(orgNames)
	require.Len(t, orgNames, 3)
	require.Equal(t, orgNames[0], "org1")
	require.Equal(t, orgNames[1], "org2")
	require.Equal(t, orgNames[2], "org3")

	// No orgs for this team except org3 which uses it as a default team
	orgNames, err = ds.GetABMTokenOrgNamesAssociatedWithTeam(ctx, &tm2.ID)
	require.NoError(t, err)
	sort.Strings(orgNames)
	require.Len(t, orgNames, 1)
	require.Equal(t, orgNames[0], "org3")
}

func testHostMDMCommands(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	addHostMDMCommandsBatchSizeOrig := addHostMDMCommandsBatchSize
	addHostMDMCommandsBatchSize = 2
	t.Cleanup(func() {
		addHostMDMCommandsBatchSize = addHostMDMCommandsBatchSizeOrig
	})

	// create a host
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

	hostCommands := []fleet.HostMDMCommand{
		{
			HostID:      h.ID,
			CommandType: "command-1",
		},
		{
			HostID:      h.ID,
			CommandType: "command-2",
		},
		{
			HostID:      h.ID,
			CommandType: "command-3",
		},
	}

	badHostID := h.ID + 1
	allCommands := hostCommands
	allCommands = append(allCommands, fleet.HostMDMCommand{
		HostID:      badHostID,
		CommandType: "command-1",
	})
	err = ds.AddHostMDMCommands(ctx, allCommands)
	require.NoError(t, err)

	commands, err := ds.GetHostMDMCommands(ctx, h.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, hostCommands, commands)

	// Remove a command
	require.NoError(t, ds.RemoveHostMDMCommand(ctx, hostCommands[0]))

	commands, err = ds.GetHostMDMCommands(ctx, h.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, hostCommands[1:], commands)

	// Clean up commands, and make sure badHost commands have been removed, but others remain.
	commands, err = ds.GetHostMDMCommands(ctx, badHostID)
	require.NoError(t, err)
	assert.Len(t, commands, 1)

	require.NoError(t, ds.CleanupHostMDMCommands(ctx))
	commands, err = ds.GetHostMDMCommands(ctx, badHostID)
	require.NoError(t, err)
	assert.Empty(t, commands)

	commands, err = ds.GetHostMDMCommands(ctx, h.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, hostCommands[1:], commands)
}

func testIngestMDMAppleDeviceFromOTAEnrollment(t *testing.T, ds *Datastore) {
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

	// mock results incoming from OTA enrollments
	otaDevices := []fleet.MDMAppleMachineInfo{
		{Serial: "abc", Product: "MacBook Pro"},
		{Serial: "abc", Product: "MacBook Pro"},
		{Serial: hosts[0].HardwareSerial, Product: "MacBook Pro"},
		{Serial: "ijk", Product: "iPad13,16"},
		{Serial: "tuv", Product: "iPhone14,6"},
		{Serial: hosts[1].HardwareSerial, Product: "MacBook Pro"},
		{Serial: "xyz", Product: "MacBook Pro"},
		{Serial: "xyz", Product: "MacBook Pro"},
		{Serial: "xyz", Product: "MacBook Pro"},
	}
	wantSerials = append(wantSerials, "abc", "xyz", "ijk", "tuv")

	for _, d := range otaDevices {
		err := ds.IngestMDMAppleDeviceFromOTAEnrollment(ctx, nil, d)
		require.NoError(t, err)
	}

	hosts = listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, len(wantSerials))
	gotSerials := []string{}
	for _, h := range hosts {
		gotSerials = append(gotSerials, h.HardwareSerial)

		switch h.HardwareSerial {
		case "abc", "xyz":
			checkMDMHostRelatedTables(t, ds, h.ID, h.HardwareSerial, "MacBook Pro")
		case "ijk":
			checkMDMHostRelatedTables(t, ds, h.ID, h.HardwareSerial, "iPad13,16")
		case "tuv":
			checkMDMHostRelatedTables(t, ds, h.ID, h.HardwareSerial, "iPhone14,6")

		}
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
}

func TestGetMDMAppleOSUpdatesSettingsByHostSerial(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	keys := []string{"ios", "ipados", "macos"}
	devicesByKey := map[string]godep.Device{
		"ios":    {SerialNumber: "dep-serial-ios-updates", DeviceFamily: "iPhone"},
		"ipados": {SerialNumber: "dep-serial-ipados-updates", DeviceFamily: "iPad"},
		"macos":  {SerialNumber: "dep-serial-macos-updates", DeviceFamily: "Mac"},
	}

	getConfigSettings := func(teamID uint, key string) *fleet.AppleOSUpdateSettings {
		var settings fleet.AppleOSUpdateSettings
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := fmt.Sprintf(`SELECT json_value->'$.mdm.%s_updates' FROM app_config_json`, key)
			if teamID > 0 {
				stmt = fmt.Sprintf(`SELECT config->'$.mdm.%s_updates' FROM teams WHERE id = %d`, key, teamID)
			}
			var raw json.RawMessage
			if err := sqlx.GetContext(context.Background(), q, &raw, stmt); err != nil {
				return err
			}
			if err := json.Unmarshal(raw, &settings); err != nil {
				return err
			}
			return nil
		})
		return &settings
	}

	setConfigSettings := func(teamID uint, key string, minVersion string) {
		var mv *string
		if minVersion != "" {
			mv = &minVersion
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt := fmt.Sprintf(`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.%s_updates.minimum_version', ?)`, key)
			if teamID > 0 {
				stmt = fmt.Sprintf(`UPDATE teams SET config = JSON_SET(config, '$.mdm.%s_updates.minimum_version', ?) WHERE id = %d`, key, teamID)
			}
			if _, err := q.ExecContext(context.Background(), stmt, mv); err != nil {
				return err
			}
			return nil
		})
	}

	checkExpectedVersion := func(t *testing.T, gotSettings *fleet.AppleOSUpdateSettings, expectedVersion string) {
		if expectedVersion == "" {
			require.True(t, gotSettings.MinimumVersion.Set)
			require.False(t, gotSettings.MinimumVersion.Valid)
			require.Empty(t, gotSettings.MinimumVersion.Value)
		} else {
			require.True(t, gotSettings.MinimumVersion.Set)
			require.True(t, gotSettings.MinimumVersion.Valid)
			require.Equal(t, expectedVersion, gotSettings.MinimumVersion.Value)
		}
	}

	checkDevice := func(t *testing.T, teamID uint, key string, wantVersion string) {
		checkExpectedVersion(t, getConfigSettings(teamID, key), wantVersion)
		gotSettings, err := ds.GetMDMAppleOSUpdatesSettingsByHostSerial(context.Background(), devicesByKey[key].SerialNumber)
		require.NoError(t, err)
		checkExpectedVersion(t, gotSettings, wantVersion)
	}

	// empty global settings to start
	for _, key := range keys {
		checkExpectedVersion(t, getConfigSettings(0, key), "")
	}

	encTok := uuid.NewString()
	abmToken, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)

	// ingest some test devices
	n, err := ds.IngestMDMAppleDevicesFromDEPSync(context.Background(), []godep.Device{devicesByKey["ios"], devicesByKey["ipados"], devicesByKey["macos"]}, abmToken.ID, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)
	hostIDsByKey := map[string]uint{}
	for key, device := range devicesByKey {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var hid uint
			err = sqlx.GetContext(context.Background(), q, &hid, "SELECT id FROM hosts WHERE hardware_serial = ?", device.SerialNumber)
			require.NoError(t, err)
			hostIDsByKey[key] = hid
			return nil
		})
	}

	// not set in global config, so devics should return empty
	checkDevice(t, 0, "ios", "")
	checkDevice(t, 0, "ipados", "")
	checkDevice(t, 0, "macos", "")

	// set the minimum version for ios
	setConfigSettings(0, "ios", "17.1")
	checkDevice(t, 0, "ios", "17.1")
	checkDevice(t, 0, "ipados", "") // no change
	checkDevice(t, 0, "macos", "")  // no change

	// set the minimum version for ipados
	setConfigSettings(0, "ipados", "17.2")
	checkDevice(t, 0, "ios", "17.1") // no change
	checkDevice(t, 0, "ipados", "17.2")
	checkDevice(t, 0, "macos", "") // no change

	// set the minimum version for macos
	setConfigSettings(0, "macos", "14.5")
	checkDevice(t, 0, "ios", "17.1")    // no change
	checkDevice(t, 0, "ipados", "17.2") // no change
	checkDevice(t, 0, "macos", "14.5")

	// create a team
	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// empty team settings to start
	for _, key := range keys {
		checkExpectedVersion(t, getConfigSettings(team.ID, key), "")
	}

	// transfer ios and ipados to the team
	err = ds.AddHostsToTeam(context.Background(), &team.ID, []uint{hostIDsByKey["ios"], hostIDsByKey["ipados"]})
	require.NoError(t, err)

	checkDevice(t, team.ID, "ios", "")    // team settings are empty to start
	checkDevice(t, team.ID, "ipados", "") // team settings are empty to start
	checkDevice(t, 0, "macos", "14.5")    // no change, still global

	setConfigSettings(team.ID, "ios", "17.3")
	checkDevice(t, team.ID, "ios", "17.3") // team settings are set for ios
	checkDevice(t, team.ID, "ipados", "")  // team settings are empty for ipados
	checkDevice(t, 0, "macos", "14.5")     // no change, still global

	setConfigSettings(team.ID, "ipados", "17.4")
	checkDevice(t, team.ID, "ios", "17.3")    // no change in team settings for ios
	checkDevice(t, team.ID, "ipados", "17.4") // team settings are set for ipados
	checkDevice(t, 0, "macos", "14.5")        // no change, still global

	// transfer macos to the team
	err = ds.AddHostsToTeam(context.Background(), &team.ID, []uint{hostIDsByKey["macos"]})
	require.NoError(t, err)
	checkDevice(t, team.ID, "macos", "") // team settings are empty for macos

	setConfigSettings(team.ID, "macos", "14.6")
	checkDevice(t, team.ID, "macos", "14.6") // team settings are set for macos

	// create a non-DEP host
	_, err = ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:  ptr.String("non-dep-osquery-id"),
		NodeKey:        ptr.String("non-dep-node-key"),
		UUID:           "non-dep-uuid",
		Hostname:       "non-dep-hostname",
		Platform:       "macos",
		HardwareSerial: "non-dep-serial",
	})

	// non-DEP host should return not found
	_, err = ds.GetMDMAppleOSUpdatesSettingsByHostSerial(context.Background(), "non-dep-serial")
	require.ErrorIs(t, err, sql.ErrNoRows)

	// deleted DEP host should return not found
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "UPDATE host_dep_assignments SET deleted_at = NOW() WHERE host_id = ?", hostIDsByKey["macos"])
		return err
	})
	_, err = ds.GetMDMAppleOSUpdatesSettingsByHostSerial(context.Background(), devicesByKey["macos"].SerialNumber)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testMDMManagedCertificates(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	initialCP := storeDummyConfigProfileForTest(t, ds)
	host, err := ds.NewHost(ctx, &fleet.Host{
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

	// Host and profile are not linked
	profile, err := ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID)
	require.NoError(t, err)
	assert.Nil(t, profile)

	err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       initialCP.ProfileUUID,
			ProfileIdentifier: initialCP.Identifier,
			ProfileName:       initialCP.Name,
			HostUUID:          host.UUID,
			Status:            &fleet.MDMDeliveryPending,
			OperationType:     fleet.MDMOperationTypeInstall,
			CommandUUID:       "command-uuid",
			Checksum:          []byte("checksum"),
		},
	},
	)
	require.NoError(t, err)

	// Host and profile do not have certificate metadata
	profile, err = ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, host.UUID, profile.HostUUID)
	assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
	assert.Nil(t, profile.ChallengeRetrievedAt)

	challengeRetrievedAt := time.Now().Add(-time.Hour).UTC().Round(time.Microsecond)
	err = ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMBulkUpsertManagedCertificatePayload{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          initialCP.ProfileUUID,
			ChallengeRetrievedAt: &challengeRetrievedAt,
		},
	})
	require.NoError(t, err)

	// Check that the managed certificate was inserted correctly
	profile, err = ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, host.UUID, profile.HostUUID)
	assert.Equal(t, initialCP.ProfileUUID, profile.ProfileUUID)
	require.NotNil(t, profile.ChallengeRetrievedAt)
	assert.Equal(t, &challengeRetrievedAt, profile.ChallengeRetrievedAt)

	// Cleanup should do nothing
	err = ds.CleanUpMDMManagedCertificates(ctx)
	require.NoError(t, err)
	profile, err = ds.GetHostMDMCertificateProfile(ctx, host.UUID, initialCP.ProfileUUID)
	require.NoError(t, err)
	require.NotNil(t, profile)

	badProfileUUID := uuid.NewString()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm_managed_certificates (host_uuid, profile_uuid) VALUES (?, ?)
		`, host.UUID, badProfileUUID)
		if err != nil {
			return err
		}
		return nil
	})
	var uid string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM host_mdm_managed_certificates WHERE profile_uuid = ?`,
			badProfileUUID)
	})
	require.Equal(t, badProfileUUID, uid)

	// Cleanup should delete the above orphaned record
	err = ds.CleanUpMDMManagedCertificates(ctx)
	require.NoError(t, err)
	err = ExecAdhocSQLWithError(ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM host_mdm_managed_certificates WHERE profile_uuid = ?`,
			badProfileUUID)
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testAppleMDMSetBatchAsyncLastSeenAt(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create some hosts, all enrolled
	enrolledHosts := make([]*fleet.Host, 2)
	for i := 0; i < len(enrolledHosts); i++ {
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

	getHostLastSeenAt := func(h *fleet.Host) time.Time {
		var lastSeenAt time.Time
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &lastSeenAt, `SELECT last_seen_at FROM nano_enrollments WHERE device_id = ?`, h.UUID)
		})
		return lastSeenAt
	}

	storage, err := ds.NewTestMDMAppleMDMStorage(2, 5*time.Second)
	require.NoError(t, err)
	commander := apple_mdm.NewMDMAppleCommander(storage, pusherFunc(okPusherFunc))

	// enqueue a command for a couple of enrolled hosts
	uuid1 := uuid.NewString()
	rawCmd1 := createRawAppleCmd("ProfileList", uuid1)
	err = commander.EnqueueCommand(ctx, []string{enrolledHosts[0].UUID, enrolledHosts[1].UUID}, rawCmd1)
	require.NoError(t, err)

	// at this point, last_seen_at is still the original value
	ts1, ts2 := getHostLastSeenAt(enrolledHosts[0]), getHostLastSeenAt(enrolledHosts[1])

	time.Sleep(time.Second + time.Millisecond) // ensure a distinct mysql timestamp

	// simulate a result for enrolledHosts[0]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[0].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid1,
		Status:      "Acknowledged",
		Raw:         []byte(rawCmd1),
	})
	require.NoError(t, err)

	// simulate a result for enrolledHosts[1]
	err = storage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: enrolledHosts[1].UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid1,
		Status:      "Error",
		Raw:         []byte(rawCmd1),
	})
	require.NoError(t, err)

	// timestamps should've been updated
	ts1b, ts2b := getHostLastSeenAt(enrolledHosts[0]), getHostLastSeenAt(enrolledHosts[1])
	require.True(t, ts1b.After(ts1))
	require.True(t, ts2b.After(ts2))
}

func testMDMAppleProfileLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	matchProfiles := func(want, got []*fleet.MDMAppleProfilePayload) {
		// match only the fields we care about
		for _, p := range got {
			assert.NotEmpty(t, p.Checksum)
			p.Checksum = nil
			p.SecretsUpdatedAt = nil
		}
		require.ElementsMatch(t, want, got)
	}

	globProf1, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "N1", "I1", "z"))
	require.NoError(t, err)
	globProf2, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "N2", "I2", "x"))
	require.NoError(t, err)

	globalPfs, err := ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, globalPfs, 2)

	// if there are no hosts, then no profilesToInstall need to be installed
	profilesToInstall, err := ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	require.Empty(t, profilesToInstall)

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
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: "test-uuid-1", HostPlatform: "darwin"},
	}, profilesToInstall)

	hostLabel, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host-name-label",
		OsqueryHostID: ptr.String("1337_label"),
		NodeKey:       ptr.String("1337_label"),
		UUID:          "test-uuid-1-label",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	// add a user enrollment for this device, nothing else should be modified
	nanoEnroll(t, ds, hostLabel, true)

	// include-any labels
	l1, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-1", Query: "select 1"})
	require.NoError(t, err)

	l2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-2", Query: "select 1"})
	require.NoError(t, err)

	l3, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-3", Query: "select 1"})
	require.NoError(t, err)

	// include-all labels
	l4, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-4", Query: "select 1"})
	require.NoError(t, err)

	l5, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-5", Query: "select 1"})
	require.NoError(t, err)

	// exclude-any labels
	l6, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-any-6", Query: "select 1"})
	require.NoError(t, err)

	l7, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-any-7", Query: "select 1"})
	require.NoError(t, err)

	profIncludeAny, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "prof-include-any", "prof-include-any", "prof-include-any", l1, l2, l3))
	require.NoError(t, err)
	profIncludeAll, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "prof-include-all", "prof-include-all", "prof-include-all", l4, l5))
	require.NoError(t, err)
	profExcludeAny, err := ds.NewMDMAppleConfigProfile(ctx, *configProfileForTest(t, "prof-exclude-any", "prof-exclude-any", "prof-exclude-any", l6, l7))
	require.NoError(t, err)

	// Update hosts' labels updated at timestamp so that the exclude any profile shows up
	hostLabel.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, hostLabel)
	require.NoError(t, err)

	host1.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, host1)
	require.NoError(t, err)

	// hostLabel is a member of l1, l4, l5
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l1.ID, hostLabel.ID}, {l4.ID, hostLabel.ID}, {l5.ID, hostLabel.ID}})
	require.NoError(t, err)

	globalPfs, err = ds.ListMDMAppleConfigProfiles(ctx, ptr.Uint(0))
	require.NoError(t, err)
	require.Len(t, globalPfs, 5)

	// still the same profiles to assign (plus the one for hostLabel) as there are no profiles for team 1
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)

	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},

		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profIncludeAny.ProfileUUID, ProfileIdentifier: profIncludeAny.Identifier, ProfileName: profIncludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profIncludeAll.ProfileUUID, ProfileIdentifier: profIncludeAll.Identifier, ProfileName: profIncludeAll.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
	}, profilesToInstall)

	// Remove the l1<->hostLabel relationship, but add l2<->hostLabel. The profile should still show
	// up since it's "include any"
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l1.ID, hostLabel.ID}})
	require.NoError(t, err)

	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l2.ID, hostLabel.ID}})
	require.NoError(t, err)

	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)

	require.NoError(t, err)
	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},

		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profIncludeAny.ProfileUUID, ProfileIdentifier: profIncludeAny.Identifier, ProfileName: profIncludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profIncludeAll.ProfileUUID, ProfileIdentifier: profIncludeAll.Identifier, ProfileName: profIncludeAll.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
	}, profilesToInstall)

	// Remove the l2<->hostLabel relationship. The profie should no longer show up since it's
	// include-any
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l2.ID, hostLabel.ID}})
	require.NoError(t, err)
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)

	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},

		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profIncludeAll.ProfileUUID, ProfileIdentifier: profIncludeAll.Identifier, ProfileName: profIncludeAll.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
	}, profilesToInstall)

	// Remove the l4<->hostLabel relationship. Since the profile is "include-all", it should no longer show
	// up even though the l5<->hostLabel connection is still there.
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{l4.ID, hostLabel.ID}})
	require.NoError(t, err)
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)

	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},

		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
	}, profilesToInstall)

	// Add a l6<->host relationship. The exclude-any profile should no longer be assigned to hostLabel.
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{l6.ID, hostLabel.ID}})
	require.NoError(t, err)
	profilesToInstall, err = ds.ListMDMAppleProfilesToInstall(ctx)
	require.NoError(t, err)

	matchProfiles([]*fleet.MDMAppleProfilePayload{
		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},
		{ProfileUUID: profExcludeAny.ProfileUUID, ProfileIdentifier: profExcludeAny.Identifier, ProfileName: profExcludeAny.Name, HostUUID: host1.UUID, HostPlatform: "darwin"},

		{ProfileUUID: globProf1.ProfileUUID, ProfileIdentifier: globProf1.Identifier, ProfileName: globProf1.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
		{ProfileUUID: globProf2.ProfileUUID, ProfileIdentifier: globProf2.Identifier, ProfileName: globProf2.Name, HostUUID: hostLabel.UUID, HostPlatform: "darwin"},
	}, profilesToInstall)
}

func testAggregateMacOSSettingsAllPlatforms(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create macOS/iOS/iPadOS devices on "No team".
	var hosts []*fleet.Host
	for i, platform := range []string{"darwin", "ios", "ipados"} {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        fmt.Sprintf("hostname_%d", i),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("osquery-host-id_%d", i)),
			NodeKey:         ptr.String(fmt.Sprintf("node-key_%d", i)),
			UUID:            fmt.Sprintf("uuid_%d", i),
			HardwareSerial:  fmt.Sprintf("serial_%d", i),
			Platform:        platform,
		})
		require.NoError(t, err)
		nanoEnrollAndSetHostMDMData(t, ds, host, false)
		hosts = append(hosts, host)
	}

	// Create a profile for "No team".
	cp, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("foobar", "barfoo", 0))
	require.NoError(t, err)

	// Upsert the profile with nil status, should be counted as pending.
	upsertHostCPs(hosts, []*fleet.MDMAppleConfigProfile{cp}, fleet.MDMOperationTypeInstall, nil, ctx, ds, t)
	res, err := ds.GetMDMAppleProfilesSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.EqualValues(t, len(hosts), res.Pending)
	require.EqualValues(t, 0, res.Failed)
	require.EqualValues(t, 0, res.Verifying)
	require.EqualValues(t, 0, res.Verified)
}

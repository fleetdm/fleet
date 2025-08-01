package mysql

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
		{"TestMDMProfilesSummaryAndHostFilters", testMDMProfilesSummaryAndHostFilters},
		{"TestIsHostConnectedToFleetMDM", testIsHostConnectedToFleetMDM},
		{"TestAreHostsConnectedToFleetMDM", testAreHostsConnectedToFleetMDM},
		{"TestBulkSetPendingMDMHostProfilesExcludeAny", testBulkSetPendingMDMHostProfilesExcludeAny},
		{"TestBulkSetPendingMDMHostProfilesLotsOfHosts", testBulkSetPendingMDMWindowsHostProfilesLotsOfHosts},
		{"TestBatchResendProfileToHosts", testBatchResendProfileToHosts},
		{"TestGetMDMConfigProfileStatus", testGetMDMConfigProfileStatus},
		{"TestDeleteMDMProfilesCancelsInstalls", testDeleteMDMProfilesCancelsInstalls},
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
		Hostname:       "test-host", // ambiguous hostname shared with macOS host
		OsqueryHostID:  ptr.String("osquery-windows"),
		NodeKey:        ptr.String("node-key-windows"),
		UUID:           uuid.NewString(),
		Platform:       "windows",
		HardwareSerial: "123456",
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
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(
		ctx,
		windowsEnrollment.HostUUID,
		windowsEnrollment.MDMDeviceID,
	)
	require.NoError(t, err)
	windowsEnrollment, err = ds.MDMWindowsGetEnrolledDeviceWithDeviceID(
		ctx,
		windowsEnrollment.MDMDeviceID,
	)
	require.NoError(t, err)

	// enroll a macOS device
	macH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "test-host", // ambiguous hostname shared with windows host
		OsqueryHostID:  ptr.String("osquery-macos"),
		NodeKey:        ptr.String("node-key-macos"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654321",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, macH, false)

	// no commands => no results
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{},
	)
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
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{},
	)
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
		Raw:         []byte(appleCmd),
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(
			ctx,
			`INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`,
			windowsEnrollment.ID,
			"",
		)
		if err != nil {
			return err
		}
		resID, _ := res.LastInsertId()
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, status_code, response_id) VALUES (?, ?, ?, ?, ?)`,
			windowsEnrollment.ID,
			winCmd.CommandUUID,
			"",
			"200",
			resID,
		)
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

	// add more windows commands
	winCmd2 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri2",
	}
	winCmd3 := &fleet.MDMWindowsCommand{
		CommandUUID:  uuid.NewString(),
		RawCommand:   []byte("<Exec></Exec>"),
		TargetLocURI: "./test/uri3",
	}
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{windowsEnrollment.MDMDeviceID}, winCmd2)
	require.NoError(t, err)
	err = ds.MDMWindowsInsertCommandForHosts(ctx, []string{windowsEnrollment.MDMDeviceID}, winCmd3)
	require.NoError(t, err)

	// add more macos commands
	appleCmdUUID2 := uuid.New().String()
	appleCmd2 := createRawAppleCmd("InstallProfile", appleCmdUUID2)
	err = commander.EnqueueCommand(ctx, []string{macH.UUID}, appleCmd2)
	require.NoError(t, err)

	appleCmdUUID3 := uuid.New().String()
	appleCmd3 := createRawAppleCmd("RemoveProfile", appleCmdUUID3)
	err = commander.EnqueueCommand(ctx, []string{macH.UUID}, appleCmd3)
	require.NoError(t, err)

	// non-existent host identifier
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			Filters: fleet.MDMCommandFilters{
				HostIdentifier: "non-existent",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, cmds, 0)

	// non-existent request type
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			Filters: fleet.MDMCommandFilters{
				RequestType: "non-existent",
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, cmds, 0)

	for _, tc := range []struct {
		name       string
		identifier string
		expected   []string
	}{
		{
			name:       "windows host by hostname ambiguous with macOS host",
			identifier: windowsH.Hostname,
			expected: []string{
				winCmd.CommandUUID, winCmd2.CommandUUID, winCmd3.CommandUUID,
				appleCmdUUID, appleCmdUUID2, appleCmdUUID3,
			},
		},
		{
			name:       "windows host by UUID",
			identifier: windowsH.UUID,
			expected:   []string{winCmd.CommandUUID, winCmd2.CommandUUID, winCmd3.CommandUUID},
		},
		{
			name:       "windows host by hardware serial",
			identifier: windowsH.HardwareSerial,
			expected:   []string{winCmd.CommandUUID, winCmd2.CommandUUID, winCmd3.CommandUUID},
		},
		{
			name:       "macOS host by hostname ambiguous with windows host",
			identifier: macH.Hostname,
			expected: []string{
				appleCmdUUID, appleCmdUUID2, appleCmdUUID3,
				winCmd.CommandUUID, winCmd2.CommandUUID, winCmd3.CommandUUID,
			},
		},
		{
			name:       "macOS host by UUID",
			identifier: macH.UUID,
			expected:   []string{appleCmdUUID, appleCmdUUID2, appleCmdUUID3},
		},
		{
			name:       "macOS host by hardware serial",
			identifier: macH.HardwareSerial,
			expected:   []string{appleCmdUUID, appleCmdUUID2, appleCmdUUID3},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmds, err := ds.ListMDMCommands(
				ctx,
				fleet.TeamFilter{User: test.UserAdmin},
				&fleet.MDMCommandListOptions{
					Filters: fleet.MDMCommandFilters{
						HostIdentifier: tc.identifier,
					},
				},
			)
			require.NoError(t, err)
			require.Len(t, cmds, len(tc.expected))
			var got []string
			for _, cmd := range cmds {
				got = append(got, cmd.CommandUUID)
			}
			require.ElementsMatch(t, tc.expected, got)
		})
	}

	// add macos host
	macH2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-2",
		OsqueryHostID:  ptr.String("osquery-macos-2"),
		NodeKey:        ptr.String("node-key-macos-2"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654322",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, macH2, false)

	// add more macos commands
	appleCmdUUID4 := uuid.New().String()
	appleCmd4 := createRawAppleCmd("InstallProfile", appleCmdUUID4)
	err = commander.EnqueueCommand(ctx, []string{macH2.UUID}, appleCmd4)
	require.NoError(t, err)

	// filter by request_type
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			Filters: fleet.MDMCommandFilters{
				RequestType: "InstallProfile",
			},
			ListOptions: fleet.ListOptions{OrderKey: "hostname", OrderDirection: fleet.OrderAscending},
		},
	)
	require.NoError(t, err)
	require.Len(t, cmds, 2)
	require.Equal(t, appleCmdUUID4, cmds[0].CommandUUID)
	require.Equal(t, appleCmdUUID2, cmds[1].CommandUUID)

	// filter by request_type and host_identifier
	cmds, err = ds.ListMDMCommands(
		ctx,
		fleet.TeamFilter{User: test.UserAdmin},
		&fleet.MDMCommandListOptions{
			Filters: fleet.MDMCommandFilters{
				RequestType:    "InstallProfile",
				HostIdentifier: macH.UUID,
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, cmds, 1)
	require.Equal(t, appleCmdUUID2, cmds[0].CommandUUID)
}

func testBatchSetMDMProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	applyAndExpect := func(
		newAppleSet []*fleet.MDMAppleConfigProfile,
		newWindowsSet []*fleet.MDMWindowsConfigProfile,
		newAppleDeclSet []*fleet.MDMAppleDeclaration,
		tmID *uint,
		wantApple []*fleet.MDMAppleConfigProfile,
		wantWindows []*fleet.MDMWindowsConfigProfile,
		wantAppleDecl []*fleet.MDMAppleDeclaration,
		wantUpdates fleet.MDMProfilesUpdates,
	) {
		updates, err := ds.BatchSetMDMProfiles(ctx, tmID, newAppleSet, newWindowsSet, newAppleDeclSet, nil)
		require.NoError(t, err)
		expectAppleProfiles(t, ds, tmID, wantApple)
		expectWindowsProfiles(t, ds, tmID, wantWindows)
		expectAppleDeclarations(t, ds, tmID, wantAppleDecl)
		assert.Equal(t, wantUpdates, updates)
	}

	withTeamIDApple := func(p *fleet.MDMAppleConfigProfile, tmID uint) *fleet.MDMAppleConfigProfile {
		p.TeamID = &tmID
		return p
	}

	withTeamIDDecl := func(d *fleet.MDMAppleDeclaration, tmID uint) *fleet.MDMAppleDeclaration {
		d.TeamID = &tmID
		return d
	}

	withTeamIDWindows := func(p *fleet.MDMWindowsConfigProfile, tmID uint) *fleet.MDMWindowsConfigProfile {
		p.TeamID = &tmID
		return p
	}

	// empty set for no team (both Apple and Windows)
	applyAndExpect(nil, nil, nil, nil, nil, nil, nil, fleet.MDMProfilesUpdates{})

	// single Apple and Windows profile set for a specific team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
		[]*fleet.MDMAppleDeclaration{declForTest("D1", "D1", "foo")},
		ptr.Uint(1),
		[]*fleet.MDMAppleConfigProfile{
			withTeamIDApple(configProfileForTest(t, "N1", "I1", "a"), 1),
		},
		[]*fleet.MDMWindowsConfigProfile{
			withTeamIDWindows(windowsConfigProfileForTest(t, "W1", "l1"), 1),
		},
		[]*fleet.MDMAppleDeclaration{withTeamIDDecl(declForTest("D1", "D1", "foo"), 1)},
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
	)

	// single Apple and Windows profile set for no team
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
		[]*fleet.MDMAppleDeclaration{declForTest("D1", "D1", "foo")},
		nil,
		[]*fleet.MDMAppleConfigProfile{configProfileForTest(t, "N1", "I1", "a")},
		[]*fleet.MDMWindowsConfigProfile{windowsConfigProfileForTest(t, "W1", "l1")},
		[]*fleet.MDMAppleDeclaration{declForTest("D1", "D1", "foo")},
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
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
		[]*fleet.MDMAppleDeclaration{
			declForTest("D1", "D1", "foo"), // unchanged
			declForTest("D2", "D2", "foo"),
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
		[]*fleet.MDMAppleDeclaration{
			withTeamIDDecl(declForTest("D1", "D1", "foo"), 1),
			withTeamIDDecl(declForTest("D2", "D2", "foo"), 1),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
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
		[]*fleet.MDMAppleDeclaration{
			declForTest("D1", "D1", "foo-updated"), // content updated
			declForTest("D2", "D2", "foo"),         // unchanged
			declForTest("D3", "D3", "bar"),         // new
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
		[]*fleet.MDMAppleDeclaration{
			withTeamIDDecl(declForTest("D1", "D1", "foo-updated"), 1),
			withTeamIDDecl(declForTest("D2", "D2", "foo"), 1),
			withTeamIDDecl(declForTest("D3", "D3", "bar"), 1),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
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
		[]*fleet.MDMAppleDeclaration{
			declForTest("D5", "D4", "foo"),
			declForTest("D4", "D5", "foo"),
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
		[]*fleet.MDMAppleDeclaration{
			declForTest("D5", "D4", "foo"),
			declForTest("D4", "D5", "foo"),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
	)

	// Apply the same profiles again -- no update should be detected
	applyAndExpect(
		[]*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "N4", "I4", "d"),
			configProfileForTest(t, "N5", "I5", "e"),
		},
		[]*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "W4", "l4"),
			windowsConfigProfileForTest(t, "W5", "l5"),
		},
		[]*fleet.MDMAppleDeclaration{
			declForTest("D5", "D4", "foo"),
			declForTest("D4", "D5", "foo"),
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
		[]*fleet.MDMAppleDeclaration{
			declForTest("D5", "D4", "foo"),
			declForTest("D4", "D5", "foo"),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: false, WindowsConfigProfile: false, AppleDeclaration: false},
	)

	// Test Case 8: Clear profiles for a specific team
	applyAndExpect(nil, nil, nil, ptr.Uint(1), nil, nil, nil,
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
	)

	// create some labels to test batch-setting label-scoped declarations
	lblExcl, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-label-1", Query: "select 1"})
	require.NoError(t, err)
	lblExcl2, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-label-2", Query: "select 2"})
	require.NoError(t, err)
	lblInclAny, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-label-3", Query: "select 3"})
	require.NoError(t, err)
	lblInclAny2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-label-4", Query: "select 4"})
	require.NoError(t, err)
	lblInclAll, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-label-5", Query: "select 5"})
	require.NoError(t, err)
	lblInclAll2, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-label-6", Query: "select 6"})
	require.NoError(t, err)

	// we only care about declarations here, as batch-setting labels for profiles
	// is tested elsewhere.
	applyAndExpect(nil, nil, []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "foo", lblExcl, lblExcl2),
		declForTest("D2", "D2", "foo", lblInclAll, lblInclAll2),
	}, nil,
		nil, nil, []*fleet.MDMAppleDeclaration{
			declForTest("D1", "D1", "foo", lblExcl, lblExcl2),
			declForTest("D2", "D2", "foo", lblInclAll, lblInclAll2),
		},
		// this removed the apple and windows profiles for no team, so updated is true
		fleet.MDMProfilesUpdates{AppleConfigProfile: true, WindowsConfigProfile: true, AppleDeclaration: true},
	)

	applyAndExpect(nil, nil, []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "foo", lblInclAny, lblInclAny2),
		declForTest("D2", "D2", "foo"),
	}, nil,
		nil, nil, []*fleet.MDMAppleDeclaration{
			declForTest("D1", "D1", "foo", lblInclAny, lblInclAny2),
			declForTest("D2", "D2", "foo"),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: false, WindowsConfigProfile: false, AppleDeclaration: true},
	)
	applyAndExpect(nil, nil, []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "foo"),
	}, nil,
		nil, nil, []*fleet.MDMAppleDeclaration{
			declForTest("D1", "D1", "foo"),
		},
		fleet.MDMProfilesUpdates{AppleConfigProfile: false, WindowsConfigProfile: false, AppleDeclaration: true},
	)
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
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, team.ID), nil)
		require.NoError(t, err)
		_, err = ds.NewMDMAppleConfigProfile(ctx, *generateCP("name_"+idf, idf, 0), nil)
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
		_, err = ds.NewMDMWindowsConfigProfile(
			ctx,
			fleet.MDMWindowsConfigProfile{Name: name, TeamID: &team.ID, SyncML: winProf},
		)
		require.NoError(t, err)
		_, err = ds.NewMDMWindowsConfigProfile(
			ctx,
			fleet.MDMWindowsConfigProfile{Name: name, TeamID: nil, SyncML: winProf},
		)
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
	profA, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("A", "A", 0), nil)
	require.NoError(t, err)
	profB, err := ds.NewMDMWindowsConfigProfile(
		ctx,
		fleet.MDMWindowsConfigProfile{Name: "B", TeamID: &team.ID, SyncML: winProf},
	)
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
			acp.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[0].Name, LabelID: labels[0].ID},
				{LabelName: labels[1].Name, LabelID: labels[1].ID},
			}
		}
		_, err = ds.NewMDMAppleConfigProfile(ctx, acp, nil)
		require.NoError(t, err)

		acp = *generateCP(string(rune('C'+inc+1)), string(rune('C'+inc+1)), team.ID)
		if i == 0 {
			acp.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[2].Name, LabelID: labels[2].ID},
				{LabelName: labels[3].Name, LabelID: labels[3].ID},
			}
		}
		_, err = ds.NewMDMAppleConfigProfile(ctx, acp, nil)
		require.NoError(t, err)

		wcp := fleet.MDMWindowsConfigProfile{
			Name:   string(rune('C' + inc + 2)),
			TeamID: nil,
			SyncML: winProf,
		}
		if i == 0 {
			wcp.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
				{LabelName: labels[4].Name, LabelID: labels[4].ID},
				{LabelName: labels[5].Name, LabelID: labels[5].ID},
			}
		}
		_, err = ds.NewMDMWindowsConfigProfile(ctx, wcp)
		require.NoError(t, err)

		wcp = fleet.MDMWindowsConfigProfile{
			Name:   string(rune('C' + inc + 3)),
			TeamID: &team.ID,
			SyncML: winProf,
		}
		if i == 0 {
			wcp.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
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
			{LabelName: labels[0].Name, LabelID: labels[0].ID, RequireAll: true},
			{LabelName: labels[1].Name, LabelID: labels[1].ID, RequireAll: true},
		},
		"D": {
			{LabelName: labels[2].Name, LabelID: labels[2].ID, RequireAll: true},
			{LabelName: labels[3].Name, LabelID: 0, Broken: true, RequireAll: true},
		},
		"E": {
			{LabelName: labels[4].Name, LabelID: 0, Broken: true, RequireAll: true},
			{LabelName: labels[5].Name, LabelID: labels[5].ID, RequireAll: true},
		},
		"F": {
			{LabelName: labels[6].Name, LabelID: labels[6].ID, RequireAll: true},
			{LabelName: labels[7].Name, LabelID: labels[7].ID, RequireAll: true},
		},
	}

	cases := []struct {
		desc      string
		tmID      *uint
		opts      fleet.ListOptions
		wantNames []string
		wantMeta  fleet.PaginationMetadata
	}{
		{
			"all global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true},
			[]string{"A", "C", "E", "G", "I", "K", "M"},
			fleet.PaginationMetadata{},
		},
		{
			"all team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true},
			[]string{"B", "D", "F", "H", "J", "L", "N"},
			fleet.PaginationMetadata{},
		},

		{
			"page 0 per page 2, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2},
			[]string{"A", "C"},
			fleet.PaginationMetadata{HasNextResults: true},
		},
		{
			"page 1 per page 2, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 1},
			[]string{"E", "G"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 2 per page 2, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 2},
			[]string{"I", "K"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 3 per page 2, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 3},
			[]string{"M"},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},
		{
			"page 4 per page 2, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 4},
			[]string{},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},

		{
			"page 0 per page 2, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2},
			[]string{"B", "D"},
			fleet.PaginationMetadata{HasNextResults: true},
		},
		{
			"page 1 per page 2, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 1},
			[]string{"F", "H"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 2 per page 2, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 2},
			[]string{"J", "L"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 3 per page 2, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 3},
			[]string{"N"},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},
		{
			"page 4 per page 2, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 2, Page: 4},
			[]string{},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},

		{
			"page 0 per page 3, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3},
			[]string{"A", "C", "E"},
			fleet.PaginationMetadata{HasNextResults: true},
		},
		{
			"page 1 per page 3, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 1},
			[]string{"G", "I", "K"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 2 per page 3, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 2},
			[]string{"M"},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},
		{
			"page 3 per page 3, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 3},
			[]string{},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},

		{
			"page 0 per page 3, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3},
			[]string{"B", "D", "F"},
			fleet.PaginationMetadata{HasNextResults: true},
		},
		{
			"page 1 per page 3, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 1},
			[]string{"H", "J", "L"},
			fleet.PaginationMetadata{HasPreviousResults: true, HasNextResults: true},
		},
		{
			"page 2 per page 3, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 2},
			[]string{"N"},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},
		{
			"page 3 per page 3, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: true, PerPage: 3, Page: 3},
			[]string{},
			fleet.PaginationMetadata{HasPreviousResults: true},
		},

		{
			"no metadata, global",
			nil,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: false, PerPage: 2, Page: 1},
			[]string{"E", "G"},
			fleet.PaginationMetadata{},
		},
		{
			"no metadata, team",
			&team.ID,
			fleet.ListOptions{OrderKey: "name", IncludeMetadata: false, PerPage: 2, Page: 1},
			[]string{"F", "H"},
			fleet.PaginationMetadata{},
		},
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
				require.Equal(t, len(wantProfs), len(p.LabelsIncludeAll), "profile name: %s", p.Name)
				if len(wantProfs) > 0 {
					// clear the profile uuids from the labels list
					for i, l := range p.LabelsIncludeAll {
						l.ProfileUUID = ""
						p.LabelsIncludeAll[i] = l
					}
					require.ElementsMatch(t, wantProfs, p.LabelsIncludeAll, "profile name: %s", p.Name)
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

type anyProfile struct {
	ProfileUUID      string
	Status           *fleet.MDMDeliveryStatus
	OperationType    fleet.MDMOperationType
	IdentifierOrName string
}

// only asserts the profile ID, status and operation
func assertHostProfiles(t *testing.T, ds *Datastore, want map[*fleet.Host][]anyProfile) {
	ctx := context.Background()
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
			require.Equal(
				t,
				wp.ProfileUUID,
				gp.ProfileUUID,
				"host uuid: %s, prof id or name: %s",
				h.UUID,
				gp.IdentifierOrName,
			)
			require.Equal(
				t,
				wp.Status,
				gp.Status,
				"host uuid: %s, prof id or name: %s",
				h.UUID,
				gp.IdentifierOrName,
			)
			require.Equal(
				t,
				wp.OperationType,
				gp.OperationType,
				"host uuid: %s, prof id or name: %s",
				h.UUID,
				gp.IdentifierOrName,
			)
		}
	}
}

func testBulkSetPendingMDMHostProfiles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	// NOTE: this test is now a monster, it's pretty much impossible to change as it's too big
	// to understand what the expected assertion 500 lines in is supposed to be. Please avoid
	// adding to it.

	hostIDsFromHosts := func(hosts ...*fleet.Host) []uint {
		ids := make([]uint, len(hosts))
		for i, h := range hosts {
			ids[i] = h.ID
		}
		return ids
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
	updates, err := ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)

	// bulk set for combination of target ids, not allowed
	_, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{1}, []uint{2}, nil, nil)
	require.Error(t, err)

	// bulk set for all created hosts, no profiles yet so nothing changed
	allHosts := darwinHosts
	allHosts = append(allHosts, unenrolledHost, linuxHost)
	allHosts = append(allHosts, windowsHosts...)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.False(t, updates.WindowsConfigProfile)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
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
	macGlobalDeclarations := []*fleet.MDMAppleDeclaration{
		declForTest("G1d", "G1d", "foo"),
		declForTest("G2d", "G2d", "bar"),
	}
	winGlobalProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "G1w", "L1"),
		windowsConfigProfileForTest(t, "G2w", "L2"),
		windowsConfigProfileForTest(t, "G3w", "L3"),
	}
	updates, err = ds.BatchSetMDMProfiles(
		ctx,
		nil,
		macGlobalProfiles,
		winGlobalProfiles,
		macGlobalDeclarations,
		nil,
	)
	require.NoError(t, err)
	macGlobalProfiles, err = ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	require.Len(t, macGlobalProfiles, 3)
	globalProfiles := getProfs(nil)
	require.Len(t, globalProfiles, 8)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)

	// list profiles to install, should result in the global profiles for all
	// enrolled hosts
	toInstallDarwin, err := ds.ListMDMAppleProfilesToInstall(ctx, "")
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
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, hostIDsFromHosts(allHosts...), nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// move darwinHosts[0] and windowsHosts[0] to that team
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{darwinHosts[0].ID, windowsHosts[0].ID}))
	require.NoError(t, err)

	// 6 are still reported as "to install" because op=install and status=nil
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx, "")
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
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(darwinHosts[0], windowsHosts[0]),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		// windows profiles are directly deleted without a pending state (there's no on-host removal of profiles)
		windowsHosts[0]: {},
		windowsHosts[1]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// create another team
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	// move enrolledHosts[1] to that team
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{darwinHosts[1].ID, windowsHosts[1].ID}))
	require.NoError(t, err)

	// 3 are still reported as "to install" because op=install and status=nil
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx, "")
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
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		nil,
		nil,
		nil,
		[]string{darwinHosts[1].UUID, windowsHosts[1].UUID},
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost:  {},
		linuxHost:       {},
		windowsHosts[0]: {},
		// windows profiles are directly deleted without a pending state
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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
	updates, err = ds.BatchSetMDMProfiles(ctx, &team1.ID, tm1DarwinProfiles, tm1WindowsProfiles, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)

	tm1Profiles := getProfs(&team1.ID)
	require.Len(t, tm1Profiles, 4)

	// 5 are now reported as "to install" (3 global + 2 team1)
	toInstallDarwin, err = ds.ListMDMAppleProfilesToInstall(ctx, "")
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
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assert.True(t, updates.WindowsConfigProfile)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
			{
				ProfileUUID:      tm1Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm1Profiles[0].Identifier,
			},
			{
				ProfileUUID:      tm1Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm1Profiles[1].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   tm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[0].ProfileUUID, ProfileIdentifier: darwinGlobalProfiles[0].Identifier,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"), Scope: fleet.PayloadScopeSystem,
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[1].ProfileUUID, ProfileIdentifier: darwinGlobalProfiles[1].Identifier,
			Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"), Scope: fleet.PayloadScopeSystem,
		},
		{
			HostUUID: darwinHosts[0].UUID, ProfileUUID: darwinGlobalProfiles[2].ProfileUUID, ProfileIdentifier: darwinGlobalProfiles[2].Identifier,
			Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Checksum: []byte("csum"), Scope: fleet.PayloadScopeSystem,
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

	updates, err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	newTm1Profiles := getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 4)

	// update status of the affected team
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      newTm1Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[0].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryFailed,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// update again -- nothing should change
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

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

	updates, err = ds.BatchSetMDMProfiles(ctx, &team1.ID, newTm1DarwinProfiles, newTm1WindowsProfiles, nil, nil)
	require.NoError(t, err)
	newTm1Profiles = getProfs(&team1.ID)
	require.Len(t, newTm1Profiles, 6)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// update status of the affected team
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team1.ID}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      newTm1Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[0].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[1].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryFailed,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[1].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[3].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[3].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   globalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   globalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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

	// TODO(roberto): add new darwin declarations for this and all subsequent assertions
	updates, err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.True(t, updates.AppleDeclaration)

	newGlobalProfiles := getProfs(nil)
	require.Len(t, newGlobalProfiles, 6)

	// update status of the affected "no-team"
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{0}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration) // host status for deleted declaration was already set on delete

	require.NoError(t, ds.MDMAppleStoreDDMStatusReport(ctx, darwinHosts[0].UUID, nil))
	require.NoError(t, ds.MDMAppleStoreDDMStatusReport(ctx, darwinHosts[1].UUID, nil))
	require.NoError(t, ds.MDMAppleStoreDDMStatusReport(ctx, darwinHosts[2].UUID, nil))

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryFailed,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[0].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[1].Identifier,
			},
			{
				ProfileUUID:      newTm1Profiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: newTm1Profiles[2].Identifier,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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

	updates, err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles, nil, nil)
	require.NoError(t, err)
	newGlobalProfiles = getProfs(nil)
	require.Len(t, newGlobalProfiles, 8)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// bulk-set only those affected by the new Apple global profile
	newDarwinProfileUUID := newGlobalProfiles[3].ProfileUUID
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newDarwinProfileUUID}, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// bulk-set only those affected by the new Apple global profile
	newWindowsProfileUUID := newGlobalProfiles[7].ProfileUUID
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newWindowsProfileUUID}, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// add a profile to team2

	tm2DarwinProfiles := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "T2.1a", "T2.1a", "a"),
	}

	tm2WindowsProfiles := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "T2.1w", "T2.1"),
	}

	updates, err = ds.BatchSetMDMProfiles(ctx, &team2.ID, tm2DarwinProfiles, tm2WindowsProfiles, nil, nil)
	require.NoError(t, err)
	tm2Profiles := getProfs(&team2.ID)
	require.Len(t, tm2Profiles, 2)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// update status via tm2 id and the global 0 id to test that custom sql statement
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID, 0}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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

	updates, err = ds.BatchSetMDMProfiles(ctx, nil, newDarwinGlobalProfiles, newWindowsGlobalProfiles, nil, nil)
	require.NoError(t, err)
	newGlobalProfiles = getProfs(nil)
	require.Len(t, newGlobalProfiles, 12)
	// TODO(mna): temporary until BatchSetMDMProfiles supports labels
	setProfileLabels(t, newGlobalProfiles[4], labels[0], labels[1])
	setProfileLabels(t, newGlobalProfiles[5], labels[2])
	setProfileLabels(t, newGlobalProfiles[10], labels[3], labels[4])
	setProfileLabels(t, newGlobalProfiles[11], labels[5])
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// simulate an entry with some values set to NULL
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(
			ctx,
			`UPDATE host_mdm_apple_profiles SET detail = NULL WHERE profile_uuid = ?`,
			globalProfiles[2].ProfileUUID,
		)
		return err
	})

	// do a sync of all hosts, should not change anything as no host is a member
	// of the new label-based profiles (indices change due to new Apple and
	// Windows profiles)
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(
			append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
			{
				ProfileUUID:      tm2Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm2Profiles[0].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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
		{labels[0].ID, darwinHosts[3].ID},
		{labels[1].ID, darwinHosts[3].ID},
		{labels[3].ID, windowsHosts[3].ID},
		{labels[4].ID, windowsHosts[3].ID},
	})
	require.NoError(t, err)

	// do a full sync, the new global hosts get the standard global profiles and
	// also the label-based profile that they are a member of
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(
			append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// make the darwinHosts[2] host a member of all labels
	// make the windowsHosts[2] host a member of all labels
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[0].ID, darwinHosts[2].ID},
		{labels[1].ID, darwinHosts[2].ID},
		{labels[2].ID, darwinHosts[2].ID},
		{labels[3].ID, darwinHosts[2].ID},
		{labels[4].ID, darwinHosts[2].ID},
		{labels[5].ID, darwinHosts[2].ID},
		{labels[0].ID, windowsHosts[2].ID},
		{labels[1].ID, windowsHosts[2].ID},
		{labels[2].ID, windowsHosts[2].ID},
		{labels[3].ID, windowsHosts[2].ID},
		{labels[4].ID, windowsHosts[2].ID},
		{labels[5].ID, windowsHosts[2].ID},
	})
	require.NoError(t, err)

	// do a sync of those hosts, they will get the two label-based profiles of their platform
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(darwinHosts[2], windowsHosts[2]),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[11].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// "break" the two G6 label-based profile by deleting labels[0] and [3]
	require.NoError(t, ds.DeleteLabel(ctx, labels[0].Name))
	require.NoError(t, ds.DeleteLabel(ctx, labels[3].Name))

	// sync the affected profiles
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		nil,
		nil,
		[]string{newGlobalProfiles[4].ProfileUUID},
		nil,
	)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		nil,
		nil,
		[]string{newGlobalProfiles[10].ProfileUUID},
		nil,
	)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// nothing changes - broken label-based profiles are simply ignored
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[11].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// update darwin/windows[2] so they are not members of labels[1][2] and [4][5], which
	// should remove the G7 label-based profile, but not G6 as it is broken.
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[2].ID},
		{labels[2].ID, darwinHosts[2].ID},
		{labels[4].ID, windowsHosts[2].ID},
		{labels[5].ID, windowsHosts[2].ID},
	})
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(darwinHosts[2], windowsHosts[2]),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// "unbreak" the two G6 label-based profiles by removing the deleted labels
	// from their requirements
	setProfileLabels(t, newGlobalProfiles[4], labels[1])
	setProfileLabels(t, newGlobalProfiles[10], labels[4])

	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		nil,
		nil,
		[]string{newGlobalProfiles[4].ProfileUUID},
		nil,
	)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		nil,
		nil,
		[]string{newGlobalProfiles[10].ProfileUUID},
		nil,
	)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
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

	updates, err = ds.BatchSetMDMProfiles(ctx, &team2.ID, tm2DarwinProfiles, tm2WindowsProfiles, nil, nil)
	require.NoError(t, err)
	tm2Profiles = getProfs(&team2.ID)
	require.Len(t, tm2Profiles, 4)
	// TODO(mna): temporary until BatchSetMDMProfiles supports labels
	setProfileLabels(t, tm2Profiles[1], labels[1], labels[2])
	setProfileLabels(t, tm2Profiles[3], labels[4], labels[5])
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// sync team 2, no changes because no host is a member of the labels (except
	// index change due to new profiles)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// make darwinHosts[1] and windowsHosts[1] members of the required labels
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[1].ID},
		{labels[2].ID, darwinHosts[1].ID},
		{labels[4].ID, windowsHosts[1].ID},
		{labels[5].ID, windowsHosts[1].ID},
	})
	require.NoError(t, err)

	// sync team 2, the label-based profile of team2 is now pending install
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
			{
				ProfileUUID:      tm2Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm2Profiles[0].Identifier,
			},
			{
				ProfileUUID:      tm2Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm2Profiles[1].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// "break" the team 2 label-based profile by deleting a label
	require.NoError(t, ds.DeleteLabel(ctx, labels[1].Name))
	require.NoError(t, ds.DeleteLabel(ctx, labels[4].Name))

	// sync team 2, the label-based profile of team2 is left untouched (broken
	// profiles are ignored)
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:      globalProfiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[0].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[2].Identifier,
			},
			{
				ProfileUUID:      globalProfiles[4].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: globalProfiles[4].Identifier,
			},
			{
				ProfileUUID:      tm2Profiles[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm2Profiles[0].Identifier,
			},
			{
				ProfileUUID:      tm2Profiles[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: tm2Profiles[1].Identifier,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// remove team 2 hosts membership from labels
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, darwinHosts[1].ID},
		{labels[2].ID, darwinHosts[1].ID},
		{labels[4].ID, windowsHosts[1].ID},
		{labels[5].ID, windowsHosts[1].ID},
	})
	require.NoError(t, err)

	// sync team 2, the label-based profile of team2 is still left untouched
	// because even if the hosts are not members anymore, the profile is broken
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// "unbreak" the profile by removing the deleted label from its requirements
	setProfileLabels(t, tm2Profiles[1], labels[2])
	setProfileLabels(t, tm2Profiles[3], labels[5])

	// sync team 2, now it sees that the hosts are not members and the profile
	// gets removed
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{team2.ID}, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})

	// sanity-check, a full sync does not change anything
	updates, err = ds.BulkSetPendingMDMHostProfiles(
		ctx,
		hostIDsFromHosts(
			append(darwinHosts, append(windowsHosts, unenrolledHost, linuxHost)...)...),
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		darwinHosts[0]: {
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryFailed,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newTm1Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		darwinHosts[1]: {
			{
				ProfileUUID:   globalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   globalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   tm2Profiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   tm2Profiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
			{
				ProfileUUID:   newGlobalProfiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		darwinHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[0].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[1].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		unenrolledHost: {},
		linuxHost:      {},
		windowsHosts[0]: {
			{
				ProfileUUID:   newTm1Profiles[3].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[4].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newTm1Profiles[5].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[1]: {
			{
				ProfileUUID:   tm2Profiles[2].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[2]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		windowsHosts[3]: {
			{
				ProfileUUID:   newGlobalProfiles[6].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[7].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[8].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[9].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
			{
				ProfileUUID:   newGlobalProfiles[10].ProfileUUID,
				Status:        &fleet.MDMDeliveryPending,
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
	})
}

func testGetHostMDMProfilesExpectedForVerification(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Setup funcs

	// ===================================================
	// MacOS base tests
	// ===================================================
	baseEarliestInstallDate := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
	overrideEarliestInstallDate := time.Now().UTC().Add(-6 * time.Hour).Truncate(time.Second)

	macosBasicTeamProfNoLabelsSetup := func() (uint, *fleet.Host) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-1",
			OsqueryHostID: ptr.String("osquery-macos-1"),
			NodeKey:       ptr.String("node-key-macos-1"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 1"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T1.1", "T1.1", "d"),
			configProfileForTest(t, "T1.2", "T1.2", "e"),
		}

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 2)

		return team.ID, host
	}

	macosLabeledTeamProfSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 2"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T2.1", "T2.1", "d"),
			configProfileForTest(t, "T2.2", "T2.2", "e"),
			configProfileForTest(t, "labeled_prof", "labeled_prof", "labeled_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_1"})
		require.NoError(t, err)

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`,
				"labeled_prof",
			)
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

		return team.ID, host
	}

	macosLabeledTeamProfWithAdditionalLabeledProfSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 3"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
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

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`,
				"labeled_prof_2",
			)
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

		return team.ID, host
	}

	macosProfWithBrokenLabelSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 4"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T4.1", "T4.1", "d"),
			configProfileForTest(t, "T4.2", "T4.2", "e"),
			configProfileForTest(t, "broken_label_prof", "broken_label_prof", "broken_label_prof"),
		}

		testLabel4, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_4"})
		require.NoError(t, err)

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`,
				"broken_label_prof",
			)
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

		return team.ID, host
	}

	macosInstallDateOverrideSetup := func() (uint, *fleet.Host) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-5",
			OsqueryHostID: ptr.String("osquery-macos-5"),
			NodeKey:       ptr.String("node-key-macos-5"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 5"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMAppleConfigProfile{
			configProfileForTest(t, "T5.1", "T5.1", "d"),
			configProfileForTest(t, "T5.2", "T5.2", "e"),
			configProfileForTest(t, "T5.3", "T5.3", "f"),
		}

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 3)

		// We cannot control the generated profile UUIDs here so we need to map them back to the
		// created profiles for test correctness
		for i := 0; i < len(profs); i++ {
			for j := 0; j < len(profiles); j++ {
				if profs[i].Identifier == profiles[j].Identifier {
					profiles[j].ProfileUUID = profs[i].ProfileUUID
					break
				}
			}
		}

		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"UPDATE mdm_apple_configuration_profiles SET uploaded_at = ? WHERE profile_uuid IN (?, ?, ?)",
					baseEarliestInstallDate,
					profiles[0].ProfileUUID,
					profiles[1].ProfileUUID,
					profiles[2].ProfileUUID,
				)
				return err
			},
		)

		// Note what we're doing here is overriding install date for the first profile, creating an
		// HMAP entry that doesn't override it for the second, then the third has no corresponding
		// HMAP entry at all(which also means no override).
		err = ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:        profiles[0].ProfileUUID,
				ProfileIdentifier:  profiles[0].Identifier,
				ProfileName:        profiles[0].Name,
				HostUUID:           host.UUID,
				CommandUUID:        uuid.NewString(),
				VariablesUpdatedAt: &overrideEarliestInstallDate,
				Status:             &fleet.MDMDeliveryVerified,
				OperationType:      fleet.MDMOperationTypeInstall,
				Scope:              fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       profiles[1].ProfileUUID,
				ProfileIdentifier: profiles[1].Identifier,
				ProfileName:       profiles[1].Name,
				HostUUID:          host.UUID,
				CommandUUID:       uuid.NewString(),
				Status:            &fleet.MDMDeliveryVerified,
				OperationType:     fleet.MDMOperationTypeInstall,
				Scope:             fleet.PayloadScopeSystem,
			},
		})
		require.NoError(t, err)
		return team.ID, host
	}

	macosLabeledProfileRulesSetup := func() (uint, *fleet.Host) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "macos-test-6",
			OsqueryHostID: ptr.String("osquery-macos-6"),
			NodeKey:       ptr.String("node-key-macos-6"),
			UUID:          uuid.NewString(),
			Platform:      "darwin",
		})
		require.NoError(t, err)
		nanoEnroll(t, ds, host, false)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "macos team 6"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// Include any labels
		includeAnyMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-macos-matched-label-1"})
		require.NoError(t, err)
		includeAnyMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-macos-matched-label-2"})
		require.NoError(t, err)
		includeAnyUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-macos-unmatched-label"})
		require.NoError(t, err)

		// Include all labels
		includeAllMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-macos-matched-label-1"})
		require.NoError(t, err)
		includeAllMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-macos-matched-label-2"})
		require.NoError(t, err)
		includeAllUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-macos-unmatched-label"})
		require.NoError(t, err)

		// Exclude labels
		excludeMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-macos-matched-label-1"})
		require.NoError(t, err)
		excludeMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-macos-matched-label-2"})
		require.NoError(t, err)
		excludeUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-macos-unmatched-label"})
		require.NoError(t, err)

		// create profiles for team
		// include_any_all_match_prof
		// include_any_one_matches_prof
		// include_all_all_match_prof
		// exclude_none_match_prof
		profiles := []*fleet.MDMAppleConfigProfile{
			// Two profiles with no label rules, thus should always be included
			configProfileForTest(t, "T6.1", "T6.1", "g"),
			configProfileForTest(t, "T6.2", "T6.2", "h"),

			// This profile will use an "include any" rule where the host has both of the labels, thus should be included
			configProfileForTest(t, "include_any_all_match_prof", "include_any_all_match_prof", "i", includeAnyMatchedLabel1, includeAnyMatchedLabel2),
			// This profile will use an "include any" rule where the host has one of the labels, thus should be included
			configProfileForTest(t, "include_any_one_matches_prof", "include_any_one_matches_prof", "j", includeAnyMatchedLabel1, includeAnyUnmatchedLabel),
			// This profile will use an "include any" rule where the host has none of the labels, thus should be excluded
			configProfileForTest(t, "include_any_none_match_prof", "include_any_none_match_prof", "k", includeAnyUnmatchedLabel),

			// This profile will use an "include all" rule where the host has all of the labels and thus should be included
			configProfileForTest(t, "include_all_all_match_prof", "include_all_all_match_prof", "l", includeAllMatchedLabel1, includeAllMatchedLabel2),
			// This profile will use an "include all" rule where the host has one of the labels and thus should be excluded
			configProfileForTest(t, "include_all_one_matches_prof", "include_all_one_matches_prof", "m", includeAllMatchedLabel1, includeAllUnmatchedLabel),
			// This profile will use an "include any" rule where the host has none of the labels and thus should be excluded
			configProfileForTest(t, "include_all_none_match_prof", "include_all_none_match_prof", "n", includeAllUnmatchedLabel),

			// This profile will use an "exclude" rule where the host has both of the labels, thus should be excluded
			configProfileForTest(t, "exclude_all_match_prof", "exclude_all_match_prof", "o", excludeMatchedLabel1, excludeMatchedLabel2),
			// This profile will use an "exclude" rule where the host has one of the labels, thus should be excluded
			configProfileForTest(t, "exclude_one_matches_prof", "exclude_one_matches_prof", "p", excludeMatchedLabel1, excludeUnmatchedLabel),
			// This profile will use an "exclude" rule where the host has none of the labels, thus should be included
			configProfileForTest(t, "exclude_none_match_prof", "exclude_none_match_prof", "q", excludeUnmatchedLabel),
		}

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, profiles, nil, nil, nil)
		require.NoError(t, err)
		assert.True(t, updates.AppleConfigProfile)
		assert.False(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		// Update labels with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					host.ID,
					includeAnyMatchedLabel1.ID,
					host.ID,
					includeAnyMatchedLabel2.ID,
					host.ID,
					includeAllMatchedLabel1.ID,
					host.ID,
					includeAllMatchedLabel2.ID,
					host.ID,
					excludeMatchedLabel1.ID,
					host.ID,
					excludeMatchedLabel2.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 11)

		return team.ID, host
	}

	// ===================================================
	// Windows
	// ===================================================

	windowsBasicTeamProfNoLabelsSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "windows team 1"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T1.1", "T1.1"),
			windowsConfigProfileForTest(t, "T1.2", "T1.2"),
		}

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles, nil, nil)
		require.NoError(t, err)
		assert.False(t, updates.AppleConfigProfile)
		assert.True(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 2)

		return team.ID, host
	}

	windowsLabeledTeamProfSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "windows team 2"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T2.1", "T2.1"),
			windowsConfigProfileForTest(t, "T2.2", "T2.2"),
			windowsConfigProfileForTest(t, "labeled_prof", "labeled_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: "test_label_6"})
		require.NoError(t, err)

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles, nil, nil)
		require.NoError(t, err)
		assert.False(t, updates.AppleConfigProfile)
		assert.True(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`,
				"labeled_prof",
			)
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

		return team.ID, host
	}

	windowsLabeledTeamProfWithAdditionalLabeledProfSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "windows team 3"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team 1
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T3.1", "T3.1"),
			windowsConfigProfileForTest(t, "T3.2", "T3.7"),
			windowsConfigProfileForTest(t, "labeled_prof_2", "labeled_prof_2"),
		}

		testLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		testLabel3, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles, nil, nil)
		require.NoError(t, err)
		assert.False(t, updates.AppleConfigProfile)
		assert.True(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`,
				"labeled_prof_2",
			)
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

		return team.ID, host
	}

	windowsProfWithBrokenLabelSetup := func() (uint, *fleet.Host) {
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
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "windows team 4"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// create profiles for team
		profiles := []*fleet.MDMWindowsConfigProfile{
			windowsConfigProfileForTest(t, "T4.1", "T4.1"),
			windowsConfigProfileForTest(t, "T4.2", "T4.2"),
			windowsConfigProfileForTest(t, "broken_label_prof", "broken_label_prof"),
		}

		label, err := ds.NewLabel(ctx, &fleet.Label{Name: uuid.NewString()})
		require.NoError(t, err)

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles, nil, nil)
		require.NoError(t, err)
		assert.False(t, updates.AppleConfigProfile)
		assert.True(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		var uid string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&uid,
				`SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE name = ?`,
				"broken_label_prof",
			)
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

		return team.ID, host
	}

	windowsLabeledProfileRulesSetup := func() (uint, *fleet.Host) {
		host, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:      "windows-test-5",
			OsqueryHostID: ptr.String("osquery-windows-5"),
			NodeKey:       ptr.String("node-key-windows-5"),
			UUID:          uuid.NewString(),
			Platform:      "windows",
		})
		require.NoError(t, err)
		windowsEnroll(t, ds, host)

		// create a team
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "windows team 5"})
		require.NoError(t, err)

		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
		require.NoError(t, err)

		// Include any labels
		includeAnyMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-windows-matched-label-1"})
		require.NoError(t, err)
		includeAnyMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-windows-matched-label-2"})
		require.NoError(t, err)
		includeAnyUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-windows-unmatched-label"})
		require.NoError(t, err)

		// Include all labels
		includeAllMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-windows-matched-label-1"})
		require.NoError(t, err)
		includeAllMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-windows-matched-label-2"})
		require.NoError(t, err)
		includeAllUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-all-windows-unmatched-label"})
		require.NoError(t, err)

		// Exclude labels
		excludeMatchedLabel1, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-windows-matched-label-1"})
		require.NoError(t, err)
		excludeMatchedLabel2, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-windows-matched-label-2"})
		require.NoError(t, err)
		excludeUnmatchedLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-windows-unmatched-label"})
		require.NoError(t, err)

		// create profiles for team
		// include_any_all_match_prof
		// include_any_one_matches_prof
		// include_all_all_match_prof
		// exclude_none_match_prof
		profiles := []*fleet.MDMWindowsConfigProfile{
			// Two profiles with no label rules, thus should always be included
			windowsConfigProfileForTest(t, "T5.1", "T5.1"),
			windowsConfigProfileForTest(t, "T5.2", "T5.2"),

			// This profile will use an "include any" rule where the host has both of the labels, thus should be included
			windowsConfigProfileForTest(t, "include_any_all_match_prof", "include_any_all_match_prof", includeAnyMatchedLabel1, includeAnyMatchedLabel2),
			// This profile will use an "include any" rule where the host has one of the labels, thus should be included
			windowsConfigProfileForTest(t, "include_any_one_matches_prof", "include_any_one_matches_prof", includeAnyMatchedLabel1, includeAnyUnmatchedLabel),
			// This profile will use an "include any" rule where the host has none of the labels, thus should be excluded
			windowsConfigProfileForTest(t, "include_any_none_match_prof", "include_any_none_match_prof", includeAnyUnmatchedLabel),

			// This profile will use an "include all" rule where the host has all of the labels and thus should be included
			windowsConfigProfileForTest(t, "include_all_all_match_prof", "include_all_all_match_prof", includeAllMatchedLabel1, includeAllMatchedLabel2),
			// This profile will use an "include all" rule where the host has one of the labels and thus should be excluded
			windowsConfigProfileForTest(t, "include_all_one_matches_prof", "include_all_one_matches_prof", includeAllMatchedLabel1, includeAllUnmatchedLabel),
			// This profile will use an "include any" rule where the host has none of the labels and thus should be excluded
			windowsConfigProfileForTest(t, "include_all_none_match_prof", "include_all_none_match_prof", includeAllUnmatchedLabel),

			// This profile will use an "exclude" rule where the host has both of the labels, thus should be excluded
			windowsConfigProfileForTest(t, "exclude_all_match_prof", "exclude_all_match_prof", excludeMatchedLabel1, excludeMatchedLabel2),
			// This profile will use an "exclude" rule where the host has one of the labels, thus should be excluded
			windowsConfigProfileForTest(t, "exclude_one_matches_prof", "exclude_one_matches_prof", excludeMatchedLabel1, excludeUnmatchedLabel),
			// This profile will use an "exclude" rule where the host has none of the labels, thus should be included
			windowsConfigProfileForTest(t, "exclude_none_match_prof", "exclude_none_match_prof", excludeUnmatchedLabel),
		}

		updates, err := ds.BatchSetMDMProfiles(ctx, &team.ID, nil, profiles, nil, nil)
		require.NoError(t, err)
		assert.False(t, updates.AppleConfigProfile)
		assert.True(t, updates.WindowsConfigProfile)
		assert.False(t, updates.AppleDeclaration)

		// Update labels with host membership
		ExecAdhocSQL(
			t, ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					host.ID,
					includeAnyMatchedLabel1.ID,
					host.ID,
					includeAnyMatchedLabel2.ID,
					host.ID,
					includeAllMatchedLabel1.ID,
					host.ID,
					includeAllMatchedLabel2.ID,
					host.ID,
					excludeMatchedLabel1.ID,
					host.ID,
					excludeMatchedLabel2.ID,
				)
				return err
			},
		)

		profs, _, err := ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, profs, 11)

		return team.ID, host
	}

	tests := []struct {
		name        string
		setupFunc   func() (uint, *fleet.Host)
		wantMac     map[string]*fleet.ExpectedMDMProfile
		wantWindows map[string]*fleet.ExpectedMDMProfile
		os          string
	}{
		{
			name:      "macos basic team profiles no labels",
			setupFunc: macosBasicTeamProfNoLabelsSetup,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T1.1": {Identifier: "T1.1"},
				"T1.2": {Identifier: "T1.2"},
			},
		},
		{
			name:      "macos labeled team profile",
			setupFunc: macosLabeledTeamProfSetup,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T2.1":         {Identifier: "T2.1"},
				"T2.2":         {Identifier: "T2.2"},
				"labeled_prof": {Identifier: "labeled_prof"},
			},
		},
		{
			name:      "macos labeled team profile with additional labeled profile",
			setupFunc: macosLabeledTeamProfWithAdditionalLabeledProfSetup,
			// Our expected profiles should not include the labeled profile, because it
			// maps to a label that is not applied to the host.
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T3.1": {Identifier: "T3.1"},
				"T3.2": {Identifier: "T3.2"},
			},
		},
		{
			name:      "macos profile with broken label",
			setupFunc: macosProfWithBrokenLabelSetup,
			// Our expected profiles should not include the labeled profile, because it is broken
			// (the label was deleted)
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T4.1": {Identifier: "T4.1"},
				"T4.2": {Identifier: "T4.2"},
			},
		},
		{
			name:      "macos basic team profiles no labels, install date overridden on first",
			setupFunc: macosInstallDateOverrideSetup,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T5.1": {Identifier: "T5.1", EarliestInstallDate: overrideEarliestInstallDate},
				"T5.2": {Identifier: "T5.2", EarliestInstallDate: baseEarliestInstallDate},
				"T5.3": {Identifier: "T5.3", EarliestInstallDate: baseEarliestInstallDate},
			},
		},
		{
			name:      "macos labels include any/all and exclude rules",
			setupFunc: macosLabeledProfileRulesSetup,
			wantMac: map[string]*fleet.ExpectedMDMProfile{
				"T6.1":                         {Identifier: "T6.1"},
				"T6.2":                         {Identifier: "T6.2"},
				"include_any_all_match_prof":   {Identifier: "include_any_all_match_prof"},
				"include_any_one_matches_prof": {Identifier: "include_any_one_matches_prof"},
				"include_all_all_match_prof":   {Identifier: "include_all_all_match_prof"},
				"exclude_none_match_prof":      {Identifier: "exclude_none_match_prof"},
			},
		},
		{
			name:      "windows basic team profiles no labels",
			setupFunc: windowsBasicTeamProfNoLabelsSetup,
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T1.1": {Name: "T1.1"},
				"T1.2": {Name: "T1.2"},
			},
		},
		{
			name:      "windows labeled team profile",
			setupFunc: windowsLabeledTeamProfSetup,
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T2.1":         {Name: "T2.1"},
				"T2.2":         {Name: "T2.2"},
				"labeled_prof": {Name: "labeled_prof"},
			},
		},
		{
			name:      "windows labeled team profile with additional labeled profile",
			setupFunc: windowsLabeledTeamProfWithAdditionalLabeledProfSetup,
			// Our expected profiles should not include the labeled profile, because it
			// maps to a label that is not applied to the host.
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T3.1": {Name: "T3.1"},
				"T3.2": {Name: "T3.2"},
			},
		},
		{
			name:      "windows profile with broken label",
			setupFunc: windowsProfWithBrokenLabelSetup,
			// Our expected profiles should not include the labeled profile, because it is broken
			// (the label was deleted)
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T4.1": {Name: "T4.1"},
				"T4.2": {Name: "T4.2"},
			},
		},
		{
			name:      "windows labels include any/all and exclude rules",
			setupFunc: windowsLabeledProfileRulesSetup,
			wantWindows: map[string]*fleet.ExpectedMDMProfile{
				"T5.1":                         {Name: "T5.1"},
				"T5.2":                         {Name: "T5.2"},
				"include_any_all_match_prof":   {Name: "include_any_all_match_prof"},
				"include_any_one_matches_prof": {Name: "include_any_one_matches_prof"},
				"include_all_all_match_prof":   {Name: "include_all_all_match_prof"},
				"exclude_none_match_prof":      {Name: "exclude_none_match_prof"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamID, host := tt.setupFunc()
			timeZero := time.Time{}
			if len(tt.wantMac) > 0 {
				got, err := ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID, host)
				require.NoError(t, err)
				for k, v := range tt.wantMac {
					require.Contains(t, got, k)
					require.Equal(t, v.Identifier, got[k].Identifier)
					// Only check earliest install date if we are overriding it in the test setup
					if v.EarliestInstallDate != timeZero {
						require.Equal(t, v.EarliestInstallDate, got[k].EarliestInstallDate)
					}
				}
			}

			if len(tt.wantWindows) > 0 {
				got, err := ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID, host.ID)
				require.NoError(t, err)
				for k, v := range tt.wantWindows {
					require.Contains(t, got, k)
					require.Equal(t, v.Name, got[k].Name)
					// windows does not currently use or care about earliest install date
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
		nil,
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
		nil,
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
	updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, ds.writer(ctx), wantOtherWin, []string{windowsProfile.ProfileUUID}, "windows")
	require.NoError(t, err)
	assert.True(t, updatedDB)
	// make it an "exclude" label on the other macos profile
	wantOtherMac := []fleet.ConfigurationProfileLabel{
		{ProfileUUID: otherMacProfile.ProfileUUID, LabelName: label.Name, LabelID: label.ID, Exclude: true},
	}
	updatedDB, err = batchSetProfileLabelAssociationsDB(ctx, ds.writer(ctx), wantOtherMac, []string{macOSProfile.ProfileUUID}, "darwin")
	require.NoError(t, err)
	assert.True(t, updatedDB)

	platforms := map[string]string{
		"darwin":  macOSProfile.ProfileUUID,
		"windows": windowsProfile.ProfileUUID,
	}

	for platform, uuid := range platforms {
		expectLabels := func(t *testing.T, profUUID, platform string, want []fleet.ConfigurationProfileLabel) {
			p := platform
			if p == "darwin" {
				p = "apple"
			}

			query := fmt.Sprintf(
				"SELECT %s_profile_uuid as profile_uuid, label_id, label_name, exclude FROM mdm_configuration_profile_labels WHERE %s_profile_uuid = ?",
				p,
				p,
			)

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
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, want, nil, platform)
				require.NoError(t, err)
				assert.False(t, updatedDB)
				return err
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
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, nil, platform)
				require.NoError(t, err)
				assert.True(t, updatedDB)
				return err
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, profileLabels)
			// does not change other profiles
			expectLabels(t, otherWinProfile.ProfileUUID, "windows", wantOtherWin)
			expectLabels(t, otherMacProfile.ProfileUUID, "darwin", wantOtherMac)

			// now set it with Exclude mode
			profileLabels = []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID, Exclude: true},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, nil, platform)
				require.NoError(t, err)
				assert.True(t, updatedDB)
				return err
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
				_, err := batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, nil, platform)
				return err
			})
			require.Error(t, err)
		})

		t.Run("invalid label data "+platform, func(t *testing.T) {
			// invalid id
			invalidProfileLabels := []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: 12345},
			}
			err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				_, err := batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, nil, platform)
				return err
			})
			require.Error(t, err)

			// both invalid
			invalidProfileLabels = []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: "xyz", LabelID: 1235},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				_, err := batchSetProfileLabelAssociationsDB(ctx, tx, invalidProfileLabels, nil, platform)
				return err
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
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID, Exclude: true},
				{ProfileUUID: uuid, LabelName: newLabel.Name, LabelID: newLabel.ID, Exclude: true},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, nil, platform)
				require.NoError(t, err)
				assert.True(t, updatedDB)
				return err
			})
			require.NoError(t, err)
			// both are stored in the DB
			expectLabels(t, uuid, platform, profileLabels)

			// batch apply again without the newLabel, and without Exclude flag
			profileLabels = []fleet.ConfigurationProfileLabel{
				{ProfileUUID: uuid, LabelName: label.Name, LabelID: label.ID},
			}
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, profileLabels, nil, platform)
				require.NoError(t, err)
				assert.True(t, updatedDB)
				return err
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, profileLabels)

			// batch apply again this time without any label
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, nil, []string{uuid}, platform)
				require.NoError(t, err)
				assert.True(t, updatedDB)
				return err
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, nil)

			// does not change other profiles
			expectLabels(t, otherWinProfile.ProfileUUID, "windows", wantOtherWin)
			expectLabels(t, otherMacProfile.ProfileUUID, "darwin", wantOtherMac)

			// batch apply again with no change returns false
			err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
				updatedDB, err := batchSetProfileLabelAssociationsDB(ctx, tx, nil, []string{uuid}, platform)
				require.NoError(t, err)
				assert.False(t, updatedDB)
				return err
			})
			require.NoError(t, err)
			expectLabels(t, uuid, platform, nil)
		})
	}

	t.Run("unsupported platform", func(t *testing.T) {
		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := batchSetProfileLabelAssociationsDB(
				ctx,
				tx,
				[]fleet.ConfigurationProfileLabel{{}},
				nil,
				"unsupported",
			)
			return err
		})
		require.Error(t, err)
	})
}

// This test was added to check for this bug report:
// https://github.com/fleetdm/fleet/issues/16273 I believe it turned out there
// was no transaction issue (see
// https://github.com/fleetdm/fleet/pull/16449#discussion_r1471486335), so we
// could eventually remove the testBatchSetMDMAppleProfilesErr and similar
// fields from the datastore that add a lot of noise in batch-set apple
// profiles.
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
		{
			"labels:e",
			"",
			"batch set windows profiles: inserting windows profile label associations: labels:e",
		},
		{
			"inselect:k",
			"",
			"batch set windows profiles: build query to load existing profiles: inselect:k",
		},
		{
			"indelete:l",
			"",
			"batch set windows profiles: build statement to delete obsolete profiles: indelete:l",
		},
		{
			"inreselect:m",
			"",
			"batch set windows profiles: build query to load newly inserted profiles: inreselect:m",
		},
		{"", "select:f", "batch set apple profiles: load existing profiles: select:f"},
		{"", "insert:g", ": insert:g"},
		{"", "delete:h", "batch set apple profiles: delete obsolete profiles: delete:h"},
		{"", "reselect:i", "batch set apple profiles: load newly inserted profiles: reselect:i"},
		{
			"",
			"labels:j",
			"batch set apple profiles: inserting apple profile label associations: labels:j",
		},
		{
			"",
			"inselect:n",
			"batch set apple profiles: build query to load existing profiles: inselect:n",
		},
		{
			"",
			"indelete:o",
			"batch set apple profiles: build statement to delete obsolete profiles: indelete:o",
		},
		{
			"",
			"inreselect:p",
			"batch set apple profiles: build query to load newly inserted profiles: inreselect:p",
		},
	}
	for _, c := range cases {
		t.Run(c.windowsErr+" "+c.appleErr, func(t *testing.T) {
			t.Cleanup(func() {
				ds.testBatchSetMDMAppleProfilesErr = ""
				ds.testBatchSetMDMWindowsProfilesErr = ""
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
			_, err := ds.BatchSetMDMProfiles(ctx, nil, appleProfs, winProfs, nil, nil)
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
			ds.testBatchSetMDMAppleProfilesErr = c.appleErr
			ds.testBatchSetMDMWindowsProfilesErr = c.windowsErr

			_, err = ds.BatchSetMDMProfiles(ctx, nil, appleProfs, winProfs, nil, nil)
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func testMDMEULA(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	eula := &fleet.MDMEULA{
		Token:  uuid.New().String(),
		Name:   "eula.pdf",
		Bytes:  []byte("contents"),
		Sha256: []byte("test-sha256"),
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
	scepDepot, err := ds.NewSCEPDepot()
	require.NoError(t, err)

	nanoStorage, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	addCert := func(notAfter time.Time, h *fleet.Host) {
		serial, err := scepDepot.Serial()
		require.NoError(t, err)
		cert := &x509.Certificate{
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName: "Fleet Identity",
			},
			NotAfter: notAfter,
			// use a random value, just to make sure they're
			// different from each other, we don't care about the
			// DER contents here
			Raw: []byte(uuid.NewString()),
		}
		err = scepDepot.Put(cert.Subject.CommonName, cert)
		require.NoError(t, err)
		req := mdm.Request{
			EnrollID: &mdm.EnrollID{ID: h.UUID},
			Context:  ctx,
		}
		certHash := certauth.HashCert(cert)
		err = nanoStorage.AssociateCertHash(&req, certHash, notAfter)
		require.NoError(t, err)
		createdat := time.Now().AddDate(0, int(serial.Int64()), 0)

		// due to mysql timestamp resolution, this test is flaky unless
		// we do this because we insert multiple certificates for the
		// same device in quick succession, and later on we assert on a
		// specific order which is based on created_at
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
                  UPDATE nano_cert_auth_associations
                  SET created_at = ?
                  WHERE sha256 = ?
		`, createdat, certHash)
			return err
		})
	}

	var i int
	setHost := func(notAfter time.Time, doUserDeviceEnrollment bool) *fleet.Host {
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
		addCert(notAfter, h)
		if doUserDeviceEnrollment {
			nanoEnrollUserDevice(t, ds, h)
		} else {
			nanoEnroll(t, ds, h, true)
		}
		return h
	}

	// certs expired at lest 1 year ago
	h1 := setHost(time.Now().AddDate(-1, -3, 0), false)
	h2 := setHost(time.Now().AddDate(-1, -2, 0), false)
	// cert that expires in 1 month
	h3 := setHost(time.Now().AddDate(0, 1, 0), false)
	// User Enrollment (Device) cert that expires in 1 month and 1 day just
	// so we can add some assertions on the returned enrollment type
	h4 := setHost(time.Now().AddDate(0, 1, 1), true)
	// cert that expires in 1 year
	h5 := setHost(time.Now().AddDate(1, 0, 0), false)
	// expired cert for a host migrated using touchless migration
	hMigrated := setHost(time.Now().AddDate(-1, -1, 0), false)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
                  UPDATE nano_enrollments
                  SET enrolled_from_migration = 1
                  WHERE id = ?
		`, hMigrated.UUID)
		return err
	})

	// list assocs that expire in the next 10 days
	assocs, err := ds.GetHostCertAssociationsToExpire(ctx, 10, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 3)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	require.Equal(t, hMigrated.UUID, assocs[2].HostUUID)
	require.True(t, assocs[2].EnrolledFromMigration)

	// list certs that expire in the next 1000 days with limit = 1
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 1)
	require.NoError(t, err)
	require.Len(t, assocs, 1)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)

	// list certs that expire in the next 50 days
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 50, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 5)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	assert.Equal(t, "Device", assocs[0].EnrollmentType)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	assert.Equal(t, "Device", assocs[1].EnrollmentType)
	require.Equal(t, hMigrated.UUID, assocs[2].HostUUID)
	assert.Equal(t, "Device", assocs[2].EnrollmentType)
	require.Equal(t, h3.UUID, assocs[3].HostUUID)
	assert.Equal(t, "Device", assocs[3].EnrollmentType)
	require.Equal(t, h4.UUID, assocs[4].HostUUID)
	assert.Equal(t, "User Enrollment (Device)", assocs[4].EnrollmentType)

	// list certs that expire in the next 1000 days
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 6)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	assert.Equal(t, "Device", assocs[0].EnrollmentType)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	assert.Equal(t, "Device", assocs[1].EnrollmentType)
	require.Equal(t, hMigrated.UUID, assocs[2].HostUUID)
	assert.Equal(t, "Device", assocs[2].EnrollmentType)
	require.Equal(t, h3.UUID, assocs[3].HostUUID)
	assert.Equal(t, "Device", assocs[3].EnrollmentType)
	require.Equal(t, h4.UUID, assocs[4].HostUUID)
	assert.Equal(t, "User Enrollment (Device)", assocs[4].EnrollmentType)
	require.Equal(t, h5.UUID, assocs[5].HostUUID)
	assert.Equal(t, "Device", assocs[5].EnrollmentType)

	// add a new host with a very old expiriy so it shows first, verify
	// that it's present before deleting it.
	h6 := setHost(time.Now().AddDate(-2, -1, 0), false)
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 7)
	require.Equal(t, h6.UUID, assocs[0].HostUUID)
	require.Equal(t, h1.UUID, assocs[1].HostUUID)
	require.Equal(t, h2.UUID, assocs[2].HostUUID)
	require.Equal(t, hMigrated.UUID, assocs[3].HostUUID)
	require.Equal(t, h3.UUID, assocs[4].HostUUID)
	require.Equal(t, h4.UUID, assocs[5].HostUUID)
	require.Equal(t, h5.UUID, assocs[6].HostUUID)

	// delete the host and verify that things work as expected
	// see https://github.com/fleetdm/fleet/issues/19149
	require.NoError(t, ds.DeleteHost(ctx, h6.ID))
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 100)
	require.NoError(t, err)
	require.Len(t, assocs, 6)
	require.Equal(t, h1.UUID, assocs[0].HostUUID)
	require.Equal(t, h2.UUID, assocs[1].HostUUID)
	require.Equal(t, hMigrated.UUID, assocs[2].HostUUID)
	require.Equal(t, h3.UUID, assocs[3].HostUUID)
	require.Equal(t, h4.UUID, assocs[4].HostUUID)
	require.Equal(t, h5.UUID, assocs[5].HostUUID)

	// add a second expired cert to one of the hosts
	addCert(time.Now().AddDate(-1, 0, 0), h1)
	assocs, err = ds.GetHostCertAssociationsToExpire(ctx, 1000, 100)
	require.Len(t, assocs, 6)
	require.Equal(t, h2.UUID, assocs[0].HostUUID)
	require.Equal(t, hMigrated.UUID, assocs[1].HostUUID)
	require.Equal(t, h1.UUID, assocs[2].HostUUID)
	require.Equal(t, h3.UUID, assocs[3].HostUUID)
	require.Equal(t, h4.UUID, assocs[4].HostUUID)
	require.Equal(t, h5.UUID, assocs[5].HostUUID)

	checkSCEPRenew := func(assoc fleet.SCEPIdentityAssociation, want *string) {
		var got *string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(
				ctx,
				q,
				&got,
				`SELECT renew_command_uuid FROM nano_cert_auth_associations WHERE id = ? AND sha256 = ?`,
				assoc.HostUUID,
				assoc.SHA256,
			)
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
	checkSCEPRenew(assocs[4], nil)
	checkSCEPRenew(assocs[5], nil)
	require.NoError(t, err)

	err = ds.SetCommandForPendingSCEPRenewal(ctx, []fleet.SCEPIdentityAssociation{assocs[0]}, "foo")
	require.NoError(t, err)
	checkSCEPRenew(assocs[0], ptr.String("foo"))
	checkSCEPRenew(assocs[1], nil)
	checkSCEPRenew(assocs[2], nil)
	checkSCEPRenew(assocs[3], nil)
	checkSCEPRenew(assocs[4], nil)
	checkSCEPRenew(assocs[5], nil)

	err = ds.SetCommandForPendingSCEPRenewal(ctx, assocs, "bar")
	require.NoError(t, err)
	checkSCEPRenew(assocs[0], ptr.String("bar"))
	checkSCEPRenew(assocs[1], ptr.String("bar"))
	checkSCEPRenew(assocs[2], ptr.String("bar"))
	checkSCEPRenew(assocs[3], ptr.String("bar"))
	checkSCEPRenew(assocs[4], ptr.String("bar"))
	checkSCEPRenew(assocs[5], ptr.String("bar"))

	err = ds.SetCommandForPendingSCEPRenewal(
		ctx,
		[]fleet.SCEPIdentityAssociation{{HostUUID: "foo", SHA256: "bar"}},
		"bar",
	)
	require.ErrorContains(t, err, "this function can only be used to update existing associations")

	err = ds.CleanSCEPRenewRefs(ctx, "does-not-exist")
	require.Error(t, err)

	err = ds.CleanSCEPRenewRefs(ctx, h1.UUID)
	require.NoError(t, err)
	checkSCEPRenew(assocs[2], nil)
}

func testMDMProfilesSummaryAndHostFilters(t *testing.T, ds *Datastore) {
	// TODO: Expand this test to include:
	// - more scenarios for windows
	// - disk encryption (mac and windows)
	// - more scenarios for labels

	ctx := context.Background()

	checkSummaryWindows := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMWindowsProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
	}

	checkSummaryMac := func(t *testing.T, teamID *uint, expected fleet.MDMProfilesSummary) {
		ps, err := ds.GetMDMAppleProfilesSummary(ctx, teamID)
		require.NoError(t, err)
		require.NotNil(t, ps)
		require.Equal(t, expected, *ps)
	}

	checkListHostsFilterOSSettings := func(t *testing.T, teamID *uint, status fleet.OSSettingsStatus, expectedIDs []uint) {
		gotHosts, err := ds.ListHosts(
			ctx,
			fleet.TeamFilter{User: test.UserAdmin},
			fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status},
		)
		require.NoError(t, err)
		if len(expectedIDs) != len(gotHosts) {
			gotIDs := make([]uint, len(gotHosts))
			for _, h := range gotHosts {
				gotIDs = append(gotIDs, h.ID)
			}
			require.Len(
				t,
				gotHosts,
				len(expectedIDs),
				fmt.Sprintf("status: %s expected: %v got: %v", status, expectedIDs, gotIDs),
			)

		}
		for _, h := range gotHosts {
			require.Contains(t, expectedIDs, h.ID)
		}

		count, err := ds.CountHosts(
			ctx,
			fleet.TeamFilter{User: test.UserAdmin},
			fleet.HostListOptions{TeamFilter: teamID, OSSettingsFilter: status},
		)
		require.NoError(t, err)
		require.Equal(t, len(expectedIDs), count, "status: %s", status)
	}

	type hostIDsByProfileStatus map[fleet.MDMDeliveryStatus][]uint

	checkExpected := func(t *testing.T, teamID *uint, ep hostIDsByProfileStatus) {
		expectSummaryWindows := map[fleet.MDMDeliveryStatus]uint{}
		expectSummaryMac := map[fleet.MDMDeliveryStatus]uint{}
		for status, ids := range ep {
			if len(ids) > 0 {
				for _, id := range ids {
					if id < 5 {
						expectSummaryWindows[status]++
					} else {
						expectSummaryMac[status]++
					}
				}
			}
		}
		checkSummaryMac(t, teamID, fleet.MDMProfilesSummary{
			Pending:   expectSummaryMac[fleet.MDMDeliveryPending],
			Failed:    expectSummaryMac[fleet.MDMDeliveryFailed],
			Verifying: expectSummaryMac[fleet.MDMDeliveryVerifying],
			Verified:  expectSummaryMac[fleet.MDMDeliveryVerified],
		})

		checkSummaryWindows(t, teamID, fleet.MDMProfilesSummary{
			Pending:   expectSummaryWindows[fleet.MDMDeliveryPending],
			Failed:    expectSummaryWindows[fleet.MDMDeliveryFailed],
			Verifying: expectSummaryWindows[fleet.MDMDeliveryVerifying],
			Verified:  expectSummaryWindows[fleet.MDMDeliveryVerified],
		})

		checkListHostsFilterOSSettings(
			t,
			teamID,
			fleet.OSSettingsVerified,
			ep[fleet.MDMDeliveryVerified],
		)
		checkListHostsFilterOSSettings(
			t,
			teamID,
			fleet.OSSettingsVerifying,
			ep[fleet.MDMDeliveryVerifying],
		)
		checkListHostsFilterOSSettings(
			t,
			teamID,
			fleet.OSSettingsFailed,
			ep[fleet.MDMDeliveryFailed],
		)
		checkListHostsFilterOSSettings(
			t,
			teamID,
			fleet.OSSettingsPending,
			ep[fleet.MDMDeliveryPending],
		)
	}

	// checkWinHostProfiles := func(t *testing.T, hostUUID string, statusByProfUUID map[string]string) {
	// 	profs, err := ds.GetHostMDMWindowsProfiles(ctx, hostUUID)
	// 	require.NoError(t, err)
	// 	require.Len(t, profs, len(statusByProfUUID))
	// 	for _, prof := range profs {
	// 		ep, ok := statusByProfUUID[prof.ProfileUUID]
	// 		require.True(t, ok)
	// 		require.Equal(t, ep, prof.Status)
	// 	}
	// }

	checkMacHostProfiles := func(t *testing.T, hostUUID string, statusByProfUUID map[string]string) {
		profs, err := ds.GetHostMDMAppleProfiles(ctx, hostUUID)
		require.NoError(t, err)
		require.Len(t, profs, len(statusByProfUUID))
		for _, prof := range profs {
			ep, ok := statusByProfUUID[prof.ProfileUUID]
			require.True(t, ok)
			require.NotNil(t, prof.Status)
			require.Equal(t, fleet.MDMDeliveryStatus(ep), *prof.Status)
		}
	}

	upsertHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, status *fleet.MDMDeliveryStatus) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			var table string
			var profType string
			switch {
			case strings.HasPrefix(profUUID, "a"):
				table = "host_mdm_apple_profiles"
				profType = "profile"
			case strings.HasPrefix(profUUID, "w"):
				table = "host_mdm_windows_profiles"
				profType = "profile"
			case strings.HasPrefix(profUUID, "d"):
				table = "host_mdm_apple_declarations"
				profType = "declaration"
			default:
				require.FailNow(t, "unknown profile type")
			}
			stmt := fmt.Sprintf(
				`INSERT INTO %s (host_uuid, %s_uuid, status) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE status = ?`,
				table,
				profType,
			)
			_, err := q.ExecContext(ctx, stmt, hostUUID, profUUID, status, status)
			if err != nil {
				require.NoError(t, err)
				return err
			}
			stmt = fmt.Sprintf(
				`UPDATE %s SET operation_type = ? WHERE host_uuid = ? AND %s_uuid = ?`,
				table,
				profType,
			)
			_, err = q.ExecContext(ctx, stmt, fleet.MDMOperationTypeInstall, hostUUID, profUUID)
			require.NoError(t, err)
			return err
		})
	}

	cleanupTables := func(t *testing.T) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_windows_profiles`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_apple_profiles`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_apple_declarations`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_disk_encryption_keys`)
			return err
		})
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_disks`)
			return err
		})
	}

	// updateHostDisks := func(t *testing.T, hostID uint, encrypted bool, updated_at time.Time) {
	// 	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
	// 		stmt := `UPDATE host_disks SET encrypted = ?, updated_at = ? where host_id = ?`
	// 		_, err := q.ExecContext(ctx, stmt, encrypted, updated_at, hostID)
	// 		return err
	// 	})
	// }

	// Create some hosts
	var hosts []*fleet.Host
	macHostsByID := make(map[uint]*fleet.Host, 5)
	winHostsByID := make(map[uint]*fleet.Host, 5)
	for i := 0; i < 10; i++ {
		p := "windows"
		if i >= 5 {
			p = "darwin"
		}
		u := uuid.New().String()
		h, err := ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         &u,
			UUID:            u,
			Hostname:        u,
			Platform:        p,
		})
		require.NoError(t, err)
		require.NotNil(t, h)
		hosts = append(hosts, h)
		if p == "darwin" {
			nanoEnroll(t, ds, h, false)
			macHostsByID[h.ID] = h
		} else {
			winHostsByID[h.ID] = h
			windowsEnrollment := &fleet.MDMWindowsEnrolledDevice{
				MDMDeviceID:            uuid.New().String(),
				MDMHardwareID:          uuid.New().String() + uuid.New().String(),
				MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
				MDMDeviceType:          "CIMClient_Windows",
				MDMDeviceName:          "DESKTOP-1C3ARC1",
				MDMEnrollType:          "ProgrammaticEnrollment",
				MDMEnrollUserID:        "",
				MDMEnrollProtoVersion:  "5.0",
				MDMEnrollClientVersion: "10.0.19045.2965",
				MDMNotInOOBE:           false,
				HostUUID:               h.UUID,
			}
			err = ds.MDMWindowsInsertEnrolledDevice(ctx, windowsEnrollment)
			require.NoError(t, err)
		}

		require.NoError(
			t,
			ds.SetOrUpdateMDMData(
				ctx,
				h.ID,
				false,
				true,
				"https://example.com",
				false,
				fleet.WellKnownMDMFleet,
				"",
				false,
			),
		)
	}

	checkExpected(t, nil, nil)

	upsertHostProfileStatus(t, hosts[0].UUID, "w1", &fleet.MDMDeliveryPending)

	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
	})

	// add some mac profiles with different statuses
	upsertHostProfileStatus(t, hosts[9].UUID, "a1", &fleet.MDMDeliveryFailed)
	upsertHostProfileStatus(t, hosts[9].UUID, "a2", &fleet.MDMDeliveryPending)
	upsertHostProfileStatus(t, hosts[9].UUID, "a3", &fleet.MDMDeliveryVerifying)
	upsertHostProfileStatus(t, hosts[9].UUID, "a4", &fleet.MDMDeliveryVerified)

	// add some mac declarations with different statuses
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", &fleet.MDMDeliveryFailed)
	upsertHostProfileStatus(t, hosts[9].UUID, "d2", &fleet.MDMDeliveryPending)
	upsertHostProfileStatus(t, hosts[9].UUID, "d3", &fleet.MDMDeliveryVerifying)
	upsertHostProfileStatus(t, hosts[9].UUID, "d4", &fleet.MDMDeliveryVerified)

	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
		fleet.MDMDeliveryFailed:  []uint{hosts[9].ID},
	})
	expectedHostProfiles := map[string]string{
		"a1": "failed",
		"a2": "pending",
		"a3": "verifying",
		"a4": "verified",
		"d1": "failed",
		"d2": "pending",
		"d3": "verifying",
		"d4": "verified",
	}
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set failed mac profile to pending, still failed because of failed declaration
	upsertHostProfileStatus(t, hosts[9].UUID, "a1", &fleet.MDMDeliveryPending)
	expectedHostProfiles["a1"] = "pending"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
		fleet.MDMDeliveryFailed:  []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set failed mac declaration to pending, now host stsatus is pending
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", &fleet.MDMDeliveryPending)
	expectedHostProfiles["d1"] = "pending"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set pending mac declaration to failed, host status is now failed
	upsertHostProfileStatus(t, hosts[9].UUID, "d2", &fleet.MDMDeliveryFailed)
	expectedHostProfiles["d2"] = "failed"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
		fleet.MDMDeliveryFailed:  []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set failed mac declaration to verifying, host status is now pending
	upsertHostProfileStatus(t, hosts[9].UUID, "d2", &fleet.MDMDeliveryVerifying)
	expectedHostProfiles["d2"] = "verifying"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set pending mac profiles to verifying, host status is still pending because d1 is still pending
	upsertHostProfileStatus(t, hosts[9].UUID, "a1", &fleet.MDMDeliveryVerifying)
	expectedHostProfiles["a1"] = "verifying"
	upsertHostProfileStatus(t, hosts[9].UUID, "a2", &fleet.MDMDeliveryVerifying)
	expectedHostProfiles["a2"] = "verifying"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set pending mac declarations to verifying, host status is now verifying
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", &fleet.MDMDeliveryVerifying)
	expectedHostProfiles["d1"] = "verifying"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending:   []uint{hosts[0].ID},
		fleet.MDMDeliveryVerifying: []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set a mac profile to failed, host status is now failed
	upsertHostProfileStatus(t, hosts[9].UUID, "a1", &fleet.MDMDeliveryFailed)
	expectedHostProfiles["a1"] = "failed"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
		fleet.MDMDeliveryFailed:  []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set mac profiles to verified, host status is now verifying because declarations are still
	// verifying
	upsertHostProfileStatus(t, hosts[9].UUID, "a1", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["a1"] = "verified"
	upsertHostProfileStatus(t, hosts[9].UUID, "a2", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["a2"] = "verified"
	upsertHostProfileStatus(t, hosts[9].UUID, "a3", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["a3"] = "verified"
	upsertHostProfileStatus(t, hosts[9].UUID, "a4", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["a4"] = "verified"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending:   []uint{hosts[0].ID},
		fleet.MDMDeliveryVerifying: []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set mac declarations to verified, host status is now verified
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["d1"] = "verified"
	upsertHostProfileStatus(t, hosts[9].UUID, "d2", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["d2"] = "verified"
	upsertHostProfileStatus(t, hosts[9].UUID, "d3", &fleet.MDMDeliveryVerified)
	expectedHostProfiles["d3"] = "verified"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending:  []uint{hosts[0].ID},
		fleet.MDMDeliveryVerified: []uint{hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// set a mac declaration to nil, host status is now pending
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", nil)
	expectedHostProfiles["d1"] = "pending"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// works as expected if we remove mac declarations
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_apple_declarations`)
		return err
	})
	delete(expectedHostProfiles, "d1")
	delete(expectedHostProfiles, "d2")
	delete(expectedHostProfiles, "d3")
	delete(expectedHostProfiles, "d4")
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending:  []uint{hosts[0].ID},
		fleet.MDMDeliveryVerified: []uint{hosts[9].ID}, // all profiles were verified
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// works as expected if we remove mac profiles
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_apple_profiles`)
		return err
	})
	delete(expectedHostProfiles, "a1")
	delete(expectedHostProfiles, "a2")
	delete(expectedHostProfiles, "a3")
	delete(expectedHostProfiles, "a4")
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	// works as expected if declarations but no profiles
	upsertHostProfileStatus(t, hosts[9].UUID, "d1", &fleet.MDMDeliveryPending)
	expectedHostProfiles["d1"] = "pending"
	checkExpected(t, nil, hostIDsByProfileStatus{
		fleet.MDMDeliveryPending: []uint{hosts[0].ID, hosts[9].ID},
	})
	checkMacHostProfiles(t, hosts[9].UUID, expectedHostProfiles)

	cleanupTables(t)
}

func testAreHostsConnectedToFleetMDM(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	notConnectedMac, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test",
		OsqueryHostID: ptr.String("osquery-macos-not-connected"),
		NodeKey:       ptr.String("node-key-macos-not-connected"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	connectedMac, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test",
		OsqueryHostID: ptr.String("osquery-macos"),
		NodeKey:       ptr.String("node-key-macos"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, connectedMac, false)
	err = ds.SetOrUpdateMDMData(ctx, connectedMac.ID, false, true, "http://foo.com", false, "foo", "", false)
	require.NoError(t, err)

	disconnectedWithoutCheckoutMac, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test-disconnected",
		OsqueryHostID: ptr.String("osquery-macos-disconnected"),
		NodeKey:       ptr.String("node-key-macos-disconnected"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, disconnectedWithoutCheckoutMac, false)
	err = ds.SetOrUpdateMDMData(ctx, disconnectedWithoutCheckoutMac.ID, false, false, "", false, "", "", false)
	require.NoError(t, err)

	notConnectedWin, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "windows-test",
		OsqueryHostID: ptr.String("osquery-windows-not-connected"),
		NodeKey:       ptr.String("node-key-windows-not-connected"),
		UUID:          uuid.NewString(),
		Platform:      "windows",
	})
	require.NoError(t, err)

	connectedWin, err := ds.NewHost(ctx, &fleet.Host{
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
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               connectedWin.UUID,
	}
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, windowsEnrollment)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, connectedWin.ID, false, true, "http://foo.com", false, "foo", "", false)
	require.NoError(t, err)

	disconnectedWithoutCheckoutWin, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "windows-test-disconnected",
		OsqueryHostID: ptr.String("osquery-windows-disconnected"),
		NodeKey:       ptr.String("node-key-windows-disconnected"),
		UUID:          uuid.NewString(),
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnrollmentDisconnectedWithoutCheckout := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1-disconnected",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               disconnectedWithoutCheckoutWin.UUID,
	}
	err = ds.MDMWindowsInsertEnrolledDevice(ctx, windowsEnrollmentDisconnectedWithoutCheckout)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, disconnectedWithoutCheckoutWin.ID, false, false, "", false, "", "", false)
	require.NoError(t, err)

	connectedMap, err := ds.AreHostsConnectedToFleetMDM(ctx, []*fleet.Host{
		notConnectedMac,
		connectedMac,
		connectedWin,
		notConnectedWin,
		disconnectedWithoutCheckoutMac,
		disconnectedWithoutCheckoutWin,
	})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{
		notConnectedMac.UUID:                false,
		connectedMac.UUID:                   true,
		connectedWin.UUID:                   true,
		notConnectedWin.UUID:                false,
		disconnectedWithoutCheckoutMac.UUID: false,
		disconnectedWithoutCheckoutWin.UUID: false,
	}, connectedMap)

	linuxHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "linux-test",
		OsqueryHostID: ptr.String("osquery-linux"),
		NodeKey:       ptr.String("node-key-linux"),
		UUID:          uuid.NewString(),
		Platform:      "linux",
	})
	require.NoError(t, err)
	connectedMap, err = ds.AreHostsConnectedToFleetMDM(ctx, []*fleet.Host{
		notConnectedMac,
		connectedMac,
		connectedWin,
		notConnectedWin,
		linuxHost,
		disconnectedWithoutCheckoutMac,
		disconnectedWithoutCheckoutWin,
	})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{
		notConnectedMac.UUID:                false,
		connectedMac.UUID:                   true,
		connectedWin.UUID:                   true,
		notConnectedWin.UUID:                false,
		linuxHost.UUID:                      false,
		disconnectedWithoutCheckoutMac.UUID: false,
		disconnectedWithoutCheckoutWin.UUID: false,
	}, connectedMap)
}

func testIsHostConnectedToFleetMDM(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	macH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test",
		OsqueryHostID: ptr.String("osquery-macos"),
		NodeKey:       ptr.String("node-key-macos"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	connected, err := ds.IsHostConnectedToFleetMDM(ctx, macH)
	require.NoError(t, err)
	require.False(t, connected)

	nanoEnroll(t, ds, macH, false)
	err = ds.SetOrUpdateMDMData(ctx, macH.ID, false, true, "http://foo.com", false, "foo", "", false)
	require.NoError(t, err)

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, macH)
	require.NoError(t, err)
	require.True(t, connected)

	byodIpadH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "ipados-test",
		OsqueryHostID: ptr.String("osquery-ipados"),
		NodeKey:       ptr.String("node-key-ipados"),
		UUID:          uuid.NewString(),
		Platform:      "ipados",
	})
	require.NoError(t, err)

	nanoEnrollUserDevice(t, ds, byodIpadH)
	err = ds.SetOrUpdateMDMData(ctx, byodIpadH.ID, false, true, "http://foo.com", false, "foo", "", false)
	require.NoError(t, err)

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, byodIpadH)
	require.NoError(t, err)
	require.True(t, connected)

	windowsH, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "windows-test",
		OsqueryHostID: ptr.String("osquery-windows"),
		NodeKey:       ptr.String("node-key-windows"),
		UUID:          uuid.NewString(),
		Platform:      "windows",
	})
	require.NoError(t, err)
	connected, err = ds.IsHostConnectedToFleetMDM(ctx, windowsH)
	require.NoError(t, err)
	require.False(t, connected)

	windowsEnrollment := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            uuid.New().String(),
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
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
	err = ds.SetOrUpdateMDMData(ctx, windowsH.ID, false, true, "http://foo.com", false, "foo", "", false)
	require.NoError(t, err)

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, windowsH)
	require.NoError(t, err)
	require.True(t, connected)

	// now simulate an un-enrollment without checkout, in this case, osquery reports the host as not-enrolled
	err = ds.SetOrUpdateMDMData(ctx, macH.ID, false, false, "", false, "", "", false)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(ctx, windowsH.ID, false, false, "", false, "", "", false)
	require.NoError(t, err)

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, macH)
	require.NoError(t, err)
	require.False(t, connected)

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, windowsH)
	require.NoError(t, err)
	require.False(t, connected)

	// Simulate the ipad checking out(user removing work account)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE nano_enrollments SET enabled = 0 WHERE id = ?`, byodIpadH.UUID)
		return err
	})

	connected, err = ds.IsHostConnectedToFleetMDM(ctx, byodIpadH)
	require.NoError(t, err)
	require.False(t, connected)
}

func testBulkSetPendingMDMHostProfilesExcludeAny(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create some "exclude" labels
	var labels []*fleet.Label
	for i := 0; i < 6; i++ {
		lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-label-" + strconv.Itoa(i), Query: "select 1"})
		require.NoError(t, err)
		labels = append(labels, lbl)
	}

	// create an Apple profile, a Windows profile and an Apple Declaration with excluded labels
	appleProfs := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "A1", "A1", uuid.NewString(), labels[0], labels[1]),
	}
	windowsProfs := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "W1", "W1", labels[2]),
	}
	appleDecls := []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "{}", labels[3], labels[4], labels[5]),
	}

	updates, err := ds.BatchSetMDMProfiles(ctx, nil, appleProfs, windowsProfs, appleDecls, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.True(t, updates.AppleDeclaration)

	// must reload them to get the profile/declaration uuid
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
	allProfs := getProfs(nil)

	// create an Apple and Windows hosts, not members of any label
	var i int
	winHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("win-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("win-uuid-%d", i),
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, winHost)

	i++
	appleHost, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("apple-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("apple-uuid-%d", i),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, appleHost, false)

	// at this point the hosts have not reported any label results, so a sync
	// does NOT install the exclude any profiles as we don't know yet if the
	// hosts will be members or not
	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {},
		winHost:   {},
	})

	// setting the LabelsUpdatedAt timestamp means that labels results were reported, so now
	// the profiles will be installed as the hosts are not members of the excluded labels.
	winHost.LabelUpdatedAt = time.Now()
	appleHost.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, winHost)
	require.NoError(t, err)
	err = ds.UpdateHost(ctx, appleHost)
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.True(t, updates.AppleDeclaration)
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {
			{
				ProfileUUID:      allProfs[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[0].Identifier,
			},
			{
				ProfileUUID:      allProfs[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[1].Identifier,
			},
		},
		winHost: {
			{
				ProfileUUID:      allProfs[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[2].Name,
			},
		},
	})

	// make all hosts members of labels[1], [2], and [3] so that all profiles are
	// excluded
	err = ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[2].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[1].ID, winHost.ID},
		{labels[2].ID, winHost.ID},
		{labels[3].ID, winHost.ID},
	})
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.True(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {
			{
				ProfileUUID:      allProfs[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: allProfs[0].Identifier,
			},
			{
				ProfileUUID:      allProfs[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeRemove,
				IdentifierOrName: allProfs[1].Identifier,
			},
		},
		// windows profiles are directly deleted without a pending state (there's no on-host removal of profiles)
		winHost: {},
	})

	// make apple host member of labels[2], and windows host member of [3], which are irrelevant
	// for their platforms' profiles, so they get all profiles
	err = ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[1].ID, winHost.ID},
		{labels[2].ID, winHost.ID},
	})
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.True(t, updates.AppleConfigProfile)
	assert.True(t, updates.WindowsConfigProfile)
	assert.True(t, updates.AppleDeclaration)

	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {
			{
				ProfileUUID:      allProfs[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[0].Identifier,
			},
			{
				ProfileUUID:      allProfs[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[1].Identifier,
			},
		},
		winHost: {
			{
				ProfileUUID:      allProfs[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[2].Name,
			},
		},
	})

	// delete labels 0, 2 and 3, breaking all profiles
	err = ds.DeleteLabel(ctx, labels[0].Name)
	require.NoError(t, err)
	err = ds.DeleteLabel(ctx, labels[2].Name)
	require.NoError(t, err)
	err = ds.DeleteLabel(ctx, labels[3].Name)
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// broken profiles do not get reported as "to remove"
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {
			{
				ProfileUUID:      allProfs[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[0].Identifier,
			},
			{
				ProfileUUID:      allProfs[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[1].Identifier,
			},
		},
		winHost: {
			{
				ProfileUUID:      allProfs[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[2].Name,
			},
		},
	})

	// create a new windows and apple host, not a member of any label
	i++
	winHost2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("win-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("win-uuid-%d", i),
		Platform:      "windows",
	})
	require.NoError(t, err)
	windowsEnroll(t, ds, winHost2)

	i++
	appleHost2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      fmt.Sprintf("apple-host%d-name", i),
		OsqueryHostID: ptr.String(fmt.Sprintf("osquery-%d", i)),
		NodeKey:       ptr.String(fmt.Sprintf("nodekey-%d", i)),
		UUID:          fmt.Sprintf("apple-uuid-%d", i),
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, appleHost2, false)

	winHost2.LabelUpdatedAt = time.Now()
	appleHost2.LabelUpdatedAt = time.Now()
	err = ds.UpdateHost(ctx, winHost2)
	require.NoError(t, err)
	err = ds.UpdateHost(ctx, appleHost2)
	require.NoError(t, err)

	updates, err = ds.BulkSetPendingMDMHostProfiles(ctx, []uint{winHost.ID, appleHost.ID, winHost2.ID, appleHost2.ID}, nil, nil, nil)
	require.NoError(t, err)
	assert.False(t, updates.AppleConfigProfile)
	assert.False(t, updates.WindowsConfigProfile)
	assert.False(t, updates.AppleDeclaration)

	// broken profiles do not get reported as "to install"
	assertHostProfiles(t, ds, map[*fleet.Host][]anyProfile{
		appleHost: {
			{
				ProfileUUID:      allProfs[0].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[0].Identifier,
			},
			{
				ProfileUUID:      allProfs[1].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[1].Identifier,
			},
		},
		winHost: {
			{
				ProfileUUID:      allProfs[2].ProfileUUID,
				Status:           &fleet.MDMDeliveryPending,
				OperationType:    fleet.MDMOperationTypeInstall,
				IdentifierOrName: allProfs[2].Name,
			},
		},
		appleHost2: {},
		winHost2:   {},
	})
}

func testBulkSetPendingMDMWindowsHostProfilesLotsOfHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var hostUUIDs []string
	// The bug this test was built to reproduce is visible down to ~16400 hosts; keeping this at 66k for scale testing
	for range 66000 {
		hostUUIDs = append(hostUUIDs, uuid.NewString())
	}

	_, err := ds.bulkSetPendingMDMWindowsHostProfilesDB(ctx, ds.writer(ctx), hostUUIDs, nil)
	require.NoError(t, err)
}

func testBatchResendProfileToHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// create some hosts and some profiles
	host1 := test.NewHost(t, ds, "host1", "1", "h1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "h2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "h3key", "host3uuid", time.Now())
	host4 := test.NewHost(t, ds, "host4", "4", "h4key", "host4uuid", time.Now())

	// create a team and make host4 part of that team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host4.ID}))
	require.NoError(t, err)

	// create some profiles , a and b for no team, c for team
	profA, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 0), nil)
	require.NoError(t, err)
	profB, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 0), nil)
	require.NoError(t, err)
	profC, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("c", "c", team.ID), nil)
	require.NoError(t, err)

	t.Logf("profA=%s, profB=%s, profC=%s", profA.ProfileUUID, profB.ProfileUUID, profC.ProfileUUID)

	assertHostProfileStatus(t, ds, host1.UUID)
	assertHostProfileStatus(t, ds, host2.UUID)
	assertHostProfileStatus(t, ds, host3.UUID)
	assertHostProfileStatus(t, ds, host4.UUID)

	// make profile A installed for all no team
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// batch-resend profile A, does not impact any host
	n, err := ds.BatchResendMDMProfileToHosts(ctx, profA.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host4.UUID)

	// batch-resend profile B, does not impact any host as it's not delievered to any yet
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profB.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host4.UUID)

	// make profile A failed on a couple hosts, verified on the other
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)

	// batch-resend profile A, impacts host 1 and 2
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profA.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 2, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host4.UUID)

	// batch-resend profile A again, no change as it is already pending on the impacted hosts
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profA.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host4.UUID)

	// make profile B failed on all hosts
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryFailed})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryFailed})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryFailed})
	assertHostProfileStatus(t, ds, host4.UUID)

	// batch-resend profile B, all hosts affected
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profB.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 3, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host4.UUID)

	// make profile C failed on host 4, other profiles Verified
	forceSetAppleHostProfileStatus(t, ds, host4.UUID, profC, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profA, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profB, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)

	// batch-resend profile C, host 4 affected
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profC.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host4.UUID,
		hostProfileStatus{profC.ProfileUUID, fleet.MDMDeliveryPending})

	// batch-resend profile C again, no change
	n, err = ds.BatchResendMDMProfileToHosts(ctx, profC.ProfileUUID, fleet.BatchResendMDMProfileFilters{ProfileStatus: fleet.MDMDeliveryFailed})
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profA.ProfileUUID, fleet.MDMDeliveryVerified},
		hostProfileStatus{profB.ProfileUUID, fleet.MDMDeliveryVerified})
	assertHostProfileStatus(t, ds, host4.UUID,
		hostProfileStatus{profC.ProfileUUID, fleet.MDMDeliveryPending})
}

func testGetMDMConfigProfileStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// create some Apple and Windows profiles and declaration for No team
	appleProfsNoTm := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "A1", "A1", uuid.NewString()),
	}
	windowsProfsNoTm := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "W1", "W1"),
	}
	appleDeclsNoTm := []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "{}"),
	}
	_, err = ds.BatchSetMDMProfiles(ctx, nil, appleProfsNoTm, windowsProfsNoTm, appleDeclsNoTm, nil)
	require.NoError(t, err)

	// create some Apple and Windows profiles and declaration for the team
	appleProfsTm := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "A2", "A2", uuid.NewString()),
	}
	windowsProfsTm := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "W2", "W2"),
	}
	appleDeclsTm := []*fleet.MDMAppleDeclaration{
		declForTest("D2", "D2", "{}"),
	}
	_, err = ds.BatchSetMDMProfiles(ctx, &team.ID, appleProfsTm, windowsProfsTm, appleDeclsTm, nil)
	require.NoError(t, err)

	// collect the profiles in a lookup table by name
	profNameToProf := make(map[string]*fleet.MDMConfigProfilePayload)
	profs, _, err := ds.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	for _, prof := range profs {
		profNameToProf[prof.Name] = prof
	}
	profs, _, err = ds.ListMDMConfigProfiles(ctx, &team.ID, fleet.ListOptions{})
	require.NoError(t, err)
	for _, prof := range profs {
		profNameToProf[prof.Name] = prof
	}

	// create some hosts, macOS and Windows, No team and Team
	host1 := test.NewHost(t, ds, "host1", "1", "h1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "h2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "h3key", "host3uuid", time.Now())
	host4 := test.NewHost(t, ds, "host4", "4", "h4key", "host4uuid", time.Now())
	host5 := test.NewHost(t, ds, "host5", "5", "h5key", "host5uuid", time.Now())
	nanoEnroll(t, ds, host1, false)
	nanoEnroll(t, ds, host2, false)
	nanoEnroll(t, ds, host3, false)
	nanoEnroll(t, ds, host4, false)
	nanoEnroll(t, ds, host5, false)

	host6 := test.NewHost(t, ds, "host6", "6", "h6key", "host6uuid", time.Now())
	host6.Platform = "windows"
	err = ds.UpdateHost(ctx, host6)
	require.NoError(t, err)
	windowsEnroll(t, ds, host6)

	host7 := test.NewHost(t, ds, "host7", "7", "h7key", "host7uuid", time.Now())
	host7.Platform = "windows"
	err = ds.UpdateHost(ctx, host7)
	require.NoError(t, err)
	windowsEnroll(t, ds, host7)

	host8 := test.NewHost(t, ds, "host8", "8", "h8key", "host8uuid", time.Now())
	host8.Platform = "windows"
	err = ds.UpdateHost(ctx, host8)
	require.NoError(t, err)
	windowsEnroll(t, ds, host8)

	for _, h := range []*fleet.Host{host1, host2, host3, host4, host5, host6, host7, host8} {
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://fleetdm.com", false, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)
	}

	// host 4, 5 and 8 are on team
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host4.ID, host5.ID, host8.ID}))
	require.NoError(t, err)
	_, _, _ = host1, host2, host3

	// currently no status for any profile
	for _, name := range []string{"A1", "W1", "D1", "A2", "W2", "D2"} {
		status, err := ds.GetMDMConfigProfileStatus(ctx, profNameToProf[name].ProfileUUID)
		require.NoError(t, err, name)
		require.Equal(t, fleet.MDMConfigProfileStatus{}, status, name)
	}
	// unknown profile with invalid prefix is not found (we don't test unknown
	// profile with valid prefix here as the not found is handled at the service
	// layer - it loads the profile for authorization)
	_, err = ds.GetMDMConfigProfileStatus(ctx, "ZZ-NOT-FOUND")
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	cases := []struct {
		desc        string
		profileUUID string
		setup       func(t *testing.T)
		want        fleet.MDMConfigProfileStatus
	}{
		{
			desc:        "macOS no team profile A1 all pending/NULL",
			profileUUID: profNameToProf["A1"].ProfileUUID,
			setup: func(t *testing.T) {
				appleA1 := test.ToMDMAppleConfigProfile(profNameToProf["A1"])
				forceSetAppleHostProfileStatus(t, ds, host1.UUID, appleA1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
				forceSetAppleHostProfileStatus(t, ds, host2.UUID, appleA1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
				forceSetAppleHostProfileStatus(t, ds, host3.UUID, appleA1, fleet.MDMOperationTypeInstall, "")
			},
			want: fleet.MDMConfigProfileStatus{Pending: 3},
		},
		{
			desc:        "windows no team profile W1 all pending/NULL",
			profileUUID: profNameToProf["W1"].ProfileUUID,
			setup: func(t *testing.T) {
				winW1 := test.ToMDMWindowsConfigProfile(profNameToProf["W1"])
				forceSetWindowsHostProfileStatus(t, ds, host6.UUID, winW1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
				forceSetWindowsHostProfileStatus(t, ds, host7.UUID, winW1, fleet.MDMOperationTypeInstall, "")
			},
			want: fleet.MDMConfigProfileStatus{Pending: 2},
		},
		{
			desc:        "windows no team profile W1 pending failed",
			profileUUID: profNameToProf["W1"].ProfileUUID,
			setup: func(t *testing.T) {
				winW1 := test.ToMDMWindowsConfigProfile(profNameToProf["W1"])
				forceSetWindowsHostProfileStatus(t, ds, host6.UUID, winW1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
				forceSetWindowsHostProfileStatus(t, ds, host7.UUID, winW1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
			},
			want: fleet.MDMConfigProfileStatus{Pending: 1, Failed: 1},
		},
		{
			desc:        "macOS no team decl D1 pending failed verified",
			profileUUID: profNameToProf["D1"].ProfileUUID,
			setup: func(t *testing.T) {
				declD1 := test.ToMDMAppleDecl(profNameToProf["D1"])
				forceSetAppleHostDeclarationStatus(t, ds, host1.UUID, declD1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
				forceSetAppleHostDeclarationStatus(t, ds, host2.UUID, declD1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
				forceSetAppleHostDeclarationStatus(t, ds, host3.UUID, declD1, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
			},
			want: fleet.MDMConfigProfileStatus{Pending: 1, Failed: 1, Verified: 1},
		},
		{
			desc:        "macOS team profile A2 verifying verified",
			profileUUID: profNameToProf["A2"].ProfileUUID,
			setup: func(t *testing.T) {
				appleA2 := test.ToMDMAppleConfigProfile(profNameToProf["A2"])
				forceSetAppleHostProfileStatus(t, ds, host4.UUID, appleA2, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
				forceSetAppleHostProfileStatus(t, ds, host5.UUID, appleA2, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
			},
			want: fleet.MDMConfigProfileStatus{Verifying: 1, Verified: 1},
		},
		{
			desc:        "macOS team decl D2 all failed",
			profileUUID: profNameToProf["D2"].ProfileUUID,
			setup: func(t *testing.T) {
				declD2 := test.ToMDMAppleDecl(profNameToProf["D2"])
				forceSetAppleHostDeclarationStatus(t, ds, host4.UUID, declD2, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
				forceSetAppleHostDeclarationStatus(t, ds, host5.UUID, declD2, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
			},
			want: fleet.MDMConfigProfileStatus{Failed: 2},
		},
		{
			desc:        "windows team profile W2 pending",
			profileUUID: profNameToProf["W2"].ProfileUUID,
			setup: func(t *testing.T) {
				winW2 := test.ToMDMWindowsConfigProfile(profNameToProf["W2"])
				forceSetWindowsHostProfileStatus(t, ds, host8.UUID, winW2, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
			},
			want: fleet.MDMConfigProfileStatus{Pending: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.setup(t)
			got, err := ds.GetMDMConfigProfileStatus(ctx, c.profileUUID)
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

func testDeleteMDMProfilesCancelsInstalls(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	SetTestABMAssets(t, ds, "fleet")

	// create some Apple and Windows profiles and declaration
	appleProfs := []*fleet.MDMAppleConfigProfile{
		configProfileForTest(t, "A1", "A1", uuid.NewString()),
		configProfileForTest(t, "A2", "A2", uuid.NewString()),
		configProfileForTest(t, "A3", "A3", uuid.NewString()),
	}
	windowsProfs := []*fleet.MDMWindowsConfigProfile{
		windowsConfigProfileForTest(t, "W1", "W1"),
		windowsConfigProfileForTest(t, "W2", "W2"),
	}
	appleDecls := []*fleet.MDMAppleDeclaration{
		declForTest("D1", "D1", "{}"),
		declForTest("D2", "D2", "{}"),
	}
	_, err := ds.BatchSetMDMProfiles(ctx, nil, appleProfs, windowsProfs, appleDecls, nil)
	require.NoError(t, err)

	// collect the profiles in a lookup table by name
	profNameToProf := make(map[string]*fleet.MDMConfigProfilePayload)
	profs, _, err := ds.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
	require.NoError(t, err)
	for _, prof := range profs {
		profNameToProf[prof.Name] = prof
	}

	// delete all kinds of profiles without any host impacted
	err = ds.DeleteMDMAppleConfigProfile(ctx, profNameToProf["A1"].ProfileUUID)
	require.NoError(t, err)
	err = ds.DeleteMDMWindowsConfigProfile(ctx, profNameToProf["W1"].ProfileUUID)
	require.NoError(t, err)
	err = ds.DeleteMDMAppleDeclaration(ctx, profNameToProf["D1"].ProfileUUID)
	require.NoError(t, err)

	// create some macOS and Windows hosts
	host1 := test.NewHost(t, ds, "host1", "1", "h1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "h2key", "host2uuid", time.Now())
	nanoEnroll(t, ds, host1, false)
	nanoEnroll(t, ds, host2, false)

	host3 := test.NewHost(t, ds, "host3", "3", "h3key", "host3uuid", time.Now())
	host3.Platform = "windows"
	err = ds.UpdateHost(ctx, host3)
	require.NoError(t, err)
	windowsEnroll(t, ds, host3)

	host4 := test.NewHost(t, ds, "host4", "4", "h4key", "host4uuid", time.Now())
	host4.Platform = "windows"
	err = ds.UpdateHost(ctx, host4)
	require.NoError(t, err)
	windowsEnroll(t, ds, host4)

	for _, h := range []*fleet.Host{host1, host2, host3, host4} {
		err = ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://fleetdm.com", false, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)
	}

	// set the declaration as pending install on host1, installed on host2
	forceSetAppleHostDeclarationStatus(t, ds, host1.UUID, test.ToMDMAppleDecl(profNameToProf["D2"]), fleet.MDMOperationTypeInstall, "")
	forceSetAppleHostDeclarationStatus(t, ds, host2.UUID, test.ToMDMAppleDecl(profNameToProf["D2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	assertHostProfileOpStatus(t, ds, host1.UUID,
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall})
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeInstall})

	err = ds.DeleteMDMAppleDeclaration(ctx, profNameToProf["D2"].ProfileUUID)
	require.NoError(t, err)

	assertHostProfileOpStatus(t, ds, host1.UUID)
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})

	// set the Windows profile as pending install on host3, installed on host4
	forceSetWindowsHostProfileStatus(t, ds, host3.UUID, test.ToMDMWindowsConfigProfile(profNameToProf["W2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	forceSetWindowsHostProfileStatus(t, ds, host4.UUID, test.ToMDMWindowsConfigProfile(profNameToProf["W2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)
	assertHostProfileOpStatus(t, ds, host3.UUID,
		hostProfileOpStatus{profNameToProf["W2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall})
	assertHostProfileOpStatus(t, ds, host4.UUID,
		hostProfileOpStatus{profNameToProf["W2"].ProfileUUID, fleet.MDMDeliveryVerified, fleet.MDMOperationTypeInstall})

	err = ds.DeleteMDMWindowsConfigProfile(ctx, profNameToProf["W2"].ProfileUUID)
	require.NoError(t, err)

	assertHostProfileOpStatus(t, ds, host3.UUID)
	assertHostProfileOpStatus(t, ds, host4.UUID)

	commander, _ := createMDMAppleCommanderAndStorage(t, ds)

	// set the Apple profile as pending install on host1, installed on host2
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToProf["A2"]), fleet.MDMOperationTypeInstall, "")
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToProf["A2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	// enqueue the corresponding command for the installed profile
	cmdUUID := uuid.New().String()
	err = commander.InstallProfile(ctx, []string{host2.UUID}, appleProfs[1].Mobileconfig, cmdUUID)
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET command_uuid = ? WHERE host_uuid = ? AND profile_uuid = ?`, cmdUUID, host2.UUID, profNameToProf["A2"].ProfileUUID)
		return err
	})
	assertHostProfileOpStatus(t, ds, host1.UUID,
		hostProfileOpStatus{profNameToProf["A2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall})
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["A2"].ProfileUUID, fleet.MDMDeliveryVerifying, fleet.MDMOperationTypeInstall},
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})

	err = ds.DeleteMDMAppleConfigProfile(ctx, profNameToProf["A2"].ProfileUUID)
	require.NoError(t, err)

	assertHostProfileOpStatus(t, ds, host1.UUID)
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["A2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove},
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})

	// nano command is still active because it was already completed (verifying)
	var active bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &active, `SELECT active FROM nano_enrollment_queue WHERE id = ? AND command_uuid = ?`, host2.UUID, cmdUUID)
	})
	require.True(t, active)

	// set the Apple profile as actually pending install (not NULL) on host1
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToProf["A3"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending)
	// enqueue the corresponding command for the installed profile
	cmdUUID = uuid.New().String()
	err = commander.InstallProfile(ctx, []string{host1.UUID}, appleProfs[2].Mobileconfig, cmdUUID)
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET command_uuid = ? WHERE host_uuid = ? AND profile_uuid = ?`, cmdUUID, host1.UUID, profNameToProf["A3"].ProfileUUID)
		return err
	})
	assertHostProfileOpStatus(t, ds, host1.UUID,
		hostProfileOpStatus{profNameToProf["A3"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall})
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["A2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove},
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})

	err = ds.DeleteMDMAppleConfigProfile(ctx, profNameToProf["A3"].ProfileUUID)
	require.NoError(t, err)

	assertHostProfileOpStatus(t, ds, host1.UUID,
		hostProfileOpStatus{profNameToProf["A3"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})
	assertHostProfileOpStatus(t, ds, host2.UUID,
		hostProfileOpStatus{profNameToProf["A2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove},
		hostProfileOpStatus{profNameToProf["D2"].ProfileUUID, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove})

	// nano command is now inactive because it was in a pending state and we want to prevent delivery
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &active, `SELECT active FROM nano_enrollment_queue WHERE id = ? AND command_uuid = ?`, host1.UUID, cmdUUID)
	})
	require.False(t, active)

	// listing the MDM commands does not return the inactive one
	cmds, err := ds.ListMDMCommands(ctx, fleet.TeamFilter{
		User:            test.UserAdmin,
		IncludeObserver: true,
	}, &fleet.MDMCommandListOptions{Filters: fleet.MDMCommandFilters{HostIdentifier: host1.UUID}})
	require.NoError(t, err)
	require.Len(t, cmds, 0)
}

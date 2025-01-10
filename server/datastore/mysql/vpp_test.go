package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVPP(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SetTeamVPPApps", testSetTeamVPPApps},
		{"VPPAppMetadata", testVPPAppMetadata},
		{"VPPAppStatus", testVPPAppStatus},
		{"VPPApps", testVPPApps},
		{"GetVPPAppByTeamAndTitleID", testGetVPPAppByTeamAndTitleID},
		{"VPPTokensCRUD", testVPPTokensCRUD},
		{"VPPTokenAppTeamAssociations", testVPPTokenAppTeamAssociations},
		{"GetOrInsertSoftwareTitleForVPPApp", testGetOrInsertSoftwareTitleForVPPApp},
		{"DeleteVPPAssignedToPolicy", testDeleteVPPAssignedToPolicy},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Helper()
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testVPPAppMetadata(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	require.NotNil(t, team1)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	require.NotNil(t, team2)

	test.CreateInsertGlobalVPPToken(t, ds)

	// get for non-existing title
	meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, 1)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// create no-team app
	va1, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	vpp1, titleID1 := va1.VPPAppID, va1.TitleID

	// get no-team app
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID1)
	require.NoError(t, err)
	require.NotZero(t, meta.VPPAppsTeamsID)
	meta.VPPAppsTeamsID = 0 // we don't care about the VPP app team PK for comparison purposes
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp1", VPPAppID: vpp1}, meta)

	// try to add the same app again, update self_service field
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}, SelfService: true},
	}, nil)
	require.NoError(t, err)

	// get no-team app
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, ptr.Uint(0), titleID1)
	require.NoError(t, err)
	title, err := ds.GetTitleInfoFromVPPAppsTeamsID(ctx, meta.VPPAppsTeamsID)
	require.NoError(t, err)
	require.Equal(t, titleID1, title.SoftwareTitleID)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp1", VPPAppID: vpp1, SelfService: true}, meta)

	// get nonexistent title
	_, err = ds.GetTitleInfoFromVPPAppsTeamsID(ctx, 0)
	require.Error(t, err)

	// create team1 app
	va2, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.app.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.MacOSPlatform}},
	}, &team1.ID)
	require.NoError(t, err)
	vpp2, titleID2 := va2.VPPAppID, va2.TitleID

	// get it for team 1
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2}, meta)

	// get it for all teams
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2}, meta)

	// try to add the same app again, fails
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.app.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.MacOSPlatform}, SelfService: true},
	}, &team1.ID)
	require.NoError(t, err)

	// get it for team 1
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2, SelfService: true}, meta)

	// get it for team 2, does not exist
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID2)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// create the same app for team2
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.app.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.MacOSPlatform}},
	}, &team2.ID)
	require.NoError(t, err)

	// get it for team 1 and team 2, both work
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0 // we don't care about the VPP app team PK
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2, SelfService: true}, meta)
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2}, meta)

	// create another no-team app
	va3, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp3", BundleIdentifier: "com.app.vpp3",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_3", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	vpp3, titleID3 := va3.VPPAppID, va3.TitleID

	// get it for team 2, does not exist
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID3)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// get it for no-team
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID3)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp3", VPPAppID: vpp3}, meta)

	// delete vpp1
	err = ds.DeleteVPPAppFromTeam(ctx, nil, vpp1)
	require.NoError(t, err)
	// it is now not found
	_, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID1)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	// vpp3 (also in no team) is left untouched
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID3)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0 // we don't care about the VPP app team PK
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp3", VPPAppID: vpp3}, meta)

	// delete vpp2 for team1
	err = ds.DeleteVPPAppFromTeam(ctx, &team1.ID, vpp2)
	require.NoError(t, err)
	// it is now not found for team1
	_, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	// but still found for team2
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID2)
	require.NoError(t, err)
	meta.VPPAppsTeamsID = 0 // we don't care about the VPP app team PK
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", VPPAppID: vpp2}, meta)

	appMeta, err := ds.GetVPPAppMetadataByAdamIDAndPlatform(ctx, meta.AdamID, meta.Platform)
	require.NoError(t, err)
	require.Equal(t, appMeta.AdamID, meta.AdamID)
	require.Equal(t, appMeta.Platform, meta.Platform)

	_, err = ds.GetVPPAppMetadataByAdamIDAndPlatform(ctx, "foo", meta.Platform)
	require.ErrorContains(t, err, "not found")

	// mark it as install_during_setup for team 2
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE global_or_team_id = ? AND adam_id = ?`, team2.ID, vpp2.AdamID)
		return err
	})
	// this prevents its deletion
	err = ds.DeleteVPPAppFromTeam(ctx, &team2.ID, vpp2)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerInstalledDuringSetup)

	// delete vpp1 again fails, not found
	err = ds.DeleteVPPAppFromTeam(ctx, nil, vpp1)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)

	// delete the software title
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM software_titles WHERE id = ?", titleID3)
		return err
	})

	// cannot be returned anymore (deleting the title breaks the relationship)
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID3)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)
}

func testVPPAppStatus(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a user
	user, err := ds.NewUser(ctx, &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	// create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	require.NotNil(t, team1)

	test.CreateInsertGlobalVPPToken(t, ds)

	// create some apps, one for no-team, one for team1, and one in both
	va1, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	vpp1 := va1.VPPAppID
	va2, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp2", BundleIdentifier: "com.app.vpp2",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_2", Platform: fleet.MacOSPlatform}},
	}, &team1.ID)
	require.NoError(t, err)
	vpp2 := va2.VPPAppID
	va3, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp3", BundleIdentifier: "com.app.vpp3",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_3", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	vpp3 := va3.VPPAppID
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp3", BundleIdentifier: "com.app.vpp3",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_3", Platform: fleet.MacOSPlatform}},
	}, &team1.ID)
	require.NoError(t, err)

	// for now they all return zeroes
	summary, err := ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 0}, summary)
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 0}, summary)
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 0}, summary)

	// create a few enrolled hosts
	h1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-1",
		OsqueryHostID:  ptr.String("osquery-macos-1"),
		NodeKey:        ptr.String("node-key-macos-1"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654321a",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, h1, false)

	h2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-2",
		OsqueryHostID:  ptr.String("osquery-macos-2"),
		NodeKey:        ptr.String("node-key-macos-2"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654321b",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, h2, false)

	h3, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-3",
		OsqueryHostID:  ptr.String("osquery-macos-3"),
		NodeKey:        ptr.String("node-key-macos-3"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654321c",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, h3, false)

	// move h3 to team1
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{h3.ID})
	require.NoError(t, err)

	// simulate an install request of vpp1 on h1
	cmd1 := createVPPAppInstallRequest(t, ds, h1, vpp1.AdamID, user.ID)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 1, Failed: 0, Installed: 0}, summary)

	// record a failed result
	createVPPAppInstallResult(t, ds, h1, cmd1, fleet.MDMAppleStatusError)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 1, Installed: 0}, summary)

	// create a new request for h1 that supercedes the failed on, and a request
	// for h2 with a successful result.
	cmd2 := createVPPAppInstallRequest(t, ds, h1, vpp1.AdamID, user.ID)
	cmd3 := createVPPAppInstallRequest(t, ds, h2, vpp1.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, h2, cmd3, fleet.MDMAppleStatusAcknowledged)

	actUser, act, err := ds.GetPastActivityDataForVPPAppInstall(ctx, &mdm.CommandResults{CommandUUID: cmd3})
	require.NoError(t, err)
	require.Equal(t, user.ID, actUser.ID)
	require.Equal(t, user.Name, actUser.Name)
	require.Equal(t, cmd3, act.CommandUUID)
	require.False(t, act.SelfService)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 1, Failed: 0, Installed: 1}, summary)

	// mark the pending request as successful too
	createVPPAppInstallResult(t, ds, h1, cmd2, fleet.MDMAppleStatusAcknowledged)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 2}, summary)

	// requesting for a team (the VPP app is not on any team) returns all zeroes
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 0}, summary)

	// simulate a successful request for team app vpp2 on h3
	cmd4 := createVPPAppInstallRequest(t, ds, h3, vpp2.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, h3, cmd4, fleet.MDMAppleStatusAcknowledged)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 1}, summary)

	// simulate a successful, failed and pending request for app vpp3 on team
	// (h3) and no team (h1, h2)
	cmd5 := createVPPAppInstallRequest(t, ds, h3, vpp3.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, h3, cmd5, fleet.MDMAppleStatusAcknowledged)
	cmd6 := createVPPAppInstallRequest(t, ds, h1, vpp3.AdamID, user.ID)
	createVPPAppInstallResult(t, ds, h1, cmd6, fleet.MDMAppleStatusCommandFormatError)
	createVPPAppInstallRequest(t, ds, h2, vpp3.AdamID, user.ID)

	// for no team, it sees the failed and pending counts
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 1, Failed: 1, Installed: 0}, summary)

	// for the team, it sees the successful count
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 1}, summary)

	// simulate a self-service request
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE host_vpp_software_installs SET self_service = true, user_id = NULL WHERE command_uuid = ?`,
			cmd3)
		return err
	})
	actUser, act, err = ds.GetPastActivityDataForVPPAppInstall(ctx, &mdm.CommandResults{CommandUUID: cmd3})
	require.NoError(t, err)
	require.Nil(t, actUser)
	require.Equal(t, cmd3, act.CommandUUID)
	require.True(t, act.SelfService)
}

// simulates creating the VPP app install request on the host, returns the command UUID.
func createVPPAppInstallRequest(t *testing.T, ds *Datastore, host *fleet.Host, adamID string, userID uint) string {
	ctx := context.Background()

	cmdUUID := uuid.NewString()
	appleCmd := createRawAppleCmd("ProfileList", cmdUUID)
	commander, _ := createMDMAppleCommanderAndStorage(t, ds)
	err := commander.EnqueueCommand(ctx, []string{host.UUID}, appleCmd)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, user_id) VALUES (?, ?, ?, ?, ?)`,
			host.ID, adamID, host.Platform, cmdUUID, userID)
		return err
	})
	return cmdUUID
}

func createVPPAppInstallResult(t *testing.T, ds *Datastore, host *fleet.Host, cmdUUID string, status string) {
	ctx := context.Background()

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, '<?xml')`,
			host.UUID, cmdUUID, status)
		return err
	})
}

func testVPPApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "foobar"})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	// create a host with some non-VPP software
	h1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-1",
		OsqueryHostID:  ptr.String("osquery-macos-1"),
		NodeKey:        ptr.String("node-key-macos-1"),
		UUID:           uuid.NewString(),
		Platform:       "darwin",
		HardwareSerial: "654321a",
	})
	require.NoError(t, err)
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", BundleIdentifier: "b1"},
		{Name: "foo", Version: "0.0.2", BundleIdentifier: "b1"},
		{Name: "bar", Version: "0.0.3", BundleIdentifier: "bar"},
	}
	_, err = ds.UpdateHostSoftware(ctx, h1.ID, software)
	require.NoError(t, err)
	err = ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)

	// Insert some VPP apps for the team, "vpp_app_1" should match the existing "foo" title
	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team.ID)
	require.NoError(t, err)

	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team.ID)
	require.NoError(t, err)

	// Insert some VPP apps for no team
	appNoTeam1 := &fleet.VPPApp{
		Name: "vpp_no_team_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "b3",
	}
	appNoTeam2 := &fleet.VPPApp{
		Name: "vpp_no_team_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "4", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "b4",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, appNoTeam1, nil)
	require.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx, appNoTeam2, nil)
	require.NoError(t, err)

	// Check that host_vpp_software_installs works
	u, err := ds.NewUser(ctx, &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: u})
	err = ds.InsertHostVPPSoftwareInstall(ctx, 1, app1.VPPAppID, "a", "b", false, nil)
	require.NoError(t, err)

	err = ds.InsertHostVPPSoftwareInstall(ctx, 2, app2.VPPAppID, "c", "d", true, nil)
	require.NoError(t, err)

	var results []struct {
		HostID            uint   `db:"host_id"`
		UserID            *uint  `db:"user_id"`
		AdamID            string `db:"adam_id"`
		CommandUUID       string `db:"command_uuid"`
		AssociatedEventID string `db:"associated_event_id"`
		SelfService       bool   `db:"self_service"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &results, `SELECT host_id, user_id, adam_id, command_uuid, associated_event_id, self_service FROM host_vpp_software_installs ORDER BY adam_id`)
	require.NoError(t, err)
	require.Len(t, results, 2)
	a1 := results[0]
	a2 := results[1]
	require.Equal(t, a1.HostID, uint(1))
	require.Equal(t, a1.UserID, ptr.Uint(u.ID))
	require.Equal(t, a1.AdamID, app1.AdamID)
	require.Equal(t, a1.CommandUUID, "a")
	require.Equal(t, a1.AssociatedEventID, "b")
	require.False(t, a1.SelfService)
	require.Equal(t, a2.HostID, uint(2))
	require.Equal(t, a2.UserID, ptr.Uint(u.ID))
	require.Equal(t, a2.AdamID, app2.AdamID)
	require.Equal(t, a2.CommandUUID, "c")
	require.Equal(t, a2.AssociatedEventID, "d")
	require.True(t, a2.SelfService)

	// Check that getting the assigned apps works
	appSet, err := ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	assert.Equal(t, map[fleet.VPPAppID]fleet.VPPAppTeam{
		app1.VPPAppID: {VPPAppID: app1.VPPAppID, InstallDuringSetup: ptr.Bool(false)},
		app2.VPPAppID: {VPPAppID: app2.VPPAppID, InstallDuringSetup: ptr.Bool(false)},
	}, appSet)

	appSet, err = ds.GetAssignedVPPApps(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, map[fleet.VPPAppID]fleet.VPPAppTeam{
		appNoTeam1.VPPAppID: {VPPAppID: appNoTeam1.VPPAppID, InstallDuringSetup: ptr.Bool(false)},
		appNoTeam2.VPPAppID: {VPPAppID: appNoTeam2.VPPAppID, InstallDuringSetup: ptr.Bool(false)},
	}, appSet)

	var appTitles []fleet.SoftwareTitle
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &appTitles, `SELECT name, bundle_identifier FROM software_titles WHERE bundle_identifier IN (?,?) ORDER BY bundle_identifier`, app1.BundleIdentifier, app2.BundleIdentifier)
	require.NoError(t, err)
	require.Len(t, appTitles, 2)
	require.Equal(t, app1.BundleIdentifier, *appTitles[0].BundleIdentifier)
	require.Equal(t, app2.BundleIdentifier, *appTitles[1].BundleIdentifier)
	require.Equal(t, "foo", appTitles[0].Name)
	require.Equal(t, app2.Name, appTitles[1].Name)
}

func testSetTeamVPPApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "vpp gang"})
	require.NoError(t, err)

	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)
	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
	assert.NoError(t, err)

	// Insert some VPP apps for no team
	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, nil)
	require.NoError(t, err)
	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, nil)
	require.NoError(t, err)
	app3 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app3, nil)
	require.NoError(t, err)
	app4 := &fleet.VPPApp{Name: "vpp_app_4", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "4", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b4"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app4, nil)
	require.NoError(t, err)

	assigned, err := ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 0)

	// Assign 2 apps
	// make app1 install_during_setup for that team
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app1.VPPAppID, InstallDuringSetup: ptr.Bool(true)},
		{VPPAppID: app2.VPPAppID, SelfService: true},
	})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 2)
	assert.Contains(t, assigned, app1.VPPAppID)
	assert.Contains(t, assigned, app2.VPPAppID)
	assert.True(t, assigned[app2.VPPAppID].SelfService)
	assert.True(t, *assigned[app1.VPPAppID].InstallDuringSetup)

	// Assign an additional app
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app1.VPPAppID, InstallDuringSetup: ptr.Bool(true)},
		{VPPAppID: app2.VPPAppID},
		{VPPAppID: app3.VPPAppID},
	})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 3)
	require.Contains(t, assigned, app1.VPPAppID)
	require.Contains(t, assigned, app2.VPPAppID)
	require.Contains(t, assigned, app3.VPPAppID)
	assert.False(t, assigned[app2.VPPAppID].SelfService)
	assert.True(t, *assigned[app1.VPPAppID].InstallDuringSetup)

	// Swap one app out for another
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app1.VPPAppID, InstallDuringSetup: ptr.Bool(true)},
		{VPPAppID: app2.VPPAppID, SelfService: true},
		{VPPAppID: app4.VPPAppID},
	})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 3)
	require.Contains(t, assigned, app1.VPPAppID)
	require.Contains(t, assigned, app2.VPPAppID)
	require.Contains(t, assigned, app4.VPPAppID)
	assert.True(t, assigned[app2.VPPAppID].SelfService)
	assert.True(t, *assigned[app1.VPPAppID].InstallDuringSetup)

	// Remove app1 fails because it is installed during setup
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app2.VPPAppID, SelfService: true},
		{VPPAppID: app4.VPPAppID},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerInstalledDuringSetup)

	// make app1 NOT install_during_setup for that team
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app1.VPPAppID, InstallDuringSetup: ptr.Bool(false)},
		{VPPAppID: app2.VPPAppID, SelfService: true},
		{VPPAppID: app4.VPPAppID},
	})
	require.NoError(t, err)

	// Remove app1 now works
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{
		{VPPAppID: app2.VPPAppID, SelfService: true},
		{VPPAppID: app4.VPPAppID},
	})
	require.NoError(t, err)

	// Remove all apps
	err = ds.SetTeamVPPApps(ctx, &team.ID, []fleet.VPPAppTeam{})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 0)
}

func testGetVPPAppByTeamAndTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

	var nfe fleet.NotFoundError

	fooApp, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "foo", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1", Name: "Foo"},
		&team.ID)
	require.NoError(t, err)

	fooTitleID := fooApp.TitleID
	gotVPPApp, err := ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, fooTitleID)
	require.NoError(t, err)
	require.Equal(t, "foo", gotVPPApp.AdamID)
	require.Equal(t, fooTitleID, gotVPPApp.TitleID)
	// title that doesn't exist
	_, err = ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, 999)
	require.ErrorAs(t, err, &nfe)

	// create an entry for the global team
	barApp, err := ds.InsertVPPAppWithTeam(ctx,
		&fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "bar", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b2", Name: "Bar"}, nil)
	require.NoError(t, err)
	barTitleID := barApp.TitleID
	// not found providing the team id
	_, err = ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, barTitleID)
	require.ErrorAs(t, err, &nfe)
	// found for the global team
	gotVPPApp, err = ds.GetVPPAppByTeamAndTitleID(ctx, nil, barTitleID)
	require.NoError(t, err)
	require.Equal(t, "bar", gotVPPApp.AdamID)
	require.Equal(t, barTitleID, gotVPPApp.TitleID)
}

func testVPPTokensCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Kritters"})
	assert.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Zingers"})
	assert.NoError(t, err)

	tokens, err := ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, tokens, 0)

	orgName := "Donkey Kong"
	location := "Jungle"
	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName, location)
	require.NoError(t, err)

	orgName2 := "Diddy Kong"
	location2 := "Mines"
	dataToken2, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName2, location2)
	require.NoError(t, err)

	orgName3 := "Cranky Cong"
	location3 := "Cranky's Cabin"
	dataToken3, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName3, location3)
	require.NoError(t, err)

	orgName4 := "Funky Kong"
	location4 := "Funky's Fishing Shack"
	dataToken4, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName4, location4)
	require.NoError(t, err)

	orgName5 := "Lanky Kong"
	location5 := "Lanky Kong's Pool"
	dataToken5, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName5, location5)
	require.NoError(t, err)

	orgName6 := "Dixie Kong"
	location6 := "Dixie's Island"
	dataToken6, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), orgName6, location6)
	require.NoError(t, err)

	// No assignments / disabled token
	tok, err := ds.InsertVPPToken(ctx, dataToken)
	tokID := tok.ID
	assert.NoError(t, err)
	assert.Equal(t, dataToken.Location, tok.Location)
	assert.Equal(t, dataToken.Token, tok.Token)
	assert.Equal(t, orgName, tok.OrgName)
	assert.Equal(t, location, tok.Location)
	assert.Nil(t, tok.Teams) // No team assigned

	tok, err = ds.GetVPPToken(ctx, tokID)
	assert.NoError(t, err)
	assert.Equal(t, tokID, tok.ID)
	assert.Equal(t, dataToken.Location, tok.Location)
	assert.Equal(t, dataToken.Token, tok.Token)
	assert.Equal(t, orgName, tok.OrgName)
	assert.Equal(t, location, tok.Location)
	assert.Nil(t, tok.Teams)

	toks, err := ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)
	assert.Equal(t, tokID, toks[0].ID)
	assert.Equal(t, dataToken.Location, toks[0].Location)
	assert.Equal(t, dataToken.Token, toks[0].Token)
	assert.Equal(t, orgName, toks[0].OrgName)
	assert.Equal(t, location, toks[0].Location)
	assert.Nil(t, toks[0].Teams)

	teamTok, err := ds.GetVPPTokenByTeamID(ctx, nil)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
	assert.Nil(t, teamTok)

	teamTok, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
	assert.Nil(t, teamTok)

	// Assign to all teams
	upTok, err := ds.UpdateVPPTokenTeams(ctx, tok.ID, []uint{})
	assert.NoError(t, err)
	assert.Equal(t, tokID, upTok.ID)
	assert.Equal(t, dataToken.Location, upTok.Location)
	assert.Equal(t, dataToken.Token, upTok.Token)
	assert.Equal(t, orgName, upTok.OrgName)
	assert.Equal(t, location, upTok.Location)
	assert.NotNil(t, upTok.Teams) // "All Teams" teamm array is non-nil but empty
	assert.Len(t, upTok.Teams, 0)

	tok, err = ds.GetVPPToken(ctx, tok.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokID, tok.ID)
	assert.Equal(t, dataToken.Location, tok.Location)
	assert.Equal(t, dataToken.Token, tok.Token)
	assert.Equal(t, orgName, tok.OrgName)
	assert.Equal(t, location, tok.Location)
	assert.NotNil(t, tok.Teams) // "All Teams" teams array is non-nil but empty
	assert.Len(t, tok.Teams, 0)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)
	assert.Equal(t, dataToken.Location, toks[0].Location)
	assert.Equal(t, dataToken.Token, toks[0].Token)
	assert.Equal(t, orgName, toks[0].OrgName)
	assert.Equal(t, location, toks[0].Location)
	assert.NotNil(t, toks[0].Teams)
	assert.Len(t, toks[0].Teams, 0)

	teamTok, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokID, teamTok.ID)
	assert.Equal(t, dataToken.Location, teamTok.Location)
	assert.Equal(t, dataToken.Token, teamTok.Token)
	assert.Equal(t, orgName, teamTok.OrgName)
	assert.Equal(t, location, teamTok.Location)
	assert.NotNil(t, teamTok.Teams)
	assert.Len(t, teamTok.Teams, 0)

	teamTok, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokID, teamTok.ID)
	assert.Equal(t, dataToken.Location, teamTok.Location)
	assert.Equal(t, dataToken.Token, teamTok.Token)
	assert.Equal(t, orgName, teamTok.OrgName)
	assert.Equal(t, location, teamTok.Location)
	assert.NotNil(t, teamTok.Teams)
	assert.Len(t, teamTok.Teams, 0)

	// Assign to team "No Team"
	upTok, err = ds.UpdateVPPTokenTeams(ctx, tok.ID, []uint{0})
	require.NoError(t, err)
	assert.Len(t, upTok.Teams, 1)
	assert.Equal(t, tokID, upTok.ID)
	assert.Equal(t, uint(0), upTok.Teams[0].ID)
	assert.Equal(t, fleet.TeamNameNoTeam, upTok.Teams[0].Name)

	tok, err = ds.GetVPPToken(ctx, tok.ID)
	assert.NoError(t, err)
	assert.Len(t, tok.Teams, 1)
	assert.Equal(t, tokID, tok.ID)
	assert.Equal(t, uint(0), tok.Teams[0].ID)
	assert.Equal(t, fleet.TeamNameNoTeam, tok.Teams[0].Name)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)
	assert.Len(t, toks[0].Teams, 1)
	assert.Equal(t, tokID, toks[0].ID)
	assert.Equal(t, uint(0), toks[0].Teams[0].ID)
	assert.Equal(t, fleet.TeamNameNoTeam, toks[0].Teams[0].Name)

	teamTok, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokID, teamTok.ID)
	assert.Equal(t, dataToken.Location, teamTok.Location)
	assert.Equal(t, dataToken.Token, teamTok.Token)
	assert.Equal(t, orgName, teamTok.OrgName)
	assert.Equal(t, location, teamTok.Location)
	assert.Len(t, teamTok.Teams, 1)
	assert.Equal(t, uint(0), teamTok.Teams[0].ID)
	assert.Equal(t, fleet.TeamNameNoTeam, teamTok.Teams[0].Name)

	_, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Assign to normal team
	upTok, err = ds.UpdateVPPTokenTeams(ctx, tok.ID, []uint{team.ID})
	assert.NoError(t, err)
	assert.Len(t, upTok.Teams, 1)
	assert.Equal(t, team.ID, upTok.Teams[0].ID)
	assert.Equal(t, team.Name, upTok.Teams[0].Name)

	tok, err = ds.GetVPPToken(ctx, tok.ID)
	assert.NoError(t, err)
	assert.Len(t, tok.Teams, 1)
	assert.Equal(t, team.ID, tok.Teams[0].ID)
	assert.Equal(t, team.Name, tok.Teams[0].Name)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)
	assert.Len(t, toks[0].Teams, 1)
	assert.Equal(t, team.ID, toks[0].Teams[0].ID)
	assert.Equal(t, team.Name, toks[0].Teams[0].Name)

	_, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	teamTok, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokID, teamTok.ID)
	assert.Equal(t, dataToken.Location, teamTok.Location)
	assert.Equal(t, dataToken.Token, teamTok.Token)
	assert.Equal(t, orgName, teamTok.OrgName)
	assert.Equal(t, location, teamTok.Location)
	assert.NotNil(t, teamTok.Teams)
	assert.Len(t, teamTok.Teams, 1)
	assert.Equal(t, team.ID, teamTok.Teams[0].ID)
	assert.Equal(t, team.Name, teamTok.Teams[0].Name)

	// make sure renewing a VPP token doesn't affect associated VPP install automations
	t1app, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, &team.ID)
	require.NoError(t, err)
	t1meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team.ID, t1app.TitleID)
	require.NoError(t, err)

	t1Policy, err := ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:           "p1",
		Query:          "SELECT 1;",
		VPPAppsTeamsID: &t1meta.VPPAppsTeamsID,
	})
	require.NoError(t, err)

	// Renew flow
	upTok, err = ds.UpdateVPPToken(ctx, tokID, dataToken6)
	assert.NoError(t, err)
	assert.Equal(t, tokID, upTok.ID)
	assert.Equal(t, dataToken6.Location, upTok.Location)
	assert.Equal(t, dataToken6.Token, upTok.Token)
	assert.Equal(t, orgName6, upTok.OrgName)
	assert.Equal(t, location6, upTok.Location)
	assert.NotNil(t, upTok.Teams)
	assert.Len(t, upTok.Teams, 1)
	assert.Equal(t, team.ID, upTok.Teams[0].ID)
	assert.Equal(t, team.Name, upTok.Teams[0].Name)

	t1Policy, err = ds.Policy(ctx, t1Policy.ID)
	require.NoError(t, err)
	require.Equal(t, t1Policy.VPPAppsTeamsID, &t1meta.VPPAppsTeamsID)

	tok, err = ds.GetVPPToken(ctx, tok.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokID, tok.ID)
	assert.Equal(t, dataToken6.Location, tok.Location)
	assert.Equal(t, dataToken6.Token, tok.Token)
	assert.Equal(t, orgName6, tok.OrgName)
	assert.Equal(t, location6, tok.Location)
	assert.NotNil(t, tok.Teams)
	assert.Len(t, tok.Teams, 1)
	assert.Equal(t, team.ID, tok.Teams[0].ID)
	assert.Equal(t, team.Name, tok.Teams[0].Name)

	// Assign back to no team / disabled
	upTok, err = ds.UpdateVPPTokenTeams(ctx, tokID, nil)
	assert.NoError(t, err)
	assert.Nil(t, upTok.Teams)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)

	_, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Delete
	err = ds.DeleteVPPToken(ctx, tokID)
	assert.NoError(t, err)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 0)

	// Multiple tokens and constraints tests
	tokNone, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokNone.ID, nil)
	assert.NoError(t, err)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 1)

	_, err = ds.InsertVPPToken(ctx, dataToken)
	assert.Error(t, err)

	tokAll, err := ds.InsertVPPToken(ctx, dataToken2)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, []uint{})
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, []uint{})
	assert.NoError(t, err)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 2)

	// Remove tokAll from All teams
	tokAll, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, nil)
	assert.NoError(t, err)

	tokTeam, err := ds.InsertVPPToken(ctx, dataToken3)
	assert.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeam.ID, []uint{team.ID})
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeam.ID, []uint{team.ID, team.ID})
	assert.Error(t, err)

	// Cannot move tokAll to all teams now
	_, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, []uint{})
	assert.Error(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeam.ID, []uint{0})
	assert.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, []uint{})
	assert.Error(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeam.ID, []uint{team.ID})
	assert.NoError(t, err)

	///

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 3)

	tokTeams, err := ds.InsertVPPToken(ctx, dataToken4)
	assert.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeams.ID, []uint{team.ID, team2.ID})
	assert.Error(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeams.ID, []uint{team2.ID})
	assert.NoError(t, err)

	// make sure updating a VPP token auto-clears associated VPP install automations
	t2app, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, &team2.ID)
	require.NoError(t, err)
	t2meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, t2app.TitleID)
	require.NoError(t, err)

	t2Policy, err := ds.NewTeamPolicy(ctx, team2.ID, nil, fleet.PolicyPayload{
		Name:           "p1",
		Query:          "SELECT 1;",
		VPPAppsTeamsID: &t2meta.VPPAppsTeamsID,
	})
	require.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeams.ID, []uint{team.ID, team2.ID})
	assert.Error(t, err)

	// errored update shouldn't have cleared anything
	t2Policy, err = ds.Policy(ctx, t2Policy.ID)
	assert.NoError(t, err)
	assert.Equal(t, t2meta.VPPAppsTeamsID, *t2Policy.VPPAppsTeamsID)

	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeams.ID, []uint{team.ID, 0})
	assert.Error(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokTeams.ID, []uint{team2.ID, 0})
	assert.NoError(t, err)

	// errored update should have cleared automation
	t2Policy, err = ds.Policy(ctx, t2Policy.ID)
	assert.NoError(t, err)
	assert.Nil(t, t2Policy.VPPAppsTeamsID)

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 4)

	tokTeams, err = ds.GetVPPToken(ctx, tokTeams.ID)
	assert.NoError(t, err)
	assert.Len(t, tokTeams.Teams, 2)
	assert.Contains(t, tokTeams.Teams, fleet.TeamTuple{ID: team2.ID, Name: team2.Name})
	assert.Contains(t, tokTeams.Teams, fleet.TeamTuple{ID: 0, Name: fleet.TeamNameNoTeam})

	tokBadConstraint, err := ds.InsertVPPToken(ctx, dataToken5)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokBadConstraint.ID, []uint{})
	assert.Error(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tokBadConstraint.ID, []uint{team.ID})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "\"Kritters\" team already has a VPP token.")
	_, err = ds.UpdateVPPTokenTeams(ctx, tokBadConstraint.ID, []uint{0})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "\"No team\" team already has a VPP token.")

	toks, err = ds.ListVPPTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, toks, 5)

	///
	tokNil, err := ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokTeams.ID, tokNil.ID)

	tokTeam1, err := ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokTeam.ID, tokTeam1.ID)

	tokTeam2, err := ds.GetVPPTokenByTeamID(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokTeam2.ID, tokTeam2.ID)
	assert.Len(t, tokTeam2.Teams, 2)
	assert.Contains(t, tokTeam2.Teams, fleet.TeamTuple{ID: team2.ID, Name: team2.Name})
	assert.Contains(t, tokTeam2.Teams, fleet.TeamTuple{ID: 0, Name: fleet.TeamNameNoTeam})

	////

	// make sure deleting a VPP token auto-clears associated VPP install automations
	t1app, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, &team.ID)
	require.NoError(t, err)
	t1meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team.ID, t1app.TitleID)
	require.NoError(t, err)

	t1Policy2, err := ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:           "t1p2",
		Query:          "SELECT 1;",
		VPPAppsTeamsID: &t1meta.VPPAppsTeamsID,
	})
	require.NoError(t, err)

	err = ds.DeleteVPPToken(ctx, tokTeam.ID)
	assert.NoError(t, err)

	t1Policy2, err = ds.Policy(ctx, t1Policy2.ID)
	assert.NoError(t, err)
	assert.Nil(t, t1Policy2.VPPAppsTeamsID)

	tokNil, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokTeams.ID, tokNil.ID)

	_, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.Error(t, err)

	tokTeam2, err = ds.GetVPPTokenByTeamID(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokTeams.ID, tokTeam2.ID)

	////
	err = ds.DeleteVPPToken(ctx, tokTeams.ID)
	assert.NoError(t, err)

	_, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetVPPTokenByTeamID(ctx, &team2.ID)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	////
	tokAll, err = ds.UpdateVPPTokenTeams(ctx, tokAll.ID, []uint{})
	assert.NoError(t, err)

	tokNil, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokAll.ID, tokNil.ID)

	tokTeam1, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokAll.ID, tokTeam1.ID)

	tokTeam2, err = ds.GetVPPTokenByTeamID(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokAll.ID, tokTeam2.ID)

	err = ds.DeleteVPPToken(ctx, tokAll.ID)
	assert.NoError(t, err)

	////
	_, err = ds.UpdateVPPTokenTeams(ctx, tokNone.ID, []uint{0, team.ID, team2.ID})
	assert.NoError(t, err)

	tokNil, err = ds.GetVPPTokenByTeamID(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, tokNone.ID, tokNil.ID)

	tokTeam1, err = ds.GetVPPTokenByTeamID(ctx, &team.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokNone.ID, tokTeam1.ID)

	tokTeam2, err = ds.GetVPPTokenByTeamID(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Equal(t, tokNone.ID, tokTeam2.ID)

	////
	err = ds.DeleteVPPToken(ctx, tokNone.ID)
	assert.NoError(t, err)

}

func testVPPTokenAppTeamAssociations(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Kritters"})
	assert.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Zingers"})
	assert.NoError(t, err)

	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)

	dataToken2, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Diddy Kong", "Mines")
	require.NoError(t, err)

	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)

	tok2, err := ds.InsertVPPToken(ctx, dataToken2)
	assert.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{team1.ID})
	assert.NoError(t, err)

	_, err = ds.UpdateVPPTokenTeams(ctx, tok2.ID, []uint{team2.ID})
	assert.NoError(t, err)

	app1 := &fleet.VPPApp{
		Name: "app1",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		BundleIdentifier: "app1",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	assert.NoError(t, err)
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team2.ID)
	assert.NoError(t, err)

	app2 := &fleet.VPPApp{
		Name: "app2",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.MacOSPlatform,
			},
		},
		BundleIdentifier: "app2",
	}
	vppApp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	_ = vppApp2
	assert.NoError(t, err)

	// team1: token 1, app1, app2
	// team2: token 2, app 1

	apps, err := ds.GetAssignedVPPApps(ctx, &team1.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Contains(t, apps, app1.VPPAppID)
	assert.Contains(t, apps, app2.VPPAppID)

	apps, err = ds.GetAssignedVPPApps(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Contains(t, apps, app1.VPPAppID)

	/// Try to move team 1 token to team 2

	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{team2.ID})
	assert.Error(t, err)

	// team1: token 1, app1, app2
	// team2: token 2, app 1

	apps, err = ds.GetAssignedVPPApps(ctx, &team1.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 2)

	apps, err = ds.GetAssignedVPPApps(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Contains(t, apps, app1.VPPAppID)

	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, nil)
	assert.NoError(t, err)

	// team1: no token, no apps
	// team2: token 2, app 1

	apps, err = ds.GetAssignedVPPApps(ctx, &team1.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 0)

	apps, err = ds.GetAssignedVPPApps(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 1)
	assert.Contains(t, apps, app1.VPPAppID)

	// Move team 2 token to team 1

	_, err = ds.UpdateVPPTokenTeams(ctx, tok2.ID, []uint{team1.ID})
	assert.NoError(t, err)

	// team1: token 2, app 1
	// team2: no token, no apps

	apps, err = ds.GetAssignedVPPApps(ctx, &team1.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 0)

	apps, err = ds.GetAssignedVPPApps(ctx, &team2.ID)
	assert.NoError(t, err)
	assert.Len(t, apps, 0)

	/// Can't assaign apps with no token

	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team2.ID)
	assert.Error(t, err)
}

func testGetOrInsertSoftwareTitleForVPPApp(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "Existing Title", Version: "0.0.1", Source: "apps", BundleIdentifier: "existing.title"},
	}
	software2 := []fleet.Software{
		{Name: "Existing Title", Version: "v0.0.2", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title", Version: "0.0.3", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title Without Bundle", Version: "0.0.3", Source: "apps"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	tests := []struct {
		name string
		app  *fleet.VPPApp
	}{
		{
			name: "title that already exists, no bundle identifier in payload",
			app: &fleet.VPPApp{
				Name:             "Existing Title",
				LatestVersion:    "0.0.1",
				BundleIdentifier: "",
			},
		},
		{
			name: "title that already exists, bundle identifier in payload",
			app: &fleet.VPPApp{
				Name:             "Existing Title",
				LatestVersion:    "0.0.2",
				BundleIdentifier: "existing.title",
			},
		},
		{
			name: "title that already exists but doesn't have a bundle identifier",
			app: &fleet.VPPApp{
				Name:             "Existing Title Without Bundle",
				LatestVersion:    "0.0.3",
				BundleIdentifier: "",
			},
		},
		{
			name: "title that already exists, no bundle identifier in DB, bundle identifier in payload",
			app: &fleet.VPPApp{
				Name:             "Existing Title Without Bundle",
				LatestVersion:    "0.0.3",
				BundleIdentifier: "new.bundle.id",
			},
		},
		{
			name: "title that doesn't exist, no bundle identifier in payload",
			app: &fleet.VPPApp{
				Name:             "New Title",
				LatestVersion:    "0.1.0",
				BundleIdentifier: "",
			},
		},
		{
			name: "title that doesn't exist, with bundle identifier in payload",
			app: &fleet.VPPApp{
				Name:             "New Title",
				LatestVersion:    "0.1.0",
				BundleIdentifier: "new.title.bundle",
			},
		},
	}

	for _, platform := range fleet.VPPAppsPlatforms {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s_%v", tt.name, platform), func(t *testing.T) {
				tt.app.Platform = platform
				var id uint
				err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
					var err error
					id, err = ds.getOrInsertSoftwareTitleForVPPApp(ctx, tx, tt.app)
					return err
				})
				require.NoError(t, err)
				require.NotEmpty(t, id)
			})
		}
	}
}

func testDeleteVPPAssignedToPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	test.CreateInsertGlobalVPPToken(t, ds)

	va1, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_app_1", Platform: fleet.MacOSPlatform}},
	}, nil)
	require.NoError(t, err)
	meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, ptr.Uint(0), va1.TitleID)
	require.NoError(t, err)

	p1, err := ds.NewTeamPolicy(ctx, fleet.PolicyNoTeamID, nil, fleet.PolicyPayload{
		Name:           "p1",
		Query:          "SELECT 1;",
		VPPAppsTeamsID: &meta.VPPAppsTeamsID,
	})
	require.NoError(t, err)

	err = ds.DeleteVPPAppFromTeam(ctx, ptr.Uint(0), va1.VPPAppID)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerWithAssociatedPolicy)

	_, err = ds.DeleteTeamPolicies(ctx, fleet.PolicyNoTeamID, []uint{p1.ID})
	require.NoError(t, err)

	err = ds.DeleteVPPAppFromTeam(ctx, ptr.Uint(0), va1.VPPAppID)
	require.NoError(t, err)
}

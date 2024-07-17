package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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

	// get for non-existing title
	meta, err := ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, 1)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// create no-team app
	vpp1, titleID1 := createVPPApp(t, ds, nil, "vpp1", "com.app.vpp1")

	// get no-team app
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID1)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp1", AppStoreID: vpp1}, meta)

	// create team1 app
	vpp2, titleID2 := createVPPApp(t, ds, &team1.ID, "vpp2", "com.app.vpp2")

	// get it for team 1
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", AppStoreID: vpp2}, meta)

	// get it for team 2, does not exist
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID2)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// create the same app for team2
	createVPPAppTeamOnly(t, ds, &team2.ID, vpp2)

	// get it for team 1 and team 2, both work
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team1.ID, titleID2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", AppStoreID: vpp2}, meta)
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", AppStoreID: vpp2}, meta)

	// create another no-team app
	vpp3, titleID3 := createVPPApp(t, ds, nil, "vpp3", "com.app.vpp3")

	// get it for team 2, does not exist
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team2.ID, titleID3)
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, meta)

	// get it for no-team
	meta, err = ds.GetVPPAppMetadataByTeamAndTitleID(ctx, nil, titleID3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp3", AppStoreID: vpp3}, meta)

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
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp3", AppStoreID: vpp3}, meta)

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
	require.Equal(t, &fleet.VPPAppStoreApp{Name: "vpp2", AppStoreID: vpp2}, meta)

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

	// create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	require.NotNil(t, team1)

	// create some apps, one for no-team, one for team1, and one in both
	vpp1, _ := createVPPApp(t, ds, nil, "vpp1", "com.app.vpp1")
	vpp2, _ := createVPPApp(t, ds, &team1.ID, "vpp2", "com.app.vpp2")
	vpp3, _ := createVPPApp(t, ds, nil, "vpp3", "com.app.vpp3")
	createVPPAppTeamOnly(t, ds, &team1.ID, vpp3)

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
	cmd1 := createVPPAppInstallRequest(t, ds, h1, vpp1)

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
	cmd2 := createVPPAppInstallRequest(t, ds, h1, vpp1)
	cmd3 := createVPPAppInstallRequest(t, ds, h2, vpp1)
	createVPPAppInstallResult(t, ds, h2, cmd3, fleet.MDMAppleStatusAcknowledged)

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
	cmd4 := createVPPAppInstallRequest(t, ds, h3, vpp2)
	createVPPAppInstallResult(t, ds, h3, cmd4, fleet.MDMAppleStatusAcknowledged)

	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp2)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 1}, summary)

	// simulate a successful, failed and pending request for app vpp3 on team
	// (h3) and no team (h1, h2)
	cmd5 := createVPPAppInstallRequest(t, ds, h3, vpp3)
	createVPPAppInstallResult(t, ds, h3, cmd5, fleet.MDMAppleStatusAcknowledged)
	cmd6 := createVPPAppInstallRequest(t, ds, h1, vpp3)
	createVPPAppInstallResult(t, ds, h1, cmd6, fleet.MDMAppleStatusCommandFormatError)
	createVPPAppInstallRequest(t, ds, h2, vpp3)

	// for no team, it sees the failed and pending counts
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, nil, vpp3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 1, Failed: 1, Installed: 0}, summary)

	// for the team, it sees the successful count
	summary, err = ds.GetSummaryHostVPPAppInstalls(ctx, &team1.ID, vpp3)
	require.NoError(t, err)
	require.Equal(t, &fleet.VPPAppStatusSummary{Pending: 0, Failed: 0, Installed: 1}, summary)
}

// simulates creating the VPP app install request on the host, returns the command UUID.
func createVPPAppInstallRequest(t *testing.T, ds *Datastore, host *fleet.Host, adamID string) string {
	ctx := context.Background()

	cmdUUID := uuid.NewString()
	appleCmd := createRawAppleCmd("ProfileList", cmdUUID)
	commander, _ := createMDMAppleCommanderAndStorage(t, ds)
	err := commander.EnqueueCommand(ctx, []string{host.UUID}, appleCmd)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid) VALUES (?, ?, ?)`,
			host.ID, adamID, cmdUUID)
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
	app1 := &fleet.VPPApp{Name: "vpp_app_1", AdamID: "1", BundleIdentifier: "b1"}
	app2 := &fleet.VPPApp{Name: "vpp_app_2", AdamID: "2", BundleIdentifier: "b2"}
	err = ds.InsertVPPAppWithTeam(ctx, app1, &team.ID)
	require.NoError(t, err)

	err = ds.InsertVPPAppWithTeam(ctx, app2, &team.ID)
	require.NoError(t, err)

	// Insert some VPP apps for no team
	appNoTeam1 := &fleet.VPPApp{Name: "vpp_no_team_app_1", AdamID: "3", BundleIdentifier: "b3"}
	appNoTeam2 := &fleet.VPPApp{Name: "vpp_no_team_app_2", AdamID: "4", BundleIdentifier: "b4"}
	err = ds.InsertVPPAppWithTeam(ctx, appNoTeam1, nil)
	require.NoError(t, err)
	err = ds.InsertVPPAppWithTeam(ctx, appNoTeam2, nil)
	require.NoError(t, err)

	// Check that getting the assigned apps works
	appSet, err := ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Equal(t, map[string]struct{}{app1.AdamID: {}, app2.AdamID: {}}, appSet)

	appSet, err = ds.GetAssignedVPPApps(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]struct{}{appNoTeam1.AdamID: {}, appNoTeam2.AdamID: {}}, appSet)

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

	// Insert some VPP apps for the team
	app1 := &fleet.VPPApp{Name: "vpp_app_1", AdamID: "1", BundleIdentifier: "b1"}
	err = ds.InsertVPPAppWithTeam(ctx, app1, nil)
	require.NoError(t, err)
	app2 := &fleet.VPPApp{Name: "vpp_app_2", AdamID: "2", BundleIdentifier: "b2"}
	err = ds.InsertVPPAppWithTeam(ctx, app2, nil)
	require.NoError(t, err)
	app3 := &fleet.VPPApp{Name: "vpp_app_3", AdamID: "3", BundleIdentifier: "b3"}
	err = ds.InsertVPPAppWithTeam(ctx, app3, nil)
	require.NoError(t, err)
	app4 := &fleet.VPPApp{Name: "vpp_app_4", AdamID: "4", BundleIdentifier: "b4"}
	err = ds.InsertVPPAppWithTeam(ctx, app4, nil)
	require.NoError(t, err)

	assigned, err := ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 0)

	// Assign 2 apps
	err = ds.SetTeamVPPApps(ctx, &team.ID, []string{app1.AdamID, app2.AdamID})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 2)
	require.Contains(t, assigned, app1.AdamID)
	require.Contains(t, assigned, app2.AdamID)

	// Assign an additional app
	err = ds.SetTeamVPPApps(ctx, &team.ID, []string{app1.AdamID, app2.AdamID, app3.AdamID})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 3)
	require.Contains(t, assigned, app1.AdamID)
	require.Contains(t, assigned, app2.AdamID)
	require.Contains(t, assigned, app3.AdamID)

	// Swap one app out for another
	err = ds.SetTeamVPPApps(ctx, &team.ID, []string{app1.AdamID, app2.AdamID, app4.AdamID})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 3)
	require.Contains(t, assigned, app1.AdamID)
	require.Contains(t, assigned, app2.AdamID)
	require.Contains(t, assigned, app4.AdamID)

	// Remove all apps
	err = ds.SetTeamVPPApps(ctx, &team.ID, []string{})
	require.NoError(t, err)

	assigned, err = ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 0)
}

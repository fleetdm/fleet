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

func TestVPPApps(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"VPPAppMetadata", testVPPAppMetadata},
		{"VPPAppStatus", testVPPAppStatus},
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
		_, err := q.ExecContext(ctx, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, '')`,
			host.UUID, cmdUUID, status)
		return err
	})
}

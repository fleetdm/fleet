package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallers(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InsertSoftwareInstallRequest", testInsertSoftwareInstallRequest},
		{"GetSoftwareInstallResults", testGetSoftwareInstallResult},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testInsertSoftwareInstallRequest(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	cases := map[string]*uint{
		"no team": nil,
		"team":    &team.ID,
	}

	for tc, teamID := range cases {
		t.Run(tc, func(t *testing.T) {
			// non-existent installer and host does the installer check first
			err := ds.InsertSoftwareInstallRequest(ctx, 1, 1, teamID)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)

			// non-existent host
			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo",
				Source:        "bar",
				InstallScript: "echo",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			installerMeta, err := ds.GetSoftwareInstallerMetadata(ctx, installerID)
			require.NoError(t, err)

			err = ds.InsertSoftwareInstallRequest(ctx, 12, installerMeta.TitleID, teamID)
			require.ErrorAs(t, err, &nfe)

			// successful insert
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tc),
				NodeKey:       ptr.String("node-key-macos" + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			err = ds.InsertSoftwareInstallRequest(ctx, host.ID, installerMeta.TitleID, teamID)
			require.NoError(t, err)
		})
	}
}

func testGetSoftwareInstallResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	teamID := team.ID

	// create a host and software installer

	installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:         "foo",
		Source:        "bar",
		InstallScript: "echo",
		TeamID:        &teamID,
	})
	require.NoError(t, err)
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "macos-test",
		OsqueryHostID: ptr.String("osquery-macos"),
		NodeKey:       ptr.String("node-key-macos"),
		UUID:          uuid.NewString(),
		Platform:      "darwin",
		TeamID:        &teamID,
	})
	require.NoError(t, err)

	// Need to insert manually so we have access to the UUID (it's generated in the SQL method)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES (?,?,?)`, "1234", host.ID, installerID)
	require.NoError(t, err)

	res, err := ds.GetSoftwareInstallResults(ctx, "1234")
	require.NoError(t, err)

	require.Equal(t, "1234", res.InstallUUID)
	require.Equal(t, fleet.SoftwareInstallerPending, res.Status)
}

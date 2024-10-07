package mysql

import (
	"bytes"
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"EnqueueSetupExperienceItems", testEnqueueSetupExperienceItems},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testEnqueueSetupExperienceItems(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	installerID1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	require.NoError(t, err)

	installerID2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     bytes.NewReader([]byte("hello")),
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team2.ID)
	require.NoError(t, err)

	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?)", vpp1.AdamID, vpp2.AdamID)
		return err
	})

	anything, err := ds.EnqueueSetupExperienceItems(ctx, "123213", team1.ID)
	require.NoError(t, err)
	require.True(t, anything)

	anything, err = ds.EnqueueSetupExperienceItems(ctx, "213321", team3.ID)
	require.NoError(t, err)
	require.False(t, anything)
}

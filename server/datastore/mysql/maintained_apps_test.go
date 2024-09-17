package mysql

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/go-kit/kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestMaintainedApps(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpsertMaintainedApps", testUpsertMaintainedApps},
		{"IngestWithBrew", testIngestWithBrew},
		{"ListAvailableApps", testListAvailableApps},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUpsertMaintainedApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	listSavedApps := func() []*fleet.MaintainedApp {
		var apps []*fleet.MaintainedApp
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &apps, "SELECT name, version, platform FROM fleet_library_apps ORDER BY token")
		})
		return apps
	}

	expectedApps := maintainedapps.IngestMaintainedApps(t, ds)
	require.Equal(t, expectedApps, listSavedApps())

	// ingesting again results in no changes
	maintainedapps.IngestMaintainedApps(t, ds)
	require.Equal(t, expectedApps, listSavedApps())

	// upsert the figma app, changing the version
	err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:         "Figma",
		Token:        "figma",
		InstallerURL: "https://desktop.figma.com/mac-arm/Figma-999.9.9.zip",
		Version:      "999.9.9",
		Platform:     fleet.MacOSPlatform,
	})
	require.NoError(t, err)

	// change the expected app data for figma
	for _, app := range expectedApps {
		if app.Name == "Figma" {
			app.Version = "999.9.9"
			break
		}
	}
	require.Equal(t, expectedApps, listSavedApps())
}

func testIngestWithBrew(t *testing.T, ds *Datastore) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	ctx := context.Background()
	err := maintainedapps.Refresh(ctx, ds, log.NewNopLogger())
	require.NoError(t, err)

	expectedTokens := maintainedapps.ExpectedAppTokens(t)
	var actualTokens []string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &actualTokens, "SELECT token FROM fleet_library_apps ORDER BY token")
	})
	require.ElementsMatch(t, expectedTokens, actualTokens)
}

func testListAvailableApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)

	err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Token:            "maintained1",
		Version:          "1.0.0",
		Platform:         fleet.MacOSPlatform,
		InstallerURL:     "http://example.com/main1",
		SHA256:           "DEADBEEF",
		BundleIdentifier: "fleet.maintained1",
		InstallScript:    "echo installed",
		UninstallScript:  "echo uninstalled",
	})
	require.NoError(t, err)
	err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained2",
		Token:            "maintained2",
		Version:          "1.0.0",
		Platform:         fleet.MacOSPlatform,
		InstallerURL:     "http://example.com/main1",
		SHA256:           "DEADBEEF",
		BundleIdentifier: "fleet.maintained2",
		InstallScript:    "echo installed",
		UninstallScript:  "echo uninstalled",
	})
	require.NoError(t, err)
	err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained3",
		Token:            "maintained3",
		Version:          "1.0.0",
		Platform:         fleet.IOSPlatform,
		InstallerURL:     "http://example.com/main1",
		SHA256:           "DEADBEEF",
		BundleIdentifier: "fleet.maintained3",
		InstallScript:    "echo installed",
		UninstallScript:  "echo uninstalled",
	})
	require.NoError(t, err)

	expectedApps := []fleet.FleetMaintainedAppAvailable{
		{
			ID:       "1",
			Name:     "Maintained1",
			Version:  "1.0.0",
			Platform: fleet.MacOSPlatform,
		},
		{
			ID:       "2",
			Name:     "Maintained2",
			Version:  "1.0.0",
			Platform: fleet.MacOSPlatform,
		},
		{
			ID:       "3",
			Name:     "Maintained3",
			Version:  "1.0.0",
			Platform: fleet.IOSPlatform,
		},
	}

	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, team.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.Equal(t, expectedApps, apps)
	require.False(t, meta.HasNextResults)
}

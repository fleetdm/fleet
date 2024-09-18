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
		{"GetMaintainedAppByID", testGetMaintainedAppByID},
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
	_, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
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

func testGetMaintainedAppByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	expApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "foo",
		Token:            "foo",
		Version:          "1.0.0",
		Platform:         "darwin",
		InstallerURL:     "https://example.com/foo.zip",
		SHA256:           "abc",
		BundleIdentifier: "abc",
		InstallScript:    "foo",
		UninstallScript:  "foo",
	})
	require.NoError(t, err)

	gotApp, err := ds.GetMaintainedAppByID(ctx, expApp.ID)
	require.NoError(t, err)

	require.Equal(t, expApp, gotApp)
}

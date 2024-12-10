package mysql

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"GetMaintainedAppByID", testGetMaintainedAppByID},
		{"GetSoftwareTitleIdByAppID", testGetSoftwareTitleIdByAppID},
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

	listSavedApps := func() []fleet.MaintainedApp {
		var apps []fleet.MaintainedApp
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
	for idx := range expectedApps {
		if expectedApps[idx].Name == "Figma" {
			expectedApps[idx].Version = "999.9.9"
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

	user := test.NewUser(t, ds, "Zaphod Beeblebrox", "zaphod@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 2"})
	require.NoError(t, err)

	maintained1, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
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
	maintained2, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
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
	maintained3, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained3",
		Token:            "maintained3",
		Version:          "1.0.0",
		Platform:         fleet.MacOSPlatform,
		InstallerURL:     "http://example.com/main1",
		SHA256:           "DEADBEEF",
		BundleIdentifier: "fleet.maintained3",
		InstallScript:    "echo installed",
		UninstallScript:  "echo uninstalled",
	})
	require.NoError(t, err)

	expectedApps := []fleet.MaintainedApp{
		{
			ID:       maintained1.ID,
			Name:     maintained1.Name,
			Version:  maintained1.Version,
			Platform: maintained1.Platform,
		},
		{
			ID:       maintained2.ID,
			Name:     maintained2.Name,
			Version:  maintained2.Version,
			Platform: maintained2.Platform,
		},
		{
			ID:       maintained3.ID,
			Name:     maintained3.Name,
			Version:  maintained3.Version,
			Platform: maintained3.Platform,
		},
	}

	// We use this assertion for UpdatedAt because we only concerned with
	// its presence, not its value. We will set it to nil after asserting
	// to make the expected vs actual comparison easier.
	assertUpdatedAt := func(apps []fleet.MaintainedApp) {
		for i, app := range apps {
			require.NotNil(t, app.UpdatedAt)
			apps[i].UpdatedAt = nil
		}
	}

	// Testing pagination
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps, apps)
	require.False(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[:1], apps)
	require.True(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[1:2], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 2, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[2:3], apps)
	require.False(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	//
	// Test excluding results for existing apps (installers)

	/// Irrelevant package
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Irrelevant Software",
		TeamID:           &team1.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: "irrelevant_1",
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps, apps)

	/// Correct package on a different team
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Maintained1",
		TeamID:           &team2.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps, apps)

	/// Correct package on the right team with the wrong platform
	_, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Maintained1",
		TeamID:           &team1.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.IOSPlatform),
		BundleIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps, apps)

	/// Correct team and platform
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET platform = ? WHERE platform = ?", fleet.MacOSPlatform, fleet.IOSPlatform)
		return err
	})

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, meta.TotalResults, 2)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[1:], apps)

	//
	// Test excluding results for existing apps (VPP)

	test.CreateInsertGlobalVPPToken(t, ds)

	// irrelevant vpp app
	vppIrrelevant := &fleet.VPPApp{
		Name: "irrelevant_app",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		BundleIdentifier: "irrelevant_2",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppIrrelevant, &team1.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, meta.TotalResults, 2)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[1:], apps)

	// right vpp app, wrong team
	vppMaintained2 := &fleet.VPPApp{
		Name: "Maintained 2",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.MacOSPlatform,
			},
		},
		BundleIdentifier: "fleet.maintained2",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppMaintained2, &team2.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.EqualValues(t, meta.TotalResults, 2)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[1:], apps)

	// right vpp app, right team
	_, err = ds.InsertVPPAppWithTeam(ctx, vppMaintained2, &team1.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 1)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[2:], apps)

	// right app, right team, wrong platform
	vppMaintained3 := &fleet.VPPApp{
		Name: "Maintained 3",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "3",
				Platform: fleet.IOSPlatform,
			},
		},
		BundleIdentifier: "fleet.maintained3",
	}

	_, err = ds.InsertVPPAppWithTeam(ctx, vppMaintained3, &team1.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 1)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps[2:], apps)

	// viewing with no team selected shouldn't exclude any results
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, nil, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	assertUpdatedAt(apps)
	require.Equal(t, expectedApps, apps)
}

func testGetMaintainedAppByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	expApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "foo",
		Token:            "token",
		Version:          "1.0.0",
		Platform:         "darwin",
		InstallerURL:     "https://example.com/foo.zip",
		SHA256:           "sha",
		BundleIdentifier: "bundle",
		InstallScript:    "install",
		UninstallScript:  "uninstall",
	})
	require.NoError(t, err)

	gotApp, err := ds.GetMaintainedAppByID(ctx, expApp.ID)
	require.NoError(t, err)

	require.Equal(t, expApp, gotApp)
}

func testGetSoftwareTitleIdByAppID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Maintained app doesn't exist, should get not found error
	_, err := ds.GetSoftwareTitleIDByMaintainedAppID(ctx, 99, nil)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	app, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "foo",
		Token:            "token",
		Version:          "1.0.0",
		Platform:         "darwin",
		InstallerURL:     "https://example.com/foo.zip",
		SHA256:           "sha",
		BundleIdentifier: "bundle",
		InstallScript:    "install",
		UninstallScript:  "uninstall",
	})
	require.NoError(t, err)

	// Valid maintained app ID, but no installer yet so we should get not found error
	_, err = ds.GetSoftwareTitleIDByMaintainedAppID(ctx, app.ID, nil)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// create a software installer for team and for no team
	installer, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	installerTm1ID, err := ds.MatchOrCreateSoftwareInstaller(context.Background(), &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		InstallerFile:     installer,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		FleetLibraryAppID: &app.ID,
	})
	require.NoError(t, err)

	_, err = ds.MatchOrCreateSoftwareInstaller(context.Background(), &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		InstallerFile:     installer,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            nil,
		FleetLibraryAppID: &app.ID,
	})
	require.NoError(t, err)

	// get the software installer metadata as we will need the associated software title id.
	installer1, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerTm1ID)
	require.NoError(t, err)
	require.NotNil(t, installer1.TitleID)

	stID, err := ds.GetSoftwareTitleIDByMaintainedAppID(ctx, app.ID, &team1.ID)
	require.NoError(t, err)
	require.Equal(t, *installer1.TitleID, stID)

	stNoTmID, err := ds.GetSoftwareTitleIDByMaintainedAppID(ctx, app.ID, nil)
	require.NoError(t, err)
	require.Equal(t, *installer1.TitleID, stNoTmID)

	require.NoError(t, err)
}

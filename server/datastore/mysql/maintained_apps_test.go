package mysql

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
		{"ListAndGetAvailableApps", testListAndGetAvailableApps},
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
		Platform:     "darwin",
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

func testListAndGetAvailableApps(t *testing.T, ds *Datastore) {
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
		Platform:         "darwin",
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
		Platform:         "darwin",
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
		Platform:         "darwin",
		InstallerURL:     "http://example.com/main1",
		SHA256:           "DEADBEEF",
		BundleIdentifier: "fleet.maintained3",
		InstallScript:    "echo installed",
		UninstallScript:  "echo uninstalled",
	})
	require.NoError(t, err)

	gotApp, err := ds.GetMaintainedAppByID(ctx, maintained1.ID, nil)
	require.NoError(t, err)
	require.Equal(t, maintained1, gotApp)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, &team1.ID)
	require.NoError(t, err)
	require.Equal(t, maintained1, gotApp)

	expectedApps := []fleet.MaintainedApp{
		{
			ID:       maintained1.ID,
			Name:     maintained1.Name,
			Platform: maintained1.Platform,
		},
		{
			ID:       maintained2.ID,
			Name:     maintained2.Name,
			Platform: maintained2.Platform,
		},
		{
			ID:       maintained3.ID,
			Name:     maintained3.Name,
			Platform: maintained3.Platform,
		},
	}

	// Testing pagination
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)
	require.False(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps[:1], apps)
	require.True(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps[1:2], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 2, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps[2:3], apps)
	require.False(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	//
	// Test including software title ID for existing apps (installers)

	/// Irrelevant package
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Irrelevant Software",
		TeamID:           &team1.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: "irrelevant_1",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

	/// Correct package on a different team
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Maintained1",
		TeamID:           &team2.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: "fleet.maintained1",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

	/// Correct package on the right team with the wrong platform
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Maintained1",
		TeamID:           &team1.ID,
		InstallScript:    "nothing",
		Filename:         "foo.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.IOSPlatform),
		BundleIdentifier: "fleet.maintained1",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, &team1.ID)
	require.NoError(t, err)
	require.Equal(t, maintained1, gotApp)

	/// Correct team and platform
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET platform = ? WHERE platform = ?", fleet.MacOSPlatform, fleet.IOSPlatform)
		return err
	})

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	expectedApps[0].TitleID = ptr.Uint(titleID)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, ptr.Uint(0))
	require.NoError(t, err)
	require.Equal(t, maintained1, gotApp)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, &team1.ID)
	require.NoError(t, err)
	maintained1.TitleID = ptr.Uint(titleID)
	require.Equal(t, maintained1, gotApp)

	//
	// Test including software title ID for existing apps (VPP)

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
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

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
	vppApp, err := ds.InsertVPPAppWithTeam(ctx, vppMaintained2, &team2.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

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
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained3.ID, &team1.ID)
	require.NoError(t, err)
	require.Equal(t, maintained3, gotApp)

	// right vpp app, right team
	_, err = ds.InsertVPPAppWithTeam(ctx, vppMaintained2, &team1.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	expectedApps[1].TitleID = ptr.Uint(vppApp.TitleID)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained2.ID, &team1.ID)
	require.NoError(t, err)
	maintained2.TitleID = ptr.Uint(vppApp.TitleID)
	require.Equal(t, maintained2, gotApp)

	// viewing with no team selected shouldn't include any title IDs
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, nil, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 3)
	require.EqualValues(t, meta.TotalResults, 3)
	expectedApps[0].TitleID = nil
	expectedApps[1].TitleID = nil
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, nil)
	require.NoError(t, err)
	maintained1.TitleID = nil
	require.Equal(t, maintained1, gotApp)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained3.ID, nil)
	require.NoError(t, err)
	maintained3.TitleID = nil
	require.Equal(t, maintained3, gotApp)
}

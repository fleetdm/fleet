package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"Sync", testSync},
		{"ListAndGetAvailableApps", testListAndGetAvailableApps},
		{"SyncAndRemoveApps", testSyncAndRemoveApps},
		{"GetMaintainedAppBySlug", testGetMaintainedAppBySlug},
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
			return sqlx.SelectContext(ctx, q, &apps, "SELECT name, platform, slug FROM fleet_maintained_apps ORDER BY slug")
		})
		return apps
	}

	expectedApps := maintained_apps.SyncApps(t, ds)
	var expectedAppsBaseInfo []fleet.MaintainedApp
	for _, app := range expectedApps {
		expectedAppsBaseInfo = append(expectedAppsBaseInfo, fleet.MaintainedApp{
			Name:     app.Name,
			Platform: app.Platform,
			Slug:     app.Slug,
		})
	}

	require.Equal(t, expectedAppsBaseInfo, listSavedApps())

	// ingesting again results in no changes
	maintained_apps.SyncApps(t, ds)
	require.Equal(t, expectedAppsBaseInfo, listSavedApps())

	// upsert the figma app, changing the version
	_, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:     "Figma 2",
		Slug:     "figma/darwin",
		Platform: "darwin",
	})
	require.NoError(t, err)

	// change the expected app data for figma
	for idx := range expectedAppsBaseInfo {
		if expectedAppsBaseInfo[idx].Slug == "figma/darwin" {
			expectedAppsBaseInfo[idx].Name = "Figma 2"
			break
		}
	}

	require.Equal(t, expectedAppsBaseInfo, listSavedApps())
}

func testSync(t *testing.T, ds *Datastore) {
	maintained_apps.SyncApps(t, ds)

	expectedSlugs := maintained_apps.ExpectedAppSlugs(t)
	var actualSlugs []string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &actualSlugs, "SELECT slug FROM fleet_maintained_apps ORDER BY slug")
	})
	require.ElementsMatch(t, expectedSlugs, actualSlugs)
}

func testListAndGetAvailableApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Zaphod Beeblebrox", "zaphod@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 2"})
	require.NoError(t, err)

	// Testing search that returns no results; nothing inserted yet case
	_, _, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.ErrorIs(t, err, &fleet.NoMaintainedAppsInDatabaseError{})

	maintained1, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})

	require.NoError(t, err)
	maintained2, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained2",
		Slug:             "maintained2",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained2",
	})
	require.NoError(t, err)
	maintained3, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained3",
		Slug:             "maintained3",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained3",
	})
	require.NoError(t, err)
	maintained4, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained4",
		Slug:             "maintained4",
		Platform:         "windows",
		UniqueIdentifier: "Maintained4 (MSI)",
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
			Slug:     "maintained1",
		},
		{
			ID:       maintained2.ID,
			Name:     maintained2.Name,
			Platform: maintained2.Platform,
			Slug:     "maintained2",
		},
		{
			ID:       maintained3.ID,
			Name:     maintained3.Name,
			Platform: maintained3.Platform,
			Slug:     "maintained3",
		},
		{
			ID:       maintained4.ID,
			Name:     maintained4.Name,
			Platform: maintained4.Platform,
			Slug:     "maintained4",
		},
	}

	// Testing pagination
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps, apps)
	require.False(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[:1], apps)
	require.True(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 1, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[1:2], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 2, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[2:3], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{PerPage: 1, Page: 3, IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[3:], apps)
	require.False(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	// Testing search
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{MatchQuery: "Maintained4", IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, meta.TotalResults)
	require.Equal(t, expectedApps[3:], apps)
	require.False(t, meta.HasNextResults)
	require.False(t, meta.HasPreviousResults)

	// Testing search that returns no results; non-error case
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{MatchQuery: "Maintained5", IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 0)
	require.EqualValues(t, 0, meta.TotalResults)
	require.False(t, meta.HasNextResults)
	require.False(t, meta.HasPreviousResults)

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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	expectedApps[0].TitleID = ptr.Uint(titleID)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, ptr.Uint(0))
	require.NoError(t, err)
	require.Equal(t, maintained1, gotApp)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained1.ID, &team1.ID)
	require.NoError(t, err)
	maintained1.TitleID = ptr.Uint(titleID)
	require.Equal(t, maintained1, gotApp)

	// we haven't added the windows app yet, so we shouldn't have a title ID for it
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Nil(t, apps[3].TitleID)

	// add Windows app
	_, windowsTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "Maintained4 (MSI)",
		TeamID:          &team1.ID,
		InstallScript:   "nothing",
		Filename:        "foo.msi",
		UserID:          user.ID,
		Platform:        "windows",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	expectedApps[3].TitleID = ptr.Uint(windowsTitleID)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained4.ID, &team1.ID)
	require.NoError(t, err)
	maintained4.TitleID = ptr.Uint(windowsTitleID)
	require.Equal(t, maintained4, gotApp)

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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
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
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained3.ID, &team1.ID)
	require.NoError(t, err)
	require.Equal(t, maintained3, gotApp)

	// right vpp app, right team
	_, err = ds.InsertVPPAppWithTeam(ctx, vppMaintained2, &team1.ID)
	require.NoError(t, err)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	expectedApps[1].TitleID = ptr.Uint(vppApp.TitleID)
	require.Equal(t, expectedApps, apps)

	gotApp, err = ds.GetMaintainedAppByID(ctx, maintained2.ID, &team1.ID)
	require.NoError(t, err)
	maintained2.TitleID = ptr.Uint(vppApp.TitleID)
	require.Equal(t, maintained2, gotApp)

	// viewing with no team selected shouldn't include any title IDs
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, nil, fleet.ListOptions{IncludeMetadata: true})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	expectedApps[0].TitleID = nil
	expectedApps[1].TitleID = nil
	expectedApps[3].TitleID = nil
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

func testSyncAndRemoveApps(t *testing.T, ds *Datastore) {
	maintained_apps.SyncAndRemoveApps(t, ds)
}

func testGetMaintainedAppBySlug(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 2"})
	require.NoError(t, err)
	user := test.NewUser(t, ds, "green banana", "yellow@banana.com", true)
	require.NoError(t, err)

	// maintained app 1
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)
	_, titleId1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            maintainedApp.Name,
		TeamID:           &team1.ID,
		InstallScript:    "echo Installing MaintainedAppForTeam1",
		Filename:         "maintained-app-team1.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: maintainedApp.UniqueIdentifier,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		URL:              "https://example.com/maintained-app-team1.pkg",
	})
	require.NoError(t, err)
	installer1, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team1.ID, titleId1, false)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/maintained-app-team1.pkg", installer1.URL)

	// maintained app 2
	maintainedApp2, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained2",
		Slug:             "maintained2",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained2",
	})
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            maintainedApp2.Name,
		TeamID:           &team2.ID,
		InstallScript:    "echo Installing MaintainedAppForTeam1",
		Filename:         "maintained-app-team2.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: maintainedApp2.UniqueIdentifier,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// get app 1 with no team specified
	gotApp, err := ds.GetMaintainedAppBySlug(ctx, "maintained1", nil)
	require.NoError(t, err)
	require.Equal(t, &fleet.MaintainedApp{
		ID:               maintainedApp.ID,
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
		TitleID:          nil,
	}, gotApp)

	// get app 1 with correct team specified
	gotApp, err = ds.GetMaintainedAppBySlug(ctx, "maintained1", &team1.ID)
	require.NoError(t, err)
	require.Equal(t, &fleet.MaintainedApp{
		ID:               maintainedApp.ID,
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
		TitleID:          &titleId1,
	}, gotApp)

	// get app 1 with team 2, so no title id exists
	gotApp, err = ds.GetMaintainedAppBySlug(ctx, "maintained1", &team2.ID)
	require.NoError(t, err)
	require.Equal(t, &fleet.MaintainedApp{
		ID:               maintainedApp.ID,
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
		TitleID:          nil,
	}, gotApp)
}

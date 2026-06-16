package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps/maintainedappstest"
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
		{"ListAvailableAppsByNameAndFilters", testListAvailableAppsByNameAndFilters},
		{"SyncAndRemoveApps", testSyncAndRemoveApps},
		{"GetMaintainedAppBySlug", testGetMaintainedAppBySlug},
		{"ListAvailableAppsWindows", testListAvailableAppsWindows},
		{"SoftwareTitleRenamingWindows", testSoftwareTitleRenamingWindows},
		{"GetFMANamesByIdentifier", testGetFMANamesByIdentifier},
		{"UpsertMaintainedAppUpdatesSoftware", testUpsertMaintainedAppUpdatesSoftware},
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

	expectedApps := maintainedappstest.SyncApps(t, ds)
	var expectedAppsBaseInfo []fleet.MaintainedApp
	for _, app := range expectedApps {
		expectedAppsBaseInfo = append(expectedAppsBaseInfo, fleet.MaintainedApp{
			Name:     app.Name,
			Platform: app.Platform,
			Slug:     app.Slug,
		})
	}

	require.ElementsMatch(t, expectedAppsBaseInfo, listSavedApps())

	// ingesting again results in no changes
	maintainedappstest.SyncApps(t, ds)
	require.ElementsMatch(t, expectedAppsBaseInfo, listSavedApps())

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

	require.ElementsMatch(t, expectedAppsBaseInfo, listSavedApps())
}

func testSync(t *testing.T, ds *Datastore) {
	maintainedappstest.SyncApps(t, ds)

	expectedSlugs := maintainedappstest.ExpectedAppSlugs(t)
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
	_, _, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 4)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps, apps)
	require.False(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[:1], apps)
	require.True(t, meta.HasNextResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, Page: 1, IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[1:2], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, Page: 2, IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[2:3], apps)
	require.True(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, Page: 3, IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, meta.TotalResults, 4)
	require.Equal(t, expectedApps[3:], apps)
	require.False(t, meta.HasNextResults)
	require.True(t, meta.HasPreviousResults)

	// Testing search
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{MatchQuery: "Maintained4", IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.EqualValues(t, 1, meta.TotalResults)
	require.Equal(t, expectedApps[3:], apps)
	require.False(t, meta.HasNextResults)
	require.False(t, meta.HasPreviousResults)

	// Testing search that returns no results; non-error case
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{MatchQuery: "Maintained5", IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, nil, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
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

	// Ordering: the combined-by-name view is only meaningfully sortable by name,
	// so "name" is the one allowed order key. expectedApps is declared in
	// ascending name order, so we derive the expected name sequences from it.
	appNames := func(apps []fleet.MaintainedApp) []string {
		got := make([]string, 0, len(apps))
		for _, a := range apps {
			got = append(got, a.Name)
		}
		return got
	}
	ascNames := appNames(expectedApps)
	descNames := make([]string, len(ascNames))
	for i, name := range ascNames {
		descNames[len(ascNames)-1-i] = name
	}

	t.Run("order_name_ascending", func(t *testing.T) {
		result, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderAscending, PerPage: 10, IncludeMetadata: true}})
		require.NoError(t, err)
		require.Equal(t, ascNames, appNames(result))
	})

	t.Run("order_name_descending", func(t *testing.T) {
		result, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderDescending, PerPage: 10, IncludeMetadata: true}})
		require.NoError(t, err)
		require.Equal(t, descNames, appNames(result))
	})

	t.Run("empty_order_key_defaults_to_name", func(t *testing.T) {
		result, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 10, IncludeMetadata: true}})
		require.NoError(t, err)
		require.Equal(t, ascNames, appNames(result))
	})

	// Only "name" is allowed. Keys that used to be in the allowlist (id,
	// platform, slug) and any other column must now be rejected, rather than
	// silently falling back to name ordering.
	for _, key := range []string{"id", "platform", "slug", "h.node_key"} {
		t.Run("rejects_"+key, func(t *testing.T) {
			_, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{OrderKey: key, IncludeMetadata: true}})
			require.Error(t, err)
		})
	}
}

func testSyncAndRemoveApps(t *testing.T, ds *Datastore) {
	maintainedappstest.SyncAndRemoveApps(t, ds)
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

func testListAvailableAppsWindows(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	maintained1, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "windows",
		UniqueIdentifier: "Maintained1 (MSI)",
	})
	require.NoError(t, err)
	maintained2, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained2",
		Slug:             "maintained2",
		Platform:         "darwin",
		UniqueIdentifier: "com.foo",
	})
	require.NoError(t, err)

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
	}
	apps, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.Nil(t, apps[0].TitleID)
	require.Nil(t, apps[1].TitleID)
	require.Equal(t, expectedApps, apps)

	// upload an installer that will create a title with a similar name, but with
	// an upgrade code so that unique identifier doesn't match
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Maintained1 (MSI)",
		UpgradeCode:          "{UPGRADE-CODE}",
		Source:               "programs",
		StorageID:            "storageid1",
		Filename:             "maintained1.msi",
		Extension:            "msi",
		Platform:             "windows",
		Version:              "1.0",
		UserID:               user.ID,
		TeamID:               &team1.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: ptr.Uint(maintained1.ID),
	})
	require.NoError(t, err)
	// create a pkg installer that should not match by similar name
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "Maintained2 ",
		BundleIdentifier: "Maintained2.ShallNotBeMatched",
		Source:           "apps",
		StorageID:        "storageid2",
		Filename:         "maintained2.pkg",
		Extension:        "pkg",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		TeamID:           &team1.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// the windows app should be found using using name, because the existing software title has an upgrade code
	apps, _, err = ds.ListAvailableFleetMaintainedApps(ctx, &team1.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.NotNil(t, apps[0].TitleID)
	require.Equal(t, titleID, *apps[0].TitleID)
	// the darwin app should not be matched by name
	require.Nil(t, apps[1].TitleID)
}

// testListAvailableAppsByNameAndFilters verifies that the list paginates by
// distinct app NAME (an app's macOS and Windows entries are combined into one
// logical app in the UI) while the total count is by distinct app row (each
// platform entry counted separately), and that the platform and available-only
// filters work server-side.
func testListAvailableAppsByNameAndFilters(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Filters"})
	require.NoError(t, err)
	user := test.NewUser(t, ds, "Filter Tester", "filters@example.com", true)

	mkApp := func(name, slug, platform, ident string) *fleet.MaintainedApp {
		app, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
			Name: name, Slug: slug, Platform: platform, UniqueIdentifier: ident,
		})
		require.NoError(t, err)
		return app
	}
	// Alpha and Delta exist on both platforms; Beta is macOS-only; Gamma is
	// Windows-only. That's 4 distinct apps across 6 rows.
	mkApp("Alpha", "alpha/darwin", "darwin", "com.example.alpha")
	mkApp("Alpha", "alpha/windows", "windows", "Alpha (MSI)")
	beta := mkApp("Beta", "beta/darwin", "darwin", "com.example.beta")
	mkApp("Gamma", "gamma/windows", "windows", "Gamma (MSI)")
	mkApp("Delta", "delta/darwin", "darwin", "com.example.delta")
	mkApp("Delta", "delta/windows", "windows", "Delta (MSI)")

	appNames := func(apps []fleet.MaintainedApp) []string {
		out := make([]string, len(apps))
		for i, a := range apps {
			out[i] = a.Name
		}
		return out
	}
	listOpts := func(o fleet.ListOptions) fleet.MaintainedAppListOptions {
		o.IncludeMetadata = true
		return fleet.MaintainedAppListOptions{ListOptions: o}
	}

	// Unfiltered: 6 apps (the count, one per platform entry) across 4 names, 6
	// rows returned.
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, listOpts(fleet.ListOptions{}))
	require.NoError(t, err)
	require.EqualValues(t, 6, meta.TotalResults)
	require.Len(t, apps, 6)
	require.False(t, meta.HasNextResults)

	// Pagination is by app name: a page of 2 names that includes a dual-platform
	// app returns ALL of that app's rows, so an app is never split across a page
	// boundary. Page 0 => Alpha (darwin+windows) + Beta (darwin) = 3 rows.
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, listOpts(fleet.ListOptions{PerPage: 2}))
	require.NoError(t, err)
	require.EqualValues(t, 6, meta.TotalResults)
	require.True(t, meta.HasNextResults)
	require.False(t, meta.HasPreviousResults)
	require.Equal(t, []string{"Alpha", "Alpha", "Beta"}, appNames(apps))

	// Page 1 => Delta (darwin+windows) + Gamma (windows) = 3 rows.
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, listOpts(fleet.ListOptions{PerPage: 2, Page: 1}))
	require.NoError(t, err)
	require.True(t, meta.HasPreviousResults)
	require.False(t, meta.HasNextResults)
	require.Equal(t, []string{"Delta", "Delta", "Gamma"}, appNames(apps))

	// Platform filter (darwin): keeps apps that have a macOS entry (Alpha, Beta,
	// Delta) and returns all of their rows so the UI can still show both columns.
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{Platform: "darwin", ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 3, meta.TotalResults)
	require.ElementsMatch(t, []string{"Alpha", "Alpha", "Beta", "Delta", "Delta"}, appNames(apps))

	// Platform filter (windows): keeps Alpha, Gamma, Delta.
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{Platform: "windows", ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 3, meta.TotalResults)
	require.ElementsMatch(t, []string{"Alpha", "Alpha", "Gamma", "Delta", "Delta"}, appNames(apps))

	// Add Beta (macOS-only) to the team so it is no longer "available".
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            beta.Name,
		TeamID:           &team.ID,
		InstallScript:    "nothing",
		Filename:         "beta.pkg",
		UserID:           user.ID,
		Platform:         string(fleet.MacOSPlatform),
		BundleIdentifier: beta.UniqueIdentifier,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Available-only hides Beta (its only platform is added) but keeps the other
	// three apps, which still have at least one not-yet-added platform. The count
	// is the 5 not-yet-added platform entries (Alpha macOS+Windows, Gamma
	// Windows, Delta macOS+Windows).
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{AvailableOnly: true, ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 5, meta.TotalResults)
	require.NotContains(t, appNames(apps), "Beta")

	// Without the filter, Beta is still listed (as added).
	apps, _, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, listOpts(fleet.ListOptions{}))
	require.NoError(t, err)
	require.Contains(t, appNames(apps), "Beta")

	// Platform and available-only combine: macOS apps not yet added are Alpha
	// and Delta (Beta's macOS entry is added; Gamma has no macOS entry).
	apps, meta, err = ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{Platform: "darwin", AvailableOnly: true, ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 2, meta.TotalResults)
	require.ElementsMatch(t, []string{"Alpha", "Alpha", "Delta", "Delta"}, appNames(apps))
}

func testSoftwareTitleRenamingWindows(t *testing.T, ds *Datastore) {

	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "Goodbye 1.00 (x64)", Version: "1.0", Source: "programs"},
		{Name: "Hello 1.00 (x64)", Version: "1.0", Source: "programs", UpgradeCode: ptr.String("{123456}")},
	}
	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	opts := fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{OrderKey: "name"}}
	sw, _, _, err := ds.ListSoftwareTitles(ctx, opts, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, sw, 2)
	require.Equal(t, "Goodbye 1.00 (x64)", sw[0].Name)
	require.Equal(t, "Hello 1.00 (x64)", sw[1].Name)

	maintained3, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "goodbye",
		Slug:             "goodbye/windows",
		Platform:         "windows",
		UniqueIdentifier: "Goodbye 1.00 (x64)",
	})
	require.NoError(t, err)
	maintained4, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Hello",
		Slug:             "hello/windows",
		Platform:         "windows",
		UniqueIdentifier: "Hello 1.00 (x64)",
	})
	require.NoError(t, err)

	sw, _, _, err = ds.ListSoftwareTitles(ctx, opts, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, sw, 2)
	require.Equal(t, "Goodbye 1.00 (x64)", sw[0].Name)
	require.Equal(t, "Hello 1.00 (x64)", sw[1].Name)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Goodbye 1.00 (x64)",
		Source:               "programs",
		StorageID:            "storageid1",
		Filename:             "goodbye.msi",
		Extension:            "msi",
		Platform:             "windows",
		Version:              "1.0",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintained3.ID),
	})
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Hello",
		UpgradeCode:          "{123456}",
		Source:               "programs",
		StorageID:            "storageid2",
		Filename:             "hello.msi",
		Extension:            "msi",
		Platform:             "windows",
		Version:              "1.0",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintained4.ID),
	})
	require.NoError(t, err)

	// After uploading installers, Goodbye 1.00 (x64) has no upgrade code so it
	// keeps its name, and Hello 1.00 (x64) updates to just Hello as it has one.
	sw, _, _, err = ds.ListSoftwareTitles(ctx, opts, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, sw, 2)
	require.Equal(t, "Goodbye 1.00 (x64)", sw[0].Name)
	require.Equal(t, "Hello", sw[1].Name)
}

func testUpsertMaintainedAppUpdatesSoftware(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create a host to associate software with
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host",
		Platform:        "darwin",
		OsqueryHostID:   ptr.String("osquery-host-id"),
		NodeKey:         ptr.String("node-key"),
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		PolicyUpdatedAt: ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
	})
	require.NoError(t, err)

	// Create software entries with osquery-reported name ("Code" instead of "Microsoft Visual Studio Code")
	software := []fleet.Software{
		{
			Name:             "Code",
			Version:          "1.85.0",
			Source:           "apps",
			BundleIdentifier: "com.microsoft.VSCode",
		},
		{
			Name:             "Code",
			Version:          "1.84.0",
			Source:           "apps",
			BundleIdentifier: "com.microsoft.VSCode",
		},
	}

	// Insert software using the normal ingestion path
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// Verify the software and software_titles were created with the osquery name "Code"
	var softwareNames []string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &softwareNames,
			`SELECT name FROM software WHERE bundle_identifier = 'com.microsoft.VSCode' ORDER BY version`)
	})
	require.Len(t, softwareNames, 2)
	require.Equal(t, "Code", softwareNames[0])
	require.Equal(t, "Code", softwareNames[1])

	var titleName string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleName,
			`SELECT name FROM software_titles WHERE bundle_identifier = 'com.microsoft.VSCode'`)
	})
	require.Equal(t, "Code", titleName)

	// Now upsert an FMA with the canonical name
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	// Verify software entries were updated to use the FMA canonical name
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &softwareNames,
			`SELECT name FROM software WHERE bundle_identifier = 'com.microsoft.VSCode' ORDER BY version`)
	})
	require.Len(t, softwareNames, 2)
	require.Equal(t, "Microsoft Visual Studio Code", softwareNames[0])
	require.Equal(t, "Microsoft Visual Studio Code", softwareNames[1])

	// Verify software_titles was also updated
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleName,
			`SELECT name FROM software_titles WHERE bundle_identifier = 'com.microsoft.VSCode'`)
	})
	require.Equal(t, "Microsoft Visual Studio Code", titleName)

	// Verify upserting the same FMA again doesn't cause issues (idempotent)
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	// Names should still be the FMA canonical name
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &softwareNames,
			`SELECT name FROM software WHERE bundle_identifier = 'com.microsoft.VSCode' ORDER BY version`)
	})
	require.Len(t, softwareNames, 2)
	require.Equal(t, "Microsoft Visual Studio Code", softwareNames[0])
	require.Equal(t, "Microsoft Visual Studio Code", softwareNames[1])

	// Verify Windows FMA does NOT update darwin software entries
	// First create darwin software with a different bundle_id
	software2 := []fleet.Software{
		{
			Name:             "Some App",
			Version:          "1.0.0",
			Source:           "apps",
			BundleIdentifier: "com.example.someapp",
		},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, append(software, software2...))
	require.NoError(t, err)

	// Upsert a Windows FMA - should not affect darwin software
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Some App Windows",
		Slug:             "some-app/windows",
		Platform:         "windows",
		UniqueIdentifier: "com.example.someapp", // Same identifier but different platform
	})
	require.NoError(t, err)

	// The darwin software should NOT have been renamed
	var someAppName string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &someAppName,
			`SELECT name FROM software WHERE bundle_identifier = 'com.example.someapp'`)
	})
	require.Equal(t, "Some App", someAppName)
}

func testGetFMANamesByIdentifier(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Initially empty
	names, err := ds.GetFMANamesByIdentifier(ctx)
	require.NoError(t, err)
	require.Empty(t, names)

	// Add some darwin FMAs
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "1Password",
		Slug:             "1password/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.1password.1password",
	})
	require.NoError(t, err)

	// Add a Windows FMA - should NOT be returned (only darwin)
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/windows",
		Platform:         "windows",
		UniqueIdentifier: "Microsoft Visual Studio Code",
	})
	require.NoError(t, err)

	// Get FMA names - should only return darwin apps
	names, err = ds.GetFMANamesByIdentifier(ctx)
	require.NoError(t, err)
	require.Len(t, names, 2)
	require.Equal(t, "Microsoft Visual Studio Code", names["com.microsoft.VSCode"])
	require.Equal(t, "1Password", names["com.1password.1password"])

	// Windows identifier should not be present
	_, ok := names["Microsoft Visual Studio Code"]
	require.False(t, ok)
}

package mysql

import (
	"context"
	"fmt"
	"slices"
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
		{"ListAvailableAppsSharedName", testListAvailableAppsSharedName},
		{"SyncAndRemoveApps", testSyncAndRemoveApps},
		{"GetMaintainedAppBySlug", testGetMaintainedAppBySlug},
		{"ListAvailableAppsWindows", testListAvailableAppsWindows},
		{"SoftwareTitleRenamingWindows", testSoftwareTitleRenamingWindows},
		{"GetFMANamesByIdentifier", testGetFMANamesByIdentifier},
		{"GetWindowsFMAMatches", testGetWindowsFMAMatches},
		{"WindowsFMANameOnIngest", testWindowsFMANameOnIngest},
		{"ReconcileWindowsSoftwareTitles", testReconcileWindowsSoftwareTitles},
		{"WindowsFMAMatchByUniqueIdentifier", testWindowsFMAMatchByUniqueIdentifier},
		{"WindowsFMAMatchByNameWhenIdentifierStale", testWindowsFMAMatchByNameWhenIdentifierStale},
		{"WindowsFMANotRenamedWithUpgradeCode", testWindowsFMANotRenamedWithUpgradeCode},
		{"WindowsFMANoCollapseWithoutInstaller", testWindowsFMANoCollapseWithoutInstaller},
		{"WindowsFMAAmbiguousMatchIsNoOp", testWindowsFMAAmbiguousMatchIsNoOp},
		{"WindowsFMAReconcileMovesTitleReferences", testWindowsFMAReconcileMovesTitleReferences},
		{"WindowsFMAReconcilePinConflict", testWindowsFMAReconcilePinConflict},
		{"WindowsFMAReconcileSameNameUpgradeCodeTitle", testWindowsFMAReconcileSameNameUpgradeCodeTitle},
		{"WindowsFMAReconcileAfterCatalogRename", testWindowsFMAReconcileAfterCatalogRename},
		{"WindowsFMAIngestAfterCatalogRename", testWindowsFMAIngestAfterCatalogRename},
		{"WindowsFMAUninstallActionAvailable", testWindowsFMAUninstallActionAvailable},
		{"WindowsFMAMultiTeamInstallersShareTitle", testWindowsFMAMultiTeamInstallersShareTitle},
		{"WindowsFMAExcludedWhenSpanningTitles", testWindowsFMAExcludedWhenSpanningTitles},
		{"WindowsFMAIgnoresInstallerWithoutTitle", testWindowsFMAIgnoresInstallerWithoutTitle},
		{"WindowsFMANameWithLikeWildcards", testWindowsFMANameWithLikeWildcards},
		{"WindowsFMAMatchesCache", testWindowsFMAMatchesCache},
		{"WindowsFMAMergeWithoutPrefixes", testWindowsFMAMergeWithoutPrefixes},
		{"WindowsFMAMergeMovesAllReferences", testWindowsFMAMergeMovesAllReferences},
		{"WindowsFMAMergeMultipleDestinations", testWindowsFMAMergeMultipleDestinations},
		{"WindowsFMAIngestIgnoresOtherSources", testWindowsFMAIngestIgnoresOtherSources},
		{"WindowsFMAReconcileIndependentOfCatalogSync", testWindowsFMAReconcileIndependentOfCatalogSync},
		{"ReconcileSoftwareNames", testReconcileSoftwareNames},
		{"ReconcileSoftwareNamesSharedIdentifier", testReconcileSoftwareNamesSharedIdentifier},
		{"ReconcileSoftwareNamesBatched", testReconcileSoftwareNamesBatched},
		{"ReconcileSoftwareNamesDiscoveryWindowed", testReconcileSoftwareNamesDiscoveryWindowed},
		{"ReconcileSoftwareNamesOrphanedInstaller", testReconcileSoftwareNamesOrphanedInstaller},
		{"ReconcileSoftwareNamesIdentifierWinsOverInstallerLink", testReconcileSoftwareNamesIdentifierWinsOverInstallerLink},
		{"ReconcileSoftwareNamesMultiTeamInstallers", testReconcileSoftwareNamesMultiTeamInstallers},
		{"ReconcileSoftwareNamesLeavesMobileSiblings", testReconcileSoftwareNamesLeavesMobileSiblings},
		{"ListAvailableAppsSharedIdentifier", testListAvailableAppsSharedIdentifier},
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

	// Ordering: the combined-by-app view is only meaningfully sortable by name,
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
// distinct app TOKEN (the slug prefix), so an app's macOS and Windows entries
// are combined into one row and never split across a page boundary, while the
// count is the number of installable platform entries (each Add button counts
// once), and that the platform and available-only filters work server-side.
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

	// Unfiltered: 6 installable platform entries (the count: alpha+delta each ship
	// on two platforms) across 4 app tokens, 6 rows returned (the raw per-platform
	// entries the UI combines into 4 rows).
	apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, listOpts(fleet.ListOptions{}))
	require.NoError(t, err)
	require.EqualValues(t, 6, meta.TotalResults)
	require.Len(t, apps, 6)
	require.False(t, meta.HasNextResults)

	// Pagination is by app token: a page of 2 apps that includes a dual-platform
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
	// is the 5 not-yet-added platform entries (Alpha macOS+Windows, Gamma Windows,
	// Delta macOS+Windows).
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

// testListAvailableAppsSharedName verifies that two distinct apps sharing a
// display name (e.g. "Gemini": gemini/darwin and google-gemini/darwin) are
// counted and listed as two separate apps. Keying on the slug token keeps them
// distinct, so the count matches the row count and neither app is hidden.
func testListAvailableAppsSharedName(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Shared Name"})
	require.NoError(t, err)

	mkApp := func(name, slug, platform, ident string) {
		_, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
			Name: name, Slug: slug, Platform: platform, UniqueIdentifier: ident,
		})
		require.NoError(t, err)
	}
	// Two different macOS apps that share the display name "Gemini".
	mkApp("Gemini", "gemini/darwin", "darwin", "com.macpaw.site.Gemini2")
	mkApp("Gemini", "google-gemini/darwin", "darwin", "com.google.GeminiMacOS")

	assertTwoGeminis := func(opt fleet.MaintainedAppListOptions) {
		apps, meta, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, opt)
		require.NoError(t, err)
		// Both apps counted (not collapsed by shared name) ...
		require.EqualValues(t, 2, meta.TotalResults)
		// ... and both rows returned, with distinct slugs.
		require.Len(t, apps, 2)
		slugs := []string{apps[0].Slug, apps[1].Slug}
		require.ElementsMatch(t, []string{"gemini/darwin", "google-gemini/darwin"}, slugs)
		require.Equal(t, "Gemini", apps[0].Name)
		require.Equal(t, "Gemini", apps[1].Name)
	}

	// Count must equal rows unfiltered and with the macOS platform filter.
	assertTwoGeminis(fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{IncludeMetadata: true}})
	assertTwoGeminis(fleet.MaintainedAppListOptions{Platform: "darwin", ListOptions: fleet.ListOptions{IncludeMetadata: true}})

	// Paginating one app at a time yields each Gemini on its own page, never
	// splitting or dropping one.
	page0, meta0, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 2, meta0.TotalResults)
	require.Len(t, page0, 1)
	require.True(t, meta0.HasNextResults)

	page1, meta1, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{ListOptions: fleet.ListOptions{PerPage: 1, Page: 1, IncludeMetadata: true}})
	require.NoError(t, err)
	require.EqualValues(t, 2, meta1.TotalResults)
	require.Len(t, page1, 1)
	require.False(t, meta1.HasNextResults)
	require.True(t, meta1.HasPreviousResults)
	// The two pages cover the two distinct apps.
	require.NotEqual(t, page0[0].Slug, page1[0].Slug)
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
		FleetMaintainedAppID: &maintained3.ID,
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
		FleetMaintainedAppID: &maintained4.ID,
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

func testReconcileSoftwareNames(t *testing.T, ds *Datastore) {
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
	require.Equal(t, []string{"Code", "Code"}, softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))

	// Now upsert an FMA with the canonical name
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	// The upsert itself must not rename; confirm the pre-reconcile state still has
	// the osquery name in both the software and software_titles tables.
	require.Equal(t, []string{"Code", "Code"}, softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))

	// The upsert doesn't rename; the reconcile pass does (unambiguous identifier).
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	// Verify software entries and the title were updated to the FMA canonical name
	require.Equal(t, []string{"Microsoft Visual Studio Code", "Microsoft Visual Studio Code"},
		softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Microsoft Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))

	// Verify upserting the same FMA again and reconciling doesn't cause issues (idempotent)
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	// Names should still be the FMA canonical name
	require.Equal(t, []string{"Microsoft Visual Studio Code", "Microsoft Visual Studio Code"},
		softwareNames(t, ds, "com.microsoft.VSCode"))

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
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	// A Windows FMA must not rename darwin software.
	require.Equal(t, []string{"Some App"}, softwareNames(t, ds, "com.example.someapp"))
}

// testReconcileSoftwareNamesSharedIdentifier: two FMAs sharing a bundle identifier
// (Firefox / Firefox ESR). Reconcile must not guess a name from the identifier
// alone, but must use the specific FMA a title was added with.
func testReconcileSoftwareNamesSharedIdentifier(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Ford Prefect", "ford@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Firefox"})
	require.NoError(t, err)

	host := newTestHostWithPlatform(t, ds, "firefox-host", "darwin", nil)

	// Two FMAs that share the same macOS bundle identifier.
	firefox, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox",
		Slug:             "firefox/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox ESR",
		Slug:             "firefox@esr/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)

	// A host reports Firefox via osquery (name "Firefox.app"), with no FMA added.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Firefox.app", Version: "120.0", Source: "apps", BundleIdentifier: "org.mozilla.firefox"},
	})
	require.NoError(t, err)

	// Ingestion must not guess a name for the shared identifier.
	require.Equal(t, "Firefox.app", softwareTitleName(t, ds, "org.mozilla.firefox"))

	// No FMA link and an ambiguous identifier: reconcile must leave it alone.
	// (Regression: the title used to flip to "Mozilla Firefox ESR" by sync order.)
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, "Firefox.app", softwareTitleName(t, ds, "org.mozilla.firefox"))
	// Reconcile updates the software table with a separate statement, so assert it
	// too: with an ambiguous identifier and no FMA link, that row is left alone.
	require.Equal(t, []string{"Firefox.app"}, softwareNames(t, ds, "org.mozilla.firefox"))

	// Add the Firefox (non-ESR) FMA, linking the existing title to a specific FMA.
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Mozilla Firefox",
		TeamID:               &team.ID,
		Source:               "apps",
		InstallScript:        "nothing",
		Filename:             "Firefox.dmg",
		UserID:               user.ID,
		Platform:             string(fleet.MacOSPlatform),
		BundleIdentifier:     "org.mozilla.firefox",
		FleetMaintainedAppID: &firefox.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// The install reuses the existing title, so the name is unchanged until reconcile.
	require.Equal(t, "Firefox.app", softwareTitleName(t, ds, "org.mozilla.firefox"))

	// Reconcile resolves the ambiguity via the installer link.
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, "Mozilla Firefox", softwareTitleName(t, ds, "org.mozilla.firefox"))

	var softwareName string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &softwareName,
			`SELECT name FROM software WHERE title_id = ? AND bundle_identifier = 'org.mozilla.firefox'`, titleID)
	})
	require.Equal(t, "Mozilla Firefox", softwareName)
}

// testReconcileSoftwareNamesBatched: reconcile renames in bounded batches to keep
// InnoDB row locks short-lived, so it must keep walking until every matching row is
// renamed, and it must not touch rows outside the match set.
func testReconcileSoftwareNamesBatched(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Force several batches over a handful of rows.
	oldBatchSize := maintainedAppNameReconcileBatchSize
	maintainedAppNameReconcileBatchSize = 2
	t.Cleanup(func() { maintainedAppNameReconcileBatchSize = oldBatchSize })

	user := test.NewUser(t, ds, "Zaphod Beeblebrox", "zaphod@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Batched"})
	require.NoError(t, err)

	host := newTestHostWithPlatform(t, ds, "batched-host", "darwin", nil)

	// Seven versions of one app, so the rename spans four batches of two. Plus an
	// unrelated app with no FMA, which must never be renamed.
	var software []fleet.Software
	for _, version := range []string{"1.0", "2.0", "3.0", "4.0", "5.0", "6.0", "7.0"} {
		software = append(software, fleet.Software{
			Name: "Code", Version: version, Source: "apps", BundleIdentifier: "com.microsoft.VSCode",
		})
	}
	software = append(software, fleet.Software{
		Name: "Unrelated", Version: "1.0", Source: "apps", BundleIdentifier: "com.example.unrelated",
	})
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	require.Equal(t, slices.Repeat([]string{"Code"}, 7), softwareNames(t, ds, "com.microsoft.VSCode"))

	vscode, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	// Pass 2 (by unambiguous bundle identifier) must rename all seven rows, not just
	// the first batch.
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, slices.Repeat([]string{"Microsoft Visual Studio Code"}, 7),
		softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Microsoft Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))

	// The app with no FMA is outside the match set and must be untouched.
	require.Equal(t, []string{"Unrelated"}, softwareNames(t, ds, "com.example.unrelated"))
	require.Equal(t, "Unrelated", softwareTitleName(t, ds, "com.example.unrelated"))

	// Link the title to the FMA via an installer and change the canonical name, so
	// pass 1 (by installer link) has to walk all seven rows too.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Microsoft Visual Studio Code",
		TeamID:               &team.ID,
		Source:               "apps",
		InstallScript:        "nothing",
		Filename:             "VSCode.dmg",
		UserID:               user.ID,
		Platform:             string(fleet.MacOSPlatform),
		BundleIdentifier:     "com.microsoft.VSCode",
		FleetMaintainedAppID: &vscode.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, slices.Repeat([]string{"Visual Studio Code"}, 7),
		softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))

	// Idempotent: a second run with everything already canonical is a no-op.
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, "Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, []string{"Unrelated"}, softwareNames(t, ds, "com.example.unrelated"))
}

// testReconcileSoftwareNamesDiscoveryWindowed: each discovery SELECT is capped by
// maintainedAppNameReconcileDiscoveryLimit so the pass never holds every mismatched row
// in memory at once. Renamed rows drop out of the next SELECT, so the pass must keep
// re-discovering until a window comes back short, not stop after the first one.
func testReconcileSoftwareNamesDiscoveryWindowed(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Seven mismatched software rows against a window of three and batches of two, so
	// the pass needs several windows and the window and batch edges never line up.
	oldBatchSize := maintainedAppNameReconcileBatchSize
	oldDiscoveryLimit := maintainedAppNameReconcileDiscoveryLimit
	maintainedAppNameReconcileBatchSize = 2
	maintainedAppNameReconcileDiscoveryLimit = 3
	t.Cleanup(func() {
		maintainedAppNameReconcileBatchSize = oldBatchSize
		maintainedAppNameReconcileDiscoveryLimit = oldDiscoveryLimit
	})

	host := newTestHostWithPlatform(t, ds, "windowed-host", "darwin", nil)

	var software []fleet.Software
	for _, version := range []string{"1.0", "2.0", "3.0", "4.0", "5.0", "6.0", "7.0"} {
		software = append(software, fleet.Software{
			Name: "Code", Version: version, Source: "apps", BundleIdentifier: "com.microsoft.VSCode",
		})
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	upsertDarwinFMA(t, ds, "Microsoft Visual Studio Code", "com.microsoft.VSCode", "visual-studio-code/darwin")

	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	// Every row, not just the distinct set, so a window that stopped early is caught.
	require.Equal(t, slices.Repeat([]string{"Microsoft Visual Studio Code"}, 7),
		softwareNames(t, ds, "com.microsoft.VSCode"))
	require.Equal(t, "Microsoft Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))
}

// softwareTitleName returns the name of the macOS software title carrying bundleID.
// additional_identifier is 0 for macOS titles, which keeps this to a single row even when iOS
// or iPadOS siblings share the identifier.
func softwareTitleName(t *testing.T, ds *Datastore, bundleID string) string {
	var name string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &name,
			`SELECT name FROM software_titles WHERE bundle_identifier = ? AND additional_identifier = 0`, bundleID)
	})
	return name
}

// softwareNames returns the names of every software row carrying bundleID, oldest first.
func softwareNames(t *testing.T, ds *Datastore, bundleID string) []string {
	var names []string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(t.Context(), q, &names,
			`SELECT name FROM software WHERE bundle_identifier = ? ORDER BY id`, bundleID)
	})
	return names
}

// upsertDarwinFMA upserts a darwin maintained app. Call it once per app: the upsert's
// ON DUPLICATE KEY UPDATE reports no insert id, so a repeat call returns ID 0.
func upsertDarwinFMA(t *testing.T, ds *Datastore, appName, bundleID, slug string) *fleet.MaintainedApp {
	app, err := ds.UpsertMaintainedApp(t.Context(), &fleet.MaintainedApp{
		Name:             appName,
		Slug:             slug,
		Platform:         "darwin",
		UniqueIdentifier: bundleID,
	})
	require.NoError(t, err)
	require.NotZero(t, app.ID)
	return app
}

// addDarwinFMAInstaller adds app to a team as an installer, linked to the software title
// carrying titleBundleID. It returns that title's ID.
func addDarwinFMAInstaller(t *testing.T, ds *Datastore, userID uint, teamID *uint, app *fleet.MaintainedApp, titleBundleID string) uint {
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(t.Context(), &fleet.UploadSoftwareInstallerPayload{
		Title:                app.Name,
		TeamID:               teamID,
		Source:               "apps",
		InstallScript:        "nothing",
		Filename:             app.Name + ".dmg",
		UserID:               userID,
		Platform:             string(fleet.MacOSPlatform),
		BundleIdentifier:     titleBundleID,
		FleetMaintainedAppID: &app.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	return titleID
}

// testReconcileSoftwareNamesOrphanedInstaller: an installer whose title was deleted has a
// NULL title_id. The installer-link pass must skip it -- its catalog subquery filters out
// the NULL group rather than carrying a group no title can join -- while the
// bundle-identifier pass still applies the canonical name.
func testReconcileSoftwareNamesOrphanedInstaller(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Trillian", "trillian@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Orphan"})
	require.NoError(t, err)

	host := newTestHostWithPlatform(t, ds, "orphan-host", "darwin", nil)

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Code", Version: "1.0", Source: "apps", BundleIdentifier: "com.microsoft.VSCode"},
	})
	require.NoError(t, err)

	vscode := upsertDarwinFMA(t, ds, "Microsoft Visual Studio Code", "com.microsoft.VSCode", "visual-studio-code/darwin")
	addDarwinFMAInstaller(t, ds, user.ID, &team.ID, vscode, "com.microsoft.VSCode")

	// Orphan the installer the way CleanupSoftwareTitles does, via the FK's ON DELETE SET NULL.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET title_id = NULL`)
		return err
	})

	// Must not error. The bundle-identifier pass still applies the canonical name.
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	require.Equal(t, "Microsoft Visual Studio Code", softwareTitleName(t, ds, "com.microsoft.VSCode"))
}

// testReconcileSoftwareNamesIdentifierWinsOverInstallerLink pins the precedence between
// the two passes. The installer link runs first, then the unambiguous bundle identifier,
// so when they disagree the identifier is what lands.
func testReconcileSoftwareNamesIdentifierWinsOverInstallerLink(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Slartibartfast", "slarti@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team Precedence"})
	require.NoError(t, err)

	// An app whose identifier is com.example.identifier, so that identifier maps
	// unambiguously to "Identifier Name".
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Identifier Name",
		Slug:             "identifier-name/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.example.identifier",
	})
	require.NoError(t, err)

	// A different app, linked by installer to a title that carries the identifier above.
	other := upsertDarwinFMA(t, ds, "Installer Link Name", "com.example.other", "installer-link/darwin")
	titleID := addDarwinFMAInstaller(t, ds, user.ID, &team.ID, other, "com.example.identifier")

	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	var name string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &name, `SELECT name FROM software_titles WHERE id = ?`, titleID)
	})
	require.Equal(t, "Identifier Name", name, "the bundle-identifier pass runs last and wins")
}

// testReconcileSoftwareNamesMultiTeamInstallers: one app added to several teams has an
// installer row per team, all pointing at the same title. The GROUP BY must collapse them
// so the title is not treated as ambiguous.
func testReconcileSoftwareNamesMultiTeamInstallers(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Marvin", "marvin@example.com", true)
	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team A"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team B"})
	require.NoError(t, err)

	host := newTestHostWithPlatform(t, ds, "multi-team-host", "darwin", nil)

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Code", Version: "1.0", Source: "apps", BundleIdentifier: "com.microsoft.VSCode"},
	})
	require.NoError(t, err)

	vscode := upsertDarwinFMA(t, ds, "Microsoft Visual Studio Code", "com.microsoft.VSCode", "visual-studio-code/darwin")
	titleA := addDarwinFMAInstaller(t, ds, user.ID, &teamA.ID, vscode, "com.microsoft.VSCode")
	titleB := addDarwinFMAInstaller(t, ds, user.ID, &teamB.ID, vscode, "com.microsoft.VSCode")
	require.Equal(t, titleA, titleB, "per-team installers must share one title")

	var installerCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerCount,
			`SELECT COUNT(*) FROM software_installers WHERE title_id = ?`, titleA)
	})
	require.Equal(t, 2, installerCount)

	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	require.Equal(t, []string{"Microsoft Visual Studio Code"}, softwareNames(t, ds, "com.microsoft.VSCode"))
}

// testReconcileSoftwareNamesLeavesMobileSiblings: iOS and iPadOS titles can share a bundle
// identifier with a macOS app but are separate products, so a macOS app's canonical name
// must not be pushed onto them.
func testReconcileSoftwareNamesLeavesMobileSiblings(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host := newTestHostWithPlatform(t, ds, "sibling-host", "darwin", nil)

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Code", Version: "1.0", Source: "apps", BundleIdentifier: "com.microsoft.VSCode"},
	})
	require.NoError(t, err)

	// iOS and iPadOS titles sharing the identifier, as VPP or in-house apps produce.
	// additional_identifier is generated from source, so these coexist with the macOS row.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		for _, source := range []string{"ios_apps", "ipados_apps"} {
			res, err := q.ExecContext(ctx,
				`INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, 'com.microsoft.VSCode')`,
				"Code Mobile", source)
			if err != nil {
				return err
			}
			titleID, err := res.LastInsertId()
			if err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx,
				`INSERT INTO software (name, version, source, bundle_identifier, title_id, checksum)
				 VALUES (?, '1.0', ?, 'com.microsoft.VSCode', ?, UNHEX(MD5(?)))`,
				"Code Mobile", source, titleID, source); err != nil {
				return err
			}
		}
		return nil
	})

	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Microsoft Visual Studio Code",
		Slug:             "visual-studio-code/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.microsoft.VSCode",
	})
	require.NoError(t, err)

	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))

	namesBySource := func(table string) map[string]string {
		out := map[string]string{}
		var rows []struct {
			Source string `db:"source"`
			Name   string `db:"name"`
		}
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				`SELECT source, name FROM `+table+` WHERE bundle_identifier = 'com.microsoft.VSCode'`)
		})
		for _, r := range rows {
			out[r.Source] = r.Name
		}
		return out
	}

	titles := namesBySource("software_titles")
	require.Equal(t, "Microsoft Visual Studio Code", titles["apps"])
	require.Equal(t, "Code Mobile", titles["ios_apps"])
	require.Equal(t, "Code Mobile", titles["ipados_apps"])

	software := namesBySource("software")
	require.Equal(t, "Microsoft Visual Studio Code", software["apps"])
	require.Equal(t, "Code Mobile", software["ios_apps"])
	require.Equal(t, "Code Mobile", software["ipados_apps"])
}

// testListAvailableAppsSharedIdentifier: adding Firefox must not mark its
// bundle-identifier sibling Firefox ESR as added.
func testListAvailableAppsSharedIdentifier(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Arthur Dent", "arthur@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 42"})
	require.NoError(t, err)

	firefox, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox",
		Slug:             "firefox/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)
	esr, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox ESR",
		Slug:             "firefox@esr/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)

	titleIDFor := func(appID uint) *uint {
		apps, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID, fleet.MaintainedAppListOptions{})
		require.NoError(t, err)
		for _, a := range apps {
			if a.ID == appID {
				return a.TitleID
			}
		}
		t.Fatalf("app %d not found in list", appID)
		return nil
	}

	// Before adding anything, neither shows as added.
	require.Nil(t, titleIDFor(firefox.ID))
	require.Nil(t, titleIDFor(esr.ID))

	// Add the Firefox (non-ESR) FMA, linked via fleet_maintained_app_id.
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Mozilla Firefox",
		TeamID:               &team.ID,
		Source:               "apps",
		InstallScript:        "nothing",
		Filename:             "Firefox.dmg",
		UserID:               user.ID,
		Platform:             string(fleet.MacOSPlatform),
		BundleIdentifier:     "org.mozilla.firefox",
		FleetMaintainedAppID: &firefox.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Firefox is added; ESR must remain available despite the shared identifier.
	require.Equal(t, &titleID, titleIDFor(firefox.ID))
	require.Nil(t, titleIDFor(esr.ID))

	// The "available only" filter must still surface ESR but hide Firefox.
	availApps, _, err := ds.ListAvailableFleetMaintainedApps(ctx, &team.ID,
		fleet.MaintainedAppListOptions{AvailableOnly: true})
	require.NoError(t, err)
	var slugs []string
	for _, a := range availApps {
		slugs = append(slugs, a.Slug)
	}
	require.Contains(t, slugs, "firefox@esr/darwin")
	require.NotContains(t, slugs, "firefox/darwin")
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

	// Two FMAs sharing a bundle identifier (Firefox/ESR) must be omitted, not
	// resolved to whichever was inserted last.
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox",
		Slug:             "firefox/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Mozilla Firefox ESR",
		Slug:             "firefox@esr/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "org.mozilla.firefox",
	})
	require.NoError(t, err)

	names, err = ds.GetFMANamesByIdentifier(ctx)
	require.NoError(t, err)
	require.Len(t, names, 2) // still only the two unambiguous identifiers
	_, ok = names["org.mozilla.firefox"]
	require.False(t, ok)
}

func testGetWindowsFMAMatches(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Initially empty
	names, err := ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Empty(t, names)

	// A catalog entry alone is not enough: only FMAs added via an installer are
	// returned, since that deliberate install is what bounds name-prefix matching.
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Obsidian",
		Slug:             "obsidian/windows",
		Platform:         "windows",
		UniqueIdentifier: "Obsidian",
	})
	require.NoError(t, err)

	names, err = ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Empty(t, names, "catalog entry with no installer must not be returned")

	// name == unique_identifier, the common case.
	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	// unique_identifier differs from the name, and is what osquery reports.
	addWindowsFMAWithInstaller(t, ds, user.ID, "CPU-Z", "CPUID CPU-Z", "cpu-z/windows")

	// A darwin FMA with an installer must not be returned.
	darwinApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Granola",
		Slug:             "granola/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.granola.app",
	})
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Granola",
		Source:               "apps",
		StorageID:            "storageid-granola-darwin",
		Filename:             "granola.pkg",
		Extension:            "pkg",
		Platform:             "darwin",
		Version:              "1.0.0",
		BundleIdentifier:     "com.granola.app",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: &darwinApp.ID,
	})
	require.NoError(t, err)

	names, err = ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	byName := make(map[string]*fleet.MaintainedApp, len(names))
	for i := range names {
		byName[names[i].Name] = &names[i]
	}
	require.Len(t, byName, 2)
	require.Equal(t, "Granola", byName["Granola"].UniqueIdentifier)
	require.Equal(t, "CPUID CPU-Z", byName["CPU-Z"].UniqueIdentifier)
	// Platform must be selected, since WinMatchPrefixes gates on it.
	require.Equal(t, "windows", byName["Granola"].Platform)

	// All name fields are offered as match candidates, longest first.
	require.Equal(t, []string{"CPUID CPU-Z", "CPU-Z"}, byName["CPU-Z"].WinMatchPrefixes())
	// Deduplicated when they are the same.
	require.Equal(t, []string{"Granola"}, byName["Granola"].WinMatchPrefixes())

	// A darwin app yields no prefixes even if the other fields are populated.
	darwin := fleet.MaintainedApp{
		Name: "Granola", UniqueIdentifier: "com.granola.app",
		Platform: "darwin", TitleID: new(uint(1)), TitleName: "Granola",
	}
	require.Empty(t, darwin.WinMatchPrefixes())
	// So does a Windows app with no resolved title.
	noTitle := fleet.MaintainedApp{Name: "Granola", UniqueIdentifier: "Granola", Platform: "windows"}
	require.Empty(t, noTitle.WinMatchPrefixes())
}

// testWindowsFMANameOnIngest: with a Windows FMA present, ingesting a versioned
// program name links onto the canonical FMA title instead of creating a new one.
func testWindowsFMANameOnIngest(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// FMA + installer create the canonical "Granola" title first.
	maintained, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Granola",
		Slug:             "granola/windows",
		Platform:         "windows",
		UniqueIdentifier: "Granola",
	})
	require.NoError(t, err)
	// A "Zoom" FMA to exercise the negative (non-)match case below.
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Zoom",
		Slug:             "zoom/windows",
		Platform:         "windows",
		UniqueIdentifier: "Zoom",
	})
	require.NoError(t, err)

	_, canonicalTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "Granola",
		Source:               "programs",
		StorageID:            "storageid1",
		Filename:             "granola.exe",
		Extension:            "exe",
		Platform:             "windows",
		Version:              "7.373.2",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: &maintained.ID,
	})
	require.NoError(t, err)

	// Ingest host inventory with the version in the name, plus an unrelated app
	// that merely shares a prefix with the "Zoom" FMA (no trailing space -> no match).
	software := []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
		{Name: "Zoombie 5.0", Version: "5.0", Source: "programs"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// "Granola 7.373.2" inventory links to the canonical "Granola" title...
	var granolaTitleID uint
	var granolaSWName string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &granolaTitleID,
			`SELECT title_id FROM software WHERE name = 'Granola 7.373.2' AND source = 'programs'`)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &granolaSWName,
			`SELECT name FROM software WHERE name = 'Granola 7.373.2' AND source = 'programs'`)
	})
	require.Equal(t, canonicalTitleID, granolaTitleID)
	// ...while the software row keeps its versioned name.
	require.Equal(t, "Granola 7.373.2", granolaSWName)

	// No standalone "Granola 7.373.2" title was created.
	var granolaVersionTitles int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &granolaVersionTitles,
			`SELECT COUNT(*) FROM software_titles WHERE name = 'Granola 7.373.2'`)
	})
	require.Zero(t, granolaVersionTitles)

	// "Zoombie 5.0" is a distinct app: it must NOT be renamed to "Zoom".
	var zoombieTitleName string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &zoombieTitleName,
			`SELECT st.name FROM software_titles st JOIN software s ON s.title_id = st.id WHERE s.name = 'Zoombie 5.0'`)
	})
	require.Equal(t, "Zoombie 5.0", zoombieTitleName)
}

// testReconcileWindowsSoftwareTitles: existing versioned Windows titles are merged
// onto the canonical FMA title by the reconcile pass (fixes already-mismatched data).
func testReconcileWindowsSoftwareTitles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Ingest two versions BEFORE any FMA exists -> two separate versioned titles
	// (the mismatched state). Include an unrelated "Zoombie" app and an MSI app.
	software := []fleet.Software{
		{Name: "Granola 7.373.1", Version: "7.373.1", Source: "programs"},
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
		{Name: "Zoombie 5.0", Version: "5.0", Source: "programs"},
		{Name: "Widget 2.0", Version: "2.0", Source: "programs", UpgradeCode: new("{ABC}")},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	titleCount := func(name string) int {
		var n int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, `SELECT COUNT(*) FROM software_titles WHERE name = ? AND source = 'programs'`, name)
		})
		return n
	}
	require.Equal(t, 1, titleCount("Granola 7.373.1"))
	require.Equal(t, 1, titleCount("Granola 7.373.2"))

	// Now the Granola + Zoom + Widget FMAs get added, each owning a canonical title.
	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	addWindowsFMAWithInstaller(t, ds, user.ID, "Zoom", "Zoom", "zoom/windows")
	addWindowsFMAWithInstaller(t, ds, user.ID, "Widget", "Widget", "widget/windows")

	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	// A single canonical "Granola" title now owns both versions; the merged-away
	// versioned titles are gone.
	require.Equal(t, 1, titleCount("Granola"))
	require.Zero(t, titleCount("Granola 7.373.1"))
	require.Zero(t, titleCount("Granola 7.373.2"))

	// Both Granola software rows now point at the single canonical title, keeping
	// their versioned names.
	var canonicalID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &canonicalID, `SELECT id FROM software_titles WHERE name = 'Granola' AND source = 'programs'`)
	})
	var linked []struct {
		Name    string `db:"name"`
		TitleID uint   `db:"title_id"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &linked, `SELECT name, title_id FROM software WHERE name LIKE 'Granola %' ORDER BY name`)
	})
	require.Len(t, linked, 2)
	for _, l := range linked {
		require.Equal(t, canonicalID, l.TitleID, "software %q should link to canonical title", l.Name)
	}

	// Negative cases. "Zoombie 5.0" only shares a prefix with the Zoom FMA and keeps its
	// own title. "Widget 2.0" reports an upgrade code, so it is excluded from name
	// merging and stays separate from the Widget installer's title.
	require.Equal(t, 1, titleCount("Zoombie 5.0"))
	require.Equal(t, 1, titleCount("Widget 2.0"))
	require.Equal(t, "Widget 2.0", titleNameForSoftware(t, ds, "Widget 2.0"))

	// Idempotent: a second run changes nothing.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))
	require.Equal(t, 1, titleCount("Granola"))
	require.Zero(t, titleCount("Granola 7.373.1"))

	// The merge deletes titles, and installer/VPP/in-house links to a title are
	// ON DELETE SET NULL, so an owner must never be left pointing at nothing.
	var orphaned int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &orphaned, `
			SELECT
				(SELECT COUNT(*) FROM software_installers WHERE title_id IS NULL) +
				(SELECT COUNT(*) FROM vpp_apps WHERE title_id IS NULL) +
				(SELECT COUNT(*) FROM in_house_apps WHERE title_id IS NULL)`)
	})
	require.Zero(t, orphaned, "no installer, VPP app or in-house app should be left without a title")
}

// testWindowsFMAReconcileSameNameUpgradeCodeTitle: (name, source, extension_for) is not
// unique on software_titles, so a title sharing the canonical name but carrying an
// upgrade code can coexist with the installer's. Only unique_identifier distinguishes
// them, which is why the merge resolves the canonical title through it. Software must
// land on the installer's title.
func testWindowsFMAReconcileSameNameUpgradeCodeTitle(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first, so only a versioned title exists and the installer below has no
	// same-named title to reuse.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	// A second title with the same name, distinguished only by its upgrade code. Reached
	// in practice by ingesting a program named exactly "Granola" that reports one.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_titles (name, source, extension_for, upgrade_code)
			 VALUES ('Granola', 'programs', '', '{22222222-2222-2222-2222-222222222222}')`)
		return err
	})

	// Two titles share the name; exactly one carries the canonical unique_identifier,
	// so resolving through it is unambiguous where resolving through the name is not.
	require.Equal(t, 2, countTitlesNamed(t, ds, "Granola"))
	var byUniqueIdentifier int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &byUniqueIdentifier,
			`SELECT COUNT(*) FROM software_titles
			 WHERE unique_identifier = 'Granola' AND source = 'programs' AND extension_for = ''`)
	})
	require.Equal(t, 1, byUniqueIdentifier)

	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Granola 7.373.2'`)
	})
	require.Equal(t, canonicalID, gotTitleID,
		"software must land on the installer's title, not the same-named upgrade_code title")
}

// testWindowsFMAReconcileAfterCatalogRename: a Windows FMA's software title is never
// renamed when the catalog name changes, so the catalog name can drift from the title
// the installer owns. The merge must follow the installer link, not the current name,
// or it would create a title nobody owns and move software onto it.
func testWindowsFMAReconcileAfterCatalogRename(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first, so a versioned title exists before the installer appears.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Zoom 6.1.0", Version: "6.1.0", Source: "programs"},
	})
	require.NoError(t, err)

	installerTitleID := addWindowsFMAWithInstaller(t, ds, user.ID, "Zoom", "Zoom", "zoom/windows")

	// A later catalog sync renames the app; the installer's title keeps the old name.
	_, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Zoom Workplace",
		Slug:             "zoom/windows",
		Platform:         "windows",
		UniqueIdentifier: "Zoom",
	})
	require.NoError(t, err)

	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Zoom 6.1.0'`)
	})
	require.Equal(t, installerTitleID, gotTitleID,
		"software must merge onto the installer's title, not one named after the new catalog name")
	require.Zero(t, countTitlesNamed(t, ds, "Zoom Workplace"),
		"no title should be invented for the renamed catalog entry")
}

// testWindowsFMAIngestAfterCatalogRename: the ingestion path must resolve the
// destination the same way the reconcile pass does, through the installer link. Using
// the current catalog name instead would create a title no installer owns, and the
// reconcile pass could not repair it because the stale-title scan skips the
// destination itself.
func testWindowsFMAIngestAfterCatalogRename(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	installerTitleID := addWindowsFMAWithInstaller(t, ds, user.ID, "Zoom", "Zoom", "zoom/windows")

	// A later catalog sync renames the app; the installer's title keeps the old name.
	_, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Zoom Workplace",
		Slug:             "zoom/windows",
		Platform:         "windows",
		UniqueIdentifier: "Zoom",
	})
	require.NoError(t, err)

	// Inventory arrives after the rename, so it goes through ingestion, not reconcile.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Zoom 6.1.0", Version: "6.1.0", Source: "programs"},
	})
	require.NoError(t, err)

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Zoom 6.1.0'`)
	})
	require.Equal(t, installerTitleID, gotTitleID,
		"ingestion must land on the installer's title, not one named after the new catalog name")
	require.Zero(t, countTitlesNamed(t, ds, "Zoom Workplace"),
		"no title should be invented for the renamed catalog entry")
}

// testWindowsFMAUninstallActionAvailable asserts the outcome the fix exists for: the
// host's software resolves an installer, which is what surfaces the uninstall action.
// The other tests check title_id values; this one checks the join those values feed.
func testWindowsFMAUninstallActionAvailable(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host.Platform = "windows"
	require.NoError(t, ds.UpdateHost(ctx, host))

	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	sw, _, err := ds.ListHostSoftware(ctx, host, fleet.HostSoftwareTitleListOptions{
		ListOptions:                fleet.ListOptions{PerPage: 50},
		IncludeAvailableForInstall: true,
	})
	require.NoError(t, err)

	var found *fleet.HostSoftwareWithInstaller
	for _, s := range sw {
		if s.Name == "Granola" {
			found = s
			break
		}
	}
	require.NotNil(t, found, "the host's Granola software should roll up under the installer's title")
	require.NotNil(t, found.SoftwarePackage,
		"an installer must resolve for the title, which is what surfaces the uninstall action")
}

// testWindowsFMAMultiTeamInstallersShareTitle: an app added to several teams has one
// installer row per team, all pointing at the same title. Those rows must collapse to
// a single entry rather than fanning out or tripping the ambiguity guard.
func testWindowsFMAMultiTeamInstallersShareTitle(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	titleID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	// Same app on a second team, reusing the same title.
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team A"})
	require.NoError(t, err)
	app, err := ds.GetMaintainedAppBySlug(ctx, "granola/windows", nil)
	require.NoError(t, err)
	_, teamTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		TeamID:               &team.ID,
		Title:                "Granola",
		Source:               "programs",
		StorageID:            "storageid-granola-team",
		Filename:             "granola.exe",
		Extension:            "exe",
		Platform:             "windows",
		Version:              "1.0.0",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: &app.ID,
	})
	require.NoError(t, err)
	require.Equal(t, titleID, teamTitleID, "both teams' installers should share one title")

	names, err := ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Len(t, names, 1, "per-team installer rows must collapse to one entry")
	require.NotNil(t, names[0].TitleID)
	require.Equal(t, titleID, *names[0].TitleID)

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)
	require.Equal(t, "Granola", titleNameForSoftware(t, ds, "Granola 7.373.2"))
}

// testWindowsFMAExcludedWhenSpanningTitles: an app whose installers somehow point at
// different titles has no single destination, so it is excluded rather than guessed at.
func testWindowsFMAExcludedWhenSpanningTitles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	names, err := ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Len(t, names, 1)

	// Point a second installer row for the same app at a different title.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_titles (name, source, extension_for, upgrade_code)
			 VALUES ('Granola Other', 'programs', '', '')`)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_installers
			   (global_or_team_id, title_id, storage_id, filename, extension, version,
			    install_script_content_id, platform, fleet_maintained_app_id, package_ids,
			    uninstall_script_content_id, patch_query)
			 SELECT 7, (SELECT id FROM software_titles WHERE name = 'Granola Other'), 'sid-other', 'g2.exe', 'exe', '2.0',
			        si.install_script_content_id, 'windows', si.fleet_maintained_app_id, '',
			        si.uninstall_script_content_id, si.patch_query
			 FROM software_installers si LIMIT 1`)
		return err
	})

	names, err = ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Empty(t, names, "an app spanning two titles has no unambiguous destination")
}

// testWindowsFMAIgnoresInstallerWithoutTitle: software_installers.title_id is nullable
// (ON DELETE SET NULL), so an installer whose title was removed must not yield a
// zero-valued destination.
func testWindowsFMAIgnoresInstallerWithoutTitle(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET title_id = NULL`)
		return err
	})

	names, err := ds.GetWindowsFMAMatches(ctx)
	require.NoError(t, err)
	require.Empty(t, names, "an installer with no title cannot be a merge destination")

	// And the reconcile pass stays a no-op rather than merging onto title 0.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))
}

// testWindowsFMANameWithLikeWildcards: an FMA name containing LIKE metacharacters must
// be matched literally, not as a wildcard pattern.
func testWindowsFMANameWithLikeWildcards(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first so the titles below are stale candidates for the reconcile pass.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "C_C 1.0", Version: "1.0", Source: "programs"},
		{Name: "CXC 2.0", Version: "2.0", Source: "programs"},
	})
	require.NoError(t, err)

	installerTitleID := addWindowsFMAWithInstaller(t, ds, user.ID, "C_C", "C_C", "c-c/windows")
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	// "C_C 1.0" merges; "CXC 2.0" must not, since _ is a literal here.
	var cUnderscoreTitle, cxcTitle uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &cUnderscoreTitle, `SELECT title_id FROM software WHERE name = 'C_C 1.0'`)
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &cxcTitle, `SELECT title_id FROM software WHERE name = 'CXC 2.0'`)
	})
	require.Equal(t, installerTitleID, cUnderscoreTitle)
	require.NotEqual(t, installerTitleID, cxcTitle, "_ must not act as a wildcard")
}

// testWindowsFMAMatchesCache: ingestion reads the Windows FMA set through a short-lived
// cache, so an app added within the TTL is not matched until it expires. Asserted
// behaviourally, by observing when a newly added app starts collapsing titles.
//
// Expiry is forced by backdating the cached entry rather than by shortening the TTL and
// sleeping: no wall-clock dependency, and the real TTL is left alone so nothing else in
// the package can observe a mutated package-level value.
func testWindowsFMAMatchesCache(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	ds.clearWindowsFMAMatchesCache()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Warm the cache while no app is added, so the entry is an empty set.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Unrelated 1.0", Version: "1.0", Source: "programs"},
	})
	require.NoError(t, err)

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	// Within the TTL the cached empty set is still in use, so no collapsing happens.
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Unrelated 1.0", Version: "1.0", Source: "programs"},
		{Name: "Granola 7.373.1", Version: "7.373.1", Source: "programs"},
	})
	require.NoError(t, err)
	require.Equal(t, "Granola 7.373.1", titleNameForSoftware(t, ds, "Granola 7.373.1"),
		"an app added within the TTL is not yet visible to ingestion")

	// After expiry a newly reported version lands on the installer's title. The cached
	// set stays in place and is simply stale, which is what a real TTL lapse looks like.
	ds.expireWindowsFMAMatchesCache()
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Unrelated 1.0", Version: "1.0", Source: "programs"},
		{Name: "Granola 7.373.1", Version: "7.373.1", Source: "programs"},
		{Name: "Granola 7.441.6", Version: "7.441.6", Source: "programs"},
	})
	require.NoError(t, err)

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Granola 7.441.6'`)
	})
	require.Equal(t, canonicalID, gotTitleID, "after the TTL the added app is matched")

	// The reconcile pass reads uncached, so it repairs what the stale window missed.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))
	require.Equal(t, "Granola", titleNameForSoftware(t, ds, "Granola 7.373.1"))
}

// testWindowsFMAMergeWithoutPrefixes: an app that yields no match prefixes must be a
// safe no-op. The prefixes build the WHERE clause, so an empty set previously risked
// joining into an empty predicate, which MySQL rejects as a syntax error.
func testWindowsFMAMergeWithoutPrefixes(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first, so the software sits on a versioned title that a working merge
	// would move. That is what makes "nothing moved" a meaningful assertion.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	titleID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"),
		"precondition: software is still on the versioned title")

	cases := []struct {
		name string
		app  fleet.MaintainedApp
	}{
		// Every name field blank: no prefixes, so nothing can be matched.
		{"no names", fleet.MaintainedApp{Platform: "windows", TitleID: &titleID}},
		// Not a Windows app: WinMatchPrefixes declines regardless of the names.
		{"not windows", fleet.MaintainedApp{
			Name: "Granola", UniqueIdentifier: "Granola", TitleName: "Granola",
			Platform: "darwin", TitleID: &titleID,
		}},
		// No destination title at all.
		{"no title", fleet.MaintainedApp{Name: "Granola", UniqueIdentifier: "Granola", Platform: "windows"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			apps := []fleet.MaintainedApp{c.app}

			// The scan builds the WHERE clause from the prefixes, so it must find no work
			// rather than emit an empty predicate.
			stale, err := ds.staleWindowsTitlesByDestination(ctx, apps, windowsFMAPrefixes(apps))
			require.NoError(t, err)
			require.Empty(t, stale)

			// And the write half is a no-op for an empty set.
			require.NoError(t, ds.mergeWindowsFMATitle(ctx, ptr.ValOrZero(c.app.TitleID), nil))
			require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"))
		})
	}
}

// testWindowsFMAMergeMovesAllReferences: the merge re-points eight tables and then
// deletes the emptied title. A statement that silently moved nothing would look
// identical to a working one unless each table is checked, and the symptom in production
// is admin configuration stranded on a title with no software.
func testWindowsFMAMergeMovesAllReferences(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first, so a versioned title exists for the installer to merge.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	staleTitleID := titleIDForTitleNamed(t, ds, "Granola 7.373.2")

	// Hang one row off the stale title in every table the merge re-points.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_software_installs (execution_id, host_id, software_title_id)
			VALUES ('exec-1', ?, ?)`, host.ID, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO upcoming_activities (id, host_id, activity_type, execution_id, payload)
			VALUES (9001, ?, 'software_install', 'exec-2', '{}')`, host.ID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_title_id)
			VALUES (9001, ?)`, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO policies (name, query, description, checksum, patch_software_title_id)
			VALUES ('patch granola', 'SELECT 1', '', UNHEX(MD5('patch granola')), ?)`, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_update_schedules (team_id, title_id, start_time, end_time)
			VALUES (0, ?, '01:00', '02:00')`, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_title_display_names (team_id, software_title_id, display_name)
			VALUES (0, ?, 'Granola (custom)')`, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
			VALUES (0, ?, 'sid-icon', 'icon.png')`, staleTitleID)
		return err
	})
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_title_team_pins (team_id, title_id, pinned_version)
			VALUES (0, ?, '7.373.2')`, staleTitleID)
		return err
	})

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	// Every reference now points at the destination, and none is left on the stale id.
	for _, ref := range []struct {
		label  string
		table  string
		column string
	}{
		{"software", "software", "title_id"},
		{"host software installs", "host_software_installs", "software_title_id"},
		{"upcoming install activities", "software_install_upcoming_activities", "software_title_id"},
		{"patch policies", "policies", "patch_software_title_id"},
		{"update schedules", "software_update_schedules", "title_id"},
		{"display names", "software_title_display_names", "software_title_id"},
		{"icons", "software_title_icons", "software_title_id"},
		{"team pins", "software_title_team_pins", "title_id"},
	} {
		//nolint:gosec // table and column come from the fixed list above, not from input
		countFor := func(titleID uint) int {
			var n int
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &n,
					fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = ?`, ref.table, ref.column), titleID)
			})
			return n
		}
		require.Equal(t, 1, countFor(canonicalID), "%s should have moved to the destination", ref.label)
		require.Zero(t, countFor(staleTitleID), "%s should not be left on the merged-away title", ref.label)
	}

	// The emptied title is gone, and its host counts with it.
	require.Zero(t, countTitlesNamed(t, ds, "Granola 7.373.2"))
	var counts int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &counts,
			`SELECT COUNT(*) FROM software_titles_host_counts WHERE software_title_id = ?`, staleTitleID)
	})
	require.Zero(t, counts, "host counts for the merged-away title should be deleted")
}

// testWindowsFMAMergeMultipleDestinations: the pass groups candidates by destination and
// merges each in its own transaction, so two apps with stale titles must both be handled
// in a single run rather than only whichever is processed first.
func testWindowsFMAMergeMultipleDestinations(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Two unrelated apps, each reported with a version in the name, before either is added.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
		{Name: "Obsidian 1.5.3", Version: "1.5.3", Source: "programs"},
	})
	require.NoError(t, err)

	granolaID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	obsidianID := addWindowsFMAWithInstaller(t, ds, user.ID, "Obsidian", "Obsidian", "obsidian/windows")

	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	require.Equal(t, granolaID, titleIDForSoftware(t, ds, "Granola 7.373.2"))
	require.Equal(t, obsidianID, titleIDForSoftware(t, ds, "Obsidian 1.5.3"))
	require.Zero(t, countTitlesNamed(t, ds, "Granola 7.373.2"))
	require.Zero(t, countTitlesNamed(t, ds, "Obsidian 1.5.3"))
}

// testWindowsFMAIngestIgnoresOtherSources: name matching is only meaningful for the
// programs table. Software from another source that happens to share a prefix with an
// added app must keep its own title.
func testWindowsFMAIngestIgnoresOtherSources(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 9.9.9", Version: "9.9.9", Source: "chrome_extensions", ExtensionID: "abc"},
	})
	require.NoError(t, err)

	require.Equal(t, "Granola 9.9.9", titleNameForSoftware(t, ds, "Granola 9.9.9"),
		"non-programs software must not be collapsed onto the installer's title")

	// And the reconcile pass leaves it alone too.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))
	require.Equal(t, "Granola 9.9.9", titleNameForSoftware(t, ds, "Granola 9.9.9"))
}

// testWindowsFMAReconcileIndependentOfCatalogSync: the Windows merge is reachable on its
// own, without the macOS catalog pass. That separation is the point of splitting them: on
// the catalog sync it was gated on a successful manifest fetch, so an instance that could
// not reach the CDN never repaired its Windows titles.
func testWindowsFMAReconcileIndependentOfCatalogSync(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"),
		"precondition: software is on the versioned title")

	// The Windows pass alone is enough; the macOS pass is not involved.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	require.Equal(t, canonicalID, titleIDForSoftware(t, ds, "Granola 7.373.2"))
	require.Zero(t, countTitlesNamed(t, ds, "Granola 7.373.2"))

	// And the macOS pass no longer performs the Windows merge, so running it on already
	// merged data is a no-op rather than a second attempt.
	require.NoError(t, ds.ReconcileMaintainedAppSoftwareNames(ctx))
	require.Equal(t, canonicalID, titleIDForSoftware(t, ds, "Granola 7.373.2"))
}

// titleIDForSoftware returns the software title that the given software row links to.
func titleIDForSoftware(t *testing.T, ds *Datastore, softwareName string) uint {
	t.Helper()
	var titleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &titleID,
			`SELECT title_id FROM software WHERE name = ?`, softwareName)
	})
	return titleID
}

// titleIDForTitleNamed returns the id of the 'programs' software title with the given
// name. Distinct from titleIDForSoftware, which resolves through a software row.
func titleIDForTitleNamed(t *testing.T, ds *Datastore, name string) uint {
	t.Helper()
	var id uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &id,
			`SELECT id FROM software_titles WHERE name = ? AND source = 'programs'`, name)
	})
	return id
}

// addWindowsFMAWithInstaller creates a Windows FMA plus an installer that owns the
// canonical title, mirroring an admin adding a Fleet-maintained app to no team.
func addWindowsFMAWithInstaller(t *testing.T, ds *Datastore, userID uint, name, uniqueIdentifier, slug string) uint {
	t.Helper()
	ctx := t.Context()

	app, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             name,
		Slug:             slug,
		Platform:         "windows",
		UniqueIdentifier: uniqueIdentifier,
	})
	require.NoError(t, err)

	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                name,
		Source:               "programs",
		StorageID:            "storageid-" + slug,
		Filename:             slug + ".exe",
		Extension:            "exe",
		Platform:             "windows",
		Version:              "1.0.0",
		UserID:               userID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: &app.ID,
	})
	require.NoError(t, err)

	return titleID
}

// titleNameForSoftware returns the name of the software title that the given
// software row is linked to.
func titleNameForSoftware(t *testing.T, ds *Datastore, softwareName string) string {
	t.Helper()
	var name string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &name,
			`SELECT st.name FROM software_titles st JOIN software s ON s.title_id = st.id WHERE s.name = ?`,
			softwareName)
	})
	return name
}

// countTitlesNamed returns how many 'programs' software titles carry the given name.
func countTitlesNamed(t *testing.T, ds *Datastore, name string) int {
	t.Helper()
	var n int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &n,
			`SELECT COUNT(*) FROM software_titles WHERE name = ? AND source = 'programs'`, name)
	})
	return n
}

// testWindowsFMAMatchByUniqueIdentifier: some Windows FMAs report a program name
// built from unique_identifier rather than the FMA's display name (osquery reports
// "CPUID CPU-Z ..." for the FMA named "CPU-Z"). The display name is not a prefix of
// the reported name, so matching must consider unique_identifier too.
func testWindowsFMAMatchByUniqueIdentifier(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "CPU-Z", "CPUID CPU-Z", "cpu-z/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "CPUID CPU-Z 2.16", Version: "2.16", Source: "programs"},
	})
	require.NoError(t, err)

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'CPUID CPU-Z 2.16'`)
	})
	require.Equal(t, canonicalID, gotTitleID, "inventory should link to the installer's title")
	require.Zero(t, countTitlesNamed(t, ds, "CPUID CPU-Z 2.16"), "no versioned title should be created")
}

// testWindowsFMAMatchByNameWhenIdentifierStale: fleet_maintained_apps.unique_identifier
// is synced from apps.json, whose entries are only ever appended, so some Windows apps
// carry a version-bearing identifier frozen at the version current when they were added
// (e.g. notion/windows records "Notion 6.1.0"). Matching must still work off the name.
func testWindowsFMAMatchByNameWhenIdentifierStale(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Notion", "Notion 6.1.0", "notion/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Notion 7.2.0", Version: "7.2.0", Source: "programs"},
	})
	require.NoError(t, err)

	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Notion 7.2.0'`)
	})
	require.Equal(t, canonicalID, gotTitleID)
	require.Zero(t, countTitlesNamed(t, ds, "Notion 7.2.0"))
}

// testWindowsFMANotRenamedWithUpgradeCode: software_titles.unique_identifier resolves to
// upgrade_code before name, so giving a program that reports an upgrade code the canonical
// FMA name produces a SECOND title with that name under a different unique_identifier.
// Programs with an upgrade code already match through it and must be left alone.
func testWindowsFMANotRenamedWithUpgradeCode(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs", UpgradeCode: new("{11111111-1111-1111-1111-111111111111}")},
	})
	require.NoError(t, err)

	require.Equal(t, 1, countTitlesNamed(t, ds, "Granola"), "must not create a duplicate canonical title")
	require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"),
		"software with an upgrade code keeps its own title")
}

// testWindowsFMANoCollapseWithoutInstaller: with no publisher available to scope the
// name match, collapsing is limited to FMAs an admin actually added. A catalog entry
// alone must not rewrite unrelated inventory titles.
func testWindowsFMANoCollapseWithoutInstaller(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	_, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Granola",
		Slug:             "granola/windows",
		Platform:         "windows",
		UniqueIdentifier: "Granola",
	})
	require.NoError(t, err)

	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"))
	require.Zero(t, countTitlesNamed(t, ds, "Granola"))

	// The reconcile pass must leave it alone too.
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))
	require.Equal(t, "Granola 7.373.2", titleNameForSoftware(t, ds, "Granola 7.373.2"))
	require.Zero(t, countTitlesNamed(t, ds, "Granola"))
}

// testWindowsFMAAmbiguousMatchIsNoOp: when a reported name matches two FMAs that
// disagree on the canonical name, there is no principled winner, so nothing is renamed.
// Mirrors the shared-bundle-identifier rule the darwin reconcile passes already apply.
func testWindowsFMAAmbiguousMatchIsNoOp(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Two distinct FMAs whose programs name both resolve to the same "Acme" prefix.
	addWindowsFMAWithInstaller(t, ds, user.ID, "Acme Reader", "Acme", "acme-reader/windows")
	addWindowsFMAWithInstaller(t, ds, user.ID, "Acme Writer", "Acme", "acme-writer/windows")

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Acme 3.0", Version: "3.0", Source: "programs"},
	})
	require.NoError(t, err)

	require.Equal(t, "Acme 3.0", titleNameForSoftware(t, ds, "Acme 3.0"),
		"an ambiguous match must not pick a winner")
}

// testWindowsFMAReconcileMovesTitleReferences: the merge re-points software onto the
// destination and deletes the emptied title, so admin configuration attached to it must
// be moved first rather than cascaded away with it.
func testWindowsFMAReconcileMovesTitleReferences(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	// Inventory first, so a versioned title exists before the FMA installer appears.
	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	var staleTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &staleTitleID,
			`SELECT id FROM software_titles WHERE name = 'Granola 7.373.2' AND source = 'programs'`)
	})

	// An admin pins a version on that title.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '7.373.2')`,
			staleTitleID)
		return err
	})

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")
	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	// Software moved onto the canonical title...
	var gotTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &gotTitleID,
			`SELECT title_id FROM software WHERE name = 'Granola 7.373.2'`)
	})
	require.Equal(t, canonicalID, gotTitleID)

	// ...the merged-away title is gone...
	var staleStillThere int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &staleStillThere,
			`SELECT COUNT(*) FROM software_titles WHERE id = ?`, staleTitleID)
	})
	require.Zero(t, staleStillThere, "merged-away title should be deleted")

	// ...and the pin moved with it rather than being cascaded away, so it still governs
	// the software the admin pinned.
	var pinnedVersion string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &pinnedVersion,
			`SELECT pinned_version FROM software_title_team_pins WHERE title_id = ?`, canonicalID)
	})
	require.Equal(t, "7.373.2", pinnedVersion, "the admin's pin must follow the software")

	var orphanPins int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &orphanPins,
			`SELECT COUNT(*) FROM software_title_team_pins WHERE title_id = ?`, staleTitleID)
	})
	require.Zero(t, orphanPins)
}

// testWindowsFMAReconcilePinConflict: when the destination already has a pin for the
// same team, the stale one cannot be moved onto it (unique on team_id, title_id). The
// destination's pin is authoritative and the stale row goes away with its title.
func testWindowsFMAReconcilePinConflict(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())

	_, err := ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "Granola 7.373.2", Version: "7.373.2", Source: "programs"},
	})
	require.NoError(t, err)

	var staleTitleID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &staleTitleID,
			`SELECT id FROM software_titles WHERE name = 'Granola 7.373.2' AND source = 'programs'`)
	})

	canonicalID := addWindowsFMAWithInstaller(t, ds, user.ID, "Granola", "Granola", "granola/windows")

	// Both titles carry a pin for the same team.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version)
			 VALUES (0, ?, '7.373.2'), (0, ?, '9.9.9')`, staleTitleID, canonicalID)
		return err
	})

	require.NoError(t, ds.ReconcileWindowsMaintainedAppSoftwareTitles(ctx))

	var pinnedVersion string
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &pinnedVersion,
			`SELECT pinned_version FROM software_title_team_pins WHERE title_id = ?`, canonicalID)
	})
	require.Equal(t, "9.9.9", pinnedVersion, "the destination's own pin wins")

	var total int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &total, `SELECT COUNT(*) FROM software_title_team_pins`)
	})
	require.Equal(t, 1, total, "the skipped pin is cascaded away with its title")
}

package mysql

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestInHouseApps(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestInHouseAppsCrud", testInHouseAppsCrud},
		{"MultipleTeams", testInHouseAppsMultipleTeams},
		{"BatchSetInHouseInstallers", testBatchSetInHouseInstallers},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testInHouseAppsCrud(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "host3key", "host3uuid", time.Now())

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID, host2.ID, host3.ID}))
	require.NoError(t, err)

	nanoEnroll(t, ds, host1, false)
	nanoEnroll(t, ds, host2, false)
	nanoEnroll(t, ds, host3, false)

	payload := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team.ID,
		UserID:           user1.ID,
		Title:            "foo",
		Filename:         "foo.ipa",
		BundleIdentifier: "com.foo",
		StorageID:        "testingtesting123",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
	}

	// -------------------------
	// Upload software installer
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.Error(t, err, "ValidatedLabels must not be nil")

	payload.ValidatedLabels = &fleet.LabelIdentsWithScope{}
	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.NoError(t, err)
	require.NotZero(t, installerID)
	require.NotZero(t, titleID)

	// both ios and ipados apps are created, both installer and title
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var countI, countS uint
		var ihaFilename, titleName string

		errI := sqlx.GetContext(ctx, q, &countI, `SELECT COUNT(*) FROM in_house_apps`)
		require.NoError(t, errI)
		require.Equal(t, uint(2), countI)
		errS := sqlx.GetContext(ctx, q, &countS, `SELECT COUNT(*) FROM software_titles`)
		require.NoError(t, errS)
		require.Equal(t, uint(2), countS)

		errI = sqlx.GetContext(ctx, q, &ihaFilename, `SELECT filename FROM in_house_apps WHERE id = ?`, installerID)
		require.NoError(t, errI)
		require.Equal(t, "foo.ipa", ihaFilename)
		errS = sqlx.GetContext(ctx, q, &titleName, `SELECT name FROM software_titles WHERE id = ?`, titleID)
		require.NoError(t, errS)
		require.Equal(t, "foo", titleName)
		return nil
	})

	installer, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Equal(t, payload.Title, installer.SoftwareTitle)
	require.Equal(t, payload.Version, installer.Version)
	require.WithinDuration(t, time.Now(), installer.UploadedAt, time.Minute)
	require.False(t, payload.SelfService)

	// Install on multiple users with pending, success, failure
	createInHouseAppInstallRequest(t, ds, host1.ID, installerID, titleID, user1)
	cmdUUID2 := createInHouseAppInstallRequest(t, ds, host2.ID, installerID, titleID, user1)
	createInHouseAppInstallResult(t, ds, host2, cmdUUID2, "Acknowledged")
	cmdUUID3 := createInHouseAppInstallRequest(t, ds, host3.ID, installerID, titleID, user1)
	createInHouseAppInstallResult(t, ds, host3, cmdUUID3, "Error")

	// Get summary and expect failed, installed, pending
	summary, err := ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, installerID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1, Pending: 1, Failed: 1}, *summary)

	// -------------------------
	// Update software installer
	label, err := ds.NewLabel(ctx, &fleet.Label{Name: "include-any-1", Query: "select 1"})
	require.NoError(t, err)

	validatedLabels := fleet.LabelIdentsWithScope{
		LabelScope: "include_any",
		ByName: map[string]fleet.LabelIdent{
			"include-any-1": {
				LabelID:   label.ID,
				LabelName: label.Name,
			},
		}}
	updatePayload := fleet.UpdateSoftwareInstallerPayload{
		TeamID:          &team.ID,
		TitleID:         titleID,
		InstallerID:     installerID,
		Filename:        "ipa_test.ipa",
		StorageID:       "new_storage_id",
		ValidatedLabels: &validatedLabels,
		SelfService:     ptr.Bool(true),
	}

	err = ds.SaveInHouseAppUpdates(ctx, &updatePayload)
	require.NoError(t, err)

	// Installer updates correctly
	var expectedLabels []fleet.SoftwareScopeLabel
	expectedLabels = append(expectedLabels, fleet.SoftwareScopeLabel{LabelID: label.ID, LabelName: label.Name, Exclude: false, TitleID: titleID})

	newInstaller, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Equal(t, "new_storage_id", newInstaller.StorageID)
	require.Equal(t, expectedLabels, newInstaller.LabelsIncludeAny)
	require.True(t, newInstaller.SelfService)

	// Summary is unchanged?
	summary2, err := ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, installerID)
	require.NoError(t, err)
	require.Equal(t, summary, summary2)

	// -------------------------
	// Delete software installer
	err = ds.DeleteInHouseApp(ctx, installerID)
	require.NoError(t, err)

	// TODO: test RemovePendingInHouseAppInstalls independently

	_, err = ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.Error(t, err)
	status, err := ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, installerID)
	require.NoError(t, err)
	require.Zero(t, *status)

	// Check that entire tables are empty for this test
	checkEmpty := func(table string) {
		var count int
		err := sqlx.GetContext(ctx, ds.reader(ctx), &count, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table))
		require.NoError(t, err)
		require.Zero(t, count, "expected %s to be empty", table)
	}

	checkEmpty("in_house_app_labels")
	checkEmpty("host_in_house_software_installs")
	checkEmpty("in_house_app_upcoming_activities")
	checkEmpty("upcoming_activities")

	// ipadOS installer should remain
	var ipadID uint
	err = sqlx.GetContext(ctx, ds.reader(ctx), &ipadID, `SELECT id FROM in_house_apps LIMIT 1`)
	require.NoError(t, err)

	// Try to upload installer again, expect duplicate error
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.Error(t, err)

	// Delete ipadOS installer
	err = ds.DeleteInHouseApp(ctx, ipadID)
	require.NoError(t, err)

	var count int
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM in_house_apps`)
	require.NoError(t, err)
	require.Zero(t, count, "expected in_house_apps to be empty")

	// create one that's self-service at creation time
	payload2 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team.ID,
		UserID:           user1.ID,
		Title:            "foo2",
		Filename:         "foo2.ipa",
		BundleIdentifier: "com.foo2",
		StorageID:        "testingtesting1234",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
		SelfService:      true,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}
	_, titleID2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload2)
	require.NoError(t, err)
	require.NotZero(t, installerID)
	require.NotZero(t, titleID)

	installer2, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID2)
	require.NoError(t, err)
	require.Equal(t, payload2.Title, installer2.SoftwareTitle)
	require.True(t, installer2.SelfService)
}

func testInHouseAppsMultipleTeams(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID}))
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host2.ID}))
	require.NoError(t, err)

	nanoEnroll(t, ds, host1, false)

	payload1 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team1.ID,
		UserID:           user1.ID,
		Title:            "foo",
		BundleIdentifier: "com.foo",
		StorageID:        "testingtesting123",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}

	payload2 := payload1
	payload2.TeamID = &team2.ID

	payloadNoTeam := payload1
	payloadNoTeam.TeamID = nil

	// Add installers for both teams
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload1)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload2)
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payloadNoTeam)
	require.NoError(t, err)

	var count int
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM software_titles`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_apps`)
	require.NoError(t, err)
	require.Equal(t, 6, count)

	// Team 2: Delete 1 installer from 1 team
	err = ds.DeleteInHouseApp(ctx, installerID2)
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_apps`)
	require.NoError(t, err)
	require.Equal(t, 5, count)

	// Team 2: Try to add installer again
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload2)
	require.Error(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_apps`)
	require.NoError(t, err)
	require.Equal(t, 5, count)

	// Test that software titles for IHA don't get cleaned up
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.CleanupSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM software_titles`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

}

func createInHouseAppInstallRequest(t *testing.T, ds *Datastore, hostID uint, appID uint, titleID uint, user *fleet.User) string {
	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	cmdUUID := uuid.NewString()

	err := ds.InsertHostInHouseAppInstall(ctx, hostID, appID, titleID, cmdUUID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	return cmdUUID
}

func createInHouseAppInstallResult(t *testing.T, ds *Datastore, host *fleet.Host, cmdUUID string, status string) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, fleet.ActivityWebhookContextKey, true)

	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	nanoCtx := &mdm.Request{EnrollID: &mdm.EnrollID{ID: host.UUID}, Context: ctx}

	cmdRes := &mdm.CommandResults{
		CommandUUID: cmdUUID,
		Status:      status,
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	// inserting the activity is what marks the upcoming activity as completed
	// (and activates the next one).
	err = ds.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      host.ID,
		CommandUUID: cmdUUID,
	}, []byte(`{}`), time.Now())
	require.NoError(t, err)
}

func createInHouseAppInstallResultVerified(t *testing.T, ds *Datastore, host *fleet.Host, cmdUUID string, status string) {
	createInHouseAppInstallResult(t, ds, host, cmdUUID, status)

	ctx := t.Context()
	timestampCol := "verification_at"
	if status != "Acknowledged" {
		timestampCol = "verification_failed_at"
	}
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE host_in_house_software_installs SET
			%s = NOW(6), verification_command_uuid = ? WHERE host_id = ? AND command_uuid = ?`, timestampCol),
			uuid.NewString(), host.ID, cmdUUID)
		return err
	})
}

func testBatchSetInHouseInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// create a couple hosts
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now(), test.WithPlatform("ios"))
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now(), test.WithPlatform("ios"))
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID, host2.ID}))
	require.NoError(t, err)
	nanoEnroll(t, ds, host1, false)
	nanoEnroll(t, ds, host2, false)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	assertTitlesAndApps := func(wantTitles []fleet.SoftwareTitleListResult, wantApps []fleet.InHouseAppPayload) {
		tmFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		titles, _, _, err := ds.ListSoftwareTitles(
			ctx,
			fleet.SoftwareTitleListOptions{TeamID: &team.ID},
			tmFilter,
		)
		require.NoError(t, err)
		require.Len(t, titles, len(wantTitles))

		sort.Slice(wantTitles, func(i, j int) bool {
			l, r := wantTitles[i], wantTitles[j]
			return l.Name < r.Name || l.Name == r.Name && l.Source < r.Source
		})
		sort.Slice(titles, func(i, j int) bool {
			l, r := titles[i], titles[j]
			return l.Name < r.Name || l.Name == r.Name && l.Source < r.Source
		})

		titleIDs := make([]uint, len(wantTitles))
		for i, want := range wantTitles {
			got := titles[i]
			require.Equal(t, want.Name, got.Name)
			require.Equal(t, want.Source, got.Source)
			require.Equal(t, want.BundleIdentifier == nil, got.BundleIdentifier == nil)
			if want.BundleIdentifier != nil {
				require.Equal(t, *want.BundleIdentifier, *got.BundleIdentifier)
			}
			titleIDs[i] = got.ID
		}

		sort.Slice(wantApps, func(i, j int) bool {
			l, r := wantApps[i], wantApps[j]
			return l.Filename < r.Filename || l.Filename == r.Filename && l.Platform < r.Platform
		})
		require.Len(t, wantApps, len(titleIDs))

		for i, want := range wantApps {
			got, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleIDs[i])
			require.NoError(t, err)
			require.Equal(t, want.Filename, got.Name)
			require.Equal(t, want.Platform, got.Platform)
			require.Equal(t, want.Version, got.Version)
			require.Equal(t, want.StorageID, got.StorageID)
			require.Equal(t, want.SelfService, got.SelfService)
			require.Equal(t, want.BundleID, got.BundleIdentifier)
		}
	}

	// batch set with everything empty
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, nil)
	require.NoError(t, err)
	apps, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, apps)
	assertTitlesAndApps(nil, nil)

	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, apps)
	assertTitlesAndApps(nil, nil)

	ipa1 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team.ID,
		UserID:           user1.ID,
		Title:            "ipa1",
		Filename:         "ipa1.ipa",
		BundleIdentifier: "com.ipa1",
		StorageID:        "ipa1",
		Extension:        "ipa",
		Version:          "1.0.0",
		SelfService:      true,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}
	// the batch-upload handler would've generated both iOS and iPadOS entries
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ios_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ipados_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.NotNil(t, apps[0].TeamID)
	require.Equal(t, team.ID, *apps[0].TeamID)
	require.NotNil(t, apps[0].TitleID)
	require.Equal(t, "https://example.com/1", apps[0].URL)

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa1.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa1.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa1")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa1.Filename, Platform: "ios", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa1.Filename, Platform: "ipados", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
	})

	// add a new installer + ipa1 installer
	ipa2 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team.ID,
		UserID:           user1.ID,
		Title:            "ipa2",
		Filename:         "ipa2.ipa",
		BundleIdentifier: "com.ipa2",
		StorageID:        "ipa2",
		Extension:        "ipa",
		Version:          "2.0.0",
		SelfService:      false,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}

	// the batch-upload handler would've generated both iOS and iPadOS entries
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ios_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ipados_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ios_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ipados_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	sort.Slice(apps, func(i, j int) bool {
		l, r := apps[i], apps[j]
		return l.URL < r.URL
	})
	require.Len(t, apps, 4)
	require.NotNil(t, apps[0].TeamID)
	require.Equal(t, team.ID, *apps[0].TeamID)
	require.NotNil(t, apps[0].TitleID)
	require.Equal(t, "https://example.com/1", apps[0].URL)

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa1.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa1.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa2.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa2")},
		{Name: ipa2.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa2")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa1.Filename, Platform: "ios", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa1.Filename, Platform: "ipados", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ios", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ipados", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
	})

	// rerun with no change
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ios_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ipados_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ios_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ipados_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 4)

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa1.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa1.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa2.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa2")},
		{Name: ipa2.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa2")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa1.Filename, Platform: "ios", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa1.Filename, Platform: "ipados", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ios", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ipados", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
	})

	// change ipa2 self-service
	ipa2.SelfService = !ipa2.SelfService
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ios_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ipados_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ios_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ipados_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 4)

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa1.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa1.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa2.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa2")},
		{Name: ipa2.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa2")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa1.Filename, Platform: "ios", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa1.Filename, Platform: "ipados", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ios", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ipados", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
	})

	// remove ipa1
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ios_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ipados_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	var iosInHouseID, iosTitleID uint
	for _, app := range apps {
		require.NotNil(t, app.TitleID)
		meta, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, *app.TitleID)
		require.NoError(t, err)
		if meta.Platform == "ios" {
			iosInHouseID = meta.InstallerID
			iosTitleID = *app.TitleID
		}
	}

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa2.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa2")},
		{Name: ipa2.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa2")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa2.Filename, Platform: "ios", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ipados", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
	})

	// add pending and completed installs for ipa1
	completedCmd := createInHouseAppInstallRequest(t, ds, host1.ID, iosInHouseID, iosTitleID, user1)
	pendingCmd := createInHouseAppInstallRequest(t, ds, host2.ID, iosInHouseID, iosTitleID, user1)
	createInHouseAppInstallResultVerified(t, ds, host1, completedCmd, "Acknowledged")

	summary, err := ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, iosInHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1, Pending: 1}, *summary)
	_ = pendingCmd

	// batch-set without changes, should not affect installs
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ios_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
		{
			StorageID:        ipa2.StorageID,
			Filename:         ipa2.Filename,
			Title:            ipa2.Title,
			Source:           "ipados_apps",
			Version:          ipa2.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/2",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa2.BundleIdentifier,
			SelfService:      ipa2.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa2.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa2")},
		{Name: ipa2.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa2")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa2.Filename, Platform: "ios", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
		{Filename: ipa2.Filename, Platform: "ipados", Version: ipa2.Version, StorageID: ipa2.StorageID, SelfService: ipa2.SelfService, BundleID: ipa2.BundleIdentifier},
	})

	summary, err = ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, iosInHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 1, Pending: 1}, *summary)

	// remove ipa2 and add ipa1
	err = ds.BatchSetInHouseAppsInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ios_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ios",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
		{
			StorageID:        ipa1.StorageID,
			Filename:         ipa1.Filename,
			Title:            ipa1.Title,
			Source:           "ipados_apps",
			Version:          ipa1.Version,
			UserID:           user1.ID,
			Platform:         "ipados",
			URL:              "https://example.com/1",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: ipa1.BundleIdentifier,
			SelfService:      ipa1.SelfService,
		},
	})
	require.NoError(t, err)

	apps, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, apps, 2)

	assertTitlesAndApps([]fleet.SoftwareTitleListResult{
		{Name: ipa1.Title, Source: "ios_apps", BundleIdentifier: ptr.String("com.ipa1")},
		{Name: ipa1.Title, Source: "ipados_apps", BundleIdentifier: ptr.String("com.ipa1")},
	}, []fleet.InHouseAppPayload{
		{Filename: ipa1.Filename, Platform: "ios", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
		{Filename: ipa1.Filename, Platform: "ipados", Version: ipa1.Version, StorageID: ipa1.StorageID, SelfService: ipa1.SelfService, BundleID: ipa1.BundleIdentifier},
	})

	// stats don't report anything about ins1 anymore
	summary, err = ds.GetSummaryHostInHouseAppInstalls(ctx, &team.ID, iosInHouseID)
	require.NoError(t, err)
	require.Equal(t, fleet.VPPAppStatusSummary{Installed: 0, Pending: 0}, *summary)

	pendingHost1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Empty(t, pendingHost1)
	pendingHost2, err := ds.ListPendingSoftwareInstalls(ctx, host2.ID)
	require.NoError(t, err)
	require.Empty(t, pendingHost2)

	/*
		// add pending and completed installs for ins0
		softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
		require.NoError(t, err)
		require.Len(t, softwareInstallers, 1)
		instDetails0, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *softwareInstallers[0].TitleID, false)
		require.NoError(t, err)

		_, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, instDetails0.InstallerID, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)
		execID2b, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, instDetails0.InstallerID, fleet.HostSoftwareInstallOptions{})
		require.NoError(t, err)

		_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
			HostID:                host2.ID,
			InstallUUID:           execID2b,
			InstallScriptExitCode: ptr.Int(1),
		})
		require.NoError(t, err)

		pendingHost1, err = ds.ListPendingSoftwareInstalls(ctx, host1.ID)
		require.NoError(t, err)
		require.Len(t, pendingHost1, 1)

		summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails0.InstallerID)
		require.NoError(t, err)
		require.Equal(t, fleet.SoftwareInstallerStatusSummary{FailedInstall: 1, PendingInstall: 1}, *summary)

		// Add software installer with same name different bundle id
		err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
			InstallScript:    "install",
			InstallerFile:    tfr0,
			StorageID:        ins0,
			Filename:         "installer0",
			Title:            "ins0",
			Source:           "apps",
			Version:          "1",
			PreInstallQuery:  "foo",
			UserID:           user1.ID,
			Platform:         "darwin",
			URL:              "https://example.com",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.different.ins0",
		}})
		require.NoError(t, err)
		softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
		require.NoError(t, err)
		require.Len(t, softwareInstallers, 1)
		assertSoftware([]fleet.SoftwareTitle{
			{Name: ins0, Source: "apps", ExtensionFor: "", BundleIdentifier: ptr.String("com.example.different.ins0")},
		})

		// Add software installer with the same bundle id but different name
		err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
			InstallScript:    "install",
			InstallerFile:    tfr0,
			StorageID:        ins0,
			Filename:         "installer0",
			Title:            "ins0-different",
			Source:           "apps",
			Version:          "1",
			PreInstallQuery:  "foo",
			UserID:           user1.ID,
			Platform:         "darwin",
			URL:              "https://example.com",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.ins0",
		}})
		require.NoError(t, err)
		softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
		require.NoError(t, err)
		require.Len(t, softwareInstallers, 1)
		assertSoftware([]fleet.SoftwareTitle{
			{Name: "ins0-different", Source: "apps", ExtensionFor: "", BundleIdentifier: ptr.String("com.example.ins0")},
		})

		// remove everything
		err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
		require.NoError(t, err)
		softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
		require.NoError(t, err)
		require.Empty(t, softwareInstallers)
		assertSoftware([]fleet.SoftwareTitle{})

		// stats don't report anything about ins0 anymore
		summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails0.InstallerID)
		require.NoError(t, err)
		require.Equal(t, fleet.SoftwareInstallerStatusSummary{FailedInstall: 0, PendingInstall: 0}, *summary)
		pendingHost1, err = ds.ListPendingSoftwareInstalls(ctx, host1.ID)
		require.NoError(t, err)
		require.Empty(t, pendingHost1)
	*/
}

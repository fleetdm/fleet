package mysql

import (
	"context"
	"fmt"
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
		{"Categories", testInHouseAppsCategories},
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

func testInHouseAppsCategories(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID})))

	nanoEnroll(t, ds, host1, false)

	payload1 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team1.ID,
		UserID:           user1.ID,
		BundleIdentifier: "com.foo",
		Filename:         "foo.ipa",
		StorageID:        "id1234",
		Extension:        "ipa",
		SelfService:      false,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		CategoryIDs:      []uint{1, 2},
	}

	// Software categories are missing from test schema
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO software_categories 
			VALUES (1,'Productivity'), (2,'Browsers'),(3,'Communication'),(4,'Developer tools')`)
		return err
	})

	// Add installers for both teams
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload1)
	require.NoError(t, err)

	var count int
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_app_software_categories WHERE in_house_app_id = ?`, installerID)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM software_titles`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_apps`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Test with empty categories

	payload2 := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team1.ID,
		UserID:           user1.ID,
		BundleIdentifier: "com.bar",
		Filename:         "bar.ipa",
		StorageID:        "id5678",
		Extension:        "ipa",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		CategoryIDs:      nil, // empty slice should work the same
	}

	secondInstallerID, secondTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload2)
	require.NoError(t, err)

	// Check that this has no categories
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_app_software_categories WHERE in_house_app_id = ?`, secondInstallerID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Update software categories
	updatePayload := fleet.UpdateSoftwareInstallerPayload{
		TeamID:          payload2.TeamID,
		TitleID:         secondTitleID,
		InstallerID:     secondInstallerID,
		Filename:        payload2.Filename,
		StorageID:       payload2.StorageID,
		ValidatedLabels: payload2.ValidatedLabels,
		CategoryIDs:     []uint{1, 2},
		SelfService:     ptr.Bool(true),
	}

	err = ds.SaveInHouseAppUpdates(ctx, &updatePayload)
	require.NoError(t, err)

	// Categories
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(id) FROM in_house_app_software_categories WHERE in_house_app_id = ?`, secondInstallerID)
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

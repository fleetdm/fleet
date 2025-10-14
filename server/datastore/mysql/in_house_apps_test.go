package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	payload := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team.ID,
		Title:            "foo",
		BundleIdentifier: "com.foo",
		StorageID:        "testingtesting123",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
	}

	// Upload software installer
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.Error(t, err, "ValidatedLabels must not be nil")

	payload.ValidatedLabels = &fleet.LabelIdentsWithScope{}
	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.NoError(t, err)
	require.NotZero(t, installerID)
	require.NotZero(t, titleID)

	installer, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Equal(t, payload.Title, installer.SoftwareTitle)
	require.Equal(t, payload.Version, installer.Version)

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
	}

	err = ds.SaveInHouseAppUpdates(ctx, &updatePayload)
	require.NoError(t, err)

	// ds.RemovePendingInHouseAppInstalls()
	// TODO: add tests for this

	// Installer updates correctly
	var expectedLabels []fleet.SoftwareScopeLabel
	expectedLabels = append(expectedLabels, fleet.SoftwareScopeLabel{LabelID: label.ID, LabelName: label.Name, Exclude: false, TitleID: titleID})

	newInstaller, err := ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Equal(t, "new_storage_id", newInstaller.StorageID)
	require.Equal(t, expectedLabels, newInstaller.LabelsIncludeAny)

	// Summary returns expected pending/failed/installed numbers
	// TODO: add tests for this
	_, err = ds.GetSummaryInHouseAppInstalls(ctx, &team.ID, installerID)
	require.NoError(t, err)

	// Delete software installer
	err = ds.DeleteInHouseApp(ctx, installerID)
	require.NoError(t, err)

	_, err = ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.Error(t, err)
	status, err := ds.GetSummaryInHouseAppInstalls(ctx, &team.ID, installerID)
	require.Zero(t, *status)

	// Check that entire tables are empty for this test
	checkEmpty := func(table string) {
		var count int
		err := sqlx.GetContext(ctx, ds.reader(ctx), &count, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table))
		require.NoError(t, err)
		require.Zero(t, count, "expected %s to be empty", table)
	}

	checkEmpty("in_house_apps")
	checkEmpty("in_house_app_labels")
	checkEmpty("host_in_house_software_installs")
	checkEmpty("in_house_app_upcoming_activities")
	checkEmpty("upcoming_activities")
}

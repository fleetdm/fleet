package mysql

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestSoftwareTitleIcons(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CreateOrUpdateSoftwareTitleIcon", testCreateOrUpdateSoftwareTitleIcon},
		{"GetSoftwareTitleIcon", testGetSoftwareTitleIcon},
		{"GetTeamIdsForIconStorageId", testGetTeamIdsForIconStorageId},
		{"DeleteSoftwareTitleIcon", testDeleteSoftwareTitleIcon},
		{"ActivityDetailsForSoftwareTitleIcon", testActivityDetailsForSoftwareTitleIcon},
		{"DeleteIconsAssociatedWithTitlesWithoutInstallers", testDeleteIconsAssociatedWithTitlesWithoutInstallers},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func createTeamAndSoftwareTitle(t *testing.T, ctx context.Context, ds *Datastore) (uint, uint, error) {
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	if err != nil {
		return 0, 0, err
	}
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "apps", BundleIdentifier: "foo.bundle.id"},
	}
	_, err = ds.UpdateHostSoftware(ctx, host1.ID, software1)
	if err != nil {
		return 0, 0, err
	}
	var titles []struct{ ID uint }
	err = ds.writer(ctx).Select(&titles, `SELECT id from software_titles`)
	if err != nil {
		return 0, 0, err
	}
	return tm.ID, titles[0].ID, nil
}

func testCreateOrUpdateSoftwareTitleIcon(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID uint
	var err error
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{"Create icon", func(ds *Datastore) {
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			icon, err := ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon.png",
			})
			require.NoError(t, err)
			require.NotNil(t, icon)
			require.Equal(t, teamID, icon.TeamID)
			require.Equal(t, titleID, icon.SoftwareTitleID)
			require.Equal(t, "storage-id-1", icon.StorageID)
			require.Equal(t, "test-icon.png", icon.Filename)
		}},
		{"Update existing icon", func(ds *Datastore) {
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)
			_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon.png",
			})
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			icon, err := ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-2",
				Filename:  "test-icon-updated.png",
			})
			require.NoError(t, err)
			require.NotNil(t, icon)
			require.Equal(t, teamID, icon.TeamID)
			require.Equal(t, titleID, icon.SoftwareTitleID)
			require.Equal(t, "storage-id-2", icon.StorageID)
			require.Equal(t, "test-icon-updated.png", icon.Filename)

			icon, err = ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)
			require.NotNil(t, icon)
			require.Equal(t, teamID, icon.TeamID)
			require.Equal(t, titleID, icon.SoftwareTitleID)
			require.Equal(t, "storage-id-2", icon.StorageID)
			require.Equal(t, "test-icon-updated.png", icon.Filename)
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testGetSoftwareTitleIcon(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID uint
	var err error
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"Icon doesn't exist",
			func(ds *Datastore) {
			}, func(t *testing.T, ds *Datastore) {
				_, err := ds.GetSoftwareTitleIcon(ctx, 1, 1)
				require.Error(t, err)
			},
		},
		{
			"Icon exists",
			func(ds *Datastore) {
				teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   titleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon.png",
				})
				require.NoError(t, err)
			}, func(t *testing.T, ds *Datastore) {
				icon, err := ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
				require.NoError(t, err)
				require.NotNil(t, icon)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testGetTeamIdsForIconStorageId(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID uint
	var err error
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			"no matching storage id exists",
			func(ds *Datastore) {
			}, func(t *testing.T, ds *Datastore) {
				teamIds, err := ds.GetTeamIdsForIconStorageId(ctx, "storage-id")
				require.NoError(t, err)
				require.Nil(t, teamIds)
			},
		},
		{
			"matching storage id exists",
			func(ds *Datastore) {
				teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   titleID,
					StorageID: "storage-id",
					Filename:  "test-icon.png",
				})
				require.NoError(t, err)
			}, func(t *testing.T, ds *Datastore) {
				teamIds, err := ds.GetTeamIdsForIconStorageId(ctx, "storage-id")
				require.NoError(t, err)
				require.Contains(t, teamIds, teamID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testDeleteSoftwareTitleIcon(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	defer TruncateTables(t, ds)

	var teamID, titleID uint
	var err error
	teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
	require.NoError(t, err)

	icon, err := ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
		TeamID:    teamID,
		TitleID:   titleID,
		StorageID: "storage-id-1",
		Filename:  "test-icon.png",
	})
	require.NoError(t, err)
	require.NotNil(t, icon)

	err = ds.DeleteSoftwareTitleIcon(ctx, teamID, titleID)
	require.NoError(t, err)

	icon, err = ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
	require.Error(t, err)
	require.Nil(t, icon)

	err = ds.DeleteSoftwareTitleIcon(ctx, teamID, titleID)
	require.ErrorContains(t, err, "software title icon not found")
}

func testActivityDetailsForSoftwareTitleIcon(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID, installerID uint
	var err error
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{"software installer", func(ds *Datastore) {
			user := test.NewUser(t, ds, "user1", "user1@example.com", false)
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)

			// create a software installer associated to the title
			tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
			require.NoError(t, err)
			installerID, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				InstallScript:    "hello",
				InstallerFile:    tfr1,
				StorageID:        "storage1",
				Filename:         "foo.pkg",
				Title:            "foo",
				Version:          "0.0.3",
				Source:           "apps",
				TeamID:           &teamID,
				UserID:           user.ID,
				BundleIdentifier: "foo.bundle.id",
				ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
			var softwareInstallerTitleIds []struct {
				ID uint `db:"title_id"`
			}
			err = ds.writer(ctx).Select(&softwareInstallerTitleIds, `SELECT title_id from software_installers`)
			require.NoError(t, err)
			require.Len(t, softwareInstallerTitleIds, 1)
			require.Equal(t, titleID, softwareInstallerTitleIds[0].ID)

			// add some labels
			label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
			require.NoError(t, err)
			label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
			require.NoError(t, err)
			// Insert exclude label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO software_installer_labels (software_installer_id, label_id, exclude) VALUES (?, ?, ?)",
				installerID, label1.ID, true)
			require.NoError(t, err)
			// Insert include label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO software_installer_labels (software_installer_id, label_id, exclude) VALUES (?, ?, ?)",
				installerID, label2.ID, false)
			require.NoError(t, err)

			_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon-updated.png",
			})
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			activity, err := ds.ActivityDetailsForSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)

			require.Equal(t, installerID, *activity.SoftwareInstallerID)
			require.Nil(t, activity.AdamID)
			require.Nil(t, activity.VPPAppTeamID)
			require.Nil(t, activity.VPPIconUrl)
			require.Equal(t, "foo", activity.SoftwareTitle)
			require.Equal(t, "foo.pkg", *activity.Filename)
			require.Equal(t, "team1", *activity.TeamName)
			require.Equal(t, teamID, activity.TeamID)
			require.False(t, activity.SelfService)
			require.Equal(t, titleID, activity.SoftwareTitleID)
			require.Len(t, activity.LabelsExcludeAny, 1)
			require.Nil(t, activity.Platform)
			require.Equal(t, "label1", activity.LabelsExcludeAny[0].Name)
			require.Len(t, activity.LabelsIncludeAny, 1)
			require.Equal(t, "label2", activity.LabelsIncludeAny[0].Name)
		}},
		{"vpp app", func(ds *Datastore) {
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)

			dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Test org"+t.Name(), "Test location"+t.Name())
			require.NoError(t, err)
			tok1, err := ds.InsertVPPToken(ctx, dataToken)
			require.NoError(t, err)
			_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
			require.NoError(t, err)

			vppApp := &fleet.VPPApp{
				Name:    "foo",
				TitleID: titleID,
				IconURL: "fleetdm.com/icon.png",
				VPPAppTeam: fleet.VPPAppTeam{
					VPPAppID: fleet.VPPAppID{
						AdamID:   "1",
						Platform: fleet.MacOSPlatform,
					},
				},
				BundleIdentifier: "foo.bundle.id",
			}
			vppApp, err = ds.InsertVPPAppWithTeam(ctx, vppApp, &teamID)
			require.NoError(t, err)

			var vppAppsTitleIds []struct {
				ID uint `db:"title_id"`
			}
			err = ds.writer(ctx).Select(&vppAppsTitleIds, `SELECT title_id from vpp_apps`)
			require.NoError(t, err)
			require.Len(t, vppAppsTitleIds, 1)
			require.Equal(t, titleID, vppAppsTitleIds[0].ID)

			// add some labels
			label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
			require.NoError(t, err)
			label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
			require.NoError(t, err)

			// Insert exclude label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO vpp_app_team_labels (vpp_app_team_id, label_id, exclude) VALUES (?, ?, ?)",
				vppApp.VPPAppTeam.AppTeamID, label1.ID, true)
			require.NoError(t, err)
			// Insert include label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO vpp_app_team_labels (vpp_app_team_id, label_id, exclude) VALUES (?, ?, ?)",
				vppApp.VPPAppTeam.AppTeamID, label2.ID, false)
			require.NoError(t, err)

			_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon-updated.png",
			})
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			activity, err := ds.ActivityDetailsForSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)

			require.Nil(t, activity.SoftwareInstallerID)
			require.Equal(t, "1", *activity.AdamID)
			require.NotEmpty(t, *activity.VPPAppTeamID)
			require.Equal(t, "fleetdm.com/icon.png", *activity.VPPIconUrl)
			require.Equal(t, "foo", activity.SoftwareTitle)
			require.Nil(t, activity.Filename)
			require.Equal(t, "team1", *activity.TeamName)
			require.Equal(t, teamID, activity.TeamID)
			require.False(t, activity.SelfService)
			require.Equal(t, titleID, activity.SoftwareTitleID)
			require.Len(t, activity.LabelsExcludeAny, 1)
			require.Equal(t, fleet.MacOSPlatform, *activity.Platform)
			require.Equal(t, "label1", activity.LabelsExcludeAny[0].Name)
			require.Len(t, activity.LabelsIncludeAny, 1)
			require.Equal(t, "label2", activity.LabelsIncludeAny[0].Name)
		}},
		{"team id 0", func(ds *Datastore) {
			user := test.NewUser(t, ds, "user1", "user1@example.com", false)
			host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
			software1 := []fleet.Software{
				{Name: "foo", Version: "0.0.3", Source: "apps", BundleIdentifier: "foo.bundle.id"},
			}
			_, err = ds.UpdateHostSoftware(ctx, host1.ID, software1)
			require.NoError(t, err)
			var titles []struct{ ID uint }
			err = ds.writer(ctx).Select(&titles, `SELECT id from software_titles`)
			require.NoError(t, err)
			teamID = uint(0)
			titleID = titles[0].ID

			tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
			require.NoError(t, err)
			installerID, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				InstallScript:    "hello",
				InstallerFile:    tfr1,
				StorageID:        "storage1",
				Filename:         "foo.pkg",
				Title:            "foo",
				Version:          "0.0.3",
				Source:           "apps",
				TeamID:           &teamID,
				UserID:           user.ID,
				BundleIdentifier: "foo.bundle.id",
				ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
			var softwareInstallerTitleIds []struct {
				ID             uint  `db:"title_id"`
				TeamID         *uint `db:"team_id"`
				GlobalOrTeamId uint  `db:"global_or_team_id"`
			}
			err = ds.writer(ctx).Select(&softwareInstallerTitleIds, `SELECT title_id, team_id, global_or_team_id from software_installers`)
			require.NoError(t, err)
			require.Len(t, softwareInstallerTitleIds, 1)
			require.Equal(t, titleID, softwareInstallerTitleIds[0].ID)
			require.Nil(t, softwareInstallerTitleIds[0].TeamID)
			require.Equal(t, teamID, softwareInstallerTitleIds[0].GlobalOrTeamId)

			_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon-updated.png",
			})
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			activity, err := ds.ActivityDetailsForSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)

			require.Equal(t, installerID, *activity.SoftwareInstallerID)
			require.Nil(t, activity.AdamID)
			require.Nil(t, activity.VPPAppTeamID)
			require.Nil(t, activity.VPPIconUrl)
			require.Equal(t, "foo", activity.SoftwareTitle)
			require.Equal(t, "foo.pkg", *activity.Filename)
			require.Nil(t, activity.TeamName)
			require.Equal(t, teamID, activity.TeamID)
			require.False(t, activity.SelfService)
			require.Equal(t, titleID, activity.SoftwareTitleID)
			require.Nil(t, activity.Platform)
			require.Nil(t, activity.LabelsExcludeAny)
			require.Nil(t, activity.LabelsIncludeAny)
		}},
		{"in house app", func(ds *Datastore) {
			user := test.NewUser(t, ds, "user1", "user1@example.com", false)
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)

			// create an in-house app that will create two titles
			payload := fleet.UploadSoftwareInstallerPayload{
				TeamID:           &teamID,
				UserID:           user.ID,
				Title:            "foo",
				Filename:         "foo.ipa",
				BundleIdentifier: "foo.bundle.id",
				StorageID:        "testingtesting123",
				Platform:         "ios",
				Extension:        "ipa",
				Version:          "1.2.3",
				ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			}
			installerID, titleID, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
			require.NoError(t, err)
			var softwareInstallerTitleIds []struct {
				ID uint `db:"title_id"`
			}
			err = ds.writer(ctx).Select(&softwareInstallerTitleIds, `SELECT title_id from in_house_apps`)
			require.NoError(t, err)
			require.Len(t, softwareInstallerTitleIds, 2) // two in house app titles
			require.Equal(t, titleID, softwareInstallerTitleIds[1].ID)

			// add some labels
			label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
			require.NoError(t, err)
			label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
			require.NoError(t, err)
			// Insert exclude label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO in_house_app_labels (in_house_app_id, label_id, exclude) VALUES (?, ?, ?)",
				installerID, label1.ID, true)
			require.NoError(t, err)
			// Insert include label
			_, err = ds.writer(ctx).ExecContext(ctx,
				"INSERT INTO in_house_app_labels (in_house_app_id, label_id, exclude) VALUES (?, ?, ?)",
				installerID, label2.ID, false)
			require.NoError(t, err)

			_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
				TeamID:    teamID,
				TitleID:   titleID,
				StorageID: "storage-id-1",
				Filename:  "test-icon-updated.png",
			})
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			activity, err := ds.ActivityDetailsForSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)

			require.Equal(t, installerID, *activity.InHouseAppID)
			require.Nil(t, activity.AdamID)
			require.Nil(t, activity.VPPAppTeamID)
			require.Nil(t, activity.VPPIconUrl)
			require.Equal(t, "foo", activity.SoftwareTitle)
			require.Equal(t, "foo.ipa", *activity.Filename)
			require.Equal(t, "team1", *activity.TeamName)
			require.Equal(t, teamID, activity.TeamID)
			require.False(t, activity.SelfService)
			require.Equal(t, titleID, activity.SoftwareTitleID)
			require.Len(t, activity.LabelsExcludeAny, 1)
			require.Nil(t, activity.Platform)
			require.Equal(t, "label1", activity.LabelsExcludeAny[0].Name)
			require.Len(t, activity.LabelsIncludeAny, 1)
			require.Equal(t, "label2", activity.LabelsIncludeAny[0].Name)
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

func testDeleteIconsAssociatedWithTitlesWithoutInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID, deletedTitleID uint
	var err error
	testCases := []struct {
		name     string
		before   func(ds *Datastore)
		testFunc func(*testing.T, *Datastore)
	}{
		{
			name: "deletes icons associated for software titles without software installers or vpp app",
			before: func(ds *Datastore) {
				teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   titleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T, ds *Datastore) {
				var count int
				err = ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM software_title_icons`)
				require.NoError(t, err)
				require.Equal(t, 1, count)

				err = ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, 1)
				require.NoError(t, err)

				err = ds.writer(ctx).Get(&count, `SELECT COUNT(*) FROM software_title_icons`)
				require.NoError(t, err)
				require.Equal(t, 0, count)
			},
		},
		{
			name: "does not delete icons still associated with a software installer",
			before: func(ds *Datastore) {
				user := test.NewUser(t, ds, "user1", "user1@example.com", false)
				teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
				require.NoError(t, err)

				// create a software installer associated to the title
				tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
				require.NoError(t, err)
				_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
					InstallScript:    "hello",
					InstallerFile:    tfr1,
					StorageID:        "storage1",
					Filename:         "foo.pkg",
					Title:            "foo",
					Version:          "0.0.3",
					Source:           "apps",
					TeamID:           &teamID,
					UserID:           user.ID,
					BundleIdentifier: "foo.bundle.id",
					ValidatedLabels:  &fleet.LabelIdentsWithScope{},
				})
				require.NoError(t, err)
				var softwareInstallerTitleIds []struct {
					ID uint `db:"title_id"`
				}
				err = ds.writer(ctx).Select(&softwareInstallerTitleIds, `SELECT title_id from software_installers`)
				require.NoError(t, err)
				require.Len(t, softwareInstallerTitleIds, 1)
				require.Equal(t, titleID, softwareInstallerTitleIds[0].ID)

				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   titleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
				result, err := ds.writer(ctx).ExecContext(ctx, `
					INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, ?)
				`, "foo2", "apps", "foo2.bundle.id")
				require.NoError(t, err)
				deletedTitleID64, err := result.LastInsertId()
				deletedTitleID = uint(deletedTitleID64) //nolint:gosec
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   deletedTitleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T, ds *Datastore) {
				var titleIds []uint
				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Contains(t, titleIds, titleID)
				require.Contains(t, titleIds, deletedTitleID)

				err = ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, 1)
				require.NoError(t, err)

				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Len(t, titleIds, 1)
				require.Contains(t, titleIds, titleID)
			},
		},
		{
			name: "does not delete icons still associated with a vpp app",
			before: func(ds *Datastore) {
				teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
				require.NoError(t, err)

				dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Test org"+t.Name(), "Test location"+t.Name())
				require.NoError(t, err)
				tok1, err := ds.InsertVPPToken(ctx, dataToken)
				require.NoError(t, err)
				_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
				require.NoError(t, err)

				vppApp := &fleet.VPPApp{
					Name:    "foo",
					TitleID: titleID,
					IconURL: "fleetdm.com/icon.png",
					VPPAppTeam: fleet.VPPAppTeam{
						VPPAppID: fleet.VPPAppID{
							AdamID:   "1",
							Platform: fleet.MacOSPlatform,
						},
					},
					BundleIdentifier: "foo.bundle.id",
				}
				_, err = ds.InsertVPPAppWithTeam(ctx, vppApp, &teamID)
				require.NoError(t, err)

				var vppAppsTitleIds []struct {
					ID uint `db:"title_id"`
				}
				err = ds.writer(ctx).Select(&vppAppsTitleIds, `SELECT title_id from vpp_apps`)
				require.NoError(t, err)
				require.Len(t, vppAppsTitleIds, 1)
				require.Equal(t, titleID, vppAppsTitleIds[0].ID)

				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   titleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
				result, err := ds.writer(ctx).ExecContext(ctx, `
					INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, ?)
				`, "foo2", "apps", "foo2.bundle.id")
				require.NoError(t, err)
				deletedTitleID64, err := result.LastInsertId()
				deletedTitleID = uint(deletedTitleID64) //nolint:gosec
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    teamID,
					TitleID:   deletedTitleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T, ds *Datastore) {
				var titleIds []uint
				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Contains(t, titleIds, titleID)
				require.Contains(t, titleIds, deletedTitleID)

				err = ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, 1)
				require.NoError(t, err)

				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Len(t, titleIds, 1)
				require.Contains(t, titleIds, titleID)
			},
		},
		{
			name: "does not delete icons still associated with an in-house app",
			before: func(ds *Datastore) {
				user := test.NewUser(t, ds, "user1", "user1@example.com", false)
				team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
				require.NoError(t, err)

				// create an in-house app that will create two titles
				payload := fleet.UploadSoftwareInstallerPayload{
					TeamID:           &team.ID,
					UserID:           user.ID,
					Title:            "foo",
					Filename:         "foo.ipa",
					BundleIdentifier: "foo.bundle.id",
					StorageID:        "testingtesting123",
					Platform:         "ios",
					Extension:        "ipa",
					Version:          "1.2.3",
					ValidatedLabels:  &fleet.LabelIdentsWithScope{},
				}
				_, titleID, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
				require.NoError(t, err)
				var softwareInstallerTitleIds []struct {
					ID uint `db:"title_id"`
				}
				err = ds.writer(ctx).Select(&softwareInstallerTitleIds, `SELECT title_id from in_house_apps`)
				require.NoError(t, err)
				require.Len(t, softwareInstallerTitleIds, 2) // iha create 2 titles
				require.Equal(t, titleID, softwareInstallerTitleIds[1].ID)

				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    team.ID,
					TitleID:   titleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)

				result, err := ds.writer(ctx).ExecContext(ctx, `
					INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, ?)
				`, "foo2", "ios_apps", "foo2.bundle.id")
				require.NoError(t, err)
				deletedTitleID64, err := result.LastInsertId()
				deletedTitleID = uint(deletedTitleID64) //nolint:gosec
				require.NoError(t, err)
				_, err = ds.CreateOrUpdateSoftwareTitleIcon(ctx, &fleet.UploadSoftwareTitleIconPayload{
					TeamID:    team.ID,
					TitleID:   deletedTitleID,
					StorageID: "storage-id-1",
					Filename:  "test-icon-updated.png",
				})
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T, ds *Datastore) {
				var titleIds []uint
				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Contains(t, titleIds, titleID)
				require.Contains(t, titleIds, deletedTitleID)

				err = ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, 1)
				require.NoError(t, err)

				err = ds.writer(ctx).Select(&titleIds, `SELECT software_title_id FROM software_title_icons where team_id = ?`, teamID)
				require.NoError(t, err)
				require.Len(t, titleIds, 1)
				require.Contains(t, titleIds, titleID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			tc.before(ds)

			tc.testFunc(t, ds)
		})
	}
}

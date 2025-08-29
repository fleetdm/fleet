package mysql

import (
	"context"
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
		{"DeleteSoftwareTitleIcon", testDeleteSoftwareTitleIcon},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func createIcon(ctx context.Context, ds *Datastore, teamID, softwareTitleID uint, storageID string, filename string) (*fleet.SoftwareTitleIcon, error) {
	icon := &fleet.SoftwareTitleIcon{
		TeamID:          teamID,
		SoftwareTitleID: softwareTitleID,
		StorageID:       storageID,
		Filename:        filename,
	}
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename) VALUES (?, ?, ?, ?)`,
		teamID, softwareTitleID, storageID, filename,
	)
	if err != nil {
		return nil, err
	}
	return icon, nil
}

func createTeamAndSoftwareTitle(t *testing.T, ctx context.Context, ds *Datastore) (uint, uint, error) {
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	if err != nil {
		return 0, 0, err
	}
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
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
			_, err = createIcon(ctx, ds, teamID, titleID, "storage-id-1", "test-icon.png")
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
		{"Icon doesn't exist", func(ds *Datastore) {
		}, func(t *testing.T, ds *Datastore) {
			_, err := ds.GetSoftwareTitleIcon(ctx, 1, 1)
			require.Error(t, err)
		}},
		{"Icon exists", func(ds *Datastore) {
			teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
			require.NoError(t, err)
			_, err = createIcon(ctx, ds, teamID, titleID, "storage-id-1", "test-icon.png")
			require.NoError(t, err)
		}, func(t *testing.T, ds *Datastore) {
			icon, err := ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
			require.NoError(t, err)
			require.NotNil(t, icon)
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

func testDeleteSoftwareTitleIcon(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var teamID, titleID uint
	var err error
	teamID, titleID, err = createTeamAndSoftwareTitle(t, ctx, ds)
	require.NoError(t, err)

	icon, err := createIcon(ctx, ds, teamID, titleID, "storage-id-1", "test-icon.png")
	require.NoError(t, err)
	require.NotNil(t, icon)

	err = ds.DeleteSoftwareTitleIcon(ctx, teamID, titleID)
	require.NoError(t, err)

	icon, err = ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
	require.Error(t, err)
	require.Nil(t, icon)
}

package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestVPP(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"VPPApps", testVPPApps},
		{"GetVPPAppByTeamAndTitleID", testGetVPPAppByTeamAndTitleID},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testVPPApps(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "foobar"})
	require.NoError(t, err)

	// Insert some VPP apps for the team
	app1 := &fleet.VPPApp{Name: "vpp_app_1", AdamID: "1", BundleIdentifier: "b1"}
	app2 := &fleet.VPPApp{Name: "vpp_app_2", AdamID: "2", BundleIdentifier: "b2"}
	err = ds.InsertVPPAppWithTeam(ctx, app1, &team.ID)
	require.NoError(t, err)

	err = ds.InsertVPPAppWithTeam(ctx, app2, &team.ID)
	require.NoError(t, err)

	// Insert some VPP apps for no team
	appNoTeam1 := &fleet.VPPApp{Name: "vpp_no_team_app_1", AdamID: "3", BundleIdentifier: "b3"}
	appNoTeam2 := &fleet.VPPApp{Name: "vpp_no_team_app_2", AdamID: "4", BundleIdentifier: "b4"}
	err = ds.InsertVPPAppWithTeam(ctx, appNoTeam1, nil)
	require.NoError(t, err)
	err = ds.InsertVPPAppWithTeam(ctx, appNoTeam2, nil)
	require.NoError(t, err)

	// Check that getting the assigned apps works
	appSet, err := ds.GetAssignedVPPApps(ctx, &team.ID)
	require.NoError(t, err)
	require.Equal(t, map[string]struct{}{app1.AdamID: {}, app2.AdamID: {}}, appSet)

	appSet, err = ds.GetAssignedVPPApps(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]struct{}{appNoTeam1.AdamID: {}, appNoTeam2.AdamID: {}}, appSet)

	var appTitles []fleet.SoftwareTitle
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &appTitles, `SELECT name, bundle_identifier FROM software_titles WHERE bundle_identifier IN (?,?) ORDER BY bundle_identifier`, app1.BundleIdentifier, app2.BundleIdentifier)
	require.NoError(t, err)
	require.Len(t, appTitles, 2)
	require.Equal(t, app1.BundleIdentifier, *appTitles[0].BundleIdentifier)
	require.Equal(t, app2.BundleIdentifier, *appTitles[1].BundleIdentifier)
	require.Equal(t, app1.Name, appTitles[0].Name)
	require.Equal(t, app2.Name, appTitles[1].Name)
}

func testGetVPPAppByTeamAndTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	// TODO(roberto): replace with actual datastore method(s) once we have them
	createVPPApp := func(adamID string, teamID *uint) uint {
		var titleID int64
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			res, err := tx.ExecContext(
				ctx,
				"INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)",
				uuid.NewString(), uuid.NewString(), "",
			)
			if err != nil {
				return err
			}

			titleID, _ = res.LastInsertId()
			return nil
		})

		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err = tx.ExecContext(
				ctx,
				"INSERT INTO vpp_apps (adam_id, title_id) VALUES (?, ?)",
				adamID,
				titleID,
			)
			return err
		})

		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			var tmID uint
			if teamID != nil {
				tmID = *teamID
			}
			_, err = tx.ExecContext(
				ctx,
				"INSERT INTO vpp_apps_teams (adam_id, team_id, global_or_team_id) VALUES (?, ?, ?)",
				adamID,
				teamID,
				tmID,
			)
			return err
		})

		return uint(titleID)
	}

	var nfe fleet.NotFoundError

	fooTitleID := createVPPApp("foo", &team.ID)
	gotVPPApp, err := ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, fooTitleID, true)
	require.NoError(t, err)
	require.Equal(t, "foo", gotVPPApp.AdamID)
	require.Equal(t, fooTitleID, gotVPPApp.TitleID)
	// title that doesn't exist
	gotVPPApp, err = ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, 999, true)
	require.ErrorAs(t, err, &nfe)

	// create an entry for the global team
	barTitleID := createVPPApp("bar", nil)
	// not found providing the team id
	gotVPPApp, err = ds.GetVPPAppByTeamAndTitleID(ctx, &team.ID, barTitleID, true)
	require.ErrorAs(t, err, &nfe)
	// found for the global team
	gotVPPApp, err = ds.GetVPPAppByTeamAndTitleID(ctx, nil, barTitleID, true)
	require.NoError(t, err)
	require.Equal(t, "bar", gotVPPApp.AdamID)
	require.Equal(t, barTitleID, gotVPPApp.TitleID)
}

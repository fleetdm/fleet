package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestVPP(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"VPPApps", testVPPApps},
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
	app1 := &fleet.VPPApp{Name: "vpp_app_1", AdamID: "1"}
	app2 := &fleet.VPPApp{Name: "vpp_app_2", AdamID: "2"}
	err = ds.InsertVPPAppWithTeam(ctx, app1, &team.ID)
	require.NoError(t, err)

	err = ds.InsertVPPAppWithTeam(ctx, app2, &team.ID)
	require.NoError(t, err)

	// Insert some VPP apps for no team
	appNoTeam1 := &fleet.VPPApp{Name: "vpp_no_team_app_1", AdamID: "3"}
	appNoTeam2 := &fleet.VPPApp{Name: "vpp_no_team_app_2", AdamID: "4"}
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
}

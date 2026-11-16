package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestPatchDeadlineSettings(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetDefault", testPatchDeadlineSettingsGetDefault},
		{"SetGet", testPatchDeadlineSettingsSetGet},
		{"SetReplaces", testPatchDeadlineSettingsSetReplaces},
		{"TeamsIndependent", testPatchDeadlineSettingsTeamsIndependent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPatchDeadlineSettingsGetDefault(t *testing.T, ds *Datastore) {
	for _, teamID := range []uint{0, 42} {
		got, err := ds.GetPatchDeadlineSettings(t.Context(), teamID)
		require.NoError(t, err)
		require.Equal(t, fleet.DefaultPatchDeadlineSettings(teamID), got)
	}
}

func testPatchDeadlineSettingsSetGet(t *testing.T, ds *Datastore) {
	require.NoError(t, ds.SetPatchDeadlineSettings(t.Context(), fleet.PatchDeadlineSettings{
		TeamID: 7, Days: []string{"wednesday"}, Time: "09:30",
	}))
	got, err := ds.GetPatchDeadlineSettings(t.Context(), 7)
	require.NoError(t, err)
	require.Equal(t, &fleet.PatchDeadlineSettings{TeamID: 7, Days: []string{"wednesday"}, Time: "09:30"}, got)

	// The SET column returns days in week order regardless of the order written.
	require.NoError(t, ds.SetPatchDeadlineSettings(t.Context(), fleet.PatchDeadlineSettings{
		TeamID: 7, Days: []string{"friday", "monday"}, Time: "09:30",
	}))
	got, err = ds.GetPatchDeadlineSettings(t.Context(), 7)
	require.NoError(t, err)
	require.Equal(t, []string{"monday", "friday"}, got.Days)
}

func testPatchDeadlineSettingsSetReplaces(t *testing.T, ds *Datastore) {
	require.NoError(t, ds.SetPatchDeadlineSettings(t.Context(), fleet.PatchDeadlineSettings{
		TeamID: 3, Days: []string{"monday", "tuesday"}, Time: "08:00",
	}))
	require.NoError(t, ds.SetPatchDeadlineSettings(t.Context(), fleet.PatchDeadlineSettings{
		TeamID: 3, Days: []string{"sunday"}, Time: "23:59",
	}))
	got, err := ds.GetPatchDeadlineSettings(t.Context(), 3)
	require.NoError(t, err)
	require.Equal(t, &fleet.PatchDeadlineSettings{TeamID: 3, Days: []string{"sunday"}, Time: "23:59"}, got)
}

func testPatchDeadlineSettingsTeamsIndependent(t *testing.T, ds *Datastore) {
	want := []fleet.PatchDeadlineSettings{
		{TeamID: 0, Days: []string{"monday"}, Time: "06:00"},
		{TeamID: 1, Days: []string{"thursday", "friday"}, Time: "12:15"},
		{TeamID: 2, Days: []string{"saturday"}, Time: "18:45"},
	}
	for _, s := range want {
		require.NoError(t, ds.SetPatchDeadlineSettings(t.Context(), s))
	}
	for _, s := range want {
		got, err := ds.GetPatchDeadlineSettings(t.Context(), s.TeamID)
		require.NoError(t, err)
		require.Equal(t, &s, got)
	}
}

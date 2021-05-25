package datastore

import (
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTeamGetSetDelete(t *testing.T, ds kolide.Datastore) {
	var createTests = []struct {
		name, description string
	}{
		{"foo_team", "foobar is the description"},
		{"bar_team", "were you hoping for more?"},
	}

	for _, tt := range createTests {
		t.Run("", func(t *testing.T) {
			team, err := ds.NewTeam(&kolide.Team{
				Name:        tt.name,
				Description: tt.description,
			})
			require.NoError(t, err)
			assert.NotZero(t, team.ID)

			team, err = ds.Team(team.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			team, err = ds.TeamByName(tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			err = ds.DeleteTeam(team.ID)
			require.NoError(t, err)

			team, err = ds.TeamByName(tt.name)
			require.Error(t, err)
		})
	}
}

func testTeamUsers(t *testing.T, ds kolide.Datastore) {
	users := createTestUsers(t, ds)
	user1 := kolide.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID}
	user2 := kolide.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID}

	team1, err := ds.NewTeam(&kolide.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(&kolide.Team{Name: "team2"})
	require.NoError(t, err)

	team1, err = ds.Team(team1.ID)
	require.NoError(t, err)
	assert.Len(t, team1.Users, 0)

	team1Users := []kolide.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(team1)
	require.NoError(t, err)

	team1, err = ds.Team(team1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, team1Users, team1.Users)
	// Ensure team 2 not effected
	team2, err = ds.Team(team2.ID)
	require.NoError(t, err)
	assert.Len(t, team2.Users, 0)

	team1Users = []kolide.TeamUser{
		{User: user2, Role: "maintainer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(team1)
	require.NoError(t, err)
	team1, err = ds.Team(team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)

	team2Users := []kolide.TeamUser{
		{User: user2, Role: "observer"},
	}
	team2.Users = team2Users
	team1, err = ds.SaveTeam(team1)
	require.NoError(t, err)
	team1, err = ds.Team(team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)
	team2, err = ds.SaveTeam(team2)
	require.NoError(t, err)
	team2, err = ds.Team(team2.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team2Users, team2.Users)
}

func testTeamListTeams(t *testing.T, ds kolide.Datastore) {
	users := createTestUsers(t, ds)
	user1 := kolide.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID}
	user2 := kolide.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID}

	team1, err := ds.NewTeam(&kolide.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(&kolide.Team{Name: "team2"})
	require.NoError(t, err)

	teams, err := ds.ListTeams(kolide.ListOptions{})
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "team1", teams[0].Name)
	assert.Equal(t, 0, teams[0].HostCount)
	assert.Equal(t, 0, teams[0].UserCount)

	assert.Equal(t, "team2", teams[1].Name)
	assert.Equal(t, 0, teams[1].HostCount)
	assert.Equal(t, 0, teams[1].UserCount)

	host1 := test.NewHost(t, ds, "1", "1", "1", "1", time.Now())
	host2 := test.NewHost(t, ds, "2", "2", "2", "2", time.Now())
	host3 := test.NewHost(t, ds, "3", "3", "3", "3", time.Now())
	require.NoError(t, ds.AddHostsToTeam(&team1.ID, []uint{host1.ID}))
	require.NoError(t, ds.AddHostsToTeam(&team2.ID, []uint{host2.ID, host3.ID}))

	team1.Users = []kolide.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1, err = ds.SaveTeam(team1)
	require.NoError(t, err)

	team2.Users = []kolide.TeamUser{
		{User: user1, Role: "maintainer"},
	}
	team1, err = ds.SaveTeam(team2)
	require.NoError(t, err)

	teams, err = ds.ListTeams(kolide.ListOptions{})
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "team1", teams[0].Name)
	assert.Equal(t, 1, teams[0].HostCount)
	assert.Equal(t, 2, teams[0].UserCount)

	assert.Equal(t, "team2", teams[1].Name)
	assert.Equal(t, 2, teams[1].HostCount)
	assert.Equal(t, 1, teams[1].UserCount)
}

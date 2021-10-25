package mysql

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeams(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetSetDelete", testTeamsGetSetDelete},
		{"Users", testTeamsUsers},
		{"List", testTeamsList},
		{"Search", testTeamsSearch},
		{"EnrollSecrets", testTeamsEnrollSecrets},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testTeamsGetSetDelete(t *testing.T, ds *Datastore) {
	var createTests = []struct {
		name, description string
	}{
		{"foo_team", "foobar is the description"},
		{"bar_team", "were you hoping for more?"},
	}

	for _, tt := range createTests {
		t.Run(tt.name, func(t *testing.T) {
			team, err := ds.NewTeam(context.Background(), &fleet.Team{
				Name:        tt.name,
				Description: tt.description,
			})
			require.NoError(t, err)
			assert.NotZero(t, team.ID)

			team, err = ds.Team(context.Background(), team.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			team, err = ds.TeamByName(context.Background(), tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			err = ds.DeleteTeam(context.Background(), team.ID)
			require.NoError(t, err)

			team, err = ds.TeamByName(context.Background(), tt.name)
			require.Error(t, err)
		})
	}
}

func testTeamsUsers(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	user1 := fleet.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID}
	user2 := fleet.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.Len(t, team1.Users, 0)

	team1Users := []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)

	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, team1Users, team1.Users)
	// Ensure team 2 not effected
	team2, err = ds.Team(context.Background(), team2.ID)
	require.NoError(t, err)
	assert.Len(t, team2.Users, 0)

	team1Users = []fleet.TeamUser{
		{User: user2, Role: "maintainer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)

	team2Users := []fleet.TeamUser{
		{User: user2, Role: "observer"},
	}
	team2.Users = team2Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)
	team2, err = ds.SaveTeam(context.Background(), team2)
	require.NoError(t, err)
	team2, err = ds.Team(context.Background(), team2.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team2Users, team2.Users)
}

func testTeamsList(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	user1 := fleet.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID, GlobalRole: ptr.String(fleet.RoleAdmin)}
	user2 := fleet.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID, GlobalRole: ptr.String(fleet.RoleObserver)}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	teams, err := ds.ListTeams(context.Background(), fleet.TeamFilter{User: &user1}, fleet.ListOptions{})
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
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{host2.ID, host3.ID}))

	team1.Users = []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)

	team2.Users = []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
	}
	team1, err = ds.SaveTeam(context.Background(), team2)
	require.NoError(t, err)

	teams, err = ds.ListTeams(context.Background(), fleet.TeamFilter{User: &user1}, fleet.ListOptions{})
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "team1", teams[0].Name)
	assert.Equal(t, 1, teams[0].HostCount)
	assert.Equal(t, 2, teams[0].UserCount)

	assert.Equal(t, "team2", teams[1].Name)
	assert.Equal(t, 2, teams[1].HostCount)
	assert.Equal(t, 1, teams[1].UserCount)
}

func testTeamsSearch(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "foobar"})
	require.NoError(t, err)
	team4, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "floobar"})
	require.NoError(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	teams, err := ds.SearchTeams(context.Background(), filter, "")
	require.NoError(t, err)
	assert.Len(t, teams, 4)

	teams, err = ds.SearchTeams(context.Background(), filter, "", team1.ID, team2.ID, team3.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)
	assert.Equal(t, team4.Name, teams[0].Name)

	teams, err = ds.SearchTeams(context.Background(), filter, "oo", team1.ID, team2.ID, team3.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)
	assert.Equal(t, team4.Name, teams[0].Name)

	teams, err = ds.SearchTeams(context.Background(), filter, "oo")
	require.NoError(t, err)
	assert.Len(t, teams, 2)

	teams, err = ds.SearchTeams(context.Background(), filter, "none")
	require.NoError(t, err)
	assert.Len(t, teams, 0)
}

func testTeamsEnrollSecrets(t *testing.T, ds *Datastore) {
	secrets := []*fleet.EnrollSecret{{Secret: "secret1"}, {Secret: "secret2"}}
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team1",
		Secrets: secrets,
	})
	require.NoError(t, err)

	enrollSecrets, err := ds.TeamEnrollSecrets(context.Background(), team1.ID)
	require.NoError(t, err)

	var justSecrets []*fleet.EnrollSecret
	for _, secret := range enrollSecrets {
		require.NotNil(t, secret.TeamID)
		assert.Equal(t, team1.ID, *secret.TeamID)
		justSecrets = append(justSecrets, &fleet.EnrollSecret{Secret: secret.Secret})
	}
	test.ElementsMatchSkipTimestampsID(t, secrets, justSecrets)
}

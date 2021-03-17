package datastore

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCreateUser(t *testing.T, ds kolide.Datastore) {
	var createTests = []struct {
		username, password, email   string
		isAdmin, passwordReset, sso bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false, true},
		{"jason", "foobar", "jason@kolide.co", true, false, false},
	}

	for _, tt := range createTests {
		u := &kolide.User{
			Username:                 tt.username,
			Password:                 []byte(tt.password),
			Admin:                    tt.isAdmin,
			AdminForcedPasswordReset: tt.passwordReset,
			Email:                    tt.email,
			SSOEnabled:               tt.sso,
		}
		user, err := ds.NewUser(u)
		assert.Nil(t, err)

		verify, err := ds.User(tt.username)
		assert.Nil(t, err)

		assert.Equal(t, user.ID, verify.ID)
		assert.Equal(t, tt.username, verify.Username)
		assert.Equal(t, tt.email, verify.Email)
		assert.Equal(t, tt.email, verify.Email)
		assert.Equal(t, tt.sso, verify.SSOEnabled)
	}
}

func testUserByID(t *testing.T, ds kolide.Datastore) {
	users := createTestUsers(t, ds)
	for _, tt := range users {
		returned, err := ds.UserByID(tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
	}

	// test missing user
	_, err := ds.UserByID(10000000000)
	assert.NotNil(t, err)
}

func createTestUsers(t *testing.T, ds kolide.Datastore) []*kolide.User {
	var createTests = []struct {
		username, password, email string
		isAdmin, passwordReset    bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "foobar", "jason@kolide.co", false, false},
	}

	var users []*kolide.User
	for _, tt := range createTests {
		u := &kolide.User{
			Username:                 tt.username,
			Name:                     tt.username,
			Password:                 []byte(tt.password),
			Admin:                    tt.isAdmin,
			AdminForcedPasswordReset: tt.passwordReset,
			Email:                    tt.email,
		}

		user, err := ds.NewUser(u)
		assert.Nil(t, err)

		users = append(users, user)
	}
	assert.NotEmpty(t, users)
	return users
}

func testSaveUser(t *testing.T, ds kolide.Datastore) {
	users := createTestUsers(t, ds)
	testAdminAttribute(t, ds, users)
	testEmailAttribute(t, ds, users)
	testPasswordAttribute(t, ds, users)
}

func testPasswordAttribute(t *testing.T, ds kolide.Datastore, users []*kolide.User) {
	for _, user := range users {
		randomText, err := kolide.RandomText(8) //GenerateRandomText(8)
		assert.Nil(t, err)
		user.Password = []byte(randomText)
		err = ds.SaveUser(user)
		assert.Nil(t, err)

		verify, err := ds.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Password, verify.Password)
	}
}

func testEmailAttribute(t *testing.T, ds kolide.Datastore, users []*kolide.User) {
	for _, user := range users {
		user.Email = fmt.Sprintf("test.%s", user.Email)
		err := ds.SaveUser(user)
		assert.Nil(t, err)

		verify, err := ds.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Email, verify.Email)
	}
}

func testAdminAttribute(t *testing.T, ds kolide.Datastore, users []*kolide.User) {
	for _, user := range users {
		user.Admin = false
		err := ds.SaveUser(user)
		assert.Nil(t, err)

		verify, err := ds.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Admin, verify.Admin)
	}
}

func testListUsers(t *testing.T, ds kolide.Datastore) {
	createTestUsers(t, ds)

	users, err := ds.ListUsers(kolide.ListOptions{})
	assert.NoError(t, err)
	require.Len(t, users, 2)

	users, err = ds.ListUsers(kolide.ListOptions{MatchQuery: "jason"})
	assert.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "jason@kolide.co", users[0].Email)

	users, err = ds.ListUsers(kolide.ListOptions{MatchQuery: "paia"})
	assert.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "mike@kolide.co", users[0].Email)
}

func testUserTeams(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewTeam(&kolide.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	users := createTestUsers(t, ds)

	assert.Len(t, users[0].Teams, 0)
	assert.Len(t, users[1].Teams, 0)

	// Add invalid team should fail
	users[0].Teams = []kolide.UserTeam{
		{
			Team: kolide.Team{ID: 13},
			Role: "foobar",
		},
	}
	err := ds.SaveUser(users[0])
	require.Error(t, err)

	// Add valid team should succeed
	users[0].Teams = []kolide.UserTeam{
		{
			Team: kolide.Team{ID: 3},
			Role: "foobar",
		},
	}
	err = ds.SaveUser(users[0])
	require.NoError(t, err)

	users, err = ds.ListUsers(kolide.ListOptions{OrderKey: "name"})
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	assert.Len(t, users[1].Teams, 0)

	users[1].Teams = []kolide.UserTeam{
		{
			Team: kolide.Team{ID: 1},
			Role: "foobar",
		},
		{
			Team: kolide.Team{ID: 2},
			Role: "foobar",
		},
		{
			Team: kolide.Team{ID: 3},
			Role: "foobar",
		},
	}
	err = ds.SaveUser(users[1])
	require.NoError(t, err)

	users, err = ds.ListUsers(kolide.ListOptions{OrderKey: "name"})
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	assert.Len(t, users[1].Teams, 3)

	// Clear teams
	users[1].Teams = []kolide.UserTeam{}
	err = ds.SaveUser(users[1])
	require.NoError(t, err)

	users, err = ds.ListUsers(kolide.ListOptions{OrderKey: "name"})
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	assert.Len(t, users[1].Teams, 0)
}

func testUserCreateWithTeams(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewTeam(&kolide.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	u := &kolide.User{
		Username: "1",
		Password: []byte("foo"),
		Teams: []kolide.UserTeam{
			{
				Team: kolide.Team{ID: 6},
				Role: "admin",
			},
			{
				Team: kolide.Team{ID: 3},
				Role: "observer",
			},
			{
				Team: kolide.Team{ID: 9},
				Role: "maintainer",
			},
		},
	}
	user, err := ds.NewUser(u)
	assert.Nil(t, err)
	assert.Len(t, user.Teams, 3)

	user, err = ds.UserByID(user.ID)
	require.NoError(t, err)
	assert.Len(t, user.Teams, 3)

	assert.Equal(t, uint(3), user.Teams[0].ID)
	assert.Equal(t, "observer", user.Teams[0].Role)
	assert.Equal(t, uint(6), user.Teams[1].ID)
	assert.Equal(t, "admin", user.Teams[1].Role)
	assert.Equal(t, uint(9), user.Teams[2].ID)
	assert.Equal(t, "maintainer", user.Teams[2].Role)
}

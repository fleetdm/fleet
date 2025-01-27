package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/test"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsers(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Create", testUsersCreate},
		{"ByID", testUsersByID},
		{"Save", testUsersSave},
		{"List", testUsersList},
		{"Teams", testUsersTeams},
		{"CreateWithTeams", testUsersCreateWithTeams},
		{"SaveMany", testUsersSaveMany},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testUsersCreate(t *testing.T, ds *Datastore) {
	createTests := []struct {
		password, email                  string
		isAdmin, passwordReset, sso, mfa bool
		resultingPasswordReset           bool
	}{
		{"foobar", "mike@fleet.co", true, false, true, false, false},
		{"foobar", "jason@fleet.co", true, false, false, true, false},
		{"foobar", "jason2@fleet.co", true, true, true, false, false},
		{"foobar", "jason3@fleet.co", true, true, false, false, true},
	}

	for _, tt := range createTests {
		u := &fleet.User{
			Password:                 []byte(tt.password),
			AdminForcedPasswordReset: tt.passwordReset,
			Email:                    tt.email,
			SSOEnabled:               tt.sso,
			MFAEnabled:               tt.mfa,
			GlobalRole:               ptr.String(fleet.RoleObserver),
		}

		// truncating because we're truncating under the hood to match the DB
		beforeUserCreate := time.Now().Truncate(time.Second)
		user, err := ds.NewUser(context.Background(), u)
		afterUserCreate := time.Now().Truncate(time.Second)
		assert.Nil(t, err)

		assert.LessOrEqual(t, beforeUserCreate, user.CreatedAt)
		assert.LessOrEqual(t, beforeUserCreate, user.UpdatedAt)
		assert.GreaterOrEqual(t, afterUserCreate, user.CreatedAt)
		assert.GreaterOrEqual(t, afterUserCreate, user.UpdatedAt)

		verify, err := ds.UserByEmail(context.Background(), tt.email)
		assert.Nil(t, err)

		assert.Equal(t, user.ID, verify.ID)
		assert.Equal(t, tt.email, verify.Email)
		assert.Equal(t, tt.email, verify.Email)
		assert.Equal(t, tt.sso, verify.SSOEnabled)
		assert.Equal(t, tt.mfa, verify.MFAEnabled)
		assert.Equal(t, tt.resultingPasswordReset, verify.AdminForcedPasswordReset)

		assert.LessOrEqual(t, beforeUserCreate, verify.CreatedAt)
		assert.LessOrEqual(t, beforeUserCreate, verify.UpdatedAt)
		assert.GreaterOrEqual(t, afterUserCreate, verify.CreatedAt)
		assert.GreaterOrEqual(t, afterUserCreate, verify.UpdatedAt)
	}
}

func testUsersByID(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	for _, tt := range users {
		returned, err := ds.UserByID(context.Background(), tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
	}

	// test missing user
	_, err := ds.UserByID(context.Background(), 10000000000)
	assert.NotNil(t, err)
}

func createTestUsers(t *testing.T, ds fleet.Datastore) []*fleet.User {
	createTests := []struct {
		password, email             string
		isAdmin, passwordReset, mfa bool
	}{
		{"foobar", "mike@fleet.co", true, false, true},
		{"foobar", "jason@fleet.co", false, false, false},
	}

	var users []*fleet.User
	for _, tt := range createTests {
		u := &fleet.User{
			Name:                     tt.email,
			Password:                 []byte(tt.password),
			AdminForcedPasswordReset: tt.passwordReset,
			Email:                    tt.email,
			MFAEnabled:               tt.mfa,
			GlobalRole:               ptr.String(fleet.RoleObserver),
		}

		user, err := ds.NewUser(context.Background(), u)
		assert.Nil(t, err)

		users = append(users, user)
	}
	assert.NotEmpty(t, users)
	return users
}

func testUsersSave(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	testUserGlobalRole(t, ds, users)
	testEmailAttribute(t, ds, users)
	testPasswordAttribute(t, ds, users)
	testMFAAttribute(t, ds, users)
	testSettingsAttribute(t, ds, users)
}

func testMFAAttribute(t *testing.T, ds fleet.Datastore, users []*fleet.User) {
	for _, user := range users {
		user.MFAEnabled = true
		err := ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err := ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.True(t, verify.MFAEnabled)

		user.MFAEnabled = false
		err = ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err = ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.False(t, verify.MFAEnabled)
	}
}

func testSettingsAttribute(t *testing.T, ds fleet.Datastore, users []*fleet.User) {
	for _, user := range users {
		user.Settings = &fleet.UserSettings{}
		err := ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err := ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		// settings should only be returned via dedicated method
		assert.Nil(t, verify.Settings)

		user.Settings.HiddenHostColumns = []string{"osquery_version"}
		err = ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		// call the settings db method here
		settings, err := ds.UserSettings(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.Equal(t, settings.HiddenHostColumns, user.Settings.HiddenHostColumns)
	}
}

func testPasswordAttribute(t *testing.T, ds fleet.Datastore, users []*fleet.User) {
	for _, user := range users {
		randomText, err := server.GenerateRandomText(8) // GenerateRandomText(8)
		assert.Nil(t, err)
		user.Password = []byte(randomText)
		err = ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err := ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.Equal(t, user.Password, verify.Password)
	}
}

func testEmailAttribute(t *testing.T, ds fleet.Datastore, users []*fleet.User) {
	for _, user := range users {
		user.Email = fmt.Sprintf("test.%s", user.Email)
		err := ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err := ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.Equal(t, user.Email, verify.Email)
	}
}

func testUserGlobalRole(t *testing.T, ds fleet.Datastore, users []*fleet.User) {
	for _, user := range users {
		user.GlobalRole = ptr.String("admin")
		err := ds.SaveUser(context.Background(), user)
		assert.Nil(t, err)

		verify, err := ds.UserByID(context.Background(), user.ID)
		assert.Nil(t, err)
		assert.Equal(t, user.GlobalRole, verify.GlobalRole)
	}
	err := ds.SaveUser(context.Background(), &fleet.User{
		Name:       "some@email.asd",
		Password:   []byte("asdasd"),
		Email:      "some@email.asd",
		GlobalRole: ptr.String(fleet.RoleObserver),
		Teams:      []fleet.UserTeam{{Role: fleet.RoleMaintainer}},
	})
	var ferr *fleet.Error
	require.True(t, errors.As(err, &ferr))
	assert.Equal(t, "Cannot specify both Global Role and Team Roles", ferr.Message)
}

func testUsersList(t *testing.T, ds *Datastore) {
	createTestUsers(t, ds)

	users, err := ds.ListUsers(context.Background(), fleet.UserListOptions{})
	assert.NoError(t, err)
	require.Len(t, users, 2)

	users, err = ds.ListUsers(context.Background(), fleet.UserListOptions{ListOptions: fleet.ListOptions{MatchQuery: "jason"}})
	assert.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "jason@fleet.co", users[0].Email)

	users, err = ds.ListUsers(context.Background(), fleet.UserListOptions{ListOptions: fleet.ListOptions{MatchQuery: "ike"}})
	assert.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "mike@fleet.co", users[0].Email)
}

func testUsersTeams(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	users := createTestUsers(t, ds)

	assert.Len(t, users[0].Teams, 0)
	assert.Len(t, users[1].Teams, 0)

	// Add invalid team should fail
	users[0].Teams = []fleet.UserTeam{
		{
			Team: fleet.Team{ID: 13},
			Role: "foobar",
		},
	}
	err := ds.SaveUser(context.Background(), users[0])
	require.Error(t, err)

	// Add valid team should succeed
	users[0].Teams = []fleet.UserTeam{
		{
			Team: fleet.Team{ID: 3},
			Role: fleet.RoleObserver,
		},
	}
	users[0].GlobalRole = nil
	err = ds.SaveUser(context.Background(), users[0])
	require.NoError(t, err)

	users, err = ds.ListUsers(
		context.Background(),
		fleet.UserListOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderDescending},
		},
	)
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	// For user with a global role, Teams should be empty
	assert.Len(t, users[1].Teams, 0)

	users[1].Teams = []fleet.UserTeam{
		{
			Team: fleet.Team{ID: 1},
			Role: fleet.RoleObserver,
		},
		{
			Team: fleet.Team{ID: 2},
			Role: fleet.RoleObserver,
		},
		{
			Team: fleet.Team{ID: 3},
			Role: fleet.RoleObserver,
		},
	}
	users[1].GlobalRole = nil
	err = ds.SaveUser(context.Background(), users[1])
	require.NoError(t, err)

	users, err = ds.ListUsers(
		context.Background(),
		fleet.UserListOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderDescending},
		},
	)
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	assert.Len(t, users[1].Teams, 3)

	// Clear teams
	users[1].Teams = []fleet.UserTeam{}
	users[1].GlobalRole = ptr.String(fleet.RoleObserver)
	err = ds.SaveUser(context.Background(), users[1])
	require.NoError(t, err)

	users, err = ds.ListUsers(
		context.Background(),
		fleet.UserListOptions{
			ListOptions: fleet.ListOptions{OrderKey: "name", OrderDirection: fleet.OrderDescending},
		},
	)
	require.NoError(t, err)

	assert.Len(t, users[0].Teams, 1)
	assert.Len(t, users[1].Teams, 0)
}

func testUsersCreateWithTeams(t *testing.T, ds *Datastore) {
	for i := 0; i < 10; i++ {
		_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	u := &fleet.User{
		Password: []byte("foo"),
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 6},
				Role: fleet.RoleObserver,
			},
			{
				Team: fleet.Team{ID: 3},
				Role: fleet.RoleObserver,
			},
			{
				Team: fleet.Team{ID: 9},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	user, err := ds.NewUser(context.Background(), u)
	assert.Nil(t, err)
	assert.Len(t, user.Teams, 3)

	user, err = ds.UserByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Len(t, user.Teams, 3)

	assert.Equal(t, uint(3), user.Teams[0].ID)
	assert.Equal(t, "observer", user.Teams[0].Role)
	assert.Equal(t, uint(6), user.Teams[1].ID)
	assert.Equal(t, "observer", user.Teams[1].Role)
	assert.Equal(t, uint(9), user.Teams[2].ID)
	assert.Equal(t, "maintainer", user.Teams[2].Role)
}

func testUsersSaveMany(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, t.Name()+"Admin1", t.Name()+"admin1@fleet.co", true)
	u2 := test.NewUser(t, ds, t.Name()+"Admin2", t.Name()+"admin2@fleet.co", true)
	u3 := test.NewUser(t, ds, t.Name()+"Admin3", t.Name()+"admin3@fleet.co", true)

	u1.Email += "m"
	u2.Email += "m"
	u3.Email += "m"

	require.NoError(t, ds.SaveUsers(context.Background(), []*fleet.User{u1, u2, u3}))

	gotU1, err := ds.UserByID(context.Background(), u1.ID)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(gotU1.Email, "fleet.com"))

	gotU2, err := ds.UserByID(context.Background(), u3.ID)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(gotU2.Email, "fleet.com"))

	gotU3, err := ds.UserByID(context.Background(), u3.ID)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(gotU3.Email, "fleet.com"))
}

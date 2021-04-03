package datastore

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
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

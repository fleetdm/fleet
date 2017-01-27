package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testChangeEmail(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}
	user := &kolide.User{
		Username: "bob",
		Password: []byte("foobar"),
		Email:    "bob@bob.com",
	}
	user, err := ds.NewUser(user)
	require.Nil(t, err)
	err = ds.PendingEmailChange(user.ID, "xxxx@yyy.com", "abcd12345")
	require.Nil(t, err)
	newMail, err := ds.ConfirmPendingEmailChange("abcd12345")
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", newMail)
	user, err = ds.UserByID(user.ID)
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", user.Email)
	// this should fail because it doesn't exist
	newMail, err = ds.ConfirmPendingEmailChange("abcd12345")
	assert.NotNil(t, err)
}

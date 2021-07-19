package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeEmail(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}
	user := &fleet.User{
		Password:   []byte("foobar"),
		Email:      "bob@bob.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	user, err := ds.NewUser(user)
	require.Nil(t, err)
	err = ds.PendingEmailChange(user.ID, "xxxx@yyy.com", "abcd12345")
	require.Nil(t, err)
	newMail, err := ds.ConfirmPendingEmailChange(user.ID, "abcd12345")
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", newMail)
	user, err = ds.UserByID(user.ID)
	require.Nil(t, err)
	assert.Equal(t, "xxxx@yyy.com", user.Email)
	// this should fail because it doesn't exist
	newMail, err = ds.ConfirmPendingEmailChange(user.ID, "abcd12345")
	assert.NotNil(t, err)

	// test that wrong user can't confirm e-mail change
	err = ds.PendingEmailChange(user.ID, "other@bob.com", "uniquetoken")
	require.Nil(t, err)
	otheruser, err := ds.NewUser(&fleet.User{
		Password:   []byte("supersecret"),
		Email:      "other@bobcom",
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.Nil(t, err)
	_, err = ds.ConfirmPendingEmailChange(otheruser.ID, "uniquetoken")
	assert.NotNil(t, err)

}

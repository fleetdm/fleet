package datastore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCreateInvite(t *testing.T, ds kolide.Datastore) {
	invite := &kolide.Invite{

		Email: "user@foo.com",
		Name:  "user",
		Token: "some_user",
	}

	invite, err := ds.NewInvite(invite)
	require.Nil(t, err)

	verify, err := ds.InviteByEmail(invite.Email)
	require.Nil(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, invite.Email, verify.Email)
}

func setupTestInvites(t *testing.T, ds kolide.Datastore) {

	var err error
	admin := &kolide.Invite{
		Email: "admin@foo.com",
		Admin: true,
		Name:  "Xadmin",
		Token: "admin",
	}

	admin, err = ds.NewInvite(admin)
	require.Nil(t, err)

	for user := 0; user < 23; user++ {
		i := kolide.Invite{
			InvitedBy: admin.ID,
			Email:     fmt.Sprintf("user%d@foo.com", user),
			Admin:     false,
			Name:      fmt.Sprintf("User%02d", user),
			Token:     fmt.Sprintf("usertoken%d", user),
		}

		_, err := ds.NewInvite(&i)
		assert.Nil(t, err, "Failure creating user", user)
	}

}

func testListInvites(t *testing.T, ds kolide.Datastore) {
	// TODO: fix this for inmem
	if ds.Name() == "inmem" {
		fmt.Println("Busted test skipped for inmem")
		return
	}

	setupTestInvites(t, ds)

	opt := kolide.ListOptions{
		Page:           0,
		PerPage:        10,
		OrderDirection: kolide.OrderAscending,
		OrderKey:       "name",
	}

	result, err := ds.ListInvites(opt)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, len(result), 10)
	assert.Equal(t, "User00", result[0].Name)
	assert.Equal(t, "User09", result[9].Name)

	opt.Page = 2
	opt.OrderDirection = kolide.OrderDescending
	result, err = ds.ListInvites(opt)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(result)) // allow for admin we created
	assert.Equal(t, "User00", result[3].Name)

}

func testDeleteInvite(t *testing.T, ds kolide.Datastore) {

	setupTestInvites(t, ds)

	invite, err := ds.InviteByEmail("user0@foo.com")

	assert.Nil(t, err)
	assert.NotNil(t, invite)

	err = ds.DeleteInvite(invite.ID)
	assert.Nil(t, err)

	invite, err = ds.InviteByEmail("user0@foo.com")
	assert.NotNil(t, err)
	assert.Nil(t, invite)

}

func testSaveInvite(t *testing.T, ds kolide.Datastore) {
	setupTestInvites(t, ds)

	invite, err := ds.InviteByEmail("user0@foo.com")
	assert.Nil(t, err)
	assert.NotNil(t, invite)

	invite, err = ds.Invite(invite.ID)
	assert.Nil(t, err)
	assert.NotNil(t, invite)

	invite.Name = "Bob"
	invite.Admin = true

	err = ds.SaveInvite(invite)
	assert.Nil(t, err)

	invite, err = ds.Invite(invite.ID)
	assert.Nil(t, err)
	assert.NotNil(t, invite)
	assert.Equal(t, "Bob", invite.Name)
	assert.True(t, invite.Admin)

}

func testInviteByToken(t *testing.T, ds kolide.Datastore) {
	setupTestInvites(t, ds)

	var inviteTests = []struct {
		token   string
		wantErr error
	}{
		{
			token: "admin",
		},
		{
			token:   "nosuchtoken",
			wantErr: errors.New("Invite with token nosuchtoken was not found in the datastore"),
		},
	}

	for _, tt := range inviteTests {
		t.Run("", func(t *testing.T) {
			invite, err := ds.InviteByToken(tt.token)
			if tt.wantErr != nil {
				require.NotNil(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
				return
			} else {
				require.Nil(t, err)
			}
			assert.NotEqual(t, invite.ID, 0)

		})
	}
}

func testInviteByEmail(t *testing.T, ds kolide.Datastore) {
	setupTestInvites(t, ds)

	var inviteTests = []struct {
		email   string
		wantErr error
	}{
		{
			email: "user0@foo.com",
		},
		{
			email:   "nosuchuser@nosuchdomain.com",
			wantErr: errors.New("Invite with email nosuchuser@nosuchdomain.com was not found in the datastore"),
		},
	}

	for _, tt := range inviteTests {
		t.Run("", func(t *testing.T) {
			invite, err := ds.InviteByEmail(tt.email)
			if tt.wantErr != nil {
				require.NotNil(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
				return
			} else {
				require.Nil(t, err)
			}
			assert.NotEqual(t, invite.ID, 0)

		})
	}
}

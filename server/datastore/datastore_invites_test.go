package datastore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

func testCreateInvite(t *testing.T, ds kolide.Datastore) {
	for i := 0; i < 3; i++ {
		_, err := ds.NewTeam(&kolide.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	invite := &kolide.Invite{

		Email: "user@foo.com",
		Name:  "user",
		Token: "some_user",
		Teams: []kolide.UserTeam{
			{Role: "observer", Team: kolide.Team{ID: 1}},
			{Role: "maintainer", Team: kolide.Team{ID: 3}},
		},
	}

	invite, err := ds.NewInvite(invite)
	require.Nil(t, err)

	verify, err := ds.InviteByEmail(invite.Email)
	require.Nil(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, invite.Email, verify.Email)
	assert.Len(t, invite.Teams, 2)
}

func setupTestInvites(t *testing.T, ds kolide.Datastore) {
	var err error
	admin := &kolide.Invite{
		Email:      "admin@foo.com",
		Admin:      true,
		Name:       "Xadmin",
		Token:      "admin",
		GlobalRole: null.StringFrom("admin"),
	}

	admin, err = ds.NewInvite(admin)
	require.Nil(t, err)

	for user := 0; user < 23; user++ {
		i := kolide.Invite{
			InvitedBy:  admin.ID,
			Email:      fmt.Sprintf("user%d@foo.com", user),
			Admin:      false,
			Name:       fmt.Sprintf("User%02d", user),
			Token:      fmt.Sprintf("usertoken%d", user),
			GlobalRole: null.StringFrom("observer"),
		}

		_, err := ds.NewInvite(&i)
		assert.Nil(t, err, "Failure creating user", user)
	}

}

func testListInvites(t *testing.T, ds kolide.Datastore) {
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
	assert.Equal(t, null.StringFrom("observer"), result[9].GlobalRole)

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

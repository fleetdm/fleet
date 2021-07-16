package mysql

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

func TestCreateInvite(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	for i := 0; i < 3; i++ {
		_, err := ds.NewTeam(&fleet.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	invite := &fleet.Invite{
		Email: "user@foo.com",
		Name:  "user",
		Token: "some_user",
		Teams: []fleet.UserTeam{
			{Role: "observer", Team: fleet.Team{ID: 1}},
			{Role: "maintainer", Team: fleet.Team{ID: 3}},
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

func setupTestInvites(t *testing.T, ds fleet.Datastore) {
	var err error
	admin := &fleet.Invite{
		Email:      "admin@foo.com",
		Name:       "Xadmin",
		Token:      "admin",
		GlobalRole: null.StringFrom("admin"),
	}

	admin, err = ds.NewInvite(admin)
	require.Nil(t, err)

	for user := 0; user < 23; user++ {
		i := fleet.Invite{
			InvitedBy:  admin.ID,
			Email:      fmt.Sprintf("user%d@foo.com", user),
			Name:       fmt.Sprintf("User%02d", user),
			Token:      fmt.Sprintf("usertoken%d", user),
			GlobalRole: null.StringFrom("observer"),
		}

		_, err := ds.NewInvite(&i)
		assert.Nil(t, err, "Failure creating user", user)
	}

}

func TestListInvites(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	setupTestInvites(t, ds)

	opt := fleet.ListOptions{
		Page:           0,
		PerPage:        10,
		OrderDirection: fleet.OrderAscending,
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
	opt.OrderDirection = fleet.OrderDescending
	result, err = ds.ListInvites(opt)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(result)) // allow for admin we created
	assert.Equal(t, "User00", result[3].Name)

}

func TestDeleteInvite(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestInviteByToken(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestInviteByEmail(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

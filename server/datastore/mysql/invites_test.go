package mysql

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

func TestInvites(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Create", testInvitesCreate},
		{"List", testInvitesList},
		{"Delete", testInvitesDelete},
		{"ByToken", testInvitesByToken},
		{"ByEmail", testInvitesByEmail},
		{"Invite", testInvitesInvite},
		{"Update", testInvitesUpdate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testInvitesCreate(t *testing.T, ds *Datastore) {
	for i := 0; i < 3; i++ {
		_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	invite := &fleet.Invite{
		Email:      "user@foo.com",
		Name:       "user",
		Token:      "some_user",
		MFAEnabled: true,
		Teams: []fleet.UserTeam{
			{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
			{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 3}},
		},
	}

	invite, err := ds.NewInvite(context.Background(), invite)
	require.NoError(t, err)

	verify, err := ds.InviteByEmail(context.Background(), invite.Email)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, invite.Email, verify.Email)
	assert.True(t, invite.MFAEnabled)
	assert.Len(t, invite.Teams, 2)

	_, err = ds.NewInvite(context.Background(), &fleet.Invite{
		Email: "anotheruser@foo.com",
		Name:  "anotheruser",
		Token: "anothersome_user",
		Teams: []fleet.UserTeam{
			{Role: fleet.RoleAdmin, Team: fleet.Team{ID: 3}},
		},
	})
	require.NoError(t, err)
}

func setupTestInvites(t *testing.T, ds fleet.Datastore) {
	admin := &fleet.Invite{
		Email:      "admin@foo.com",
		Name:       "Xadmin",
		Token:      "admin",
		GlobalRole: null.StringFrom("admin"),
	}

	admin, err := ds.NewInvite(context.Background(), admin)
	require.NoError(t, err)

	for user := 0; user < 23; user++ {
		i := fleet.Invite{
			InvitedBy:  admin.ID,
			Email:      fmt.Sprintf("user%d@foo.com", user),
			Name:       fmt.Sprintf("User%02d", user),
			Token:      fmt.Sprintf("usertoken%d", user),
			GlobalRole: null.StringFrom("observer"),
		}

		_, err := ds.NewInvite(context.Background(), &i)
		require.NoError(t, err, "Failure creating user", user)
	}
}

func testInvitesList(t *testing.T, ds *Datastore) {
	setupTestInvites(t, ds)

	opt := fleet.ListOptions{
		Page:           0,
		PerPage:        10,
		OrderDirection: fleet.OrderAscending,
		OrderKey:       "name",
	}

	result, err := ds.ListInvites(context.Background(), opt)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, len(result), 10)
	assert.Equal(t, "User00", result[0].Name)
	assert.Equal(t, "User09", result[9].Name)
	assert.Equal(t, null.StringFrom("observer"), result[9].GlobalRole)

	opt.Page = 2
	opt.OrderDirection = fleet.OrderDescending
	result, err = ds.ListInvites(context.Background(), opt)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(result)) // allow for admin we created
	assert.Equal(t, "User00", result[3].Name)
}

func testInvitesDelete(t *testing.T, ds *Datastore) {
	setupTestInvites(t, ds)

	invite, err := ds.InviteByEmail(context.Background(), "user0@foo.com")

	assert.Nil(t, err)
	assert.NotNil(t, invite)

	err = ds.DeleteInvite(context.Background(), invite.ID)
	assert.Nil(t, err)

	invite, err = ds.InviteByEmail(context.Background(), "user0@foo.com")
	assert.NotNil(t, err)
	assert.Nil(t, invite)
}

func testInvitesByToken(t *testing.T, ds *Datastore) {
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
		t.Run(tt.token, func(t *testing.T) {
			invite, err := ds.InviteByToken(context.Background(), tt.token)
			if tt.wantErr != nil {
				require.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				return
			}
			require.Nil(t, err)
			assert.NotEqual(t, invite.ID, 0)
		})
	}
}

func testInvitesByEmail(t *testing.T, ds *Datastore) {
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
		t.Run(tt.email, func(t *testing.T) {
			invite, err := ds.InviteByEmail(context.Background(), tt.email)
			if tt.wantErr != nil {
				require.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.wantErr.Error())
				return
			}
			require.Nil(t, err)
			assert.NotEqual(t, invite.ID, 0)
		})
	}
}

func testInvitesInvite(t *testing.T, ds *Datastore) {
	admin := &fleet.Invite{
		Email:      "admin@foo.com",
		Name:       "Xadmin",
		Token:      "admin",
		GlobalRole: null.StringFrom("admin"),
	}

	admin, err := ds.NewInvite(context.Background(), admin)
	require.NoError(t, err)

	gotI, err := ds.Invite(context.Background(), admin.ID)
	require.NoError(t, err)
	assert.Equal(t, admin.ID, gotI.ID)
}

func testInvitesUpdate(t *testing.T, ds *Datastore) {
	for i := 0; i < 3; i++ {
		_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: fmt.Sprintf("%d", i)})
		require.NoError(t, err)
	}

	invite := &fleet.Invite{
		Email: "user@foo.com",
		Name:  "user",
		Token: "some_user",
		Teams: []fleet.UserTeam{
			{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
			{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 3}},
		},
	}

	sentinelInvite := &fleet.Invite{
		Email: "user@sentinel.com",
		Name:  "sentinel",
		Token: "some_sentinel",
		Teams: []fleet.UserTeam{
			{Role: fleet.RoleAdmin, Team: fleet.Team{ID: 1}},
		},
	}

	invite, err := ds.NewInvite(context.Background(), invite)
	require.NoError(t, err)
	sentinelInvite, err = ds.NewInvite(context.Background(), sentinelInvite)
	require.NoError(t, err)

	invite.Name = "someothername"
	invite.MFAEnabled = true

	invite, err = ds.UpdateInvite(context.Background(), invite.ID, invite)
	require.NoError(t, err)

	verify, err := ds.InviteByEmail(context.Background(), invite.Email)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, verify.Name, "someothername")
	assert.Equal(t, verify.MFAEnabled, true)

	invite.Teams = []fleet.UserTeam{
		{Role: fleet.RoleObserver, Team: fleet.Team{ID: 1}},
		{Role: fleet.RoleMaintainer, Team: fleet.Team{ID: 2}},
	}
	invite.MFAEnabled = false

	invite, err = ds.UpdateInvite(context.Background(), invite.ID, invite)
	require.NoError(t, err)

	verify, err = ds.InviteByEmail(context.Background(), invite.Email)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	require.Len(t, verify.Teams, 2)
	assert.Equal(t, uint(1), verify.Teams[0].ID)
	assert.Equal(t, uint(2), verify.Teams[1].ID)
	assert.False(t, verify.MFAEnabled)

	invite.GlobalRole = null.StringFrom(fleet.RoleAdmin)
	invite.Teams = nil

	invite, err = ds.UpdateInvite(context.Background(), invite.ID, invite)
	require.NoError(t, err)

	verify, err = ds.InviteByEmail(context.Background(), invite.Email)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, null.StringFrom(fleet.RoleAdmin), verify.GlobalRole)
	assert.Len(t, verify.Teams, 0)

	// Make sure it only updates the specified invite
	verify, err = ds.InviteByEmail(context.Background(), sentinelInvite.Email)
	require.NoError(t, err)
	assert.Equal(t, verify.ID, sentinelInvite.ID)
	assert.Equal(t, verify.Name, sentinelInvite.Name)
	require.Len(t, verify.Teams, 1)
	assert.Equal(t, fleet.RoleAdmin, verify.Teams[0].Role)
}

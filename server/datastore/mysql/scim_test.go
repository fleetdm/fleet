package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScim(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ScimUserCreate", testScimUserCreate},
		{"ScimUserByID", testScimUserByID},
		{"ScimUserByUserName", testScimUserByUserName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds, "scim_users", "scim_user_emails")
			c.fn(t, ds)
		})
	}
}

func testScimUserCreate(t *testing.T, ds *Datastore) {
	usersToCreate := []fleet.ScimUser{
		{
			UserName:   "user1",
			ExternalID: nil,
			GivenName:  nil,
			FamilyName: nil,
			Active:     nil,
			Emails:     []fleet.ScimUserEmail{},
		},
		{
			UserName:   "user2",
			ExternalID: ptr.String("ext-123"),
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "john.doe@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "user3",
			ExternalID: ptr.String("ext-456"),
			GivenName:  ptr.String("Jane"),
			FamilyName: ptr.String("Smith"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "jane.personal@example.com",
					Primary: ptr.Bool(false),
					Type:    ptr.String("home"),
				},
				{
					Email:   "jane.smith@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
	}

	for _, u := range usersToCreate {
		var err error
		userCopy := u
		userCopy.ID, err = ds.CreateScimUser(context.Background(), &u)
		assert.Nil(t, err)

		verify, err := ds.ScimUserByUserName(context.Background(), u.UserName)
		assert.Nil(t, err)

		assert.Equal(t, userCopy.ID, verify.ID)
		assert.Equal(t, userCopy.UserName, verify.UserName)
		assert.Equal(t, userCopy.ExternalID, verify.ExternalID)
		assert.Equal(t, userCopy.GivenName, verify.GivenName)
		assert.Equal(t, userCopy.FamilyName, verify.FamilyName)
		assert.Equal(t, userCopy.Active, verify.Active)

		// Verify emails
		assert.Equal(t, len(userCopy.Emails), len(verify.Emails))
		for i, email := range userCopy.Emails {
			assert.Equal(t, email.Email, verify.Emails[i].Email)
			assert.Equal(t, email.Primary, verify.Emails[i].Primary)
			assert.Equal(t, email.Type, verify.Emails[i].Type)
			assert.Equal(t, u.ID, verify.Emails[i].ScimUserID)
		}
	}
}

func testScimUserByID(t *testing.T, ds *Datastore) {
	users := createTestScimUsers(t, ds)
	for _, tt := range users {
		returned, err := ds.ScimUserByID(context.Background(), tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.UserName, returned.UserName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)
		assert.Equal(t, tt.GivenName, returned.GivenName)
		assert.Equal(t, tt.FamilyName, returned.FamilyName)
		assert.Equal(t, tt.Active, returned.Active)

		// Verify emails
		assert.Equal(t, len(tt.Emails), len(returned.Emails))
		for i, email := range tt.Emails {
			assert.Equal(t, email.Email, returned.Emails[i].Email)
			assert.Equal(t, email.Primary, returned.Emails[i].Primary)
			assert.Equal(t, email.Type, returned.Emails[i].Type)
			assert.Equal(t, tt.ID, returned.Emails[i].ScimUserID)
		}
	}

	// test missing user
	_, err := ds.ScimUserByID(context.Background(), 10000000000)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimUserByUserName(t *testing.T, ds *Datastore) {
	users := createTestScimUsers(t, ds)
	for _, tt := range users {
		returned, err := ds.ScimUserByUserName(context.Background(), tt.UserName)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.UserName, returned.UserName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)
		assert.Equal(t, tt.GivenName, returned.GivenName)
		assert.Equal(t, tt.FamilyName, returned.FamilyName)
		assert.Equal(t, tt.Active, returned.Active)

		// Verify emails
		assert.Equal(t, len(tt.Emails), len(returned.Emails))
		for i, email := range tt.Emails {
			assert.Equal(t, email.Email, returned.Emails[i].Email)
			assert.Equal(t, email.Primary, returned.Emails[i].Primary)
			assert.Equal(t, email.Type, returned.Emails[i].Type)
			assert.Equal(t, tt.ID, returned.Emails[i].ScimUserID)
		}
	}

	// test missing user
	_, err := ds.ScimUserByUserName(context.Background(), "nonexistent-user")
	assert.NotNil(t, err)
}

func createTestScimUsers(t *testing.T, ds *Datastore) []*fleet.ScimUser {
	createUsers := []fleet.ScimUser{
		{
			UserName:   "test-user1",
			ExternalID: ptr.String("ext-test-123"),
			GivenName:  ptr.String("Test"),
			FamilyName: ptr.String("User"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "test.user@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "test-user2",
			ExternalID: ptr.String("ext-test-456"),
			GivenName:  ptr.String("Another"),
			FamilyName: ptr.String("User"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "another.personal@example.com",
					Primary: ptr.Bool(false),
					Type:    ptr.String("home"),
				},
				{
					Email:   "another.user@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
	}

	var users []*fleet.ScimUser
	for _, u := range createUsers {
		var err error
		u.ID, err = ds.CreateScimUser(context.Background(), &u)
		require.Nil(t, err)
		users = append(users, &u)
	}
	return users
}

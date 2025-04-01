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
		{"ReplaceScimUser", testReplaceScimUser},
		{"DeleteScimUser", testDeleteScimUser},
		{"ListScimUsers", testListScimUsers},
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

func testReplaceScimUser(t *testing.T, ds *Datastore) {
	// Create a test user
	user := fleet.ScimUser{
		UserName:   "replace-test-user",
		ExternalID: ptr.String("ext-replace-123"),
		GivenName:  ptr.String("Original"),
		FamilyName: ptr.String("User"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "original.user@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
		},
	}

	var err error
	user.ID, err = ds.CreateScimUser(context.Background(), &user)
	require.Nil(t, err)

	// Verify the user was created correctly
	createdUser, err := ds.ScimUserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, user.UserName, createdUser.UserName)
	assert.Equal(t, user.ExternalID, createdUser.ExternalID)
	assert.Equal(t, user.GivenName, createdUser.GivenName)
	assert.Equal(t, user.FamilyName, createdUser.FamilyName)
	assert.Equal(t, user.Active, createdUser.Active)
	assert.Equal(t, 1, len(createdUser.Emails))
	assert.Equal(t, "original.user@example.com", createdUser.Emails[0].Email)

	// Modify the user
	updatedUser := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "replace-test-user",           // Same username
		ExternalID: ptr.String("ext-replace-456"), // Changed external ID
		GivenName:  ptr.String("Updated"),         // Changed given name
		FamilyName: ptr.String("User"),            // Same family name
		Active:     ptr.Bool(false),               // Changed active status
		Emails: []fleet.ScimUserEmail{ // Changed emails
			{
				Email:   "updated.user@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
			{
				Email:   "personal.user@example.com",
				Primary: ptr.Bool(false),
				Type:    ptr.String("home"),
			},
		},
	}

	// Replace the user
	err = ds.ReplaceScimUser(context.Background(), &updatedUser)
	require.Nil(t, err)

	// Verify the user was updated correctly
	replacedUser, err := ds.ScimUserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, updatedUser.UserName, replacedUser.UserName)
	assert.Equal(t, updatedUser.ExternalID, replacedUser.ExternalID)
	assert.Equal(t, updatedUser.GivenName, replacedUser.GivenName)
	assert.Equal(t, updatedUser.FamilyName, replacedUser.FamilyName)
	assert.Equal(t, updatedUser.Active, replacedUser.Active)

	// Verify emails were replaced
	assert.Equal(t, 2, len(replacedUser.Emails))
	assert.Equal(t, "personal.user@example.com", replacedUser.Emails[0].Email) // Alphabetical order
	assert.Equal(t, "updated.user@example.com", replacedUser.Emails[1].Email)

	// Test replacing a non-existent user
	nonExistentUser := fleet.ScimUser{
		ID:         99999, // Non-existent ID
		UserName:   "non-existent",
		ExternalID: ptr.String("ext-non-existent"),
		GivenName:  ptr.String("Non"),
		FamilyName: ptr.String("Existent"),
		Active:     ptr.Bool(true),
	}

	err = ds.ReplaceScimUser(context.Background(), &nonExistentUser)
	assert.True(t, fleet.IsNotFound(err))
}

func testDeleteScimUser(t *testing.T, ds *Datastore) {
	// Create a test user
	user := fleet.ScimUser{
		UserName:   "delete-test-user",
		ExternalID: ptr.String("ext-delete-123"),
		GivenName:  ptr.String("Delete"),
		FamilyName: ptr.String("User"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "delete.user@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
		},
	}

	var err error
	user.ID, err = ds.CreateScimUser(context.Background(), &user)
	require.Nil(t, err)

	// Verify the user was created correctly
	createdUser, err := ds.ScimUserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, user.UserName, createdUser.UserName)

	// Delete the user
	err = ds.DeleteScimUser(context.Background(), user.ID)
	require.Nil(t, err)

	// Verify the user was deleted
	_, err = ds.ScimUserByID(context.Background(), user.ID)
	assert.True(t, fleet.IsNotFound(err))

	// Test deleting a non-existent user
	err = ds.DeleteScimUser(context.Background(), 99999) // Non-existent ID
	assert.True(t, fleet.IsNotFound(err))
}

func testListScimUsers(t *testing.T, ds *Datastore) {
	// Create test users with different attributes and emails
	users := []fleet.ScimUser{
		{
			UserName:   "list-test-user1",
			ExternalID: ptr.String("ext-list-123"),
			GivenName:  ptr.String("List"),
			FamilyName: ptr.String("User1"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "list.user1@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "list-test-user2",
			ExternalID: ptr.String("ext-list-456"),
			GivenName:  ptr.String("List"),
			FamilyName: ptr.String("User2"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "list.user2@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
				{
					Email:   "personal.user2@example.com",
					Primary: ptr.Bool(false),
					Type:    ptr.String("home"),
				},
			},
		},
		{
			UserName:   "different-user3",
			ExternalID: ptr.String("ext-list-789"),
			GivenName:  ptr.String("Different"),
			FamilyName: ptr.String("User3"),
			Active:     ptr.Bool(false),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "different.user3@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
	}

	// Create the users
	for i := range users {
		var err error
		users[i].ID, err = ds.CreateScimUser(context.Background(), &users[i])
		require.Nil(t, err)
	}

	// Test 1: List all users without filters
	allUsers, totalResults, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:    1,
		PerPage: 10,
	})
	require.Nil(t, err)
	assert.Equal(t, 3, len(allUsers))
	assert.Equal(t, uint(3), totalResults)

	// Verify that our test users are in the results
	foundUsers := 0
	for _, u := range allUsers {
		for _, testUser := range users {
			if u.ID == testUser.ID {
				foundUsers++
				break
			}
		}
	}
	assert.Equal(t, 3, foundUsers)

	// Test 2: Pagination - first page with 2 items
	page1Users, totalPage1, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:    1,
		PerPage: 2,
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(page1Users))
	assert.Equal(t, uint(3), totalPage1) // Total should still be 3

	// Test 3: Pagination - second page with 2 items
	page2Users, totalPage2, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:    2,
		PerPage: 2,
	})
	require.Nil(t, err)
	assert.Equal(t, 1, len(page2Users))
	assert.Equal(t, uint(3), totalPage2) // Total should still be 3

	// Verify that page1 and page2 contain different users
	for _, p1User := range page1Users {
		for _, p2User := range page2Users {
			assert.NotEqual(t, p1User.ID, p2User.ID, "Users should not appear on multiple pages")
		}
	}

	// Test 4: Filter by username
	listUsers, totalListUsers, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:           1,
		PerPage:        10,
		UserNameFilter: ptr.String("list-test-user2"),
	})

	require.Nil(t, err)
	require.Len(t, listUsers, 1)
	assert.Equal(t, uint(1), totalListUsers)
	assert.Equal(t, "list-test-user2", listUsers[0].UserName)

	// Test 5: Filter by email type and value
	homeEmailUsers, totalHomeEmailUsers, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:             1,
		PerPage:          10,
		EmailTypeFilter:  ptr.String("home"),
		EmailValueFilter: ptr.String("personal.user2@example.com"),
	})
	require.Nil(t, err)
	require.Len(t, homeEmailUsers, 1)
	assert.Equal(t, uint(1), totalHomeEmailUsers)
	assert.Equal(t, users[1].ID, homeEmailUsers[0].ID)
	assert.Equal(t, 2, len(homeEmailUsers[0].Emails))

	// Test 6: Filter by email type and value - work emails
	workEmailUsers, totalWorkEmailUsers, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:             1,
		PerPage:          10,
		EmailTypeFilter:  ptr.String("work"),
		EmailValueFilter: ptr.String("different.user3@example.com"),
	})
	require.Nil(t, err)
	assert.Len(t, workEmailUsers, 1)
	assert.Equal(t, uint(1), totalWorkEmailUsers)

	// Test 7: No results for non-matching filters
	noUsers, totalNoUsers1, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:           1,
		PerPage:        10,
		UserNameFilter: ptr.String("nonexistent"),
	})
	require.Nil(t, err)
	assert.Empty(t, noUsers)
	assert.Equal(t, uint(0), totalNoUsers1)

	noUsers, totalNoUsers2, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		Page:             1,
		PerPage:          10,
		EmailTypeFilter:  ptr.String("nonexistent"),
		EmailValueFilter: ptr.String("nonexistent"),
	})
	require.Nil(t, err)
	assert.Empty(t, noUsers)
	assert.Equal(t, uint(0), totalNoUsers2)
}

package mysql

import (
	"context"
	"sort"
	"strings"
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
		{"ScimUserCreateValidation", testScimUserCreateValidation},
		{"ScimUserByID", testScimUserByID},
		{"ScimUserByUserName", testScimUserByUserName},
		{"ReplaceScimUser", testReplaceScimUser},
		{"ReplaceScimUserValidation", testScimUserReplaceValidation},
		{"DeleteScimUser", testDeleteScimUser},
		{"ListScimUsers", testListScimUsers},
		{"ScimGroupCreate", testScimGroupCreate},
		{"ScimGroupCreateValidation", testScimGroupCreateValidation},
		{"ScimGroupByID", testScimGroupByID},
		{"ScimGroupByDisplayName", testScimGroupByDisplayName},
		{"ReplaceScimGroup", testReplaceScimGroup},
		{"ReplaceScimGroupValidation", testScimGroupReplaceValidation},
		{"DeleteScimGroup", testDeleteScimGroup},
		{"ListScimGroups", testListScimGroups},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds, "scim_users", "scim_user_emails", "scim_groups", "scim_user_group")
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

	// Create test groups and associate them with users
	groups := createTestScimGroups(t, ds, []uint{users[0].ID, users[1].ID})

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

		// Verify groups
		// User 0 and 1 should be in groups, User 2 should not be in any group
		if tt.ID == users[0].ID || tt.ID == users[1].ID {
			assert.NotEmpty(t, returned.Groups, "User should have groups")

			// Check if the user is in the expected groups
			var foundInGroups bool
			for _, group := range groups {
				for _, userID := range group.ScimUsers {
					if userID == tt.ID {
						foundInGroups = true

						// Verify the group ID is in the user's Groups field
						var foundGroupID bool
						for _, groupID := range returned.Groups {
							if groupID == group.ID {
								foundGroupID = true
								break
							}
						}
						assert.True(t, foundGroupID, "User's Groups field should contain the group ID")
						break
					}
				}
				if foundInGroups {
					break
				}
			}
			assert.True(t, foundInGroups, "User should be found in at least one group")
		} else {
			assert.Empty(t, returned.Groups, "User should not have any groups")
		}
	}

	// test missing user
	_, err := ds.ScimUserByID(context.Background(), 10000000000)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimUserByUserName(t *testing.T, ds *Datastore) {
	users := createTestScimUsers(t, ds)

	// Create test groups and associate them with users
	groups := createTestScimGroups(t, ds, []uint{users[0].ID, users[1].ID})

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

		// Verify groups
		// User 0 and 1 should be in groups, User 2 should not be in any group
		if tt.ID == users[0].ID || tt.ID == users[1].ID {
			assert.NotEmpty(t, returned.Groups, "User should have groups")

			// Check if the user is in the expected groups
			var foundInGroups bool
			for _, group := range groups {
				for _, userID := range group.ScimUsers {
					if userID == tt.ID {
						foundInGroups = true

						// Verify the group ID is in the user's Groups field
						var foundGroupID bool
						for _, groupID := range returned.Groups {
							if groupID == group.ID {
								foundGroupID = true
								break
							}
						}
						assert.True(t, foundGroupID, "User's Groups field should contain the group ID")
						break
					}
				}
				if foundInGroups {
					break
				}
			}
			assert.True(t, foundInGroups, "User should be found in at least one group")
		} else {
			assert.Empty(t, returned.Groups, "User should not have any groups")
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

	// Create a test group and associate it with the user
	group := fleet.ScimGroup{
		DisplayName: "Test Group for User",
		ExternalID:  ptr.String("ext-group-for-user"),
		ScimUsers:   []uint{user.ID},
	}
	group.ID, err = ds.CreateScimGroup(context.Background(), &group)
	require.Nil(t, err)

	// Verify the user was created correctly and has the group
	createdUser, err := ds.ScimUserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, user.UserName, createdUser.UserName)
	assert.Equal(t, user.ExternalID, createdUser.ExternalID)
	assert.Equal(t, user.GivenName, createdUser.GivenName)
	assert.Equal(t, user.FamilyName, createdUser.FamilyName)
	assert.Equal(t, user.Active, createdUser.Active)
	assert.Equal(t, 1, len(createdUser.Emails))
	assert.Equal(t, "original.user@example.com", createdUser.Emails[0].Email)

	// Verify the user has the group
	require.Len(t, createdUser.Groups, 1)
	assert.Equal(t, group.ID, createdUser.Groups[0])

	// Modify the user and attempt to modify the Groups field
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
		Groups: []uint{999}, // Attempt to modify Groups (should be ignored)
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

	// Verify that the Groups field was NOT modified (it should still contain the original group)
	require.Len(t, replacedUser.Groups, 1, "Groups field should not be modified by ReplaceScimUser")
	assert.Equal(t, group.ID, replacedUser.Groups[0], "Groups field should still contain the original group")

	// Now remove the user from the group using the group methods
	updatedGroup := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: group.DisplayName,
		ExternalID:  group.ExternalID,
		ScimUsers:   []uint{}, // Remove the user
	}
	err = ds.ReplaceScimGroup(context.Background(), &updatedGroup)
	require.Nil(t, err)

	// Verify that the user no longer has the group
	userAfterGroupUpdate, err := ds.ScimUserByID(context.Background(), user.ID)
	require.Nil(t, err)
	assert.Empty(t, userAfterGroupUpdate.Groups, "User should no longer have any groups")

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

	// Create a group and associate it with the first user
	group := fleet.ScimGroup{
		DisplayName: "Test Group for ListUsers",
		ExternalID:  ptr.String("ext-group-for-list"),
		ScimUsers:   []uint{users[0].ID},
	}
	var err error
	group.ID, err = ds.CreateScimGroup(context.Background(), &group)
	require.Nil(t, err)

	// Test 1: List all users without filters
	allUsers, totalResults, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
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

				// Verify Groups field for the first user
				if testUser.ID == users[0].ID {
					require.Len(t, u.Groups, 1, "First user should have exactly one group")
					assert.Equal(t, group.ID, u.Groups[0], "First user should be in the test group")
				} else {
					assert.Empty(t, u.Groups, "Other users should not have groups")
				}

				break
			}
		}
	}
	assert.Equal(t, 3, foundUsers)

	// Test 2: Pagination - first page with 2 items
	page1Users, totalPage1, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    2,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(page1Users))
	assert.Equal(t, uint(3), totalPage1) // Total should still be 3

	// Test 3: Pagination - second page with 2 items
	page2Users, totalPage2, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 3, // StartIndex is 1-based, so for the second page with 2 items per page, we start at index 3
			PerPage:    2,
		},
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
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		UserNameFilter: ptr.String("list-test-user2"),
	})

	require.Nil(t, err)
	require.Len(t, listUsers, 1)
	assert.Equal(t, uint(1), totalListUsers)
	assert.Equal(t, "list-test-user2", listUsers[0].UserName)

	// Test 5: Filter by email type and value
	homeEmailUsers, totalHomeEmailUsers, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
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
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		EmailTypeFilter:  ptr.String("work"),
		EmailValueFilter: ptr.String("different.user3@example.com"),
	})
	require.Nil(t, err)
	assert.Len(t, workEmailUsers, 1)
	assert.Equal(t, uint(1), totalWorkEmailUsers)

	// Test 7: No results for non-matching filters
	noUsers, totalNoUsers1, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		UserNameFilter: ptr.String("nonexistent"),
	})
	require.Nil(t, err)
	assert.Empty(t, noUsers)
	assert.Equal(t, uint(0), totalNoUsers1)

	noUsers, totalNoUsers2, err := ds.ListScimUsers(context.Background(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		EmailTypeFilter:  ptr.String("nonexistent"),
		EmailValueFilter: ptr.String("nonexistent"),
	})
	require.Nil(t, err)
	assert.Empty(t, noUsers)
	assert.Equal(t, uint(0), totalNoUsers2)
}

func testScimGroupCreate(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	groupsToCreate := []fleet.ScimGroup{
		{
			DisplayName: "Group1",
			ExternalID:  nil,
			ScimUsers:   []uint{},
		},
		{
			DisplayName: "Group2",
			ExternalID:  ptr.String("ext-group-123"),
			ScimUsers:   []uint{userIDs[0]},
		},
		{
			DisplayName: "Group3",
			ExternalID:  ptr.String("ext-group-456"),
			ScimUsers:   userIDs,
		},
	}

	for _, g := range groupsToCreate {
		var err error
		groupCopy := g
		groupCopy.ID, err = ds.CreateScimGroup(context.Background(), &g)
		assert.Nil(t, err)

		verify, err := ds.ScimGroupByID(context.Background(), g.ID)
		assert.Nil(t, err)

		assert.Equal(t, groupCopy.ID, verify.ID)
		assert.Equal(t, groupCopy.DisplayName, verify.DisplayName)
		assert.Equal(t, groupCopy.ExternalID, verify.ExternalID)

		// Verify users
		assert.Equal(t, len(groupCopy.ScimUsers), len(verify.ScimUsers))
		if len(groupCopy.ScimUsers) > 0 {
			// Sort the user IDs for comparison
			sort.Slice(groupCopy.ScimUsers, func(i, j int) bool {
				return groupCopy.ScimUsers[i] < groupCopy.ScimUsers[j]
			})
			sort.Slice(verify.ScimUsers, func(i, j int) bool {
				return verify.ScimUsers[i] < verify.ScimUsers[j]
			})
			assert.Equal(t, groupCopy.ScimUsers, verify.ScimUsers)
		}
	}
}

func testScimGroupCreateValidation(t *testing.T, ds *Datastore) {
	// Test validation for ExternalID
	longString := strings.Repeat("a", SCIMMaxFieldLength+1) // String longer than allowed

	// Test ExternalID validation
	groupWithLongExternalID := fleet.ScimGroup{
		DisplayName: "Valid Name",
		ExternalID:  ptr.String(longString),
		ScimUsers:   []uint{},
	}
	_, err := ds.CreateScimGroup(context.Background(), &groupWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test DisplayName validation
	groupWithLongDisplayName := fleet.ScimGroup{
		DisplayName: longString,
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	_, err = ds.CreateScimGroup(context.Background(), &groupWithLongDisplayName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "display_name exceeds maximum length")

	// Test with valid values
	validGroup := fleet.ScimGroup{
		DisplayName: "Valid Name",
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	_, err = ds.CreateScimGroup(context.Background(), &validGroup)
	assert.NoError(t, err)
}

func testScimGroupByID(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Create test groups
	groups := createTestScimGroups(t, ds, userIDs)

	// Test retrieving each group
	for _, tt := range groups {
		returned, err := ds.ScimGroupByID(context.Background(), tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.DisplayName, returned.DisplayName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)

		// Verify users
		assert.Equal(t, len(tt.ScimUsers), len(returned.ScimUsers))
		if len(tt.ScimUsers) > 0 {
			// Sort the user IDs for comparison
			sort.Slice(tt.ScimUsers, func(i, j int) bool {
				return tt.ScimUsers[i] < tt.ScimUsers[j]
			})
			sort.Slice(returned.ScimUsers, func(i, j int) bool {
				return returned.ScimUsers[i] < returned.ScimUsers[j]
			})
			assert.Equal(t, tt.ScimUsers, returned.ScimUsers)
		}
	}

	// Test missing group
	_, err := ds.ScimGroupByID(context.Background(), 10000000000)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimGroupByDisplayName(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Create test groups
	groups := createTestScimGroups(t, ds, userIDs)

	// Test retrieving each group by display name
	for _, tt := range groups {
		returned, err := ds.ScimGroupByDisplayName(context.Background(), tt.DisplayName)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.DisplayName, returned.DisplayName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)

		// Verify users
		assert.Equal(t, len(tt.ScimUsers), len(returned.ScimUsers))
		if len(tt.ScimUsers) > 0 {
			// Sort the user IDs for comparison
			sort.Slice(tt.ScimUsers, func(i, j int) bool {
				return tt.ScimUsers[i] < tt.ScimUsers[j]
			})
			sort.Slice(returned.ScimUsers, func(i, j int) bool {
				return returned.ScimUsers[i] < returned.ScimUsers[j]
			})
			assert.Equal(t, tt.ScimUsers, returned.ScimUsers)
		}
	}

	// Test missing group
	_, err := ds.ScimGroupByDisplayName(context.Background(), "Nonexistent Group")
	assert.True(t, fleet.IsNotFound(err))
}

// createTestScimGroups creates test SCIM groups for testing
func createTestScimGroups(t *testing.T, ds *Datastore, userIDs []uint) []*fleet.ScimGroup {
	createGroups := []fleet.ScimGroup{
		{
			DisplayName: "Test Group 1",
			ExternalID:  ptr.String("ext-test-group-123"),
			ScimUsers:   []uint{},
		},
		{
			DisplayName: "Test Group 2",
			ExternalID:  ptr.String("ext-test-group-456"),
			ScimUsers:   []uint{userIDs[0]},
		},
		{
			DisplayName: "Test Group 3",
			ExternalID:  ptr.String("ext-test-group-789"),
			ScimUsers:   userIDs,
		},
	}

	var groups []*fleet.ScimGroup
	for _, g := range createGroups {
		var err error
		g.ID, err = ds.CreateScimGroup(context.Background(), &g)
		require.Nil(t, err)
		groups = append(groups, &g)
	}
	return groups
}

func testReplaceScimGroup(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Create a test group
	group := fleet.ScimGroup{
		DisplayName: "Replace Test Group",
		ExternalID:  ptr.String("ext-replace-group-123"),
		ScimUsers:   []uint{userIDs[0]},
	}

	var err error
	group.ID, err = ds.CreateScimGroup(context.Background(), &group)
	require.Nil(t, err)

	// Verify the group was created correctly
	createdGroup, err := ds.ScimGroupByID(context.Background(), group.ID)
	require.Nil(t, err)
	assert.Equal(t, group.DisplayName, createdGroup.DisplayName)
	assert.Equal(t, group.ExternalID, createdGroup.ExternalID)
	assert.Equal(t, 1, len(createdGroup.ScimUsers))
	assert.Equal(t, userIDs[0], createdGroup.ScimUsers[0])

	// Modify the group
	updatedGroup := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: "Updated Group",
		ExternalID:  ptr.String("ext-replace-group-456"),
		ScimUsers:   userIDs, // Add all users
	}

	// Replace the group
	err = ds.ReplaceScimGroup(context.Background(), &updatedGroup)
	require.Nil(t, err)

	// Verify the group was updated correctly
	replacedGroup, err := ds.ScimGroupByID(context.Background(), group.ID)
	require.Nil(t, err)
	assert.Equal(t, updatedGroup.DisplayName, replacedGroup.DisplayName)
	assert.Equal(t, updatedGroup.ExternalID, replacedGroup.ExternalID)

	// Verify users were updated
	assert.Equal(t, len(userIDs), len(replacedGroup.ScimUsers))

	// Sort the user IDs for comparison
	sort.Slice(userIDs, func(i, j int) bool {
		return userIDs[i] < userIDs[j]
	})
	sort.Slice(replacedGroup.ScimUsers, func(i, j int) bool {
		return replacedGroup.ScimUsers[i] < replacedGroup.ScimUsers[j]
	})
	assert.Equal(t, userIDs, replacedGroup.ScimUsers)

	// Test replacing a non-existent group
	nonExistentGroup := fleet.ScimGroup{
		ID:          99999, // Non-existent ID
		DisplayName: "Non-existent",
		ExternalID:  ptr.String("ext-non-existent"),
		ScimUsers:   []uint{},
	}

	err = ds.ReplaceScimGroup(context.Background(), &nonExistentGroup)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimGroupReplaceValidation(t *testing.T, ds *Datastore) {
	// Create a valid group first
	group := fleet.ScimGroup{
		DisplayName: "Validation Test Group",
		ExternalID:  ptr.String("ext-validation-group"),
		ScimUsers:   []uint{},
	}

	var err error
	group.ID, err = ds.CreateScimGroup(context.Background(), &group)
	require.NoError(t, err)

	// Test validation for ExternalID
	longString := strings.Repeat("a", SCIMMaxFieldLength+1) // String longer than allowed

	// Test ExternalID validation
	groupWithLongExternalID := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: "Valid Name",
		ExternalID:  ptr.String(longString),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(context.Background(), &groupWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test DisplayName validation
	groupWithLongDisplayName := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: longString,
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(context.Background(), &groupWithLongDisplayName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "display_name exceeds maximum length")

	// Test with valid values
	validGroup := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: "Updated Valid Name",
		ExternalID:  ptr.String("updated-valid-external-id"),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(context.Background(), &validGroup)
	assert.NoError(t, err)
}

func testDeleteScimGroup(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Create a test group
	group := fleet.ScimGroup{
		DisplayName: "Delete Test Group",
		ExternalID:  ptr.String("ext-delete-group"),
		ScimUsers:   userIDs,
	}

	var err error
	group.ID, err = ds.CreateScimGroup(context.Background(), &group)
	require.Nil(t, err)

	// Verify the group was created correctly
	createdGroup, err := ds.ScimGroupByID(context.Background(), group.ID)
	require.Nil(t, err)
	assert.Equal(t, group.DisplayName, createdGroup.DisplayName)

	// Delete the group
	err = ds.DeleteScimGroup(context.Background(), group.ID)
	require.Nil(t, err)

	// Verify the group was deleted
	_, err = ds.ScimGroupByID(context.Background(), group.ID)
	assert.True(t, fleet.IsNotFound(err))

	// Test deleting a non-existent group
	err = ds.DeleteScimGroup(context.Background(), 99999) // Non-existent ID
	assert.True(t, fleet.IsNotFound(err))
}

func testListScimGroups(t *testing.T, ds *Datastore) {
	// Create test users first
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Create test groups
	groups := []fleet.ScimGroup{
		{
			DisplayName: "List Test Group 1",
			ExternalID:  ptr.String("ext-list-group-123"),
			ScimUsers:   []uint{},
		},
		{
			DisplayName: "List Test Group 2",
			ExternalID:  ptr.String("ext-list-group-456"),
			ScimUsers:   []uint{userIDs[0]},
		},
		{
			DisplayName: "List Test Group 3",
			ExternalID:  ptr.String("ext-list-group-789"),
			ScimUsers:   userIDs,
		},
	}

	// Create the groups
	for i := range groups {
		var err error
		groups[i].ID, err = ds.CreateScimGroup(context.Background(), &groups[i])
		require.Nil(t, err)
	}

	// Test 1: List all groups
	allGroups, totalResults, err := ds.ListScimGroups(context.Background(), fleet.ScimListOptions{
		StartIndex: 1,
		PerPage:    10,
	})
	require.Nil(t, err)
	assert.GreaterOrEqual(t, len(allGroups), 3) // There might be other groups from previous tests
	assert.GreaterOrEqual(t, totalResults, uint(3))

	// Verify that our test groups are in the results
	foundGroups := 0
	for _, g := range allGroups {
		for _, testGroup := range groups {
			if g.ID == testGroup.ID {
				foundGroups++
				break
			}
		}
	}
	assert.Equal(t, 3, foundGroups)

	// Test 2: Pagination - first page with 2 items
	page1Groups, totalPage1, err := ds.ListScimGroups(context.Background(), fleet.ScimListOptions{
		StartIndex: 1,
		PerPage:    2,
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(page1Groups))
	assert.GreaterOrEqual(t, totalPage1, uint(3)) // Total should be at least 3

	// Test 3: Pagination - second page with 2 items
	page2Groups, totalPage2, err := ds.ListScimGroups(context.Background(), fleet.ScimListOptions{
		StartIndex: 3, // StartIndex is 1-based, so for the second page with 2 items per page, we start at index 3
		PerPage:    2,
	})
	require.Nil(t, err)
	assert.GreaterOrEqual(t, len(page2Groups), 1) // At least 1 item on the second page
	assert.GreaterOrEqual(t, totalPage2, uint(3)) // Total should be at least 3

	// Verify that page1 and page2 contain different groups
	for _, p1Group := range page1Groups {
		for _, p2Group := range page2Groups {
			assert.NotEqual(t, p1Group.ID, p2Group.ID, "Groups should not appear on multiple pages")
		}
	}
}

func testScimUserCreateValidation(t *testing.T, ds *Datastore) {
	// Test validation for ExternalID
	longString := strings.Repeat("a", SCIMMaxFieldLength+1) // String longer than SCIMMaxFieldLength

	// Test ExternalID validation
	userWithLongExternalID := fleet.ScimUser{
		UserName:   "valid-username",
		ExternalID: ptr.String(longString),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	_, err := ds.CreateScimUser(context.Background(), &userWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test UserName validation
	userWithLongUserName := fleet.ScimUser{
		UserName:   longString,
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	_, err = ds.CreateScimUser(context.Background(), &userWithLongUserName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_name exceeds maximum length")

	// Test GivenName validation
	userWithLongGivenName := fleet.ScimUser{
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String(longString),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	_, err = ds.CreateScimUser(context.Background(), &userWithLongGivenName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "given_name exceeds maximum length")

	// Test FamilyName validation
	userWithLongFamilyName := fleet.ScimUser{
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String(longString),
		Active:     ptr.Bool(true),
	}
	_, err = ds.CreateScimUser(context.Background(), &userWithLongFamilyName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "family_name exceeds maximum length")

	// Test with valid values
	validUser := fleet.ScimUser{
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	_, err = ds.CreateScimUser(context.Background(), &validUser)
	assert.NoError(t, err)
}

func testScimUserReplaceValidation(t *testing.T, ds *Datastore) {
	// Create a valid user first
	user := fleet.ScimUser{
		UserName:   "replace-validation-user",
		ExternalID: ptr.String("ext-replace-validation"),
		GivenName:  ptr.String("Original"),
		FamilyName: ptr.String("User"),
		Active:     ptr.Bool(true),
	}

	var err error
	user.ID, err = ds.CreateScimUser(context.Background(), &user)
	require.NoError(t, err)

	// Test validation for ExternalID
	longString := strings.Repeat("a", SCIMMaxFieldLength+1) // String longer than SCIMMaxFieldLength

	// Test ExternalID validation
	userWithLongExternalID := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "valid-username",
		ExternalID: ptr.String(longString),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	err = ds.ReplaceScimUser(context.Background(), &userWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test UserName validation
	userWithLongUserName := fleet.ScimUser{
		ID:         user.ID,
		UserName:   longString,
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	err = ds.ReplaceScimUser(context.Background(), &userWithLongUserName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_name exceeds maximum length")

	// Test GivenName validation
	userWithLongGivenName := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String(longString),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	err = ds.ReplaceScimUser(context.Background(), &userWithLongGivenName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "given_name exceeds maximum length")

	// Test FamilyName validation
	userWithLongFamilyName := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String(longString),
		Active:     ptr.Bool(true),
	}
	err = ds.ReplaceScimUser(context.Background(), &userWithLongFamilyName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "family_name exceeds maximum length")

	// Test with valid values
	validUser := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "updated-username",
		ExternalID: ptr.String("updated-external-id"),
		GivenName:  ptr.String("Updated"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	err = ds.ReplaceScimUser(context.Background(), &validUser)
	assert.NoError(t, err)
}

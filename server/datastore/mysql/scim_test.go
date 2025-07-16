package mysql

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		{"ScimUserByUserNameOrEmail", testScimUserByUserNameOrEmail},
		{"ScimUserByHostID", testScimUserByHostID},
		{"ReplaceScimUser", testReplaceScimUser},
		{"ReplaceScimUserEmails", testReplaceScimUserEmails},
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
		{"ScimLastRequest", testScimLastRequest},
		{"ScimUsersExist", testScimUsersExist},
		{"TriggerResendIdPProfiles", testTriggerResendIdPProfiles},
		{"TriggerResendIdPProfilesOnTeam", testTriggerResendIdPProfilesOnTeam},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
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
			Department: nil,
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
			Department: ptr.String(""),
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
			Department: ptr.String("Development"),
		},
	}

	for _, u := range usersToCreate {
		var err error
		userCopy := u
		userCopy.ID, err = ds.CreateScimUser(t.Context(), &u)
		assert.Nil(t, err)

		verify, err := ds.ScimUserByUserName(t.Context(), u.UserName)
		assert.Nil(t, err)

		assert.Equal(t, userCopy.ID, verify.ID)
		assert.Equal(t, userCopy.UserName, verify.UserName)
		assert.Equal(t, userCopy.ExternalID, verify.ExternalID)
		assert.Equal(t, userCopy.GivenName, verify.GivenName)
		assert.Equal(t, userCopy.FamilyName, verify.FamilyName)
		assert.Equal(t, userCopy.Active, verify.Active)
		assert.Equal(t, userCopy.Department, verify.Department)
		assert.False(t, verify.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

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
		returned, err := ds.ScimUserByID(t.Context(), tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.UserName, returned.UserName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)
		assert.Equal(t, tt.GivenName, returned.GivenName)
		assert.Equal(t, tt.FamilyName, returned.FamilyName)
		assert.Equal(t, tt.Active, returned.Active)
		assert.Equal(t, tt.Department, returned.Department)

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

						// Verify the group is in the user's Groups field
						var foundGroup bool
						for _, userGroup := range returned.Groups {
							if userGroup.ID == group.ID {
								foundGroup = true
								assert.Equal(t, group.DisplayName, userGroup.DisplayName, "Group display name should match")
								break
							}
						}
						assert.True(t, foundGroup, "User's Groups field should contain the group")
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
	_, err := ds.ScimUserByID(t.Context(), 10000000000)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimUserByUserName(t *testing.T, ds *Datastore) {
	users := createTestScimUsers(t, ds)

	// Create test groups and associate them with users
	groups := createTestScimGroups(t, ds, []uint{users[0].ID, users[1].ID})

	for _, tt := range users {
		returned, err := ds.ScimUserByUserName(t.Context(), tt.UserName)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
		assert.Equal(t, tt.UserName, returned.UserName)
		assert.Equal(t, tt.ExternalID, returned.ExternalID)
		assert.Equal(t, tt.GivenName, returned.GivenName)
		assert.Equal(t, tt.FamilyName, returned.FamilyName)
		assert.Equal(t, tt.Active, returned.Active)
		assert.Equal(t, tt.Department, returned.Department)
		assert.False(t, returned.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

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

						// Verify the group is in the user's Groups field
						var foundGroup bool
						for _, userGroup := range returned.Groups {
							if userGroup.ID == group.ID {
								foundGroup = true
								assert.Equal(t, group.DisplayName, userGroup.DisplayName, "Group display name should match")
								break
							}
						}
						assert.True(t, foundGroup, "User's Groups field should contain the group")
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
	_, err := ds.ScimUserByUserName(t.Context(), "nonexistent-user")
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
			Department: nil,
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
			Department: ptr.String("QA"),
		},
	}

	var users []*fleet.ScimUser
	for _, u := range createUsers {
		var err error
		u.ID, err = ds.CreateScimUser(t.Context(), &u)
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
	user.ID, err = ds.CreateScimUser(t.Context(), &user)
	require.Nil(t, err)

	// Create a test group and associate it with the user
	group := fleet.ScimGroup{
		DisplayName: "Test Group for User",
		ExternalID:  ptr.String("ext-group-for-user"),
		ScimUsers:   []uint{user.ID},
	}
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Verify the user was created correctly and has the group
	createdUser, err := ds.ScimUserByID(t.Context(), user.ID)
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
	assert.Equal(t, fleet.ScimUserGroup{ID: group.ID, DisplayName: group.DisplayName}, createdUser.Groups[0])

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
		Groups: []fleet.ScimUserGroup{{ID: 999, DisplayName: "Ignored Group"}}, // Attempt to modify Groups (should be ignored)
	}

	// Replace the user
	err = ds.ReplaceScimUser(t.Context(), &updatedUser)
	require.Nil(t, err)

	// Verify the user was updated correctly
	replacedUser, err := ds.ScimUserByID(t.Context(), user.ID)
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
	assert.Equal(t, group.ID, replacedUser.Groups[0].ID, "Groups field should still contain the original group ID")
	assert.Equal(t, group.DisplayName, replacedUser.Groups[0].DisplayName, "Groups field should still contain the original group display name")

	// Now remove the user from the group using the group methods
	updatedGroup := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: group.DisplayName,
		ExternalID:  group.ExternalID,
		ScimUsers:   []uint{}, // Remove the user
	}
	err = ds.ReplaceScimGroup(t.Context(), &updatedGroup)
	require.Nil(t, err)

	// Verify that the user no longer has the group
	userAfterGroupUpdate, err := ds.ScimUserByID(t.Context(), user.ID)
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

	err = ds.ReplaceScimUser(t.Context(), &nonExistentUser)
	assert.True(t, fleet.IsNotFound(err))
}

func testReplaceScimUserEmails(t *testing.T, ds *Datastore) {
	// Create a test user
	user := fleet.ScimUser{
		UserName:   "email-test-user",
		ExternalID: ptr.String("ext-email-123"),
		GivenName:  ptr.String("Email"),
		FamilyName: ptr.String("Test"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "original.email@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
		},
	}

	var err error
	user.ID, err = ds.CreateScimUser(t.Context(), &user)
	require.Nil(t, err)

	// Smoke test email optimization - replacing with the same emails should not update emails
	// First, get the current user to have a reference point
	currentUser, err := ds.ScimUserByID(t.Context(), user.ID)
	require.NoError(t, err)

	// Create a copy of the user with the same emails
	sameEmailsUser := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "multi-update@example.com",
		ExternalID: ptr.String("ext-replace-456"),
		GivenName:  ptr.String("Multiple"),
		FamilyName: ptr.String("Updates"),
		Active:     ptr.Bool(true),
		Emails:     currentUser.Emails, // Same emails as current user
	}

	// Replace the user
	err = ds.ReplaceScimUser(t.Context(), &sameEmailsUser)
	require.NoError(t, err)

	// Verify the user was updated correctly but emails remain the same
	sameEmailsResult, err := ds.ScimUserByID(t.Context(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, sameEmailsUser.UserName, sameEmailsResult.UserName)
	assert.Equal(t, sameEmailsUser.ExternalID, sameEmailsResult.ExternalID)
	assert.Equal(t, sameEmailsUser.GivenName, sameEmailsResult.GivenName)
	assert.Equal(t, sameEmailsUser.FamilyName, sameEmailsResult.FamilyName)
	assert.Equal(t, sameEmailsUser.Active, sameEmailsResult.Active)

	// Verify emails are the same as before
	assert.Equal(t, len(currentUser.Emails), len(sameEmailsResult.Emails))
	for i := range currentUser.Emails {
		assert.Equal(t, currentUser.Emails[i].Email, sameEmailsResult.Emails[i].Email)
		assert.Equal(t, currentUser.Emails[i].Type, sameEmailsResult.Emails[i].Type)
		assert.Equal(t, currentUser.Emails[i].Primary, sameEmailsResult.Emails[i].Primary)
	}

	// Test validation for multiple primary emails
	multiPrimaryUser := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "multi-primary@example.com",
		ExternalID: ptr.String("ext-multi-primary"),
		GivenName:  ptr.String("Multi"),
		FamilyName: ptr.String("Primary"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "primary1@example.com",
				Primary: ptr.Bool(true), // First primary
				Type:    ptr.String("work"),
			},
			{
				Email:   "primary2@example.com",
				Primary: ptr.Bool(true), // Second primary - should cause validation error
				Type:    ptr.String("home"),
			},
		},
	}

	// This should fail with a validation error
	err = ds.ReplaceScimUser(t.Context(), &multiPrimaryUser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one email can be marked as primary")

	// Test email comparison behavior with different combinations of nil/non-nil fields
	// First, create a user with an email that has all fields set
	userWithAllFields := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "all-fields@example.com",
		ExternalID: ptr.String("ext-all-fields"),
		GivenName:  ptr.String("All"),
		FamilyName: ptr.String("Fields"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "all-fields@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
		},
	}

	err = ds.ReplaceScimUser(t.Context(), &userWithAllFields)
	require.NoError(t, err)

	// Now create a user with the same email but with nil Primary field
	userWithNilPrimary := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "all-fields@example.com",
		ExternalID: ptr.String("ext-all-fields"),
		GivenName:  ptr.String("All"),
		FamilyName: ptr.String("Fields"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "all-fields@example.com",
				Primary: nil, // Changed from true to nil
				Type:    ptr.String("work"),
			},
		},
	}

	// This should update the emails since the Primary field changed
	err = ds.ReplaceScimUser(t.Context(), &userWithNilPrimary)
	require.NoError(t, err)

	// Verify the email was updated
	var nilPrimaryUser *fleet.ScimUser
	nilPrimaryUser, err = ds.ScimUserByID(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, nilPrimaryUser.Emails, 1)
	assert.Equal(t, "all-fields@example.com", nilPrimaryUser.Emails[0].Email)
	assert.Nil(t, nilPrimaryUser.Emails[0].Primary, "Primary field should be nil")

	// Now create a user with the same email but with nil Type field
	userWithNilType := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "all-fields@example.com",
		ExternalID: ptr.String("ext-all-fields"),
		GivenName:  ptr.String("All"),
		FamilyName: ptr.String("Fields"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "all-fields@example.com",
				Primary: nil,
				Type:    nil, // Changed from "work" to nil
			},
		},
	}

	// This should update the emails since the Type field changed
	err = ds.ReplaceScimUser(t.Context(), &userWithNilType)
	require.NoError(t, err)

	// Verify the email was updated
	var nilTypeUser *fleet.ScimUser
	nilTypeUser, err = ds.ScimUserByID(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, nilTypeUser.Emails, 1)
	assert.Equal(t, "all-fields@example.com", nilTypeUser.Emails[0].Email)
	assert.Nil(t, nilTypeUser.Emails[0].Type, "Type field should be nil")
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
	user.ID, err = ds.CreateScimUser(t.Context(), &user)
	require.Nil(t, err)

	// Verify the user was created correctly
	createdUser, err := ds.ScimUserByID(t.Context(), user.ID)
	require.Nil(t, err)
	assert.Equal(t, user.UserName, createdUser.UserName)

	// Delete the user
	err = ds.DeleteScimUser(t.Context(), user.ID)
	require.NoError(t, err)

	// Verify the user was deleted
	_, err = ds.ScimUserByID(t.Context(), user.ID)
	assert.True(t, fleet.IsNotFound(err))

	// Test deleting a non-existent user
	err = ds.DeleteScimUser(t.Context(), 99999) // Non-existent ID
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
		users[i].ID, err = ds.CreateScimUser(t.Context(), &users[i])
		require.Nil(t, err)
	}

	// Create a group and associate it with the first user
	group := fleet.ScimGroup{
		DisplayName: "Test Group for ListUsers",
		ExternalID:  ptr.String("ext-group-for-list"),
		ScimUsers:   []uint{users[0].ID},
	}
	var err error
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Test 1: List all users without filters
	allUsers, totalResults, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
				assert.False(t, u.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

				// Verify Groups field for the first user
				if testUser.ID == users[0].ID {
					require.Len(t, u.Groups, 1, "First user should have exactly one group")
					assert.Equal(t, group.ID, u.Groups[0].ID, "First user should be in the test group")
					assert.Equal(t, group.DisplayName, u.Groups[0].DisplayName, "Group display name should match")
				} else {
					assert.Empty(t, u.Groups, "Other users should not have groups")
				}

				break
			}
		}
	}
	assert.Equal(t, 3, foundUsers)

	// Test 2: Pagination - first page with 2 items
	page1Users, totalPage1, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    2,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(page1Users))
	assert.Equal(t, uint(3), totalPage1) // Total should still be 3

	// Test 3: Pagination - second page with 2 items
	page2Users, totalPage2, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
	listUsers, totalListUsers, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
	assert.False(t, listUsers[0].UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Test 5: Filter by email type and value
	homeEmailUsers, totalHomeEmailUsers, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
	assert.False(t, homeEmailUsers[0].UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Test 6: Filter by email type and value - work emails
	workEmailUsers, totalWorkEmailUsers, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
	assert.False(t, workEmailUsers[0].UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Test 7: No results for non-matching filters
	noUsers, totalNoUsers1, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		UserNameFilter: ptr.String("nonexistent"),
	})
	require.Nil(t, err)
	assert.Empty(t, noUsers)
	assert.Equal(t, uint(0), totalNoUsers1)

	noUsers, totalNoUsers2, err := ds.ListScimUsers(t.Context(), fleet.ScimUsersListOptions{
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
		groupCopy.ID, err = ds.CreateScimGroup(t.Context(), &g)
		require.NoError(t, err)

		verify, err := ds.ScimGroupByID(t.Context(), g.ID, false)
		require.NoError(t, err)

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
	longString := strings.Repeat("a", fleet.SCIMMaxFieldLength+1) // String longer than allowed

	// Test ExternalID validation
	groupWithLongExternalID := fleet.ScimGroup{
		DisplayName: "Valid Name",
		ExternalID:  ptr.String(longString),
		ScimUsers:   []uint{},
	}
	_, err := ds.CreateScimGroup(t.Context(), &groupWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test DisplayName validation
	groupWithLongDisplayName := fleet.ScimGroup{
		DisplayName: longString,
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	_, err = ds.CreateScimGroup(t.Context(), &groupWithLongDisplayName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "display_name exceeds maximum length")

	// Test with valid values
	validGroup := fleet.ScimGroup{
		DisplayName: "Valid Name",
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	_, err = ds.CreateScimGroup(t.Context(), &validGroup)
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
		returned, err := ds.ScimGroupByID(t.Context(), tt.ID, false)
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
	_, err := ds.ScimGroupByID(t.Context(), 10000000000, false)
	assert.True(t, fleet.IsNotFound(err))

	// Test with excludeUsers=true
	for _, tt := range groups {
		returnedWithoutUsers, err := ds.ScimGroupByID(t.Context(), tt.ID, true)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returnedWithoutUsers.ID)
		assert.Equal(t, tt.DisplayName, returnedWithoutUsers.DisplayName)
		assert.Equal(t, tt.ExternalID, returnedWithoutUsers.ExternalID)
		// Verify that users were not fetched
		assert.Empty(t, returnedWithoutUsers.ScimUsers, "ScimUsers should be empty when excludeUsers=true")
	}
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
		returned, err := ds.ScimGroupByDisplayName(t.Context(), tt.DisplayName)
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
	_, err := ds.ScimGroupByDisplayName(t.Context(), "Nonexistent Group")
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
		g.ID, err = ds.CreateScimGroup(t.Context(), &g)
		require.NoError(t, err)
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
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Verify the group was created correctly
	createdGroup, err := ds.ScimGroupByID(t.Context(), group.ID, false)
	require.NoError(t, err)
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
	err = ds.ReplaceScimGroup(t.Context(), &updatedGroup)
	require.Nil(t, err)

	// Verify the group was updated correctly
	replacedGroup, err := ds.ScimGroupByID(t.Context(), group.ID, false)
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

	err = ds.ReplaceScimGroup(t.Context(), &nonExistentGroup)
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
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Test validation for ExternalID
	longString := strings.Repeat("a", fleet.SCIMMaxFieldLength+1) // String longer than allowed

	// Test ExternalID validation
	groupWithLongExternalID := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: "Valid Name",
		ExternalID:  ptr.String(longString),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(t.Context(), &groupWithLongExternalID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "external_id exceeds maximum length")

	// Test DisplayName validation
	groupWithLongDisplayName := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: longString,
		ExternalID:  ptr.String("valid-external-id"),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(t.Context(), &groupWithLongDisplayName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "display_name exceeds maximum length")

	// Test with valid values
	validGroup := fleet.ScimGroup{
		ID:          group.ID,
		DisplayName: "Updated Valid Name",
		ExternalID:  ptr.String("updated-valid-external-id"),
		ScimUsers:   []uint{},
	}
	err = ds.ReplaceScimGroup(t.Context(), &validGroup)
	require.NoError(t, err)
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
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Verify the group was created correctly
	createdGroup, err := ds.ScimGroupByID(t.Context(), group.ID, false)
	require.Nil(t, err)
	assert.Equal(t, group.DisplayName, createdGroup.DisplayName)

	// Delete the group
	err = ds.DeleteScimGroup(t.Context(), group.ID)
	require.Nil(t, err)

	// Verify the group was deleted
	_, err = ds.ScimGroupByID(t.Context(), group.ID, false)
	assert.True(t, fleet.IsNotFound(err))

	// Test deleting a non-existent group
	err = ds.DeleteScimGroup(t.Context(), 99999) // Non-existent ID
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
		groups[i].ID, err = ds.CreateScimGroup(t.Context(), &groups[i])
		require.NoError(t, err)
	}

	// Test 1: List all groups
	allGroups, totalResults, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
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
	page1Groups, totalPage1, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    2,
		},
	})
	require.Nil(t, err)
	assert.Equal(t, 2, len(page1Groups))
	assert.GreaterOrEqual(t, totalPage1, uint(3)) // Total should be at least 3

	// Test 3: Pagination - second page with 2 items
	page2Groups, totalPage2, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 3, // StartIndex is 1-based, so for the second page with 2 items per page, we start at index 3
			PerPage:    2,
		},
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

	// Test 4: Filter by display name
	displayName := "List Test Group 2"
	filteredGroups, totalFilteredResults, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		DisplayNameFilter: &displayName,
	})
	require.Nil(t, err)
	assert.Equal(t, 1, len(filteredGroups), "Should find exactly one group with the specified display name")
	assert.Equal(t, uint(1), totalFilteredResults)
	assert.Equal(t, displayName, filteredGroups[0].DisplayName)

	// Test 5: Filter by non-existent display name
	nonExistentName := "Non-Existent Group"
	emptyResults, totalEmptyResults, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		DisplayNameFilter: &nonExistentName,
	})
	require.Nil(t, err)
	assert.Empty(t, emptyResults, "Should find no groups with a non-existent display name")
	assert.Equal(t, uint(0), totalEmptyResults)

	// Test 6: List groups with ExcludeUsers=true
	groupsWithoutUsers, totalWithoutUsers, err := ds.ListScimGroups(t.Context(), fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: 1,
			PerPage:    10,
		},
		ExcludeUsers: true,
	})
	require.Nil(t, err)
	assert.GreaterOrEqual(t, len(groupsWithoutUsers), 3, "Should find at least 3 groups")
	assert.Equal(t, totalResults, totalWithoutUsers, "Total count should be the same with or without users")

	// Verify that users were not fetched
	for _, group := range groupsWithoutUsers {
		assert.Empty(t, group.ScimUsers, "ScimUsers should be empty when ExcludeUsers=true")
	}
}

func testScimUserCreateValidation(t *testing.T, ds *Datastore) {
	// Test validation for ExternalID
	longString := strings.Repeat("a", fleet.SCIMMaxFieldLength+1) // String longer than SCIMMaxFieldLength

	// Test ExternalID validation
	userWithLongExternalID := fleet.ScimUser{
		UserName:   "valid-username",
		ExternalID: ptr.String(longString),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
	}
	_, err := ds.CreateScimUser(t.Context(), &userWithLongExternalID)
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
	_, err = ds.CreateScimUser(t.Context(), &userWithLongUserName)
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
	_, err = ds.CreateScimUser(t.Context(), &userWithLongGivenName)
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
	_, err = ds.CreateScimUser(t.Context(), &userWithLongFamilyName)
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
	_, err = ds.CreateScimUser(t.Context(), &validUser)
	assert.NoError(t, err)
}

func testScimUserByHostID(t *testing.T, ds *Datastore) {
	// Create a test SCIM user with emails
	user1 := fleet.ScimUser{
		UserName:   "host-test-user1",
		ExternalID: ptr.String("ext-host-123"),
		GivenName:  ptr.String("Host"),
		FamilyName: ptr.String("User"),
		Active:     ptr.Bool(true),
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "host.user@example.com",
				Primary: ptr.Bool(true),
				Type:    ptr.String("work"),
			},
		},
		Department: ptr.String("Engineering"),
	}

	var err error
	user1.ID, err = ds.CreateScimUser(t.Context(), &user1)
	require.Nil(t, err)

	// Create a group and associate it with the user
	group := fleet.ScimGroup{
		DisplayName: "Host Test Group",
		ExternalID:  ptr.String("ext-host-group"),
		ScimUsers:   []uint{user1.ID},
	}
	group.ID, err = ds.CreateScimGroup(t.Context(), &group)
	require.NoError(t, err)

	// Create a second test SCIM user without emails, without groups nor department
	user2 := fleet.ScimUser{
		UserName:   "host-test-user2",
		ExternalID: ptr.String("ext-host-456"),
		GivenName:  ptr.String("No"),
		FamilyName: ptr.String("Emails"),
		Active:     ptr.Bool(true),
		Emails:     []fleet.ScimUserEmail{},
		Department: nil,
	}
	user2.ID, err = ds.CreateScimUser(t.Context(), &user2)
	require.Nil(t, err)

	// Create test hosts
	hostID1 := uint(1000001) // Use a dummy host ID for testing
	hostID2 := uint(1000002) // Use a different dummy host ID for testing

	// Associate the hosts with the SCIM users
	_, err = ds.writer(t.Context()).ExecContext(
		t.Context(),
		"INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?), (?, ?)",
		hostID1, user1.ID,
		hostID2, user2.ID,
	)
	require.Nil(t, err)

	// Test 1: Get SCIM user with emails and groups by host ID
	result1, err := ds.ScimUserByHostID(t.Context(), hostID1)
	assert.Nil(t, err)
	assert.NotNil(t, result1)
	assert.Equal(t, user1.ID, result1.ID)
	assert.Equal(t, user1.UserName, result1.UserName)
	assert.Equal(t, user1.ExternalID, result1.ExternalID)
	assert.Equal(t, user1.GivenName, result1.GivenName)
	assert.Equal(t, user1.FamilyName, result1.FamilyName)
	assert.Equal(t, user1.Active, result1.Active)
	assert.Equal(t, user1.Department, result1.Department)
	assert.False(t, result1.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Verify emails
	require.Equal(t, 1, len(result1.Emails))
	assert.Equal(t, "host.user@example.com", result1.Emails[0].Email)
	assert.Equal(t, true, *result1.Emails[0].Primary)
	assert.Equal(t, "work", *result1.Emails[0].Type)

	// Verify groups
	require.Equal(t, 1, len(result1.Groups))
	assert.Equal(t, group.ID, result1.Groups[0].ID)
	assert.Equal(t, group.DisplayName, result1.Groups[0].DisplayName)

	// Test 2: Get SCIM user without emails and without groups by host ID
	result2, err := ds.ScimUserByHostID(t.Context(), hostID2)
	assert.Nil(t, err)
	assert.NotNil(t, result2)
	assert.Equal(t, user2.ID, result2.ID)
	assert.Equal(t, user2.UserName, result2.UserName)
	assert.Equal(t, user2.ExternalID, result2.ExternalID)
	assert.Equal(t, user2.GivenName, result2.GivenName)
	assert.Equal(t, user2.FamilyName, result2.FamilyName)
	assert.Equal(t, user2.Active, result2.Active)
	assert.Equal(t, user2.Department, result2.Department)
	assert.False(t, result2.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Verify no emails
	assert.Empty(t, result2.Emails)

	// Verify no groups
	assert.Empty(t, result2.Groups)

	// Test 3: Get SCIM user for a host that doesn't have an associated user
	nonExistentHostID := uint(9999999)
	_, err = ds.ScimUserByHostID(t.Context(), nonExistentHostID)
	assert.NotNil(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testScimUserByUserNameOrEmail(t *testing.T, ds *Datastore) {
	// Create test users with different attributes and emails
	users := []fleet.ScimUser{
		{
			UserName:   "email-test-user1",
			ExternalID: ptr.String("ext-email-123"),
			GivenName:  ptr.String("Email"),
			FamilyName: ptr.String("User1"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "email.user1@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "email-test-user2",
			ExternalID: ptr.String("ext-email-456"),
			GivenName:  ptr.String("Email"),
			FamilyName: ptr.String("User2"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "email.user2@example.com",
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "duplicate-email-user1",
			ExternalID: ptr.String("ext-dup-123"),
			GivenName:  ptr.String("Duplicate"),
			FamilyName: ptr.String("Email1"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "duplicate@example.com", // Duplicate email
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
		{
			UserName:   "duplicate-email-user2",
			ExternalID: ptr.String("ext-dup-456"),
			GivenName:  ptr.String("Duplicate"),
			FamilyName: ptr.String("Email2"),
			Active:     ptr.Bool(true),
			Emails: []fleet.ScimUserEmail{
				{
					Email:   "duplicate@example.com", // Duplicate email
					Primary: ptr.Bool(true),
					Type:    ptr.String("work"),
				},
			},
		},
	}

	// Create the users
	for i := range users {
		var err error
		users[i].ID, err = ds.CreateScimUser(t.Context(), &users[i])
		require.Nil(t, err)
	}

	// Test 1: Find user by userName
	email := "email.user1@example.com"
	user, err := ds.ScimUserByUserNameOrEmail(t.Context(), "email-test-user1", email)
	assert.Nil(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "email-test-user1", user.UserName)
	assert.Equal(t, users[0].ID, user.ID)
	assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Test 2: Find user by email when userName is empty
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "", email)
	assert.Nil(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "email-test-user1", user.UserName)
	assert.Equal(t, users[0].ID, user.ID)

	// Test 3: Find user by email when userName doesn't exist
	email = "email.user2@example.com"
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "nonexistent-user", email)
	assert.Nil(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "email-test-user2", user.UserName)
	assert.Equal(t, users[1].ID, user.ID)
	assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should not be zero")

	// Test 4: Handle case where multiple users have the same email
	email = "duplicate@example.com"
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "nonexistent-user", email)
	assert.Nil(t, err)
	assert.Nil(t, user, "Should return nil when multiple users have the same email")

	// Test 5: Handle case where neither userName nor email match any user
	email = "nonexistent@example.com"
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "nonexistent-user", email)
	assert.NotNil(t, err)
	assert.True(t, fleet.IsNotFound(err))
	assert.Nil(t, user)

	// Test 6: Handle case where email is empty
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "nonexistent-user", "")
	assert.NotNil(t, err)
	assert.True(t, fleet.IsNotFound(err))
	assert.Nil(t, user)

	// Test 7: Find user when email is used as userName
	// This tests the case where the userName field contains an email address
	user, err = ds.ScimUserByUserNameOrEmail(t.Context(), "nonexistent-username", "email-test-user1")
	require.NoError(t, err)
	assert.Equal(t, "email-test-user1", user.UserName)
	assert.Equal(t, users[0].ID, user.ID)
}

func testScimUserReplaceValidation(t *testing.T, ds *Datastore) {
	// Create a valid user first
	user := fleet.ScimUser{
		UserName:   "replace-validation-user",
		ExternalID: ptr.String("ext-replace-validation"),
		GivenName:  ptr.String("Original"),
		FamilyName: ptr.String("User"),
		Active:     ptr.Bool(true),
		Department: ptr.String("Customer support"),
	}

	var err error
	user.ID, err = ds.CreateScimUser(t.Context(), &user)
	require.NoError(t, err)

	// Test validation for ExternalID
	longString := strings.Repeat("a", fleet.SCIMMaxFieldLength+1) // String longer than SCIMMaxFieldLength

	// Test ExternalID validation
	userWithLongExternalID := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "valid-username",
		ExternalID: ptr.String(longString),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
		Department: ptr.String("Customer support"),
	}
	err = ds.ReplaceScimUser(t.Context(), &userWithLongExternalID)
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
		Department: ptr.String("Customer support"),
	}
	err = ds.ReplaceScimUser(t.Context(), &userWithLongUserName)
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
		Department: ptr.String("Customer support"),
	}
	err = ds.ReplaceScimUser(t.Context(), &userWithLongGivenName)
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
		Department: ptr.String("Customer support"),
	}
	err = ds.ReplaceScimUser(t.Context(), &userWithLongFamilyName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "family_name exceeds maximum length")

	// Test Department validation
	userWithLongDepartment := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "valid-username",
		ExternalID: ptr.String("valid-external-id"),
		GivenName:  ptr.String("Valid"),
		FamilyName: ptr.String("Valid"),
		Active:     ptr.Bool(true),
		Department: ptr.String(longString),
	}
	err = ds.ReplaceScimUser(t.Context(), &userWithLongDepartment)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "department exceeds maximum length")

	// Test with valid values
	validUser := fleet.ScimUser{
		ID:         user.ID,
		UserName:   "updated-username",
		ExternalID: ptr.String("updated-external-id"),
		GivenName:  ptr.String("Updated"),
		FamilyName: ptr.String("Name"),
		Active:     ptr.Bool(true),
		Department: ptr.String("Customer support updated"),
	}
	err = ds.ReplaceScimUser(t.Context(), &validUser)
	assert.NoError(t, err)

	updated, err := ds.ScimUserByID(t.Context(), user.ID)
	assert.Nil(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, updated.ID, user.ID)
	assert.Equal(t, updated.UserName, validUser.UserName)
	assert.Equal(t, updated.ExternalID, validUser.ExternalID)
	assert.Equal(t, updated.GivenName, validUser.GivenName)
	assert.Equal(t, updated.FamilyName, validUser.FamilyName)
	assert.Equal(t, updated.Active, validUser.Active)
	assert.Equal(t, updated.Department, validUser.Department)
	assert.Greater(t, updated.UpdatedAt, user.UpdatedAt)
}

func testScimLastRequest(t *testing.T, ds *Datastore) {
	// Initially, there should be no last request
	initialRequest, err := ds.ScimLastRequest(t.Context())
	assert.NoError(t, err)
	assert.Nil(t, initialRequest)

	// Validation tests for UpdateScimLastRequest
	// Nil request should return nil
	err = ds.UpdateScimLastRequest(t.Context(), nil)
	assert.NoError(t, err)

	// Status exceeding max length should return error
	longStatus := strings.Repeat("a", SCIMMaxStatusLength+1)
	invalidStatusRequest := &fleet.ScimLastRequest{
		Status:  longStatus,
		Details: "Valid details",
	}
	err = ds.UpdateScimLastRequest(t.Context(), invalidStatusRequest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status exceeds maximum length")

	// Details exceeding max length should return error
	longDetails := strings.Repeat("b", fleet.SCIMMaxFieldLength+1) // 256 characters
	invalidDetailsRequest := &fleet.ScimLastRequest{
		Status:  "valid",
		Details: longDetails,
	}
	err = ds.UpdateScimLastRequest(t.Context(), invalidDetailsRequest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "details exceeds maximum length")

	// Create a new last request with valid values
	newRequest := &fleet.ScimLastRequest{
		Status:  "success",
		Details: "Initial SCIM request",
	}
	err = ds.UpdateScimLastRequest(t.Context(), newRequest)
	assert.NoError(t, err)

	// Retrieve the last request and verify it matches
	retrievedRequest, err := ds.ScimLastRequest(t.Context())
	require.NoError(t, err)
	require.NotNil(t, retrievedRequest)
	assert.Equal(t, "success", retrievedRequest.Status)
	assert.Equal(t, "Initial SCIM request", retrievedRequest.Details)
	assert.False(t, retrievedRequest.RequestedAt.IsZero(), "RequestedAt should not be zero")

	// Do and check the same request again -- timestamp should update
	err = ds.UpdateScimLastRequest(t.Context(), newRequest)
	assert.NoError(t, err)
	retrievedSameRequest, err := ds.ScimLastRequest(t.Context())
	require.NoError(t, err)
	require.NotNil(t, retrievedSameRequest)
	assert.Equal(t, retrievedRequest.Status, retrievedSameRequest.Status)
	assert.Equal(t, retrievedRequest.Details, retrievedSameRequest.Details)
	// Verify that the timestamp is newer
	assert.True(t, retrievedSameRequest.RequestedAt.After(retrievedRequest.RequestedAt),
		"Same request timestamp should be after the original timestamp")

	// Update the last request with new valid values
	updatedRequest := &fleet.ScimLastRequest{
		Status:  "error",
		Details: "Updated SCIM request with error",
	}
	err = ds.UpdateScimLastRequest(t.Context(), updatedRequest)
	assert.NoError(t, err)

	// Retrieve the updated last request and verify it matches
	retrievedUpdatedRequest, err := ds.ScimLastRequest(t.Context())
	require.NoError(t, err)
	require.NotNil(t, retrievedUpdatedRequest)
	assert.Equal(t, "error", retrievedUpdatedRequest.Status)
	assert.Equal(t, "Updated SCIM request with error", retrievedUpdatedRequest.Details)
	assert.False(t, retrievedUpdatedRequest.RequestedAt.IsZero(), "RequestedAt should not be zero")

	// Verify that the updated timestamp is newer
	assert.True(t, retrievedUpdatedRequest.RequestedAt.After(retrievedSameRequest.RequestedAt),
		"Updated request timestamp should be after the original timestamp")
}

func testTriggerResendIdPProfiles(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// create some hosts to deploy profiles to
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "host3key", "host3uuid", time.Now())

	// create profiles that use the IdP variables, and one that doesn't
	profUsername, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", 0), nil)
	require.NoError(t, err)
	profGroup, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", 0), nil)
	require.NoError(t, err)
	profAll, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("c", "c", 0), nil)
	require.NoError(t, err)
	profNone, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("d", "d", 0), nil)
	require.NoError(t, err)

	t.Logf("profUsername=%s, profGroup=%s, profAll=%s, profNone=%s", profUsername.ProfileUUID, profGroup.ProfileUUID, profAll.ProfileUUID, profNone.ProfileUUID)

	// insert the relationship between profile and variables
	varsPerProfile := map[string][]string{
		profUsername.ProfileUUID: {fleet.FleetVarHostEndUserIDPUsername, fleet.FleetVarHostEndUserIDPUsernameLocalPart},
		profGroup.ProfileUUID:    {fleet.FleetVarHostEndUserIDPGroups},
		profAll.ProfileUUID:      {fleet.FleetVarHostEndUserIDPUsername, fleet.FleetVarHostEndUserIDPUsernameLocalPart, fleet.FleetVarHostEndUserIDPGroups},
	}
	for profUUID, vars := range varsPerProfile {
		for _, v := range vars {
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `INSERT INTO mdm_configuration_profile_variables (apple_profile_uuid, fleet_variable_id)
					SELECT ?, id FROM fleet_variables WHERE name = ?`, profUUID, "FLEET_VAR_"+v)
				return err
			})
		}
	}

	// create some scim users and assign one to hosts 1 and 2
	scimUser1, err := ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "a@example.com"})
	require.NoError(t, err)
	scimUser2, err := ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "b@example.com"})
	require.NoError(t, err)
	scimUser3, err := ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "c@example.com"})
	require.NoError(t, err)
	err = ds.associateHostWithScimUser(ctx, host1.ID, scimUser1)
	require.NoError(t, err)
	err = ds.associateHostWithScimUser(ctx, host2.ID, scimUser2)
	require.NoError(t, err)

	// no profiles exist yet for any host, so this setup hasn't triggered anything
	assertHostProfileStatus(t, ds, host1.UUID)
	assertHostProfileStatus(t, ds, host2.UUID)
	assertHostProfileStatus(t, ds, host3.UUID)

	// mark all profiles as installed on all hosts
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// change username of scim user 1
	err = ds.ReplaceScimUser(ctx, &fleet.ScimUser{ID: scimUser1, UserName: "A@example.com"})
	require.NoError(t, err)

	// this triggered a resend of profUsername and profAll on host1
	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// reset the status for host1
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// create a scim group for user1 and user2
	group1, err := ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "g1", ScimUsers: []uint{scimUser1, scimUser2}})
	require.NoError(t, err)

	// this triggered a resend of profGroup and profAll on host1 and host2
	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// reset the statuses
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// create another scim group with no user and update other properties of
	// user1, does not trigger anything
	group2, err := ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "g2"})
	require.NoError(t, err)
	err = ds.ReplaceScimUser(ctx, &fleet.ScimUser{ID: scimUser1, UserName: "A@example.com", GivenName: ptr.String("A")})
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// change group1's name, affects host1 and host2 (via user1 and user2)
	err = ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: group1, DisplayName: "G1", ScimUsers: []uint{scimUser1, scimUser2}})
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// reset the statuses
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// assign user3 as IdP user of host3
	err = ds.associateHostWithScimUser(ctx, host3.ID, scimUser3)
	require.NoError(t, err)

	// affects host3
	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})

	// reset the statuses
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// add user3 and remove user2 from the group
	err = ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: group1, DisplayName: "G1", ScimUsers: []uint{scimUser1, scimUser3}})
	require.NoError(t, err)

	// affects host2 and host3, not host1 because user2 is not its IdP user (it
	// is an extra one)
	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})

	// reset the statuses
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// delete group2, has no user so no effect
	err = ds.DeleteScimGroup(ctx, group2)
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// delete user3, affects only host3 (not the official IdP user for host1)
	err = ds.DeleteScimUser(ctx, scimUser3)
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})

	// reset the statuses
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profUsername, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host3.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// delete user1
	err = ds.DeleteScimUser(ctx, scimUser1)
	require.NoError(t, err)
	// add user2 as new user for host1
	err = ds.associateHostWithScimUser(ctx, host1.ID, scimUser2)
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})

	// reset the statuses, but set username operation to remove
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profUsername, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host1.UUID, profAll, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	// update name of user2, will affect host1 and host2, but NOT the
	// profUsername of host1 because it is not installed (it is removed)
	err = ds.ReplaceScimUser(ctx, &fleet.ScimUser{ID: scimUser2, UserName: "B@example.com", GivenName: ptr.String("B")})
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryPending},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryPending})
	assertHostProfileStatus(t, ds, host3.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profUsername.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profAll.ProfileUUID, fleet.MDMDeliveryVerifying})
}

// for https://github.com/fleetdm/fleet/issues/28820
func testTriggerResendIdPProfilesOnTeam(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// create a couple hosts to deploy profiles to
	host1 := test.NewHost(t, ds, "host1", "1", "h1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "h2key", "host2uuid", time.Now())

	// create a team and make host2 part of that team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{host2.ID})
	require.NoError(t, err)

	// create some profiles with/without vars on the team
	profGroup, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("a", "a", team.ID), []string{fleet.FleetVarHostEndUserIDPGroups})
	require.NoError(t, err)
	profNone, err := ds.NewMDMAppleConfigProfile(ctx, *generateCP("b", "b", team.ID), nil)
	require.NoError(t, err)

	t.Logf("profGroup=%s, profNone=%s", profGroup.ProfileUUID, profNone.ProfileUUID)

	// create some scim data
	scimUser1, err := ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "a@example.com"})
	require.NoError(t, err)
	scimUser2, err := ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "b@example.com"})
	require.NoError(t, err)
	err = ds.associateHostWithScimUser(ctx, host2.ID, scimUser1)
	require.NoError(t, err)
	group1, err := ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "g1", ScimUsers: []uint{scimUser1, scimUser2}})
	require.NoError(t, err)

	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profGroup, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
	forceSetAppleHostProfileStatus(t, ds, host2.UUID, profNone, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)

	assertHostProfileStatus(t, ds, host1.UUID)
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryVerifying})

	// remove user 1 from group, affects the host2 profile
	err = ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: group1, DisplayName: "g1", ScimUsers: []uint{scimUser2}})
	require.NoError(t, err)

	assertHostProfileStatus(t, ds, host1.UUID)
	// the bug was failing this check, profNone was set to Pending although only
	// profGroup should have changed.
	assertHostProfileStatus(t, ds, host2.UUID,
		hostProfileStatus{profNone.ProfileUUID, fleet.MDMDeliveryVerifying},
		hostProfileStatus{profGroup.ProfileUUID, fleet.MDMDeliveryPending})
}

type hostProfileStatus struct {
	ProfileUUID string
	Status      fleet.MDMDeliveryStatus
}

func assertHostProfileStatus(t *testing.T, ds *Datastore, hostUUID string, wantProfiles ...hostProfileStatus) {
	withOpProfiles := make([]hostProfileOpStatus, 0, len(wantProfiles))
	for _, p := range wantProfiles {
		withOpProfiles = append(withOpProfiles, hostProfileOpStatus{
			ProfileUUID: p.ProfileUUID,
			Status:      p.Status,
			OpType:      fleet.MDMOperationTypeInstall,
		})
	}
	assertHostProfileOpStatus(t, ds, hostUUID, withOpProfiles...)
}

type hostProfileOpStatus struct {
	ProfileUUID string
	Status      fleet.MDMDeliveryStatus
	OpType      fleet.MDMOperationType
}

func assertHostProfileOpStatus(t *testing.T, ds *Datastore, hostUUID string, wantProfiles ...hostProfileOpStatus) {
	ctx := t.Context()
	winProfs, err := ds.GetHostMDMWindowsProfiles(ctx, hostUUID)
	require.NoError(t, err)
	appleProfs, err := ds.GetHostMDMAppleProfiles(ctx, hostUUID)
	require.NoError(t, err)

	type commonHostProf struct {
		Status      fleet.MDMDeliveryStatus
		Type        fleet.MDMOperationType
		ProfileUUID string
	}
	profs := make([]commonHostProf, 0, len(appleProfs)+len(winProfs))
	for _, wp := range winProfs {
		var status fleet.MDMDeliveryStatus
		if wp.Status == nil {
			status = fleet.MDMDeliveryPending
		} else {
			status = *wp.Status
		}
		profs = append(profs, commonHostProf{
			ProfileUUID: wp.ProfileUUID,
			Status:      status,
			Type:        wp.OperationType,
		})
	}
	for _, ap := range appleProfs {
		var status fleet.MDMDeliveryStatus
		if ap.Status == nil {
			status = fleet.MDMDeliveryPending
		} else {
			status = *ap.Status
		}
		profs = append(profs, commonHostProf{
			ProfileUUID: ap.ProfileUUID,
			Status:      status,
			Type:        ap.OperationType,
		})
	}

	require.Len(t, profs, len(wantProfiles))

	// index the status of the actual profiles for quick lookup
	profStatus := make(map[string]fleet.MDMDeliveryStatus, len(profs))
	profOpType := make(map[string]fleet.MDMOperationType, len(profs))
	for _, prof := range profs {
		profStatus[prof.ProfileUUID] = prof.Status
		profOpType[prof.ProfileUUID] = prof.Type
	}
	for _, want := range wantProfiles {
		status, ok := profStatus[want.ProfileUUID]
		require.True(t, ok, "profile %s not found in host %s", want.ProfileUUID, hostUUID)
		assert.Equal(t, want.Status, status, "profile %s", want.ProfileUUID)
		assert.Equal(t, want.OpType, profOpType[want.ProfileUUID], "profile %s", want.ProfileUUID)
	}
}

// helper function to force-set a host profile status
func forceSetAppleHostProfileStatus(t *testing.T, ds *Datastore, hostUUID string, profile *fleet.MDMAppleConfigProfile, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_apple_profiles
				(profile_identifier, host_uuid, status, operation_type, command_uuid, profile_name, checksum, profile_uuid)
			VALUES
				(?, ?, ?, ?, ?, ?, UNHEX(MD5(?)), ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			profile.Identifier, hostUUID, actualStatus, operation, uuid.NewString(), profile.Name, profile.Mobileconfig, profile.ProfileUUID)
		return err
	})
}

func testScimUsersExist(t *testing.T, ds *Datastore) {
	// Create test users
	users := createTestScimUsers(t, ds)
	userIDs := make([]uint, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	// Test 1: Empty slice should return true
	exist, err := ds.ScimUsersExist(t.Context(), []uint{})
	require.NoError(t, err)
	assert.True(t, exist, "Empty slice should return true")

	// Test 2: All existing users should return true
	exist, err = ds.ScimUsersExist(t.Context(), userIDs)
	require.NoError(t, err)
	assert.True(t, exist, "All existing users should return true")

	// Test 3: Mix of existing and non-existing users should return false
	nonExistentIDs := userIDs
	nonExistentIDs = append(nonExistentIDs, 99999)
	exist, err = ds.ScimUsersExist(t.Context(), nonExistentIDs)
	require.NoError(t, err)
	assert.False(t, exist, "Mix of existing and non-existing users should return false")

	// Test 4: Only non-existing users should return false
	exist, err = ds.ScimUsersExist(t.Context(), []uint{99999, 100000})
	require.NoError(t, err)
	assert.False(t, exist, "Only non-existing users should return false")

	// Test 5: Test with a large number of IDs to verify batching works
	// First, create a large number of test users
	largeUserIDs := make([]uint, 0, 25000)
	largeUserIDs = append(largeUserIDs, userIDs...) // Add existing users

	// Add some non-existent IDs to test batching with mixed results
	for i := 0; i < 24990; i++ {
		largeUserIDs = append(largeUserIDs, uint(1000000)+uint(i)) // nolint:gosec // dismiss G115 integer overflow
	}

	exist, err = ds.ScimUsersExist(t.Context(), largeUserIDs)
	require.NoError(t, err)
	assert.False(t, exist, "Large batch with non-existing users should return false")

	// Test 6: Test with a large number of existing IDs
	// This is a bit tricky to test thoroughly without creating thousands of users,
	// so we'll just verify the function handles a large slice without errors
	largeExistingIDs := make([]uint, 0, 25000)
	for i := 0; i < 25000; i++ {
		largeExistingIDs = append(largeExistingIDs, userIDs[i%len(userIDs)])
	}

	exist, err = ds.ScimUsersExist(t.Context(), largeExistingIDs)
	require.NoError(t, err)
	assert.True(t, exist, "Large batch with only existing users should return true")
}

func forceSetWindowsHostProfileStatus(t *testing.T, ds *Datastore, hostUUID string, profile *fleet.MDMWindowsConfigProfile, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_windows_profiles
				(host_uuid, status, operation_type, command_uuid, profile_name, checksum, profile_uuid)
			VALUES
				(?, ?, ?, ?, ?, UNHEX(MD5(?)), ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			hostUUID, actualStatus, operation, uuid.NewString(), profile.Name, profile.SyncML, profile.ProfileUUID)
		return err
	})
}

func forceSetAppleHostDeclarationStatus(t *testing.T, ds *Datastore, hostUUID string, profile *fleet.MDMAppleDeclaration, operation fleet.MDMOperationType, status fleet.MDMDeliveryStatus) {
	ctx := t.Context()

	// empty status string means set to NULL
	var actualStatus *fleet.MDMDeliveryStatus
	if status != "" {
		actualStatus = &status
	}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO host_mdm_apple_declarations
				(declaration_identifier, host_uuid, status, operation_type, token, declaration_name, declaration_uuid)
			VALUES
				(?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				status = VALUES(status),
				operation_type = VALUES(operation_type)
			`,
			profile.Identifier, hostUUID, actualStatus, operation, uuid.NewString(), profile.Name, profile.DeclarationUUID)
		return err
	})
}

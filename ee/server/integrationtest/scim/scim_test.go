package scim

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/elimity-com/scim/errors"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCIM(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.SCIM")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"Auth", testAuth},
		{"BaseEndpoints", testBaseEndpoints},
		{"Users", testUsersBasicCRUD},
		{"Groups", testGroupsBasicCRUD},
		{"CreateUser", testCreateUser},
		{"CreateGroup", testCreateGroup},
		{"UpdateUser", testUpdateUser},
		{"UpdateGroup", testUpdateGroup},
		{"PatchUserEmails", testPatchUserEmails},
		{"PatchUserAttributes", testPatchUserAttributes},
		{"PatchGroupAttributes", testPatchGroupAttributes},
		{"PatchGroupMembers", testPatchGroupMembers},
		{"UsersPagination", testUsersPagination},
		{"GroupsPagination", testGroupsPagination},
		{"UsersAndGroups", testUsersAndGroups},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS, []string{
				"host_scim_user", "scim_users", "scim_user_emails", "scim_groups",
				"scim_user_group", "scim_last_request",
			}...)
			c.fn(t, s)
		})
	}
}

func testAuth(t *testing.T, s *Suite) {
	t.Cleanup(func() {
		s.Token = s.GetTestAdminToken(t)
	})

	// Unauthenticated
	s.Token = "bozo"
	var resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusUnauthorized, &resp)
	assert.Contains(t, resp["detail"], "Authentication")
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	scimDetails := contract.ScimDetailsResponse{}
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusUnauthorized, &scimDetails)
	// Make sure unauthenticated response wasn't saved as the last SCIM request
	s.Token = s.GetTestToken(t, service.TestMaintainerUserEmail, test.GoodPassword)
	scimDetails = contract.ScimDetailsResponse{}
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusOK, &scimDetails)
	assert.Nil(t, scimDetails.LastRequest, "last_request should NOT be present for unauthenticated requests")

	// Unauthorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestObserverUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusForbidden, &resp)
	assert.Contains(t, resp["detail"], "forbidden")
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusForbidden, &scimDetails)
	// Make sure unauthorized response WAS saved as the last SCIM request
	s.Token = s.GetTestToken(t, service.TestMaintainerUserEmail, test.GoodPassword)
	scimDetails = contract.ScimDetailsResponse{}
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusOK, &scimDetails)
	require.NotNil(t, scimDetails.LastRequest)
	assert.Equal(t, "error", scimDetails.LastRequest.Status)
	assert.NotZero(t, scimDetails.LastRequest.RequestedAt)
	assert.Equal(t, authz.ForbiddenErrorMessage, scimDetails.LastRequest.Details)

	// Authorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestMaintainerUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &resp)
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
}

func testBaseEndpoints(t *testing.T, s *Suite) {
	// Make sure SCIM details.last_request DOES NOT exist
	scimDetails := contract.ScimDetailsResponse{}
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusOK, &scimDetails)
	assert.Nil(t, scimDetails.LastRequest)

	t.Run("Test /Schemas endpoint", func(t *testing.T) {
		var schemasResp map[string]interface{}
		s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &schemasResp)

		// Verify last request was recorded
		scimDetails := contract.ScimDetailsResponse{}
		s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusOK, &scimDetails)
		require.NotNil(t, scimDetails.LastRequest)
		assert.Equal(t, "success", scimDetails.LastRequest.Status)
		assert.NotZero(t, scimDetails.LastRequest.RequestedAt)
		assert.Empty(t, scimDetails.LastRequest.Details)

		// Verify schemas response
		assert.EqualValues(t, schemasResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
		resources, ok := schemasResp["Resources"].([]interface{})
		assert.True(t, ok, "Resources should be an array")
		assert.GreaterOrEqual(t, len(resources), 2, "Should have at least 2 schemas (User and Group)")

		// Check for User and Group schemas
		foundUser := false
		foundGroup := false
		for _, resource := range resources {
			schema, ok := resource.(map[string]interface{})
			assert.True(t, ok, "Schema should be an object")

			id, ok := schema["id"].(string)
			assert.True(t, ok, "Schema ID should be a string")

			if id == "urn:ietf:params:scim:schemas:core:2.0:User" {
				foundUser = true
				attributes := schema["attributes"].([]interface{})
				assert.NotNil(t, attributes, "User schema should have attributes")
			} else if id == "urn:ietf:params:scim:schemas:core:2.0:Group" {
				foundGroup = true
				attributes := schema["attributes"].([]interface{})
				assert.NotNil(t, attributes, "Group schema should have attributes")
			}
		}
		assert.True(t, foundUser, "User schema should be present")
		assert.True(t, foundGroup, "Group schema should be present")
	})

	t.Run("Test /ServiceProviderConfig endpoint", func(t *testing.T) {
		var configResp map[string]interface{}
		s.DoJSON(t, "GET", scimPath("/ServiceProviderConfig"), nil, http.StatusOK, &configResp)

		// Verify service provider config response
		assert.EqualValues(t, configResp["schemas"], []interface{}{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"})
		assert.NotNil(t, configResp["documentationUri"])
	})

	t.Run("Test /ResourceTypes endpoint", func(t *testing.T) {
		var resourceTypesResp map[string]interface{}
		s.DoJSON(t, "GET", scimPath("/ResourceTypes"), nil, http.StatusOK, &resourceTypesResp)

		// Verify resource types response
		assert.EqualValues(t, resourceTypesResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
		resourceTypes, ok := resourceTypesResp["Resources"].([]interface{})
		assert.True(t, ok, "Resources should be an array")
		assert.GreaterOrEqual(t, len(resourceTypes), 2, "Should have at least 2 resource types (User and Group)")

		// Check for User and Group resource types
		foundUserResource := false
		foundGroupResource := false
		for _, resource := range resourceTypes {
			resourceType, ok := resource.(map[string]interface{})
			assert.True(t, ok, "Resource type should be an object")

			name, ok := resourceType["name"].(string)
			assert.True(t, ok, "Resource type name should be a string")

			if name == "User" {
				foundUserResource = true
				assert.Equal(t, "/Users", resourceType["endpoint"])
				assert.Equal(t, "urn:ietf:params:scim:schemas:core:2.0:User", resourceType["schema"])
			} else if name == "Group" {
				foundGroupResource = true
				assert.Equal(t, "/Groups", resourceType["endpoint"])
				assert.Equal(t, "urn:ietf:params:scim:schemas:core:2.0:Group", resourceType["schema"])
			}
		}
		assert.True(t, foundUserResource, "User resource type should be present")
		assert.True(t, foundGroupResource, "Group resource type should be present")
	})
}

// createTestUser creates a test user with the given username and returns the user ID and response
func createTestUser(t *testing.T, s *Suite, userName string) (string, map[string]interface{}) {
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": userName,
		"name": map[string]interface{}{
			"givenName":  "Test",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   userName,
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)

	// Verify the created user
	assert.Equal(t, userName, createResp["userName"])
	assert.Equal(t, true, createResp["active"])

	// Extract the user ID
	userID := createResp["id"].(string)
	assert.NotEmpty(t, userID)

	return userID, createResp
}

func testUsersBasicCRUD(t *testing.T, s *Suite) {
	// Test creating a user
	userName := "testuser@example.com"
	userID, _ := createTestUser(t, s, userName)

	// Test getting a user by ID
	var getResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userID), nil, http.StatusOK, &getResp)
	assert.Equal(t, userID, getResp["id"])
	assert.Equal(t, userName, getResp["userName"])
	assert.Equal(t, true, getResp["active"])

	// Test getting a user with a bad ID
	var errResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/99999"), nil, http.StatusNotFound, &errResp)
	assert.Contains(t, errResp["detail"], "Resource 99999 not found")
	assert.EqualValues(t, errResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	// Make sure the error is reflected in the last request
	scimDetails := contract.ScimDetailsResponse{}
	s.DoJSON(t, "GET", scimPath("/details"), nil, http.StatusOK, &scimDetails)
	require.NotNil(t, scimDetails.LastRequest)
	assert.Equal(t, "error", scimDetails.LastRequest.Status)
	assert.NotZero(t, scimDetails.LastRequest.RequestedAt)
	assert.Equal(t, errResp["detail"], scimDetails.LastRequest.Details)

	// Test listing users
	var listResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &listResp)
	assert.EqualValues(t, listResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	resources, ok := listResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, len(resources), 1, "Should have 1 user")

	// Test filtering users by userName
	var filterResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &filterResp, "filter", `userName eq "testuser@example.com"`)
	assert.EqualValues(t, filterResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	filterResources, ok := filterResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 1, len(filterResources), "Should have exactly 1 user matching the filter")

	// Test filtering users by userName with random capitalization (case insensitivity)
	randomCapUserName := "TeStUsEr@ExAmPlE.cOm" // Randomly capitalized version of testuser@example.com
	var caseInsensitiveResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &caseInsensitiveResp, "filter", `userName eq "`+randomCapUserName+`"`)
	assert.EqualValues(t, caseInsensitiveResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	caseInsensitiveResources, ok := caseInsensitiveResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 1, len(caseInsensitiveResources), "Should have exactly 1 user matching the case-insensitive filter")

	// Verify it's the same user
	if len(caseInsensitiveResources) > 0 {
		user, ok := caseInsensitiveResources[0].(map[string]interface{})
		assert.True(t, ok, "User should be an object")
		assert.Equal(t, userID, user["id"], "Should be the same user despite case differences in userName filter")
		assert.Equal(t, userName, user["userName"], "Original userName should be preserved")
	}

	// Test filtering users by non-existent userName
	filterResp = nil
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &filterResp, "filter", `userName eq "bozo@example.com"`)
	assert.EqualValues(t, filterResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	filterResources, ok = filterResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Empty(t, filterResources, "Should have no users matching the filter")

	// Test updating a user
	updateUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": userName,
		"name": map[string]interface{}{
			"givenName":  "Updated",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   userName,
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var updateResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+userID), updateUserPayload, http.StatusOK, &updateResp)
	assert.Equal(t, userName, updateResp["userName"])

	// Verify the name was updated
	name, ok := updateResp["name"].(map[string]interface{})
	assert.True(t, ok, "Name should be an object")
	assert.Equal(t, "Updated", name["givenName"])

	// Test patching a user (updating just the active status)
	patchUserPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "Replace",
				"path":  "active",
				"value": false,
			},
		},
	}

	var patchResp map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchUserPayload, http.StatusOK, &patchResp)
	assert.Equal(t, false, patchResp["active"])

	// Test patching a user without path attribute (updating just the active status)
	patchUserPayload = map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op": "replace",
				"value": map[string]interface{}{
					"active": true,
				},
			},
		},
	}

	patchResp = nil
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchUserPayload, http.StatusOK, &patchResp)
	assert.Equal(t, true, patchResp["active"])

	// Test deleting a user
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)

	// Verify the user was deleted by trying to get it (should return 404)
	var errorResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userID), nil, http.StatusNotFound, &errorResp)
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp["detail"], "not found")

	// Test replacing a user that doesn't exist
	nonExistentUserID := "99999"
	updateNonExistentUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "nonexistent@example.com",
		"name": map[string]interface{}{
			"givenName":  "Non",
			"familyName": "Existent",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "nonexistent@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var updateNonExistentResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+nonExistentUserID), updateNonExistentUserPayload, http.StatusNotFound, &updateNonExistentResp)
	assert.EqualValues(t, updateNonExistentResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, updateNonExistentResp["detail"], "not found")

	// Test deleting a user that was already deleted
	var deleteAgainResp map[string]interface{}
	s.DoJSON(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNotFound, &deleteAgainResp)
	assert.EqualValues(t, deleteAgainResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, deleteAgainResp["detail"], "not found")
}

// createTestGroup creates a test group with the given display name and members and returns the group ID and response
func createTestGroup(t *testing.T, s *Suite, displayName string, memberIDs []string) (string, map[string]interface{}) {
	members := make([]map[string]interface{}, 0, len(memberIDs))
	for _, id := range memberIDs {
		members = append(members, map[string]interface{}{
			"value": id,
		})
	}

	createGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": displayName,
	}

	if len(members) > 0 {
		createGroupPayload["members"] = members
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), createGroupPayload, http.StatusCreated, &createResp)

	// Verify the created group
	assert.Equal(t, displayName, createResp["displayName"])

	// Extract the group ID
	groupID := createResp["id"].(string)
	assert.NotEmpty(t, groupID)

	return groupID, createResp
}

func testGroupsBasicCRUD(t *testing.T, s *Suite) {
	// First, create a user to add as a member of the group
	userID, _ := createTestUser(t, s, "groupmember@example.com")

	// Test creating a group
	groupID, createResp := createTestGroup(t, s, "Test Group", []string{userID})

	// Verify members
	members, ok := createResp["members"].([]interface{})
	assert.True(t, ok, "Members should be an array")
	require.Equal(t, 1, len(members), "Should have 1 member")
	member := members[0].(map[string]interface{})
	assert.Equal(t, userID, member["value"])
	assert.Equal(t, "User", member["type"])

	// Test getting a group by ID
	var getResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups/"+groupID), nil, http.StatusOK, &getResp)
	assert.Equal(t, groupID, getResp["id"])
	assert.Equal(t, "Test Group", getResp["displayName"])

	// Verify members in the GET response
	getMembers, ok := getResp["members"].([]interface{})
	assert.True(t, ok, "Members should be an array")
	assert.Equal(t, 1, len(getMembers), "Should have 1 member")
	getMember := getMembers[0].(map[string]interface{})
	assert.Equal(t, userID, getMember["value"])

	// Test getting a group by ID with excludedAttributes=members
	var getExcludedResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups/"+groupID), nil, http.StatusOK, &getExcludedResp, "excludedAttributes", "members")
	assert.Equal(t, groupID, getExcludedResp["id"])
	assert.Equal(t, "Test Group", getExcludedResp["displayName"])

	// Verify members are not included in the response
	membersExist := getExcludedResp["members"] != nil
	assert.False(t, membersExist, "Members should not be included when excludedAttributes=members")

	// Test getting a group with a bad ID
	var errResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups/99999"), nil, http.StatusNotFound, &errResp)
	assert.Contains(t, errResp["detail"], "Resource 99999 not found")
	assert.EqualValues(t, errResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

	// Test listing groups
	var listResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &listResp)
	assert.EqualValues(t, listResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	resources, ok := listResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.GreaterOrEqual(t, len(resources), 1, "Should have at least 1 group")

	// Test listing groups with excludedAttributes=members
	var listExcludedResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &listExcludedResp, "excludedAttributes", "members")
	assert.EqualValues(t, listExcludedResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	resourcesExcluded, ok := listExcludedResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.GreaterOrEqual(t, len(resourcesExcluded), 1, "Should have at least 1 group")

	// Verify members are not included in any of the groups
	for _, resource := range resourcesExcluded {
		group, ok := resource.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		_, hasMembersField := group["members"]
		assert.False(t, hasMembersField, "Group should not have members field when excludedAttributes=members")
	}

	// Create a second group with a different display name for filtering tests
	secondGroupID, _ := createTestGroup(t, s, "Second Test Group", []string{userID})

	// Test filtering groups by displayName - first group
	var filterResp1 map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &filterResp1, "filter", `displayName eq "Test Group"`)
	assert.EqualValues(t, filterResp1["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	filterResources1, ok := filterResp1["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 1, len(filterResources1), "Should have exactly 1 group matching the filter")

	// Verify it's the correct group
	if len(filterResources1) > 0 {
		group, ok := filterResources1[0].(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		assert.Equal(t, groupID, group["id"], "Should be the first group")
		assert.Equal(t, "Test Group", group["displayName"], "Should have the correct display name")
	}

	// Test filtering groups by displayName - second group
	var filterResp2 map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &filterResp2, "filter", `displayName eq "Second Test Group"`)
	assert.EqualValues(t, filterResp2["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	filterResources2, ok := filterResp2["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 1, len(filterResources2), "Should have exactly 1 group matching the filter")

	// Verify it's the correct group
	if len(filterResources2) > 0 {
		group, ok := filterResources2[0].(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		assert.Equal(t, secondGroupID, group["id"], "Should be the second group")
		assert.Equal(t, "Second Test Group", group["displayName"], "Should have the correct display name")
	}

	// Test filtering groups by non-existent displayName
	var filterResp3 map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &filterResp3, "filter", `displayName eq "Non-Existent Group"`)
	assert.EqualValues(t, filterResp3["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	filterResources3, ok := filterResp3["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Empty(t, filterResources3, "Should have no groups matching the filter")

	// Test updating a group
	updateGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Updated Test Group",
		"members":     []map[string]interface{}{}, // Remove all members
	}

	var updateResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+groupID), updateGroupPayload, http.StatusOK, &updateResp)
	assert.Equal(t, "Updated Test Group", updateResp["displayName"])

	// Verify members were removed
	updateMembersExist := updateResp["members"] != nil
	assert.False(t, updateMembersExist, "Members should not be present or should be empty")
	assert.False(t, membersExist, "Members should not be present or should be empty")

	// Test deleting a group
	s.Do(t, "DELETE", scimPath("/Groups/"+groupID), nil, http.StatusNoContent)

	// Verify the group was deleted by trying to get it (should return 404)
	var errorResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups/"+groupID), nil, http.StatusNotFound, &errorResp)
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp["detail"], "not found")

	// Test replacing a group that doesn't exist
	nonExistentGroupID := "99999"
	updateNonExistentGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Non-Existent Group",
		"members":     []map[string]interface{}{},
	}

	var updateNonExistentResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+nonExistentGroupID), updateNonExistentGroupPayload, http.StatusNotFound, &updateNonExistentResp)
	assert.EqualValues(t, updateNonExistentResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, updateNonExistentResp["detail"], "not found")

	// Test deleting a group that was already deleted
	var deleteAgainResp map[string]interface{}
	s.DoJSON(t, "DELETE", scimPath("/Groups/"+groupID), nil, http.StatusNotFound, &deleteAgainResp)
	assert.EqualValues(t, deleteAgainResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, deleteAgainResp["detail"], "not found")

	// Delete the user we created
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
}

func testCreateGroup(t *testing.T, s *Suite) {
	// Create multiple test users to be added as members
	userIDs := make([]string, 0, 5)

	for i := 1; i <= 5; i++ {
		userName := fmt.Sprintf("group-test-user-%d@example.com", i)
		createUserPayload := map[string]interface{}{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": userName,
			"name": map[string]interface{}{
				"givenName":  fmt.Sprintf("User%d", i),
				"familyName": "GroupTest",
			},
			"emails": []map[string]interface{}{
				{
					"value":   userName,
					"type":    "work",
					"primary": true,
				},
			},
			"active": true,
		}

		var createResp map[string]interface{}
		s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)
		userID := createResp["id"].(string)
		userIDs = append(userIDs, userID)
	}

	// Test 1: Create a group with 0 members
	emptyGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Empty Group",
	}

	var emptyGroupResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), emptyGroupPayload, http.StatusCreated, &emptyGroupResp)

	// Verify the created group
	assert.Equal(t, "Empty Group", emptyGroupResp["displayName"])

	// Verify no members
	_, membersExist := emptyGroupResp["members"]
	assert.False(t, membersExist, "Members should not be present for an empty group")

	emptyGroupID := emptyGroupResp["id"].(string)
	assert.NotEmpty(t, emptyGroupID)

	// Test 2: Create a group with many members
	members := make([]map[string]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		members = append(members, map[string]interface{}{
			"value": userID,
		})
	}

	manyMembersGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Many Members Group",
		"members":     members,
	}

	var manyMembersGroupResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), manyMembersGroupPayload, http.StatusCreated, &manyMembersGroupResp)

	// Verify the created group
	assert.Equal(t, "Many Members Group", manyMembersGroupResp["displayName"])

	// Verify members
	respMembers, ok := manyMembersGroupResp["members"].([]interface{})
	assert.True(t, ok, "Members should be an array")
	assert.Equal(t, len(userIDs), len(respMembers), "Should have the same number of members as we added")

	// Verify each member is in the response
	memberValues := make([]string, 0, len(respMembers))
	for _, member := range respMembers {
		memberMap, ok := member.(map[string]interface{})
		assert.True(t, ok, "Member should be an object")
		memberValues = append(memberValues, memberMap["value"].(string))
		assert.Equal(t, "User", memberMap["type"])
	}

	for _, userID := range userIDs {
		assert.Contains(t, memberValues, userID, "User ID should be in the members list")
	}

	manyMembersGroupID := manyMembersGroupResp["id"].(string)
	assert.NotEmpty(t, manyMembersGroupID)

	// Test 3: Create a group with externalId
	externalIDGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "External ID Group",
		"externalId":  "external-system-group-789",
		"members": []map[string]interface{}{
			{
				"value": userIDs[0], // Just add the first user as a member
			},
		},
	}

	var externalIDGroupResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), externalIDGroupPayload, http.StatusCreated, &externalIDGroupResp)

	// Verify the created group
	assert.Equal(t, "External ID Group", externalIDGroupResp["displayName"])
	assert.Equal(t, "external-system-group-789", externalIDGroupResp["externalId"])

	// Verify members
	externalIDGroupMembers, ok := externalIDGroupResp["members"].([]interface{})
	assert.True(t, ok, "Members should be an array")
	assert.Equal(t, 1, len(externalIDGroupMembers), "Should have 1 member")

	externalIDGroupID := externalIDGroupResp["id"].(string)
	assert.NotEmpty(t, externalIDGroupID)

	// Test 4: Try to create a group with the same display name (should fail)
	duplicateGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Empty Group", // Same as the first group
	}

	var errorResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), duplicateGroupPayload, http.StatusConflict, &errorResp)

	// Verify error response
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test 4: Try to create a group without displayName (should fail)
	noDisplayNamePayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		// No displayName
	}

	errorResp = nil
	s.DoJSON(t, "POST", scimPath("/Groups"), noDisplayNamePayload, http.StatusBadRequest, &errorResp)

	// Verify error response
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp["detail"], "A required value was missing")

	// Test 5: Try to create a group with invalid member ID (should fail)
	invalidMemberPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Invalid Member Group",
		"members": []map[string]interface{}{
			{
				"value": "invalid-user-id",
			},
		},
	}

	errorResp = nil
	s.DoJSON(t, "POST", scimPath("/Groups"), invalidMemberPayload, http.StatusBadRequest, &errorResp)

	// Verify error response
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

	// Delete the groups we created
	s.Do(t, "DELETE", scimPath("/Groups/"+emptyGroupID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Groups/"+manyMembersGroupID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Groups/"+externalIDGroupID), nil, http.StatusNoContent)

	// Delete the users we created
	for _, userID := range userIDs {
		s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
	}
}

func testCreateUser(t *testing.T, s *Suite) {
	// Test creating a user without givenName
	userWithoutGivenName := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "no-given-name@example.com",
		"name": map[string]interface{}{
			"familyName": "NoGivenName",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "no-given-name@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var errorResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutGivenName, http.StatusBadRequest, &errorResp)
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Equal(t, errors.ScimErrorInvalidValue.Detail, errorResp["detail"])

	// Test creating a user without familyName
	userWithoutFamilyName := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "no-family-name@example.com",
		"name": map[string]interface{}{
			"givenName": "NoFamilyName",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "no-family-name@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	errorResp = nil
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutFamilyName, http.StatusBadRequest, &errorResp)
	assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Equal(t, errors.ScimErrorInvalidValue.Detail, errorResp["detail"])

	// Test creating a user without emails
	userWithoutEmails := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "no-emails@example.com",
		"name": map[string]interface{}{
			"givenName":  "No",
			"familyName": "Emails",
		},
		"active": true,
	}

	var createResp3 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutEmails, http.StatusCreated, &createResp3)
	assert.Equal(t, "no-emails@example.com", createResp3["userName"])
	userID3 := createResp3["id"].(string)

	// Verify emails is not present or empty
	_, hasEmails := createResp3["emails"]
	assert.False(t, hasEmails, "emails should not be present")

	// Test creating a user without active status
	userWithoutActive := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "no-active@example.com",
		"name": map[string]interface{}{
			"givenName":  "No",
			"familyName": "Active",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "no-active@example.com",
				"type":    "work",
				"primary": true,
			},
		},
	}

	var createResp4 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutActive, http.StatusCreated, &createResp4)
	assert.Equal(t, "no-active@example.com", createResp4["userName"])
	userID4 := createResp4["id"].(string)

	// Verify active is not present or nil
	_, hasActive := createResp4["active"]
	assert.False(t, hasActive, "active should not be present")

	// Test creating a user with multiple emails
	userWithMultipleEmails := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "multiple-emails@example.com",
		"name": map[string]interface{}{
			"givenName":  "Multiple",
			"familyName": "Emails",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "multiple-emails@example.com",
				"type":    "work",
				"primary": true,
			},
			{
				"value":   "multiple-emails-home@example.com",
				"type":    "home",
				"primary": false,
			},
			{
				"value":   "multiple-emails-other@example.com",
				"type":    "other",
				"primary": false,
			},
		},
		"active": true,
	}

	var createResp5 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithMultipleEmails, http.StatusCreated, &createResp5)
	assert.Equal(t, "multiple-emails@example.com", createResp5["userName"])
	userID5 := createResp5["id"].(string)

	// Verify multiple emails are present
	emails, ok := createResp5["emails"].([]interface{})
	assert.True(t, ok, "Emails should be an array")
	assert.Equal(t, 3, len(emails), "Should have 3 emails")

	// Test creating a user with empty userName
	userWithEmptyUserName := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "", // Empty userName
		"name": map[string]interface{}{
			"givenName":  "Empty",
			"familyName": "UserName",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "empty-username@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var errorResp1 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithEmptyUserName, http.StatusBadRequest, &errorResp1)

	// Verify error response
	assert.EqualValues(t, errorResp1["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp1["detail"], "Bad Request")

	// Test creating a user with duplicate userName
	duplicateUserNamePayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "multiple-emails@example.com", // Same as userWithMultipleEmails
		"name": map[string]interface{}{
			"givenName":  "Duplicate",
			"familyName": "UserName",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "duplicate@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var errorResp2 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), duplicateUserNamePayload, http.StatusConflict, &errorResp2)

	// Verify error response
	assert.EqualValues(t, errorResp2["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp2["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test creating a user with duplicate userName using different case.
	// userName must be case insensitive
	duplicateUserNamePayload = map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "Multiple-Emails@example.com", // Same as userWithMultipleEmails
		"name": map[string]interface{}{
			"givenName":  "Duplicate",
			"familyName": "UserName",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "duplicate@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var errorResp3 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), duplicateUserNamePayload, http.StatusConflict, &errorResp3)

	// Verify error response
	assert.EqualValues(t, errorResp3["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp3["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test creating a user with externalId
	userWithExternalID := map[string]interface{}{
		"schemas":    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName":   "external-id-user@example.com",
		"externalId": "external-system-123456",
		"name": map[string]interface{}{
			"givenName":  "External",
			"familyName": "IDUser",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "external-id-user@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp6 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithExternalID, http.StatusCreated, &createResp6)
	assert.Equal(t, "external-id-user@example.com", createResp6["userName"])
	userID6 := createResp6["id"].(string)

	// Verify externalId is present and correct
	assert.Equal(t, "external-system-123456", createResp6["externalId"])

	// Test creating a user with department
	userWithDepartment := map[string]interface{}{
		"schemas": []string{
			"urn:ietf:params:scim:schemas:core:2.0:User",
			"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
		},
		"userName": "user-with-department@example.com",
		"name": map[string]interface{}{
			"givenName":  "Foo",
			"familyName": "Bar",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "foobar@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
		"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": map[string]interface{}{
			"department": "Engineering",
		},
	}

	var createResp7 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithDepartment, http.StatusCreated, &createResp7)
	assert.Equal(t, "user-with-department@example.com", createResp7["userName"])
	userID7 := createResp7["id"].(string)

	// Verify department is present and correct
	m_, ok := createResp7["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"]
	require.True(t, ok)
	m, ok := m_.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Engineering", m["department"])

	// Make sure these users can be deleted.
	s.Do(t, "DELETE", scimPath("/Users/"+userID3), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID4), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID5), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID6), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID7), nil, http.StatusNoContent)
}

func testUpdateUser(t *testing.T, s *Suite) {
	// Create first user
	firstUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "first-user@example.com",
		"name": map[string]interface{}{
			"givenName":  "First",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "first-user@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var firstUserResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), firstUserPayload, http.StatusCreated, &firstUserResp)
	firstUserID := firstUserResp["id"].(string)
	assert.NotEmpty(t, firstUserID)

	// Create second user
	secondUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "second-user@example.com",
		"name": map[string]interface{}{
			"givenName":  "Second",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "second-user@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var secondUserResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), secondUserPayload, http.StatusCreated, &secondUserResp)
	secondUserID := secondUserResp["id"].(string)
	assert.NotEmpty(t, secondUserID)

	// Test 1: Try to update first user's userName to be exactly the same as second user's userName
	updatePayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "second-user@example.com", // Same as second user
		"name": map[string]interface{}{
			"givenName":  "First",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "first-user@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var errorResp1 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+firstUserID), updatePayload, http.StatusConflict, &errorResp1)

	// Verify error response
	assert.EqualValues(t, errorResp1["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp1["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test 2: Try to update first user's userName to be a case-randomized version of second user's userName
	updatePayload["userName"] = "SeCoNd-UsEr@ExAmPlE.cOm" // Case-randomized version of second user's userName

	var errorResp2 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+firstUserID), updatePayload, http.StatusConflict, &errorResp2)

	// Verify error response
	assert.EqualValues(t, errorResp2["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp2["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test 3: Try to update first user's userName to be exactly the same as its current userName (should succeed)
	updatePayload["userName"] = "first-user@example.com" // Same as current userName

	var updateResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+firstUserID), updatePayload, http.StatusOK, &updateResp)

	// Verify the update was successful
	assert.Equal(t, "first-user@example.com", updateResp["userName"])

	// Test 4: Try to update first user's userName to be a case-randomized version of its current userName (should succeed)
	updatePayload["userName"] = "FiRsT-uSeR@eXaMpLe.CoM" // Case-randomized version of current userName

	var updateResp2 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+firstUserID), updatePayload, http.StatusOK, &updateResp2)

	// Verify the update was successful.
	assert.Equal(t, "FiRsT-uSeR@eXaMpLe.CoM", updateResp2["userName"])

	// Test 5: Try to update first user's department (should succeed)
	schemas_, ok := updatePayload["schemas"]
	require.True(t, ok)
	schemas, ok := schemas_.([]string)
	require.True(t, ok)
	updatePayload["schemas"] = append(schemas, "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User")
	updatePayload["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"] = map[string]interface{}{
		"department": "Engineering",
	}

	var updateResp3 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+firstUserID), updatePayload, http.StatusOK, &updateResp3)

	// Verify the update was successful.
	m_, ok := updateResp3["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"]
	require.True(t, ok)
	m, ok := m_.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Engineering", m["department"])

	// Delete the users we created.
	s.Do(t, "DELETE", scimPath("/Users/"+firstUserID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+secondUserID), nil, http.StatusNoContent)
}

func testUpdateGroup(t *testing.T, s *Suite) {
	// Create a test user to be added as a member of the groups
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "group-update-test@example.com",
		"name": map[string]interface{}{
			"givenName":  "Group",
			"familyName": "UpdateTest",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "group-update-test@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createUserResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createUserResp)
	userID := createUserResp["id"].(string)
	assert.NotEmpty(t, userID)

	// Create first group
	firstGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "First Test Group",
		"members": []map[string]interface{}{
			{
				"value": userID,
			},
		},
	}

	var firstGroupResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), firstGroupPayload, http.StatusCreated, &firstGroupResp)
	firstGroupID := firstGroupResp["id"].(string)
	assert.NotEmpty(t, firstGroupID)

	// Create second group
	secondGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Second Test Group",
		"members": []map[string]interface{}{
			{
				"value": userID,
			},
		},
	}

	var secondGroupResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), secondGroupPayload, http.StatusCreated, &secondGroupResp)
	secondGroupID := secondGroupResp["id"].(string)
	assert.NotEmpty(t, secondGroupID)

	// Test 1: Try to update first group's displayName to be exactly the same as second group's displayName
	updatePayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Second Test Group", // Same as second group
		"members": []map[string]interface{}{
			{
				"value": userID,
			},
		},
	}

	var errorResp1 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+firstGroupID), updatePayload, http.StatusConflict, &errorResp1)

	// Verify error response
	assert.EqualValues(t, errorResp1["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp1["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test 2: Try to update first group's displayName to be a case-randomized version of second group's displayName
	updatePayload["displayName"] = "SeCoNd TeSt GrOuP" // Case-randomized version of second group's displayName

	var errorResp2 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+firstGroupID), updatePayload, http.StatusConflict, &errorResp2)

	// Verify error response
	assert.EqualValues(t, errorResp2["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp2["detail"], "One or more of the attribute values are already in use or are reserved")

	// Test 3: Try to update first group's displayName to be exactly the same as its current displayName (should succeed)
	updatePayload["displayName"] = "First Test Group" // Same as current displayName

	var updateResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+firstGroupID), updatePayload, http.StatusOK, &updateResp)

	// Verify the update was successful
	assert.Equal(t, "First Test Group", updateResp["displayName"])

	// Test 4: Try to update first group's displayName to be a case-randomized version of its current displayName (should succeed)
	updatePayload["displayName"] = "FiRsT TeSt GrOuP" // Case-randomized version of current displayName

	var updateResp2 map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+firstGroupID), updatePayload, http.StatusOK, &updateResp2)

	// Verify the update was successful but the displayName is normalized
	assert.Equal(t, "FiRsT TeSt GrOuP", updateResp2["displayName"])

	// Delete the users and groups we created.
	s.Do(t, "DELETE", scimPath("/Groups/"+firstGroupID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Groups/"+secondGroupID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
}

func testUsersPagination(t *testing.T, s *Suite) {
	// Create multiple users for pagination testing
	userIDs := make([]string, 0, 10)

	for i := 1; i <= 10; i++ {
		userName := fmt.Sprintf("pagination-user-%d@example.com", i)
		userID, _ := createTestUser(t, s, userName)
		userIDs = append(userIDs, userID)
	}

	// Test 1: Get first page with 3 users per page
	var page1Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &page1Resp, "startIndex", "1", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page1Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page1Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources1, ok := page1Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources1), "First page should have 3 users")

	// Verify the users on the first page
	userNames1 := make([]string, 0, 3)
	for _, resource := range resources1 {
		user, ok := resource.(map[string]interface{})
		assert.True(t, ok, "User should be an object")
		userName, ok := user["userName"].(string)
		assert.True(t, ok, "userName should be a string")
		userNames1 = append(userNames1, userName)
	}
	assert.Contains(t, userNames1, "pagination-user-1@example.com", "First page should contain user 1")
	assert.Contains(t, userNames1, "pagination-user-2@example.com", "First page should contain user 2")
	assert.Contains(t, userNames1, "pagination-user-3@example.com", "First page should contain user 3")

	// Test 2: Get second page with 3 users per page
	var page2Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &page2Resp, "startIndex", "4", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page2Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page2Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources2, ok := page2Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources2), "Second page should have 3 users")

	// Verify the users on the second page
	userNames2 := make([]string, 0, 3)
	for _, resource := range resources2 {
		user, ok := resource.(map[string]interface{})
		assert.True(t, ok, "User should be an object")
		userName, ok := user["userName"].(string)
		assert.True(t, ok, "userName should be a string")
		userNames2 = append(userNames2, userName)
	}
	assert.Contains(t, userNames2, "pagination-user-4@example.com", "Second page should contain user 4")
	assert.Contains(t, userNames2, "pagination-user-5@example.com", "Second page should contain user 5")
	assert.Contains(t, userNames2, "pagination-user-6@example.com", "Second page should contain user 6")

	// Test 3: Get third page with 3 users per page
	var page3Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &page3Resp, "startIndex", "7", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page3Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page3Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources3, ok := page3Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources3), "Third page should have 3 users")

	// Verify the users on the third page
	userNames3 := make([]string, 0, 3)
	for _, resource := range resources3 {
		user, ok := resource.(map[string]interface{})
		assert.True(t, ok, "User should be an object")
		userName, ok := user["userName"].(string)
		assert.True(t, ok, "userName should be a string")
		userNames3 = append(userNames3, userName)
	}
	assert.Contains(t, userNames3, "pagination-user-7@example.com", "Third page should contain user 7")
	assert.Contains(t, userNames3, "pagination-user-8@example.com", "Third page should contain user 8")
	assert.Contains(t, userNames3, "pagination-user-9@example.com", "Third page should contain user 9")

	// Test 4: Get fourth page with 3 users per page (should contain only 1 user)
	var page4Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &page4Resp, "startIndex", "10", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page4Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page4Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources4, ok := page4Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	require.Len(t, resources4, 1, "Fourth page should have 1 user")

	// Verify the user on the fourth page
	user4, ok := resources4[0].(map[string]interface{})
	assert.True(t, ok, "User should be an object")
	userName4, ok := user4["userName"].(string)
	assert.True(t, ok, "userName should be a string")
	assert.Equal(t, "pagination-user-10@example.com", userName4, "Fourth page should contain user 10")

	// Test 5: Get page with startIndex beyond the total results (should return empty resources)
	var emptyPageResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &emptyPageResp, "startIndex", "11", "count", "3")

	// Verify response structure
	assert.EqualValues(t, emptyPageResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), emptyPageResp["totalResults"], "Total results should be 10")

	// Verify resources
	emptyResources, ok := emptyPageResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Empty(t, emptyResources, "Page beyond total results should have 0 users")

	// Test 6: Get all users in a single page
	var allUsersResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users"), nil, http.StatusOK, &allUsersResp, "count", "20")

	// Verify response structure
	assert.EqualValues(t, allUsersResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), allUsersResp["totalResults"], "Total results should be 10")

	// Verify resources
	allResources, ok := allUsersResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 10, len(allResources), "All users page should have 10 users")

	// Delete all created users
	for _, userID := range userIDs {
		s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
	}
}

func testGroupsPagination(t *testing.T, s *Suite) {
	// First, create a user to be added as a member of some groups
	userID, _ := createTestUser(t, s, "group-pagination-member@example.com")

	// Create multiple groups for pagination testing
	groupIDs := make([]string, 0, 10)

	for i := 1; i <= 10; i++ {
		// Add the user as a member to even-numbered groups
		var memberIDs []string
		if i%2 == 0 {
			memberIDs = []string{userID}
		}

		groupID, _ := createTestGroup(t, s, fmt.Sprintf("Pagination Group %d", i), memberIDs)
		groupIDs = append(groupIDs, groupID)
	}

	// Test 1: Get first page with 3 groups per page
	var page1Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &page1Resp, "startIndex", "1", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page1Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page1Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources1, ok := page1Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources1), "First page should have 3 groups")

	// Verify the groups on the first page
	displayNames1 := make([]string, 0, 3)
	for _, resource := range resources1 {
		group, ok := resource.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		displayName, ok := group["displayName"].(string)
		assert.True(t, ok, "displayName should be a string")
		displayNames1 = append(displayNames1, displayName)
	}
	assert.Contains(t, displayNames1, "Pagination Group 1", "First page should contain group 1")
	assert.Contains(t, displayNames1, "Pagination Group 2", "First page should contain group 2")
	assert.Contains(t, displayNames1, "Pagination Group 3", "First page should contain group 3")

	// Test 2: Get second page with 3 groups per page
	var page2Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &page2Resp, "startIndex", "4", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page2Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page2Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources2, ok := page2Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources2), "Second page should have 3 groups")

	// Verify the groups on the second page
	displayNames2 := make([]string, 0, 3)
	for _, resource := range resources2 {
		group, ok := resource.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		displayName, ok := group["displayName"].(string)
		assert.True(t, ok, "displayName should be a string")
		displayNames2 = append(displayNames2, displayName)
	}
	assert.Contains(t, displayNames2, "Pagination Group 4", "Second page should contain group 4")
	assert.Contains(t, displayNames2, "Pagination Group 5", "Second page should contain group 5")
	assert.Contains(t, displayNames2, "Pagination Group 6", "Second page should contain group 6")

	// Test 3: Get third page with 3 groups per page
	var page3Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &page3Resp, "startIndex", "7", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page3Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page3Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources3, ok := page3Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 3, len(resources3), "Third page should have 3 groups")

	// Verify the groups on the third page
	displayNames3 := make([]string, 0, 3)
	for _, resource := range resources3 {
		group, ok := resource.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		displayName, ok := group["displayName"].(string)
		assert.True(t, ok, "displayName should be a string")
		displayNames3 = append(displayNames3, displayName)
	}
	assert.Contains(t, displayNames3, "Pagination Group 7", "Third page should contain group 7")
	assert.Contains(t, displayNames3, "Pagination Group 8", "Third page should contain group 8")
	assert.Contains(t, displayNames3, "Pagination Group 9", "Third page should contain group 9")

	// Test 4: Get fourth page with 3 groups per page (should contain only 1 group)
	var page4Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &page4Resp, "startIndex", "10", "count", "3")

	// Verify response structure
	assert.EqualValues(t, page4Resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), page4Resp["totalResults"], "Total results should be 10")

	// Verify resources
	resources4, ok := page4Resp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	require.Len(t, resources4, 1, "Fourth page should have 1 group")

	// Verify the group on the fourth page
	group4, ok := resources4[0].(map[string]interface{})
	assert.True(t, ok, "Group should be an object")
	displayName4, ok := group4["displayName"].(string)
	assert.True(t, ok, "displayName should be a string")
	assert.Equal(t, "Pagination Group 10", displayName4, "Fourth page should contain group 10")

	// Test 5: Get page with startIndex beyond the total results (should return empty resources)
	var emptyPageResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &emptyPageResp, "startIndex", "11", "count", "3")

	// Verify response structure
	assert.EqualValues(t, emptyPageResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), emptyPageResp["totalResults"], "Total results should be 10")

	// Verify resources
	emptyResources, ok := emptyPageResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Empty(t, emptyResources, "Page beyond total results should have 0 groups")

	// Test 6: Get all groups in a single page
	var allGroupsResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Groups"), nil, http.StatusOK, &allGroupsResp, "count", "20")

	// Verify response structure
	assert.EqualValues(t, allGroupsResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
	assert.Equal(t, float64(10), allGroupsResp["totalResults"], "Total results should be 10")

	// Verify resources
	allResources, ok := allGroupsResp["Resources"].([]interface{})
	assert.True(t, ok, "Resources should be an array")
	assert.Equal(t, 10, len(allResources), "All groups page should have 10 groups")

	// Test 7: Verify that even-numbered groups have the user as a member
	for _, resource := range allResources {
		group, ok := resource.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		displayName, ok := group["displayName"].(string)
		assert.True(t, ok, "displayName should be a string")

		// Extract the group number from the display name
		var groupNum int
		_, err := fmt.Sscanf(displayName, "Pagination Group %d", &groupNum)
		assert.NoError(t, err, "Should be able to extract group number from display name")

		// Check if the group has members based on its number
		if groupNum%2 == 0 {
			// Even-numbered groups should have the user as a member
			members, ok := group["members"].([]interface{})
			assert.True(t, ok, "members should be an array")
			assert.Equal(t, 1, len(members), "Even-numbered group should have 1 member")

			if len(members) > 0 {
				member, ok := members[0].(map[string]interface{})
				assert.True(t, ok, "Member should be an object")
				assert.Equal(t, userID, member["value"], "Member should be the test user")
			}
		} else {
			// Odd-numbered groups should not have members
			_, hasMembersField := group["members"]
			assert.False(t, hasMembersField, "Odd-numbered group should not have members field")
		}
	}

	// Delete all created groups
	for _, groupID := range groupIDs {
		s.Do(t, "DELETE", scimPath("/Groups/"+groupID), nil, http.StatusNoContent)
	}

	// Delete the user we created
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
}

func testUsersAndGroups(t *testing.T, s *Suite) {
	// Create multiple test users
	userIDs := make([]string, 0, 3)
	for i := 1; i <= 3; i++ {
		userName := fmt.Sprintf("user-group-test-%d@example.com", i)
		userID, _ := createTestUser(t, s, userName)
		userIDs = append(userIDs, userID)
	}

	// Create two groups with different membership patterns
	// Group 1: Contains users 1 and 2
	group1ID, _ := createTestGroup(t, s, "Test Group 1", []string{userIDs[0], userIDs[1]})

	// Group 2: Contains users 2 and 3
	group2ID, _ := createTestGroup(t, s, "Test Group 2", []string{userIDs[1], userIDs[2]})

	// Test 1: Verify that User 1 is in Group 1 only
	var user1Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userIDs[0]), nil, http.StatusOK, &user1Resp)

	// Check groups field in user response
	user1Groups, ok := user1Resp["groups"].([]interface{})
	assert.True(t, ok, "User should have groups field")
	assert.Equal(t, 1, len(user1Groups), "User 1 should be in 1 group")

	// Verify the group is Group 1
	if len(user1Groups) > 0 {
		group, ok := user1Groups[0].(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		assert.Equal(t, group1ID, group["value"], "User 1 should be in Group 1")
		assert.Equal(t, "Groups/"+group1ID, group["$ref"], "Group $ref should be correct")
		assert.Equal(t, "Test Group 1", group["display"], "Group display name should be correct")
	}

	// Test 2: Verify that User 2 is in both Group 1 and Group 2
	var user2Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userIDs[1]), nil, http.StatusOK, &user2Resp)

	// Check groups field in user response
	user2Groups, ok := user2Resp["groups"].([]interface{})
	assert.True(t, ok, "User should have groups field")
	assert.Equal(t, 2, len(user2Groups), "User 2 should be in 2 groups")

	// Verify the groups include both Group 1 and Group 2
	groupValues := make([]string, 0, 2)
	groupDisplays := make(map[string]string)
	for _, g := range user2Groups {
		group, ok := g.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		groupID := group["value"].(string)
		groupValues = append(groupValues, groupID)
		groupDisplays[groupID] = group["display"].(string)
	}
	assert.Contains(t, groupValues, group1ID, "User 2 should be in Group 1")
	assert.Contains(t, groupValues, group2ID, "User 2 should be in Group 2")
	assert.Equal(t, "Test Group 1", groupDisplays[group1ID], "Group 1 display name should be correct")
	assert.Equal(t, "Test Group 2", groupDisplays[group2ID], "Group 2 display name should be correct")

	// Test 3: Verify that User 3 is in Group 2 only
	var user3Resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userIDs[2]), nil, http.StatusOK, &user3Resp)

	// Check groups field in user response
	user3Groups, ok := user3Resp["groups"].([]interface{})
	assert.True(t, ok, "User should have groups field")
	assert.Equal(t, 1, len(user3Groups), "User 3 should be in 1 group")

	// Verify the group is Group 2
	if len(user3Groups) > 0 {
		group, ok := user3Groups[0].(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		assert.Equal(t, group2ID, group["value"], "User 3 should be in Group 2")
		assert.Equal(t, "Groups/"+group2ID, group["$ref"], "Group $ref should be correct")
		assert.Equal(t, "Test Group 2", group["display"], "Group display name should be correct")
	}

	// Test 4: Update Group 1 to remove User 1 and add User 3
	updateGroup1Payload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Test Group 1",
		"members": []map[string]interface{}{
			{
				"value": userIDs[1], // Keep User 2
			},
			{
				"value": userIDs[2], // Add User 3
			},
		},
	}

	var updateGroup1Resp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Groups/"+group1ID), updateGroup1Payload, http.StatusOK, &updateGroup1Resp)

	// Test 5: Verify that User 1 is no longer in any group
	user1Resp = nil
	s.DoJSON(t, "GET", scimPath("/Users/"+userIDs[0]), nil, http.StatusOK, &user1Resp)

	// Check groups field in user response
	_, hasGroups := user1Resp["groups"]
	assert.False(t, hasGroups, "User 1 should not have groups field or it should be empty")

	// Test 6: Verify that User 3 is now in both groups
	user3Resp = nil
	s.DoJSON(t, "GET", scimPath("/Users/"+userIDs[2]), nil, http.StatusOK, &user3Resp)

	// Check groups field in user response
	user3Groups, ok = user3Resp["groups"].([]interface{})
	assert.True(t, ok, "User should have groups field")
	assert.Equal(t, 2, len(user3Groups), "User 3 should be in 2 groups")

	// Verify the groups include both Group 1 and Group 2
	groupValues = make([]string, 0, 2)
	groupDisplays = make(map[string]string)
	for _, g := range user3Groups {
		group, ok := g.(map[string]interface{})
		assert.True(t, ok, "Group should be an object")
		groupID := group["value"].(string)
		groupValues = append(groupValues, groupID)
		groupDisplays[groupID] = group["display"].(string)
	}
	assert.Contains(t, groupValues, group1ID, "User 3 should be in Group 1")
	assert.Contains(t, groupValues, group2ID, "User 3 should be in Group 2")
	assert.Equal(t, "Test Group 1", groupDisplays[group1ID], "Group 1 display name should be correct")
	assert.Equal(t, "Test Group 2", groupDisplays[group2ID], "Group 2 display name should be correct")

	// Delete the groups we created
	s.Do(t, "DELETE", scimPath("/Groups/"+group1ID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Groups/"+group2ID), nil, http.StatusNoContent)

	// Delete the users we created
	for _, userID := range userIDs {
		s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
	}
}

func testPatchUserEmails(t *testing.T, s *Suite) {
	// Create a test user with initial email
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "patch-emails-test@example.com",
		"name": map[string]interface{}{
			"givenName":  "Patch",
			"familyName": "EmailsTest",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "patch-emails-test@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)
	userID := createResp["id"].(string)

	t.Run("Patch the user to replace emails with a new set of emails", func(t *testing.T) {
		patchEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "new-primary@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "secondary@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchEmailsPayload, http.StatusOK, &patchResp)

		// Verify the emails were updated
		emails, _ := patchResp["emails"].([]interface{})
		assert.Equal(t, 2, len(emails), "Should have 2 emails after patch")

		// Verify the email values
		emailValues := make([]string, 0, 2)
		primaryFound := false
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			emailValues = append(emailValues, email["value"].(string))

			// Check if this is the primary email
			if primary, ok := email["primary"].(bool); ok && primary {
				primaryFound = true
				assert.Equal(t, "new-primary@example.com", email["value"], "Primary email should be new-primary@example.com")
				assert.Equal(t, "work", email["type"], "Primary email should be of type work")
			}
		}
		assert.EqualValues(t, []string{"new-primary@example.com", "secondary@example.com"}, emailValues,
			"Emails should be new-primary@example.com and secondary@example.com")
		assert.True(t, primaryFound, "One email should be marked as primary")
	})

	t.Run("Verify that patching with no primary email is allowed", func(t *testing.T) {
		patchNoPrimaryPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value": "no-primary1@example.com",
								"type":  "work",
								// No primary field
							},
							{
								"value": "no-primary2@example.com",
								"type":  "home",
								// No primary field
							},
						},
					},
				},
			},
		}

		var noPrimaryResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchNoPrimaryPayload, http.StatusOK, &noPrimaryResp)

		// Verify the emails were updated
		noPrimaryEmails, _ := noPrimaryResp["emails"].([]interface{})
		assert.Equal(t, 2, len(noPrimaryEmails), "Should have 2 emails after patch")

		// Verify the email values
		noPrimaryEmailValues := make([]string, 0, 2)
		for _, e := range noPrimaryEmails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			noPrimaryEmailValues = append(noPrimaryEmailValues, email["value"].(string))
		}
		assert.EqualValues(t, []string{"no-primary1@example.com", "no-primary2@example.com"}, noPrimaryEmailValues,
			"Emails should be no-primary1@example.com and no-primary2@example.com")
	})

	t.Run("Verify that patching with an empty emails array removes all emails", func(t *testing.T) {
		patchEmptyEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{},
					},
				},
			},
		}

		var emptyEmailsResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchEmptyEmailsPayload, http.StatusOK, &emptyEmailsResp)

		// Verify the emails were updated (should be empty or not present)
		noEmails := emptyEmailsResp["emails"]
		assert.Empty(t, noEmails, "Emails should be empty after patch")
	})

	t.Run("Patch emails with explicit path", func(t *testing.T) {
		patchEmailsWithPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "emails",
					"value": []map[string]interface{}{
						{
							"value":   "explicit-path-primary@example.com",
							"type":    "work",
							"primary": true,
						},
						{
							"value":   "explicit-path-secondary@example.com",
							"type":    "home",
							"primary": false,
						},
					},
				},
			},
		}

		var patchEmailsWithPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchEmailsWithPathPayload, http.StatusOK, &patchEmailsWithPathResp)
		emails, ok := patchEmailsWithPathResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should have 2 emails after patch with explicit path")

		// Verify the email values
		emailValues := make([]string, 0, 2)
		primaryFound := false
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			emailValues = append(emailValues, email["value"].(string))

			// Check if this is the primary email
			if primary, ok := email["primary"].(bool); ok && primary {
				primaryFound = true
				assert.Equal(t, "explicit-path-primary@example.com", email["value"], "Primary email should be explicit-path-primary@example.com")
				assert.Equal(t, "work", email["type"], "Primary email should be of type work")
			}
		}
		assert.EqualValues(t, []string{"explicit-path-primary@example.com", "explicit-path-secondary@example.com"}, emailValues,
			"Email values should be updated.")
		assert.True(t, primaryFound, "One email should be marked as primary")
	})

	t.Run("Add a new email with path specified", func(t *testing.T) {
		// First, ensure we have a clean starting state with one email
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Now add a new email with path specified
		addEmailPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": "emails",
					"value": []map[string]interface{}{
						{
							"value":   "added-email@example.com",
							"type":    "home",
							"primary": false,
						},
					},
				},
			},
		}

		var addEmailResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addEmailPayload, http.StatusOK, &addEmailResp)

		// Verify both emails are present
		emails, ok := addEmailResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should have 2 emails after adding one")

		// Check that both the original and new emails are present
		emailValues := make([]string, 0, 2)
		emailTypes := make(map[string]string)
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			emailValue := email["value"].(string)
			emailValues = append(emailValues, emailValue)
			emailTypes[emailValue] = email["type"].(string)
		}

		assert.Contains(t, emailValues, "work-email@example.com", "Original work email should still be present")
		assert.Contains(t, emailValues, "added-email@example.com", "Added home email should be present")
		assert.Equal(t, "work", emailTypes["work-email@example.com"], "Original email should be of type work")
		assert.Equal(t, "home", emailTypes["added-email@example.com"], "Added email should be of type home")
	})

	t.Run("Add a new email without path specified", func(t *testing.T) {
		// First, ensure we have a clean starting state with one email
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Now add a new email without path specified (should add to the resource)
		addEmailPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "add",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "added-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var addEmailResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addEmailPayload, http.StatusOK, &addEmailResp)

		// Verify both emails are present
		emails, ok := addEmailResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should have 2 emails after adding one")

		// Check that both the original and new emails are present
		emailValues := make([]string, 0, 2)
		emailTypes := make(map[string]string)
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			emailValue := email["value"].(string)
			emailValues = append(emailValues, emailValue)
			emailTypes[emailValue] = email["type"].(string)
		}

		assert.Contains(t, emailValues, "work-email@example.com", "Original work email should still be present")
		assert.Contains(t, emailValues, "added-email@example.com", "Added home email should be present")
		assert.Equal(t, "work", emailTypes["work-email@example.com"], "Original email should be of type work")
		assert.Equal(t, "home", emailTypes["added-email@example.com"], "Added email should be of type home")
	})

	t.Run("Remove an email by type filter", func(t *testing.T) {
		// First, ensure we have both work and home emails
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Delete the home email using a type filter
		deleteEmailPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": `emails[type eq "home"]`,
				},
			},
		}

		var deleteEmailResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), deleteEmailPayload, http.StatusOK, &deleteEmailResp)

		// Verify only work email remains
		emails, ok := deleteEmailResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 1, len(emails), "Should have only 1 email after deleting home email")

		// Check that only the work email remains
		email, ok := emails[0].(map[string]interface{})
		assert.True(t, ok, "Email should be an object")
		assert.Equal(t, "work-email@example.com", email["value"], "Work email should remain")
		assert.Equal(t, "work", email["type"], "Remaining email should be of type work")
		assert.Equal(t, true, email["primary"], "Work email should be primary")
	})

	t.Run("Remove all emails", func(t *testing.T) {
		// First, ensure we have both work and home emails
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Delete all emails by removing the entire emails attribute
		deleteAllEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "emails",
				},
			},
		}

		var deleteAllEmailsResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), deleteAllEmailsPayload, http.StatusOK, &deleteAllEmailsResp)

		// Verify emails attribute is not present or is empty
		_, hasEmails := deleteAllEmailsResp["emails"]
		assert.False(t, hasEmails, "Emails attribute should be removed after deleting all emails")
	})

	t.Run("Combined add and remove operations in a single request", func(t *testing.T) {
		// First, ensure we have both work and home emails
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Perform combined operations: remove home email and add other email
		combinedOpsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": `emails[type eq "home"]`,
				},
				{
					"op":   "add",
					"path": "emails",
					"value": []map[string]interface{}{
						{
							"value":   "other-email@example.com",
							"type":    "other",
							"primary": false,
						},
					},
				},
			},
		}

		var combinedOpsResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), combinedOpsPayload, http.StatusOK, &combinedOpsResp)

		// Verify we have two emails: work and other (home was removed)
		emails, ok := combinedOpsResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should have 2 emails after combined operations")

		// Check that the work email remains and other email was added
		emailValues := make([]string, 0, 2)
		emailTypes := make(map[string]string)
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			emailValue := email["value"].(string)
			emailValues = append(emailValues, emailValue)
			emailTypes[emailValue] = email["type"].(string)
		}

		assert.Contains(t, emailValues, "work-email@example.com", "Work email should remain")
		assert.Contains(t, emailValues, "other-email@example.com", "Other email should be added")
		assert.NotContains(t, emailValues, "home-email@example.com", "Home email should be removed")
		assert.Equal(t, "work", emailTypes["work-email@example.com"], "Work email should be of type work")
		assert.Equal(t, "other", emailTypes["other-email@example.com"], "Other email should be of type other")
	})

	t.Run("Patch emails by type filter", func(t *testing.T) {
		// First, ensure we have both work and home emails
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Now patch only the work email using the filter
		patchWorkEmailPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": `emails[type eq "work"]`,
					"value": map[string]interface{}{
						"value":   "updated-work@example.com",
						"type":    "work",
						"primary": true,
					},
				},
			},
		}

		var patchWorkResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchWorkEmailPayload, http.StatusOK, &patchWorkResp)
		emails, ok := patchWorkResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should still have 2 emails after patching work email")

		// Check that only the work email was updated and home email remains unchanged
		var workEmail, homeEmail map[string]interface{}
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")

			if email["type"].(string) == "work" {
				workEmail = email
			} else if email["type"].(string) == "home" {
				homeEmail = email
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		require.NotNil(t, homeEmail, "Home email should exist")

		assert.Equal(t, "updated-work@example.com", workEmail["value"], "Work email should be updated")
		assert.Equal(t, "home-email@example.com", homeEmail["value"], "Home email should remain unchanged")
		assert.Equal(t, true, workEmail["primary"], "Work email should be primary")
		assert.Equal(t, false, homeEmail["primary"], "Home email should not be primary")
	})

	t.Run("Patch individual field of email by type", func(t *testing.T) {
		// First, ensure we have both work and home emails with known values
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Now patch only the primary field of the work email
		patchWorkPrimaryPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  `emails[type eq "work"].primary`,
					"value": false,
				},
			},
		}

		var patchPrimaryResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchWorkPrimaryPayload, http.StatusOK, &patchPrimaryResp)
		emails, ok := patchPrimaryResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should still have 2 emails after patching primary flag")

		// Check that only the primary flag of work email was updated
		var workEmail, homeEmail map[string]interface{}
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")

			if email["type"].(string) == "work" {
				workEmail = email
			} else if email["type"].(string) == "home" {
				homeEmail = email
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		require.NotNil(t, homeEmail, "Home email should exist")

		assert.Equal(t, "work-email@example.com", workEmail["value"], "Work email value should remain unchanged")
		assert.Equal(t, "home-email@example.com", homeEmail["value"], "Home email value should remain unchanged")
		assert.Equal(t, false, workEmail["primary"], "Work email primary flag should be updated to false")
		assert.Equal(t, false, homeEmail["primary"], "Home email primary flag should remain false")

		// Now patch only the value field of the home email
		patchHomeValuePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  `emails[type eq "home"].value`,
					"value": "updated-home@example.com",
				},
			},
		}

		var patchHomeValueResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchHomeValuePayload, http.StatusOK, &patchHomeValueResp)

		// Verify the value was updated correctly
		emails, ok = patchHomeValueResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should still have 2 emails after patching home email value")

		// Check that only the value of home email was updated
		workEmail, homeEmail = nil, nil
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")

			if email["type"].(string) == "work" {
				workEmail = email
			} else if email["type"].(string) == "home" {
				homeEmail = email
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		require.NotNil(t, homeEmail, "Home email should exist")

		assert.Equal(t, "work-email@example.com", workEmail["value"], "Work email value should remain unchanged")
		assert.Equal(t, "updated-home@example.com", homeEmail["value"], "Home email value should be updated")
		assert.Equal(t, false, workEmail["primary"], "Work email primary flag should remain false")
		assert.Equal(t, false, homeEmail["primary"], "Home email primary flag should remain false")
	})

	t.Run("Patch email type by type", func(t *testing.T) {
		// First, ensure we have both work and home emails with known values
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Now patch the type of the home email to "other"
		patchHomeTypePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  `emails[type eq "home"].type`,
					"value": "other",
				},
			},
		}

		var patchTypeResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchHomeTypePayload, http.StatusOK, &patchTypeResp)

		// Verify the type was updated correctly
		emails, ok := patchTypeResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should still have 2 emails after patching email type")

		// Check that the home email type was changed to "other"
		var workEmail, otherEmail map[string]interface{}
		var homeEmailFound bool

		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			switch email["type"].(string) {
			case "work":
				workEmail = email
			case "home":
				homeEmailFound = true
			case "other":
				otherEmail = email
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		require.NotNil(t, otherEmail, "Other email (formerly home) should exist")
		assert.False(t, homeEmailFound, "Home email should no longer exist")

		assert.Equal(t, "work-email@example.com", workEmail["value"], "Work email value should remain unchanged")
		assert.Equal(t, "home-email@example.com", otherEmail["value"], "Other email value should be the same as the former home email")
		assert.Equal(t, "other", otherEmail["type"], "Home email type should be changed to 'other'")
		assert.Equal(t, true, workEmail["primary"], "Work email primary flag should remain true")
		assert.Equal(t, false, otherEmail["primary"], "Other email primary flag should remain false")
	})

	t.Run("Add individual email attribute", func(t *testing.T) {
		// First, ensure we have a work email with known values
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Add a new email with just the value attribute first, because it is required
		addEmailTypePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": `emails[type eq "home"]`,
					"value": []map[string]interface{}{
						{
							"value": "home-email@example.com",
						},
					},
				},
			},
		}

		var addEmailTypeResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addEmailTypePayload, http.StatusOK, &addEmailTypeResp)

		// Now modify the type attribute of the email
		addEmailValuePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  `emails[type eq "home"].type`,
					"value": "other",
				},
			},
		}

		var addEmailValueResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addEmailValuePayload, http.StatusOK, &addEmailValueResp)

		// Finally add the primary attribute to the email
		addEmailPrimaryPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  `emails[type eq "other"].primary`,
					"value": true,
				},
			},
		}

		var addEmailPrimaryResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addEmailPrimaryPayload, http.StatusOK, &addEmailPrimaryResp)

		// Verify both emails are present and home email is now primary
		emails, ok := addEmailPrimaryResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")
		assert.Equal(t, 2, len(emails), "Should have 2 emails")

		var workEmail, otherEmail map[string]interface{}
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			if email["type"] == "work" {
				workEmail = email
			} else if email["type"] == "other" {
				otherEmail = email
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		require.NotNil(t, otherEmail, "Other email should exist")

		assert.Equal(t, "work-email@example.com", workEmail["value"], "Work email value should be correct")
		assert.Equal(t, "home-email@example.com", otherEmail["value"], "Other email value should be correct")
		assert.Equal(t, false, workEmail["primary"], "Work email should no longer be primary")
		assert.Equal(t, true, otherEmail["primary"], "Other email should now be primary")
	})

	t.Run("Remove individual email attribute", func(t *testing.T) {
		// First, ensure we have both work and home emails with known values
		setupEmailsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"emails": []map[string]interface{}{
							{
								"value":   "work-email@example.com",
								"type":    "work",
								"primary": true,
							},
							{
								"value":   "home-email@example.com",
								"type":    "home",
								"primary": false,
							},
						},
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupEmailsPayload, http.StatusOK, &setupResp)

		// Remove the primary attribute from the work email
		removePrimaryPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": `emails[type eq "work"].primary`,
				},
			},
		}

		var removePrimaryResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), removePrimaryPayload, http.StatusOK, &removePrimaryResp)

		// Verify the primary attribute was removed
		emails, ok := removePrimaryResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")

		var workEmail map[string]interface{}
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			if email["type"] == "work" {
				workEmail = email
				break
			}
		}

		require.NotNil(t, workEmail, "Work email should exist")
		_, hasPrimary := workEmail["primary"]
		assert.False(t, hasPrimary, "Work email should not have primary attribute")
	})

	// Test failure cases using table-driven tests
	t.Run("Email validation failure cases", func(t *testing.T) {
		// Define test cases
		testCases := []struct {
			name         string
			payload      map[string]interface{}
			errorMessage string
		}{
			{
				name: "Multiple primary emails",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "primary1@example.com",
										"type":    "work",
										"primary": true,
									},
									{
										"value":   "primary2@example.com",
										"type":    "home",
										"primary": true, // Second primary email
									},
								},
							},
						},
					},
				},
				errorMessage: "Only one email can be marked as primary",
			},
			{
				name: "Invalid email format",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "not-an-email", // Invalid email format (missing @ and domain)
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Empty email value",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "", // Empty email value
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Email missing @ symbol",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "testinvalid.com", // Email missing @ symbol
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Email value as a number",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   123, // Number instead of string
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Email value as a boolean",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   true, // Boolean instead of string
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Email type as a number",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "valid@example.com",
										"type":    123, // Number instead of string
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Email type as a boolean",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "valid@example.com",
										"type":    true, // Boolean instead of string
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Primary flag as a string",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "valid@example.com",
										"type":    "work",
										"primary": "true", // String instead of boolean
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Primary flag as a number",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   "valid@example.com",
										"type":    "work",
										"primary": 1, // Number instead of boolean
									},
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "Null email value",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": []map[string]interface{}{
									{
										"value":   nil, // Null email value
										"type":    "work",
										"primary": true,
									},
								},
							},
						},
					},
				},
				errorMessage: "Bad Request. Invalid parameter provided in request",
			},
			{
				name: "Null emails field",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"emails": nil,
							},
						},
					},
				},
				errorMessage: "A required value was missing",
			},
			{
				name: "Add operation with invalid email path",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "add",
							// We only support type as a subattribute filter
							"path": `emails[primary eq "true"]`,
							"value": map[string]interface{}{
								"value":   "new-email@example.com",
								"type":    "work",
								"primary": true,
							},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Add operation with missing values",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":    "add",
							"path":  "emails",
							"value": []map[string]interface{}{},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Remove operation with invalid email path",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "remove",
							"path": `emails[type eq "nonexistent"]`,
						},
					},
				},
				errorMessage: "Bad Request",
			},
		}

		// Run each test case
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel() // these failure test cases do not modify state, so they can run in parallel
				var errorResp map[string]interface{}
				s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), tc.payload, http.StatusBadRequest, &errorResp)
				assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
				assert.Contains(t, errorResp["detail"], tc.errorMessage)
			})
		}
	})
}

func testPatchUserAttributes(t *testing.T, s *Suite) {
	// Create a test user
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "patch-attributes-test@example.com",
		"name": map[string]interface{}{
			"givenName":  "Original",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "patch-attributes-test@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)
	userID := createResp["id"].(string)

	t.Run("Patch userName", func(t *testing.T) {
		patchUserNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"userName": "new-username@example.com",
					},
				},
			},
		}

		var patchUserNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchUserNamePayload, http.StatusOK, &patchUserNameResp)
		assert.Equal(t, "new-username@example.com", patchUserNameResp["userName"], "userName should be updated")
	})

	t.Run("Patch entire name object", func(t *testing.T) {
		patchNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"name": map[string]interface{}{
							"givenName":  "CompletelyNew",
							"familyName": "FullName",
						},
					},
				},
			},
		}

		var patchNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchNamePayload, http.StatusOK, &patchNameResp)
		name, ok := patchNameResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "CompletelyNew", name["givenName"], "givenName should be updated")
		assert.Equal(t, "FullName", name["familyName"], "familyName should be updated")
	})

	t.Run("Patch active status", func(t *testing.T) {
		patchActivePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"active": false,
					},
				},
			},
		}

		var patchActiveResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchActivePayload, http.StatusOK, &patchActiveResp)
		assert.Equal(t, false, patchActiveResp["active"], "active should be updated to false")
	})

	t.Run("Patch multiple attributes at once (userName, name, active)", func(t *testing.T) {
		patchMultiplePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"userName": "multi-update@example.com",
						"name": map[string]interface{}{
							"givenName":  "Multiple",
							"familyName": "Updates",
						},
						"active": true,
					},
				},
			},
		}

		var patchMultipleResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchMultiplePayload, http.StatusOK, &patchMultipleResp)
		assert.Equal(t, "multi-update@example.com", patchMultipleResp["userName"], "userName should be updated")
		assert.Equal(t, true, patchMultipleResp["active"], "active should be updated to true")
		name, ok := patchMultipleResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "Multiple", name["givenName"], "givenName should be updated")
		assert.Equal(t, "Updates", name["familyName"], "familyName should be updated")
	})

	t.Run("Patch department", func(t *testing.T) {
		patchDepartmentPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department": "QA",
					},
				},
			},
		}

		var patchDepartmentResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchDepartmentPayload, http.StatusOK, &patchDepartmentResp)
		// Verify department is present and correct
		m_, ok := patchDepartmentResp["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"]
		require.True(t, ok)
		m, ok := m_.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "QA", m["department"])

		// Now remove department using path.
		patchDepartmentPayload2 := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
				},
			},
		}

		var patchDepartmentResp2 map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchDepartmentPayload2, http.StatusOK, &patchDepartmentResp2)
		// Verify department is not present (if there are no extension attributes, then the whole map is not there).
		_, ok = patchDepartmentResp2["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"]
		require.False(t, ok)

		// Now re-add department using path.
		patchDepartmentPayload3 := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
					"value": "Engineering",
				},
			},
		}

		var patchDepartmentResp3 map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchDepartmentPayload3, http.StatusOK, &patchDepartmentResp3)
		// Verify department is present and correct
		m_, ok = patchDepartmentResp3["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"]
		require.True(t, ok)
		m, ok = m_.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Engineering", m["department"])
	})

	// ///////////////////////////////////////////////
	// Tests for patching with explicit operation path

	t.Run("Patch userName with explicit path", func(t *testing.T) {
		patchUserNameWithPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "userName",
					"value": "explicit-path-username@example.com",
				},
			},
		}

		var patchUserNameWithPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchUserNameWithPathPayload, http.StatusOK, &patchUserNameWithPathResp)
		assert.Equal(t, "explicit-path-username@example.com", patchUserNameWithPathResp["userName"], "userName should be updated with explicit path")
	})

	t.Run("Patch name.givenName with explicit path", func(t *testing.T) {
		patchGivenNameWithPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "name.givenName",
					"value": "ExplicitPathGiven",
				},
			},
		}

		var patchGivenNameWithPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchGivenNameWithPathPayload, http.StatusOK, &patchGivenNameWithPathResp)
		nameObj1, ok := patchGivenNameWithPathResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "ExplicitPathGiven", nameObj1["givenName"], "givenName should be updated with explicit path")
		assert.Equal(t, "Updates", nameObj1["familyName"], "familyName should remain unchanged")
	})

	t.Run("Patch name.familyName with explicit path", func(t *testing.T) {
		patchFamilyNameWithPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "name.familyName",
					"value": "ExplicitPathFamily",
				},
			},
		}

		var patchFamilyNameWithPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchFamilyNameWithPathPayload, http.StatusOK, &patchFamilyNameWithPathResp)
		nameObj2, ok := patchFamilyNameWithPathResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "ExplicitPathGiven", nameObj2["givenName"], "givenName should remain unchanged")
		assert.Equal(t, "ExplicitPathFamily", nameObj2["familyName"], "familyName should be updated with explicit path")
	})

	t.Run("Patch active with explicit path", func(t *testing.T) {
		patchActiveWithPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "active",
					"value": false,
				},
			},
		}

		var patchActiveWithPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), patchActiveWithPathPayload, http.StatusOK, &patchActiveWithPathResp)
		assert.Equal(t, false, patchActiveWithPathResp["active"], "active should be updated to false with explicit path")
	})

	t.Run("Add a new attribute", func(t *testing.T) {
		// Add externalId attribute
		addExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "externalId",
					"value": "external-id-123",
				},
			},
		}

		var addExternalIdResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addExternalIdPayload, http.StatusOK, &addExternalIdResp)
		assert.Equal(t, "external-id-123", addExternalIdResp["externalId"], "externalId should be added")
	})

	t.Run("Delete an attribute", func(t *testing.T) {
		// First, ensure the user has an externalId
		setupExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"externalId": "external-id-to-delete",
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupExternalIdPayload, http.StatusOK, &setupResp)
		assert.Equal(t, "external-id-to-delete", setupResp["externalId"], "externalId should be set")

		// Now delete the externalId
		deleteExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "externalId",
				},
			},
		}

		var deleteExternalIdResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), deleteExternalIdPayload, http.StatusOK, &deleteExternalIdResp)
		_, hasExternalId := deleteExternalIdResp["externalId"]
		assert.False(t, hasExternalId, "externalId should be deleted")
	})

	t.Run("Add a new attribute without path specified", func(t *testing.T) {
		// First, ensure externalId is not present
		removeExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "externalId",
				},
			},
		}

		var removeResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), removeExternalIdPayload, http.StatusOK, &removeResp)

		// Add externalId attribute without path specified
		addExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "add",
					"value": map[string]interface{}{
						"externalId": "external-id-no-path",
					},
				},
			},
		}

		var addExternalIdResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addExternalIdPayload, http.StatusOK, &addExternalIdResp)
		assert.Equal(t, "external-id-no-path", addExternalIdResp["externalId"], "externalId should be added without path specified")
	})

	t.Run("Combined add and remove operations for attributes", func(t *testing.T) {
		// Setup initial state with externalId and active attributes
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"externalId": "initial-external-id",
						"active":     true,
					},
				},
			},
		}

		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), setupPayload, http.StatusOK, &setupResp)
		assert.Equal(t, "initial-external-id", setupResp["externalId"], "externalId should be set")
		assert.Equal(t, true, setupResp["active"], "active should be set to true")

		// Perform combined operations: remove externalId and add a new email
		combinedOpsPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "externalId",
				},
				{
					"op":   "add",
					"path": "emails",
					"value": []map[string]interface{}{
						{
							"value":   "new-combined-email@example.com",
							"type":    "work",
							"primary": true,
						},
					},
				},
			},
		}

		var combinedOpsResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), combinedOpsPayload, http.StatusOK, &combinedOpsResp)

		// Verify externalId is removed and new email is added
		_, hasExternalId := combinedOpsResp["externalId"]
		assert.False(t, hasExternalId, "externalId should be removed")

		emails, ok := combinedOpsResp["emails"].([]interface{})
		require.True(t, ok, "Response should have emails array")

		// Find the new email
		var foundNewEmail bool
		for _, e := range emails {
			email, ok := e.(map[string]interface{})
			assert.True(t, ok, "Email should be an object")
			if email["value"] == "new-combined-email@example.com" {
				foundNewEmail = true
				assert.Equal(t, "work", email["type"], "Email type should be work")
				assert.Equal(t, true, email["primary"], "Email should be primary")
			}
		}
		assert.True(t, foundNewEmail, "New email should be added")
	})

	t.Run("Add userName attribute", func(t *testing.T) {
		// Add userName attribute
		addUserNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "userName",
					"value": "new-username@example.com",
				},
			},
		}

		var addUserNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addUserNamePayload, http.StatusOK, &addUserNameResp)
		assert.Equal(t, "new-username@example.com", addUserNameResp["userName"], "userName should be updated")
	})

	t.Run("Add name attributes", func(t *testing.T) {
		// Add givenName attribute
		addGivenNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "name.givenName",
					"value": "NewGiven",
				},
			},
		}

		var addGivenNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addGivenNamePayload, http.StatusOK, &addGivenNameResp)
		name, ok := addGivenNameResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "NewGiven", name["givenName"], "givenName should be updated")

		// Add familyName attribute
		addFamilyNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "name.familyName",
					"value": "NewFamily",
				},
			},
		}

		var addFamilyNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addFamilyNamePayload, http.StatusOK, &addFamilyNameResp)
		name, ok = addFamilyNameResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "NewFamily", name["familyName"], "familyName should be updated")

		// Add entire name object
		addNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": "name",
					"value": map[string]interface{}{
						"givenName":  "CompletelyNew",
						"familyName": "FullName",
					},
				},
			},
		}

		var addNameResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), addNamePayload, http.StatusOK, &addNameResp)
		name, ok = addNameResp["name"].(map[string]interface{})
		require.True(t, ok, "Response should have name object")
		assert.Equal(t, "CompletelyNew", name["givenName"], "givenName should be updated")
		assert.Equal(t, "FullName", name["familyName"], "familyName should be updated")
	})

	// Failure tests using table-driven approach
	t.Run("Failure cases", func(t *testing.T) {
		testCases := []struct {
			name         string
			payload      map[string]interface{}
			errorMessage string
		}{
			{
				name: "Invalid userName (empty string)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"userName": "", // Empty userName
							},
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "userName as wrong type (number)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"userName": 12345, // Number instead of string
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "name as wrong type (string instead of object)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"name": "John Doe", // String instead of object
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "name.givenName without required familyName",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"name": map[string]interface{}{
									"givenName": "NewFirstName",
									// Missing familyName
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "givenName as wrong type (number)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"name": map[string]interface{}{
									"givenName":  12345, // Number instead of string
									"familyName": "NewLastName",
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "familyName as wrong type (boolean)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"name": map[string]interface{}{
									"givenName":  "NewFirstName",
									"familyName": true, // Boolean instead of string
								},
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "active as wrong type (string)",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							"value": map[string]interface{}{
								"active": "true", // String instead of boolean
							},
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "unsupported operation",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":    "bad",
							"path":  "active",
							"value": false,
						},
					},
				},
				errorMessage: errors.ScimErrorInvalidValue.Detail,
			},
			{
				name: "no path and invalid value format",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "replace",
							// No path specified
							"value": "not-a-map", // Should be a map with attributes
						},
					},
				},
				errorMessage: "A required value was missing",
			},
			{
				name: "wrong value type for active using path",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":    "replace",
							"path":  "active",
							"value": "not-a-boolean", // Should be a boolean
						},
					},
				},
				errorMessage: "A required value was missing",
			},
			{
				name: "Add operation with invalid path",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":    "add",
							"path":  "nonExistentAttribute",
							"value": "some value",
						},
					},
				},
				errorMessage: `The "path" attribute was invalid or malformed.`,
			},
			{
				name: "Add operation without value",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "add",
							"path": "externalId",
							// Missing value
						},
					},
				},
				errorMessage: "A required value was missing",
			},
			{
				name: "Remove operation without path",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op": "remove",
							// Missing path
						},
					},
				},
				errorMessage: "A required value was missing",
			},
			{
				name: "Remove required attribute - userName",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "remove",
							"path": "userName",
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Remove required attribute - givenName",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "remove",
							"path": "name.givenName",
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Remove required attribute - familyName",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "remove",
							"path": "name.familyName",
						},
					},
				},
				errorMessage: "Bad Request",
			},
			{
				name: "Remove required attribute - entire name object",
				payload: map[string]interface{}{
					"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
					"Operations": []map[string]interface{}{
						{
							"op":   "remove",
							"path": "name",
						},
					},
				},
				errorMessage: "Bad Request",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel() // Since these failure tests do not modify state, we can run them in parallel
				var errorResp map[string]interface{}
				s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), tc.payload, http.StatusBadRequest, &errorResp)
				assert.EqualValues(t, errorResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
				assert.Contains(t, errorResp["detail"], tc.errorMessage)
			})
		}
	})
}

func testPatchGroupAttributes(t *testing.T, s *Suite) {
	// Create a test user to be added as a member of the group
	userID, _ := createTestUser(t, s, "group-patch-test@example.com")

	// Create a test group
	createGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Original Group Name",
		"externalId":  "original-external-id",
		"members": []map[string]interface{}{
			{
				"value": userID,
			},
		},
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), createGroupPayload, http.StatusCreated, &createResp)
	groupID := createResp["id"].(string)

	t.Run("Replace displayName with explicit path", func(t *testing.T) {
		patchDisplayNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "displayName",
					"value": "Updated Group Name",
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchDisplayNamePayload, http.StatusOK, &patchResp)
		assert.Equal(t, "Updated Group Name", patchResp["displayName"], "displayName should be updated")
	})

	t.Run("Replace externalId with explicit path", func(t *testing.T) {
		patchExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "externalId",
					"value": "updated-external-id",
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchExternalIdPayload, http.StatusOK, &patchResp)
		assert.Equal(t, "updated-external-id", patchResp["externalId"], "externalId should be updated")
	})

	t.Run("Remove and add operation for externalId", func(t *testing.T) {
		// First, remove the externalId
		removeExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "externalId",
				},
			},
		}

		var removeResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), removeExternalIdPayload, http.StatusOK, &removeResp)
		_, hasExternalId := removeResp["externalId"]
		assert.False(t, hasExternalId, "externalId should be removed")

		// Now add a new externalId
		addExternalIdPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "externalId",
					"value": "added-external-id",
				},
			},
		}

		var addResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), addExternalIdPayload, http.StatusOK, &addResp)
		assert.Equal(t, "added-external-id", addResp["externalId"], "externalId should be added")
	})

	t.Run("Multiple operations in one request", func(t *testing.T) {
		multiOperationPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  "externalId",
					"value": "multi-op-external-id",
				},
				{
					"op":    "replace",
					"path":  "displayName",
					"value": "Multi-Op Group Name",
				},
			},
		}

		var multiOpResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), multiOperationPayload, http.StatusOK, &multiOpResp)
		assert.Equal(t, "multi-op-external-id", multiOpResp["externalId"], "externalId should be added")
		assert.Equal(t, "Multi-Op Group Name", multiOpResp["displayName"], "displayName should be updated")
	})

	t.Run("Operations without path attribute", func(t *testing.T) {
		noPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "replace",
					"value": map[string]interface{}{
						"displayName": "No-Path Group Name",
						"externalId":  "no-path-external-id",
					},
				},
			},
		}

		var noPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), noPathPayload, http.StatusOK, &noPathResp)
		assert.Equal(t, "no-path-external-id", noPathResp["externalId"], "externalId should be updated")
		assert.Equal(t, "No-Path Group Name", noPathResp["displayName"], "displayName should be updated")
	})

	t.Run("Attempt to remove required displayName attribute", func(t *testing.T) {
		removeDisplayNamePayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "displayName",
				},
			},
		}

		var errorResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), removeDisplayNamePayload, http.StatusBadRequest, &errorResp)
		assert.Contains(t, errorResp["detail"], "Bad Request", "Should return error for removing required attribute")
	})

	t.Run("Add members to group", func(t *testing.T) {
		// First, ensure the group has a known state with just the original user
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userID,
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member after setup")

		// Now add the second user to the group
		secondUserID, _ := createTestUser(t, s, "second-group-patch-test@example.com")
		addMembersPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": secondUserID,
						},
					},
				},
			},
		}

		var addResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), addMembersPayload, http.StatusOK, &addResp)

		// Verify both users are now in the group
		members, ok = addResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 2, len(members), "Group should have 2 members after adding one")

		// Check that both users are present
		memberValues := make([]string, 0, 2)
		for _, m := range members {
			member, ok := m.(map[string]interface{})
			assert.True(t, ok, "Member should be an object")
			memberValues = append(memberValues, member["value"].(string))
		}
		assert.Contains(t, memberValues, userID, "Original user should still be in the group")
		assert.Contains(t, memberValues, secondUserID, "Second user should be added to the group")
	})

	t.Run("Replace members in group", func(t *testing.T) {
		// First, ensure the group has a known state with multiple members
		setupUserID, _ := createTestUser(t, s, "setup-user@example.com")
		defer s.Do(t, "DELETE", scimPath("/Users/"+setupUserID), nil, http.StatusNoContent)

		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userID,
						},
						{
							"value": setupUserID,
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 2, len(members), "Group should have 2 members after setup")

		// Now replace all members with just the new user
		newUserID, _ := createTestUser(t, s, "replace-members-test@example.com")
		replaceMembersPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": newUserID,
						},
					},
				},
			},
		}

		var replaceResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), replaceMembersPayload, http.StatusOK, &replaceResp)

		// Verify only the new user is in the group
		members, ok = replaceResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have only 1 member after replacing")

		// Check that only the new user is present
		member, ok := members[0].(map[string]interface{})
		assert.True(t, ok, "Member should be an object")
		assert.Equal(t, newUserID, member["value"], "New user should be the only member")
	})

	t.Run("Remove all members from group", func(t *testing.T) {
		// First, ensure the group has members to remove
		setupUserID, _ := createTestUser(t, s, "remove-test-user@example.com")
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userID,
						},
						{
							"value": setupUserID,
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 2, len(members), "Group should have 2 members after setup")

		// Now remove all members
		removeMembersPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "members",
				},
			},
		}

		var removeResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), removeMembersPayload, http.StatusOK, &removeResp)

		// Verify no members are in the group
		_, hasMembers := removeResp["members"]
		assert.False(t, hasMembers, "Group should have no members after removing all")
	})

	t.Run("Add members without path attribute", func(t *testing.T) {
		// First, ensure the group has no members
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": "members",
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		_, hasMembers := setupResp["members"]
		assert.False(t, hasMembers, "Group should have no members after setup")

		// Now add a member without specifying path
		addMembersNoPathPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op": "add",
					"value": map[string]interface{}{
						"members": []map[string]interface{}{
							{
								"value": userID,
							},
						},
					},
				},
			},
		}

		var addNoPathResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), addMembersNoPathPayload, http.StatusOK, &addNoPathResp)

		// Verify the user is now in the group
		var members []interface{}
		var ok bool
		members, ok = addNoPathResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member after adding")

		// Check that the user is present
		member, ok := members[0].(map[string]interface{})
		assert.True(t, ok, "Member should be an object")
		assert.Equal(t, userID, member["value"], "Original user should be added back to the group")
	})

	t.Run("Invalid member ID format", func(t *testing.T) {
		// Try to add a member with invalid ID format
		invalidMemberPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": "invalid-user-id",
						},
					},
				},
			},
		}

		var errorResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), invalidMemberPayload, http.StatusBadRequest, &errorResp)
		assert.Contains(t, errorResp["detail"], "Bad Request", "Should return error for invalid member ID format")
	})

	t.Run("Non-existent member ID", func(t *testing.T) {
		// Try to add a member with non-existent ID
		nonExistentMemberPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": "4294967295", // Non-existent user ID
						},
					},
				},
			},
		}

		var errorResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), nonExistentMemberPayload, http.StatusBadRequest, &errorResp)
		assert.Contains(t, errorResp["detail"], "Bad Request", "Should return error for non-existent member ID")
	})
}

func testPatchGroupMembers(t *testing.T, s *Suite) {
	// Create test users to be added as members
	userIDs := make([]string, 0, 3)
	for i := 1; i <= 3; i++ {
		userName := fmt.Sprintf("group-patch-test-user-%d@example.com", i)
		userID, _ := createTestUser(t, s, userName)
		userIDs = append(userIDs, userID)
	}

	// Create a group with the first user as a member
	createGroupPayload := map[string]interface{}{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Patch Members Test Group",
		"members": []map[string]interface{}{
			{
				"value": userIDs[0],
			},
		},
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Groups"), createGroupPayload, http.StatusCreated, &createResp)
	groupID := createResp["id"].(string)

	t.Run("Add a member using path filtering", func(t *testing.T) {
		// Setup: Ensure the group has only the first user
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userIDs[0],
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member after setup")

		// Test: Add the second user to the group using path filtering
		patchAddMemberPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":    "add",
					"path":  fmt.Sprintf(`members[value eq "%s"]`, userIDs[1]),
					"value": map[string]interface{}{},
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchAddMemberPayload, http.StatusOK, &patchResp)

		// Verify the member was added
		members, ok = patchResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 2, len(members), "Group should now have 2 members")

		// Check that both users are in the members list
		memberValues := make([]string, 0, 2)
		for _, m := range members {
			member, ok := m.(map[string]interface{})
			assert.True(t, ok, "Member should be an object")
			memberValues = append(memberValues, member["value"].(string))
		}
		assert.Contains(t, memberValues, userIDs[0], "First user should still be a member")
		assert.Contains(t, memberValues, userIDs[1], "Second user should be added as a member")
	})

	t.Run("Remove a member using path filtering", func(t *testing.T) {
		// Setup: Ensure the group has both first and second users
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userIDs[0],
						},
						{
							"value": userIDs[1],
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 2, len(members), "Group should have 2 members after setup")

		// Test: Remove the first user from the group using path filtering
		patchRemoveMemberPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": fmt.Sprintf(`members[value eq "%s"]`, userIDs[0]),
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchRemoveMemberPayload, http.StatusOK, &patchResp)

		// Verify the member was removed
		members, ok = patchResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should now have 1 member")

		// Check that only the second user is in the members list
		member, ok := members[0].(map[string]interface{})
		assert.True(t, ok, "Member should be an object")
		assert.Equal(t, userIDs[1], member["value"], "Only the second user should remain as a member")
	})

	t.Run("Replace a member using path filtering", func(t *testing.T) {
		// Setup: Ensure the group has only the second user
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userIDs[1],
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member after setup")

		// Test: Replace the second user with the third user using path filtering
		patchReplaceMemberPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": fmt.Sprintf(`members[value eq "%s"]`, userIDs[1]),
				},
				{
					"op":    "add",
					"path":  fmt.Sprintf(`members[value eq "%s"]`, userIDs[2]),
					"value": map[string]interface{}{},
				},
			},
		}

		var patchResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchReplaceMemberPayload, http.StatusOK, &patchResp)

		// Verify the member was replaced
		members, ok = patchResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should still have 1 member")

		// Check that only the third user is in the members list
		member, ok := members[0].(map[string]interface{})
		assert.True(t, ok, "Member should be an object")
		assert.Equal(t, userIDs[2], member["value"], "Only the third user should be a member")
	})

	t.Run("Try to remove a non-existent member", func(t *testing.T) {
		// Setup: Ensure the group has only the third user
		setupPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]interface{}{
						{
							"value": userIDs[2],
						},
					},
				},
			},
		}
		var setupResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), setupPayload, http.StatusOK, &setupResp)

		// Verify setup was successful
		members, ok := setupResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member after setup")

		// Test: Try to remove a user that is not a member
		patchRemoveNonExistentPayload := map[string]interface{}{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]interface{}{
				{
					"op":   "remove",
					"path": fmt.Sprintf(`members[value eq "%s"]`, userIDs[0]),
				},
			},
		}

		var removeMemberResp map[string]interface{}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+groupID), patchRemoveNonExistentPayload, http.StatusOK, &removeMemberResp)
		members, ok = removeMemberResp["members"].([]interface{})
		require.True(t, ok, "Response should have members array")
		assert.Equal(t, 1, len(members), "Group should have 1 member")
	})
}

func scimPath(suffix string) string {
	paths := []string{"/api/v1/fleet/scim", "/api/latest/fleet/scim"}
	prefix := paths[time.Now().UnixNano()%int64(len(paths))]
	return prefix + suffix
}

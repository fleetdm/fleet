package scim

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/service"
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
		{"CreateUser", testCreateUser},
		{"PatchUserFailure", testPatchUserFailure},
		{"UsersPagination", testUsersPagination},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS, tablesToTruncate...)
			c.fn(t, s)
		})
	}
}

var tablesToTruncate = []string{"host_scim_user", "scim_users", "scim_groups"}

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

	// Unauthorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestObserverUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusForbidden, &resp)
	assert.Contains(t, resp["detail"], "forbidden")
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

	// Authorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestMaintainerUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &resp)
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
}

func testBaseEndpoints(t *testing.T, s *Suite) {
	// Test /Schemas endpoint
	var schemasResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &schemasResp)

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
		} else if id == "urn:ietf:params:scim:schemas:core:2.0:Group" {
			foundGroup = true
		}
	}
	assert.True(t, foundUser, "User schema should be present")
	assert.True(t, foundGroup, "Group schema should be present")

	// Test /ServiceProviderConfig endpoint
	var configResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/ServiceProviderConfig"), nil, http.StatusOK, &configResp)

	// Verify service provider config response
	assert.EqualValues(t, configResp["schemas"], []interface{}{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"})
	assert.NotNil(t, configResp["documentationUri"])

	// Test /ResourceTypes endpoint
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
}

func testUsersBasicCRUD(t *testing.T, s *Suite) {
	// Test creating a user
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "testuser@example.com",
		"name": map[string]interface{}{
			"givenName":  "Test",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "testuser@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)

	// Verify the created user
	assert.Equal(t, "testuser@example.com", createResp["userName"])
	assert.Equal(t, true, createResp["active"])

	// Extract the user ID for subsequent operations
	userID := createResp["id"].(string)
	assert.NotEmpty(t, userID)

	// Test getting a user by ID
	var getResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/"+userID), nil, http.StatusOK, &getResp)
	assert.Equal(t, userID, getResp["id"])
	assert.Equal(t, "testuser@example.com", getResp["userName"])
	assert.Equal(t, true, getResp["active"])

	// Test getting a user with a bad ID
	var errResp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Users/99999"), nil, http.StatusNotFound, &errResp)
	assert.Contains(t, errResp["detail"], "Resource 99999 not found")
	assert.EqualValues(t, errResp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

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
		"userName": "testuser@example.com",
		"name": map[string]interface{}{
			"givenName":  "Updated",
			"familyName": "User",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "testuser@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var updateResp map[string]interface{}
	s.DoJSON(t, "PUT", scimPath("/Users/"+userID), updateUserPayload, http.StatusOK, &updateResp)
	assert.Equal(t, "testuser@example.com", updateResp["userName"])

	// Verify the name was updated
	name, ok := updateResp["name"].(map[string]interface{})
	assert.True(t, ok, "Name should be an object")
	assert.Equal(t, "Updated", name["givenName"])

	// Test patching a user (updating just the active status)
	patchUserPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "replace",
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

	var createResp1 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutGivenName, http.StatusCreated, &createResp1)
	assert.Equal(t, "no-given-name@example.com", createResp1["userName"])
	userID1 := createResp1["id"].(string)

	// Verify name only has familyName
	name1, ok := createResp1["name"].(map[string]interface{})
	assert.True(t, ok, "Name should be an object")
	assert.Equal(t, "NoGivenName", name1["familyName"])
	assert.Nil(t, name1["givenName"], "givenName should be nil")

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

	var createResp2 map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), userWithoutFamilyName, http.StatusCreated, &createResp2)
	assert.Equal(t, "no-family-name@example.com", createResp2["userName"])
	userID2 := createResp2["id"].(string)

	// Verify name only has givenName
	name2, ok := createResp2["name"].(map[string]interface{})
	assert.True(t, ok, "Name should be an object")
	assert.Equal(t, "NoFamilyName", name2["givenName"])
	assert.Nil(t, name2["familyName"], "familyName should be nil")

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

	// Make sure these users can be deleted.
	s.Do(t, "DELETE", scimPath("/Users/"+userID1), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID2), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID3), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID4), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID5), nil, http.StatusNoContent)
}

func testPatchUserFailure(t *testing.T, s *Suite) {
	// First create a user to patch
	createUserPayload := map[string]interface{}{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "patch-test@example.com",
		"name": map[string]interface{}{
			"givenName":  "Patch",
			"familyName": "Test",
		},
		"emails": []map[string]interface{}{
			{
				"value":   "patch-test@example.com",
				"type":    "work",
				"primary": true,
			},
		},
		"active": true,
	}

	var createResp map[string]interface{}
	s.DoJSON(t, "POST", scimPath("/Users"), createUserPayload, http.StatusCreated, &createResp)
	userID := createResp["id"].(string)

	// Test 1: Patch with unsupported operation (add instead of replace)
	unsupportedOpPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "add", // Only "replace" is supported
				"path":  "active",
				"value": false,
			},
		},
	}

	var errorResp1 map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), unsupportedOpPayload, http.StatusBadRequest, &errorResp1)
	assert.EqualValues(t, errorResp1["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp1["detail"], "Bad Request.")

	// Test 2: Patch with unsupported field (userName instead of active)
	unsupportedFieldPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "replace",
				"path":  "userName", // Only "active" is supported
				"value": "new-username@example.com",
			},
		},
	}

	var errorResp2 map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), unsupportedFieldPayload, http.StatusBadRequest, &errorResp2)
	assert.EqualValues(t, errorResp2["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp2["detail"], "Bad Request.")

	// Test 3: Patch with no path and invalid value format
	invalidValuePayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op": "replace",
				// No path specified
				"value": "not-a-map", // Should be a map with "active" key
			},
		},
	}

	var errorResp3 map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), invalidValuePayload, http.StatusBadRequest, &errorResp3)
	assert.EqualValues(t, errorResp3["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp3["detail"], "A required value was missing")

	// Test 4 A: Patch with multiple operations (only one is supported)
	multipleOpsPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "replace",
				"path":  "active",
				"value": false,
			},
			{
				"op":    "replace",
				"path":  "active",
				"value": true,
			},
		},
	}

	var errorResp4 map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), multipleOpsPayload, http.StatusBadRequest, &errorResp4)
	assert.EqualValues(t, errorResp4["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp4["detail"], "Bad Request. Invalid parameter provided in request: Operations")

	// Test 4 B: Patch with multiple values, only one is supported
	multipleValuesPayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op": "replace",
				"value": map[string]interface{}{
					"active": false,
					"name": map[string]interface{}{
						"givenName": "Updated",
					},
				},
			},
		},
	}

	var errorResp4B map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), multipleValuesPayload, http.StatusBadRequest, &errorResp4B)
	assert.EqualValues(t, errorResp4B["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp4B["detail"], "Bad Request")

	// Test 5: Patch with wrong value type for active
	wrongTypePayload := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]interface{}{
			{
				"op":    "replace",
				"path":  "active",
				"value": "not-a-boolean", // Should be a boolean
			},
		},
	}

	var errorResp5 map[string]interface{}
	s.DoJSON(t, "PATCH", scimPath("/Users/"+userID), wrongTypePayload, http.StatusBadRequest, &errorResp5)
	assert.EqualValues(t, errorResp5["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
	assert.Contains(t, errorResp5["detail"], "A required value was missing")

	// Clean up the created user
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
}

func testUsersPagination(t *testing.T, s *Suite) {
	// Create multiple users for pagination testing
	userIDs := make([]string, 0, 10)

	for i := 1; i <= 10; i++ {
		userName := fmt.Sprintf("pagination-user-%d@example.com", i)
		createUserPayload := map[string]interface{}{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			"userName": userName,
			"name": map[string]interface{}{
				"givenName":  fmt.Sprintf("User%d", i),
				"familyName": "Pagination",
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

	// Clean up all created users
	for _, userID := range userIDs {
		s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
	}
}

// func responseAsJSON(t *testing.T, resp map[string]interface{}) {
// 	formattedResp, err := json.MarshalIndent(resp, "", "  ")
// 	require.NoError(t, err)
// 	t.Logf("formatted resp: %s", string(formattedResp))
// }

func scimPath(suffix string) string {
	paths := []string{"/api/v1/fleet/scim", "/api/latest/fleet/scim"}
	prefix := paths[time.Now().UnixNano()%int64(len(paths))]
	return prefix + suffix
}

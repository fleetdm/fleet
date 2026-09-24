package scim

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupMembersByValue fetches a group and returns its members keyed by the
// member "value" attribute, mapped to the member "type" attribute.
func groupMembersByValue(t *testing.T, s *Suite, groupID string) map[string]string {
	var resp map[string]any
	s.DoJSON(t, "GET", scimPath("/Groups/"+groupID), nil, http.StatusOK, &resp)

	byValue := make(map[string]string)
	membersIntf, ok := resp["members"]
	if !ok {
		return byValue
	}
	members, ok := membersIntf.([]any)
	require.True(t, ok, "Members should be an array")
	for _, m := range members {
		member, ok := m.(map[string]any)
		require.True(t, ok, "Member should be an object")
		byValue[member["value"].(string)] = member["type"].(string)
	}
	return byValue
}

// testNestedGroups covers the SCIM API behavior for nested (parent -> child)
// group members, which Entra ID provisions as group-type members with values
// like "group-<id>" instead of flattening them into user members.
func testNestedGroups(t *testing.T, s *Suite) {
	userID, _ := createTestUser(t, s, "nested-groups-user@example.com")
	childAID, _ := createTestGroup(t, s, "Nested Child A", nil)
	childBID, _ := createTestGroup(t, s, "Nested Child B", nil)

	var parentID string
	t.Run("Create group with a user and a nested group member", func(t *testing.T) {
		parentID, _ = createTestGroup(t, s, "Nested Parent", []string{userID, childAID})

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 2)
		assert.Equal(t, "User", members[userID])
		assert.Equal(t, "Group", members[childAID])

		// Nested group members must also be excluded when the members attribute
		// is excluded: child groups are loaded separately from user members, but
		// both are part of "members" and must honor excludedAttributes.
		var resp map[string]any
		s.DoJSON(t, "GET", scimPath("/Groups/"+parentID)+"?excludedAttributes=members", nil, http.StatusOK, &resp)
		_, hasMembers := resp["members"]
		assert.False(t, hasMembers, "Members should be excluded")
	})

	t.Run("Create group with duplicate nested group members stores a single edge", func(t *testing.T) {
		dupGroupID, _ := createTestGroup(t, s, "Nested Dup Parent", []string{childAID, childAID})
		defer s.Do(t, "DELETE", scimPath("/Groups/"+dupGroupID), nil, http.StatusNoContent)

		members := groupMembersByValue(t, s, dupGroupID)
		require.Len(t, members, 1)
		assert.Equal(t, "Group", members[childAID])
	})

	t.Run("Create group with malformed nested member value fails", func(t *testing.T) {
		payload := map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": "Nested Invalid Member",
			"members": []map[string]any{
				{"value": "group-abc"},
			},
		}
		var errResp map[string]any
		s.DoJSON(t, "POST", scimPath("/Groups"), payload, http.StatusBadRequest, &errResp)
		assert.EqualValues(t, []any{"urn:ietf:params:scim:api:messages:2.0:Error"}, errResp["schemas"])
	})

	t.Run("Replace group members with PUT swaps nested groups", func(t *testing.T) {
		putPayload := map[string]any{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"displayName": "Nested Parent",
			"members": []map[string]any{
				{"value": userID},
				{"value": childBID},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PUT", scimPath("/Groups/"+parentID), putPayload, http.StatusOK, &resp)

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 2)
		assert.Equal(t, "User", members[userID])
		assert.Equal(t, "Group", members[childBID])
		assert.NotContains(t, members, childAID)
	})

	t.Run("Patch add nested group member without path filter", func(t *testing.T) {
		patchPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]any{
						{"value": childAID},
					},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), patchPayload, http.StatusOK, &resp)

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 3)
		assert.Equal(t, "Group", members[childAID])
		assert.Equal(t, "Group", members[childBID])
	})

	t.Run("Patch add nonexistent nested group fails", func(t *testing.T) {
		patchPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]any{
						{"value": "group-999999"},
					},
				},
			},
		}
		var errResp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), patchPayload, http.StatusBadRequest, &errResp)
		assert.EqualValues(t, []any{"urn:ietf:params:scim:api:messages:2.0:Error"}, errResp["schemas"])

		// The group is unchanged.
		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 3)
	})

	t.Run("Patch replace members with nested groups only", func(t *testing.T) {
		patchPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]any{
						{"value": childBID},
					},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), patchPayload, http.StatusOK, &resp)

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 1)
		assert.Equal(t, "Group", members[childBID])
	})

	t.Run("Patch nested group member with path filter", func(t *testing.T) {
		// Add child A via members[value eq "group-<id>"].
		addPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":    "add",
					"path":  `members[value eq "` + childAID + `"]`,
					"value": map[string]any{},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), addPayload, http.StatusOK, &resp)

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 2)
		assert.Equal(t, "Group", members[childAID])
		assert.Equal(t, "Group", members[childBID])

		// Remove child B via members[value eq "group-<id>"].
		removePayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "remove",
					"path": `members[value eq "` + childBID + `"]`,
				},
			},
		}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), removePayload, http.StatusOK, &resp)

		members = groupMembersByValue(t, s, parentID)
		require.Len(t, members, 1)
		assert.Equal(t, "Group", members[childAID])

		// Adding a nonexistent nested group via path filter fails.
		addMissingPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":    "add",
					"path":  `members[value eq "group-999999"]`,
					"value": map[string]any{},
				},
			},
		}
		var errResp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), addMissingPayload, http.StatusBadRequest, &errResp)
	})

	t.Run("Patch remove all members removes nested groups", func(t *testing.T) {
		// Start from a known state with a user and a nested group.
		replacePayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]any{
						{"value": userID},
						{"value": childAID},
					},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), replacePayload, http.StatusOK, &resp)
		require.Len(t, groupMembersByValue(t, s, parentID), 2)

		removePayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "remove",
					"path": "members",
				},
			},
		}
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), removePayload, http.StatusOK, &resp)
		assert.Empty(t, groupMembersByValue(t, s, parentID))
	})

	t.Run("Deleting a child group removes it from the parent's members", func(t *testing.T) {
		replacePayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "replace",
					"path": "members",
					"value": []map[string]any{
						{"value": userID},
						{"value": childAID},
						{"value": childBID},
					},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+parentID), replacePayload, http.StatusOK, &resp)
		require.Len(t, groupMembersByValue(t, s, parentID), 3)

		s.Do(t, "DELETE", scimPath("/Groups/"+childBID), nil, http.StatusNoContent)

		members := groupMembersByValue(t, s, parentID)
		require.Len(t, members, 2)
		assert.Equal(t, "User", members[userID])
		assert.Equal(t, "Group", members[childAID])
	})

	t.Run("Nested group cycles are tolerated", func(t *testing.T) {
		cycleUserID, _ := createTestUser(t, s, "nested-cycle-user@example.com")
		cycle1ID, _ := createTestGroup(t, s, "Nested Cycle 1", nil)
		cycle2ID, _ := createTestGroup(t, s, "Nested Cycle 2", []string{cycle1ID})
		defer s.Do(t, "DELETE", scimPath("/Users/"+cycleUserID), nil, http.StatusNoContent)
		defer s.Do(t, "DELETE", scimPath("/Groups/"+cycle1ID), nil, http.StatusNoContent)
		defer s.Do(t, "DELETE", scimPath("/Groups/"+cycle2ID), nil, http.StatusNoContent)

		// Close the cycle (1 -> 2 -> 1) and add a user to cycle 1 in the same
		// patch. Entra ID prevents cycles on its side, but Fleet must not error
		// or loop if one is ever provisioned (the recursive membership expansion
		// uses UNION, which guarantees termination).
		patchPayload := map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{
					"op":   "add",
					"path": "members",
					"value": []map[string]any{
						{"value": cycleUserID},
						{"value": cycle2ID},
					},
				},
			},
		}
		var resp map[string]any
		s.DoJSON(t, "PATCH", scimPath("/Groups/"+cycle1ID), patchPayload, http.StatusOK, &resp)

		members := groupMembersByValue(t, s, cycle1ID)
		require.Len(t, members, 2)
		assert.Equal(t, "User", members[cycleUserID])
		assert.Equal(t, "Group", members[cycle2ID])
		members = groupMembersByValue(t, s, cycle2ID)
		require.Len(t, members, 1)
		assert.Equal(t, "Group", members[cycle1ID])

		// The user's effective membership walks the cycle: direct member of
		// cycle 1, transitive member of cycle 2 (its parent), and cycle 1 is not
		// revisited. Each group must appear exactly once, proving the UNION
		// dedup in getScimUserGroups terminates the cycle.
		var userResp map[string]any
		s.DoJSON(t, "GET", scimPath("/Users/"+cycleUserID), nil, http.StatusOK, &userResp)
		groupsIntf, ok := userResp["groups"].([]any)
		require.True(t, ok, "User should have a groups array")
		groupValues := make([]string, 0, len(groupsIntf))
		groupDisplays := make([]string, 0, len(groupsIntf))
		for _, g := range groupsIntf {
			group, ok := g.(map[string]any)
			require.True(t, ok, "Group should be an object")
			groupValues = append(groupValues, group["value"].(string))
			groupDisplays = append(groupDisplays, group["display"].(string))
		}
		assert.ElementsMatch(t, []string{cycle1ID, cycle2ID}, groupValues)
		assert.ElementsMatch(t, []string{"Nested Cycle 1", "Nested Cycle 2"}, groupDisplays)
	})

	// Clean up (the suite also truncates SCIM tables after each case).
	s.Do(t, "DELETE", scimPath("/Groups/"+parentID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Groups/"+childAID), nil, http.StatusNoContent)
	s.Do(t, "DELETE", scimPath("/Users/"+userID), nil, http.StatusNoContent)
}

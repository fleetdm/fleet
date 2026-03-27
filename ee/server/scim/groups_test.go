package scim

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/scim2/filter-parser/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGroupHandler(ds *mock.Store) *GroupHandler {
	return NewGroupHandler(ds, slog.New(slog.DiscardHandler)).(*GroupHandler)
}

func newTestScimGroup(id uint, displayName string, memberIDs []uint) *fleet.ScimGroup {
	return &fleet.ScimGroup{
		ID:          id,
		DisplayName: displayName,
		ScimUsers:   memberIDs,
	}
}

// TestGroupHandlerPatchUnrecognizedAttributes tests that unrecognized attributes in a PATCH
// request are skipped rather than causing the entire operation to fail.
func TestGroupHandlerPatchUnrecognizedAttributes(t *testing.T) {
	setupMocks := func(t *testing.T, group *fleet.ScimGroup) (*mock.Store, **fleet.ScimGroup) {
		t.Helper()
		ds := new(mock.Store)
		ds.ScimGroupByIDFunc = func(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
			return group, nil
		}
		var saved *fleet.ScimGroup
		ds.ReplaceScimGroupFunc = func(ctx context.Context, g *fleet.ScimGroup) error {
			saved = g
			return nil
		}
		return ds, &saved
	}

	t.Run("members update succeeds when batched with an unrecognized path attribute", func(t *testing.T) {
		// When Okta (or another IdP) sends a PATCH with multiple Operations, an unrecognized
		// attribute path must not abort the batch. The member update must still be applied.
		group := newTestScimGroup(1, "Engineering", []uint{})
		ds, saved := setupMocks(t, group)
		ds.ScimUsersExistFunc = func(ctx context.Context, ids []uint) (bool, error) {
			return true, nil
		}

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)

		membersPath, err := filter.ParsePath([]byte(membersAttr))
		require.NoError(t, err)
		unknownPath, err := filter.ParsePath([]byte("unknownAttribute"))
		require.NoError(t, err)

		_, err = handler.Patch(req, "group-1", []scim.PatchOperation{
			// Unrecognized attribute comes first.
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "ignored"},
			// Member addition must still be applied.
			{Op: scim.PatchOperationAdd, Path: &membersPath, Value: []any{
				map[string]any{"value": "1"},
			}},
		})
		require.NoError(t, err)
		require.True(t, ds.ReplaceScimGroupFuncInvoked)
		require.NotNil(t, *saved)
		assert.Equal(t, []uint{1}, (*saved).ScimUsers)
	})

	t.Run("displayName update succeeds when batched with an unrecognized no-path field", func(t *testing.T) {
		// Same issue in the no-path handler: when op.Path is nil, the code iterates over the
		// value map keys. An unrecognized key must be skipped, not abort the request.
		group := newTestScimGroup(1, "OldName", []uint{})
		ds, saved := setupMocks(t, group)

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)

		_, err := handler.Patch(req, "group-1", []scim.PatchOperation{
			{
				Op:   scim.PatchOperationReplace,
				Path: nil,
				Value: map[string]any{
					"unknownField":  "ignored",
					displayNameAttr: "NewName",
				},
			},
		})
		require.NoError(t, err)
		require.True(t, ds.ReplaceScimGroupFuncInvoked)
		require.NotNil(t, *saved)
		assert.Equal(t, "NewName", (*saved).DisplayName)
	})

	t.Run("members update still fails correctly for truly invalid member data", func(t *testing.T) {
		// Skipping unknown attributes must not suppress errors for genuinely bad data
		// (e.g. a member reference pointing to a non-existent user).
		group := newTestScimGroup(1, "Engineering", []uint{})
		ds, _ := setupMocks(t, group)
		ds.ScimUsersExistFunc = func(ctx context.Context, ids []uint) (bool, error) {
			return false, nil
		}

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)

		membersPath, err := filter.ParsePath([]byte(membersAttr))
		require.NoError(t, err)

		_, err = handler.Patch(req, "group-1", []scim.PatchOperation{
			{Op: scim.PatchOperationAdd, Path: &membersPath, Value: []any{
				map[string]any{"value": "999"},
			}},
		})
		require.Error(t, err)
		assert.False(t, ds.ReplaceScimGroupFuncInvoked)
	})

	t.Run("group not found returns 404, not affected by the skip-unknown change", func(t *testing.T) {
		ds := new(mock.Store)
		ds.ScimGroupByIDFunc = func(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
			return nil, platform_mysql.NotFound("ScimGroup")
		}

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-999", nil)

		unknownPath, err := filter.ParsePath([]byte("unknownAttribute"))
		require.NoError(t, err)

		_, err = handler.Patch(req, "group-999", []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "ignored"},
		})
		require.Error(t, err)
		assert.False(t, ds.ReplaceScimGroupFuncInvoked)
	})

	t.Run("Patch with only unrecognized path attributes does not write to database", func(t *testing.T) {
		// When all operations contain only unrecognized attributes, there is nothing to persist.
		// ReplaceScimGroup must not be called to avoid an unnecessary DB round-trip.
		group := newTestScimGroup(1, "Engineering", []uint{})
		ds, _ := setupMocks(t, group)

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)

		unknownPath, err := filter.ParsePath([]byte("unknownAttribute"))
		require.NoError(t, err)

		_, err = handler.Patch(req, "group-1", []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "ignored"},
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "also ignored"},
		})
		require.NoError(t, err)
		assert.False(t, ds.ReplaceScimGroupFuncInvoked)
	})

	t.Run("no-path Patch with only unrecognized value fields does not write to database", func(t *testing.T) {
		// Same guard for the nil-path code path: if all keys in the value map are unrecognized,
		// ReplaceScimGroup must not be called.
		group := newTestScimGroup(1, "Engineering", []uint{})
		ds, _ := setupMocks(t, group)

		handler := newTestGroupHandler(ds)
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)

		_, err := handler.Patch(req, "group-1", []scim.PatchOperation{
			{
				Op:    scim.PatchOperationReplace,
				Path:  nil,
				Value: map[string]any{"unknownField": "ignored"},
			},
		})
		require.NoError(t, err)
		assert.False(t, ds.ReplaceScimGroupFuncInvoked)
	})
}

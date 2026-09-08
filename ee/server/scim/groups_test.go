package scim

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/scim2/filter-parser/v2"
	"github.com/stretchr/testify/require"
)

type groupTestMocks struct {
	ds *mock.Store

	patched  fleet.ScimGroupMemberDeltas
	replaced *fleet.ScimGroup
}

func newGroupTestMocks(staleMembers, staleChildGroups []uint) *groupTestMocks {
	m := &groupTestMocks{ds: new(mock.Store)}
	m.ds.ScimGroupByIDFunc = func(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
		return &fleet.ScimGroup{
			ID:          id,
			DisplayName: "Engineering",
			ScimUsers:   append([]uint{}, staleMembers...),
			ChildGroups: append([]uint{}, staleChildGroups...),
		}, nil
	}
	m.ds.ScimUsersExistFunc = func(ctx context.Context, ids []uint) (bool, error) { return true, nil }
	m.ds.ScimGroupsExistFunc = func(ctx context.Context, ids []uint) (bool, error) { return true, nil }
	m.ds.ApplyScimGroupPatchFunc = func(ctx context.Context, group *fleet.ScimGroup, deltas fleet.ScimGroupMemberDeltas) error {
		m.patched = deltas
		return nil
	}
	m.ds.ReplaceScimGroupFunc = func(ctx context.Context, group *fleet.ScimGroup) error {
		m.replaced = group
		return nil
	}
	return m
}

func (m *groupTestMocks) newTestHandler() *GroupHandler {
	return &GroupHandler{ds: m.ds, logger: slog.New(slog.DiscardHandler)}
}

// requireDeltas requires that the patch was written as targeted deltas matching
// want, and not as a full membership rewrite.
func requireDeltas(t *testing.T, mocks *groupTestMocks, want fleet.ScimGroupMemberDeltas) {
	t.Helper()
	require.True(t, mocks.ds.ApplyScimGroupPatchFuncInvoked, "patch should be applied as deltas")
	require.False(t, mocks.ds.ReplaceScimGroupFuncInvoked, "patch must not rewrite the whole membership")
	require.ElementsMatch(t, want.AddUsers, mocks.patched.AddUsers)
	require.ElementsMatch(t, want.RemoveUsers, mocks.patched.RemoveUsers)
	require.ElementsMatch(t, want.AddChildGroups, mocks.patched.AddChildGroups)
	require.ElementsMatch(t, want.RemoveChildGroups, mocks.patched.RemoveChildGroups)
}

func requireFullWrite(t *testing.T, mocks *groupTestMocks) {
	t.Helper()
	require.True(t, mocks.ds.ReplaceScimGroupFuncInvoked, "patch should rewrite the whole membership")
	require.False(t, mocks.ds.ApplyScimGroupPatchFuncInvoked, "patch should not be applied as deltas")
}

// patchOp builds a patch operation, parsing path unless it is empty.
func patchOp(t *testing.T, op, path string, value any) scim.PatchOperation {
	t.Helper()
	operation := scim.PatchOperation{Op: op, Value: value}
	if path != "" {
		parsed, err := filter.ParsePath([]byte(path))
		require.NoError(t, err)
		operation.Path = &parsed
	}
	return operation
}

func memberValues(ids ...string) []any {
	members := make([]any, 0, len(ids))
	for _, id := range ids {
		members = append(members, map[string]any{"value": id})
	}
	return members
}

func TestGroupPatchMemberDeltas(t *testing.T) {
	patch := func(t *testing.T, mocks *groupTestMocks, ops ...scim.PatchOperation) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)
		_, err := mocks.newTestHandler().Patch(req, "group-1", ops)
		require.NoError(t, err)
	}

	t.Run("add never removes members missing from a stale read", func(t *testing.T) {
		// The read is stale: users 2 and 3 are members in the database but missing here.
		mocks := newGroupTestMocks([]uint{1}, nil)
		patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{AddUsers: []uint{4}})
	})

	t.Run("add without a path records the added members only", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		patch(t, mocks, patchOp(t, scim.PatchOperationAdd, "", map[string]any{membersAttr: memberValues("4", "group-7")}))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{AddUsers: []uint{4}, AddChildGroups: []uint{7}})
	})

	t.Run("displayName-only patch leaves membership untouched", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, []uint{7})
		patch(t, mocks, patchOp(t, scim.PatchOperationReplace, displayNameAttr, "Engineering EMEA"))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{})
	})

	t.Run("filtered remove records a targeted removal", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2, 3}, []uint{7})
		patch(t, mocks,
			patchOp(t, scim.PatchOperationRemove, `members[value eq "2"]`, nil),
			patchOp(t, scim.PatchOperationRemove, `members[value eq "group-7"]`, nil),
		)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{2}, RemoveChildGroups: []uint{7}})
	})

	t.Run("filtered remove of a member missing from the read still removes it", func(t *testing.T) {
		// A read that doesn't show the member may simply be stale.
		mocks := newGroupTestMocks([]uint{1}, nil)
		patch(t, mocks, patchOp(t, scim.PatchOperationRemove, `members[value eq "2"]`, nil))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{2}})
	})

	t.Run("filtered replace with a nil value records a removal", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, nil)
		patch(t, mocks, patchOp(t, scim.PatchOperationReplace, `members[value eq "2"]`, nil))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{2}})
	})

	t.Run("add then remove of the same member folds to a single removal", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		patch(t, mocks,
			patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")),
			patchOp(t, scim.PatchOperationRemove, `members[value eq "4"]`, nil),
		)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{4}})
	})

	t.Run("remove then add of the same member folds to a single add", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 4}, nil)
		patch(t, mocks,
			patchOp(t, scim.PatchOperationRemove, `members[value eq "4"]`, nil),
			patchOp(t, scim.PatchOperationAdd, `members[value eq "4"]`, nil),
		)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{AddUsers: []uint{4}})
	})

	t.Run("unfiltered replace writes the full membership state", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, nil)
		patch(t, mocks, patchOp(t, scim.PatchOperationReplace, membersAttr, memberValues("3")))

		requireFullWrite(t, mocks)
		require.Equal(t, []uint{3}, mocks.replaced.ScimUsers)
	})

	t.Run("remove all members writes the full membership state", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, []uint{7})
		patch(t, mocks, patchOp(t, scim.PatchOperationRemove, membersAttr, nil))

		requireFullWrite(t, mocks)
		require.Empty(t, mocks.replaced.ScimUsers)
		require.Empty(t, mocks.replaced.ChildGroups)
	})

	t.Run("unfiltered remove with a value removes only the named members", func(t *testing.T) {
		// The form Entra ID sends to remove a single member.
		mocks := newGroupTestMocks([]uint{1, 4}, []uint{7, 8})
		patch(t, mocks, patchOp(t, scim.PatchOperationRemove, membersAttr, memberValues("4", "group-7")))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{4}, RemoveChildGroups: []uint{7}})
	})

	t.Run("unfiltered remove with an empty value is a no-op, not a remove-all", func(t *testing.T) {
		// An empty list names nobody; only a remove with no value clears the group.
		mocks := newGroupTestMocks([]uint{1, 2}, []uint{7})
		patch(t, mocks, patchOp(t, scim.PatchOperationRemove, membersAttr, []any{}))

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{})
	})

	t.Run("an add after a replace still writes the full membership state", func(t *testing.T) {
		// The replace is what decides the write, so a later add must not downgrade it
		// back to a delta write, which would leave members 1 and 2 in place.
		mocks := newGroupTestMocks([]uint{1, 2}, nil)
		patch(t, mocks,
			patchOp(t, scim.PatchOperationReplace, membersAttr, memberValues("3")),
			patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")),
		)

		requireFullWrite(t, mocks)
		require.ElementsMatch(t, []uint{3, 4}, mocks.replaced.ScimUsers)
	})

	t.Run("the reads a patch depends on go to the primary", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		var groupReadPrimary, existsCheckPrimary bool
		mocks.ds.ScimGroupByIDFunc = func(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
			groupReadPrimary = ctxdb.IsPrimaryRequired(ctx)
			return &fleet.ScimGroup{ID: id, DisplayName: "Engineering", ScimUsers: []uint{1}}, nil
		}
		mocks.ds.ScimUsersExistFunc = func(ctx context.Context, ids []uint) (bool, error) {
			existsCheckPrimary = ctxdb.IsPrimaryRequired(ctx)
			return true, nil
		}

		patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")))

		require.True(t, groupReadPrimary, "the group read must not come from a replica")
		require.True(t, existsCheckPrimary, "the member existence check must not come from a replica")
	})
}

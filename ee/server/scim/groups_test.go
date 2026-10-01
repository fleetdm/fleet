package scim

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/scim2/filter-parser/v2"
	"github.com/stretchr/testify/require"
)

type groupTestMocks struct {
	ds *mock.Store

	patched  fleet.ScimGroupMemberDeltas
	replaced *fleet.ScimGroup
	created  *fleet.ScimGroup

	// Every ID is treated as existing unless listed here.
	unknownUsers  []uint
	unknownGroups []uint
}

func existingExcept(ids, unknown []uint) map[uint]struct{} {
	found := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if !slices.Contains(unknown, id) {
			found[id] = struct{}{}
		}
	}
	return found
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
	m.ds.ExistingScimUserIDsFunc = func(ctx context.Context, ids []uint) (map[uint]struct{}, error) {
		return existingExcept(ids, m.unknownUsers), nil
	}
	m.ds.ExistingScimGroupIDsFunc = func(ctx context.Context, ids []uint) (map[uint]struct{}, error) {
		return existingExcept(ids, m.unknownGroups), nil
	}
	m.ds.ScimGroupByDisplayNameFunc = func(ctx context.Context, displayName string) (*fleet.ScimGroup, error) {
		return nil, common_mysql.NotFound("ScimGroup")
	}
	m.ds.CreateScimGroupFunc = func(ctx context.Context, group *fleet.ScimGroup) (uint, error) {
		m.created = group
		return 1, nil
	}
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

	t.Run("the group read a patch depends on goes to the primary", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		var groupReadPrimary bool
		mocks.ds.ScimGroupByIDFunc = func(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
			groupReadPrimary = ctxdb.IsPrimaryRequired(ctx)
			return &fleet.ScimGroup{ID: id, DisplayName: "Engineering", ScimUsers: []uint{1}}, nil
		}

		patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")))

		require.True(t, groupReadPrimary, "the group read must not come from a replica")
	})
}

func responseMemberValues(t *testing.T, res scim.Resource) []string {
	t.Helper()
	members, _ := res.Attributes[membersAttr].([]scim.ResourceAttributes)
	values := make([]string, 0, len(members))
	for _, m := range members {
		values = append(values, m["value"].(string))
	}
	return values
}

func requireBadParams(t *testing.T, err error, wantValues ...string) {
	t.Helper()
	var scimErr scimerrors.ScimError
	require.ErrorAs(t, err, &scimErr)
	require.Equal(t, http.StatusBadRequest, scimErr.Status)
	require.Equal(t, scimerrors.ScimErrorBadParams(wantValues).Detail, scimErr.Detail)
}

func requireNoGroupWrite(t *testing.T, mocks *groupTestMocks) {
	t.Helper()
	require.False(t, mocks.ds.CreateScimGroupFuncInvoked)
	require.False(t, mocks.ds.ReplaceScimGroupFuncInvoked)
	require.False(t, mocks.ds.ApplyScimGroupPatchFuncInvoked)
}

func groupAttributes(memberIDs ...string) scim.ResourceAttributes {
	members := make([]any, 0, len(memberIDs))
	for _, id := range memberIDs {
		members = append(members, map[string]any{"value": id})
	}
	return scim.ResourceAttributes{displayNameAttr: "Engineering", membersAttr: members}
}

func TestGroupUnknownMembers(t *testing.T) {
	patch := func(t *testing.T, mocks *groupTestMocks, ops ...scim.PatchOperation) (scim.Resource, error) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Groups/group-1", nil)
		return mocks.newTestHandler().Patch(req, "group-1", ops)
	}
	create := func(t *testing.T, mocks *groupTestMocks, attrs scim.ResourceAttributes) (scim.Resource, error) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Groups", nil)
		return mocks.newTestHandler().Create(req, attrs)
	}
	replace := func(t *testing.T, mocks *groupTestMocks, attrs scim.ResourceAttributes) (scim.Resource, error) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/scim/v2/Groups/group-1", nil)
		return mocks.newTestHandler().Replace(req, "group-1", attrs)
	}

	t.Run("patch add skips an unknown user and adds the valid one", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		mocks.unknownUsers = []uint{999}
		res, err := patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4", "999")))
		require.NoError(t, err)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{AddUsers: []uint{4}})
		require.ElementsMatch(t, []string{"1", "4"}, responseMemberValues(t, res))
	})

	t.Run("patch add skips an unknown child group and adds the valid user", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		mocks.unknownGroups = []uint{77}
		res, err := patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4", "group-77")))
		require.NoError(t, err)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{AddUsers: []uint{4}})
		require.ElementsMatch(t, []string{"4"}, responseMemberValues(t, res))
	})

	t.Run("patch adding only unknown members is applied when it also removes members", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, nil)
		mocks.unknownUsers = []uint{999}
		res, err := patch(t, mocks,
			patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("999")),
			patchOp(t, scim.PatchOperationRemove, `members[value eq "2"]`, nil),
		)
		require.NoError(t, err)

		requireDeltas(t, mocks, fleet.ScimGroupMemberDeltas{RemoveUsers: []uint{2}})
		require.ElementsMatch(t, []string{"1"}, responseMemberValues(t, res))
	})

	t.Run("unfiltered patch replace skips unknown members", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1, 2}, nil)
		mocks.unknownUsers = []uint{999}
		res, err := patch(t, mocks, patchOp(t, scim.PatchOperationReplace, membersAttr, memberValues("3", "999")))
		require.NoError(t, err)

		requireFullWrite(t, mocks)
		require.Equal(t, []uint{3}, mocks.replaced.ScimUsers)
		require.ElementsMatch(t, []string{"3"}, responseMemberValues(t, res))
	})

	t.Run("create skips unknown members", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		mocks.unknownUsers = []uint{999}
		mocks.unknownGroups = []uint{77}
		res, err := create(t, mocks, groupAttributes("4", "999", "group-7", "group-77"))
		require.NoError(t, err)

		require.Equal(t, []uint{4}, mocks.created.ScimUsers)
		require.Equal(t, []uint{7}, mocks.created.ChildGroups)
		require.ElementsMatch(t, []string{"4", "group-7"}, responseMemberValues(t, res))
	})

	t.Run("replace skips unknown members", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		mocks.unknownUsers = []uint{999}
		res, err := replace(t, mocks, groupAttributes("4", "999"))
		require.NoError(t, err)

		require.Equal(t, []uint{4}, mocks.replaced.ScimUsers)
		require.ElementsMatch(t, []string{"4"}, responseMemberValues(t, res))
	})

	rejected := []struct {
		name string
		call func(t *testing.T, mocks *groupTestMocks) error
		want []string
	}{
		{
			name: "patch add",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("998", "999", "group-77")))
				return err
			},
			want: []string{"998", "999", "group-77"},
		},
		{
			name: "filtered patch add",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := patch(t, mocks, patchOp(t, scim.PatchOperationAdd, `members[value eq "999"]`, nil))
				return err
			},
			want: []string{"999"},
		},
		{
			name: "patch add with a displayName change",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := patch(t, mocks,
					patchOp(t, scim.PatchOperationReplace, displayNameAttr, "Renamed"),
					patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("999")),
				)
				return err
			},
			want: []string{"999"},
		},
		{
			name: "unfiltered patch replace",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := patch(t, mocks, patchOp(t, scim.PatchOperationReplace, membersAttr, memberValues("999")))
				return err
			},
			want: []string{"999"},
		},
		{
			name: "patch remove all then add",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := patch(t, mocks,
					patchOp(t, scim.PatchOperationRemove, membersAttr, nil),
					patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("999")),
				)
				return err
			},
			want: []string{"999"},
		},
		{
			name: "create",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := create(t, mocks, groupAttributes("999"))
				return err
			},
			want: []string{"999"},
		},
		{
			name: "replace",
			call: func(t *testing.T, mocks *groupTestMocks) error {
				_, err := replace(t, mocks, groupAttributes("999"))
				return err
			},
			want: []string{"999"},
		},
	}
	for _, tc := range rejected {
		t.Run(tc.name+" where every added member is unknown is rejected", func(t *testing.T) {
			mocks := newGroupTestMocks([]uint{1}, nil)
			mocks.unknownUsers = []uint{998, 999}
			mocks.unknownGroups = []uint{77}

			requireBadParams(t, tc.call(t, mocks), tc.want...)
			requireNoGroupWrite(t, mocks)
		})
	}

	t.Run("an existence lookup error fails the patch without writing", func(t *testing.T) {
		mocks := newGroupTestMocks([]uint{1}, nil)
		mocks.ds.ExistingScimUserIDsFunc = func(ctx context.Context, ids []uint) (map[uint]struct{}, error) {
			return nil, errors.New("db down")
		}
		_, err := patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")))

		require.ErrorContains(t, err, "db down")
		requireNoGroupWrite(t, mocks)
	})

	t.Run("replace of a missing group with only unknown members is a not found", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		mocks.unknownUsers = []uint{999}
		mocks.unknownGroups = []uint{1}
		_, err := replace(t, mocks, groupAttributes("999"))

		var scimErr scimerrors.ScimError
		require.ErrorAs(t, err, &scimErr)
		require.Equal(t, http.StatusNotFound, scimErr.Status)
		requireNoGroupWrite(t, mocks)
	})

	t.Run("replace fails when checking whether the group exists fails", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		mocks.unknownUsers = []uint{999}
		mocks.ds.ExistingScimGroupIDsFunc = func(ctx context.Context, ids []uint) (map[uint]struct{}, error) {
			return nil, errors.New("db down")
		}
		_, err := replace(t, mocks, groupAttributes("999"))

		require.ErrorContains(t, err, "db down")
		var scimErr scimerrors.ScimError
		require.NotErrorAs(t, err, &scimErr)
		requireNoGroupWrite(t, mocks)
	})

	t.Run("an empty member list is not rejected", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		_, err := create(t, mocks, groupAttributes())
		require.NoError(t, err)
		_, err = replace(t, mocks, groupAttributes())
		require.NoError(t, err)

		require.Empty(t, mocks.created.ScimUsers)
		require.Empty(t, mocks.replaced.ScimUsers)
	})

	t.Run("the existence check goes to the primary", func(t *testing.T) {
		mocks := newGroupTestMocks(nil, nil)
		var primary []bool
		mocks.ds.ExistingScimUserIDsFunc = func(ctx context.Context, ids []uint) (map[uint]struct{}, error) {
			primary = append(primary, ctxdb.IsPrimaryRequired(ctx))
			return existingExcept(ids, nil), nil
		}

		_, err := create(t, mocks, groupAttributes("4"))
		require.NoError(t, err)
		_, err = replace(t, mocks, groupAttributes("4"))
		require.NoError(t, err)
		_, err = patch(t, mocks, patchOp(t, scim.PatchOperationAdd, membersAttr, memberValues("4")))
		require.NoError(t, err)

		require.Equal(t, []bool{true, true, true}, primary)
	})
}

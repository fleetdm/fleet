package cron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDirectory struct {
	users     []*fleet.ScimUser
	groups    []*fleet.GoogleWorkspaceGroup
	usersErr  error
	groupsErr error
}

func (f *fakeDirectory) ListUsers(context.Context) ([]*fleet.ScimUser, error) {
	return f.users, f.usersErr
}

func (f *fakeDirectory) ListGroups(context.Context) ([]*fleet.GoogleWorkspaceGroup, error) {
	return f.groups, f.groupsErr
}

func fakeFactory(dir fleet.GoogleWorkspaceDirectory, err error) GoogleWorkspaceDirectoryFactory {
	return func(context.Context, *fleet.GoogleWorkspaceIntegration, *slog.Logger) (fleet.GoogleWorkspaceDirectory, error) {
		return dir, err
	}
}

func gwAppConfig() *fleet.AppConfig {
	ac := &fleet.AppConfig{}
	ac.Integrations.GoogleWorkspace = []*fleet.GoogleWorkspaceIntegration{{
		Domain:                "example.com",
		ImpersonatedUserEmail: "admin@example.com",
	}}
	return ac
}

// syncRecorder wires a mock.Store with recording stubs for the scim datastore
// methods the sync engine uses, seeded with the given existing state.
type syncRecorder struct {
	ds *mock.Store

	createdUsers  []*fleet.ScimUser
	replacedUsers []*fleet.ScimUser
	deletedUsers  []uint

	createdGroups  []*fleet.ScimGroup
	replacedGroups []*fleet.ScimGroup
	deletedGroups  []uint
	// groupWriteNames records the display name of every group create/replace
	// attempt, including ones rejected by groupNameTaken.
	groupWriteNames []string
	// groupNameTaken, when set, makes group writes with a matching display name
	// fail the way the datastore reports a display-name collision.
	groupNameTaken func(name string) bool

	lastRequest *fleet.ScimLastRequest
}

func newSyncRecorder(appConfig *fleet.AppConfig, existingUsers []fleet.ScimUser, existingGroups []fleet.ScimGroup) *syncRecorder {
	r := &syncRecorder{ds: new(mock.Store)}
	nextID := uint(1000)

	r.ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return appConfig, nil
	}
	r.ds.ListScimUsersFunc = func(_ context.Context, _ fleet.ScimUsersListOptions) ([]fleet.ScimUser, uint, error) {
		return existingUsers, uint(len(existingUsers)), nil
	}
	r.ds.ListScimGroupsFunc = func(_ context.Context, _ fleet.ScimGroupsListOptions) ([]fleet.ScimGroup, uint, error) {
		return existingGroups, uint(len(existingGroups)), nil
	}
	r.ds.CreateScimUserFunc = func(_ context.Context, user *fleet.ScimUser) (uint, error) {
		nextID++
		user.ID = nextID
		r.createdUsers = append(r.createdUsers, user)
		return nextID, nil
	}
	r.ds.ReplaceScimUserFunc = func(_ context.Context, user *fleet.ScimUser) ([]fleet.ActivityTypeResentCertificate, error) {
		r.replacedUsers = append(r.replacedUsers, user)
		return nil, nil
	}
	r.ds.DeleteScimUserFunc = func(_ context.Context, id uint) ([]fleet.ActivityTypeResentCertificate, error) {
		r.deletedUsers = append(r.deletedUsers, id)
		return nil, nil
	}
	r.ds.CreateScimGroupFunc = func(_ context.Context, group *fleet.ScimGroup) (uint, error) {
		r.groupWriteNames = append(r.groupWriteNames, group.DisplayName)
		if r.groupNameTaken != nil && r.groupNameTaken(group.DisplayName) {
			return 0, groupNameTakenError{name: group.DisplayName}
		}
		nextID++
		group.ID = nextID
		r.createdGroups = append(r.createdGroups, group)
		return nextID, nil
	}
	r.ds.ReplaceScimGroupFunc = func(_ context.Context, group *fleet.ScimGroup) error {
		r.groupWriteNames = append(r.groupWriteNames, group.DisplayName)
		if r.groupNameTaken != nil && r.groupNameTaken(group.DisplayName) {
			return groupNameTakenError{name: group.DisplayName}
		}
		r.replacedGroups = append(r.replacedGroups, group)
		return nil
	}
	r.ds.DeleteScimGroupFunc = func(_ context.Context, id uint) error {
		r.deletedGroups = append(r.deletedGroups, id)
		return nil
	}
	r.ds.UpdateScimLastRequestFunc = func(_ context.Context, lastRequest *fleet.ScimLastRequest) error {
		r.lastRequest = lastRequest
		return nil
	}
	return r
}

type groupNameTakenError struct{ name string }

func (e groupNameTakenError) Error() string {
	return fmt.Sprintf("ScimGroup %q already exists", e.name)
}
func (e groupNameTakenError) IsExists() bool { return true }

func scimUser(extID, userName, dept string, active bool) fleet.ScimUser {
	return fleet.ScimUser{
		ExternalID: new(extID),
		UserName:   userName,
		Department: new(dept),
		Active:     new(active),
		Emails:     []fleet.ScimUserEmail{{Email: userName, Primary: new(true)}},
	}
}

func gwUser(extID, userName, dept string, active bool) *fleet.ScimUser {
	u := scimUser(extID, userName, dept, active)
	return &u
}

func runSync(t *testing.T, r *syncRecorder, dir fleet.GoogleWorkspaceDirectory) error {
	t.Helper()
	return cronGoogleWorkspaceSync(t.Context(), r.ds, fakeFactory(dir, nil), slog.New(slog.DiscardHandler))
}

func TestGoogleWorkspaceSyncCreatesUsersAndGroups(t *testing.T) {
	r := newSyncRecorder(gwAppConfig(), nil, nil)
	dir := &fakeDirectory{
		users: []*fleet.ScimUser{
			gwUser("g1", "alice@example.com", "Engineering", true),
			gwUser("g2", "bob@example.com", "Sales", true),
		},
		groups: []*fleet.GoogleWorkspaceGroup{
			{ExternalID: "grp1", DisplayName: "Engineering", MemberExternalIDs: []string{"g1", "g2"}},
		},
	}

	require.NoError(t, runSync(t, r, dir))

	require.Len(t, r.createdUsers, 2)
	assert.Empty(t, r.replacedUsers)
	assert.Empty(t, r.deletedUsers)

	require.Len(t, r.createdGroups, 1)
	// Members resolved to the IDs returned by CreateScimUser.
	require.Len(t, r.createdGroups[0].ScimUsers, 2)
	assert.ElementsMatch(t, []uint{r.createdUsers[0].ID, r.createdUsers[1].ID}, r.createdGroups[0].ScimUsers)

	require.NotNil(t, r.lastRequest)
	assert.Equal(t, "success", r.lastRequest.Status)
}

func TestGoogleWorkspaceSyncUpdatesChangedUser(t *testing.T) {
	existing := scimUser("g1", "alice@example.com", "Engineering", true)
	existing.ID = 1
	r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{existing}, nil)

	dir := &fakeDirectory{users: []*fleet.ScimUser{
		gwUser("g1", "alice@example.com", "Marketing", true), // department changed
	}}

	require.NoError(t, runSync(t, r, dir))
	assert.Empty(t, r.createdUsers)
	require.Len(t, r.replacedUsers, 1)
	assert.Equal(t, uint(1), r.replacedUsers[0].ID)
	assert.Empty(t, r.deletedUsers)
}

func TestGoogleWorkspaceSyncIsIdempotent(t *testing.T) {
	existing := scimUser("g1", "alice@example.com", "Engineering", true)
	existing.ID = 1
	r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{existing}, nil)

	dir := &fakeDirectory{users: []*fleet.ScimUser{
		gwUser("g1", "alice@example.com", "Engineering", true), // identical
	}}

	require.NoError(t, runSync(t, r, dir))
	assert.Empty(t, r.createdUsers)
	assert.Empty(t, r.replacedUsers, "unchanged user must not be replaced")
	assert.Empty(t, r.deletedUsers)
	require.NotNil(t, r.lastRequest)
	assert.Equal(t, "success", r.lastRequest.Status)
}

func TestGoogleWorkspaceSyncDeletesRemovedUser(t *testing.T) {
	keep := scimUser("g1", "alice@example.com", "Engineering", true)
	keep.ID = 1
	gone := scimUser("g2", "bob@example.com", "Sales", true)
	gone.ID = 2
	r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{keep, gone}, nil)

	dir := &fakeDirectory{users: []*fleet.ScimUser{
		gwUser("g1", "alice@example.com", "Engineering", true),
	}}

	require.NoError(t, runSync(t, r, dir))
	assert.Empty(t, r.createdUsers)
	assert.Empty(t, r.replacedUsers)
	require.Equal(t, []uint{2}, r.deletedUsers)
}

func TestGoogleWorkspaceSyncEmptyPullDoesNotDelete(t *testing.T) {
	existing := scimUser("g1", "alice@example.com", "Engineering", true)
	existing.ID = 1
	existingGroup := fleet.ScimGroup{ID: 5, ExternalID: new("grp1"), DisplayName: "Engineering"}
	r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{existing}, []fleet.ScimGroup{existingGroup})

	dir := &fakeDirectory{} // empty pull

	require.NoError(t, runSync(t, r, dir))
	assert.Empty(t, r.deletedUsers, "empty pull must not wipe users")
	assert.Empty(t, r.deletedGroups, "empty pull must not wipe groups")
}

// TestGoogleWorkspaceSyncLimitErrorDoesNotDelete pins the invariant the directory
// sync limits rely on: a pull that hits a limit must abort, leaving existing IdP
// data alone. Reconciliation treats the pull as authoritative, so a limit that
// truncated instead of erroring would delete every record past it.
func TestGoogleWorkspaceSyncLimitErrorDoesNotDelete(t *testing.T) {
	existingUser := scimUser("g1", "alice@example.com", "Engineering", true)
	existingUser.ID = 1
	existingGroup := fleet.ScimGroup{ID: 5, ExternalID: new("grp1"), DisplayName: "Engineering"}

	for _, tc := range []struct {
		name string
		dir  *fakeDirectory
	}{
		{
			name: "users limit",
			dir:  &fakeDirectory{usersErr: errors.New("exceeded the limit of 500000 users; raise google_workspace.max_users to sync this domain")},
		},
		{
			name: "groups limit",
			dir: &fakeDirectory{
				users:     []*fleet.ScimUser{gwUser("g1", "alice@example.com", "Engineering", true)},
				groupsErr: errors.New("exceeded the limit of 100000 groups; raise google_workspace.max_groups to sync this domain"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{existingUser}, []fleet.ScimGroup{existingGroup})

			err := runSync(t, r, tc.dir)
			require.Error(t, err)
			assert.Empty(t, r.deletedUsers, "a limit error must not delete users")
			assert.Empty(t, r.deletedGroups, "a limit error must not delete groups")
			require.NotNil(t, r.lastRequest)
			assert.Equal(t, "error", r.lastRequest.Status)
			assert.Contains(t, r.lastRequest.Details, "raise google_workspace.max_")
		})
	}
}

func TestGoogleWorkspaceSyncDirectoryErrorRecordsStatus(t *testing.T) {
	r := newSyncRecorder(gwAppConfig(), nil, nil)
	dir := &fakeDirectory{usersErr: errors.New("delegation not authorized")}

	err := runSync(t, r, dir)
	require.Error(t, err)
	require.NotNil(t, r.lastRequest)
	assert.Equal(t, "error", r.lastRequest.Status)
	assert.Contains(t, r.lastRequest.Details, "delegation not authorized")
}

func TestGoogleWorkspaceSyncLongErrorIsTruncated(t *testing.T) {
	r := newSyncRecorder(gwAppConfig(), nil, nil)
	// Enforce the real scim_last_request.details VARCHAR(255) constraint so an
	// over-length, un-truncated message would fail to record (the original bug).
	r.ds.UpdateScimLastRequestFunc = func(_ context.Context, lastRequest *fleet.ScimLastRequest) error {
		if utf8.RuneCountInString(lastRequest.Details) > fleet.SCIMMaxFieldLength {
			return fmt.Errorf("details exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)
		}
		r.lastRequest = lastRequest
		return nil
	}
	longMsg := strings.Repeat("é", fleet.SCIMMaxFieldLength*2) // multi-byte to exercise rune-safe truncation
	dir := &fakeDirectory{usersErr: errors.New(longMsg)}

	err := runSync(t, r, dir)
	require.Error(t, err) // the sync error itself still propagates
	require.NotNil(t, r.lastRequest, "status must be recorded even for an over-length error")
	assert.Equal(t, "error", r.lastRequest.Status)
	assert.LessOrEqual(t, utf8.RuneCountInString(r.lastRequest.Details), fleet.SCIMMaxFieldLength)
	assert.True(t, utf8.ValidString(r.lastRequest.Details), "truncation must not split a multi-byte rune")
}

func TestGoogleWorkspaceSyncBestEffortContinuesPastFailedUser(t *testing.T) {
	r := newSyncRecorder(gwAppConfig(), nil, nil)
	// One user's creation fails (e.g. a unique-constraint conflict); the others
	// must still be ingested rather than the whole sync aborting.
	r.ds.CreateScimUserFunc = func(_ context.Context, user *fleet.ScimUser) (uint, error) {
		if user.UserName == "bad@example.com" {
			return 0, errors.New("ScimUser \"bad@example.com\" already exists")
		}
		r.createdUsers = append(r.createdUsers, user)
		return uint(1000 + len(r.createdUsers)), nil
	}
	dir := &fakeDirectory{users: []*fleet.ScimUser{
		gwUser("g1", "good1@example.com", "Eng", true),
		gwUser("g2", "bad@example.com", "Eng", true),
		gwUser("g3", "good2@example.com", "Eng", true),
	}}

	err := runSync(t, r, dir)
	require.Error(t, err) // partial failure is surfaced...
	assert.Contains(t, err.Error(), "partial sync")
	// ...but the two good users were still created despite the one failure.
	assert.Len(t, r.createdUsers, 2)
	require.NotNil(t, r.lastRequest)
	assert.Equal(t, "error", r.lastRequest.Status)
	assert.Contains(t, r.lastRequest.Details, "partial sync")
}

func TestGoogleWorkspaceSyncNotConfiguredNoOp(t *testing.T) {
	r := newSyncRecorder(&fleet.AppConfig{}, nil, nil) // no GoogleWorkspace integration
	dir := &fakeDirectory{users: []*fleet.ScimUser{gwUser("g1", "a@example.com", "Eng", true)}}

	require.NoError(t, runSync(t, r, dir))
	assert.Empty(t, r.createdUsers)
	assert.Nil(t, r.lastRequest, "no status recorded when not configured")
}

func TestGoogleWorkspaceSyncGroupMembershipUpdate(t *testing.T) {
	user := scimUser("g1", "alice@example.com", "Engineering", true)
	user.ID = 1
	group := fleet.ScimGroup{ID: 5, ExternalID: new("grp1"), DisplayName: "Engineering", ScimUsers: []uint{}}
	r := newSyncRecorder(gwAppConfig(), []fleet.ScimUser{user}, []fleet.ScimGroup{group})

	dir := &fakeDirectory{
		users: []*fleet.ScimUser{gwUser("g1", "alice@example.com", "Engineering", true)},
		groups: []*fleet.GoogleWorkspaceGroup{
			{ExternalID: "grp1", DisplayName: "Engineering", MemberExternalIDs: []string{"g1"}}, // alice added
		},
	}

	require.NoError(t, runSync(t, r, dir))
	require.Len(t, r.replacedGroups, 1)
	assert.Equal(t, []uint{1}, r.replacedGroups[0].ScimUsers)
}

func scimGroup(id uint, extID, name string) fleet.ScimGroup {
	return fleet.ScimGroup{ID: id, ExternalID: new(extID), DisplayName: name}
}

func gwGroup(extID, name string) *fleet.GoogleWorkspaceGroup {
	return &fleet.GoogleWorkspaceGroup{ExternalID: extID, DisplayName: name}
}

func reversed[T any](s []T) []T {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

func TestGoogleWorkspaceSyncGroupNameCollisions(t *testing.T) {
	t.Run("existing group keeps its name against a new case variant", func(t *testing.T) {
		pull := []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "infrastructure"), gwGroup("grpB", "Infrastructure")}
		for _, order := range [][]*fleet.GoogleWorkspaceGroup{pull, reversed(pull)} {
			r := newSyncRecorder(gwAppConfig(), nil, []fleet.ScimGroup{scimGroup(5, "grpB", "Infrastructure")})
			require.NoError(t, runSync(t, r, &fakeDirectory{groups: order}))

			assert.Equal(t, []string{"infrastructure (grpA)"}, r.groupWriteNames)
			assert.Empty(t, r.replacedGroups)
			assert.Equal(t, "success", r.lastRequest.Status)
		}
	})

	t.Run("existing group keeps its name against a new exact duplicate", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, []fleet.ScimGroup{scimGroup(5, "grpB", "X")})
		dir := &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "X"), gwGroup("grpB", "X")}}
		require.NoError(t, runSync(t, r, dir))

		assert.Equal(t, []string{"X (grpA)"}, r.groupWriteNames)
		assert.Empty(t, r.replacedGroups)
	})

	t.Run("new case variants in one pull", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, nil)
		dir := &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grp1", "Foo"), gwGroup("grp2", "foo")}}
		require.NoError(t, runSync(t, r, dir))

		require.Len(t, r.createdGroups, 2)
		assert.Equal(t, "Foo", r.createdGroups[0].DisplayName)
		assert.Equal(t, "foo (grp2)", r.createdGroups[1].DisplayName)
		assert.Equal(t, "success", r.lastRequest.Status)
	})

	t.Run("case-only rename of the holder", func(t *testing.T) {
		existing := []fleet.ScimGroup{scimGroup(5, "grpA", "Foo"), scimGroup(6, "grpB", "foo (grpB)")}
		r := newSyncRecorder(gwAppConfig(), nil, existing)
		dir := &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpB", "foo"), gwGroup("grpA", "FOO")}}
		require.NoError(t, runSync(t, r, dir))

		require.Len(t, r.replacedGroups, 1)
		assert.Equal(t, uint(5), r.replacedGroups[0].ID)
		assert.Equal(t, "FOO", r.replacedGroups[0].DisplayName)
		assert.Empty(t, r.createdGroups)
	})

	t.Run("steady state makes no writes in either order", func(t *testing.T) {
		pull := []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "Foo"), gwGroup("grpB", "foo")}
		for _, order := range [][]*fleet.GoogleWorkspaceGroup{pull, reversed(pull)} {
			existing := []fleet.ScimGroup{scimGroup(5, "grpA", "Foo"), scimGroup(6, "grpB", "foo (grpB)")}
			r := newSyncRecorder(gwAppConfig(), nil, existing)
			require.NoError(t, runSync(t, r, &fakeDirectory{groups: order}))
			assert.Empty(t, r.groupWriteNames)
		}
	})

	t.Run("create retries with the suffix when the name is taken", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, nil)
		r.groupNameTaken = func(name string) bool { return name == "Cafe" }
		require.NoError(t, runSync(t, r, &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "Cafe")}}))

		assert.Equal(t, []string{"Cafe", "Cafe (grpA)"}, r.groupWriteNames)
		require.Len(t, r.createdGroups, 1)
		assert.Equal(t, "Cafe (grpA)", r.createdGroups[0].DisplayName)
		assert.Equal(t, "success", r.lastRequest.Status)
	})

	t.Run("update retries with the suffix when the name is taken", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, []fleet.ScimGroup{scimGroup(5, "grpA", "Old")})
		r.groupNameTaken = func(name string) bool { return name == "Cafe" }
		require.NoError(t, runSync(t, r, &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "Cafe")}}))

		require.Len(t, r.replacedGroups, 1)
		assert.Equal(t, "Cafe (grpA)", r.replacedGroups[0].DisplayName)
		assert.Equal(t, "success", r.lastRequest.Status)
	})

	t.Run("update skips the retry write when already stored under the suffix", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, []fleet.ScimGroup{scimGroup(5, "grpA", "Cafe (grpA)")})
		r.groupNameTaken = func(name string) bool { return name == "Cafe" }
		require.NoError(t, runSync(t, r, &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "Cafe")}}))

		assert.Equal(t, []string{"Cafe"}, r.groupWriteNames)
		assert.Empty(t, r.replacedGroups)
		assert.Equal(t, "success", r.lastRequest.Status)
	})

	t.Run("failure after the retry is reported", func(t *testing.T) {
		r := newSyncRecorder(gwAppConfig(), nil, nil)
		r.groupNameTaken = func(string) bool { return true }
		err := runSync(t, r, &fakeDirectory{groups: []*fleet.GoogleWorkspaceGroup{gwGroup("grpA", "Cafe")}})
		require.ErrorContains(t, err, "partial sync")

		assert.Equal(t, []string{"Cafe", "Cafe (grpA)"}, r.groupWriteNames)
		assert.Empty(t, r.createdGroups)
		assert.Equal(t, "error", r.lastRequest.Status)
	})
}

// The mock datastore can't reproduce the display_name collation, so this runs
// the group sync against MySQL.
func TestGoogleWorkspaceSyncGroupNameCollisionsMySQL(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()

	_, err := ds.CreateScimGroup(ctx, &fleet.ScimGroup{ExternalID: new("grpB"), DisplayName: "Infrastructure"})
	require.NoError(t, err)

	pull := []*fleet.GoogleWorkspaceGroup{
		gwGroup("grpA", "infrastructure"),
		gwGroup("grpB", "Infrastructure"),
		gwGroup("grpC", "Cafe"),
		gwGroup("grpD", "Café"),
	}
	want := map[string]string{
		"grpA": "infrastructure (grpA)",
		"grpB": "Infrastructure",
		"grpC": "Cafe",
		"grpD": "Café (grpD)",
	}
	intg := gwAppConfig().Integrations.GoogleWorkspace[0]
	for i, order := range [][]*fleet.GoogleWorkspaceGroup{pull, reversed(pull), pull} {
		dir := &fakeDirectory{groups: order}
		require.NoError(t, syncGoogleWorkspaceDirectory(ctx, ds, fakeFactory(dir, nil), intg, slog.New(slog.DiscardHandler)), "run %d", i)

		groups, err := listAllScimGroups(ctx, ds)
		require.NoError(t, err)
		got := make(map[string]string, len(groups))
		for _, g := range groups {
			got[*g.ExternalID] = g.DisplayName
		}
		require.Equal(t, want, got, "run %d", i)
	}
}

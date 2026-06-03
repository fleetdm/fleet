package googleworkspace

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	directory "google.golang.org/api/admin/directory/v1"
)

func mockIntegration() *fleet.GoogleWorkspaceIntegration {
	return &fleet.GoogleWorkspaceIntegration{
		Domain:     "example.com",
		AdminEmail: "admin@example.com",
		CustomerID: fleet.DefaultGoogleWorkspaceCustomerID,
		ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
			fleet.GoogleCalendarEmail:      MockEmail,
			fleet.GoogleCalendarPrivateKey: "unused-by-mock",
		}},
	}
}

// fakeSCIMStore is an in-memory SCIM datastore for exercising Sync.
type fakeSCIMStore struct {
	*mock.Store
	mu        sync.Mutex
	users     map[uint]*fleet.ScimUser
	groups    map[uint]*fleet.ScimGroup
	nextUser  uint
	nextGroup uint
}

func newFakeSCIMStore() *fakeSCIMStore {
	s := &fakeSCIMStore{
		Store:  &mock.Store{},
		users:  map[uint]*fleet.ScimUser{},
		groups: map[uint]*fleet.ScimGroup{},
	}

	s.ScimUserByUserNameFunc = func(_ context.Context, userName string) (*fleet.ScimUser, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, u := range s.users {
			if u.UserName == userName {
				cp := *u
				return &cp, nil
			}
		}
		return nil, notFoundErr{}
	}
	s.CreateScimUserFunc = func(_ context.Context, user *fleet.ScimUser) (uint, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nextUser++
		user.ID = s.nextUser
		cp := *user
		s.users[user.ID] = &cp
		return user.ID, nil
	}
	s.ReplaceScimUserFunc = func(_ context.Context, user *fleet.ScimUser) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		cp := *user
		s.users[user.ID] = &cp
		return nil
	}
	s.DeleteScimUserFunc = func(_ context.Context, id uint) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.users, id)
		return nil
	}
	s.ListScimUsersFunc = func(_ context.Context, _ fleet.ScimUsersListOptions) ([]fleet.ScimUser, uint, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		var out []fleet.ScimUser
		for _, u := range s.users {
			out = append(out, *u)
		}
		return out, uint(len(out)), nil
	}

	s.ScimGroupByDisplayNameFunc = func(_ context.Context, displayName string) (*fleet.ScimGroup, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, g := range s.groups {
			if g.DisplayName == displayName {
				cp := *g
				return &cp, nil
			}
		}
		return nil, notFoundErr{}
	}
	s.CreateScimGroupFunc = func(_ context.Context, group *fleet.ScimGroup) (uint, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.nextGroup++
		group.ID = s.nextGroup
		cp := *group
		s.groups[group.ID] = &cp
		return group.ID, nil
	}
	s.ReplaceScimGroupFunc = func(_ context.Context, group *fleet.ScimGroup) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		cp := *group
		s.groups[group.ID] = &cp
		return nil
	}
	s.DeleteScimGroupFunc = func(_ context.Context, id uint) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.groups, id)
		return nil
	}
	s.ListScimGroupsFunc = func(_ context.Context, _ fleet.ScimGroupsListOptions) ([]fleet.ScimGroup, uint, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		var out []fleet.ScimGroup
		for _, g := range s.groups {
			out = append(out, *g)
		}
		return out, uint(len(out)), nil
	}
	return s
}

type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

func gwUser(id, email, given, family, dept string, suspended bool) *directory.User {
	u := &directory.User{
		Id:           id,
		PrimaryEmail: email,
		Suspended:    suspended,
		Name:         &directory.UserName{GivenName: given, FamilyName: family},
	}
	if dept != "" {
		u.Organizations = []map[string]any{{"department": dept, "primary": true}}
	}
	return u
}

func TestSyncMapsUsersGroupsAndMembers(t *testing.T) {
	t.Cleanup(ResetMockDirectory)
	ds := newFakeSCIMStore()
	logger := slog.New(slog.DiscardHandler)

	SetMockDirectory(
		[]*directory.User{
			gwUser("g1", "alice@example.com", "Alice", "Smith", "Engineering", false),
			gwUser("g2", "bob@example.com", "Bob", "Jones", "Sales", true),
			gwUser("g3", "", "No", "Email", "", false), // skipped: no primary email
		},
		[]*directory.Group{
			{Id: "grp1", Name: "Engineers", Email: "eng@example.com"},
		},
		map[string][]*directory.Member{
			"grp1": {
				{Id: "g1", Email: "alice@example.com", Type: "USER"},
				{Id: "nested", Email: "sub@example.com", Type: "GROUP"}, // skipped: nested
				{Id: "external", Email: "x@other.com", Type: "USER"},    // skipped: not synced
			},
		},
	)

	require.NoError(t, Sync(context.Background(), ds, mockIntegration(), logger))

	require.Len(t, ds.users, 2) // g3 skipped
	var alice *fleet.ScimUser
	for _, u := range ds.users {
		if u.UserName == "alice@example.com" {
			alice = u
		}
	}
	require.NotNil(t, alice)
	require.NotNil(t, alice.ExternalID)
	assert.Equal(t, "g1", *alice.ExternalID)
	require.NotNil(t, alice.GivenName)
	assert.Equal(t, "Alice", *alice.GivenName)
	require.NotNil(t, alice.Department)
	assert.Equal(t, "Engineering", *alice.Department)
	require.NotNil(t, alice.Active)
	assert.True(t, *alice.Active)
	require.Len(t, alice.Emails, 1)
	assert.Equal(t, "alice@example.com", alice.Emails[0].Email)

	// bob is suspended => inactive
	for _, u := range ds.users {
		if u.UserName == "bob@example.com" {
			require.NotNil(t, u.Active)
			assert.False(t, *u.Active)
		}
	}

	// group has only alice as a resolved member (nested + external skipped)
	require.Len(t, ds.groups, 1)
	for _, g := range ds.groups {
		assert.Equal(t, "Engineers", g.DisplayName)
		require.Len(t, g.ScimUsers, 1)
		assert.Equal(t, alice.ID, g.ScimUsers[0])
	}
}

func TestSyncReconcilesDeletes(t *testing.T) {
	t.Cleanup(ResetMockDirectory)
	ds := newFakeSCIMStore()
	logger := slog.New(slog.DiscardHandler)

	// Pre-seed a stale user/group that GW no longer knows about.
	ds.users[100] = &fleet.ScimUser{ID: 100, UserName: "stale@example.com", ExternalID: new("gone")}
	ds.nextUser = 100
	ds.groups[200] = &fleet.ScimGroup{ID: 200, DisplayName: "Stale", ExternalID: new("gone-grp")}
	ds.nextGroup = 200

	SetMockDirectory(
		[]*directory.User{gwUser("g1", "alice@example.com", "Alice", "Smith", "Engineering", false)},
		[]*directory.Group{{Id: "grp1", Name: "Engineers"}},
		map[string][]*directory.Member{"grp1": {{Id: "g1", Type: "USER"}}},
	)

	require.NoError(t, Sync(context.Background(), ds, mockIntegration(), logger))

	_, staleUserExists := ds.users[100]
	assert.False(t, staleUserExists, "stale user should be deleted")
	_, staleGroupExists := ds.groups[200]
	assert.False(t, staleGroupExists, "stale group should be deleted")
	assert.True(t, ds.DeleteScimUserFuncInvoked)
	assert.True(t, ds.DeleteScimGroupFuncInvoked)
}

func TestSyncDoesNotDeleteOnFetchError(t *testing.T) {
	t.Cleanup(ResetMockDirectory)
	ds := newFakeSCIMStore()
	logger := slog.New(slog.DiscardHandler)

	ds.users[100] = &fleet.ScimUser{ID: 100, UserName: "stale@example.com", ExternalID: new("gone")}

	SetMockDirectory(nil, nil, nil)
	SetMockDirectoryErrors(nil, errors.New("transient list users failure"), nil)

	err := Sync(context.Background(), ds, mockIntegration(), logger)
	require.Error(t, err)

	_, staleUserExists := ds.users[100]
	assert.True(t, staleUserExists, "must not delete users when the directory fetch failed")
	assert.False(t, ds.DeleteScimUserFuncInvoked)
}

func TestPrimaryDepartment(t *testing.T) {
	assert.Empty(t, primaryDepartment(nil))
	assert.Equal(t, "Eng", primaryDepartment([]map[string]any{{"department": "Eng", "primary": true}}))
	// prefers primary over the first listed
	assert.Equal(t, "Sales", primaryDepartment([]map[string]any{
		{"department": "Eng", "primary": false},
		{"department": "Sales", "primary": true},
	}))
	// falls back to first with a department when none primary
	assert.Equal(t, "Eng", primaryDepartment([]map[string]any{
		{"department": "Eng", "primary": false},
	}))
}

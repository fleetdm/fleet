package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures
var (
	johnUser = &activity.User{ID: 100, Name: "John Doe", Email: "john@example.com", Gravatar: "gravatar1", APIOnly: false}
	janeUser = &activity.User{ID: 200, Name: "Jane Smith", Email: "jane@example.com", Gravatar: "gravatar2", APIOnly: true}
)

// Mock implementations

type mockAuthorizer struct {
	authErr error
}

func (m *mockAuthorizer) Authorize(ctx context.Context, subject platform_authz.AuthzTyper, action platform_authz.Action) error {
	return m.authErr
}

type mockDatastore struct {
	activities []*api.Activity
	meta       *api.PaginationMetadata
	err        error
	lastOpt    types.ListOptions
}

func (m *mockDatastore) ListActivities(ctx context.Context, opt types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	m.lastOpt = opt
	return m.activities, m.meta, m.err
}

type mockUserProvider struct {
	users         []*activity.User
	listUsersErr  error
	searchUserIDs []uint
	searchErr     error
	lastIDs       []uint
	lastQuery     string
}

func (m *mockUserProvider) UsersByIDs(ctx context.Context, ids []uint) ([]*activity.User, error) {
	m.lastIDs = ids
	return m.users, m.listUsersErr
}

func (m *mockUserProvider) FindUserIDs(ctx context.Context, query string) ([]uint, error) {
	m.lastQuery = query
	return m.searchUserIDs, m.searchErr
}

// testSetup holds test dependencies with pre-configured mocks
type testSetup struct {
	svc   *Service
	authz *mockAuthorizer
	ds    *mockDatastore
	users *mockUserProvider
}

// setupTest creates a service with default working mocks.
// Use functional options to customize mock behavior.
func setupTest(opts ...func(*testSetup)) *testSetup {
	ts := &testSetup{
		authz: &mockAuthorizer{},
		ds:    &mockDatastore{},
		users: &mockUserProvider{},
	}
	for _, opt := range opts {
		opt(ts)
	}
	ts.svc = NewService(ts.authz, ts.ds, ts.users, log.NewNopLogger())
	return ts
}

// Setup options

func withAuthError(err error) func(*testSetup) {
	return func(ts *testSetup) { ts.authz.authErr = err }
}

func withActivities(activities []*api.Activity) func(*testSetup) {
	return func(ts *testSetup) { ts.ds.activities = activities }
}

func withMeta(meta *api.PaginationMetadata) func(*testSetup) {
	return func(ts *testSetup) { ts.ds.meta = meta }
}

func withDatastoreError(err error) func(*testSetup) {
	return func(ts *testSetup) { ts.ds.err = err }
}

func withUsers(users []*activity.User) func(*testSetup) {
	return func(ts *testSetup) { ts.users.users = users }
}

func withUsersByIDsError(err error) func(*testSetup) {
	return func(ts *testSetup) { ts.users.listUsersErr = err }
}

func withSearchUserIDs(ids []uint) func(*testSetup) {
	return func(ts *testSetup) { ts.users.searchUserIDs = ids }
}

func withSearchError(err error) func(*testSetup) {
	return func(ts *testSetup) { ts.users.searchErr = err }
}

func TestListActivitiesBasic(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	details := json.RawMessage(`{"key": "value"}`)

	ts := setupTest(
		withActivities([]*api.Activity{
			{ID: 1, Type: "test_activity", Details: &details},
			{ID: 2, Type: "another_activity"},
		}),
		withMeta(&api.PaginationMetadata{HasNextResults: true}),
	)

	activities, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{
		PerPage: 10,
		Page:    0,
	})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.NotNil(t, meta)
	assert.True(t, meta.HasNextResults)

	// Verify options were passed correctly
	assert.Equal(t, uint(10), ts.ds.lastOpt.PerPage)
	assert.Equal(t, uint(0), ts.ds.lastOpt.Page)
}

func TestListActivitiesWithUserEnrichment(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	ts := setupTest(
		withActivities([]*api.Activity{
			{ID: 1, Type: "test_activity", ActorID: ptr.Uint(johnUser.ID)},
			{ID: 2, Type: "another_activity", ActorID: ptr.Uint(janeUser.ID)},
			{ID: 3, Type: "system_activity"}, // No actor
		}),
		withUsers([]*activity.User{johnUser, janeUser}),
	)

	activities, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	assert.Nil(t, meta)

	// Verify user IDs were passed to UsersByIDs
	assert.ElementsMatch(t, []uint{johnUser.ID, janeUser.ID}, ts.users.lastIDs)

	// Verify activity 1 was enriched with John's data
	assert.Equal(t, johnUser.Email, *activities[0].ActorEmail)
	assert.Equal(t, johnUser.Gravatar, *activities[0].ActorGravatar)
	assert.Equal(t, johnUser.Name, *activities[0].ActorFullName)
	assert.Equal(t, johnUser.APIOnly, *activities[0].ActorAPIOnly)

	// Verify activity 2 was enriched with Jane's data
	assert.Equal(t, janeUser.Email, *activities[1].ActorEmail)
	assert.Equal(t, janeUser.Gravatar, *activities[1].ActorGravatar)
	assert.Equal(t, janeUser.Name, *activities[1].ActorFullName)
	assert.Equal(t, janeUser.APIOnly, *activities[1].ActorAPIOnly)

	// Verify activity 3 has no user enrichment (no actor)
	assert.Nil(t, activities[2].ActorEmail)
	assert.Nil(t, activities[2].ActorGravatar)
	assert.Nil(t, activities[2].ActorFullName)
	assert.Nil(t, activities[2].ActorAPIOnly)
}

// TestListActivitiesWithMatchQuery verifies that user search queries are properly
// translated into user ID filters for the datastore.
//
// When user searches for "john", the service:
// 1. Calls FindUserIDs("john") which returns matching user IDs (100, 200, 300)
// 2. Passes ALL matching user IDs to the datastore query
// 3. Datastore filters activities to those by any of the matching users
//
// Note: The search may return users who have no activities;
// the datastore simply won't find any activities for those user IDs.
func TestListActivitiesWithMatchQuery(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	ts := setupTest(
		withActivities([]*api.Activity{
			{ID: 1, Type: "test_activity", ActorID: ptr.Uint(johnUser.ID)},
		}),
		withSearchUserIDs([]uint{100, 200, 300}), // 3 users match "john", but only user 100 has activities
		withUsers([]*activity.User{johnUser}),
	)

	activities, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{
		MatchQuery: "john",
	})
	require.NoError(t, err)

	// Verify the activity was returned and enriched
	require.Len(t, activities, 1)
	assert.Equal(t, uint(1), activities[0].ID)
	assert.Equal(t, johnUser.Email, *activities[0].ActorEmail)
	assert.Nil(t, meta, "metadata not configured in test setup")

	// Verify FindUserIDs was called with the query
	assert.Equal(t, "john", ts.users.lastQuery)

	// Verify all matching user IDs were passed to datastore (even those without activities)
	assert.ElementsMatch(t, []uint{100, 200, 300}, ts.ds.lastOpt.MatchingUserIDs)
}

// TestListActivitiesWithMatchQueryNoMatchingUsers verifies behavior when the user
// search returns no matching users. The empty user ID list should still be passed
// to the datastore, which will then return no activities (since no users matched).
func TestListActivitiesWithMatchQueryNoMatchingUsers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	ts := setupTest(
		withActivities([]*api.Activity{}), // Datastore returns nothing when filtering by empty user list
		withSearchUserIDs([]uint{}),       // No users match "nonexistent"
	)

	activities, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{
		MatchQuery: "nonexistent",
	})
	require.NoError(t, err)
	assert.Empty(t, activities)
	assert.Nil(t, meta)

	// Verify FindUserIDs was called
	assert.Equal(t, "nonexistent", ts.users.lastQuery)

	// Empty slice should be passed to datastore (not nil)
	assert.NotNil(t, ts.ds.lastOpt.MatchingUserIDs)
	assert.Empty(t, ts.ds.lastOpt.MatchingUserIDs)
}

func TestListActivitiesWithDuplicateUserIDs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Multiple activities by the same user
	ts := setupTest(
		withActivities([]*api.Activity{
			{ID: 1, Type: "created_policy", ActorID: ptr.Uint(johnUser.ID)},
			{ID: 2, Type: "deleted_policy", ActorID: ptr.Uint(johnUser.ID)},
			{ID: 3, Type: "edited_policy", ActorID: ptr.Uint(johnUser.ID)},
		}),
		withUsers([]*activity.User{johnUser}),
	)

	activities, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	assert.Nil(t, meta)

	// UsersByIDs should only be called with unique IDs (deduplication)
	assert.Equal(t, []uint{johnUser.ID}, ts.users.lastIDs)

	// All activities should be enriched with John's data
	for i, a := range activities {
		require.NotNil(t, a.ActorEmail, "activity %d should have ActorEmail", i)
		assert.Equal(t, johnUser.Email, *a.ActorEmail)
		assert.Equal(t, johnUser.Name, *a.ActorFullName)
	}
}

func TestListActivitiesCursorPaginationMetadata(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	ts := setupTest(
		withActivities([]*api.Activity{{ID: 1}}),
		withMeta(&api.PaginationMetadata{HasNextResults: true}),
	)

	// Without cursor (After="") - should include metadata
	_, meta, err := ts.svc.ListActivities(ctx, api.ListOptions{
		PerPage: 10,
	})
	require.NoError(t, err)
	assert.True(t, ts.ds.lastOpt.IncludeMetadata, "should include metadata when After is empty")
	assert.NotNil(t, meta)

	// With cursor (After="123") - should not include metadata
	_, _, err = ts.svc.ListActivities(ctx, api.ListOptions{
		PerPage: 10,
		After:   "123",
	})
	require.NoError(t, err)
	assert.False(t, ts.ds.lastOpt.IncludeMetadata, "should not include metadata when After is set")
}

// TestListActivitiesErrors tests hard-fail error scenarios (authorization denied, datastore errors).
func TestListActivitiesErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		opts        []func(*testSetup)
		listOpts    api.ListOptions
		errContains string
	}{
		{
			name:        "authorization denied",
			opts:        []func(*testSetup){withAuthError(errors.New("forbidden"))},
			errContains: "forbidden",
		},
		{
			name:        "datastore error",
			opts:        []func(*testSetup){withDatastoreError(errors.New("database error"))},
			errContains: "database error",
		},
		{
			name: "user enrichment error",
			opts: []func(*testSetup){
				withActivities([]*api.Activity{
					{ID: 1, Type: "test_activity", ActorID: ptr.Uint(100)},
				}),
				withUsersByIDsError(errors.New("user service error")),
			},
			errContains: "user service error",
		},
		{
			name: "search users error",
			opts: []func(*testSetup){
				withActivities([]*api.Activity{
					{ID: 1, Type: "test_activity"},
				}),
				withSearchError(errors.New("search error")),
			},
			listOpts:    api.ListOptions{MatchQuery: "john"},
			errContains: "search error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ts := setupTest(tc.opts...)

			activities, meta, err := ts.svc.ListActivities(ctx, tc.listOpts)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errContains)
			assert.Nil(t, activities)
			assert.Nil(t, meta)
		})
	}
}

// forbiddenError is a mock error that implements platform_authz.Forbidden for testing.
type forbiddenError struct {
	msg string
}

func (e forbiddenError) Error() string { return e.msg }
func (e forbiddenError) Forbidden()    {}

// TestListActivitiesGracefulDegradation verifies that forbidden errors during
// user enrichment and search are handled gracefully (logged but not returned to caller).
// Non-forbidden errors are tested in TestListActivitiesErrors.
func TestListActivitiesGracefulDegradation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                  string
		opts                  []func(*testSetup)
		listOpts              api.ListOptions
		expectedCount         int
		expectUserEnrichment  bool
		expectMatchingUserIDs bool
	}{
		{
			name: "forbidden error on user enrichment returns activities without enrichment",
			opts: []func(*testSetup){
				withActivities([]*api.Activity{
					{ID: 1, Type: "test_activity", ActorID: ptr.Uint(100)},
				}),
				withUsersByIDsError(forbiddenError{msg: "forbidden"}),
			},
			expectedCount:        1,
			expectUserEnrichment: false,
		},
		{
			name: "forbidden error on user search returns activities without user filter",
			opts: []func(*testSetup){
				withActivities([]*api.Activity{
					{ID: 1, Type: "test_activity"},
					{ID: 2, Type: "another_activity"},
				}),
				withSearchError(forbiddenError{msg: "forbidden"}),
			},
			listOpts:              api.ListOptions{MatchQuery: "john"},
			expectedCount:         2,
			expectMatchingUserIDs: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ts := setupTest(tc.opts...)

			activities, meta, err := ts.svc.ListActivities(ctx, tc.listOpts)

			require.NoError(t, err)
			require.Len(t, activities, tc.expectedCount)
			assert.Nil(t, meta)

			if !tc.expectUserEnrichment && len(activities) > 0 && activities[0].ActorID != nil {
				assert.Nil(t, activities[0].ActorEmail)
				assert.Nil(t, activities[0].ActorFullName)
			}

			if tc.listOpts.MatchQuery != "" && !tc.expectMatchingUserIDs {
				assert.Nil(t, ts.ds.lastOpt.MatchingUserIDs)
			}
		})
	}
}

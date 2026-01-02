package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations

type mockAuthorizer struct {
	authErr error
}

func (m *mockAuthorizer) Authorize(ctx context.Context, subject platform_authz.AuthzTyper, action string) error {
	return m.authErr
}

type mockDatastore struct {
	activities []*types.Activity
	meta       *types.PaginationMetadata
	err        error
	lastOpt    types.ListOptions
}

func (m *mockDatastore) ListActivities(ctx context.Context, opt types.ListOptions) ([]*types.Activity, *types.PaginationMetadata, error) {
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

func (m *mockUserProvider) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	m.lastIDs = ids
	return m.users, m.listUsersErr
}

func (m *mockUserProvider) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	m.lastQuery = query
	return m.searchUserIDs, m.searchErr
}

func TestListActivities(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"AuthorizationDenied", testListActivitiesAuthorizationDenied},
		{"Basic", testListActivitiesBasic},
		{"WithUserEnrichment", testListActivitiesWithUserEnrichment},
		{"WithMatchQuery", testListActivitiesWithMatchQuery},
		{"DatastoreError", testListActivitiesDatastoreError},
		{"UserEnrichmentError", testListActivitiesUserEnrichmentError},
		{"SearchUsersError", testListActivitiesSearchUsersError},
	}
	for _, c := range cases {
		t.Run(c.name, c.fn)
	}
}

func testListActivitiesAuthorizationDenied(t *testing.T) {
	ctx := t.Context()

	svc := NewService(
		&mockAuthorizer{authErr: errors.New("forbidden")},
		&mockDatastore{},
		&mockUserProvider{},
		log.NewNopLogger(),
	)

	activities, meta, err := svc.ListActivities(ctx, types.ListOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	assert.Nil(t, activities)
	assert.Nil(t, meta)
}

func testListActivitiesBasic(t *testing.T) {
	ctx := t.Context()

	details := json.RawMessage(`{"key": "value"}`)
	mockDS := &mockDatastore{
		activities: []*types.Activity{
			{ID: 1, Type: "test_activity", Details: &details},
			{ID: 2, Type: "another_activity"},
		},
		meta: &types.PaginationMetadata{HasNextResults: true},
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		&mockUserProvider{},
		log.NewNopLogger(),
	)

	activities, meta, err := svc.ListActivities(ctx, types.ListOptions{
		PerPage: 10,
		Page:    0,
	})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.NotNil(t, meta)
	assert.True(t, meta.HasNextResults)

	// Verify options were passed correctly
	assert.Equal(t, uint(10), mockDS.lastOpt.PerPage)
	assert.Equal(t, uint(0), mockDS.lastOpt.Page)
}

func testListActivitiesWithUserEnrichment(t *testing.T) {
	ctx := t.Context()

	userIDOne := uint(100)
	userIDTwo := uint(200)

	mockDS := &mockDatastore{
		activities: []*types.Activity{
			{ID: 1, Type: "test_activity", ActorID: &userIDOne},
			{ID: 2, Type: "another_activity", ActorID: &userIDTwo},
			{ID: 3, Type: "system_activity"}, // No actor
		},
	}

	mockUsers := &mockUserProvider{
		users: []*activity.User{
			{ID: 100, Name: "John Doe", Email: "john@example.com", Gravatar: "gravatar1", APIOnly: false},
			{ID: 200, Name: "Jane Smith", Email: "jane@example.com", Gravatar: "gravatar2", APIOnly: true},
		},
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		mockUsers,
		log.NewNopLogger(),
	)

	activities, _, err := svc.ListActivities(ctx, types.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 3)

	// Verify user IDs were passed to ListUsers
	assert.ElementsMatch(t, []uint{100, 200}, mockUsers.lastIDs)

	// Verify activity 1 was enriched with user 100's data
	assert.Equal(t, "john@example.com", *activities[0].ActorEmail)
	assert.Equal(t, "gravatar1", *activities[0].ActorGravatar)
	assert.Equal(t, "John Doe", *activities[0].ActorFullName)
	assert.Equal(t, false, *activities[0].ActorAPIOnly)

	// Verify activity 2 was enriched with user 200's data
	assert.Equal(t, "jane@example.com", *activities[1].ActorEmail)
	assert.Equal(t, "gravatar2", *activities[1].ActorGravatar)
	assert.Equal(t, "Jane Smith", *activities[1].ActorFullName)
	assert.Equal(t, true, *activities[1].ActorAPIOnly)

	// Verify activity 3 has no user enrichment (no actor)
	assert.Nil(t, activities[2].ActorEmail)
	assert.Nil(t, activities[2].ActorGravatar)
	assert.Nil(t, activities[2].ActorFullName)
	assert.Nil(t, activities[2].ActorAPIOnly)
}

func testListActivitiesWithMatchQuery(t *testing.T) {
	ctx := t.Context()

	mockDS := &mockDatastore{
		activities: []*types.Activity{
			{ID: 1, Type: "test_activity", ActorID: ptr.Uint(100)},
		},
	}

	mockUsers := &mockUserProvider{
		searchUserIDs: []uint{100, 200, 300},
		users: []*activity.User{
			{ID: 100, Name: "John", Email: "john@example.com"},
		},
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		mockUsers,
		log.NewNopLogger(),
	)

	_, _, err := svc.ListActivities(ctx, types.ListOptions{
		MatchQuery: "john",
	})
	require.NoError(t, err)

	// Verify SearchUsers was called with the query
	assert.Equal(t, "john", mockUsers.lastQuery)

	// Verify MatchingUserIDs were set on the options passed to datastore
	assert.ElementsMatch(t, []uint{100, 200, 300}, mockDS.lastOpt.MatchingUserIDs)
}

func testListActivitiesDatastoreError(t *testing.T) {
	ctx := t.Context()

	mockDS := &mockDatastore{
		err: errors.New("database error"),
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		&mockUserProvider{},
		log.NewNopLogger(),
	)

	activities, meta, err := svc.ListActivities(ctx, types.ListOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
	assert.Nil(t, activities)
	assert.Nil(t, meta)
}

func testListActivitiesUserEnrichmentError(t *testing.T) {
	ctx := t.Context()

	userID := uint(100)
	mockDS := &mockDatastore{
		activities: []*types.Activity{
			{ID: 1, Type: "test_activity", ActorID: &userID},
		},
	}

	mockUsers := &mockUserProvider{
		listUsersErr: errors.New("user service error"),
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		mockUsers,
		log.NewNopLogger(),
	)

	// User enrichment errors are logged but don't fail the request
	activities, _, err := svc.ListActivities(ctx, types.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// User data not enriched due to error
	assert.Nil(t, activities[0].ActorEmail)
	assert.Nil(t, activities[0].ActorGravatar)
}

func testListActivitiesSearchUsersError(t *testing.T) {
	ctx := t.Context()

	mockDS := &mockDatastore{
		activities: []*types.Activity{
			{ID: 1, Type: "test_activity"},
		},
	}

	mockUsers := &mockUserProvider{
		searchErr: errors.New("search error"),
	}

	svc := NewService(
		&mockAuthorizer{},
		mockDS,
		mockUsers,
		log.NewNopLogger(),
	)

	// Search errors are logged but don't fail the request
	activities, _, err := svc.ListActivities(ctx, types.ListOptions{
		MatchQuery: "john",
	})
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// MatchingUserIDs should be nil (search failed)
	assert.Nil(t, mockDS.lastOpt.MatchingUserIDs)
}

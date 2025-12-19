package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/stretchr/testify/require"
)

// mockStore implements Datastore for testing.
type mockStore struct {
	pingErr error
}

func (m *mockStore) Ping(_ context.Context) error {
	return m.pingErr
}

func (m *mockStore) ListActivities(_ context.Context, _ activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	return nil, nil, nil
}

func (m *mockStore) ListUsers(_ context.Context, _ activity.UserListOptions) ([]*activity.User, error) {
	return nil, nil
}

// mockAuthorizer implements activity.Authorizer for testing.
type mockAuthorizer struct{}

func (m *mockAuthorizer) SkipAuthorization(_ context.Context) {}

func (m *mockAuthorizer) Authorize(_ context.Context, _, _ any) error {
	return nil
}

func TestNewService(t *testing.T) {
	t.Parallel()

	authz := &mockAuthorizer{}
	store := &mockStore{}
	svc := NewService(authz, store)

	require.NotNil(t, svc)
	require.Equal(t, store, svc.store)
	require.Equal(t, authz, svc.authz)
}

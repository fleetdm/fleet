package service

import (
	"context"
	"testing"
	"time"

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

func (m *mockStore) NewActivity(_ context.Context, _ *activity.Actor, _ activity.Details, _ []byte, _ time.Time) error {
	return nil
}

func (m *mockStore) ListActivities(_ context.Context, _ activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	return nil, nil, nil
}

func (m *mockStore) ListHostPastActivities(_ context.Context, _ uint, _ activity.ListOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	return nil, nil, nil
}

func (m *mockStore) MarkActivitiesAsStreamed(_ context.Context, _ []uint) error {
	return nil
}

func (m *mockStore) CleanupActivitiesAndAssociatedData(_ context.Context, _ int, _ int) error {
	return nil
}

func TestNewService(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	svc, err := NewService(store)

	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, store, svc.store)
	require.NotNil(t, svc.authz)
}

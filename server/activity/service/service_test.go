package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockStore implements Datastore for testing.
type mockStore struct {
	pingErr error
}

func (m *mockStore) Ping(_ context.Context) error {
	return m.pingErr
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

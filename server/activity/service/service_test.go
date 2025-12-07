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

func TestNewService(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	svc, err := NewService(store)

	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, store, svc.store)
	require.NotNil(t, svc.authz)
}

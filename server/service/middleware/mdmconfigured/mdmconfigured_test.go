package mdmconfigured

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
	fleet.Service

	mdmConfigured atomic.Bool
}

func (m *mockService) VerifyMDMAppleConfigured(ctx context.Context) error {
	if !m.mdmConfigured.Load() {
		return fleet.ErrMDMNotConfigured
	}
	return nil
}

func TestMDMConfigured(t *testing.T) {
	svc := mockService{}
	svc.mdmConfigured.Store(true)
	mw := NewAppleMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.Verify()(next)
	_, err := f(context.Background(), struct{}{})
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestMDMNotConfigured(t *testing.T) {
	svc := mockService{}
	svc.mdmConfigured.Store(false)
	mw := NewAppleMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.Verify()(next)
	_, err := f(context.Background(), struct{}{})
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.False(t, nextCalled)
}

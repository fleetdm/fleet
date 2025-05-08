package mdmconfigured

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	mock.Mock
	fleet.Service

	mdmConfigured   atomic.Bool
	msMdmConfigured atomic.Bool
}

func (m *mockService) VerifyMDMAppleConfigured(ctx context.Context) error {
	if !m.mdmConfigured.Load() {
		return fleet.ErrMDMNotConfigured
	}
	return nil
}

func (m *mockService) VerifyMDMWindowsConfigured(ctx context.Context) error {
	if !m.msMdmConfigured.Load() {
		return fleet.ErrWindowsMDMNotConfigured
	}
	return nil
}

func (m *mockService) VerifyMDMAppleOrWindowsConfigured(ctx context.Context) error {
	if !m.mdmConfigured.Load() && !m.msMdmConfigured.Load() {
		return fleet.ErrMDMNotConfigured
	}
	return nil
}

func TestMDMConfigured(t *testing.T) {
	svc := mockService{}
	svc.mdmConfigured.Store(true)
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyAppleMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestMDMNotConfigured(t *testing.T) {
	svc := mockService{}
	svc.mdmConfigured.Store(false)
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyAppleMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.False(t, nextCalled)
}

func TestWindowsMDMConfigured(t *testing.T) {
	svc := mockService{}
	svc.msMdmConfigured.Store(true)
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyWindowsMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.NoError(t, err)
	require.True(t, nextCalled)
}

func TestWindowsMDMNotConfigured(t *testing.T) {
	svc := mockService{}
	svc.msMdmConfigured.Store(false)
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyWindowsMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.ErrorIs(t, err, fleet.ErrWindowsMDMNotConfigured)
	require.False(t, nextCalled)
}

func TestAppleOrWindowsMDMConfigured(t *testing.T) {
	svc := mockService{}
	mw := NewMDMConfigMiddleware(&svc)

	cases := []struct {
		apple   bool
		windows bool
	}{
		{true, false},
		{false, true},
		{true, true},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("apple:%t;windows:%t", c.apple, c.windows), func(t *testing.T) {
			svc.mdmConfigured.Store(c.apple)
			svc.msMdmConfigured.Store(c.windows)
			nextCalled := false
			next := func(ctx context.Context, req interface{}) (interface{}, error) {
				nextCalled = true
				return struct{}{}, nil
			}

			f := mw.VerifyAppleOrWindowsMDM()(next)
			_, err := f(context.Background(), struct{}{})
			require.NoError(t, err)
			require.True(t, nextCalled)
		})
	}
}

func TestAppleOrWindowsMDMNotConfigured(t *testing.T) {
	svc := mockService{}
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyAppleOrWindowsMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.False(t, nextCalled)
}

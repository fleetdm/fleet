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

// mockService implements the minimal subset of fleet.Service required by the
// middleware tests. Each Verify* method gates execution via simple atomic flags
// so tests can toggle the configured MDM platforms.
type mockService struct {
	mock.Mock
	fleet.Service

	mdmConfigured     atomic.Bool
	msMdmConfigured   atomic.Bool
	androidConfigured atomic.Bool
}

// VerifyMDMAppleConfigured marks whether Apple MDM is enabled for the test.
func (m *mockService) VerifyMDMAppleConfigured(ctx context.Context) error {
	if !m.mdmConfigured.Load() {
		return fleet.ErrMDMNotConfigured
	}
	return nil
}

// VerifyMDMWindowsConfigured marks whether Windows MDM is enabled for the test.
func (m *mockService) VerifyMDMWindowsConfigured(ctx context.Context) error {
	if !m.msMdmConfigured.Load() {
		return fleet.ErrWindowsMDMNotConfigured
	}
	return nil
}

// VerifyAnyMDMConfigured is the mock implementation that mirrors the production
// VerifyAnyMDMConfigured service method, adding Android to the Apple/Windows check.
func (m *mockService) VerifyAnyMDMConfigured(ctx context.Context) error {
	if !m.mdmConfigured.Load() && !m.msMdmConfigured.Load() && !m.androidConfigured.Load() {
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

// TestAnyMDMConfigured exercises the new middleware that recognizes Apple,
// Windows, or Android MDM individually or in combination.
func TestAnyMDMConfigured(t *testing.T) {
	svc := mockService{}
	mw := NewMDMConfigMiddleware(&svc)

	cases := []struct {
		apple   bool
		windows bool
		android bool
	}{
		{apple: true},
		{windows: true},
		{android: true},
		{apple: true, windows: true},
		{apple: true, android: true},
		{windows: true, android: true},
		{apple: true, windows: true, android: true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("apple:%t;windows:%t;android:%t", c.apple, c.windows, c.android), func(t *testing.T) {
			svc.mdmConfigured.Store(c.apple)
			svc.msMdmConfigured.Store(c.windows)
			svc.androidConfigured.Store(c.android)

			nextCalled := false
			next := func(ctx context.Context, req interface{}) (interface{}, error) {
				nextCalled = true
				return struct{}{}, nil
			}

			f := mw.VerifyAnyMDM()(next)
			_, err := f(context.Background(), struct{}{})
			require.NoError(t, err)
			require.True(t, nextCalled)
		})
	}
}

// TestAnyMDMNotConfigured ensures the new middleware continues to return
// ErrMDMNotConfigured when no platform has MDM enabled.
func TestAnyMDMNotConfigured(t *testing.T) {
	svc := mockService{}
	mw := NewMDMConfigMiddleware(&svc)

	nextCalled := false
	next := func(ctx context.Context, req interface{}) (interface{}, error) {
		nextCalled = true
		return struct{}{}, nil
	}

	f := mw.VerifyAnyMDM()(next)
	_, err := f(context.Background(), struct{}{})
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.False(t, nextCalled)
}

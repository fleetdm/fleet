package android

import (
	"context"
	"testing"

	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubService implements android.Service with only GetZeroTouchConfiguration.
// All other methods panic if called — that's fine because the ee wrapper
// delegates them via embedding and we only test the overridden method here.
type stubService struct {
	android.Service // embedded to satisfy the interface
	called          bool
}

func (s *stubService) GetZeroTouchConfiguration(ctx context.Context, teamID *uint) (*android.ZeroTouchConfigurationResponse, error) {
	s.called = true
	return &android.ZeroTouchConfigurationResponse{
		DPCExtras: `{"test": "extras"}`,
		ExpiresAt: "2126-09-08T00:00:00Z",
	}, nil
}

func TestGetZeroTouchConfiguration_PremiumRequired(t *testing.T) {
	inner := &stubService{}
	svc := NewService(inner)

	// No license in context — should return ErrMissingLicense
	ctx := t.Context()
	_, err := svc.GetZeroTouchConfiguration(ctx, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
	assert.False(t, inner.called, "core service should not be called without a premium license")
}

func TestGetZeroTouchConfiguration_FreeLicense(t *testing.T) {
	inner := &stubService{}
	svc := NewService(inner)

	// Free license in context
	ctx := licensectx.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	_, err := svc.GetZeroTouchConfiguration(ctx, nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
	assert.False(t, inner.called, "core service should not be called with a free license")
}

func TestGetZeroTouchConfiguration_PremiumLicense(t *testing.T) {
	inner := &stubService{}
	svc := NewService(inner)

	// Premium license in context — should delegate to core
	ctx := licensectx.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	resp, err := svc.GetZeroTouchConfiguration(ctx, nil)
	require.NoError(t, err)
	assert.True(t, inner.called, "core service should be called with a premium license")
	assert.Contains(t, resp.DPCExtras, "test")
}

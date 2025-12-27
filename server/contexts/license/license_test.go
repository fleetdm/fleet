package license

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockLicense implements LicenseChecker for testing.
type mockLicense struct {
	tier string
}

func (m *mockLicense) IsPremium() bool {
	return m.tier == "premium"
}

func (m *mockLicense) IsAllowDisableTelemetry() bool {
	return m.tier == "premium"
}

func (m *mockLicense) GetTier() string {
	return m.tier
}

func (m *mockLicense) GetOrganization() string {
	return "test-org"
}

func (m *mockLicense) GetDeviceCount() int {
	return 100
}

func TestIsPremium(t *testing.T) {
	cases := []struct {
		desc string
		ctx  context.Context
		want bool
	}{
		{"no license", context.Background(), false},
		{"free license", NewContext(context.Background(), &mockLicense{tier: "free"}), false},
		{"premium license", NewContext(context.Background(), &mockLicense{tier: "premium"}), true},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := IsPremium(c.ctx)
			require.Equal(t, c.want, got)
		})
	}
}

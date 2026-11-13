package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFleetDefaultPolicyID(t *testing.T) {
	require.Equal(t, "fleet0", FleetDefaultPolicyID(0))
	require.Equal(t, "fleet42", FleetDefaultPolicyID(42))
}

func TestIsDefaultPolicyID(t *testing.T) {
	tests := []struct {
		policyID string
		want     bool
	}{
		{policyID: "1", want: true},
		{policyID: FleetDefaultPolicyID(0), want: true},
		{policyID: FleetDefaultPolicyID(42), want: true},
		{policyID: "fleet", want: false},
		{policyID: "fleet-1", want: false},
		{policyID: "fleetabc", want: false},
		{policyID: "2", want: false},
		{policyID: "", want: false},
		{policyID: "a1b2c3d4e5f6a7b8", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.policyID, func(t *testing.T) {
			require.Equal(t, tt.want, IsDefaultPolicyID(tt.policyID))
		})
	}
}

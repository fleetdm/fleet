package main

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestTrayFailingPoliciesCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		sum  fleet.DesktopSummary
		want *uint
	}{
		{
			name: "prefers the unhidden count when the server reports it",
			sum:  fleet.DesktopSummary{FailingPolicies: new(uint(3)), FailingUnhiddenPolicies: new(uint(1))},
			want: new(uint(1)),
		},
		{
			name: "only hidden policies failing shows as no issues",
			sum:  fleet.DesktopSummary{FailingPolicies: new(uint(2)), FailingUnhiddenPolicies: new(uint(0))},
			want: new(uint(0)),
		},
		{
			name: "falls back to the total on servers without the unhidden count",
			sum:  fleet.DesktopSummary{FailingPolicies: new(uint(2))},
			want: new(uint(2)),
		},
		{
			name: "no counts at all",
			sum:  fleet.DesktopSummary{},
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, trayFailingPoliciesCount(tc.sum))
		})
	}
}

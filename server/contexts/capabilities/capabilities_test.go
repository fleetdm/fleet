package capabilities

import (
	"context"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesExist(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  fleet.CapabilityMap
	}{
		{"empty", "", fleet.CapabilityMap{}},
		{"one", "test", fleet.CapabilityMap{fleet.Capability("test"): struct{}{}}},
		{
			"many",
			"test,foo,bar",
			fleet.CapabilityMap{
				fleet.Capability("test"): struct{}{},
				fleet.Capability("foo"):  struct{}{},
				fleet.Capability("bar"):  struct{}{},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := http.Request{
				Header: http.Header{fleet.CapabilitiesHeader: []string{tt.in}},
			}
			ctx := NewContext(context.Background(), &r)
			mp, ok := FromContext(ctx)
			require.True(t, ok)
			require.Equal(t, tt.out, mp)
		})
	}
}

func TestCapabilitiesNotExist(t *testing.T) {
	mp, ok := FromContext(context.Background())
	require.False(t, ok)
	require.Nil(t, mp)
}

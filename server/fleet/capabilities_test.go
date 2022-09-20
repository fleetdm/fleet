package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilityPopulateFromString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  CapabilityMap
	}{
		{"empty", "", CapabilityMap{}},
		{"one", "test", CapabilityMap{Capability("test"): struct{}{}}},
		{
			"many",
			"test,foo,bar",
			CapabilityMap{
				Capability("test"): struct{}{},
				Capability("foo"):  struct{}{},
				Capability("bar"):  struct{}{},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := CapabilityMap{}
			c.PopulateFromString(tt.in)
			require.Equal(t, tt.out, c)
		})
	}
}

func TestCapabilityString(t *testing.T) {
	cases := []struct {
		name string
		in   CapabilityMap
		out  string
	}{
		{"empty", CapabilityMap{}, ""},
		{"one", CapabilityMap{Capability("test"): struct{}{}}, "test"},
		{
			"many",
			CapabilityMap{
				Capability("test"): struct{}{},
				Capability("foo"):  struct{}{},
				Capability("bar"):  struct{}{},
			},
			"test,foo,bar",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.in.String(), tt.out)
		})
	}
}

package fleet

import (
	"strings"
	"sync"
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
			require.ElementsMatch(t, strings.Split(tt.in.String(), ","), strings.Split(tt.out, ","))
		})
	}
}

func TestCapabilityConcurrentWrites(t *testing.T) {
	var wg sync.WaitGroup

	c := make(CapabilityMap)

	numIterations := 1000
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				c.PopulateFromString("test,foo,bar")
			}
		}()
	}

	wg.Wait()

	require.ElementsMatch(t, []string{"test", "foo", "bar"}, strings.Split(c.String(), ","))
}

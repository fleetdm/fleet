package cached_mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

type unclonableTeamMDMConfig fleet.TeamMDM

// This exported variable is to make sure that the compiler doesn't optimize
// away the benchmarked function call.
var Result interface{}

// On my laptop, results are as follows. Under load, the reflection-based
// approach really adds up and is CPU intensive, resulting in drastic
// performance drops.
//
// goos: linux
// goarch: amd64
// pkg: github.com/fleetdm/fleet/v4/server/datastore/cached_mysql
// cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
// BenchmarkCacheGetFallbackClone-8   	   53228	     22706 ns/op	   11976 B/op	     217 allocs/op
// BenchmarkCacheGetCustomClone-8     	 5741186	       196.3 ns/op	     177 B/op	       3 allocs/op

func BenchmarkCacheGetFallbackClone(b *testing.B) {
	v := unclonableTeamMDMConfig(cachedValue())
	benchmarkCacheGet(b, &v)
}

func BenchmarkCacheGetCustomClone(b *testing.B) {
	v := cachedValue()
	benchmarkCacheGet(b, &v)
}

func benchmarkCacheGet(b *testing.B, v any) {
	c := &cloneCache{cache.New(time.Minute, time.Minute)}
	c.Set("k", v, cache.DefaultExpiration)

	b.ResetTimer()

	var ok bool
	for i := 0; i < b.N; i++ {
		Result, ok = c.Get("k")
		if !ok {
			b.Fatal("expected ok")
		}
	}
	require.Equal(b, v, Result)
}

func cachedValue() fleet.TeamMDM {
	return fleet.TeamMDM{
		EnableDiskEncryption: true,
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.10.10"),
			Deadline:       optjson.SetString("1992-03-01"),
		},
		MacOSSettings: fleet.MacOSSettings{
			CustomSettings:                 []string{"a", "b"},
			DeprecatedEnableDiskEncryption: ptr.Bool(false),
		},
		MacOSSetup: fleet.MacOSSetup{
			BootstrapPackage: optjson.SetString("bootstrap"),
		},
	}
}

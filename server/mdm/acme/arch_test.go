package acme_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

var (
	fleetDeps = regexp.MustCompile(`^github\.com/fleetdm/`)

	// Common allowed dependencies across acme packages
	acmePkgs = []string{
		m + "/server/mdm/acme",
		m + "/server/mdm/acme/api",
		m + "/server/mdm/acme/api/http",
		m + "/server/mdm/acme/internal/types",
		// TODO: redis_nonces_store should not leak through the API layer.
		// It's here because api.Service exposes NoncesStore() and the HTTP
		// response types reference it for nonce generation in BeforeRender.
		m + "/server/mdm/acme/internal/redis_nonces_store",
	}

	platformPkgs = []string{
		m + "/server/ptr",
		m + "/server/platform/...",
		m + "/server/contexts/...",
		m + "/server/mdm/internal/commonmdm",
		m + "/pkg/fleethttp",
	}
)

// TestACMEPackageDependencies runs architecture tests for all ACME packages.
// Each package has specific rules about what dependencies are allowed.
func TestACMEPackageDependencies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		pkg               string
		shouldNotDepend   []string // defaults to m + "/..." if empty
		ignoreDeps        []string
		ignoreRecursively []string // for test infra packages whose transitive deps we don't control
		skip              bool     // Temp flag to skip tests that will end up using the redis package, and therefore pull in all of server/fleet, we need to move the datastore redis package out, to enable these.
	}{
		{
			name: "root package has no Fleet dependencies",
			pkg:  m + "/server/mdm/acme",
		},
		{
			name:       "api package only depends on acme and platform packages",
			pkg:        m + "/server/mdm/acme/api",
			ignoreDeps: append(acmePkgs, platformPkgs...),
			skip:       true,
		},
		{
			name: "api/http depends on api and platform",
			pkg:  m + "/server/mdm/acme/api/http",
			ignoreDeps: append(append([]string{
				m + "/server/mdm/acme/api",
			}, acmePkgs...), platformPkgs...),
			skip: true,
		},
		{
			name:       "internal/types only depends on api",
			pkg:        m + "/server/mdm/acme/internal/types",
			ignoreDeps: []string{m + "/server/mdm/acme/api"},
		},
		{
			name: "internal/mysql depends on api, types, and platform",
			pkg:  m + "/server/mdm/acme/internal/mysql",
			ignoreDeps: append([]string{
				m + "/server/mdm/acme/api",
				m + "/server/mdm/acme/internal/types",
				m + "/server/mdm/acme/internal/testutils",
			}, platformPkgs...),
		},
		{
			name: "internal/service depends on acme and platform packages",
			pkg:  m + "/server/mdm/acme/internal/service",
			ignoreDeps: append(append([]string{
				m + "/server/ptr",
				m + "/server/mdm/acme/internal/redis_nonces_store",
			}, acmePkgs...), platformPkgs...),
			skip: true,
		},
		{
			name: "bootstrap depends on acme and platform packages",
			pkg:  m + "/server/mdm/acme/bootstrap",
			ignoreDeps: append(append([]string{
				m + "/server/mdm/acme/internal/mysql",
				m + "/server/mdm/acme/internal/service",
				m + "/server/ptr",
			}, acmePkgs...), platformPkgs...),
			skip: true,
		},
		{
			name: "all packages only depend on acme and platform",
			pkg:  m + "/server/mdm/acme/...",
			ignoreDeps: append(append([]string{
				m + "/server/ptr",
				m + "/server/mdm/acme/internal/mysql",
				m + "/server/mdm/acme/internal/service",
				m + "/server/mdm/acme/internal/testutils",
				m + "/server/mdm/acme/testhelpers",
			}, acmePkgs...), platformPkgs...),
			// Test infrastructure packages whose transitive deps (fleet, etc.) we don't control.
			ignoreRecursively: []string{
				m + "/server/datastore/redis/redistest",
				m + "/server/test",
			},
			skip: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test, due to pulling in server/datastore/redis, which pulls in server/fleet.")
			}
			t.Parallel()

			shouldNotDepend := tc.shouldNotDepend
			if len(shouldNotDepend) == 0 {
				shouldNotDepend = []string{m + "/..."}
			}

			test := archtest.NewPackageTest(t, tc.pkg).
				OnlyInclude(fleetDeps).
				ShouldNotDependOn(shouldNotDepend...).
				WithTests()

			if len(tc.ignoreDeps) > 0 {
				test.IgnoreDeps(tc.ignoreDeps...)
			}
			if len(tc.ignoreRecursively) > 0 {
				test.IgnoreRecursively(tc.ignoreRecursively...)
			}

			test.Check()
		})
	}
}

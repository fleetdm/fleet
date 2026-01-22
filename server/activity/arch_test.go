package activity_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

var (
	fleetDeps = regexp.MustCompile(`^github\.com/fleetdm/`)

	// Common allowed dependencies across activity packages
	activityPkgs = []string{
		m + "/server/activity",
		m + "/server/activity/api",
		m + "/server/activity/api/http",
		m + "/server/activity/internal/types",
	}

	platformPkgs = []string{
		m + "/server/platform/...",
		m + "/server/contexts/ctxerr",
		m + "/server/contexts/viewer",
		m + "/server/contexts/license",
		m + "/server/contexts/logging",
		m + "/server/contexts/authz",
		m + "/server/contexts/publicip",
	}
)

// TestActivityPackageDependencies runs architecture tests for all activity packages.
// Each package has specific rules about what dependencies are allowed.
func TestActivityPackageDependencies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		pkg             string
		shouldNotDepend []string // defaults to m + "/..." if empty
		ignoreDeps      []string
	}{
		{
			name: "root package has no Fleet dependencies",
			pkg:  m + "/server/activity",
		},
		{
			name: "api package has no Fleet dependencies",
			pkg:  m + "/server/activity/api",
		},
		{
			name:       "api/http only depends on api",
			pkg:        m + "/server/activity/api/http",
			ignoreDeps: []string{m + "/server/activity/api"},
		},
		{
			name:       "internal/types only depends on api",
			pkg:        m + "/server/activity/internal/types",
			ignoreDeps: []string{m + "/server/activity/api"},
		},
		{
			name: "internal/mysql depends on api, types, and platform",
			pkg:  m + "/server/activity/internal/mysql",
			ignoreDeps: []string{
				m + "/server/activity/api",
				m + "/server/activity/internal/types",
				m + "/server/activity/internal/testutils",
				m + "/server/platform/http",
				m + "/server/platform/mysql",
				m + "/server/platform/mysql/testing_utils",
				m + "/server/contexts/ctxerr",
				m + "/server/ptr",
			},
		},
		{
			name: "internal/service depends on activity and platform packages",
			pkg:  m + "/server/activity/internal/service",
			ignoreDeps: append(append([]string{
				m + "/server/ptr",
			}, activityPkgs...), platformPkgs...),
		},
		{
			name: "bootstrap depends on activity and platform packages",
			pkg:  m + "/server/activity/bootstrap",
			ignoreDeps: append(append([]string{
				m + "/server/activity/internal/mysql",
				m + "/server/activity/internal/service",
			}, activityPkgs...), platformPkgs...),
		},
		{
			name: "all packages only depend on activity and platform",
			pkg:  m + "/server/activity/...",
			ignoreDeps: append(append([]string{
				m + "/server/ptr",
				m + "/server/activity/internal/mysql",
				m + "/server/activity/internal/service",
				m + "/server/activity/internal/testutils",
			}, activityPkgs...), platformPkgs...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

			test.Check()
		})
	}
}

package chart_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

var (
	fleetDeps = regexp.MustCompile(`^github\.com/fleetdm/`)

	// Common allowed dependencies across chart packages.
	chartPkgs = []string{
		m + "/server/chart",
		m + "/server/chart/api",
		m + "/server/chart/api/http",
		m + "/server/chart/internal/types",
	}

	platformPkgs = []string{
		m + "/server/platform/...",
		m + "/server/contexts/...",
		m + "/pkg/fleethttp",
		m + "/pkg/str",
	}
)

// TestChartPackageDependencies runs architecture tests for all chart packages.
// Each package has specific rules about what dependencies are allowed.
func TestChartPackageDependencies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		pkg             string
		shouldNotDepend []string // defaults to m + "/..." if empty
		ignoreDeps      []string
	}{
		{
			// Root package only depends on api (for dataset implementations).
			name:       "root package only depends on api",
			pkg:        m + "/server/chart",
			ignoreDeps: []string{m + "/server/chart/api"},
		},
		{
			name: "api package has no Fleet dependencies",
			pkg:  m + "/server/chart/api",
		},
		{
			name:       "api/http only depends on api",
			pkg:        m + "/server/chart/api/http",
			ignoreDeps: []string{m + "/server/chart/api"},
		},
		{
			name:       "internal/types only depends on api",
			pkg:        m + "/server/chart/internal/types",
			ignoreDeps: []string{m + "/server/chart/api"},
		},
		{
			name: "internal/mysql depends on chart, types, and platform",
			pkg:  m + "/server/chart/internal/mysql",
			ignoreDeps: slices.Concat(chartPkgs, platformPkgs, []string{
				m + "/server/chart/internal/testutils",
			}),
		},
		{
			name:       "internal/service depends on chart and platform packages",
			pkg:        m + "/server/chart/internal/service",
			ignoreDeps: slices.Concat(chartPkgs, platformPkgs),
		},
		{
			name: "bootstrap depends on chart and platform packages",
			pkg:  m + "/server/chart/bootstrap",
			ignoreDeps: slices.Concat([]string{
				m + "/server/chart/internal/mysql",
				m + "/server/chart/internal/service",
			}, chartPkgs, platformPkgs),
		},
		{
			name: "all packages only depend on chart and platform",
			pkg:  m + "/server/chart/...",
			ignoreDeps: slices.Concat([]string{
				m + "/server/chart/internal/mysql",
				m + "/server/chart/internal/service",
				m + "/server/chart/internal/testutils",
			}, chartPkgs, platformPkgs),
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

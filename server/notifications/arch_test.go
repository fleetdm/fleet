package notifications_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

var (
	fleetDeps = regexp.MustCompile(`^github\.com/fleetdm/`)

	// Common allowed dependencies across notifications packages
	notificationsPkgs = []string{
		m + "/server/notifications",
		m + "/server/notifications/api",
		m + "/server/notifications/api/http",
		m + "/server/notifications/internal/types",
	}

	platformPkgs = []string{
		m + "/server/platform/...",
		m + "/server/contexts/ctxerr",
		m + "/server/contexts/viewer",
		m + "/server/contexts/license",
		m + "/server/contexts/logging",
		m + "/server/contexts/authz",
		m + "/server/contexts/publicip",
		m + "/pkg/fleethttp",
	}
)

// TestNotificationsPackageDependencies runs architecture tests for all notifications packages.
// Each package has specific rules about what dependencies are allowed.
func TestNotificationsPackageDependencies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		pkg             string
		shouldNotDepend []string // defaults to m + "/..." if empty
		ignoreDeps      []string
	}{
		{
			name: "root package has no Fleet dependencies",
			pkg:  m + "/server/notifications",
		},
		{
			name: "api package has no Fleet dependencies",
			pkg:  m + "/server/notifications/api",
		},
		{
			name:       "api/http only depends on api",
			pkg:        m + "/server/notifications/api/http",
			ignoreDeps: []string{m + "/server/notifications/api"},
		},
		{
			name:       "internal/types only depends on api",
			pkg:        m + "/server/notifications/internal/types",
			ignoreDeps: []string{m + "/server/notifications/api"},
		},
		{
			name: "internal/mysql depends on api, types, and platform",
			pkg:  m + "/server/notifications/internal/mysql",
			ignoreDeps: []string{
				m + "/server/notifications/api",
				m + "/server/notifications/internal/types",
				m + "/server/platform/mysql",
				m + "/server/platform/mysql/testing_utils",
				m + "/server/platform/errors",
				m + "/server/contexts/ctxerr",
			},
		},
		{
			name:       "internal/service depends on notifications and platform packages",
			pkg:        m + "/server/notifications/internal/service",
			ignoreDeps: slices.Concat(notificationsPkgs, platformPkgs),
		},
		{
			name: "bootstrap depends on notifications and platform packages",
			pkg:  m + "/server/notifications/bootstrap",
			ignoreDeps: append(append([]string{
				m + "/server/notifications/internal/mysql",
				m + "/server/notifications/internal/service",
			}, notificationsPkgs...), platformPkgs...),
		},
		{
			name: "all packages only depend on notifications and platform",
			pkg:  m + "/server/notifications/...",
			ignoreDeps: append(append([]string{
				m + "/server/notifications/internal/mysql",
				m + "/server/notifications/internal/service",
			}, notificationsPkgs...), platformPkgs...),
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

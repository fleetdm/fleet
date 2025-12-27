package activity_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

// TestAllActivityPackageDependencies checks that activity packages are not dependent on other Fleet packages
// to maintain decoupling and modularity.
// If coupling is necessary, it should be done in the main server/fleet, server/service, or other package.
func TestAllActivityPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/activity/...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		// Should not depend on any Fleet packages
		ShouldNotDependOn("github.com/fleetdm/fleet/v4/...").
		IgnoreDeps(
			// Except for its own packages
			"github.com/fleetdm/fleet/v4/server/activity...",
			// And these packages
			"github.com/fleetdm/fleet/v4/server/platform/...",
			"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck",
			"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils",
			"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit",
			"github.com/fleetdm/fleet/v4/server/contexts/authz",
			"github.com/fleetdm/fleet/v4/server/contexts/ctxerr",
			"github.com/fleetdm/fleet/v4/server/contexts/license",
			"github.com/fleetdm/fleet/v4/server/contexts/logging",
			"github.com/fleetdm/fleet/v4/server/contexts/publicip",
		).
		Check()
}

// TestActivityPackageDependencies checks that activity package is NOT dependent on ANY other Fleet packages
// to maintain decoupling and modularity. This package should only contain basic structs and interfaces.
// If coupling is necessary, it should be done in the main server/fleet or another package.
func TestActivityPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/activity").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn("github.com/fleetdm/fleet/v4/...").
		Check()
}

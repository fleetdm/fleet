package platform_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

// TestEndpointerPackageDependencies checks that endpointer package is not dependent on other Fleet domain packages
// to maintain decoupling and modularity.
func TestEndpointerPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/service/middleware/endpoint_utils").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Platform packages
			m+"/server/platform...",
			// Other infra packages
			m+"/server/contexts/authz",
			m+"/server/contexts/ctxerr",
			m+"/server/contexts/license",
			m+"/server/contexts/logging",
			m+"/server/contexts/publicip",
			m+"/server/service/middleware/authzcheck",
			m+"/server/service/middleware/ratelimit",
		).
		Check()
}

// TestPlatformPackageDependencies checks that platform packages are NOT dependent on ANY other Fleet packages
// to maintain decoupling and modularity.
func TestPlatformPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m + "/...").
		Check()
}

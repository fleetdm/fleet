package platform_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

// TestPlatformPackageDependencies checks that platform packages are NOT dependent on Fleet domain packages
// to maintain decoupling and modularity. This is a catch-all test.
func TestPlatformPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Platform packages can depend on each other
			m+"/server/platform...",
			// Infra packages
			m+"/server/contexts/authz",
			m+"/server/contexts/ctxerr",
			m+"/server/contexts/license",
			m+"/server/contexts/logging",
			m+"/server/contexts/publicip",
		).
		Check()
}

// TestEndpointerPackageDependencies checks that endpointer package is not dependent on other Fleet domain packages
// to maintain decoupling and modularity.
func TestEndpointerPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/endpointer").
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
		).
		Check()
}

func TestHTTPPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/http").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m + "/...").
		Check()
}

func TestAuthzCheckPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/middleware/authzcheck").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		IgnoreDeps(
			// Platform packages
			m+"/server/platform/http",
			// Other infra packages
			m+"/server/contexts/authz",
		).
		Check()
}

func TestRatelimitPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/middleware/ratelimit").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Platform packages
			m+"/server/platform/http",
			// Other infra packages
			m+"/server/contexts/authz",
			m+"/server/contexts/ctxerr",
			m+"/server/contexts/publicip",
		).
		Check()
}

// TestMysqlPackageDependencies checks that mysql package is not dependent on other Fleet domain packages
// to maintain decoupling and modularity.
func TestMysqlPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/mysql...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Ignore our own packages
			m+"/server/platform/mysql...",
			// Other infra packages
			m+"/server/platform/http",
			m+"/server/contexts/ctxerr",
		).
		Check()
}

func TestLoggingPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/platform/logging...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m + "/...").
		Check()
}

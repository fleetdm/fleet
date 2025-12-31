package common_mysql_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

// TestMysqlPackageDependencies checks that mysql package is not dependent on other Fleet domain packages
// to maintain decoupling and modularity.
func TestMysqlPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/datastore/mysql/common_mysql...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Ignore our own packages
			m+"/server/datastore/mysql/common_mysql...",
			// Other infra packages
			m+"/server/platform/http",
			m+"/server/contexts/ctxerr",
		).
		Check()
}

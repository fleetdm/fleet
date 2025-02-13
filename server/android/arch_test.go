package android_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

// TestAndroidPackageDependencies checks that android packages are not dependent on other Fleet packages
// to maintain decoupling and modularity.
func TestAndroidPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/android...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		IgnorePackages(
			"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql",
			"github.com/fleetdm/fleet/v4/server/service/externalsvc", // TODO(#26218): remove this dependency on Jira and Zendesk
			"github.com/fleetdm/fleet/v4/server/service/middleware/auth",
			"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck",
			"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils",
			"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit",
		).
		ShouldNotDependOn(
			"github.com/fleetdm/fleet/v4/server/service...",
			"github.com/fleetdm/fleet/v4/server/datastore...",
		)
}

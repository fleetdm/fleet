package android_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

// TestAllAndroidPackageDependencies checks that android packages are not dependent on other Fleet packages
// to maintain decoupling and modularity.
// If coupling is necessary, it should be done in the main server/fleet, server/service, or other package.
func TestAllAndroidPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/mdm/android...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		IgnoreXTests("github.com/fleetdm/fleet/v4/server/fleet"). // ignore fleet_test package
		IgnorePackages(
			"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql...",
			"github.com/fleetdm/fleet/v4/server/service/externalsvc", // dependency on Jira and Zendesk
			"github.com/fleetdm/fleet/v4/server/service/middleware/auth",
			"github.com/fleetdm/fleet/v4/server/service/middleware/authzcheck",
			"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils",
			"github.com/fleetdm/fleet/v4/server/service/middleware/log",
			"github.com/fleetdm/fleet/v4/server/service/middleware/ratelimit",
		).
		ShouldNotDependOn(
			"github.com/fleetdm/fleet/v4/server/service...",
			"github.com/fleetdm/fleet/v4/server/datastore...",
		)
}

// TestAndroidPackageDependencies checks that android package is NOT dependent on ANY other Fleet packages
// to maintain decoupling and modularity. This package should only contain basic structs and interfaces.
// If coupling is necessary, it should be done in the main server/fleet or another package.
func TestAndroidPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/mdm/android").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn("github.com/fleetdm/fleet/v4/...")
}

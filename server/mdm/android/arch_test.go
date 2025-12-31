package android_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

// TestAllAndroidPackageDependencies checks that android packages are not dependent on other Fleet packages
// to maintain decoupling and modularity.
// If coupling is necessary, it should be done in the main server/fleet, server/service, or other package.
func TestAllAndroidPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/mdm/android...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(
			m+"/server/service...",
			m+"/server/datastore/mysql...",
		).
		IgnoreRecursively(
			m+"/server/mdm/android/tests",
		).
		IgnoreDeps(
			// Android packages
			m+"/server/mdm/android...",
			// Other/infra packages
			m+"/server/datastore/mysql/common_mysql",
			m+"/server/service/externalsvc", // dependency on Jira and Zendesk
			m+"/server/service/middleware/auth",
			m+"/server/service/middleware/authzcheck",
			m+"/server/service/middleware/endpoint_utils",
			m+"/server/service/middleware/log",
			m+"/server/service/middleware/ratelimit",
			m+"/server/service/modules/activities",
		).
		Check()
}

// TestAndroidPackageDependencies checks that android package is NOT dependent on ANY other Fleet packages
// to maintain decoupling and modularity. This package should only contain basic structs and interfaces.
// If coupling is necessary, it should be done in the main server/fleet or another package.
func TestAndroidPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/mdm/android").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m + "/...").
		Check()
}

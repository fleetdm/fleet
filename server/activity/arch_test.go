package activity_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

const m = archtest.ModuleName

// TestActivityRootPackageDependencies ensures the root activity package has NO Fleet dependencies.
// This package only contains types needed by the ACL layer (User, UserProvider).
func TestActivityRootPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m + "/...").
		Check()
}

// TestActivityAPIPackageDependencies ensures the public API activity package has NO Fleet dependencies.
func TestActivityAPIPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity/api").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m + "/...").
		Check()
}

func TestActivityInternalTypesDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity/internal/types").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m + "/...").
		IgnoreDeps(
			m + "/server/activity/api",
		).
		Check()
}

// TestActivityInternalMySQLDependencies ensures the mysql package doesn't depend on legacy packages.
func TestActivityInternalMySQLDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity/internal/mysql").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Activity packages (api is the public interface)
			m+"/server/activity/api",
			m+"/server/activity/internal/types",
			// Platform/infra packages (allowed)
			m+"/server/platform/http",
			m+"/server/platform/mysql",
			m+"/server/contexts/ctxerr",
		).
		Check()
}

// TestActivityInternalServiceDependencies ensures the service package doesn't depend on legacy packages.
func TestActivityInternalServiceDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity/internal/service").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Activity packages (api is the public interface)
			m+"/server/activity",
			m+"/server/activity/api",
			m+"/server/activity/internal/types",
			// Platform/infra packages (allowed)
			m+"/server/platform/...",
			m+"/server/contexts/ctxerr",
			m+"/server/contexts/viewer",
			m+"/server/contexts/license",
			m+"/server/contexts/logging",
			m+"/server/contexts/authz",    // transitive via authzcheck
			m+"/server/contexts/publicip", // transitive via ratelimit
		).
		Check()
}

// TestActivityBootstrapDependencies ensures bootstrap only depends on what's needed for wiring.
// Note: endpoint_utils has transitive dependencies on fleet that we must allow.
// This is acceptable as these are infra/platform packages, not business logic.
func TestActivityBootstrapDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, m+"/server/activity/bootstrap").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		ShouldNotDependOn(m+"/...").
		IgnoreDeps(
			// Activity packages
			m+"/server/activity...",
			// Platform/infra packages (allowed) - including their transitive deps
			m+"/server/platform/...",
			m+"/server/contexts/ctxerr",
			m+"/server/contexts/viewer",
			m+"/server/contexts/license",
			m+"/server/contexts/logging",
			m+"/server/contexts/authz",    // transitive via authzcheck
			m+"/server/contexts/publicip", // transitive via ratelimit
		).
		Check()
}

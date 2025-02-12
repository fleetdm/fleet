package common_test

import (
	"regexp"
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest"
)

// TestFleetCommonPackageDependencies checks that common package is NOT dependent on ANY other Fleet packages
// to maintain decoupling and modularity.
func TestFleetCommonPackageDependencies(t *testing.T) {
	t.Parallel()
	archtest.NewPackageTest(t, "github.com/fleetdm/fleet/v4/server/fleet/common...").
		OnlyInclude(regexp.MustCompile(`^github\.com/fleetdm/`)).
		WithTests().
		ShouldNotDependOn(
			"github.com/fleetdm/fleet/v4/...",
		)
}

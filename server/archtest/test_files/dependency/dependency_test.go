package dependency

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest/test_files/testfiledeps/transitivetestdep"
)

func TestDependency(t *testing.T) {
	transitivetestdep.TransitiveTestHelper()
}

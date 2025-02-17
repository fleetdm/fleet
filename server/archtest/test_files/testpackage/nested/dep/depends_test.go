package dep

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/archtest/test_files/testfiledeps/testonlydependency"
)

func TestDoIBreakYou(t *testing.T) {
	testonlydependency.OohNoBadCode()
}

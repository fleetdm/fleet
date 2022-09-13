package msrc

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func Analyze(
	ctx context.Context,
	ds fleet.Datastore,
	os fleet.OperatingSystem,
	vulnPath string,
	collectVulns bool,
) error {
	panic("not implemented")
}

package inmem

import (
	"time"

	"github.com/fleetdm/fleet/server/kolide"
)

func (d *Datastore) CountHostsInTargets(hostIDs, labelIDs []uint, now time.Time) (kolide.TargetMetrics, error) {
	// noop
	return kolide.TargetMetrics{}, nil
}

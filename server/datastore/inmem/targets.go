package inmem

import (
	"time"

	"github.com/fleetdm/fleet/server/kolide"
)

func (d *Datastore) CountHostsInTargets(filter kolide.TeamFilter, targets kolide.HostTargets, now time.Time) (kolide.TargetMetrics, error) {
	// noop
	return kolide.TargetMetrics{}, nil
}

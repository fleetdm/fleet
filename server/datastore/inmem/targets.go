package inmem

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (d *Datastore) CountHostsInTargets(filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
	// noop
	return fleet.TargetMetrics{}, nil
}

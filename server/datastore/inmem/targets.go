package inmem

import (
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (d *Datastore) CountHostsInTargets(hostIDs, labelIDs []uint, now time.Time) (kolide.TargetMetrics, error) {
	// noop
	return kolide.TargetMetrics{}, nil
}

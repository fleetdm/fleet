// Package notificationsacl is the only place that imports both notifications
// and fleet types, which is what lets the notifications bounded context stay
// independent of the rest of Fleet.
package notificationsacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/notifications"
)

type FleetServiceAdapter struct {
	ds fleet.Datastore
}

func NewFleetServiceAdapter(ds fleet.Datastore) *FleetServiceAdapter {
	return &FleetServiceAdapter{ds: ds}
}

var _ notifications.ScriptQueueProvider = (*FleetServiceAdapter)(nil)

func (a *FleetServiceAdapter) QueueScriptForHosts(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error) {
	return a.ds.BatchNewInternalHostScriptExecutionRequests(ctx, hostIDs, contents)
}

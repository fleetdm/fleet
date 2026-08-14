// Package notificationsacl provides the anti-corruption layer between the
// notifications bounded context and the rest of Fleet.
//
// This package is the ONLY place that imports both notifications types and
// fleet types. It translates between them, allowing the notifications
// context to remain decoupled from the rest of the codebase.
package notificationsacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/notifications"
)

// FleetServiceAdapter provides access to Fleet datastore methods for data
// that the notifications bounded context doesn't own.
type FleetServiceAdapter struct {
	ds fleet.Datastore
}

// NewFleetServiceAdapter creates a new adapter for the Fleet datastore.
func NewFleetServiceAdapter(ds fleet.Datastore) *FleetServiceAdapter {
	return &FleetServiceAdapter{ds: ds}
}

// Ensure FleetServiceAdapter implements the required interfaces
var _ notifications.ScriptQueueProvider = (*FleetServiceAdapter)(nil)

// QueueScriptForHosts queues contents once and activates it next for each
// host, delegating to the script queue in server/datastore/mysql that the
// notifications context shares with everything else a host runs.
func (a *FleetServiceAdapter) QueueScriptForHosts(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error) {
	return a.ds.BatchNewInternalHostScriptExecutionRequests(ctx, hostIDs, contents)
}

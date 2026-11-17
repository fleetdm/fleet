// Package async batches host updates in memory so that the hot path of host
// check-ins doesn't write to mysql on every request.
package async

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type Task struct {
	datastore   fleet.Datastore
	seenHostSet seenHostSet
	otelEnabled bool
}

func NewTask(ds fleet.Datastore, otelEnabled bool) *Task {
	return &Task{
		datastore:   ds,
		otelEnabled: otelEnabled,
	}
}

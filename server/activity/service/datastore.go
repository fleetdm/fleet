package service

import (
	"context"
)

// Datastore defines the datastore interface for the activity bounded context.
// This interface is internal to the activity context and should not be
// imported by other bounded contexts.
//
// Other bounded contexts should use the public service interface instead.
type Datastore interface {
	// Ping verifies database connectivity.
	// This is a placeholder method that will be replaced by actual activity methods.
	Ping(ctx context.Context) error
}

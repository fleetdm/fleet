package api

import "context"

// DispatchService expires due notifications and queues scripts for the rest.
type DispatchService interface {
	// Dispatch expires notifications past their expiry, then queues a script
	// for each notification that is due.
	Dispatch(ctx context.Context) error
}

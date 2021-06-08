package fleet

import "context"

type StatusService interface {
	// StatusResultStore returns nil if the result store is functioning
	// correctly, or an error indicating the problem.
	StatusResultStore(ctx context.Context) error

	// StatusLiveQuery returns nil if live queries are enabled, or an
	// error indicating the problem.
	StatusLiveQuery(ctx context.Context) error
}

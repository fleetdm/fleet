package kolide

import "context"

type StatusService interface {
	// StatusResultStore returns nil if the result store is functioning
	// correctly, or an error indicating the problem.
	StatusResultStore(ctx context.Context) error
}

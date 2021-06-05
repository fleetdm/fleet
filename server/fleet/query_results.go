package fleet

import (
	"context"
)

// QueryResultStore defines functions for sending and receiving distributed
// query results over a pub/sub system. It is implemented by structs in package
// pubsub.
type QueryResultStore interface {
	// WriteResult writes a distributed query result submitted by an
	// osqueryd client
	WriteResult(result DistributedQueryResult) error

	// ReadChannel returns a channel to be read for incoming distributed
	// query results. Channel values should be either
	// DistributedQueryResult or error
	ReadChannel(ctx context.Context, query DistributedQueryCampaign) (<-chan interface{}, error)

	// HealthCheck returns nil if the store is functioning properly, or an
	// error describing the problem.
	HealthCheck() error
}

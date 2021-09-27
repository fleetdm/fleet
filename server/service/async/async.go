package async

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type Task struct {
	Datastore fleet.Datastore
	Pool      fleet.RedisPool
	// AsyncEnabled indicates if async processing is enabled in the
	// configuration. Note that Pool can be nil if this is false.
	AsyncEnabled bool // TODO: should this be read in a different way, more dynamically, if config changes while fleet is running?
}

func (t *Task) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
	host.LabelUpdatedAt = ts
	return t.Datastore.RecordLabelQueryExecutions(ctx, host, results, ts)
}

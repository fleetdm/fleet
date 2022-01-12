package async

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const collectorLockKey = "locks:async_collector:{%s}"

type Task struct {
	Datastore fleet.Datastore
	Pool      fleet.RedisPool
	// AsyncEnabled indicates if async processing is enabled in the
	// configuration. Note that Pool can be nil if this is false.
	AsyncEnabled bool

	LockTimeout        time.Duration
	LogStatsInterval   time.Duration
	InsertBatch        int
	DeleteBatch        int
	UpdateBatch        int
	RedisPopCount      int
	RedisScanKeysCount int
	CollectorInterval  time.Duration
}

// Collect runs the various collectors as distinct background goroutines if
// async processing is enabled.  Each collector will stop processing when ctx
// is done.
func (t *Task) StartCollectors(ctx context.Context, jitterPct int, logger kitlog.Logger) {
	if !t.AsyncEnabled {
		level.Debug(logger).Log("task", "async disabled, not starting collectors")
		return
	}
	level.Debug(logger).Log("task", "async enabled, starting collectors", "interval", t.CollectorInterval, "jitter", jitterPct)

	collectorErrHandler := func(name string, err error) {
		level.Error(logger).Log("err", fmt.Sprintf("%s collector", name), "details", err)
	}

	labelColl := &collector{
		name:         "collect_labels",
		pool:         t.Pool,
		ds:           t.Datastore,
		execInterval: t.CollectorInterval,
		jitterPct:    jitterPct,
		lockTimeout:  t.LockTimeout,
		handler:      t.collectLabelQueryExecutions,
		errHandler:   collectorErrHandler,
	}

	policyColl := &collector{
		name:         "collect_policies",
		pool:         t.Pool,
		ds:           t.Datastore,
		execInterval: t.CollectorInterval,
		jitterPct:    jitterPct,
		lockTimeout:  t.LockTimeout,
		handler:      t.collectPolicyQueryExecutions,
		errHandler:   collectorErrHandler,
	}

	colls := []*collector{labelColl, policyColl}
	for _, coll := range colls {
		go coll.Start(ctx)
	}

	// log stats at regular intervals
	if t.LogStatsInterval > 0 {
		go func() {
			tick := time.Tick(t.LogStatsInterval)
			for {
				select {
				case <-tick:
					for _, coll := range colls {
						stats := coll.ReadStats()
						level.Debug(logger).Log("stats", fmt.Sprintf("%#v", stats), "name", coll.name)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

package async

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	hostSeenRecordedHostIDsKey   = "{host_seen:host_ids}"            // the SET of current (pending) host ids
	hostSeenProcessingHostIDsKey = "{host_seen:host_ids}:processing" // the SET of host ids in the process of being collected
	hostSeenKeysMinTTL           = 7 * 24 * time.Hour                // 1 week
)

// RecordHostLastSeen records that the specified host ID was seen.
func (t *Task) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	cfg := t.taskConfigs[config.AsyncTaskHostLastSeen]
	if !cfg.Enabled {
		t.seenHostSet.addHostID(hostID)
		return nil
	}

	// set an expiration on the SET key, ensuring that if async processing is
	// disabled, the set (eventually) does not use any redis space. Ensure that
	// TTL is reasonably big to avoid deleting information that hasn't been
	// collected yet - 1 week or 10 * the collector interval, whichever is
	// biggest.
	ttl := hostSeenKeysMinTTL
	if maxTTL := 10 * cfg.CollectInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// keys and arguments passed to the script are:
	// KEYS[1]: recorded set (hostSeenRecordedHostIDsKey)
	// ARGV[1]: host id
	// ARGV[2]: ttl for the key
	script := redigo.NewScript(1, `
    redis.call('SADD', KEYS[1], ARGV[1])
    return redis.call('EXPIRE', KEYS[1], ARGV[2])
  `)

	conn := t.pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.pool, conn, hostSeenRecordedHostIDsKey); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, hostSeenRecordedHostIDsKey, hostID, int(ttl.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}
	return nil
}

// FlushHostsLastSeen updates the last seen timestamp for the hosts that have
// been recorded since the last time FlushHostsLastSeen was called. It is a
// no-op if asychronous host processing is enabled, because then it is the
// task collector that will process the writes to mysql.
func (t *Task) FlushHostsLastSeen(ctx context.Context, now time.Time) error {
	cfg := t.taskConfigs[config.AsyncTaskHostLastSeen]
	if !cfg.Enabled {
		// Create a root span for this synchronous flush task if OTEL is enabled
		if t.otelEnabled {
			tracer := otel.Tracer("async")
			var span trace.Span
			ctx, span = tracer.Start(ctx, "async.flush_hosts_last_seen",
				trace.WithAttributes(
					attribute.String("async.mode", "synchronous"),
					attribute.String("async.task", "host_last_seen"),
				),
			)
			defer span.End()
		}

		hostIDs := t.seenHostSet.getAndClearHostIDs()
		return t.datastore.MarkHostsSeen(ctx, hostIDs, now)
	}

	// no-op, flushing the hosts' last seen is done via the cron that runs the
	// Task's collectors.
	return nil
}

func (t *Task) collectHostsLastSeen(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	// Create a root span for this async collection task if OTEL is enabled
	if t.otelEnabled {
		tracer := otel.Tracer("async")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "async.collect_hosts_last_seen",
			trace.WithAttributes(
				attribute.String("async.task", "host_last_seen"),
			),
		)
		defer span.End()
	}

	cfg := t.taskConfigs[config.AsyncTaskHostLastSeen]

	hostIDs, err := t.loadSeenHostsIDs(ctx, pool)
	if err != nil {
		return err
	}
	stats.RedisCmds++ // the script to load seen hosts
	stats.Keys = 2    // the reported and processing set keys
	stats.Items = len(hostIDs)

	// process in batches, as there could be many thousand host IDs
	if len(hostIDs) > 0 {
		// globally sort the host IDs so they are sent ordered as batches to MarkHostsSeen
		sort.Slice(hostIDs, func(i, j int) bool { return hostIDs[i] < hostIDs[j] })

		ts := t.clock.Now()
		batch := make([]uint, cfg.InsertBatch)
		for {
			n := copy(batch, hostIDs)
			if n == 0 {
				break
			}
			if err := ds.MarkHostsSeen(ctx, batch[:n], ts); err != nil {
				return err
			}
			stats.Inserts++
			hostIDs = hostIDs[n:]
		}
	}

	conn := pool.Get()
	defer conn.Close()
	if _, err := conn.Do("DEL", hostSeenProcessingHostIDsKey); err != nil {
		return ctxerr.Wrap(ctx, err, "delete processing set key")
	}

	return nil
}

func (t *Task) loadSeenHostsIDs(ctx context.Context, pool fleet.RedisPool) ([]uint, error) {
	cfg := t.taskConfigs[config.AsyncTaskHostLastSeen]

	// compute the TTL for the processing key just as we do for the storage key,
	// in case the collection fails before removing the working key, we don't
	// want it to stick around forever.
	ttl := hostSeenKeysMinTTL
	if maxTTL := 10 * cfg.CollectInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// keys and arguments passed to the script are:
	// KEYS[1]: recorded set (hostSeenRecordedHostIDsKey)
	// KEYS[2]: processing set (hostSeenProcessingHostIDsKey)
	// ARGV[1]: ttl for the processing key
	script := redigo.NewScript(2, `
    redis.call('SUNIONSTORE', KEYS[2], KEYS[1], KEYS[2])
    redis.call('DEL', KEYS[1])
    return redis.call('EXPIRE', KEYS[2], ARGV[1])
  `)

	conn := pool.Get()
	defer conn.Close()
	if err := redis.BindConn(pool, conn, hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey, int(ttl.Seconds())); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "run redis script")
	}

	var ids []uint
	cursor := 0
	for {
		res, err := redigo.Values(conn.Do("SSCAN", hostSeenProcessingHostIDsKey, cursor, "COUNT", cfg.RedisScanKeysCount))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scan seen host ids")
		}
		var scanIDs []uint
		if _, err := redigo.Scan(res, &cursor, &scanIDs); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "convert scan results")
		}
		ids = append(ids, scanIDs...)

		if cursor == 0 {
			// iteration completed
			return ids, nil
		}
	}
}

// seenHostSet implements synchronized storage for the set of seen hosts.
type seenHostSet struct {
	mutex   sync.Mutex
	hostIDs map[uint]bool
}

// addHostID adds the host identified by ID to the set
func (m *seenHostSet) addHostID(id uint) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.hostIDs == nil {
		m.hostIDs = make(map[uint]bool)
	}
	m.hostIDs[id] = true
}

// getAndClearHostIDs gets the list of unique host IDs from the set and empties
// the set.
func (m *seenHostSet) getAndClearHostIDs() []uint {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var ids []uint
	for id := range m.hostIDs {
		ids = append(ids, id)
	}
	m.hostIDs = make(map[uint]bool)
	return ids
}

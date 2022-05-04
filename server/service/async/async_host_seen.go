package async

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	hostSeenRecordedHostIDsKey   = "{host_seen:host_ids}"            // the SET of current (pending) host ids
	hostSeenProcessingHostIDsKey = "{host_seen:host_ids}:processing" // the SET of host ids in the process of being collected
	hostSeenKeysMinTTL           = 7 * 24 * time.Hour                // 1 week
)

// RecordHostLastSeen records that the specified host ID was seen.
func (t *Task) RecordHostLastSeen(ctx context.Context, hostID uint) error {
	if !t.AsyncEnabled {
		t.seenHostSet.addHostID(hostID)
		return nil
	}

	// set an expiration on the SET key, ensuring that if async processing is
	// disabled, the set (eventually) does not use any redis space. Ensure that
	// TTL is reasonably big to avoid deleting information that hasn't been
	// collected yet - 1 week or 10 * the collector interval, whichever is
	// biggest.
	ttl := hostSeenKeysMinTTL
	if maxTTL := 10 * t.CollectorInterval; maxTTL > ttl {
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

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, hostSeenRecordedHostIDsKey); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, hostSeenRecordedHostIDsKey, int(ttl.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}
	return nil
}

// FlushHostsLastSeen updates the last seen timestamp for the hosts that have
// been recorded since the last time FlushHostsLastSeen was called. It is a
// no-op if asychronous host processing is enabled, because then it is the
// task collector that will process the writes to mysql.
func (t *Task) FlushHostsLastSeen(ctx context.Context, now time.Time) error {
	if !t.AsyncEnabled {
		hostIDs := t.seenHostSet.getAndClearHostIDs()
		return t.Datastore.MarkHostsSeen(ctx, hostIDs, now)
	}

	// no-op, flushing the hosts' last seen is done via the cron that runs the
	// Task's collectors.
	return nil
}

func (t *Task) collectHostsLastSeen(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	hostIDs, err := t.loadSeenHostsIDs(ctx)
	if err != nil {
		return err
	}
	stats.RedisCmds++ // the script to load seen hosts
	stats.Keys = 2    // the reported and processing set keys
	stats.Items = len(hostIDs)
	if err := t.Datastore.MarkHostsSeen(ctx, hostIDs, time.Now()); err != nil {
		return err
	}
	// TODO: cannot really increment INSERTs or UPDATEs, maybe an UPSERT count?

	conn := t.Pool.Get()
	defer conn.Close()
	if _, err := conn.Do("DEL", hostSeenProcessingHostIDsKey); err != nil {
		return ctxerr.Wrap(ctx, err, "delete processing set key")
	}

	return nil

	/*
		// Based on those pages, the best approach appears to be INSERT with multiple
		// rows in the VALUES section (short of doing LOAD FILE, which we can't):
		// https://www.databasejournal.com/features/mysql/optimize-mysql-inserts-using-batch-processing.html
		// https://dev.mysql.com/doc/refman/5.7/en/insert-optimization.html
		// https://dev.mysql.com/doc/refman/5.7/en/optimizing-innodb-bulk-data-loading.html
		//
		// Given that there are no UNIQUE constraints in label_membership (well,
		// apart from the primary key columns), no AUTO_INC column and no FOREIGN
		// KEY, there is no obvious setting to tweak (based on the recommendations of
		// the third link above).
		//
		// However, in label_membership, updated_at defaults to the current timestamp
		// both on INSERT and when UPDATEd, so it does not need to be provided.

		runInsertBatch := func(batch [][2]uint) error {
			stats.Inserts++
			return ds.AsyncBatchInsertLabelMembership(ctx, batch)
		}

		runDeleteBatch := func(batch [][2]uint) error {
			stats.Deletes++
			return ds.AsyncBatchDeleteLabelMembership(ctx, batch)
		}

		runUpdateBatch := func(ids []uint, ts time.Time) error {
			stats.Updates++
			return ds.AsyncBatchUpdateLabelTimestamp(ctx, ids, ts)
		}

		insertBatch := make([][2]uint, 0, t.InsertBatch)
		deleteBatch := make([][2]uint, 0, t.DeleteBatch)
		for _, host := range hosts {
			hid := host.HostID
			ins, del, err := getKeyTuples(hid)
			if err != nil {
				return err
			}
			insertBatch = append(insertBatch, ins...)
			deleteBatch = append(deleteBatch, del...)

			if len(insertBatch) >= t.InsertBatch {
				if err := runInsertBatch(insertBatch); err != nil {
					return err
				}
				insertBatch = insertBatch[:0]
			}
			if len(deleteBatch) >= t.DeleteBatch {
				if err := runDeleteBatch(deleteBatch); err != nil {
					return err
				}
				deleteBatch = deleteBatch[:0]
			}
		}

		// process any remaining batch that did not reach the batchSize limit in the
		// loop.
		if len(insertBatch) > 0 {
			if err := runInsertBatch(insertBatch); err != nil {
				return err
			}
		}
		if len(deleteBatch) > 0 {
			if err := runDeleteBatch(deleteBatch); err != nil {
				return err
			}
		}
		if len(hosts) > 0 {
			hostIDs := make([]uint, len(hosts))
			for i, host := range hosts {
				hostIDs[i] = host.HostID
			}

			ts := time.Now()
			updateBatch := make([]uint, t.UpdateBatch)
			for {
				n := copy(updateBatch, hostIDs)
				if n == 0 {
					break
				}
				if err := runUpdateBatch(updateBatch[:n], ts); err != nil {
					return err
				}
				hostIDs = hostIDs[n:]
			}

			// batch-remove any host ID from the active set that still has its score to
			// the initial value, so that the active set does not keep all (potentially
			// 100K+) host IDs to process at all times - only those with reported
			// results to process.
			if _, err := removeProcessedHostIDs(pool, labelMembershipActiveHostIDsKey, hosts); err != nil {
				return ctxerr.Wrap(ctx, err, "remove processed host ids")
			}
		}
	*/
}

func (t *Task) loadSeenHostsIDs(ctx context.Context) ([]uint, error) {
	// compute the TTL for the processing key just as we do for the storage key,
	// in case the collection fails before removing the working key, we don't
	// want it to stick around forever.
	ttl := hostSeenKeysMinTTL
	if maxTTL := 10 * t.CollectorInterval; maxTTL > ttl {
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

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey, int(ttl.Seconds())); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "run redis script")
	}

	var ids []uint
	cursor := 0
	for {
		res, err := redigo.Values(conn.Do("SSCAN", hostSeenProcessingHostIDsKey, cursor, "COUNT", t.RedisScanKeysCount))
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

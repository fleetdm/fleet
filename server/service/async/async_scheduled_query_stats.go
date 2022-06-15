package async

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	scheduledQueryStatsHostQueriesKey = "scheduled_query_stats:{%d}" // the hash of scheduled query stats for a given host
	scheduledQueryStatsHostIDsKey     = "scheduled_query_stats:active_host_ids"
	scheduledQueryStatsKeyMinTTL      = 7 * 24 * time.Hour // 1 week
)

// RecordScheduledQueryStats records the scheduled query stats for a given host.
func (t *Task) RecordScheduledQueryStats(ctx context.Context, hostID uint, stats []fleet.PackStats, ts time.Time) error {
	cfg := t.taskConfigs[config.AsyncTaskScheduledQueryStats]
	if !cfg.Enabled {
		return t.datastore.SaveHostPackStats(ctx, hostID, stats)
	}

	// set an expiration on the  key, ensuring that if async processing is
	// disabled, the key (eventually) does not use any redis space. Ensure that
	// TTL is reasonably big to avoid deleting information that hasn't been
	// collected yet - 1 week or 10 * the collector interval, whichever is
	// biggest.
	ttl := scheduledQueryStatsKeyMinTTL
	if maxTTL := 10 * cfg.CollectInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// the redis key is a hash where the field names are the
	// packname\x00queryname and the values are the json-marshaled
	// fleet.PackStats struct. Using this structure ensures the fields are always
	// overridden with the latest stats for the pack/query tuple, which mirrors
	// the behavior of the mysql table (we only keep the latest stats).

	// keys and arguments passed to the script are:
	// KEYS[1]: hash key (scheduledQueryStatsHostQueriesKey)
	// ARGV[1]: ttl for the key
	// ARGV[2:]: arguments for HSET
	script := redigo.NewScript(1, `
    redis.call('HSET', KEYS[1], unpack(ARGV, 2))
    return redis.call('EXPIRE', KEYS[1], ARGV[1])
  `)

	// build the arguments
	key := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, hostID)
	args := redigo.Args{key, int(ttl.Seconds())}
	for _, ps := range stats {
		for _, qs := range ps.QueryStats {
			field := fmt.Sprintf("%s\x00%s", ps.PackName, qs.ScheduledQueryName)
			jsonStat, err := json.Marshal(ps)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "marshal pack stats")
			}
			args = args.Add(field, string(jsonStat))
		}
	}

	conn := t.pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.pool, conn, key); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}

	// Storing the host id in the set of active host IDs for scheduled query stats
	// outside of the redis script because in Redis Cluster mode the key may not
	// live on the same node as the host's keys. At the same time, purge any
	// entry in the set that is older than now - TTL.
	if _, err := storePurgeActiveHostID(t.pool, scheduledQueryStatsHostIDsKey, hostID, ts, ts.Add(-ttl)); err != nil {
		return ctxerr.Wrap(ctx, err, "store active host id")
	}
	return nil
}

func (t *Task) collectScheduledQueryStats(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	cfg := t.taskConfigs[config.AsyncTaskScheduledQueryStats]

	hosts, err := loadActiveHostIDs(pool, scheduledQueryStatsHostIDsKey, cfg.RedisScanKeysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load active host ids")
	}
	stats.Keys = len(hosts)

	// TODO(mna): packs typically target many hosts, so it's very likely that the
	// batch of recorded stats have a small-ish set of packName+schedQueryName that
	// is repeated many times over. Instead of loading those schedQuery IDs for each
	// insertion of the stats, load it once before processing the batch.

	getHostStats := func(hostID uint) (packStats []fleet.PackStats, schedQueryNames [][2]string, err error) {
		keyHash := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, hostID)
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		var cursor int
		for {
			stats.RedisCmds++

			res, err := redigo.Values(conn.Do("HSCAN", keyHash, cursor, "COUNT", cfg.RedisScanKeysCount))
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "redis HSCAN")
			}
			var hashFieldVals []interface{}
			if _, err := redigo.Scan(res, &cursor, &hashFieldVals); err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "scan HSCAN result")
			}
			items := len(hashFieldVals)
			stats.Items += items

			for len(hashFieldVals) > 0 {
				var packSchedName, packStatJSON string
				hashFieldVals, err = redigo.Scan(hashFieldVals, &packSchedName, &packStatJSON)
				if err != nil {
					return nil, nil, ctxerr.Wrap(ctx, err, "scan HSCAN field and value")
				}

				// decode the pack name and scheduled query name from the field
				parts := strings.SplitN(packSchedName, "\x00", 2)
				if len(parts) != 2 {
					return nil, nil, ctxerr.Errorf(ctx, "invalid format for hash field: %s", packSchedName)
				}
				schedQueryNames = append(schedQueryNames, [2]string{parts[0], parts[1]})

				// unmarshal the pack stats
				var ps fleet.PackStats
				if err := json.Unmarshal([]byte(packStatJSON), &ps); err != nil {
					return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal pack stats hash value")
				}
				packStats = append(packStats, ps)
			}
			if cursor == 0 {
				// iteration completed
				return packStats, schedQueryNames, nil
			}
		}
	}

	// get all hosts' stats
	// index the pack + scheduled query names
	// load scheduled query IDs
	// batch upsert the stats

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

		insertBatch := make([][2]uint, 0, cfg.InsertBatch)
		deleteBatch := make([][2]uint, 0, cfg.DeleteBatch)
		for _, host := range hosts {
			hid := host.HostID
			ins, del, err := getKeyTuples(hid)
			if err != nil {
				return err
			}
			insertBatch = append(insertBatch, ins...)
			deleteBatch = append(deleteBatch, del...)

			if len(insertBatch) >= cfg.InsertBatch {
				if err := runInsertBatch(insertBatch); err != nil {
					return err
				}
				insertBatch = insertBatch[:0]
			}
			if len(deleteBatch) >= cfg.DeleteBatch {
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

			ts := t.clock.Now()
			updateBatch := make([]uint, cfg.UpdateBatch)
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

	if len(hosts) > 0 {
		// batch-remove any host ID from the active set that still has its score to
		// the initial value, so that the active set does not keep all (potentially
		// 100K+) host IDs to process at all times - only those with reported
		// results to process.
		if _, err := removeProcessedHostIDs(pool, scheduledQueryStatsHostIDsKey, hosts); err != nil {
			return ctxerr.Wrap(ctx, err, "remove processed host ids")
		}
	}
	return nil
}

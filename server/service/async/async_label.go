package async

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	labelMembershipActiveHostIDsKey = "label_membership:active_host_ids"
	labelMembershipHostKey          = "label_membership:{%d}"
	labelMembershipReportedKey      = "label_membership_reported:{%d}"
	labelMembershipKeysMinTTL       = 7 * 24 * time.Hour // 1 week
)

func (t *Task) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
	cfg := t.taskConfigs[config.AsyncTaskLabelMembership]
	if !cfg.Enabled {
		host.LabelUpdatedAt = ts
		return t.datastore.RecordLabelQueryExecutions(ctx, host, results, ts, deferred)
	}

	keySet := fmt.Sprintf(labelMembershipHostKey, host.ID)
	keyTs := fmt.Sprintf(labelMembershipReportedKey, host.ID)

	// set an expiration on both keys (set and ts), ensuring that a deleted host
	// (eventually) does not use any redis space. Ensure that TTL is reasonably
	// big to avoid deleting information that hasn't been collected yet - 1 week
	// or 10 * the collector interval, whichever is biggest.
	//
	// This means that it will only expire if that host hasn't reported labels
	// during that (TTL) time (each time it does report, the TTL is reset), and
	// the collector will have plenty of time to run (multiple times) to try to
	// persist all the data in mysql.
	ttl := labelMembershipKeysMinTTL
	if maxTTL := 10 * cfg.CollectInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// keys and arguments passed to the script are:
	// KEYS[1]: keySet (labelMembershipHostKey)
	// KEYS[2]: keyTs  (labelMembershipReportedKey)
	// ARGV[1]: timestamp for "reported at"
	// ARGV[2]: ttl for both keys
	// ARGV[3..]: the arguments to ZADD to keySet
	script := redigo.NewScript(2, `
    redis.call('ZADD', KEYS[1], unpack(ARGV, 3))
    redis.call('EXPIRE', KEYS[1], ARGV[2])
    redis.call('SET', KEYS[2], ARGV[1])
    return redis.call('EXPIRE', KEYS[2], ARGV[2])
  `)

	// convert results to ZADD arguments, store as -1 for delete, +1 for insert
	args := make(redigo.Args, 0, 4+(len(results)*2))
	args = args.Add(keySet, keyTs, ts.Unix(), int(ttl.Seconds()))
	for k, v := range results {
		score := -1
		if v != nil && *v {
			score = 1
		}
		args = args.Add(score, k)
	}

	conn := t.pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.pool, conn, keySet, keyTs); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}

	// Storing the host id in the set of active host IDs for label membership
	// outside of the redis script because in Redis Cluster mode the key may not
	// live on the same node as the host's keys. At the same time, purge any
	// entry in the set that is older than now - TTL.
	if _, err := storePurgeActiveHostID(t.pool, labelMembershipActiveHostIDsKey, host.ID, ts, ts.Add(-ttl)); err != nil {
		return ctxerr.Wrap(ctx, err, "store active host id")
	}
	return nil
}

func (t *Task) collectLabelQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	cfg := t.taskConfigs[config.AsyncTaskLabelMembership]

	hosts, err := loadActiveHostIDs(pool, labelMembershipActiveHostIDsKey, cfg.RedisScanKeysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load active host ids")
	}
	stats.Keys = len(hosts)

	getKeyTuples := func(hostID uint) (inserts, deletes [][2]uint, err error) {
		keySet := fmt.Sprintf(labelMembershipHostKey, hostID)
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		for {
			stats.RedisCmds++

			vals, err := redigo.Ints(conn.Do("ZPOPMIN", keySet, cfg.RedisPopCount))
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "redis ZPOPMIN")
			}
			items := len(vals) / 2 // each item has the label id and the score (-1=delete, +1=insert)
			stats.Items += items

			for i := 0; i < len(vals); i += 2 {
				labelID := vals[i]

				var score int
				if i+1 < len(vals) { // just to be safe we received all pairs
					score = vals[i+1]
				}

				switch score {
				case 1:
					inserts = append(inserts, [2]uint{uint(labelID), hostID}) //nolint:gosec // dismiss G115
				case -1:
					deletes = append(deletes, [2]uint{uint(labelID), hostID}) //nolint:gosec // dismiss G115
				}
			}
			if items < cfg.RedisPopCount {
				return inserts, deletes, nil
			}
		}
	}

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

	return nil
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	cfg := t.taskConfigs[config.AsyncTaskLabelMembership]

	if cfg.Enabled {
		conn := redis.ConfigureDoer(t.pool, t.pool.Get())
		defer conn.Close()

		key := fmt.Sprintf(labelMembershipReportedKey, host.ID)
		epoch, err := redigo.Int64(conn.Do("GET", key))
		if err == nil {
			if reported := time.Unix(epoch, 0); reported.After(host.LabelUpdatedAt) {
				return reported
			}
		}
	}
	return host.LabelUpdatedAt
}

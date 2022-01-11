package async

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	labelMembershipActiveHostIDsKey = "label_membership:active_host_ids"
	labelMembershipHostKey          = "label_membership:{%d}"
	labelMembershipReportedKey      = "label_membership_reported:{%d}"
	labelMembershipKeysMinTTL       = 7 * 24 * time.Hour // 1 week
	collectorLockKey                = "locks:async_collector:{%s}"
)

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

	labelColl := &collector{
		name:         "collect_labels",
		pool:         t.Pool,
		ds:           t.Datastore,
		execInterval: t.CollectorInterval,
		jitterPct:    jitterPct,
		lockTimeout:  t.LockTimeout,
		handler:      t.collectLabelQueryExecutions,
		errHandler: func(name string, err error) {
			level.Error(logger).Log("err", fmt.Sprintf("%s collector", name), "details", err)
		},
	}
	go labelColl.Start(ctx)

	// log stats at regular intervals
	if t.LogStatsInterval > 0 {
		go func() {
			tick := time.Tick(t.LogStatsInterval)
			for {
				select {
				case <-tick:
					stats := labelColl.ReadStats()
					level.Debug(logger).Log("stats", fmt.Sprintf("%#v", stats), "name", labelColl.name)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func (t *Task) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
	if !t.AsyncEnabled {
		host.LabelUpdatedAt = ts
		return t.Datastore.RecordLabelQueryExecutions(ctx, host, results, ts, false)
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
	if maxTTL := 10 * t.CollectorInterval; maxTTL > ttl {
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

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, keySet, keyTs); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}

	// Storing the host id in the set of active host IDs for label membership
	// outside of the redis script because in Redis Cluster mode the key may not
	// live on the same node as the host's keys. At the same time, purge any
	// entry in the set that is older than now - TTL.
	if err := storePurgeActiveHostID(t.Pool, host.ID, ts, ts.Add(-ttl)); err != nil {
		return ctxerr.Wrap(ctx, err, "store active host id")
	}
	return nil
}

func (t *Task) collectLabelQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	hosts, err := loadActiveHostIDs(pool, t.RedisScanKeysCount)
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

			vals, err := redigo.Ints(conn.Do("ZPOPMIN", keySet, t.RedisPopCount))
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
					inserts = append(inserts, [2]uint{uint(labelID), hostID})
				case -1:
					deletes = append(deletes, [2]uint{uint(labelID), hostID})
				}
			}
			if items < t.RedisPopCount {
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
		if err := removeProcessedHostIDs(pool, hosts); err != nil {
			return ctxerr.Wrap(ctx, err, "remove processed host ids")
		}
	}

	return nil
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := redis.ConfigureDoer(t.Pool, t.Pool.Get())
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

func storePurgeActiveHostID(pool fleet.RedisPool, hid uint, reportedAt, purgeOlder time.Time) error {
	// KEYS[1]: labelMembershipActiveHostIDsKey
	// ARGV[1]: the host ID to add
	// ARGV[2]: the added host's reported-at timestamp
	// ARGV[3]: purge any entry with score older than this (purgeOlder timestamp)
	script := redigo.NewScript(1, `
    redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
    return redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[3])
  `)

	conn := pool.Get()
	defer conn.Close()

	if err := redis.BindConn(pool, conn, labelMembershipActiveHostIDsKey); err != nil {
		return fmt.Errorf("bind redis connection: %w", err)
	}

	if _, err := script.Do(conn, labelMembershipActiveHostIDsKey, hid, reportedAt.Unix(), purgeOlder.Unix()); err != nil {
		return fmt.Errorf("run redis script: %w", err)
	}
	return nil
}

func removeProcessedHostIDs(pool fleet.RedisPool, batch []hostIDLastReported) error {
	// This script removes from the set of active hosts for label membership all
	// those that still have the same score as when the batch was read (via
	// loadActiveHostIDs). This is so that any host that would've reported new
	// data since the call to loadActiveHostIDs would *not* get deleted (as the
	// score would change if that was the case).
	//
	// Note that this approach is correct - in that it is safe and won't delete
	// any host that has unsaved reported data - but it is potentially slow, as
	// it needs to check the score of each member before deleting it. Should that
	// become too slow, we have some options:
	//
	// * split the batch in smaller, capped ones (that would be if the redis
	//   server gets blocked for too long processing a single batch)
	// * use ZREMRANGEBYSCORE to remove in one command all members with a score
	//   (reported-at timestamp) lower than the maximum timestamp in batch.
	//   While this would be almost certainly faster, it might be incorrect as
	//   new data could be reported with timestamps older than the maximum one,
	//   e.g. if the clocks are not exactly in sync between fleet instances, or
	//   if hosts report new data while the ZSCAN is going on and don't get picked
	//   up by the SCAN (this is possible, as part of the guarantees of SCAN).

	// KEYS[1]: labelMembershipActiveHostIDsKey
	// ARGV...: the list of host ID-last reported timestamp pairs
	script := redigo.NewScript(1, `
    local count = 0
    for i = 1, #ARGV, 2 do
      local member, ts = ARGV[i], ARGV[i+1]
      if redis.call('ZSCORE', KEYS[1], member) == ts then
        count = count + 1
        redis.call('ZREM', KEYS[1], member)
      end
    end
    return count
  `)

	conn := pool.Get()
	defer conn.Close()

	if err := redis.BindConn(pool, conn, labelMembershipActiveHostIDsKey); err != nil {
		return fmt.Errorf("bind redis connection: %w", err)
	}

	args := redigo.Args{labelMembershipActiveHostIDsKey}
	for _, host := range batch {
		args = args.Add(host.HostID, host.LastReported)
	}
	if _, err := script.Do(conn, args...); err != nil {
		return fmt.Errorf("run redis script: %w", err)
	}
	return nil
}

type hostIDLastReported struct {
	HostID       uint
	LastReported int64 // timestamp in unix epoch
}

func loadActiveHostIDs(pool fleet.RedisPool, scanCount int) ([]hostIDLastReported, error) {
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	// using ZSCAN instead of fetching in one shot, as there may be 100K+ hosts
	// and we don't want to block the redis server too long.
	var hosts []hostIDLastReported
	cursor := 0
	for {
		res, err := redigo.Values(conn.Do("ZSCAN", labelMembershipActiveHostIDsKey, cursor, "COUNT", scanCount))
		if err != nil {
			return nil, fmt.Errorf("scan active host ids: %w", err)
		}
		var hostVals []uint
		if _, err := redigo.Scan(res, &cursor, &hostVals); err != nil {
			return nil, fmt.Errorf("convert scan results: %w", err)
		}
		for i := 0; i < len(hostVals); i += 2 {
			hosts = append(hosts, hostIDLastReported{HostID: hostVals[i], LastReported: int64(hostVals[i+1])})
		}

		if cursor == 0 {
			// iteration completed
			return hosts, nil
		}
	}
}

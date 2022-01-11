package async

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
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

	// TODO(mna): regarding migrations of previous version to this one, first of
	// all I believe we don't know of any user that activated the feature, but in
	// any case, given that we use the same key names, just that we maintain an
	// additional set of sets, there is no migration required - when the hosts
	// start reporting new results, we will update their existing keys, set the
	// new TTLs and insert the host IDs in the set of sets, resulting in the
	// pending data being collected. The only edge case would be if that host
	// never checks in again, those keys would stick around (no TTL set before).

	keySet := fmt.Sprintf(labelMembershipHostKey, host.ID)
	keyTs := fmt.Sprintf(labelMembershipReportedKey, host.ID)

	// TODO(mna): set an expiration on both keys (set and ts), ensuring that a
	// deleted host (eventually) does not use any redis space. Ensure that TTL
	// is reasonably big to avoid deleting information that hasn't been collected
	// yet - e.g. 1 week or 10 * the collector interval, whichever is biggest.
	// Or should it be a multiple of osquery.label_update_interval?
	//
	// This means that it will only expire if that host hasn't reported labels
	// during that (TTL) time (each time it does report, the TTL is reset), and
	// the collector will have plenty of time to run (multiple times) to try to
	// persist all the data in mysql.

	ttl := labelMembershipKeysMinTTL
	if maxTTL := 10 * t.CollectorInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// TODO(mna): also add the host ID to the "set of sets", to avoid a SCAN KEYS.
	// (outside the script, as in Redis Cluster the key may not live on the same node).

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
	return nil
}

// TODO(mna): not needed anymore with the "set of sets"
var reHostFromKey = regexp.MustCompile(`\{(\d+)\}$`)

func (t *Task) collectLabelQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	// TODO(mna): scan from the "set of sets" instead, that contains all host IDs
	// to collect.

	labelMembershipHostKeyPattern := "label_membership:{*}"
	keys, err := redis.ScanKeys(pool, labelMembershipHostKeyPattern, t.RedisScanKeysCount)
	if err != nil {
		return err
	}
	stats.Keys = len(keys)

	getKeyTuples := func(key string) (hostID uint, inserts, deletes [][2]uint, err error) {
		if matches := reHostFromKey.FindStringSubmatch(key); matches != nil {
			id, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && id > 0 && id <= math.MaxUint32 { // required for CodeQL vulnerability scanning in CI
				hostID = uint(id)
			}
		}

		// just ignore if there is no valid host id
		if hostID == 0 {
			return hostID, nil, nil, nil
		}

		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		for {
			stats.RedisCmds++

			vals, err := redigo.Ints(conn.Do("ZPOPMIN", key, t.RedisPopCount))
			if err != nil {
				return hostID, nil, nil, ctxerr.Wrap(ctx, err, "redis ZPOPMIN")
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
				return hostID, inserts, deletes, nil
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
	hostIDs := make([]uint, 0, len(keys))
	for _, key := range keys { // TODO(mna): range over host IDs, not keys
		hostID, ins, del, err := getKeyTuples(key)
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
		if hostID > 0 {
			hostIDs = append(hostIDs, hostID)
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
	if len(hostIDs) > 0 {
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

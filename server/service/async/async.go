package async

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	labelMembershipHostKeyPattern = "label_membership:{*}"
	labelMembershipHostKey        = "label_membership:{%d}"
	labelMembershipReportedKey    = "label_membership_reported:{%d}"
	policyPassHostKeyPattern      = "policy_pass:{*}"
	policyPassHostKey             = "policy_pass:{%d}"
	policyPassReportedKey         = "policy_pass_reported:{%d}"
	collectorLockKey              = "locks:async_collector:{%s}"
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
}

// Collect runs the various collectors as distinct background goroutines if
// async processing is enabled.  Each collector will stop processing when ctx
// is done.
func (t *Task) StartCollectors(ctx context.Context, interval time.Duration, jitterPct int, logger kitlog.Logger) {
	if !t.AsyncEnabled {
		level.Debug(logger).Log("task", "async disabled, not starting collectors")
		return
	}
	level.Debug(logger).Log("task", "async enabled, starting collectors", "interval", interval, "jitter", jitterPct)

	labelColl := &collector{
		name:         "collect_labels",
		pool:         t.Pool,
		ds:           t.Datastore,
		execInterval: interval,
		jitterPct:    jitterPct,
		lockTimeout:  t.LockTimeout,
		handler:      t.collectLabelQueryExecutions,
		errHandler: func(name string, err error) {
			level.Error(logger).Log("err", fmt.Sprintf("%s collector", name), "details", err)
		},
	}
	go labelColl.Start(ctx)

	policyColl := &collector{
		name:         "collect_policies",
		pool:         t.Pool,
		ds:           t.Datastore,
		execInterval: interval,
		jitterPct:    jitterPct,
		lockTimeout:  time.Minute,
		handler:      t.collectPolicyQueryExecutions,
		errHandler: func(name string, err error) {
			level.Error(logger).Log("err", fmt.Sprintf("%s collector", name), "details", err)
		},
	}
	go policyColl.Start(ctx)

	// log stats at regular intervals
	if t.LogStatsInterval > 0 {
		go func() {
			tick := time.Tick(t.LogStatsInterval)
			for {
				select {
				case <-tick:
					stats := labelColl.ReadStats()
					level.Debug(logger).Log("stats", fmt.Sprintf("%#v", stats), "name", labelColl.name)
					stats = policyColl.ReadStats()
					level.Debug(logger).Log("stats", fmt.Sprintf("%#v", stats), "name", policyColl.name)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

var (
	// redis list will be LTRIM'd if there are more policy IDs than this.
	maxRedisPolicyResultsPerHost = 1000
)

func (t *Task) RecordPolicyQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
	if !t.AsyncEnabled {
		host.PolicyUpdatedAt = ts
		return t.Datastore.RecordPolicyQueryExecutions(ctx, host, results, ts)
	}

	keyList := fmt.Sprintf(policyPassHostKey, host.ID)
	keyTs := fmt.Sprintf(policyPassReportedKey, host.ID)

	script := redigo.NewScript(2, `
    redis.call('LPUSH', KEYS[1], unpack(ARGV, 3))
    redis.call('LTRIM', KEYS[1], 0, ARGV[2])
    return redis.call('SET', KEYS[2], ARGV[1])
  `)

	// convert results to LPUSH arguments, store as policy_id=1 for pass,
	// policy_id=-1 for fail.
	args := make(redigo.Args, 0, 4+len(results))
	args = args.Add(keyList, keyTs, ts.Unix(), maxRedisPolicyResultsPerHost) // TODO: const or config for max items
	for k, v := range results {
		pass := -1
		if v != nil && *v {
			pass = 1
		}
		args = args.Add(fmt.Sprintf("%d=%d", k, pass))
	}

	conn := t.Pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.Pool, conn, keyList, keyTs); err != nil {
		return errors.Wrap(err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return err
	}
	return nil
}

func (t *Task) collectPolicyQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	keys, err := redis.ScanKeys(pool, policyPassHostKeyPattern, t.RedisScanKeysCount)
	if err != nil {
		return err
	}
	stats.Keys = len(keys)

	type policyTuple struct {
		HostID   uint
		PolicyID uint
		Passes   bool
	}

	// need to use a script as the RPOP command only supports a COUNT since
	// 6.2. Because we use LTRIM when inserting, we know the total number
	// of results is at most maxRedisPolicyResultsPerHost, so it is capped
	// and can be returned in one go.
	script := redigo.NewScript(1, `
    local res = redis.call('LRANGE', KEYS[1], 0, -1)
    redis.call('DEL', KEYS[1])
    return res
  `)

	getKeyTuples := func(key string) (inserts []policyTuple, err error) {
		var hostID uint
		if matches := reHostFromKey.FindStringSubmatch(key); matches != nil {
			if id, _ := strconv.ParseUint(matches[1], 10, 32); id > 0 {
				hostID = uint(id)
			}
		}

		// just ignore if there is no valid host id
		if hostID == 0 {
			return nil, nil
		}

		conn := pool.ConfigureDoer(pool.Get())
		defer conn.Close()

		stats.RedisCmds++
		res, err := redigo.Strings(script.Do(conn, key))
		if err != nil {
			return nil, errors.Wrap(err, "redis LRANGE script")
		}

		inserts = make([]policyTuple, 0, len(res))
		stats.Items += len(res)
		for _, item := range res {
			parts := strings.Split(item, "=")
			if len(parts) != 2 {
				continue
			}

			var tup policyTuple
			if id, _ := strconv.ParseUint(parts[0], 10, 32); id > 0 {
				tup.HostID = hostID
				tup.PolicyID = uint(id)
				switch parts[1] {
				case "1":
					tup.Passes = true
				case "-1":
					tup.Passes = false
				default:
					continue
				}
				inserts = append(inserts, tup)
			}
		}
		return inserts, nil
	}

	runInsertBatch := func(batch []policyTuple) error {
		stats.Inserts++

		// TODO: INSERT IGNORE, to avoid failing if policy id does not exist? Or this should
		// never happen as policies cannot come and go like labels do?
		sql := `INSERT INTO policy_membership_history (policy_id, host_id, passes) VALUES `
		sql += strings.Repeat(`(?, ?, ?),`, len(batch))
		sql = strings.TrimSuffix(sql, ",")

		vals := make([]interface{}, 0, len(batch)*3)
		for _, tup := range batch {
			vals = append(vals, tup.PolicyID, tup.HostID, tup.Passes)
		}
		return ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, sql, vals...)
			return errors.Wrap(err, "insert into policy_membership_history")
		})
	}

	runUpdateBatch := func(ids []uint, ts time.Time) error {
		stats.Updates++

		sql := `
      UPDATE
        hosts
      SET
        policy_updated_at = ?
      WHERE
        id IN (?)`
		query, args, err := sqlx.In(sql, ts, ids)
		if err != nil {
			return errors.Wrap(err, "building query to update hosts.policy_updated_at")
		}
		return ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, query, args...)
			return errors.Wrap(err, "update hosts.policy_updated_at")
		})
	}

	insertBatch := make([]policyTuple, 0, t.InsertBatch)
	hostIDs := make([]uint, 0, len(keys))
	for _, key := range keys {
		ins, err := getKeyTuples(key)
		if err != nil {
			return err
		}
		insertBatch = append(insertBatch, ins...)

		if len(insertBatch) >= t.InsertBatch {
			if err := runInsertBatch(insertBatch); err != nil {
				return err
			}
			insertBatch = insertBatch[:0]
		}
		if len(ins) > 0 {
			hostIDs = append(hostIDs, ins[0].HostID)
		}
	}

	// process any remaining batch that did not reach the batchSize limit in the
	// loop.
	if len(insertBatch) > 0 {
		if err := runInsertBatch(insertBatch); err != nil {
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

func (t *Task) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
	if !t.AsyncEnabled {
		host.LabelUpdatedAt = ts
		return t.Datastore.RecordLabelQueryExecutions(ctx, host, results, ts)
	}

	keySet := fmt.Sprintf(labelMembershipHostKey, host.ID)
	keyTs := fmt.Sprintf(labelMembershipReportedKey, host.ID)

	script := redigo.NewScript(2, `
    redis.call('ZADD', KEYS[1], unpack(ARGV, 2))
    return redis.call('SET', KEYS[2], ARGV[1])
  `)

	// convert results to ZADD arguments, store as -1 for delete, +1 for insert
	args := make(redigo.Args, 0, 3+(len(results)*2))
	args = args.Add(keySet, keyTs, ts.Unix())
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
		return errors.Wrap(err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return err
	}
	return nil
}

var reHostFromKey = regexp.MustCompile(`\{(\d+)\}$`)

func (t *Task) collectLabelQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
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

		conn := pool.ConfigureDoer(pool.Get())
		defer conn.Close()

		for {
			stats.RedisCmds++

			vals, err := redigo.Ints(conn.Do("ZPOPMIN", key, t.RedisPopCount))
			if err != nil {
				return hostID, nil, nil, errors.Wrap(err, "redis ZPOPMIN")
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

		sql := `INSERT INTO label_membership (label_id, host_id) VALUES `
		sql += strings.Repeat(`(?, ?),`, len(batch))
		sql = strings.TrimSuffix(sql, ",")
		sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

		vals := make([]interface{}, 0, len(batch)*2)
		for _, tup := range batch {
			vals = append(vals, tup[0], tup[1])
		}
		return ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, sql, vals...)
			return errors.Wrap(err, "insert into label_membership")
		})
	}

	runDeleteBatch := func(batch [][2]uint) error {
		stats.Deletes++

		rest := strings.Repeat(`UNION ALL SELECT ?, ? `, len(batch)-1)
		sql := fmt.Sprintf(`
    DELETE
      lm
    FROM
      label_membership lm
    JOIN
      (SELECT ? label_id, ? host_id %s) del_list
    ON
      lm.label_id = del_list.label_id AND
      lm.host_id = del_list.host_id`, rest)

		vals := make([]interface{}, 0, len(batch)*2)
		for _, tup := range batch {
			vals = append(vals, tup[0], tup[1])
		}
		return ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, sql, vals...)
			return errors.Wrap(err, "delete from label_membership")
		})
	}

	runUpdateBatch := func(ids []uint, ts time.Time) error {
		stats.Updates++

		sql := `
      UPDATE
        hosts
      SET
        label_updated_at = ?
      WHERE
        id IN (?)`
		query, args, err := sqlx.In(sql, ts, ids)
		if err != nil {
			return errors.Wrap(err, "building query to update hosts.label_updated_at")
		}
		return ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, query, args...)
			return errors.Wrap(err, "update hosts.label_updated_at")
		})
	}

	insertBatch := make([][2]uint, 0, t.InsertBatch)
	deleteBatch := make([][2]uint, 0, t.DeleteBatch)
	hostIDs := make([]uint, 0, len(keys))
	for _, key := range keys {
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

func (t *Task) GetHostPolicyReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := t.Pool.ConfigureDoer(t.Pool.Get())
		defer conn.Close()

		key := fmt.Sprintf(policyPassReportedKey, host.ID)
		epoch, err := redigo.Int64(conn.Do("GET", key))
		if err == nil {
			if reported := time.Unix(epoch, 0); reported.After(host.PolicyUpdatedAt) {
				return reported
			}
		}
	}
	return host.PolicyUpdatedAt
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := t.Pool.ConfigureDoer(t.Pool.Get())
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

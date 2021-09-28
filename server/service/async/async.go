package async

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

const (
	labelMembershipHostKeyPattern = "label_membership:{*}"
	labelMembershipHostKey        = "label_membership:{%d}"
	labelMembershipReportedKey    = "label_membership_reported:{%d}"
	collectorLockKey              = "locks:async_collector:{%s}"
)

type Task struct {
	Datastore fleet.Datastore
	Pool      fleet.RedisPool
	// AsyncEnabled indicates if async processing is enabled in the
	// configuration. Note that Pool can be nil if this is false.
	AsyncEnabled bool // TODO: should this be read in a different way, more dynamically, if config changes while fleet is running? Or does that require a restart?
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

func collectLabelQueryExecutions(ctx context.Context, pool fleet.RedisPool, stats *collectorExecStats) error {
	keys, err := redis.ScanKeys(pool, labelMembershipHostKeyPattern, 1000)
	if err != nil {
		return err
	}
	stats.Keys = len(keys)

	getKeyTuples := func(key string) (inserts, deletes [][2]uint, err error) {
		var hostID uint
		if matches := reHostFromKey.FindStringSubmatch(key); matches != nil {
			id, err := strconv.ParseUint(matches[1], 10, 64)
			if err == nil {
				hostID = uint(id)
			}
		}

		// just ignore if there is no valid host id
		if hostID == 0 {
			return nil, nil, nil
		}

		conn := pool.ConfigureDoer(pool.Get())
		defer conn.Close()

		const popCount = 1000
		for {
			vals, err := redigo.Ints(conn.Do("ZPOPMIN", key, "COUNT", popCount))
			if err != nil {
				return nil, nil, err
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
			if items < popCount {
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
	const batchSize = 10000

	insertBatch := make([][2]uint, 0, batchSize)
	deleteBatch := make([][2]uint, 0, batchSize)
	for _, key := range keys {
		ins, del, err := getKeyTuples(key)
		if err != nil {
			return err
		}
		insertBatch = append(insertBatch, ins...)
		deleteBatch = append(deleteBatch, del...)

		if len(insertBatch) >= batchSize {
			insertBatch = insertBatch[:0]
		}
		if len(deleteBatch) >= batchSize {
			deleteBatch = deleteBatch[:0]
		}
	}

	// process any remaining batch that did not reach the batchSize limit in the
	// loop.
	if len(insertBatch) > 0 {
	}
	if len(deleteBatch) > 0 {
	}

	return nil
}

func (t *Task) GetHostLabelReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := t.Pool.ConfigureDoer(t.Pool.Get())
		defer conn.Close()

		key := fmt.Sprintf(labelMembershipReportedKey, host.ID)
		epoch, err := redigo.Int64(conn.Do("GET", key))
		if err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return host.LabelUpdatedAt
}

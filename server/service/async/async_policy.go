package async

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

const (
	policyPassHostIDsKey  = "policy_pass:active_host_ids"
	policyPassHostKey     = "policy_pass:{%d}"
	policyPassReportedKey = "policy_pass_reported:{%d}"
	policyPassKeysMinTTL  = 7 * 24 * time.Hour // 1 week
)

var (
	// redis list will be LTRIM'd if there are more policy IDs than this.
	maxRedisPolicyResultsPerHost = 1000
)

func (t *Task) RecordPolicyQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
	if !t.AsyncEnabled {
		host.PolicyUpdatedAt = ts
		return t.Datastore.RecordPolicyQueryExecutions(ctx, host, results, ts, deferred)
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
	args = args.Add(keyList, keyTs, ts.Unix(), maxRedisPolicyResultsPerHost)
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
	policyPassHostKeyPattern := "policy_pass:{*}"
	keys, err := redis.ScanKeys(pool, policyPassHostKeyPattern, t.RedisScanKeysCount)
	if err != nil {
		return err
	}
	stats.Keys = len(keys)

	// need to use a script as the RPOP command only supports a COUNT since
	// 6.2. Because we use LTRIM when inserting, we know the total number
	// of results is at most maxRedisPolicyResultsPerHost, so it is capped
	// and can be returned in one go.
	script := redigo.NewScript(1, `
    local res = redis.call('LRANGE', KEYS[1], 0, -1)
    redis.call('DEL', KEYS[1])
    return res
  `)

	var reHostFromKey = regexp.MustCompile(`\{(\d+)\}$`)
	getKeyTuples := func(key string) (inserts []fleet.PolicyMembershipResult, err error) {
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

		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		stats.RedisCmds++
		res, err := redigo.Strings(script.Do(conn, key))
		if err != nil {
			return nil, errors.Wrap(err, "redis LRANGE script")
		}

		inserts = make([]fleet.PolicyMembershipResult, 0, len(res))
		stats.Items += len(res)
		for _, item := range res {
			parts := strings.Split(item, "=")
			if len(parts) != 2 {
				continue
			}

			var tup fleet.PolicyMembershipResult
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

	runInsertBatch := func(batch []fleet.PolicyMembershipResult) error {
		stats.Inserts++
		return ds.AsyncBatchInsertPolicyMembership(ctx, batch)
	}

	runUpdateBatch := func(ids []uint, ts time.Time) error {
		stats.Updates++
		return ds.AsyncBatchUpdatePolicyTimestamp(ctx, ids, ts)
	}

	insertBatch := make([]fleet.PolicyMembershipResult, 0, t.InsertBatch)
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

func (t *Task) GetHostPolicyReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	if t.AsyncEnabled {
		conn := redis.ConfigureDoer(t.Pool, t.Pool.Get())
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

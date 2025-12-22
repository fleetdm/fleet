package async

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	redigo "github.com/gomodule/redigo/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	policyPassHostIDsKey  = "policy_pass:active_host_ids"
	policyPassHostKey     = "policy_pass:{%d}"
	policyPassReportedKey = "policy_pass_reported:{%d}"
	policyPassKeysMinTTL  = 7 * 24 * time.Hour // 1 week
)

// redis list will be LTRIM'd if there are more policy IDs than this.
var maxRedisPolicyResultsPerHost = 1000

func (t *Task) RecordPolicyQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
	cfg := t.taskConfigs[config.AsyncTaskPolicyMembership]
	if !cfg.Enabled {
		host.PolicyUpdatedAt = ts
		return t.datastore.RecordPolicyQueryExecutions(ctx, host, results, ts, deferred)
	}

	keyList := fmt.Sprintf(policyPassHostKey, host.ID)
	keyTs := fmt.Sprintf(policyPassReportedKey, host.ID)

	// set an expiration on both keys (list and ts), ensuring that a deleted host
	// (eventually) does not use any redis space. Ensure that TTL is reasonably
	// big to avoid deleting information that hasn't been collected yet - 1 week
	// or 10 * the collector interval, whichever is biggest.
	//
	// This means that it will only expire if that host hasn't reported policies
	// during that (TTL) time (each time it does report, the TTL is reset), and
	// the collector will have plenty of time to run (multiple times) to try to
	// persist all the data in mysql.
	ttl := policyPassKeysMinTTL
	if maxTTL := 10 * cfg.CollectInterval; maxTTL > ttl {
		ttl = maxTTL
	}

	// There are two versions of the script (1) and (2).
	// Script (1) is used when the are no policy results to report and
	// script (2) is used when there are policy results.

	// (1)
	// KEYS[1]: keyTs (policyPassReportedKey)
	// ARGV[1]: timestamp for "reported at"
	// ARGV[2]: ttl for the key
	scriptSrc := `
		redis.call('SET', KEYS[1], ARGV[1])
		return redis.call('EXPIRE', KEYS[1], ARGV[2])
	`
	keyCount := 1
	args := make(redigo.Args, 0, 3)
	args = args.Add(keyTs, ts.Unix(), int(ttl.Seconds()))

	if len(results) > 0 {
		// (2)
		// KEYS[1]: keyList (policyPassHostKey)
		// KEYS[2]: keyTs (policyPassReportedKey)
		// ARGV[1]: timestamp for "reported at"
		// ARGV[2]: max policy results to keep per host (list is trimmed to that size)
		// ARGV[3]: ttl for both keys
		// ARGV[4..]: policy_id=pass entries to LPUSH to the list
		keyCount = 2
		scriptSrc = `
		redis.call('LPUSH', KEYS[1], unpack(ARGV, 4))
		redis.call('LTRIM', KEYS[1], 0, ARGV[2])
		redis.call('EXPIRE', KEYS[1], ARGV[3])
		redis.call('SET', KEYS[2], ARGV[1])
		return redis.call('EXPIRE', KEYS[2], ARGV[3])
		`
		// convert results to LPUSH arguments, store as policy_id=1 for pass,
		// policy_id=-1 for fail, policy_id=0 for null result.
		args = make(redigo.Args, 0, 5+len(results))
		args = args.Add(keyList, keyTs, ts.Unix(), maxRedisPolicyResultsPerHost, int(ttl.Seconds()))
		for k, v := range results {
			pass := 0
			if v != nil {
				if *v {
					pass = 1
				} else {
					pass = -1
				}
			}
			args = args.Add(fmt.Sprintf("%d=%d", k, pass))
		}
	}

	script := redigo.NewScript(keyCount, scriptSrc)

	conn := t.pool.Get()
	defer conn.Close()
	if err := redis.BindConn(t.pool, conn, keyList, keyTs); err != nil {
		return ctxerr.Wrap(ctx, err, "bind redis connection")
	}

	if _, err := script.Do(conn, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "run redis script")
	}

	// Storing the host id in the set of active host IDs for policy membership
	// outside of the redis script because in Redis Cluster mode the key may not
	// live on the same node as the host's keys. At the same time, purge any
	// entry in the set that is older than now - TTL.
	if _, err := storePurgeActiveHostID(t.pool, policyPassHostIDsKey, host.ID, ts, ts.Add(-ttl)); err != nil {
		return ctxerr.Wrap(ctx, err, "store active host id")
	}
	return nil
}

func (t *Task) collectPolicyQueryExecutions(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
	// Create a root span for this async collection task if OTEL is enabled
	if t.otelEnabled {
		tracer := otel.Tracer("async")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "async.collect_policy_query_executions",
			trace.WithAttributes(
				attribute.String("async.task", "policy_membership"),
			),
		)
		defer span.End()
	}

	cfg := t.taskConfigs[config.AsyncTaskPolicyMembership]

	hosts, err := loadActiveHostIDs(pool, policyPassHostIDsKey, cfg.RedisScanKeysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load active host ids")
	}
	stats.Keys = len(hosts)

	// need to use a script as the RPOP command only supports a COUNT since
	// 6.2. Because we use LTRIM when inserting, we know the total number
	// of results is at most maxRedisPolicyResultsPerHost, so it is capped
	// and can be returned in one go.
	script := redigo.NewScript(1, `
    local res = redis.call('LRANGE', KEYS[1], 0, -1)
    redis.call('DEL', KEYS[1])
    return res
  `)

	getKeyTuples := func(hostID uint) (inserts []fleet.PolicyMembershipResult, err error) {
		keyList := fmt.Sprintf(policyPassHostKey, hostID)
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		stats.RedisCmds++
		res, err := redigo.Strings(script.Do(conn, keyList))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "redis LRANGE script")
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
					tup.Passes = ptr.Bool(true)
				case "-1":
					tup.Passes = ptr.Bool(false)
				case "0":
					tup.Passes = nil
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

	insertBatch := make([]fleet.PolicyMembershipResult, 0, cfg.InsertBatch)
	for _, host := range hosts {
		hid := host.HostID
		ins, err := getKeyTuples(hid)
		if err != nil {
			return err
		}
		insertBatch = append(insertBatch, ins...)

		if len(insertBatch) >= cfg.InsertBatch {
			if err := runInsertBatch(insertBatch); err != nil {
				return err
			}
			insertBatch = insertBatch[:0]
		}
	}

	// process any remaining batch that did not reach the batchSize limit in the
	// loop.
	if len(insertBatch) > 0 {
		if err := runInsertBatch(insertBatch); err != nil {
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
		if _, err := removeProcessedHostIDs(pool, policyPassHostIDsKey, hosts); err != nil {
			return ctxerr.Wrap(ctx, err, "remove processed host ids")
		}
	}

	return nil
}

func (t *Task) GetHostPolicyReportedAt(ctx context.Context, host *fleet.Host) time.Time {
	cfg := t.taskConfigs[config.AsyncTaskPolicyMembership]

	if cfg.Enabled {
		conn := redis.ConfigureDoer(t.pool, t.pool.Get())
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

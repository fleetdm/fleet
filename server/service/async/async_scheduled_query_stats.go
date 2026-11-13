package async

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	scheduledQueryStatsHostQueriesKey = "scheduled_query_stats:{%d}" // the hash of scheduled query stats for a given host
	scheduledQueryStatsHostIDsKey     = "scheduled_query_stats:active_host_ids"
	scheduledQueryStatsKeyMinTTL      = 7 * 24 * time.Hour // 1 week
)

// RecordScheduledQueryStats records the scheduled query stats for a given host.
func (t *Task) RecordScheduledQueryStats(ctx context.Context, teamID *uint, hostID uint, stats []fleet.PackStats, ts time.Time) error {
	cfg := t.taskConfigs[config.AsyncTaskScheduledQueryStats]
	if !cfg.Enabled {
		return t.datastore.SaveHostPackStats(ctx, teamID, hostID, stats)
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
	// packname\x00schedqueryname and the values are the json-marshaled
	// fleet.ScheduledQueryStats struct. Using this structure ensures the fields
	// are always overridden with the latest stats for the pack/query tuple,
	// which mirrors the behavior of the mysql table (we only keep the latest
	// stats).

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
		// Same rule as SaveHostPackStats: only the "Global" pack and the host's own
		// "team-<id>" pack carry its reports, so a host can't record stats for
		// another fleet's reports.
		if ps.PackName != "Global" && (teamID == nil || ps.PackName != fmt.Sprintf("team-%d", *teamID)) {
			continue
		}
		for _, qs := range ps.QueryStats {
			field := fmt.Sprintf("%s\x00%s", ps.PackName, qs.ScheduledQueryName)
			jsonStat, err := json.Marshal(qs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "marshal scheduled query stats")
			}
			args = args.Add(field, string(jsonStat))
		}
	}

	if len(args) > 2 {
		// only if there are fields to HSET
		conn := t.pool.Get()
		defer conn.Close()
		if err := redis.BindConn(t.pool, conn, key); err != nil {
			return ctxerr.Wrap(ctx, err, "bind redis connection")
		}

		if _, err := script.Do(conn, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "run redis script")
		}
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
	// Create a root span for this async collection task if OTEL is enabled
	if t.otelEnabled {
		tracer := otel.Tracer("async")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "async.collect_scheduled_query_stats",
			trace.WithAttributes(
				attribute.String("async.task", "scheduled_query_stats"),
			),
		)
		defer span.End()
	}

	cfg := t.taskConfigs[config.AsyncTaskScheduledQueryStats]

	hosts, err := loadActiveHostIDs(pool, scheduledQueryStatsHostIDsKey, cfg.RedisScanKeysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load active host ids")
	}
	stats.Keys = len(hosts)

	getHostStats := func(hostID uint) (sqStats []fleet.ScheduledQueryStats, schedQueryNames [][2]string, err error) {
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
			stats.Items += items / 2 // because keys/values are alternating

			for len(hashFieldVals) > 0 {
				var packSchedName, schedQueryStatJSON string
				hashFieldVals, err = redigo.Scan(hashFieldVals, &packSchedName, &schedQueryStatJSON)
				if err != nil {
					return nil, nil, ctxerr.Wrap(ctx, err, "scan HSCAN field and value")
				}

				// decode the pack name and scheduled query name from the field
				parts := strings.SplitN(packSchedName, "\x00", 2)
				if len(parts) != 2 {
					return nil, nil, ctxerr.Errorf(ctx, "invalid format for hash field: %s", packSchedName)
				}
				schedQueryNames = append(schedQueryNames, [2]string{parts[0], parts[1]})

				// unmarshal the scheduled query stats
				var sqStat fleet.ScheduledQueryStats
				if err := json.Unmarshal([]byte(schedQueryStatJSON), &sqStat); err != nil {
					return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal scheduled query stats hash value")
				}
				sqStat.PackName = parts[0]
				sqStats = append(sqStats, sqStat)
			}
			if cursor == 0 {
				// iteration completed, clear the hash but do not fail on error
				_, _ = conn.Do("DEL", keyHash)

				return sqStats, schedQueryNames, nil
			}
		}
	}

	// NOTE(mna): packs typically target many hosts, so it's very likely that the
	// batch of recorded stats have a small-ish set of packName+schedQueryName that
	// is repeated many times over. Instead of loading those report IDs for each
	// insertion of the stats, load them once before processing the batch.

	// get all hosts' stats and index the report names
	hostsStats := make(map[uint][]fleet.ScheduledQueryStats, len(hosts)) // key is host ID
	reportNames := make(map[string]fleet.TeamScopedQueryName)            // key is TeamScopedQueryName.Key()
	for _, host := range hosts {
		sqStats, names, err := getHostStats(host.HostID)
		if err != nil {
			return err
		}
		hostsStats[host.HostID] = sqStats
		for _, nm := range names {
			if rn, ok := statReportName(nm[0], nm[1]); ok {
				reportNames[rn.Key()] = rn
			}
		}
	}

	// Read the primary so stats for a report created moments ago aren't
	// dropped (the Redis entries are already gone) before it reaches the replica.
	reports, err := ds.QueriesByName(ctxdb.RequirePrimary(ctx, true), slices.Collect(maps.Values(reportNames)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "batch-load report ids from names")
	}

	// build the batch of stats to upsert, ignoring stats that don't resolve to
	// a report (2017 packs, deleted reports, unknown fleets)
	batchStatsByHost := make(map[uint][]fleet.ScheduledQueryStats, len(hostsStats))
	for hid, sqStats := range hostsStats {
		batchStats := batchStatsByHost[hid]
		for _, sqStat := range sqStats {
			rn, ok := statReportName(sqStat.PackName, sqStat.ScheduledQueryName)
			if !ok {
				continue
			}
			report, ok := reports[rn.Key()]
			if !ok {
				continue
			}
			sqStat.ScheduledQueryID = report.ID
			batchStats = append(batchStats, sqStat)
		}
		batchStatsByHost[hid] = batchStats
	}

	countExecs, err := ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, batchStatsByHost, cfg.InsertBatch)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "batch-save scheduled query stats for hosts")
	}
	stats.Inserts += countExecs // technically, could be updates

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

// statReportName returns the report a stat was recorded for. Reports reach
// osquery in the "Global" and "team-<id>" packs; stats for any other pack
// (e.g. 2017 packs) don't map to a report.
func statReportName(packName, schedQueryName string) (fleet.TeamScopedQueryName, bool) {
	if packName == "Global" {
		return fleet.TeamScopedQueryName{Name: schedQueryName}, true
	}
	idStr, ok := strings.CutPrefix(packName, "team-")
	if !ok {
		return fleet.TeamScopedQueryName{}, false
	}
	teamID, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		return fleet.TeamScopedQueryName{}, false
	}
	return fleet.TeamScopedQueryName{TeamID: new(uint(teamID)), Name: schedQueryName}, true
}

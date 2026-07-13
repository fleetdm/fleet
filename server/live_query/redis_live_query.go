// Package live_query implements an interface for storing and
// retrieving live queries.
//
// # Design
//
// This package operates by storing a single redis key for host
// targeting information. This key has a known prefix, and the data
// is a bitfield representing _all_ the hosts in fleet.
//
// In this model, a live query creation is a few redis writes. While a host
// checkin needs to scan the keys stored in a set representing all active live
// queries, and then fetch the bitfield value for their id.  This model fits
// very well with having a lot of hosts and very few live queries.
//
// A contrasting model, for the case of fewer hosts, but a lot of live
// queries, is to have a set per host. In this case, the LQ is pushed
// into each host's set. This model has many potential writes for LQ
// creation, but a host checkin has very few.
//
// The bitfield model fits "many hosts, few queries", but it scales poorly when
// many queries run concurrently: a host checkin must probe (GETBIT) every active
// query's bitfield, so the per-checkin cost grows with the number of queries -
// even queries that target a single host force a probe on every host. To handle
// that case, this package uses a hybrid: queries targeting at most
// smallTargetThreshold hosts are stored using the per-host set model above
// (the "reverse index"), while larger ("broadcast") queries keep the bitfield.
//
// # Implementation
//
// There are three keys for each bitfield (broadcast) live query: the bitfield,
// the SQL of the query and the set containing the IDs of all active live
// queries:
//
//	livequery:<ID> is the bitfield that indicates the hosts
//	sql:livequery:<ID> is the SQL of the query.
//	livequery:active is the set containing the active live query IDs
//
// Both the bitfield and sql keys have an expiration, and <ID> is the campaign
// ID of the query.  To make efficient use of Redis Cluster (without impacting
// standalone Redis), the <ID> is stored in braces (hash tags, e.g.
// livequery:{1} and sql:livequery:{1}), so that the two keys for the same <ID>
// are always stored on the same node (as they hash to the same cluster slot).
// See https://redis.io/topics/cluster-spec#keys-hash-tags for details.
//
// Small-target queries instead use the reverse index. There is no bitfield;
// the campaign ID is added to a per-host set for each targeted host, and the
// campaign ID is also added to a set of reverse-model queries:
//
//	livequery:host:<hostID> is the set of campaign IDs targeting that host
//	livequery:active:reverse is the set of campaign IDs using the reverse model
//
// The sql:livequery:<ID> and livequery:active keys are used by both models. A
// host checkin reads its own livequery:host:<hostID> set once (instead of one
// GETBIT per small-target query) and probes the bitfield only for the remaining
// broadcast queries. The per-host sets have a TTL and stale entries (campaigns
// no longer active) are filtered against the active set at read time, so they do
// not need to be removed on StopQuery.
//
// It is a noted downside that the active live queries set will necessarily
// live on a single node in cluster mode (a "hot key"), and that node will see
// increased activity due to that. Should that become a significant problem, an
// alternative approach will be required.
package live_query

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	bitsInByte              = 8
	queryKeyPrefix          = "livequery:"
	sqlKeyPrefix            = "sql:"
	activeQueriesKey        = "livequery:active"
	activeReverseQueriesKey = "livequery:active:reverse"
	reverseHostKeyPrefix    = "livequery:host:"
	queryExpiration         = 7 * 24 * time.Hour
	queryResultsCountPrefix = "query_results_count:"
)

type redisLiveQuery struct {
	// connection pool
	pool fleet.RedisPool
	// in memory cache
	cache memCache
	// in memory cache expiration
	cacheExpiration time.Duration

	// smallTargetThreshold is the maximum number of targeted hosts for a query to
	// use the per-host reverse index instead of the bitfield. A value of 0
	// disables the reverse index entirely (all queries use the bitfield).
	smallTargetThreshold int

	logger *slog.Logger
}

// memCache is an in-memory cache for live queries. It stores the SQL of the
// queries and the active queries set. It also stores the expiration time of the
// cache.
type memCache struct {
	sqlCache           map[string]string
	activeQueriesCache []string
	// reverseActiveCache holds the campaign IDs (among the active queries) that
	// use the reverse per-host index. It is used by the read path to exclude
	// those queries from the per-host bitfield (GETBIT) probes.
	reverseActiveCache map[string]struct{}
	cacheExp           time.Time
	mu                 sync.RWMutex
}

// cacheIsExpired is a thread-safe method to check if the cache is expired.
func (r *redisLiveQuery) cacheIsExpired() bool {
	r.cache.mu.RLock()
	defer r.cache.mu.RUnlock()
	return r.cache.cacheExp.Before(time.Now())
}

// getSQLByCampaignID is a thread-safe method to get the SQL of a live query by its
// campaign ID.
func (r *redisLiveQuery) getSQLByCampaignID(campaignID string) (string, bool) {
	r.cache.mu.RLock()
	defer r.cache.mu.RUnlock()
	sql, found := r.cache.sqlCache[campaignID]
	return sql, found
}

// isReverse is a thread-safe method that reports whether the given active
// campaign ID is stored using the reverse per-host index (rather than a
// bitfield).
func (r *redisLiveQuery) isReverse(campaignID string) bool {
	r.cache.mu.RLock()
	defer r.cache.mu.RUnlock()
	_, found := r.cache.reverseActiveCache[campaignID]
	return found
}

// hasReverseActiveQueries is a thread-safe method that reports whether any
// active query uses the reverse per-host index. It gates the per-host reverse
// read on the checkin path so that, when no reverse query is active, a checkin
// does not issue a SMEMBERS for a key that does not exist.
func (r *redisLiveQuery) hasReverseActiveQueries() bool {
	r.cache.mu.RLock()
	defer r.cache.mu.RUnlock()
	return len(r.cache.reverseActiveCache) > 0
}

// NewRedisLiveQuery creates a new Redis implementation of the live query store
// using the provided Redis connection pool.
//
// smallTargetThreshold is the maximum number of targeted hosts for a query to
// use the reverse per-host index instead of the bitfield; a value of 0 disables
// the reverse index entirely (kill-switch), so all queries use the bitfield.
func NewRedisLiveQuery(pool fleet.RedisPool, logger *slog.Logger, memCacheExp time.Duration, smallTargetThreshold int) *redisLiveQuery {
	return &redisLiveQuery{
		pool:                 pool,
		cache:                newMemCache(),
		cacheExpiration:      memCacheExp,
		smallTargetThreshold: smallTargetThreshold,
		logger:               logger,
	}
}

func newMemCache() memCache {
	return memCache{
		sqlCache:           make(map[string]string),
		activeQueriesCache: make([]string, 0),
		reverseActiveCache: make(map[string]struct{}),
	}
}

// generate keys for the bitfield and sql of a query - those always go in pair
// and should live on the same cluster node when Redis Cluster is used, so
// the common part of the key (the 'name' parameter) is used as key tag.
func generateKeys(name string) (targetsKey, sqlKey string) {
	keyTag := "{" + name + "}"
	return queryKeyPrefix + keyTag, sqlKeyPrefix + queryKeyPrefix + keyTag
}

// returns the base name part of a target key, i.e. so that this is true:
//
//	tkey, _ := generateKeys(name)
//	baseName := extractTargetKeyName(tkey)
//	baseName == name
func extractTargetKeyName(key string) string {
	name := strings.TrimPrefix(key, queryKeyPrefix)
	if len(name) > 0 && name[0] == '{' {
		name = name[1:]
	}
	if len(name) > 0 && name[len(name)-1] == '}' {
		name = name[:len(name)-1]
	}
	return name
}

// reverseHostKey returns the key of the per-host set that stores the campaign
// IDs of the small-target live queries targeting the given host. The host ID is
// used as the cluster hash tag so that a host's set always lives on a single
// node (the set is read on every checkin for that host).
func reverseHostKey(hostID uint) string {
	return reverseHostKeyPrefix + "{" + strconv.FormatUint(uint64(hostID), 10) + "}"
}

// RunQuery stores the live query information in ephemeral storage for the
// duration of the query or its TTL. Note that hostIDs *must* be sorted
// in ascending order. The name is the campaign ID as a string.
func (r *redisLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return errors.New("no hosts targeted")
	}

	// Small-target queries use the per-host reverse index so that a host checkin
	// does not have to probe this query's bitfield (one GETBIT per query). Large
	// (broadcast) queries keep the bitfield, which is compact relative to a large
	// target set and cheap to create/stop. A threshold of 0 disables the reverse
	// index (no query has <= 0 targets), so all queries use the bitfield.
	if len(hostIDs) <= r.smallTargetThreshold {
		if err := r.storeQueryInfoReverse(name, sql, hostIDs); err != nil {
			return fmt.Errorf("store reverse query info: %w", err)
		}
		// mark the campaign id as using the reverse model
		if err := r.storeReverseQueryName(name); err != nil {
			return fmt.Errorf("store reverse query name: %w", err)
		}
	} else {
		// store the sql and targeted hosts information (bitfield)
		if err := r.storeQueryInfo(name, sql, hostIDs); err != nil {
			return fmt.Errorf("store query info: %w", err)
		}
	}

	// store name (campaign id) into the active live queries set (both models)
	if err := r.storeQueryNames(name); err != nil {
		return fmt.Errorf("store query name: %w", err)
	}

	return nil
}

func (r *redisLiveQuery) StopQuery(name string) error {
	// remove the sql and targeted hosts keys (DEL of the bitfield key is a no-op
	// for reverse queries, which don't have one)
	if err := r.removeQueryInfo(name); err != nil {
		return fmt.Errorf("remove query info: %w", err)
	}

	// remove name (campaign id) from the livequery set
	if err := r.removeQueryNames(name); err != nil {
		return fmt.Errorf("remove query name: %w", err)
	}

	// remove from the reverse model set. The per-host sets cannot be enumerated
	// by campaign, so they are left to expire via their TTL and are filtered out
	// at read time against the active set. This is safe only because campaign IDs
	// are monotonic (MySQL auto-increment) and never reused: a stale per-host
	// membership can therefore never collide with a different, newly-active
	// campaign that happens to share the same ID.
	if err := r.removeReverseQueryNames(name); err != nil {
		return fmt.Errorf("remove reverse query name: %w", err)
	}

	return nil
}

// this is a variable so it can be changed in tests
var cleanupExpiredQueriesModulo int64 = 10

func (r *redisLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	// Get keys for active queries (this also (re)loads the in-memory cache, which
	// is what isReverse below relies on).
	names, err := r.LoadActiveQueryNames()
	if err != nil {
		return nil, fmt.Errorf("load active queries: %w", err)
	}

	queries := make(map[string]string)

	// Broadcast queries: probe this host's bit in each query's bitfield. Reverse
	// (small-target) queries are excluded here - probing them is the per-checkin
	// command storm this whole change is meant to avoid.
	keyNames := make([]string, 0, len(names))
	for _, name := range names {
		if r.isReverse(name) {
			continue
		}
		tkey, _ := generateKeys(name)
		keyNames = append(keyNames, tkey)
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, keyNames...)
	for _, qkeys := range keysBySlot {
		if err := r.collectBatchQueriesForHost(hostID, qkeys, queries); err != nil {
			return nil, err
		}
	}

	// Reverse (small-target) queries: a single read of this host's own set.
	// Skip it entirely when no active query uses the reverse model.
	if r.hasReverseActiveQueries() {
		if err := r.collectReverseQueriesForHost(hostID, queries); err != nil {
			return nil, err
		}
	}

	return queries, nil
}

// collectReverseQueriesForHost reads the per-host reverse-index set and adds any
// still-active small-target queries targeting this host to queriesByHost. Stale
// campaign IDs (lingering in the per-host set after the query was stopped) are
// filtered out because their SQL is no longer in the cache.
func (r *redisLiveQuery) collectReverseQueriesForHost(hostID uint, queriesByHost map[string]string) error {
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	// Stale-entry filtering below relies on the SQL cache holding only active
	// queries. Refresh it on expiry here so this path stays correct on its own,
	// independent of any cache (re)load done by the caller or the bitfield path
	// (which is skipped when every active query is small-target).
	if r.cacheIsExpired() {
		if err := r.loadCache(); err != nil {
			return fmt.Errorf("load cache: %w", err)
		}
	}

	names, err := redigo.Strings(conn.Do("SMEMBERS", reverseHostKey(hostID)))
	if err != nil && err != redigo.ErrNil {
		return fmt.Errorf("smembers reverse host key: %w", err)
	}

	for _, name := range names {
		// The SQL cache only holds active queries, so a missing entry means the
		// campaign is no longer active (stale entry) and is skipped.
		if sql, found := r.getSQLByCampaignID(name); found {
			queriesByHost[name] = sql
		}
	}
	return nil
}

func (r *redisLiveQuery) collectBatchQueriesForHost(hostID uint, queryKeys []string, queriesByHost map[string]string) error {
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	if r.cacheIsExpired() {
		if err := r.loadCache(); err != nil {
			return fmt.Errorf("load cache: %w", err)
		}
	}

	// Pipeline redis calls to check for this host in the bitfield of the
	// targets of the query.
	for _, key := range queryKeys {
		if err := conn.Send("GETBIT", key, hostID); err != nil {
			return fmt.Errorf("getbit query targets: %w", err)
		}
	}

	// Flush calls to begin receiving results.
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("flush pipeline: %w", err)
	}

	// Receive target and SQL in order of pipelined calls.
	for _, key := range queryKeys {
		name := extractTargetKeyName(key)

		// the result of GETBIT will not fail if the key does not exist, it will
		// just return 0, so it can't be used to detect if the livequery still
		// exists.
		targeted, err := redigo.Int(conn.Receive())
		if err != nil {
			return fmt.Errorf("receive target: %w", err)
		}

		if targeted == 1 {
			if sql, found := r.getSQLByCampaignID(name); found {
				queriesByHost[name] = sql
			} else {
				r.logger.WarnContext(context.TODO(), "live query not found in cache", "name", name)
			}
		}
	}
	return nil
}

func (r *redisLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Clear completion in both models without depending on which one this query
	// uses: exactly one of these has an effect, the other is a harmless no-op
	// (SREM on an absent member, and the guarded SETBIT below on an absent key).
	// This avoids relying on a possibly-stale cache to pick the model, where a
	// wrong guess would leave the host still receiving the query.
	if _, err := conn.Do("SREM", reverseHostKey(hostID), name); err != nil {
		return fmt.Errorf("srem reverse host key: %w", err)
	}

	targetKey, _ := generateKeys(name)

	// Update the bitfield for this host only if the key exists.
	// If the key doesn't exist (e.g. query marked as completed or cancelled)
	// then we don't want to call SETBIT because it will create a new
	// key (that won't expire and linger "forever").
	const setBitScript = `
	if redis.call('EXISTS', KEYS[1]) == 1 then
		return redis.call('SETBIT', KEYS[1], ARGV[1], ARGV[2])
	else
		return nil
	end`
	if _, err := conn.Do("EVAL", setBitScript, 1, targetKey, hostID, 0); err != nil {
		return fmt.Errorf("setbit query key: %w", err)
	}

	// NOTE(mna): we could remove the query here if all bits are now off, meaning
	// that all hosts have completed this query, but the BITCOUNT command can be
	// costly on large strings and we will have quite large ones. This should not be
	// needed anyway as StopQuery appears to be called every time a campaign is
	// run (see svc.CompleteCampaign).

	return nil
}

func (r *redisLiveQuery) storeQueryInfo(name, sql string, hostIDs []uint) error {
	conn := r.pool.Get()
	defer conn.Close()

	// Map the targeted host IDs to a bitfield. Store targets in one key and SQL
	// in another.
	targetKey, sqlKey := generateKeys(name)
	targets := mapBitfield(hostIDs)

	// Ensure to set SQL first or else we can end up in a weird state in which a
	// client reads that the query exists but cannot look up the SQL.
	err := conn.Send("SET", sqlKey, sql, "EX", queryExpiration.Seconds())
	if err != nil {
		return fmt.Errorf("set sql: %w", err)
	}
	_, err = conn.Do("SET", targetKey, targets, "EX", queryExpiration.Seconds())
	if err != nil {
		return fmt.Errorf("set targets: %w", err)
	}
	return nil
}

// storeQueryInfoReverse stores the SQL of the query and adds the campaign id to
// the per-host set of every targeted host (the reverse index). The per-host
// sets are given a TTL so that orphaned entries (e.g. if StopQuery is missed)
// eventually expire; they are also filtered against the active set at read time.
func (r *redisLiveQuery) storeQueryInfoReverse(name, sql string, hostIDs []uint) error {
	// Store the SQL first, so a host never sees the query as targeted before its
	// SQL can be looked up (same ordering guarantee as the bitfield path).
	_, sqlKey := generateKeys(name)
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	if _, err := conn.Do("SET", sqlKey, sql, "EX", queryExpiration.Seconds()); err != nil {
		conn.Close()
		return fmt.Errorf("set sql: %w", err)
	}
	conn.Close()

	// Add the campaign id to each targeted host's set, pipelined per cluster slot.
	hostKeys := make([]string, len(hostIDs))
	for i, hostID := range hostIDs {
		hostKeys[i] = reverseHostKey(hostID)
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, hostKeys...)
	for _, keys := range keysBySlot {
		if err := r.storeBatchReverseHostKeys(name, keys); err != nil {
			return err
		}
	}
	return nil
}

func (r *redisLiveQuery) storeBatchReverseHostKeys(name string, hostKeys []string) error {
	conn := r.pool.Get()
	defer conn.Close()

	for _, hostKey := range hostKeys {
		if err := conn.Send("SADD", hostKey, name); err != nil {
			return fmt.Errorf("sadd reverse host key: %w", err)
		}
		if err := conn.Send("EXPIRE", hostKey, int(queryExpiration.Seconds())); err != nil {
			return fmt.Errorf("expire reverse host key: %w", err)
		}
	}
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("flush pipeline: %w", err)
	}
	// drain replies (2 per host key) to complete the pipeline
	for range hostKeys {
		if _, err := conn.Receive(); err != nil {
			return fmt.Errorf("receive sadd reply: %w", err)
		}
		if _, err := conn.Receive(); err != nil {
			return fmt.Errorf("receive expire reply: %w", err)
		}
	}
	return nil
}

func (r *redisLiveQuery) storeReverseQueryName(name string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SADD", activeReverseQueriesKey, name)
	return err
}

func (r *redisLiveQuery) removeReverseQueryNames(names ...string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(activeReverseQueriesKey)
	args = args.AddFlat(names)
	_, err := conn.Do("SREM", args...)
	return err
}

func (r *redisLiveQuery) storeQueryNames(names ...string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(activeQueriesKey)
	args = args.AddFlat(names)
	_, err := conn.Do("SADD", args...)
	return err
}

func (r *redisLiveQuery) removeQueryInfo(name string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	targetKey, sqlKey := generateKeys(name)
	if _, err := conn.Do("DEL", targetKey, sqlKey); err != nil {
		return fmt.Errorf("del query keys: %w", err)
	}
	return nil
}

func (r *redisLiveQuery) removeQueryNames(names ...string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(activeQueriesKey)
	args = args.AddFlat(names)
	_, err := conn.Do("SREM", args...)
	return err
}

func (r *redisLiveQuery) LoadActiveQueryNames() ([]string, error) {
	// copyActiveQueries returns a copy of the active queries cache to
	// ensure thread safety.
	copyActiveQueries := func() []string {
		r.cache.mu.RLock()
		defer r.cache.mu.RUnlock()

		names := make([]string, len(r.cache.activeQueriesCache))
		copy(names, r.cache.activeQueriesCache)

		return names
	}

	if !r.cacheIsExpired() {
		return copyActiveQueries(), nil
	}

	if err := r.loadCache(); err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	return copyActiveQueries(), nil
}

func (r *redisLiveQuery) loadCache() error {
	expiredQueries := make(map[string]struct{})
	sqlCache := make(map[string]string)
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	activeIDs, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
	if err != nil && err != redigo.ErrNil {
		return fmt.Errorf("get active queries: %w", err)
	}

	// Load which active campaigns use the reverse per-host index, so the read
	// path can exclude them from the per-host bitfield (GETBIT) probes.
	reverseIDs, err := redigo.Strings(conn.Do("SMEMBERS", activeReverseQueriesKey))
	if err != nil && err != redigo.ErrNil {
		return fmt.Errorf("get reverse active queries: %w", err)
	}
	reverseActive := make(map[string]struct{}, len(reverseIDs))
	for _, id := range reverseIDs {
		reverseActive[id] = struct{}{}
	}

	for _, id := range activeIDs {
		_, sqlKey := generateKeys(id)

		sql, err := redigo.String(conn.Do("GET", sqlKey))
		if err != nil {
			if err != redigo.ErrNil {
				return fmt.Errorf("get query sql: %w", err)
			}

			// It is possible the livequery key has expired but was still in the set
			// - handle this gracefully by collecting the keys to remove them from
			// the set and keep going.
			expiredQueries[id] = struct{}{}
			continue
		}

		sqlCache[id] = sql
	}

	// remove expired queries from the names list
	if len(expiredQueries) > 0 {
		trimmedIDs := make([]string, 0, len(activeIDs)-len(expiredQueries))
		for _, name := range activeIDs {
			if _, found := expiredQueries[name]; !found {
				trimmedIDs = append(trimmedIDs, name)
			}
		}
		activeIDs = trimmedIDs
	}

	r.cache.mu.Lock()
	r.cache.sqlCache = sqlCache
	r.cache.activeQueriesCache = activeIDs
	r.cache.reverseActiveCache = reverseActive
	r.cache.cacheExp = time.Now().Add(r.cacheExpiration)
	r.cache.mu.Unlock()

	if len(expiredQueries) > 0 {
		// a certain percentage of the time so that we don't overwhelm redis with a
		// bunch of similar deletion commands at the same time, clean up the
		// expired queries.
		if time.Now().UnixNano()%cleanupExpiredQueriesModulo == 0 {
			names := make([]string, 0, len(expiredQueries))
			for k := range expiredQueries {
				names = append(names, k)
			}

			go func() {
				err = r.removeQueryNames(names...)
				if err != nil {
					r.logger.WarnContext(context.TODO(), "removing expired live queries", "err", err)
				}
			}()
		}
	}

	return nil
}

func (r *redisLiveQuery) CleanupInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error {
	// the following logic is used to cleanup inactive queries:
	// 	* the inactive campaign IDs are removed from the livequery:active set
	//
	// At this point, all inactive queries are already effectively deleted - the
	// rest is just best effort cleanup to save Redis memory space, but those
	// keys would otherwise be ignored and without effect.
	//
	// * remove the livequery:<ID> and sql:livequery:<ID> for every inactive
	// 	campaign ID.

	if len(inactiveCampaignIDs) == 0 {
		return nil
	}

	if err := r.removeInactiveQueries(ctx, inactiveCampaignIDs); err != nil {
		return err
	}

	keysToDel := make([]string, 0, len(inactiveCampaignIDs)*2)
	for _, id := range inactiveCampaignIDs {
		targetKey, sqlKey := generateKeys(strconv.FormatUint(uint64(id), 10))
		keysToDel = append(keysToDel, targetKey, sqlKey)
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, keysToDel...)
	for _, keys := range keysBySlot {
		if err := r.removeBatchInactiveKeys(ctx, keys); err != nil {
			return err
		}
	}
	return nil
}

func (r *redisLiveQuery) removeBatchInactiveKeys(ctx context.Context, keys []string) error {
	conn := r.pool.Get()
	defer conn.Close()

	args := redigo.Args{}.AddFlat(keys)
	if _, err := conn.Do("DEL", args...); err != nil {
		return ctxerr.Wrap(ctx, err, "remove batch of inactive keys")
	}
	return nil
}

func (r *redisLiveQuery) removeInactiveQueries(ctx context.Context, inactiveCampaignIDs []uint) error {
	conn := r.pool.Get()
	defer conn.Close()

	args := redigo.Args{}.Add(activeQueriesKey).AddFlat(inactiveCampaignIDs)
	if _, err := conn.Do("SREM", args...); err != nil {
		return ctxerr.Wrap(ctx, err, "remove inactive campaign IDs")
	}

	// Also remove from the reverse model set. The per-host sets are left to expire
	// via their TTL and are filtered against the active set at read time.
	reverseArgs := redigo.Args{}.Add(activeReverseQueriesKey).AddFlat(inactiveCampaignIDs)
	if _, err := conn.Do("SREM", reverseArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "remove inactive reverse campaign IDs")
	}
	return nil
}

// mapBitfield takes the given host IDs and maps them into a bitfield compatible
// with Redis. It is expected that the input IDs are in ascending order.
func mapBitfield(hostIDs []uint) []byte {
	if len(hostIDs) == 0 {
		return []byte{}
	}

	// NOTE(mna): note that this is efficient storage if the host IDs are mostly
	// sequential and starting from 1, e.g. as in a newly created database. If
	// there's substantial churn in hosts (e.g. some are coming on and off) or
	// for some reason the auto_increment had to be bumped (e.g. it increments
	// with failed inserts, even if there's an "on duplicate" clause), then it
	// could get quite inefficient. If the id gets, say, to 10M then the bitfield
	// will take over 1MB (10M bytes / 8 bits), regardless of the number of hosts
	// - at which point it could start to be more interesting to store a set of
	// host IDs (even though the members of a SET are stored as strings so the
	// storage bytes depends on the length of the string representation).
	//
	// Running the following in Redis v6.2.6 (using i+100000 so that IDs reflect
	// the high numbers and take the corresponding number of storage bytes):
	//
	//     > eval 'for i=1, 100000 do redis.call("sadd", KEYS[1], i+100000) end' 1 myset
	//     > memory usage myset
	//     (integer) 4248715
	//
	// So it would take a bit under 4MB to store ALL 100K host IDs, obviously
	// less to store a subset of those. On the other hand, the large bitfield
	// usage of memory would be true even if there was only one host selected in
	// the query, should that host be one of the high IDs.  Something to keep in
	// mind if at some point we have reports of unexpectedly large redis memory
	// usage, as that storage is repeated for each live query.

	// As the input IDs are in ascending order, we get two optimizations here:
	// 1. We can calculate the length of the bitfield necessary by using the
	// last ID in the slice. Then we allocate the slice all at once.
	// 2. We benefit from accessing the elements of the slice in order,
	// potentially making more effective use of the processor cache.
	byteLen := hostIDs[len(hostIDs)-1]/bitsInByte + 1
	field := make([]byte, byteLen)
	for _, id := range hostIDs {
		byteIndex := id / bitsInByte
		bitIndex := bitsInByte - (id % bitsInByte) - 1
		field[byteIndex] |= 1 << bitIndex
	}

	return field
}

func queryResultsCountKey(queryID uint) string {
	return fmt.Sprintf("%s%d", queryResultsCountPrefix, queryID)
}

// GetQueryResultsCounts returns the current count of query results for multiple queries.
// Returns a map of query ID -> count. Missing keys are returned with a count of 0.
func (r *redisLiveQuery) GetQueryResultsCounts(queryIDs []uint) (map[uint]int, error) {
	if len(queryIDs) == 0 {
		return make(map[uint]int), nil
	}

	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	// Pipeline GET requests for all query IDs
	for _, queryID := range queryIDs {
		key := queryResultsCountKey(queryID)
		if err := conn.Send("GET", key); err != nil {
			return nil, fmt.Errorf("send get query results count: %w", err)
		}
	}

	if err := conn.Flush(); err != nil {
		return nil, fmt.Errorf("flush pipeline: %w", err)
	}

	// Receive results and build the map
	results := make(map[uint]int, len(queryIDs))
	for _, queryID := range queryIDs {
		count, err := redigo.Int(conn.Receive())
		if err != nil {
			if err == redigo.ErrNil {
				results[queryID] = 0
				continue
			}
			return nil, fmt.Errorf("receive query results count: %w", err)
		}
		results[queryID] = count
	}

	return results, nil
}

// IncrQueryResultsCounts increments the query results counts by the given amounts.
// Takes a map of query ID -> amount to increment.
func (r *redisLiveQuery) IncrQueryResultsCounts(queryIDsToAmounts map[uint]int) error {
	if len(queryIDsToAmounts) == 0 {
		return nil
	}

	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Pipeline INCRBY requests for all query IDs
	for queryID, amount := range queryIDsToAmounts {
		key := queryResultsCountKey(queryID)
		if err := conn.Send("INCRBY", key, amount); err != nil {
			return fmt.Errorf("send incrby query results count: %w", err)
		}
	}

	if err := conn.Flush(); err != nil {
		return fmt.Errorf("flush pipeline: %w", err)
	}

	// Receive all results to complete the pipeline (we don't need the values)
	for range queryIDsToAmounts {
		if _, err := conn.Receive(); err != nil {
			return fmt.Errorf("receive incrby result: %w", err)
		}
	}

	return nil
}

// SetQueryResultsCount sets the query results count for a query to a specific value.
// Used to reset counts to zero when a query is modified, or to adjust the count
// in the cleanup cron job after deleting excess rows.
func (r *redisLiveQuery) SetQueryResultsCount(queryID uint, count int) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	key := queryResultsCountKey(queryID)
	if _, err := conn.Do("SET", key, count); err != nil {
		return fmt.Errorf("set query results count: %w", err)
	}

	return nil
}

// DeleteQueryResultsCount deletes the query results count for a query.
// Used when deleting a query, to remove the Redis key.
func (r *redisLiveQuery) DeleteQueryResultsCount(queryID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	key := queryResultsCountKey(queryID)
	if _, err := conn.Do("DEL", key); err != nil {
		return fmt.Errorf("delete query results count: %w", err)
	}

	return nil
}

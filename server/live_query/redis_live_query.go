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
// We believe that normal fleet usage has many hosts, and a small
// number of live queries targeting all of them. This was a big
// factor in choosing this implementation.
//
// # Implementation
//
// As mentioned in the Design section, there are three keys for each
// live query: the bitfield, the SQL of the query and the set containing
// the IDs of all active live queries:
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
// It is a noted downside that the active live queries set will necessarily
// live on a single node in cluster mode (a "hot key"), and that node will see
// increased activity due to that. Should that become a significant problem, an
// alternative approach will be required.
package live_query

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	bitsInByte       = 8
	queryKeyPrefix   = "livequery:"
	sqlKeyPrefix     = "sql:"
	activeQueriesKey = "livequery:active"
	queryExpiration  = 7 * 24 * time.Hour
)

type redisLiveQuery struct {
	// connection pool
	pool fleet.RedisPool
	// in memory cache
	cache memCache
	// in memory cache expiration
	cacheExpiration time.Duration

	logger kitlog.Logger
}

// memCache is an in-memory cache for live queries. It stores the SQL of the
// queries and the active queries set. It also stores the expiration time of the
// cache.
type memCache struct {
	sqlCache           map[string]string
	activeQueriesCache []string
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

// NewRedisQueryResults creates a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisLiveQuery(pool fleet.RedisPool, logger kitlog.Logger, memCacheExp time.Duration) *redisLiveQuery {
	return &redisLiveQuery{
		pool:            pool,
		cache:           newMemCache(),
		cacheExpiration: memCacheExp,
		logger:          logger,
	}
}

func newMemCache() memCache {
	return memCache{
		sqlCache:           make(map[string]string),
		activeQueriesCache: make([]string, 0),
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

// RunQuery stores the live query information in ephemeral storage for the
// duration of the query or its TTL. Note that hostIDs *must* be sorted
// in ascending order. The name is the campaign ID as a string.
func (r *redisLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return errors.New("no hosts targeted")
	}

	// store the sql and targeted hosts information
	if err := r.storeQueryInfo(name, sql, hostIDs); err != nil {
		return fmt.Errorf("store query info: %w", err)
	}

	// store name (campaign id) into the active live queries set
	if err := r.storeQueryNames(name); err != nil {
		return fmt.Errorf("store query name: %w", err)
	}

	return nil
}

func (r *redisLiveQuery) StopQuery(name string) error {
	// remove the sql and targeted hosts keys
	if err := r.removeQueryInfo(name); err != nil {
		return fmt.Errorf("remove query info: %w", err)
	}

	// remove name (campaign id) from the livequery set
	if err := r.removeQueryNames(name); err != nil {
		return fmt.Errorf("remove query name: %w", err)
	}

	return nil
}

// this is a variable so it can be changed in tests
var cleanupExpiredQueriesModulo int64 = 10

func (r *redisLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	// Get keys for active queries
	names, err := r.LoadActiveQueryNames()
	if err != nil {
		return nil, fmt.Errorf("load active queries: %w", err)
	}

	// convert the query name (campaign id) to the key name
	keyNames := make([]string, 0, len(names))
	for _, name := range names {
		tkey, _ := generateKeys(name)
		keyNames = append(keyNames, tkey)
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, keyNames...)
	queries := make(map[string]string)
	for _, qkeys := range keysBySlot {
		if err := r.collectBatchQueriesForHost(hostID, qkeys, queries); err != nil {
			return nil, err
		}
	}

	return queries, nil
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
				level.Warn(r.logger).Log("msg", "live query not found in cache", "name", name)
			}
		}
	}
	return nil
}

func (r *redisLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	targetKey, _ := generateKeys(name)

	// Update the bitfield for this host.
	if _, err := conn.Do("SETBIT", targetKey, hostID, 0); err != nil {
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
					level.Warn(r.logger).Log("msg", "removing expired live queries", "err", err)
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

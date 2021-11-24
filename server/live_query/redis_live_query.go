// Package live_query implements an interface for storing and
// retrieving live queries.
//
// Design
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
// Implementation
//
// As mentioned in the Design section, there are three keys for each
// live query: the bitfield, the SQL of the query and the set containing
// the IDs of all active live queries:
//
//     livequery:<ID> is the bitfield that indicates the hosts
//     sql:livequery:<ID> is the SQL of the query.
//     livequery:active is the set containing the active live query IDs
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
//
package live_query

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
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
}

// NewRedisQueryResults creates a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisLiveQuery(pool fleet.RedisPool) *redisLiveQuery {
	return &redisLiveQuery{pool: pool}
}

// generate keys for the bitfield and sql of a query - those always go in pair
// and should live on the same cluster node when Redis Cluster is used, so
// the common part of the key (the 'name' parameter) is used as key tag.
func generateKeys(name string) (targetsKey, sqlKey string) {
	keyTag := "{" + name + "}"
	return queryKeyPrefix + keyTag, sqlKeyPrefix + queryKeyPrefix + keyTag
}

// returns the base name part of a target key, i.e. so that this is true:
//     tkey, _ := generateKeys(name)
//     baseName := extractTargetKeyName(tkey)
//     baseName == name
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

// MigrateKeys migrates keys using a deprecated format to the new format. It
// should be called at startup and never after that, so for this reason it is
// not added to the fleet.LiveQueryStore interface.
func (r *redisLiveQuery) MigrateKeys() error {
	qkeys, err := redis.ScanKeys(r.pool, queryKeyPrefix+"*", 100)
	if err != nil {
		return err
	}

	// TODO(mna): using the same scan, migrate keys from pre-set to
	// use the livequery set.

	// identify which of those keys are in a deprecated format
	var oldKeys []string
	for _, key := range qkeys {
		name := extractTargetKeyName(key)
		if !strings.Contains(key, "{"+name+"}") {
			// add the corresponding sql key to the list
			oldKeys = append(oldKeys, key, sqlKeyPrefix+key)
		}
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, oldKeys...)
	for _, keys := range keysBySlot {
		if err := migrateBatchKeys(r.pool, keys); err != nil {
			return err
		}
	}
	return nil
}

func migrateBatchKeys(pool fleet.RedisPool, keys []string) error {
	readConn := pool.Get()
	defer readConn.Close()

	writeConn := pool.Get()
	defer writeConn.Close()

	// use a retry conn so that we follow MOVED redirections in a Redis Cluster,
	// as we will attempt to write new keys which may not belong to the same
	// cluster slot.  It returns an error if writeConn is not a redis cluster
	// connection, in which case we simply continue with the standalone Redis
	// writeConn.
	if rc, err := redisc.RetryConn(writeConn, 3, 100*time.Millisecond); err == nil {
		writeConn = rc
	}

	// using a straightforward "read one, write one" approach as this is meant to
	// run at startup, not on a hot path, and we expect a relatively small number
	// of queries vs hosts (as documented in the design comment at the top).
	for _, key := range keys {
		s, err := redigo.String(readConn.Do("GET", key))
		if err != nil {
			if err == redigo.ErrNil {
				// key may have expired since the scan, ignore
				continue
			}
			return err
		}

		var newKey string
		if strings.HasPrefix(key, sqlKeyPrefix) {
			name := extractTargetKeyName(strings.TrimPrefix(key, sqlKeyPrefix))
			_, newKey = generateKeys(name)
		} else {
			name := extractTargetKeyName(key)
			newKey, _ = generateKeys(name)
		}
		if _, err := writeConn.Do("SET", newKey, s, "EX", queryExpiration.Seconds()); err != nil {
			return err
		}

		// best-effort deletion of the old key, ignore error
		readConn.Do("DEL", key)
	}
	return nil
}

// RunQuery stores the live query information in ephemeral storage for the
// duration of the query or its TTL. Note that hostIDs *must* be sorted
// in ascending order.
func (r *redisLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return errors.New("no hosts targeted")
	}

	// store the sql and targeted hosts information
	if err := r.storeQueryInfo(name, sql, hostIDs); err != nil {
		return fmt.Errorf("store query info: %w", err)
	}

	// store name (campaign id) into the active live queries set
	if err := r.storeQueryName(name); err != nil {
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
	if err := r.removeQueryName(name); err != nil {
		return fmt.Errorf("remove query name: %w", err)
	}

	return nil
}

func (r *redisLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	// Get keys for active queries
	names, err := r.loadActiveQueryNames()
	if err != nil {
		return nil, fmt.Errorf("load active queries: %w", err)
	}

	// convert the query name (campaign id) to the key name
	for i, name := range names {
		tkey, _ := generateKeys(name)
		names[i] = tkey
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, names...)
	queries := make(map[string]string)
	expired := make(map[string]struct{})
	for _, qkeys := range keysBySlot {
		if err := r.collectBatchQueriesForHost(hostID, qkeys, queries, expired); err != nil {
			return nil, err
		}
	}

	if len(expired) > 0 {
		// TODO(mna): a certain percentage of the time, clean up the expired queries
	}

	return queries, nil
}

func (r *redisLiveQuery) collectBatchQueriesForHost(hostID uint, queryKeys []string, queriesByHost map[string]string, expiredQueries map[string]struct{}) error {
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	// Pipeline redis calls to check for this host in the bitfield of the
	// targets of the query.
	for _, key := range queryKeys {
		if err := conn.Send("GETBIT", key, hostID); err != nil {
			return fmt.Errorf("getbit query targets: %w", err)
		}

		// Additionally get SQL even though we don't yet know whether this query
		// is targeted to the host. This allows us to avoid an additional
		// roundtrip to the Redis server and likely has little cost due to the
		// small number of queries and limited size of SQL
		if err := conn.Send("GET", sqlKeyPrefix+key); err != nil {
			return fmt.Errorf("get query sql: %w", err)
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

		// Be sure to read SQL even if we are not going to include this query.
		// Otherwise we will read an incorrect number of returned results from
		// the pipeline.
		sql, err := redigo.String(conn.Receive())
		if err != nil {
			if err != redigo.ErrNil {
				return fmt.Errorf("receive sql: %w", err)
			}

			// It is possible the livequery key has expired but was still in the set
			// - handle this gracefully by collecting the keys to remove them from
			// the set and keep going.
			expiredQueries[name] = struct{}{}
			continue
		}

		if targeted == 0 {
			// Host not targeted with this query
			continue
		}
		queriesByHost[name] = sql
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
	// costly on large strings and we may have very large ones. This should not be
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

func (r *redisLiveQuery) storeQueryName(name string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SADD", activeQueriesKey, name)
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

func (r *redisLiveQuery) removeQueryName(name string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SREM", activeQueriesKey, name)
	return err
}

func (r *redisLiveQuery) loadActiveQueryNames() ([]string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	names, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
	if err != nil && err != redigo.ErrNil {
		return nil, err
	}
	return names, nil
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
	// for some reason the auto_increment had to be bumped (e.g. it increments with
	// failed inserts, even if there's an "on duplicate" clause), then it could get
	// quite inefficient. If the id gets, say, to 10M then the bitfield will take
	// over 1MB, even if there are only 100K hosts - at which point it would
	// likely become more efficient to store a set of host IDs (without any
	// redis-internal storage optimization, that would be 100K * 4 bytes =
	// ~380KB). This large usage of memory would even be true if there was only
	// one host selected in the query, should that host be one of the high IDs.
	// Something to keep in mind if at some point we have reports of unexpectedly
	// large redis memory usage, as that storage is repeated for each live query.

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

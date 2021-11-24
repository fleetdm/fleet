// package live_query implements an interface for storing and
// retrieving live queries.
//
// Design
//
// This package operates by storing a single redis key for host
// targeting information. This key has a known prefix, and the data
// is a bitfield representing _all_ the hosts in fleet.
//
// In this model, a live query creation is a few redis writes. While a
// host checkin needs to scan the keyspace for matching key, and then
// fetch the bitfield value for their id. While this scan might be
// expensive, this model fits very well with having a lot of hosts and
// very few live queries.
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
// As mentioned in the Design section, there are two keys for each
// live query: the bitfield and the SQL of the query:
//
//     livequery:<ID> is the bitfield that indicates the hosts
//     sql:livequery:<ID> is the SQL of the query.
//
// Both have an expiration, and <ID> is the campaign ID of the query.  To make
// efficient use of Redis Cluster (without impacting standalone Redis), the
// <ID> is stored in braces (hash tags, e.g. livequery:{1} and
// sql:livequery:{1}), so that the two keys for the same <ID> are always stored
// on the same node (as they hash to the same cluster slot). See
// https://redis.io/topics/cluster-spec#keys-hash-tags for details.
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
	bitsInByte      = 8
	queryKeyPrefix  = "livequery:"
	sqlKeyPrefix    = "sql:"
	queryExpiration = 7 * 24 * time.Hour
)

type redisLiveQuery struct {
	// connection pool
	pool fleet.RedisPool
}

// NewRedisQueryResults creats a new Redis implementation of the
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

func (r *redisLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return errors.New("no hosts targeted")
	}

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

func (r *redisLiveQuery) StopQuery(name string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	targetKey, sqlKey := generateKeys(name)
	if _, err := conn.Do("DEL", targetKey, sqlKey); err != nil {
		return fmt.Errorf("del query keys: %w", err)
	}

	return nil
}

func (r *redisLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	// Get keys for active queries
	queryKeys, err := redis.ScanKeys(r.pool, queryKeyPrefix+"*", 100)
	if err != nil {
		return nil, fmt.Errorf("scan active queries: %w", err)
	}

	keysBySlot := redis.SplitKeysBySlot(r.pool, queryKeys...)
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

		targeted, err := redigo.Int(conn.Receive())
		if err != nil {
			return fmt.Errorf("receive target: %w", err)
		}

		// Be sure to read SQL even if we are not going to include this query.
		// Otherwise we will read an incorrect number of returned results from
		// the pipeline.
		sql, err := redigo.String(conn.Receive())
		if err != nil {
			// Not being able to get the sql for a matched query could mean things
			// have ended up in a weird state. Or it could be that the query was
			// stopped since we did the key scan. In any case, attempt to clean
			// up here.
			_ = r.StopQuery(name)
			return fmt.Errorf("receive sql: %w", err)
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

	return nil
}

// mapBitfield takes the given host IDs and maps them into a bitfield compatible
// with Redis. It is expected that the input IDs are in ascending order.
func mapBitfield(hostIDs []uint) []byte {
	if len(hostIDs) == 0 {
		return []byte{}
	}

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

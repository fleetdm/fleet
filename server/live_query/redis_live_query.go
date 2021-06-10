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
package live_query

import (
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
)

const (
	bitsInByte      = 8
	queryKeyPrefix  = "livequery:"
	sqlKeyPrefix    = "sql:"
	queryExpiration = 7 * 24 * time.Hour
)

type redisLiveQuery struct {
	// connection pool
	pool *redisc.Cluster
}

// NewRedisQueryResults creats a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisLiveQuery(pool *redisc.Cluster) *redisLiveQuery {
	return &redisLiveQuery{pool: pool}
}

func generateKeys(name string) (targetsKey, sqlKey string) {
	return queryKeyPrefix + name, sqlKeyPrefix + queryKeyPrefix + name
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
		return errors.Wrap(err, "set sql")
	}
	_, err = conn.Do("SET", targetKey, targets, "EX", queryExpiration.Seconds())
	if err != nil {
		return errors.Wrap(err, "set targets")
	}

	return nil
}

func (r *redisLiveQuery) StopQuery(name string) error {
	conn := r.pool.Get()
	defer conn.Close()

	targetKey, sqlKey := generateKeys(name)
	if _, err := conn.Do("DEL", targetKey, sqlKey); err != nil {
		return errors.Wrap(err, "del query keys")
	}

	return nil
}

func (r *redisLiveQuery) QueriesForHost(hostID uint) (map[string]string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	// Get keys for active queries
	queryKeys, err := scanKeys(conn, queryKeyPrefix+"*")
	if err != nil {
		return nil, errors.Wrap(err, "scan active queries")
	}

	// Pipeline redis calls to check for this host in the bitfield of the
	// targets of the query.
	for _, key := range queryKeys {
		if err := conn.Send("GETBIT", key, hostID); err != nil {
			return nil, errors.Wrap(err, "getbit query targets")
		}

		// Additionally get SQL even though we don't yet know whether this query
		// is targeted to the host. This allows us to avoid an additional
		// roundtrip to the Redis server and likely has little cost due to the
		// small number of queries and limited size of SQL
		if err = conn.Send("GET", sqlKeyPrefix+key); err != nil {
			return nil, errors.Wrap(err, "get query sql")
		}
	}

	// Flush calls to begin receiving results.
	if err := conn.Flush(); err != nil {
		return nil, errors.Wrap(err, "flush pipeline")
	}

	// Receive target and SQL in order of pipelined calls.
	queries := make(map[string]string)
	for _, key := range queryKeys {
		name := strings.TrimPrefix(key, queryKeyPrefix)

		targeted, err := redis.Int(conn.Receive())
		if err != nil {
			return nil, errors.Wrap(err, "receive target")
		}

		// Be sure to read SQL even if we are not going to include this query.
		// Otherwise we will read an incorrect number of returned results from
		// the pipeline.
		sql, err := redis.String(conn.Receive())
		if err != nil {
			// Not being able to get the sql for a matched could mean things
			// have ended up in a weird state. Or it could be that the query was
			// stopped since we did the key scan. In any case, attempt to clean
			// up here.
			_ = r.StopQuery(name)
			return nil, errors.Wrap(err, "receive sql")
		}

		if targeted == 0 {
			// Host not targeted with this query
			continue
		}

		queries[name] = sql
	}

	return queries, nil
}

func (r *redisLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	conn := r.pool.Get()
	defer conn.Close()

	targetKey, _ := generateKeys(name)

	// Update the bitfield for this host.
	if _, err := conn.Do("SETBIT", targetKey, hostID, 0); err != nil {
		return errors.Wrap(err, "setbit query key")
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

func scanKeys(conn redis.Conn, pattern string) ([]string, error) {
	var keys []string
	cursor := 0
	for {
		res, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", pattern))
		if err != nil {
			return nil, errors.Wrap(err, "scan keys")
		}
		var curKeys []string
		_, err = redis.Scan(res, &cursor, &curKeys)
		if err != nil {
			return nil, errors.Wrap(err, "convert scan results")
		}
		keys = append(keys, curKeys...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

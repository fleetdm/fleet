package live_query

import (
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

const (
	bitsInByte      = 8
	queryKeyPrefix  = "query:"
	queryExpiration = 7 * 24 * time.Hour
)

type redisLiveQuery struct {
	// connection pool
	pool *redis.Pool
}

// NewRedisQueryResults creats a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisLiveQuery(pool *redis.Pool) *redisLiveQuery {
	return &redisLiveQuery{pool: pool}
}

func (r *redisLiveQuery) RunQuery(name, sql string, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return errors.New("no hosts targeted")
	}

	conn := r.pool.Get()
	defer conn.Close()

	// Map the targeted host IDs to a bitfield and store in a key containing the
	// query anme and SQL.
	key := fmt.Sprintf(queryKeyPrefix+"%s:%s", name, sql)
	bitfield := mapBitfield(hostIDs)
	_, err := conn.Do("SET", key, bitfield, "EX", queryExpiration.Seconds())
	if err != nil {
		return errors.Wrap(err, "set query in Redis")
	}
	return nil
}

func (r *redisLiveQuery) StopQuery(name string) error {
	conn := r.pool.Get()
	defer conn.Close()

	// Find key for this query.
	keys, err := scanKeys(conn, queryKeyPrefix+name+":*")
	if err != nil {
		return errors.Wrap(err, "scan for query key")
	}
	if len(keys) == 0 {
		return errors.Errorf("query %s not found", name)
	}
	if len(keys) > 1 {
		return errors.Errorf("found more than one query matching %s", name)
	}

	// Set the bitfield for this host.
	key := keys[0]
	if _, err := conn.Do("DEL", key); err != nil {
		return errors.Wrap(err, "del query key")
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
			return nil, errors.Wrap(err, "getbit query key")
		}
	}

	// Flush calls to begin receiving results.
	if err := conn.Flush(); err != nil {
		return nil, errors.Wrap(err, "flush pipeline")
	}

	// Receive target information in order of pipelined calls.
	queries := make(map[string]string)
	for _, key := range queryKeys {
		targeted, err := redis.Int(conn.Receive())
		if err != nil {
			return nil, errors.Wrap(err, "receive int")
		}
		if targeted == 0 {
			// Host not targeted with this query
			continue
		}

		// Split the key to get the query name and SQL
		splits := strings.SplitN(key, ":", 3)
		if len(splits) != 3 {
			return nil, errors.Errorf("query key did not have 3 components: %s", key)
		}
		name, sql := splits[1], splits[2]
		queries[name] = sql
	}

	return queries, nil
}

func (r *redisLiveQuery) QueryCompletedByHost(name string, hostID uint) error {
	conn := r.pool.Get()
	defer conn.Close()

	// Find key for this query.
	keys, err := scanKeys(conn, queryKeyPrefix+name+":*")
	if err != nil {
		return errors.Wrap(err, "scan for query key")
	}
	if len(keys) == 0 {
		return errors.Errorf("query %s not found", name)
	}
	if len(keys) > 1 {
		return errors.Errorf("found more than one query matching %s", name)
	}

	// Set the bitfield for this host.
	key := keys[0]
	if _, err := conn.Do("SETBIT", key, hostID, 0); err != nil {
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

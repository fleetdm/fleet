package redis

import (
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
)

// this is an adapter type to implement the same Stats method as for
// redisc.Cluster, so both can satisfy the same interface.
type standalonePool struct {
	*redis.Pool
	addr string
}

func (p *standalonePool) Stats() map[string]redis.PoolStats {
	return map[string]redis.PoolStats{
		p.addr: p.Pool.Stats(),
	}
}

// NewRedisPool creates a Redis connection pool using the provided server
// address, password and database.
func NewRedisPool(
	server, password string, database int, useTLS bool, connTimeout, keepAlive time.Duration,
) (fleet.RedisPool, error) {
	cluster := newCluster(server, password, database, useTLS, connTimeout, keepAlive)
	if err := cluster.Refresh(); err != nil {
		if isClusterDisabled(err) || isClusterCommandUnknown(err) {
			// not a Redis Cluster setup, use a standalone Redis pool
			pool, _ := cluster.CreatePool(server)
			cluster.Close()
			return &standalonePool{pool, server}, nil
		}
		return nil, errors.Wrap(err, "refresh cluster")
	}

	return cluster, nil
}

// SplitRedisKeysBySlot takes a list of redis keys and groups them by hash slot
// so that keys in a given group are guaranteed to hash to the same slot, making
// them safe to run e.g. in a pipeline on the same connection or as part of a
// multi-key command in a Redis Cluster setup. When using standalone Redis, it
// simply returns all keys in the same group (i.e. the top-level slice has a
// length of 1).
func SplitRedisKeysBySlot(pool fleet.RedisPool, keys ...string) [][]string {
	if _, isCluster := pool.(*redisc.Cluster); isCluster {
		return redisc.SplitBySlot(keys...)
	}
	return [][]string{keys}
}

// EachRedisNode calls fn for each node in the redis cluster, with a connection
// to that node, until all nodes have been visited. The connection is
// automatically closed after the call. If fn returns an error, the iteration
// of nodes stops and EachRedisNode returns that error. For standalone redis,
// fn is called only once.
func EachRedisNode(pool fleet.RedisPool, fn func(conn redis.Conn) error) error {
	if cluster, isCluster := pool.(*redisc.Cluster); isCluster {
		return cluster.EachNode(false, func(_ string, conn redis.Conn) error {
			return fn(conn)
		})
	}

	conn := pool.Get()
	defer conn.Close()
	return fn(conn)
}

func newCluster(server, password string, database int, useTLS bool, connTimeout, keepAlive time.Duration) *redisc.Cluster {
	return &redisc.Cluster{
		StartupNodes: []string{server},
		CreatePool: func(server string, opts ...redis.DialOption) (*redis.Pool, error) {
			return &redis.Pool{
				MaxIdle:     3,
				IdleTimeout: 240 * time.Second,
				Dial: func() (redis.Conn, error) {
					c, err := redis.Dial(
						"tcp",
						server,
						redis.DialDatabase(database),
						redis.DialUseTLS(useTLS),
						redis.DialConnectTimeout(connTimeout),
						redis.DialKeepAlive(keepAlive),
						// Read/Write timeouts not set here because we may see results
						// only rarely on the pub/sub channel.
					)
					if err != nil {
						return nil, err
					}
					if password != "" {
						if _, err := c.Do("AUTH", password); err != nil {
							c.Close()
							return nil, err
						}
					}
					return c, err
				},
				TestOnBorrow: func(c redis.Conn, t time.Time) error {
					if time.Since(t) < time.Minute {
						return nil
					}
					_, err := c.Do("PING")
					return err
				},
			}, nil
		},
	}
}

func isClusterDisabled(err error) bool {
	return strings.Contains(err.Error(), "ERR This instance has cluster support disabled")
}

// On GCP Memorystore the CLUSTER command is entirely unavailable and fails with
// this error. See
// https://cloud.google.com/memorystore/docs/redis/product-constraints#blocked_redis_commands
func isClusterCommandUnknown(err error) bool {
	return strings.Contains(err.Error(), "ERR unknown command `CLUSTER`")
}

func ScanKeys(pool fleet.RedisPool, pattern string) ([]string, error) {
	var keys []string

	err := EachRedisNode(pool, func(conn redis.Conn) error {
		cursor := 0
		for {
			res, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", pattern))
			if err != nil {
				return errors.Wrap(err, "scan keys")
			}
			var curKeys []string
			_, err = redis.Scan(res, &cursor, &curKeys)
			if err != nil {
				return errors.Wrap(err, "convert scan results")
			}
			keys = append(keys, curKeys...)
			if cursor == 0 {
				return nil
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

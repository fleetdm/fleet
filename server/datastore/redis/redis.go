package redis

import (
	"bufio"
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
func NewRedisPool(server, password string, database int, useTLS bool) (fleet.RedisPool, error) {
	cluster := newCluster(server, password, database, useTLS)
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
		addrs, err := getClusterPrimaryAddrs(cluster)
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			err := func() error {
				// NOTE(mna): using CreatePool means that we respect the redis timeouts
				// and configs.  This is a temporary pool as we can't reuse the
				// (internal) cluster pools for each host at the moment, would require
				// a change to redisc (one that would make sense to make for that
				// use-case of visiting each node, IMO).
				tempPool, err := cluster.CreatePool(addr)
				if err != nil {
					return errors.Wrap(err, "create pool")
				}
				defer tempPool.Close()

				conn := tempPool.Get()
				defer conn.Close()
				return fn(conn)
			}()
			if err != nil {
				return err
			}
		}
		return nil
	}

	conn := pool.Get()
	defer conn.Close()
	return fn(conn)
}

func getClusterPrimaryAddrs(pool *redisc.Cluster) ([]string, error) {
	conn := pool.Get()
	defer conn.Close()
	nodes, err := redis.String(conn.Do("CLUSTER", "NODES"))
	if err != nil {
		return nil, errors.Wrap(err, "get cluster nodes")
	}

	var addrs []string
	s := bufio.NewScanner(strings.NewReader(nodes))
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) > 2 {
			flags := fields[2]
			if strings.Contains(flags, "master") {
				addrField := fields[1]
				if ix := strings.Index(addrField, "@"); ix >= 0 {
					addrField = addrField[:ix]
				}
				addrs = append(addrs, addrField)
			}
		}
	}
	return addrs, nil
}

func newCluster(server, password string, database int, useTLS bool) *redisc.Cluster {
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
						redis.DialConnectTimeout(5*time.Second),
						redis.DialKeepAlive(10*time.Second),
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

package redis

import (
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
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

type clusterPool struct {
	*redisc.Cluster
	followRedirs bool
	readReplica  bool
}

// PoolConfig holds the redis pool configuration options.
type PoolConfig struct {
	Server                    string
	Password                  string
	Database                  int
	UseTLS                    bool
	ConnTimeout               time.Duration
	KeepAlive                 time.Duration
	ConnectRetryAttempts      int
	ClusterFollowRedirections bool
	ClusterReadFromReplica    bool

	// allows for testing dial retries and other dial-related scenarios
	testRedisDialFunc func(net, addr string, opts ...redis.DialOption) (redis.Conn, error)
}

// NewPool creates a Redis connection pool using the provided server
// address, password and database.
func NewPool(config PoolConfig) (fleet.RedisPool, error) {
	cluster := newCluster(config)
	if err := cluster.Refresh(); err != nil {
		if isClusterDisabled(err) || isClusterCommandUnknown(err) {
			// not a Redis Cluster setup, use a standalone Redis pool
			pool, _ := cluster.CreatePool(config.Server)
			cluster.Close()
			return &standalonePool{pool, config.Server}, nil
		}
		return nil, errors.Wrap(err, "refresh cluster")
	}

	return &clusterPool{
		cluster,
		config.ClusterFollowRedirections,
		config.ClusterReadFromReplica,
	}, nil
}

// ReadOnlyConn turns conn into a connection that will try to connect to a
// replica instead of a primary. Note that this is not guaranteed that it will
// do so (there may not be any replica, or due to redirections it may end up on
// a primary, etc.), and it will only try to do so if pool is a Redis Cluster
// pool. The returned connection should only be used to run read-only
// commands.
func ReadOnlyConn(pool fleet.RedisPool, conn redis.Conn) redis.Conn {
	if p, isCluster := pool.(*clusterPool); isCluster && p.readReplica {
		// it only fails if the connection is not a redisc connection or the
		// connection is already bound, in which case we just return the connection
		// as-is.
		_ = redisc.ReadOnlyConn(conn)
	}
	return conn
}

// ConfigureDoer configures conn to follow redirections if the redis
// configuration requested it and the pool is a Redis Cluster pool. If the conn
// is already in error, or if it is not a redisc cluster connection, it is
// returned unaltered.
func ConfigureDoer(pool fleet.RedisPool, conn redis.Conn) redis.Conn {
	if p, isCluster := pool.(*clusterPool); isCluster {
		if err := conn.Err(); err == nil && p.followRedirs {
			rc, err := redisc.RetryConn(conn, 3, 300*time.Millisecond)
			if err == nil {
				return rc
			}
		}
	}
	return conn
}

// SplitKeysBySlot takes a list of redis keys and groups them by hash slot
// so that keys in a given group are guaranteed to hash to the same slot, making
// them safe to run e.g. in a pipeline on the same connection or as part of a
// multi-key command in a Redis Cluster setup. When using standalone Redis, it
// simply returns all keys in the same group (i.e. the top-level slice has a
// length of 1).
func SplitKeysBySlot(pool fleet.RedisPool, keys ...string) [][]string {
	if _, isCluster := pool.(*clusterPool); isCluster {
		return redisc.SplitBySlot(keys...)
	}
	return [][]string{keys}
}

// EachNode calls fn for each node in the redis cluster, with a connection
// to that node, until all nodes have been visited. The connection is
// automatically closed after the call. If fn returns an error, the iteration
// of nodes stops and EachNode returns that error. For standalone redis,
// fn is called only once.
func EachNode(pool fleet.RedisPool, fn func(conn redis.Conn) error) error {
	if cluster, isCluster := pool.(*clusterPool); isCluster {
		return cluster.EachNode(false, func(_ string, conn redis.Conn) error {
			return fn(conn)
		})
	}

	conn := pool.Get()
	defer conn.Close()
	return fn(conn)
}

func newCluster(config PoolConfig) *redisc.Cluster {
	opts := []redis.DialOption{
		redis.DialDatabase(config.Database),
		redis.DialUseTLS(config.UseTLS),
		redis.DialConnectTimeout(config.ConnTimeout),
		redis.DialKeepAlive(config.KeepAlive),
		// Read/Write timeouts not set here because we may see results
		// only rarely on the pub/sub channel.
	}
	if config.Password != "" {
		opts = append(opts, redis.DialPassword(config.Password))
	}

	dialFn := redis.Dial
	if config.testRedisDialFunc != nil {
		dialFn = config.testRedisDialFunc
	}

	return &redisc.Cluster{
		StartupNodes: []string{config.Server},
		CreatePool: func(server string, _ ...redis.DialOption) (*redis.Pool, error) {
			return &redis.Pool{
				MaxIdle:     3,
				IdleTimeout: 240 * time.Second,

				Dial: func() (redis.Conn, error) {
					var conn redis.Conn
					op := func() error {
						c, err := dialFn("tcp", server, opts...)

						var netErr net.Error
						if errors.As(err, &netErr) {
							if netErr.Temporary() || netErr.Timeout() {
								// retryable error
								return err
							}
						}
						if err != nil {
							// at this point, this is a non-retryable error
							return backoff.Permanent(err)
						}

						// success, store the connection to use
						conn = c
						return nil
					}

					if config.ConnectRetryAttempts > 0 {
						boff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(config.ConnectRetryAttempts))
						if err := backoff.Retry(op, boff); err != nil {
							return nil, err
						}
					} else if err := op(); err != nil {
						return nil, err
					}
					return conn, nil
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

package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
)

// this is an adapter type to implement the same Stats method as for
// redisc.Cluster, so both can satisfy the same interface.
type standalonePool struct {
	*redis.Pool
	addr            string
	connWaitTimeout time.Duration
}

func (p *standalonePool) Get() redis.Conn {
	if p.connWaitTimeout <= 0 {
		return p.Pool.Get()
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.connWaitTimeout)
	defer cancel()

	// GetContext always returns an "errorConn" as valid connection when there is
	// an error, so there's no need to care about the second return value (as for
	// the no-wait case, the errorConn will fail on first use with the actual
	// error).
	conn, _ := p.Pool.GetContext(ctx)
	return conn
}

func (p *standalonePool) Stats() map[string]redis.PoolStats {
	return map[string]redis.PoolStats{
		p.addr: p.Pool.Stats(),
	}
}

func (p *standalonePool) Mode() fleet.RedisMode {
	return fleet.RedisStandalone
}

type clusterPool struct {
	*redisc.Cluster
	followRedirs bool
	readReplica  bool
}

func (p *clusterPool) Mode() fleet.RedisMode {
	return fleet.RedisCluster
}

// PoolConfig holds the redis pool configuration options.
type PoolConfig struct {
	Server                    string
	Username                  string
	Password                  string
	Database                  int
	UseTLS                    bool
	ConnTimeout               time.Duration
	KeepAlive                 time.Duration
	ConnectRetryAttempts      int
	ClusterFollowRedirections bool
	ClusterReadFromReplica    bool
	TLSCert                   string
	TLSKey                    string
	TLSCA                     string
	TLSServerName             string
	TLSHandshakeTimeout       time.Duration
	MaxIdleConns              int
	MaxOpenConns              int
	ConnMaxLifetime           time.Duration
	IdleTimeout               time.Duration
	ConnWaitTimeout           time.Duration
	TLSSkipVerify             bool
	WriteTimeout              time.Duration
	ReadTimeout               time.Duration

	// allows for testing dial retries and other dial-related scenarios
	testRedisDialFunc func(net, addr string, opts ...redis.DialOption) (redis.Conn, error)
}

// NewPool creates a Redis connection pool using the provided server
// address, username, password and database.
func NewPool(config PoolConfig) (fleet.RedisPool, error) {
	cluster, err := newCluster(config)
	if err != nil {
		return nil, err
	}
	if err := cluster.Refresh(); err != nil {
		if isClusterDisabled(err) || isClusterCommandUnknown(err) {
			// not a Redis Cluster setup, use a standalone Redis pool
			pool, _ := cluster.CreatePool(config.Server, cluster.DialOptions...)
			cluster.Close()
			return &standalonePool{pool, config.Server, config.ConnWaitTimeout}, nil
		}
		return nil, fmt.Errorf("refresh cluster: %w", err)
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
//
// If replicas is true, it will visit each replica node instead, otherwise the
// primary nodes are visited. Keep in mind that if replicas is true, it will
// visit all known replicas - which is great e.g. to run diagnostics on each
// node, but can be surprising if the goal is e.g. to collect all keys, as it
// is possible that more than one node is acting as replica for the same
// primary, meaning that the same keys could be seen multiple times - you
// should be prepared to handle this scenario. The connection provided to fn is
// not a ReadOnly connection (conn.ReadOnly hasn't been called on it), it is up
// to fn to execute the READONLY redis command if required.
func EachNode(pool fleet.RedisPool, replicas bool, fn func(conn redis.Conn) error) error {
	if cluster, isCluster := pool.(*clusterPool); isCluster {
		return cluster.EachNode(replicas, func(_ string, conn redis.Conn) error {
			return fn(conn)
		})
	}

	conn := pool.Get()
	defer conn.Close()
	return fn(conn)
}

// BindConn binds the connection to the redis node that serves those keys.
// In a Redis Cluster setup, all keys must hash to the same slot, otherwise
// an error is returned. In a Redis Standalone setup, it is a no-op and never
// fails. On successful return, the connection is ready to be used with those
// keys.
func BindConn(pool fleet.RedisPool, conn redis.Conn, keys ...string) error {
	if _, isCluster := pool.(*clusterPool); isCluster {
		return redisc.BindConn(conn, keys...)
	}
	return nil
}

// PublishHasListeners is like the PUBLISH redis command, but it also returns a
// boolean indicating if channel still has subscribed listeners. It is required
// because the redis command only returns the count of subscribers active on
// the same node as the one that is used to publish, which may not always be
// the case in Redis Cluster (especially with the read from replica option
// set).
//
// In Standalone mode, it is the same as PUBLISH (with the count of subscribers
// turned into a boolean), and in Cluster mode, if the count returned by
// PUBLISH is 0, it gets the number of subscribers on each node in the cluster
// to get the accurate count.
func PublishHasListeners(pool fleet.RedisPool, conn redis.Conn, channel, message string) (bool, error) {
	n, err := redis.Int(conn.Do("PUBLISH", channel, message))
	if n > 0 || err != nil {
		return n > 0, err
	}

	// otherwise n == 0, check the actual number of subscribers if this is a
	// redis cluster.
	if _, isCluster := pool.(*clusterPool); !isCluster {
		return false, nil
	}

	errDone := errors.New("done")
	var count int

	// subscribers can be subscribed on replicas, so we need to iterate on both
	// primaries and replicas.
	for _, replicas := range []bool{true, false} {
		err = EachNode(pool, replicas, func(conn redis.Conn) error {
			res, err := redis.Values(conn.Do("PUBSUB", "NUMSUB", channel))
			if err != nil {
				return err
			}
			var (
				name string
				n    int
			)
			_, err = redis.Scan(res, &name, &n)
			if err != nil {
				return err
			}
			count += n
			if count > 0 {
				// end early if we know it has subscribers
				return errDone
			}
			return nil
		})

		if err == errDone {
			break
		}
	}

	// if it completed successfully
	if err == nil || err == errDone {
		return count > 0, nil
	}
	return false, fmt.Errorf("checking for active subscribers: %w", err)
}

func newCluster(conf PoolConfig) (*redisc.Cluster, error) {
	opts := []redis.DialOption{
		redis.DialDatabase(conf.Database),
		redis.DialUseTLS(conf.UseTLS),
		redis.DialConnectTimeout(conf.ConnTimeout),
		redis.DialKeepAlive(conf.KeepAlive),
		redis.DialUsername(conf.Username),
		redis.DialPassword(conf.Password),
		redis.DialWriteTimeout(conf.WriteTimeout),
		redis.DialReadTimeout(conf.ReadTimeout),
	}

	if conf.UseTLS {
		tlsCfg := config.TLS{
			TLSCA:         conf.TLSCA,
			TLSCert:       conf.TLSCert,
			TLSKey:        conf.TLSKey,
			TLSServerName: conf.TLSServerName,
		}
		cfg, err := tlsCfg.ToTLSConfig()
		if err != nil {
			return nil, err
		}
		cfg.InsecureSkipVerify = conf.TLSSkipVerify

		opts = append(opts,
			redis.DialTLSConfig(cfg),
			redis.DialUseTLS(true),
			redis.DialTLSHandshakeTimeout(conf.TLSHandshakeTimeout))
	}

	dialFn := redis.Dial
	if conf.testRedisDialFunc != nil {
		dialFn = conf.testRedisDialFunc
	}

	return &redisc.Cluster{
		StartupNodes: []string{conf.Server},
		PoolWaitTime: conf.ConnWaitTimeout,
		DialOptions:  opts,
		CreatePool: func(server string, opts ...redis.DialOption) (*redis.Pool, error) {
			return &redis.Pool{
				MaxIdle:         conf.MaxIdleConns,
				MaxActive:       conf.MaxOpenConns,
				IdleTimeout:     conf.IdleTimeout,
				MaxConnLifetime: conf.ConnMaxLifetime,
				Wait:            conf.ConnWaitTimeout > 0,

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

					if conf.ConnectRetryAttempts > 0 {
						boff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(),
							uint64(conf.ConnectRetryAttempts)) //nolint:gosec // G115 false positive
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
	}, nil
}

func isClusterDisabled(err error) bool {
	return strings.Contains(err.Error(), "ERR This instance has cluster support disabled") ||
		strings.Contains(err.Error(), "NOPERM this user has no permissions to run the 'cluster' command")
}

// On GCP Memorystore the CLUSTER command is entirely unavailable and fails with
// this error. See
// https://cloud.google.com/memorystore/docs/redis/product-constraints#blocked_redis_commands
//
// At some point it seems like the error message changed from wrapping the
// command name with backticks to single quotes.
//
// On RedisLabs, user reports indicate that the CLUSTER command fails with "ERR
// command is not allowed" when cluster mode is disabled.
func isClusterCommandUnknown(err error) bool {
	return strings.Contains(err.Error(), "ERR unknown command `CLUSTER`") ||
		strings.Contains(err.Error(), "ERR unknown command 'CLUSTER'") ||
		strings.Contains(err.Error(), "ERR unknown command CLUSTER") ||
		strings.Contains(err.Error(), `ERR unknown command "CLUSTER"`) ||
		strings.Contains(err.Error(), `ERR command is not allowed`)
}

func ScanKeys(pool fleet.RedisPool, pattern string, count int) ([]string, error) {
	var keys []string

	err := EachNode(pool, false, func(conn redis.Conn) error {
		cursor := 0
		for {
			res, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", pattern, "COUNT", count))
			if err != nil {
				return fmt.Errorf("scan keys: %w", err)
			}
			var curKeys []string
			_, err = redis.Scan(res, &cursor, &curKeys)
			if err != nil {
				return fmt.Errorf("convert scan results: %w", err)
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

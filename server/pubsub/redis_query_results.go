package pubsub

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
)

type redisQueryResults struct {
	// connection pool
	pool             fleet.RedisPool
	duplicateResults bool
}

var _ fleet.QueryResultStore = &redisQueryResults{}

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

// EachRedisNode calls fn for each node in the redis cluster, with a connection
// to that node, until all nodes have been visited. The connection is
// automatically closed after the call. If fn returns an error, the iteration
// of nodes stops and EachRedisNode returns that error. For standalone redis,
// fn is called only once.
func EachRedisNode(pool fleet.RedisPool, fn func(conn redis.Conn) error) error {
	// TODO: ideally, NewRedisPool, EachRedisNode and other helper functions
	// would live in a datastore/redis package or something like that? This is
	// not related to pubsub specifically.

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

// NewRedisQueryResults creats a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisQueryResults(pool fleet.RedisPool, duplicateResults bool) *redisQueryResults {
	return &redisQueryResults{pool: pool, duplicateResults: duplicateResults}
}

func pubSubForID(id uint) string {
	return fmt.Sprintf("results_%d", id)
}

// Pool returns the redisc connection pool (used in tests).
func (r *redisQueryResults) Pool() fleet.RedisPool {
	return r.pool
}

func (r *redisQueryResults) WriteResult(result fleet.DistributedQueryResult) error {
	conn := r.pool.Get()
	defer conn.Close()

	channelName := pubSubForID(result.DistributedQueryCampaignID)

	jsonVal, err := json.Marshal(&result)
	if err != nil {
		return errors.Wrap(err, "marshalling JSON for result")
	}

	n, err := redis.Int(conn.Do("PUBLISH", channelName, string(jsonVal)))

	if n != 0 && r.duplicateResults {
		// Ignore errors, duplicate result publishing is on a "best-effort" basis.
		_, _ = redis.Int(conn.Do("PUBLISH", "LQDuplicate", string(jsonVal)))
	}

	if err != nil {
		return errors.Wrap(err, "PUBLISH failed to channel "+channelName)
	}
	if n == 0 {
		return noSubscriberError{channelName}
	}

	return nil
}

// writeOrDone tries to write the item into the channel taking into account context.Done(). If context is done, returns
// true, otherwise false
func writeOrDone(ctx context.Context, ch chan<- interface{}, item interface{}) bool {
	select {
	case ch <- item:
	case <-ctx.Done():
		return true
	}
	return false
}

// receiveMessages runs in a goroutine, forwarding messages from the Pub/Sub
// connection over the provided channel. This effectively allows a select
// statement to run on conn.Receive() (by selecting on outChan that is
// passed into this function)
func receiveMessages(ctx context.Context, conn *redis.PubSubConn, outChan chan<- interface{}) {
	defer close(outChan)
	// conn.Close() needs to be here in this function because Receive and Close should not be called
	// concurrently. Otherwise we end up with a hang when Close is called.
	// See https://github.com/gomodule/redigo/issues/187.
	defer conn.Close()

	for {
		// Add a timeout to try to cleanup in the case the server has somehow gone completely unresponsive.
		msg := conn.ReceiveWithTimeout(1 * time.Hour)

		// Pass the message back to ReadChannel.
		if writeOrDone(ctx, outChan, msg) {
			return
		}

		switch msg := msg.(type) {
		case error:
			// If an error occurred (i.e. connection was closed), then we should exit.
			return
		case redis.Subscription:
			// If the subscription count is 0, the ReadChannel call that invoked this goroutine has unsubscribed,
			// and we can exit.
			if msg.Count == 0 {
				return
			}
		}
	}
}

func (r *redisQueryResults) ReadChannel(ctx context.Context, query fleet.DistributedQueryCampaign) (<-chan interface{}, error) {
	outChannel := make(chan interface{})
	msgChannel := make(chan interface{})

	conn := r.pool.Get()
	psc := &redis.PubSubConn{Conn: conn}
	pubSubName := pubSubForID(query.ID)
	if err := psc.Subscribe(pubSubName); err != nil {
		// Explicit conn.Close() here because we can't defer it until in the goroutine
		_ = conn.Close()
		return nil, errors.Wrapf(err, "subscribe to channel %s", pubSubName)
	}

	// Run a separate goroutine feeding redis messages into
	// msgChannel
	go receiveMessages(ctx, psc, msgChannel)

	go func() {
		// Unsubscribe here, but do not Close. This allows receiveMessages to finish with the final
		// receive and non-concurrently call the Close.
		defer psc.Unsubscribe(pubSubName)
		defer close(outChannel)

		for {
			// Loop reading messages from conn.Receive() (via msgChannel) until the context is cancelled.
			select {
			case msg, ok := <-msgChannel:
				if !ok {
					writeOrDone(ctx, outChannel, errors.New("unexpected exit in receiveMessages"))
					return
				}

				switch msg := msg.(type) {
				case redis.Message:
					var res fleet.DistributedQueryResult
					err := json.Unmarshal(msg.Data, &res)
					if err != nil {
						if writeOrDone(ctx, outChannel, err) {
							return
						}
					}
					if writeOrDone(ctx, outChannel, res) {
						return
					}
				case error:
					if writeOrDone(ctx, outChannel, errors.Wrap(msg, "read from redis")) {
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return outChannel, nil
}

// HealthCheck verifies that the redis backend can be pinged, returning an error
// otherwise.
func (r *redisQueryResults) HealthCheck() error {
	conn := r.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PING"); err != nil {
		return errors.Wrap(err, "reading from redis")
	}
	return nil
}

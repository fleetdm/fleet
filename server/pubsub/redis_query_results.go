package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
)

type redisQueryResults struct {
	pool             fleet.RedisPool
	duplicateResults bool
	logger           log.Logger
}

var _ fleet.QueryResultStore = &redisQueryResults{}

// NewRedisQueryResults creats a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisQueryResults(pool fleet.RedisPool, duplicateResults bool, logger log.Logger) *redisQueryResults {
	return &redisQueryResults{
		pool:             pool,
		duplicateResults: duplicateResults,
		logger:           logger,
	}
}

func pubSubForID(id uint) string {
	return fmt.Sprintf("results_%d", id)
}

// Pool returns the redisc connection pool (used in tests).
func (r *redisQueryResults) Pool() fleet.RedisPool {
	return r.pool
}

func (r *redisQueryResults) WriteResult(result fleet.DistributedQueryResult) error {
	// pub-sub can publish and listen on any node in the cluster
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	channelName := pubSubForID(result.DistributedQueryCampaignID)

	jsonVal, err := json.Marshal(&result)
	if err != nil {
		return fmt.Errorf("marshalling JSON for result: %w", err)
	}

	hasSubs, err := redis.PublishHasListeners(r.pool, conn, channelName, string(jsonVal))

	if hasSubs && r.duplicateResults {
		// Ignore errors, duplicate result publishing is on a "best-effort" basis.
		_, _ = redigo.Int(conn.Do("PUBLISH", "LQDuplicate", string(jsonVal)))
	}

	if err != nil {
		return fmt.Errorf("PUBLISH failed to channel "+channelName+": %w", err)
	}
	if !hasSubs {
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
func receiveMessages(ctx context.Context, conn *redigo.PubSubConn, outChan chan<- interface{}, logger log.Logger) {
	defer close(outChan)

	for {
		beforeReceive := time.Now()

		// Add a timeout to try to cleanup in the case the server has somehow gone
		// completely unresponsive.
		msg := conn.ReceiveWithTimeout(1 * time.Hour)

		if recvTime := time.Since(beforeReceive); recvTime > time.Minute {
			level.Info(logger).Log("msg", "conn.ReceiveWithTimeout connection was blocked for significant time", "duration", recvTime, "connection", fmt.Sprintf("%p", conn))
		}

		// Pass the message back to ReadChannel.
		if writeOrDone(ctx, outChan, msg) {
			return
		}

		switch msg := msg.(type) {
		case error:
			// If an error occurred (i.e. connection was closed), then we should exit.
			level.Error(logger).Log("msg", "conn.ReceiveWithTimeout failed", "err", msg)
			return
		case redigo.Subscription:
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

	// pub-sub can publish and listen on any node in the cluster
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	psc := &redigo.PubSubConn{Conn: conn}
	pubSubName := pubSubForID(query.ID)
	if err := psc.Subscribe(pubSubName); err != nil {
		// Explicit conn.Close() here because we can't defer it until in the goroutine
		_ = conn.Close()
		return nil, ctxerr.Wrapf(ctx, err, "subscribe to channel %s", pubSubName)
	}

	var wg sync.WaitGroup

	logger := log.With(r.logger, "campaignID", query.ID)

	// Run a separate goroutine feeding redis messages into msgChannel.
	wg.Add(+1)
	go func() {
		defer wg.Done()

		receiveMessages(ctx, psc, msgChannel, logger)
	}()

	wg.Add(+1)
	go func() {
		defer wg.Done()
		defer close(outChannel)

		for {
			// Loop reading messages from conn.Receive() (via msgChannel) until the context is cancelled.
			select {
			case msg, ok := <-msgChannel:
				if !ok {
					level.Error(logger).Log("msg", "unexpected exit in receiveMessages")
					// NOTE(lucas): The below error string should not be modified. The UI is relying on it to detect
					// when Fleet's connection to Redis has been interrupted unexpectedly.
					//
					// TODO(lucas): We should add a unit test (at the time it required many changes to this production code
					// which increases risk).
					writeOrDone(ctx, outChannel, ctxerr.Errorf(ctx, "unexpected exit in receiveMessages, campaignID=%d", query.ID))
					return
				}

				switch msg := msg.(type) {
				case redigo.Message:
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
					level.Error(logger).Log("msg", "error received from pubsub channel", "err", msg)
					if writeOrDone(ctx, outChannel, ctxerr.Wrap(ctx, msg, "read from redis")) {
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		psc.Unsubscribe(pubSubName) //nolint:errcheck
		conn.Close()
		level.Debug(logger).Log("msg", "proper close of Redis connection in ReadChannel", "connection", fmt.Sprintf("%p", conn))
	}()

	return outChannel, nil
}

// HealthCheck verifies that the redis backend can be pinged, returning an error
// otherwise.
func (r *redisQueryResults) HealthCheck() error {
	conn := r.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PING"); err != nil {
		return fmt.Errorf("reading from redis: %w", err)
	}
	return nil
}

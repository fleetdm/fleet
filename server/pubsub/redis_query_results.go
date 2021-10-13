package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

type redisQueryResults struct {
	// connection pool
	pool             fleet.RedisPool
	duplicateResults bool
}

var _ fleet.QueryResultStore = &redisQueryResults{}

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
	// pub-sub can publish and listen on any node in the cluster
	conn := redis.ReadOnlyConn(r.pool, r.pool.Get())
	defer conn.Close()

	channelName := pubSubForID(result.DistributedQueryCampaignID)

	jsonVal, err := json.Marshal(&result)
	if err != nil {
		return errors.Wrap(err, "marshalling JSON for result")
	}

	hasSubs, err := redis.PublishHasListeners(r.pool, conn, channelName, string(jsonVal))

	if hasSubs && r.duplicateResults {
		// Ignore errors, duplicate result publishing is on a "best-effort" basis.
		_, _ = redigo.Int(conn.Do("PUBLISH", "LQDuplicate", string(jsonVal)))
	}

	if err != nil {
		return errors.Wrap(err, "PUBLISH failed to channel "+channelName)
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
func receiveMessages(ctx context.Context, conn *redigo.PubSubConn, outChan chan<- interface{}) {
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

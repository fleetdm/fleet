package pubsub

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/garyburd/redigo/redis"
	"github.com/kolide/kolide-ose/server/kolide"
)

type redisQueryResults struct {
	// connection pool
	pool *redis.Pool
}

var _ kolide.QueryResultStore = &redisQueryResults{}

// NewRedisPool creates a Redis connection pool using the provided server
// address and password.
func NewRedisPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
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
	}
}

// NewRedisQueryResults creats a new Redis implementation of the
// QueryResultStore interface using the provided Redis connection pool.
func NewRedisQueryResults(pool *redis.Pool) *redisQueryResults {
	return &redisQueryResults{pool: pool}
}

func pubSubForID(id uint) string {
	return fmt.Sprintf("results_%d", id)
}

func (r *redisQueryResults) WriteResult(result kolide.DistributedQueryResult) error {
	conn := r.pool.Get()
	defer conn.Close()

	channelName := pubSubForID(result.DistributedQueryCampaignID)

	jsonVal, err := json.Marshal(&result)
	if err != nil {
		return errors.New("error marshalling JSON for writing result: " + err.Error())
	}

	n, err := redis.Int(conn.Do("PUBLISH", channelName, string(jsonVal)))
	if err != nil {
		return fmt.Errorf("PUBLISH failed to channel %s: %s", channelName, err.Error())
	}
	if n == 0 {
		return fmt.Errorf("no subscribers for channel %s", channelName)
	}

	return nil
}

// receiveMessages runs in a goroutine, forwarding messages from the Pub/Sub
// connection over the provided channel. This effectively allows a select
// statement to run on conn.Receive() (by running on the channel that is being
// fed by this function)
func receiveMessages(conn *redis.PubSubConn, outChan chan<- interface{}) {
	defer func() {
		close(outChan)
	}()

	for {
		msg := conn.Receive()
		outChan <- msg
		switch msg := msg.(type) {
		case error:
			// If an error occurred (i.e. connection was closed),
			// then we should exit
			return
		case redis.Subscription:
			// If the subscription count is 0, the ReadChannel call
			// that invoked this goroutine has unsubscribed, and we
			// can exit
			if msg.Count == 0 {
				return
			}
		}
	}
}

func (r *redisQueryResults) ReadChannel(ctx context.Context, query kolide.DistributedQueryCampaign) (<-chan interface{}, error) {
	outChannel := make(chan interface{})

	conn := redis.PubSubConn{Conn: r.pool.Get()}

	pubSubName := pubSubForID(query.ID)
	conn.Subscribe(pubSubName)

	msgChannel := make(chan interface{})
	// Run a separate goroutine feeding redis messages into
	// msgChannel
	go receiveMessages(&conn, msgChannel)

	go func() {
		defer close(outChannel)
		defer conn.Close()

		for {
			// Loop reading messages from conn.Receive() (via
			// msgChannel) until the context is cancelled.
			select {
			case msg, ok := <-msgChannel:
				if !ok {
					return
				}
				switch msg := msg.(type) {
				case redis.Message:
					var res kolide.DistributedQueryResult
					err := json.Unmarshal(msg.Data, &res)
					if err != nil {
						outChannel <- err
					}
					outChannel <- res
				case error:
					outChannel <- msg
				}

			case <-ctx.Done():
				conn.Unsubscribe()

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

	_, err := conn.Do("PING")
	return err
}

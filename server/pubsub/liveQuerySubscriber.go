package pubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	liveQueryChannel = "live_query"
)

type liveQuerySubscriber struct {
	pool fleet.RedisPool
}

func NewLiveQuerySubscriber(pool fleet.RedisPool) *liveQuerySubscriber {
	return &liveQuerySubscriber{
		pool: pool,
	}
}

func (l *liveQuerySubscriber) Start(ctx context.Context, chMap *SafeHostHostMap, logger log.Logger) error {
	conn := redis.ReadOnlyConn(l.pool, (l.pool).Get())
	psc := &redigo.PubSubConn{Conn: conn}

	if err := psc.Subscribe(liveQueryChannel); err != nil {
		_ = conn.Close()
		return fmt.Errorf("subscribe to channel %s: %w", liveQueryChannel, err)
	}

	var wg sync.WaitGroup

	// Listen to Redis PubSub
	wg.Add(1)
	go func() {
		defer wg.Done()
		processSubscriptionMessages(ctx, psc, liveQueryChannel, chMap, logger)
	}()

	// Close resources when all goroutines have finished
	go func() {
		wg.Wait()
		psc.Unsubscribe(liveQueryChannel) //nolint:errcheck
		conn.Close()
	}()

	return nil
}

func processSubscriptionMessages(ctx context.Context, conn *redigo.PubSubConn, channelName string, chMap *SafeHostHostMap, logger log.Logger) {
	for {
		msg := conn.ReceiveWithTimeout(1 * time.Hour)
		if msg == nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		fmt.Println("Received message from Live Query Chan: ", msg)

		fmt.Printf("sending to %d hosts", chMap.Len())
		// send message to all channels in map
		chMap.BroadcastSignalToAllHosts()
	}
}

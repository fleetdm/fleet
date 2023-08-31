package pubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	liveQueryChannel = "live_query"
)

type HostLiveQueryChannelMap struct {
	sync.RWMutex
	channels map[string]chan struct{}
}

func NewLiveQuerySubscriber(ctx context.Context, pool *redigo.Pool, chMap *HostLiveQueryChannelMap, logger log.Logger) error {
	// Initialize map of channels if it is nil
	if chMap.channels == nil {
		return fmt.Errorf("hostlivequery channel map is nil")
	}

	conn := pool.Get()
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

func processSubscriptionMessages(ctx context.Context, conn *redigo.PubSubConn, channelName string, chMap *HostLiveQueryChannelMap, logger log.Logger) {
	for {
		msg := conn.ReceiveWithTimeout(1 * time.Hour)
		if msg == nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		// send message to all channels in map
		chMap.RLock()
		for _, ch := range chMap.channels {
			select {
			case ch <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
		chMap.RUnlock()
	}
}

// func main() {
// 	// Create a new Redis pool (you'll need to replace this with your actual Redis pool)
// 	pool := redigo.Pool{
// 		Dial: func() (redigo.Conn, error) {
// 			return redigo.Dial("tcp", ":6379")
// 		},
// }

// 	}

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	logger := log.NewNopLogger() // Replace with your logger
// 	channelName := "your_custom_channel_name"
// 	chMap := &ChannelMap{}

// 	err := NewSubscriberRoutine(ctx, pool, channelName, chMap, logger)
// 	if err != nil {
// 		fmt.Printf("Error initializing subscriber: %v\n", err)
// 		return
// 	}

// 	// Assume you have some logic to know which keys are relevant at the moment
// 	relevantKey := "some_key_based_on_msg"

// 	// Access the Go channel for a specific key
// 	chMap.RLock()
// 	ch, ok := chMap.channels[relevantKey]
// 	chMap.RUnlock()

// 	if ok {
// 		// Do something with the Go channel (e.g., read messages from it)
// 		for msg := range ch {
// 			fmt.Printf("Received message: %v\n", msg)
// 		}
// 	}
// }

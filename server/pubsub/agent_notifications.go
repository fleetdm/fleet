package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// agentNotificationsChannel is the Redis pub/sub channel for agent check-in
// wake-ups. A single channel is shared by all server instances: every instance
// subscribes at boot and each delivers notifications only to the agents whose
// WebSocket connections it holds.
const agentNotificationsChannel = "agent_notifications"

// publishHostIDsChunkSize bounds the size of a single published message.
const publishHostIDsChunkSize = 10_000

// subscribeRetryBaseInterval/Max bound the resubscribe backoff when the
// subscription connection to Redis fails.
const (
	subscribeRetryBaseInterval = 1 * time.Second
	subscribeRetryMaxInterval  = 30 * time.Second
)

// AgentNotification is the payload published on the agent notifications
// channel: only the notification type, the targeted host IDs and the
// triggering reason (e.g. "live-<campaign ID>") — never query content or
// results.
type AgentNotification struct {
	Type    string `json:"type"`
	HostIDs []uint `json:"host_ids"`
	Reason  string `json:"reason,omitempty"`
}

// RedisAgentNotifier publishes and subscribes to agent check-in wake-ups over
// Redis pub/sub. It implements fleet.AgentCheckInNotifier.
type RedisAgentNotifier struct {
	pool   fleet.RedisPool
	logger *slog.Logger
}

var _ fleet.AgentCheckInNotifier = (*RedisAgentNotifier)(nil)

func NewRedisAgentNotifier(pool fleet.RedisPool, logger *slog.Logger) *RedisAgentNotifier {
	return &RedisAgentNotifier{
		pool:   pool,
		logger: logger,
	}
}

// NotifyAgentsForLiveQuery publishes a distributed/read wake-up for the hosts
// targeted by a newly created live query campaign. In Redis Cluster, PUBLISH
// is broadcast cluster-wide, so publishing on any node reaches all
// subscribers.
func (n *RedisAgentNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	conn := redis.ReadOnlyConn(n.pool, n.pool.Get())
	defer conn.Close()

	for chunk := range slices.Chunk(hostIDs, publishHostIDsChunkSize) {
		payload, err := json.Marshal(AgentNotification{
			Type:    fleet.AgentWSMessageTypeDistributedRead,
			HostIDs: chunk,
			Reason:  fleet.AgentWSReasonLiveQuery(campaignID),
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal agent notification")
		}
		if _, err := conn.Do("PUBLISH", agentNotificationsChannel, payload); err != nil {
			return ctxerr.Wrap(ctx, err, "publish agent notification")
		}
	}
	return nil
}

// Subscribe is the boot-time, process-wide subscription loop: it delivers
// every notification published on the agent notifications channel to the
// deliver callback until ctx is done, resubscribing with backoff when the
// Redis connection fails.
//
// Run it in its own goroutine; deliver is called synchronously from the loop
// and must not block for long.
func (n *RedisAgentNotifier) Subscribe(ctx context.Context, deliver func(AgentNotification)) {
	backoff := subscribeRetryBaseInterval
	for {
		if ctx.Err() != nil {
			return
		}

		err := n.subscribeOnce(ctx, deliver, func() { backoff = subscribeRetryBaseInterval })
		if ctx.Err() != nil {
			return
		}
		n.logger.ErrorContext(ctx, "agent notifications subscription failed; resubscribing",
			"err", err, "backoff", backoff.String())

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff = min(backoff*2, subscribeRetryMaxInterval)
	}
}

// subscribeOnce holds one subscription until it fails or ctx is done. The
// onSubscribed callback runs once the SUBSCRIBE succeeds (used to reset the
// caller's backoff).
func (n *RedisAgentNotifier) subscribeOnce(ctx context.Context, deliver func(AgentNotification), onSubscribed func()) error {
	// pub-sub can publish and listen on any node in the cluster
	conn := redis.ReadOnlyConn(n.pool, n.pool.Get())
	defer conn.Close()

	psc := &redigo.PubSubConn{Conn: conn}
	if err := psc.Subscribe(agentNotificationsChannel); err != nil {
		return ctxerr.Wrap(ctx, err, "subscribe to agent notifications channel")
	}
	onSubscribed()

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	// the conn must not be closed (deferred above) while the unsubscribe may still be writing
	defer wg.Wait()
	defer close(done)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			// Unsubscribe when ctx is done to unblock ReceiveWithTimeout: redigo allows one
			// concurrent sender and one receiver on a conn, but closing a pooled conn while
			// another goroutine receives on it races. Closing is the fallback if the
			// unsubscribe itself cannot be sent, as the conn is already broken then.
			if err := psc.Unsubscribe(agentNotificationsChannel); err != nil {
				_ = conn.Close()
			}
		case <-done:
		}
	}()

	for {
		switch msg := psc.ReceiveWithTimeout(1 * time.Hour).(type) {
		case redigo.Message:
			var notification AgentNotification
			if err := json.Unmarshal(msg.Data, &notification); err != nil {
				n.logger.ErrorContext(ctx, "unmarshal agent notification", "err", err)
				continue
			}
			deliver(notification)
		case error:
			return ctxerr.Wrap(ctx, msg, "receive agent notification")
		case redigo.Subscription:
			// Count reaches 0 once the ctx-done unsubscribe is confirmed.
			if msg.Count == 0 {
				return nil
			}
		}
	}
}

// DelayedAgentNotifier decorates an AgentCheckInNotifier, delaying each
// notification by a fixed duration before publishing it in the background.
//
// It exists because of the live query store's in-memory active-queries cache
// (see live_query.NewRedisLiveQuery): a notified agent reads within
// milliseconds, and a read served from a cache snapshot predating the
// campaign misses it. Delaying by at least the cache TTL guarantees every
// instance's stale snapshot has expired by the time a notified agent reads.
type DelayedAgentNotifier struct {
	inner  fleet.AgentCheckInNotifier
	delay  time.Duration
	logger *slog.Logger
}

func NewDelayedAgentNotifier(inner fleet.AgentCheckInNotifier, delay time.Duration, logger *slog.Logger) *DelayedAgentNotifier {
	return &DelayedAgentNotifier{inner: inner, delay: delay, logger: logger}
}

var _ fleet.AgentCheckInNotifier = (*DelayedAgentNotifier)(nil)

// NotifyAgentsForLiveQuery schedules the notification to be published after
// the configured delay and returns immediately; publish errors are logged
// rather than returned.
func (n *DelayedAgentNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	// The campaign-creation request context ends as soon as the campaign is
	// created; detach so the delayed publish is not canceled with it.
	ctx = context.WithoutCancel(ctx)
	time.AfterFunc(n.delay, func() {
		if err := n.inner.NotifyAgentsForLiveQuery(ctx, hostIDs, campaignID); err != nil {
			n.logger.ErrorContext(ctx, "delayed notify agents for live query",
				"campaign_id", campaignID, "err", err)
		}
	})
	return nil
}

package pubsub

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
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
// channel: only the notification type, the targeted hosts and the triggering
// reason (e.g. "live-<campaign ID>") — never query content or results.
type AgentNotification struct {
	Type    string `json:"type"`
	HostIDs []uint `json:"host_ids"`
	Reason  string `json:"reason,omitempty"`
	// Scope, when set, targets every host in scope instead of HostIDs (see
	// fleet.AgentNotificationScope).
	Scope fleet.AgentNotificationScope `json:"scope,omitempty"`
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
// targeted by a newly created live query campaign.
func (n *RedisAgentNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	return n.publishHosts(ctx, fleet.AgentWSMessageTypeDistributedRead, fleet.AgentWSReasonLiveQuery(campaignID), hostIDs)
}

// NotifyOrbitConfigHosts publishes an orbit/config wake-up for hostIDs.
func (n *RedisAgentNotifier) NotifyOrbitConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	return n.publishHosts(ctx, fleet.AgentWSMessageTypeOrbitConfig, reason, hostIDs)
}

// NotifyOrbitConfigScope publishes a single orbit/config wake-up for every
// host in scope; each instance resolves the scope against the connections it
// holds.
func (n *RedisAgentNotifier) NotifyOrbitConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	return n.publish(ctx, AgentNotification{
		Type:   fleet.AgentWSMessageTypeOrbitConfig,
		Reason: reason,
		Scope:  scope,
	})
}

func (n *RedisAgentNotifier) publishHosts(ctx context.Context, msgType, reason string, hostIDs []uint) error {
	for chunk := range slices.Chunk(hostIDs, publishHostIDsChunkSize) {
		if err := n.publish(ctx, AgentNotification{Type: msgType, HostIDs: chunk, Reason: reason}); err != nil {
			return err
		}
	}
	return nil
}

// publish sends one notification. In Redis Cluster, PUBLISH is broadcast
// cluster-wide, so publishing on any node reaches all subscribers.
func (n *RedisAgentNotifier) publish(ctx context.Context, notification AgentNotification) error {
	payload, err := json.Marshal(notification)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal agent notification")
	}

	conn := redis.ReadOnlyConn(n.pool, n.pool.Get())
	defer conn.Close()
	if _, err := conn.Do("PUBLISH", agentNotificationsChannel, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "publish agent notification")
	}
	return nil
}

// Subscribe is the boot-time, process-wide subscription loop: it delivers
// every notification published on the agent notifications channel to the
// deliver callback until ctx is done, resubscribing with backoff when the
// Redis connection fails.
//
// Notifications published while the subscription is down are lost, so
// onResubscribe (if not nil) is called each time the subscription is
// re-established after a failure, letting the caller resync its agents.
//
// Run it in its own goroutine; deliver and onResubscribe are called
// synchronously from the loop and must not block for long.
func (n *RedisAgentNotifier) Subscribe(ctx context.Context, deliver func(AgentNotification), onResubscribe func()) {
	backoff := subscribeRetryBaseInterval
	failed := false
	for {
		if ctx.Err() != nil {
			return
		}

		err := n.subscribeOnce(ctx, deliver, func() {
			backoff = subscribeRetryBaseInterval
			if failed && onResubscribe != nil {
				onResubscribe()
			}
		})
		failed = true
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

// DelayedAgentNotifier decorates an AgentCheckInNotifier, delaying each kind
// of notification by its own duration before publishing it in the background.
//
// The delays let caches that the notified agent's next request reads from
// expire first, since a notified agent calls back within milliseconds:
//   - Live queries: the live query store's in-memory active-queries cache
//     (see live_query.NewRedisLiveQuery); a read served from a snapshot
//     predating the campaign misses it.
//   - Scoped orbit config: the per-instance fleet agent options and MDM config
//     caches (see cached_mysql) that GetOrbitConfig reads.
//
// Host orbit config notifications are fired after the change commits and read
// nothing cached; their delay is a batching window instead: notifications with
// the same reason within it are merged into one publish, since writers notify
// per host (e.g. a cron activating scripts on thousands of hosts).
//
// Pending notifications are held in memory only: those of an instance that
// exits before publishing them are lost, and agents pick the change up on
// their next poll instead.
type DelayedAgentNotifier struct {
	inner  fleet.AgentCheckInNotifier
	delays AgentNotifierDelays
	logger *slog.Logger

	mu sync.Mutex
	// pendingHosts holds the host IDs of the orbit config notifications
	// waiting for their batching window to end, by reason.
	pendingHosts map[string]map[uint]struct{}
}

// AgentNotifierDelays holds the delay of each kind of notification.
type AgentNotifierDelays struct {
	LiveQuery        time.Duration
	OrbitConfigHosts time.Duration
	OrbitConfigScope time.Duration
}

func NewDelayedAgentNotifier(inner fleet.AgentCheckInNotifier, delays AgentNotifierDelays, logger *slog.Logger) *DelayedAgentNotifier {
	return &DelayedAgentNotifier{
		inner:        inner,
		delays:       delays,
		logger:       logger,
		pendingHosts: make(map[string]map[uint]struct{}),
	}
}

var _ fleet.AgentCheckInNotifier = (*DelayedAgentNotifier)(nil)

// NotifyAgentsForLiveQuery schedules the notification to be published after
// the configured delay and returns immediately; publish errors are logged
// rather than returned.
func (n *DelayedAgentNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	n.after(ctx, n.delays.LiveQuery, func(ctx context.Context) error {
		return n.inner.NotifyAgentsForLiveQuery(ctx, hostIDs, campaignID)
	}, "delayed notify agents for live query", "campaign_id", campaignID)
	return nil
}

// NotifyOrbitConfigHosts queues the notification, merging it with the others
// of the same reason until the batching window ends, and returns immediately.
func (n *DelayedAgentNotifier) NotifyOrbitConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	pending, ok := n.pendingHosts[reason]
	if !ok {
		pending = make(map[uint]struct{}, len(hostIDs))
		n.pendingHosts[reason] = pending
		n.after(ctx, n.delays.OrbitConfigHosts, func(ctx context.Context) error {
			n.mu.Lock()
			ids := slices.Sorted(maps.Keys(n.pendingHosts[reason]))
			delete(n.pendingHosts, reason)
			n.mu.Unlock()
			return n.inner.NotifyOrbitConfigHosts(ctx, ids, reason)
		}, "delayed notify orbit config hosts", "reason", reason)
	}
	for _, id := range hostIDs {
		pending[id] = struct{}{}
	}
	return nil
}

// NotifyOrbitConfigScope is NotifyAgentsForLiveQuery's orbit/config
// counterpart for a scope.
func (n *DelayedAgentNotifier) NotifyOrbitConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	n.after(ctx, n.delays.OrbitConfigScope, func(ctx context.Context) error {
		return n.inner.NotifyOrbitConfigScope(ctx, scope, reason)
	}, "delayed notify orbit config scope", "scope", scope, "reason", reason)
	return nil
}

func (n *DelayedAgentNotifier) after(ctx context.Context, delay time.Duration, notify func(context.Context) error, errMsg string, logArgs ...any) {
	// The triggering request's context typically ends right away; detach so
	// the delayed publish is not canceled with it.
	ctx = context.WithoutCancel(ctx)
	time.AfterFunc(delay, func() {
		if err := notify(ctx); err != nil {
			n.logger.ErrorContext(ctx, errMsg, append(logArgs, "err", err)...)
		}
	})
}

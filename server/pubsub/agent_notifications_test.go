package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHostIDOffset/testCampaignIDOffset push the IDs used by these tests far
// beyond any real ID. Redis pub/sub is server-wide (SELECTed databases don't
// isolate channels), so a dev server sharing the local Redis receives these
// test notifications on the agent_notifications channel; unrealistic IDs keep
// it from matching real connected hosts or looking like a real campaign.
const (
	testHostIDOffset     uint = 1 << 40
	testCampaignIDOffset uint = 1 << 40
)

func TestAgentNotificationsRoundTrip(t *testing.T) {
	runTest := func(t *testing.T, cluster bool) {
		pool := redistest.SetupRedis(t, agentNotificationsChannel, cluster, false, false)
		notifier := NewRedisAgentNotifier(pool, slog.New(slog.DiscardHandler))

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var mu sync.Mutex
		var received []AgentNotification
		subscribed := make(chan struct{})
		go func() {
			close(subscribed)
			notifier.Subscribe(ctx, func(n AgentNotification) {
				mu.Lock()
				defer mu.Unlock()
				received = append(received, n)
			}, nil)
		}()
		<-subscribed
		// Give the SUBSCRIBE command time to reach Redis before publishing.
		time.Sleep(200 * time.Millisecond)

		// Host and campaign IDs are offset far beyond any real ID: a dev server
		// sharing this Redis subscribes to the same channel, and realistic IDs
		// would make it notify real connected hosts about a test "campaign".
		require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, []uint{testHostIDOffset + 1, testHostIDOffset + 2, testHostIDOffset + 3}, testCampaignIDOffset+42))
		require.NoError(t, notifier.NotifyOrbitConfigHosts(ctx, []uint{testHostIDOffset + 4}, fleet.AgentWSReasonActivity))
		// A team ID beyond any real one, for the same reason as the host IDs.
		testScope := fleet.AgentNotificationScopeTeam(testHostIDOffset)
		require.NoError(t, notifier.NotifyOrbitConfigScope(ctx, testScope, fleet.AgentWSReasonSettings))

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == 3
		}, 5*time.Second, 50*time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, received[0].Type)
		assert.Equal(t, []uint{testHostIDOffset + 1, testHostIDOffset + 2, testHostIDOffset + 3}, received[0].HostIDs)
		assert.Equal(t, fleet.AgentWSReasonLiveQuery(testCampaignIDOffset+42), received[0].Reason)
		assert.Empty(t, received[0].Scope)

		assert.Equal(t, AgentNotification{
			Type:    fleet.AgentWSMessageTypeOrbitConfig,
			HostIDs: []uint{testHostIDOffset + 4},
			Reason:  fleet.AgentWSReasonActivity,
		}, received[1])
		assert.Equal(t, AgentNotification{
			Type:   fleet.AgentWSMessageTypeOrbitConfig,
			Reason: fleet.AgentWSReasonSettings,
			Scope:  testScope,
		}, received[2])
	}

	t.Run("standalone", func(t *testing.T) { runTest(t, false) })
	t.Run("cluster", func(t *testing.T) { runTest(t, true) })
}

type captureNotifier struct {
	mu       sync.Mutex
	calls    []string
	notified chan struct{}
}

func (c *captureNotifier) record(call string) error {
	c.mu.Lock()
	c.calls = append(c.calls, call)
	c.mu.Unlock()
	c.notified <- struct{}{}
	return nil
}

func (c *captureNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	return c.record(fmt.Sprintf("live %v %d", hostIDs, campaignID))
}

func (c *captureNotifier) NotifyOrbitConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	return c.record(fmt.Sprintf("hosts %v %s", hostIDs, reason))
}

func (c *captureNotifier) NotifyOrbitConfigScope(ctx context.Context, scope fleet.AgentNotificationScope, reason string) error {
	return c.record(fmt.Sprintf("scope %s %s", scope, reason))
}

func TestDelayedAgentNotifier(t *testing.T) {
	const (
		liveDelay  = 100 * time.Millisecond
		scopeDelay = 300 * time.Millisecond
	)

	for _, c := range []struct {
		name   string
		notify func(ctx context.Context, n *DelayedAgentNotifier) error
		delay  time.Duration
		want   string
	}{
		{"live query", func(ctx context.Context, n *DelayedAgentNotifier) error {
			return n.NotifyAgentsForLiveQuery(ctx, []uint{1, 2}, 42)
		}, liveDelay, "live [1 2] 42"},
		{"orbit config hosts", func(ctx context.Context, n *DelayedAgentNotifier) error {
			return n.NotifyOrbitConfigHosts(ctx, []uint{3}, fleet.AgentWSReasonActivity)
		}, 0, "hosts [3] activity"},
		{"orbit config scope", func(ctx context.Context, n *DelayedAgentNotifier) error {
			return n.NotifyOrbitConfigScope(ctx, fleet.AgentNotificationScopeTeam(5), fleet.AgentWSReasonSettings)
		}, scopeDelay, "scope team:5 settings"},
	} {
		t.Run(c.name, func(t *testing.T) {
			inner := &captureNotifier{notified: make(chan struct{}, 1)}
			notifier := NewDelayedAgentNotifier(inner, AgentNotifierDelays{
				LiveQuery:        liveDelay,
				OrbitConfigScope: scopeDelay,
			}, slog.New(slog.DiscardHandler))

			// The call returns immediately; the publish happens after the
			// delay, and survives the caller's context being canceled (the
			// triggering request ends right away).
			ctx, cancel := context.WithCancel(t.Context())
			start := time.Now()
			require.NoError(t, c.notify(ctx, notifier))
			if c.delay > 0 {
				require.Less(t, time.Since(start), c.delay)
			}
			cancel()

			select {
			case <-inner.notified:
			case <-time.After(5 * time.Second):
				t.Fatal("delayed notification never published")
			}
			require.GreaterOrEqual(t, time.Since(start), c.delay)
			inner.mu.Lock()
			defer inner.mu.Unlock()
			assert.Equal(t, []string{c.want}, inner.calls)
		})
	}
}

func TestDelayedAgentNotifierBatchesHosts(t *testing.T) {
	inner := &captureNotifier{notified: make(chan struct{}, 2)}
	notifier := NewDelayedAgentNotifier(inner, AgentNotifierDelays{OrbitConfigHosts: 200 * time.Millisecond},
		slog.New(slog.DiscardHandler))

	require.NoError(t, notifier.NotifyOrbitConfigHosts(t.Context(), []uint{3, 1}, fleet.AgentWSReasonActivity))
	require.NoError(t, notifier.NotifyOrbitConfigHosts(t.Context(), []uint{2, 1}, fleet.AgentWSReasonActivity))
	require.NoError(t, notifier.NotifyOrbitConfigHosts(t.Context(), []uint{4}, fleet.AgentWSReasonMDM))

	for range 2 {
		select {
		case <-inner.notified:
		case <-time.After(5 * time.Second):
			t.Fatal("batched notification never published")
		}
	}
	inner.mu.Lock()
	assert.ElementsMatch(t, []string{"hosts [1 2 3] activity", "hosts [4] mdm"}, inner.calls)
	inner.mu.Unlock()

	// The window is over: the next notification starts a new batch.
	require.NoError(t, notifier.NotifyOrbitConfigHosts(t.Context(), []uint{5}, fleet.AgentWSReasonActivity))
	select {
	case <-inner.notified:
	case <-time.After(5 * time.Second):
		t.Fatal("batched notification never published")
	}
	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, "hosts [5] activity", inner.calls[2])
}

func TestAgentNotificationsResubscribe(t *testing.T) {
	pool := redistest.SetupRedis(t, agentNotificationsChannel, false, false, false)
	notifier := NewRedisAgentNotifier(pool, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var resubscribes atomic.Int32
	var received atomic.Int32
	go notifier.Subscribe(ctx, func(n AgentNotification) { received.Add(1) }, func() { resubscribes.Add(1) })
	time.Sleep(200 * time.Millisecond)
	// The first subscription is not a resubscription.
	require.Zero(t, resubscribes.Load())

	// Kill the subscription connection; Subscribe reconnects after its 1s
	// backoff. This also kills other pub/sub clients of this Redis (e.g. a
	// dev server), which recover the same way.
	conn := pool.Get()
	_, err := conn.Do("CLIENT", "KILL", "TYPE", "pubsub")
	conn.Close()
	require.NoError(t, err)

	require.Eventually(t, func() bool { return resubscribes.Load() == 1 }, 5*time.Second, 50*time.Millisecond)

	// Still delivering after resubscribing.
	require.NoError(t, notifier.NotifyOrbitConfigHosts(ctx, []uint{testHostIDOffset + 1}, fleet.AgentWSReasonActivity))
	require.Eventually(t, func() bool { return received.Load() == 1 }, 5*time.Second, 50*time.Millisecond)
}

func TestAgentNotificationsChunking(t *testing.T) {
	pool := redistest.SetupRedis(t, agentNotificationsChannel, false, false, false)
	notifier := NewRedisAgentNotifier(pool, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var mu sync.Mutex
	var received []AgentNotification
	go notifier.Subscribe(ctx, func(n AgentNotification) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, n)
	}, nil)
	time.Sleep(200 * time.Millisecond)

	// 2.5 chunks worth of host IDs must arrive as 3 messages covering all IDs.
	hostIDs := make([]uint, publishHostIDsChunkSize*2+publishHostIDsChunkSize/2)
	for i := range hostIDs {
		hostIDs[i] = testHostIDOffset + uint(i+1)
	}
	require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, hostIDs, testCampaignIDOffset+7))

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(received) == 3
	}, 5*time.Second, 50*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	var total int
	for _, n := range received {
		total += len(n.HostIDs)
		assert.Equal(t, fleet.AgentWSReasonLiveQuery(testCampaignIDOffset+7), n.Reason)
	}
	assert.Equal(t, len(hostIDs), total)
}

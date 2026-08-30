package pubsub

import (
	"context"
	"log/slog"
	"sync"
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
			})
		}()
		<-subscribed
		// Give the SUBSCRIBE command time to reach Redis before publishing.
		time.Sleep(200 * time.Millisecond)

		// Host and campaign IDs are offset far beyond any real ID: a dev server
		// sharing this Redis subscribes to the same channel, and realistic IDs
		// would make it notify real connected hosts about a test "campaign".
		require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, []uint{testHostIDOffset + 1, testHostIDOffset + 2, testHostIDOffset + 3}, testCampaignIDOffset+42))

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == 1
		}, 5*time.Second, 50*time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, received[0].Type)
		assert.Equal(t, []uint{testHostIDOffset + 1, testHostIDOffset + 2, testHostIDOffset + 3}, received[0].HostIDs)
		assert.Equal(t, fleet.AgentWSReasonLiveQuery(testCampaignIDOffset+42), received[0].Reason)
	}

	t.Run("standalone", func(t *testing.T) { runTest(t, false) })
	t.Run("cluster", func(t *testing.T) { runTest(t, true) })
}

type captureNotifier struct {
	mu         sync.Mutex
	hostIDs    []uint
	campaignID uint
	notified   chan struct{}
}

func (c *captureNotifier) NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error {
	c.mu.Lock()
	c.hostIDs = hostIDs
	c.campaignID = campaignID
	c.mu.Unlock()
	close(c.notified)
	return nil
}

func TestDelayedAgentNotifier(t *testing.T) {
	inner := &captureNotifier{notified: make(chan struct{})}
	const delay = 100 * time.Millisecond
	notifier := NewDelayedAgentNotifier(inner, delay, slog.New(slog.DiscardHandler))

	// The call returns immediately; the publish happens after the delay, and
	// survives the caller's context being canceled (the campaign-creation
	// request ends right away).
	ctx, cancel := context.WithCancel(t.Context())
	start := time.Now()
	require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, []uint{1, 2}, 42))
	require.Less(t, time.Since(start), delay)
	cancel()

	select {
	case <-inner.notified:
	case <-time.After(5 * time.Second):
		t.Fatal("delayed notification never published")
	}
	require.GreaterOrEqual(t, time.Since(start), delay)
	inner.mu.Lock()
	defer inner.mu.Unlock()
	assert.Equal(t, []uint{1, 2}, inner.hostIDs)
	assert.Equal(t, uint(42), inner.campaignID)
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
	})
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

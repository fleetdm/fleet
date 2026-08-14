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

		require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, []uint{1, 2, 3}, 42))

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(received) == 1
		}, 5*time.Second, 50*time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, received[0].Type)
		assert.Equal(t, []uint{1, 2, 3}, received[0].HostIDs)
		assert.Equal(t, uint(42), received[0].CampaignID)
	}

	t.Run("standalone", func(t *testing.T) { runTest(t, false) })
	t.Run("cluster", func(t *testing.T) { runTest(t, true) })
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
		hostIDs[i] = uint(i + 1)
	}
	require.NoError(t, notifier.NotifyAgentsForLiveQuery(ctx, hostIDs, 7))

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
		assert.Equal(t, uint(7), n.CampaignID)
	}
	assert.Equal(t, len(hostIDs), total)
}

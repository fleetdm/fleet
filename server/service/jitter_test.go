package service

import (
	crand "crypto/rand"
	"log/slog"
	"math"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

func TestJitterForHost(t *testing.T) {
	jh := newJitterHashTable(30)

	histogram := make(map[int64]int)
	hostCount := 3000
	for i := 0; i < hostCount; i++ {
		hostID, err := crand.Int(crand.Reader, big.NewInt(10000))
		require.NoError(t, err)
		jitter := jh.jitterForHost(uint(hostID.Int64() + 10000)) //nolint:gosec // dismiss G115
		jitterMinutes := int64(jitter.Minutes())
		histogram[jitterMinutes]++
	}
	minVal, maxVal := math.MaxInt, 0
	for jitterMinutes, count := range histogram {
		if count < minVal {
			minVal = count
		}
		if count > maxVal {
			maxVal = count
		}
		t.Logf("jitterMinutes=%d \t count=%d\n", jitterMinutes, count)
	}
	variation := maxVal - minVal
	t.Logf("min=%d \t max=%d \t variation=%d\n", minVal, maxVal, variation)

	// check that variation is below 1% of the total amount of hosts
	require.Less(t, variation, int(float32(hostCount)/0.01))
}

func TestNoJitter(t *testing.T) {
	jh := newJitterHashTable(0)

	hostCount := 3000
	for i := 0; i < hostCount; i++ {
		hostID, err := crand.Int(crand.Reader, big.NewInt(10000))
		require.NoError(t, err)
		jitter := jh.jitterForHost(uint(hostID.Int64() + 10000)) //nolint:gosec // dismiss G115
		jitterMinutes := int64(jitter.Minutes())
		require.Equal(t, int64(0), jitterMinutes)
	}
}

func TestShouldUpdateOverdueQuerySplay(t *testing.T) {
	const interval = 1 * time.Hour
	const window = 10 * time.Minute

	newSvc := func(window time.Duration) (*Service, *clock.MockClock) {
		mockClock := clock.NewMockClock()
		cfg := config.TestConfig()
		cfg.Osquery.OverdueQuerySplayWindow = window
		return &Service{
			clock:     mockClock,
			config:    cfg,
			logger:    slog.New(slog.DiscardHandler),
			jitterH:   make(map[time.Duration]*jitterHashTable),
			jitterMu:  new(sync.RWMutex),
			startedAt: mockClock.Now(),
		}, mockClock
	}

	t.Run("host not due is never sent queries", func(t *testing.T) {
		svc, mockClock := newSvc(window)
		require.False(t, svc.shouldUpdate(mockClock.Now().Add(-30*time.Minute), interval, 1))
	})

	t.Run("host due after server start is sent queries immediately", func(t *testing.T) {
		svc, mockClock := newSvc(window)
		mockClock.AddTime(2 * time.Hour)
		// Due 1 minute ago, well after server start.
		require.True(t, svc.shouldUpdate(mockClock.Now().Add(-interval-time.Minute), interval, 1))
	})

	t.Run("host overdue from before server start is splayed", func(t *testing.T) {
		svc, mockClock := newSvc(window)
		lastUpdated := mockClock.Now().Add(-2 * interval) // due 1h before server start

		// Host 601 has a splay of 1s (601 % 600): not admitted at start, admitted after.
		require.False(t, svc.shouldUpdate(lastUpdated, interval, 601))
		mockClock.AddTime(2 * time.Second)
		require.True(t, svc.shouldUpdate(lastUpdated, interval, 601))

		// Host with maximum splay (599s) is admitted only near the end of the window.
		require.False(t, svc.shouldUpdate(lastUpdated, interval, 599))
		mockClock.AddTime(10 * time.Minute)
		require.True(t, svc.shouldUpdate(lastUpdated, interval, 599))
	})

	t.Run("overdue hosts drain uniformly over the window", func(t *testing.T) {
		svc, mockClock := newSvc(window)
		lastUpdated := mockClock.Now().Add(-2 * interval)
		mockClock.AddTime(window / 2) // halfway through the window

		admitted := 0
		for hostID := uint(1); hostID <= 600; hostID++ {
			if svc.shouldUpdate(lastUpdated, interval, hostID) {
				admitted++
			}
		}
		require.Equal(t, 300, admitted)
	})

	t.Run("zero last updated is sent queries immediately", func(t *testing.T) {
		svc, _ := newSvc(window)
		require.True(t, svc.shouldUpdate(time.Time{}, interval, 1))
	})

	t.Run("disabled window keeps legacy behavior", func(t *testing.T) {
		svc, mockClock := newSvc(0)
		require.True(t, svc.shouldUpdate(mockClock.Now().Add(-2*interval), interval, 599))
	})

	t.Run("window is capped at the interval", func(t *testing.T) {
		svc, mockClock := newSvc(3 * time.Hour)
		lastUpdated := mockClock.Now().Add(-2 * interval)
		// Splay for host 3599 with a capped 1h window is 3599s; with the
		// uncapped 3h window it would be 3599s too, so use a host whose
		// offsets differ: host 7199 -> 3599s capped, 7199s uncapped.
		mockClock.AddTime(interval + time.Second)
		require.True(t, svc.shouldUpdate(lastUpdated, interval, 7199))
	})
}

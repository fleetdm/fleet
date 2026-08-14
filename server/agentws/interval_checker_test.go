package agentws

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDueLister struct {
	mu      sync.Mutex
	batches [][]uint
	due     map[uint]bool
	err     error
}

func (f *fakeDueLister) ListHostIDsDueForDistributedRead(ctx context.Context, hostIDs []uint) ([]uint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batches = append(f.batches, slices.Clone(hostIDs))
	if f.err != nil {
		return nil, f.err
	}
	var due []uint
	for _, id := range hostIDs {
		if f.due[id] {
			due = append(due, id)
		}
	}
	return due, nil
}

func (f *fakeDueLister) batchSizes() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	sizes := make([]int, len(f.batches))
	for i, b := range f.batches {
		sizes[i] = len(b)
	}
	return sizes
}

func TestIntervalCheckerNotifiesDueHosts(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws1 := dial(t, srv, "key-1")
	ws2 := dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	lister := &fakeDueLister{due: map[uint]bool{1: true}}
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       lister,
		Interval:  time.Minute,
		Grace:     5 * time.Minute,
		BatchSize: 10,
		Logger:    discardLogger(),
	}
	checker.checkOnce(t.Context())

	// Host 1 is due and gets the notification.
	require.NoError(t, ws1.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, ws1.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, msg.Type)

	// Host 2 is not due: nothing arrives.
	require.NoError(t, ws2.SetReadDeadline(time.Now().Add(300*time.Millisecond)))
	_, _, err := ws2.ReadMessage()
	require.Error(t, err)

	// Within the grace period the due host is not re-notified.
	lister.mu.Lock()
	lister.batches = nil
	lister.mu.Unlock()
	checker.checkOnce(t.Context())
	lister.mu.Lock()
	for _, batch := range lister.batches {
		assert.NotContains(t, batch, uint(1))
	}
	lister.mu.Unlock()
}

func TestIntervalCheckerBatching(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	dial(t, srv, "key-1")
	dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	lister := &fakeDueLister{}
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       lister,
		Interval:  time.Minute,
		Grace:     5 * time.Minute,
		BatchSize: 1,
		Logger:    discardLogger(),
	}
	checker.checkOnce(t.Context())

	// Two connected hosts with batch size 1 → two due-check batches.
	assert.Equal(t, []int{1, 1}, lister.batchSizes())
}

func TestIntervalCheckerListError(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	dial(t, srv, "key-1")
	waitForConnCount(t, hub, 1)

	lister := &fakeDueLister{err: context.DeadlineExceeded}
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       lister,
		Interval:  time.Minute,
		Grace:     5 * time.Minute,
		BatchSize: 10,
		Logger:    discardLogger(),
	}
	// Must not panic or notify anyone.
	checker.checkOnce(t.Context())
}

func TestIntervalCheckerRunStopsOnContextDone(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       &fakeDueLister{},
		Interval:  10 * time.Millisecond,
		Grace:     time.Minute,
		BatchSize: 10,
		Logger:    discardLogger(),
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		checker.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop on context cancellation")
	}
}

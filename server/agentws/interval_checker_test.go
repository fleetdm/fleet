package agentws

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDueLister struct {
	mu      sync.Mutex
	batches [][]uint
	due     map[uint]string // host ID → reason
	err     error
}

func (f *fakeDueLister) ListHostIDsDueForDistributedRead(ctx context.Context, hostIDs []uint) (map[uint]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batches = append(f.batches, slices.Clone(hostIDs))
	if f.err != nil {
		return nil, f.err
	}
	due := make(map[uint]string)
	for _, id := range hostIDs {
		if reason, ok := f.due[id]; ok {
			due[id] = reason
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

	lister := &fakeDueLister{due: map[uint]string{1: fleet.AgentWSReasonLabel}}
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       lister,
		Interval:  time.Minute,
		BatchSize: 10,
		Logger:    discardLogger(),
	}
	checker.checkOnce(t.Context())

	// Host 1 is due and gets the notification, carrying the due reason.
	require.NoError(t, ws1.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, ws1.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSMessageTypeDistributedRead, msg.Type)
	assert.Equal(t, fleet.AgentWSReasonLabel, msg.Reason)

	// Host 2 is not due: nothing arrives.
	require.NoError(t, ws2.SetReadDeadline(time.Now().Add(300*time.Millisecond)))
	_, _, err := ws2.ReadMessage()
	require.Error(t, err)

	// While the host stays due in the database it is re-notified on every
	// tick: the database is the source of truth, and the agent coalesces
	// repeated triggers into one read+write iteration at a time.
	checker.checkOnce(t.Context())
	require.NoError(t, ws1.SetReadDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, ws1.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSReasonLabel, msg.Reason)

	// Once ingestion clears the due state, notifications stop.
	lister.mu.Lock()
	lister.due = nil
	lister.mu.Unlock()
	checker.checkOnce(t.Context())
	require.NoError(t, ws1.SetReadDeadline(time.Now().Add(300*time.Millisecond)))
	_, _, err = ws1.ReadMessage()
	require.Error(t, err)
}

func TestIntervalCheckerDisconnectsDeletedHosts(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	ws1 := dial(t, srv, "key-1")
	ws2 := dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	// Host 1 was deleted while its agent held a connection; host 2 is due.
	lister := &fakeDueLister{due: map[uint]string{
		1: fleet.AgentWSReasonHostNotFound,
		2: fleet.AgentWSReasonLabel,
	}}
	checker := &IntervalChecker{
		Hub:       hub,
		Svc:       lister,
		Interval:  time.Minute,
		BatchSize: 10,
		Logger:    discardLogger(),
	}
	checker.checkOnce(t.Context())

	// The deleted host's connection is closed (the agent reconnects under its
	// new identity) and dropped from the hub, never notified.
	require.NoError(t, ws1.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err := ws1.ReadMessage()
	require.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseAbnormalClosure) || errors.Is(err, io.ErrUnexpectedEOF), "unexpected error: %v", err)
	waitForConnCount(t, hub, 1)
	assert.Equal(t, []uint{2}, hub.HeldHostIDs())

	// The due host is notified as usual.
	require.NoError(t, ws2.SetReadDeadline(time.Now().Add(2*time.Second)))
	var msg fleet.AgentWSMessage
	require.NoError(t, ws2.ReadJSON(&msg))
	assert.Equal(t, fleet.AgentWSReasonLabel, msg.Reason)
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

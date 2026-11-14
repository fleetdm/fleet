package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/hostidentity"
	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/osquery_perf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWSTransportCoalescesTriggers verifies the simulator mirrors orbit's
// one-iteration-at-a-time coalescing: triggers arriving while a distributed
// read+write cycle is in flight collapse into exactly one queued follow-up.
func TestWSTransportCoalescesTriggers(t *testing.T) {
	var reads atomic.Int64
	release := make(chan struct{}, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/osquery/distributed/read" {
			reads.Add(1)
			<-release // hold the read until the test releases it
			_, _ = w.Write([]byte(`{"queries":{}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	a := &agent{
		serverAddress:      srv.URL,
		nodeKey:            "test-node-key",
		stats:              &osquery_perf.Stats{},
		hostIdentityClient: &hostidentity.Client{},
		onlineFraction:     1,
	}
	ws := newWSTransport(a)
	// Install the transport as syncWSTransport does: an active transport is
	// what routes the reads to the unversioned /api/osquery path.
	a.ws = ws

	ws.trigger()
	require.Eventually(t, func() bool { return reads.Load() == 1 }, 2*time.Second, time.Millisecond)

	// Triggers during the in-flight cycle coalesce into one pending bit.
	ws.trigger()
	ws.trigger()
	ws.mu.Lock()
	busy, pending := ws.busy, ws.pending
	ws.mu.Unlock()
	assert.True(t, busy)
	assert.True(t, pending)
	assert.Equal(t, int64(1), reads.Load())

	// Finishing the cycle fires exactly one queued follow-up cycle.
	release <- struct{}{}
	require.Eventually(t, func() bool { return reads.Load() == 2 }, 2*time.Second, time.Millisecond)
	release <- struct{}{}
	require.Eventually(t, func() bool {
		ws.mu.Lock()
		defer ws.mu.Unlock()
		return !ws.busy && !ws.pending
	}, 2*time.Second, time.Millisecond)
	assert.Equal(t, int64(2), reads.Load())
}

// TestWSTransportOfflineHost verifies an offline host neither connects nor checks in over the
// WebSocket transport, so Fleet stops seeing it like a powered-off machine.
func TestWSTransportOfflineHost(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	a := &agent{
		serverAddress:      srv.URL,
		nodeKey:            "test-node-key",
		stats:              &osquery_perf.Stats{},
		hostIdentityClient: &hostidentity.Client{},
		stale:              true,
	}
	ws := newWSTransport(a)
	a.ws = ws
	done := make(chan struct{})
	go func() {
		ws.run()
		close(done)
	}()

	ws.trigger()
	time.Sleep(200 * time.Millisecond)
	assert.Zero(t, requests.Load(), "an offline host dialed or read")

	ws.stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the transport didn't stop while waiting to come online")
	}
}

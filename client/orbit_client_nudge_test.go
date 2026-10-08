package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configServer serves the orbit config, counting requests and recording the
// fallback changes header.
type configServer struct {
	mu      sync.Mutex
	cfg     fleet.OrbitConfig
	headers []string
	count   atomic.Int32
}

func (s *configServer) set(cfg fleet.OrbitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *configServer) fallbackHeaders() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headers
}

func newNudgeTestClient(t *testing.T, srv *configServer) *OrbitClient {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.count.Add(1)
		srv.mu.Lock()
		if h := r.Header.Get(fleet.OrbitConfigFallbackChangesHeader); h != "" {
			srv.headers = append(srv.headers, h)
		}
		cfg := srv.cfg
		srv.mu.Unlock()
		_ = json.NewEncoder(w).Encode(fleet.OrbitGetConfigResponse{OrbitConfig: cfg})
	}))
	t.Cleanup(ts.Close)

	oc, err := NewOrbitClient(t.TempDir(), ts.URL, "", true, "secret", nil, fleet.OrbitHostInfo{}, nil, nil, "", false)
	require.NoError(t, err)
	oc.TestNodeKey = "node-key"
	return oc
}

func runConfigReceivers(t *testing.T, oc *OrbitClient) {
	done := make(chan error, 1)
	go func() { done <- oc.ExecuteConfigReceivers() }()
	t.Cleanup(func() {
		oc.receiverUpdateCancelFunc()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("ExecuteConfigReceivers did not stop")
		}
	})
}

func nudgedConfig(pollInterval int) fleet.OrbitConfig {
	return fleet.OrbitConfig{WebSocketTransport: &fleet.OrbitWebSocketTransportConfig{
		Enabled:                 true,
		OrbitConfigPollInterval: &pollInterval,
	}}
}

func TestConfigRefreshBypassesCache(t *testing.T) {
	srv := &configServer{}
	oc := newNudgeTestClient(t, srv)
	oc.ReceiverUpdateInterval = time.Hour

	var runs atomic.Int32
	oc.RegisterConfigReceiver(fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		runs.Add(1)
		return nil
	}))

	// Cache a config, as the startup fetch does.
	_, err := oc.GetConfig()
	require.NoError(t, err)
	require.EqualValues(t, 1, srv.count.Load())

	runConfigReceivers(t, oc)
	// Within the cache TTL, a nudge still fetches.
	oc.TriggerConfigRefresh()
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 2, srv.count.Load())
}

func TestConfigRefreshCoalescesDuringRun(t *testing.T) {
	srv := &configServer{}
	oc := newNudgeTestClient(t, srv)
	oc.ReceiverUpdateInterval = time.Hour

	release := make(chan struct{})
	var runs atomic.Int32
	oc.RegisterConfigReceiver(fleet.OrbitConfigReceiverFunc(func(cfg *fleet.OrbitConfig) error {
		if runs.Add(1) == 1 {
			<-release
		}
		return nil
	}))

	runConfigReceivers(t, oc)
	oc.TriggerConfigRefresh()
	require.Eventually(t, func() bool { return runs.Load() == 1 }, 5*time.Second, 10*time.Millisecond)

	// Nudges during the run are merged into exactly one follow-up run.
	for range 5 {
		oc.TriggerConfigRefresh()
	}
	close(release)
	require.Eventually(t, func() bool { return runs.Load() == 2 }, 5*time.Second, 10*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	require.EqualValues(t, 2, runs.Load())
	require.EqualValues(t, 2, srv.count.Load())
}

func TestNextConfigInterval(t *testing.T) {
	oc := clientWithConfig(nil)
	oc.ReceiverUpdateInterval = 30 * time.Second
	connected := true
	oc.SetTransportConnectedFunc(func() bool { return connected })

	quiet := nudgedConfig(300)
	assert.Equal(t, 5*time.Minute, oc.nextConfigInterval(&quiet))

	// No hint: the server doesn't nudge.
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(&fleet.OrbitConfig{}))
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(nil))

	// Pending work: receivers use the poll as their clock.
	busy := nudgedConfig(300)
	busy.Notifications.PendingScriptExecutionIDs = []string{"a"}
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(&busy))

	// A receiver with local work to retry.
	poller := &frequentPoller{}
	oc.RegisterConfigReceiver(poller)
	poller.needs = true
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(&quiet))
	poller.needs = false
	assert.Equal(t, 5*time.Minute, oc.nextConfigInterval(&quiet))

	// Not connected: changes aren't nudged.
	connected = false
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(&quiet))

	// The hint never makes polling faster than the default.
	connected = true
	fast := nudgedConfig(5)
	assert.Equal(t, 30*time.Second, oc.nextConfigInterval(&fast))
}

type frequentPoller struct{ needs bool }

func (p *frequentPoller) Run(*fleet.OrbitConfig) error { return nil }
func (p *frequentPoller) NeedsFrequentPolling() bool   { return p.needs }

func TestConfigPollSlowsDownAndReportsFallbackChanges(t *testing.T) {
	origTTL := configCacheTTL
	configCacheTTL = 0
	t.Cleanup(func() { configCacheTTL = origTTL })

	srv := &configServer{}
	// The hint is in seconds, so test with a 1s fallback against a 100ms default.
	srv.set(nudgedConfig(1))
	oc := newNudgeTestClient(t, srv)
	oc.ReceiverUpdateInterval = 100 * time.Millisecond
	var connected atomic.Bool
	connected.Store(true)
	oc.SetTransportConnectedFunc(connected.Load)

	runConfigReceivers(t, oc)
	// Slow polling: ~1 fetch per second rather than 10.
	time.Sleep(2500 * time.Millisecond)
	require.LessOrEqual(t, srv.count.Load(), int32(4))

	// A change the fallback poll finds is reported on the next request.
	changed := nudgedConfig(1)
	changed.ScriptExeTimeout = 42
	changed.Notifications.RunDiskEncryptionEscrow = true
	srv.set(changed)
	require.Eventually(t, func() bool { return len(srv.fallbackHeaders()) > 0 }, 5*time.Second, 50*time.Millisecond)
	assert.Equal(t, "notifications.run_disk_encryption_escrow,script_execution_timeout", srv.fallbackHeaders()[0])

	// The escrow request isn't quiet: back to the default interval.
	before := srv.count.Load()
	time.Sleep(500 * time.Millisecond)
	require.GreaterOrEqual(t, srv.count.Load()-before, int32(3))

	// Quiet again, then disconnected: the slow timer is cut short.
	srv.set(nudgedConfig(1))
	time.Sleep(300 * time.Millisecond)
	connected.Store(false)
	oc.TransportDisconnected()
	before = srv.count.Load()
	time.Sleep(500 * time.Millisecond)
	require.GreaterOrEqual(t, srv.count.Load()-before, int32(3))
}

func TestTransportConnectedFetchGating(t *testing.T) {
	origTTL := configCacheTTL
	configCacheTTL = 0
	t.Cleanup(func() { configCacheTTL = origTTL })

	srv := &configServer{}
	oc := newNudgeTestClient(t, srv)
	oc.ReceiverUpdateInterval = 300 * time.Millisecond
	oc.SetTransportConnectedFunc(func() bool { return true })
	runConfigReceivers(t, oc)

	// No hint from the server: connecting fetches nothing.
	require.Eventually(t, func() bool { return srv.count.Load() >= 1 }, 5*time.Second, 10*time.Millisecond)
	srv.set(nudgedConfig(3600))
	require.Eventually(t, func() bool { return srv.count.Load() >= 2 }, 5*time.Second, 10*time.Millisecond)
	// Now on the slow interval; a connect right after a fetch is skipped.
	before := srv.count.Load()
	oc.TransportConnected()
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, before, srv.count.Load())

	// Once the last fetch is a default interval old, a connect fetches.
	time.Sleep(400 * time.Millisecond)
	oc.TransportConnected()
	require.Eventually(t, func() bool { return srv.count.Load() == before+1 }, 5*time.Second, 10*time.Millisecond)
}

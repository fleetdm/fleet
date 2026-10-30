package wstransport

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/gen/osquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configCall struct {
	etag           string
	fallbackChange bool
}

// fakeConfigClient answers each fetch with the next response; the last one
// repeats.
type fakeConfigClient struct {
	mu        sync.Mutex
	responses []string
	errs      []error
	calls     []configCall
	// block, when set, is received from before answering.
	block chan struct{}
}

func (c *fakeConfigClient) OsqueryConfig(etag string, fallbackChange bool) (json.RawMessage, error) {
	if c.block != nil {
		<-c.block
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	i := len(c.calls)
	c.calls = append(c.calls, configCall{etag: etag, fallbackChange: fallbackChange})
	if i < len(c.errs) && c.errs[i] != nil {
		return nil, c.errs[i]
	}
	return json.RawMessage(c.responses[min(i, len(c.responses)-1)]), nil
}

func (c *fakeConfigClient) getCalls() []configCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]configCall(nil), c.calls...)
}

type fetcherTest struct {
	f         *ConfigFetcher
	client    *fakeConfigClient
	connected atomic.Bool
	now       time.Time
	nowMu     sync.Mutex
	fetched   chan struct{}
}

func newFetcherTest(t *testing.T, client *fakeConfigClient) *fetcherTest {
	ft := &fetcherTest{client: client, now: time.Unix(1_000_000, 0), fetched: make(chan struct{}, 10)}
	ft.f = NewConfigFetcher(client, ft.connected.Load, DefaultServerConfigRefresh)
	ft.f.now = func() time.Time {
		ft.nowMu.Lock()
		defer ft.nowMu.Unlock()
		return ft.now
	}
	ft.f.fetched = func() { ft.fetched <- struct{}{} }
	return ft
}

func (ft *fetcherTest) advance(d time.Duration) {
	ft.nowMu.Lock()
	defer ft.nowMu.Unlock()
	ft.now = ft.now.Add(d)
}

func (ft *fetcherTest) waitFetch(t *testing.T) {
	t.Helper()
	select {
	case <-ft.fetched:
	case <-time.After(5 * time.Second):
		t.Fatal("no fetch")
	}
}

func (ft *fetcherTest) noFetch(t *testing.T) {
	t.Helper()
	select {
	case <-ft.fetched:
		t.Fatal("unexpected fetch")
	case <-time.After(50 * time.Millisecond):
	}
}

func (ft *fetcherTest) genConfig(t *testing.T) map[string]any {
	t.Helper()
	cfg, err := ft.f.GenConfig()
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(cfg), &m))
	return m
}

func TestConfigFetcherFirstFetchPreparesConfig(t *testing.T) {
	client := &fakeConfigClient{responses: []string{
		`{"etag":"v1","options":{"pack_delimiter":"/"},"packs":{"Global":{}}}`,
	}}
	ft := newFetcherTest(t, client)
	ft.f.serverRefresh = 300 * time.Second

	plugin := NewConfigPlugin(ft.f)
	resp := plugin.Call(context.Background(), osquery.ExtensionPluginRequest{"action": "genConfig"})
	require.EqualValues(t, 0, resp.Status.Code, resp.Status.Message)
	require.Len(t, resp.Response, 1)
	require.Contains(t, resp.Response[0], configSourceName)
	ft.waitFetch(t)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(resp.Response[0][configSourceName]), &cfg))
	assert.NotContains(t, cfg, "etag")
	assert.Equal(t, map[string]any{"pack_delimiter": "/"}, cfg["options"])
	assert.Equal(t, map[string]any{"Global": map[string]any{}}, cfg["packs"])
	assert.Equal(t, []configCall{{etag: ""}}, client.getCalls())

	// While disconnected, the server is polled at osquery's former config
	// refresh, not at its local one.
	ft.advance(299 * time.Second)
	ft.genConfig(t)
	ft.noFetch(t)
	ft.advance(time.Second)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Equal(t, configCall{etag: "v1"}, client.getCalls()[1])
}

func TestConfigFetcherFirstFetchError(t *testing.T) {
	client := &fakeConfigClient{
		responses: []string{`{"options":{}}`},
		errs:      []error{errors.New("boom")},
	}
	ft := newFetcherTest(t, client)

	_, err := ft.f.GenConfig()
	require.Error(t, err)
	ft.waitFetch(t)

	// Retried at the disconnected cadence, however often osquery asks.
	_, err = ft.f.GenConfig()
	require.Error(t, err)
	ft.noFetch(t)

	ft.advance(DefaultServerConfigRefresh)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Len(t, client.getCalls(), 2)
}

func TestConfigFetcherConcurrentFirstFetches(t *testing.T) {
	client := &fakeConfigClient{
		responses: []string{`{"etag":"v1","options":{}}`},
		block:     make(chan struct{}),
	}
	ft := newFetcherTest(t, client)

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			_, _ = ft.f.GenConfig()
		})
	}
	client.block <- struct{}{}
	wg.Wait()
	ft.waitFetch(t)
	ft.noFetch(t)
	assert.Len(t, client.getCalls(), 1)
}

func TestConfigFetcherShorterPollInterval(t *testing.T) {
	client := &fakeConfigClient{responses: []string{`{"etag":"v1","options":{}}`, `{"etag":"ok"}`}}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)

	ft.genConfig(t)
	ft.waitFetch(t)

	// The server's interval arrives after the first fetch scheduled the
	// default one.
	ft.f.SetPollInterval(2 * time.Minute)
	ft.advance(2 * time.Minute)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Len(t, client.getCalls(), 2)

	// A longer one doesn't postpone the scheduled poll.
	ft.f.SetPollInterval(5 * time.Minute)
	ft.advance(2 * time.Minute)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Len(t, client.getCalls(), 3)
}

func TestConfigFetcherConnectedPollsSlowly(t *testing.T) {
	client := &fakeConfigClient{responses: []string{
		`{"etag":"v1","options":{}}`,
		`{"etag":"ok"}`,
		`{"etag":"v2","options":{"distributed_interval":30}}`,
		`{"etag":"ok"}`,
	}}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)
	ft.f.SetPollInterval(5 * time.Minute)

	first := ft.genConfig(t)
	ft.waitFetch(t)

	// Connected: no fetch at the disconnected cadence.
	ft.advance(time.Minute)
	ft.genConfig(t)
	ft.noFetch(t)

	// The fallback poll finds nothing changed.
	ft.advance(4 * time.Minute)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Equal(t, first, ft.genConfig(t))

	// The next one finds a change, reported on the following fetch.
	ft.advance(5 * time.Minute)
	ft.genConfig(t)
	ft.waitFetch(t)
	cfg := ft.genConfig(t)
	assert.Equal(t, map[string]any{"distributed_interval": float64(30)}, cfg["options"])

	ft.advance(5 * time.Minute)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Equal(t, []configCall{
		{etag: ""},
		{etag: "v1"},
		{etag: "v1"},
		{etag: "v2", fallbackChange: true},
	}, client.getCalls())
}

func TestConfigFetcherNudge(t *testing.T) {
	client := &fakeConfigClient{responses: []string{
		`{"etag":"v1","options":{}}`,
		`{"etag":"v2","options":{"logger_tls_period":20}}`,
	}}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)

	// A nudge before the first fetch is moot.
	ft.f.Nudge()
	ft.noFetch(t)

	ft.genConfig(t)
	ft.waitFetch(t)

	ft.f.Nudge()
	ft.waitFetch(t)
	cfg := ft.genConfig(t)
	assert.Equal(t, map[string]any{"logger_tls_period": float64(20)}, cfg["options"])
	// A nudged change is not a fallback change.
	ft.f.Nudge()
	ft.waitFetch(t)
	assert.Equal(t, configCall{etag: "v2"}, client.getCalls()[2])
}

func TestConfigFetcherNudgesCoalesce(t *testing.T) {
	client := &fakeConfigClient{responses: []string{`{"options":{}}`}}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)
	ft.genConfig(t)
	ft.waitFetch(t)

	client.block = make(chan struct{})
	ft.f.Nudge()
	// Nudges during the fetch queue a single follow-up.
	ft.f.Nudge()
	ft.f.Nudge()
	client.block <- struct{}{}
	ft.waitFetch(t)
	client.block <- struct{}{}
	ft.waitFetch(t)
	ft.noFetch(t)
	assert.Len(t, client.getCalls(), 3)
}

func TestConfigFetcherDisconnected(t *testing.T) {
	client := &fakeConfigClient{responses: []string{`{"options":{}}`}}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)
	ft.genConfig(t)
	ft.waitFetch(t)

	ft.connected.Store(false)
	ft.f.Disconnected()
	ft.advance(disconnectSpread)
	ft.genConfig(t)
	ft.waitFetch(t)
	// Not counted as a fallback poll.
	ft.advance(DefaultServerConfigRefresh)
	ft.genConfig(t)
	ft.waitFetch(t)
	for _, c := range client.getCalls() {
		assert.False(t, c.fallbackChange)
	}
}

func TestConfigFetcherErrorKeepsConfig(t *testing.T) {
	client := &fakeConfigClient{
		responses: []string{`{"options":{"pack_delimiter":"/"}}`},
		errs:      []error{nil, errors.New("boom")},
	}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)
	cfg := ft.genConfig(t)
	ft.waitFetch(t)

	ft.f.Nudge()
	ft.waitFetch(t)
	assert.Equal(t, cfg, ft.genConfig(t))

	// The failed fetch is retried at the disconnected cadence.
	ft.advance(DefaultServerConfigRefresh)
	ft.genConfig(t)
	ft.waitFetch(t)
	assert.Len(t, client.getCalls(), 3)
}

func TestPrepareConfig(t *testing.T) {
	for _, tc := range []struct {
		name      string
		raw       string
		want      string
		etag      string
		unchanged bool
		wantErr   bool
	}{
		{name: "no etag", raw: `{"options":{"pack_delimiter":"/"}}`, want: `{"options":{"pack_delimiter":"/"}}`},
		{name: "etag", raw: `{"etag":"abc","decorators":{}}`, want: `{"decorators":{}}`, etag: "abc"},
		{name: "unchanged", raw: `{"etag":"ok"}`, unchanged: true},
		{name: "unchanged, reformatted", raw: `{ "etag" : "ok" }`, unchanged: true},
		{name: "unchanged with a config", raw: `{"etag":"ok","options":{}}`, wantErr: true},
		{name: "null", raw: `null`, wantErr: true},
		{name: "not an object", raw: `[]`, wantErr: true},
		{name: "bad etag", raw: `{"etag":1}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, etag, unchanged, err := prepareConfig(json.RawMessage(tc.raw))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.unchanged, unchanged)
			if tc.unchanged {
				return
			}
			assert.JSONEq(t, tc.want, cfg)
			assert.Equal(t, tc.etag, etag)
		})
	}
}

func TestConfigFetcherNudgeDuringFirstFetch(t *testing.T) {
	client := &fakeConfigClient{
		responses: []string{`{"options":{}}`, `{"options":{"pack_delimiter":"_"}}`},
		block:     make(chan struct{}),
	}
	ft := newFetcherTest(t, client)
	ft.connected.Store(true)

	done := make(chan error, 1)
	go func() {
		_, err := ft.f.GenConfig()
		done <- err
	}()
	require.Eventually(t, func() bool {
		ft.f.mu.Lock()
		defer ft.f.mu.Unlock()
		return ft.f.fetching
	}, 5*time.Second, time.Millisecond)
	ft.f.Nudge()
	client.block <- struct{}{}
	ft.waitFetch(t)
	require.NoError(t, <-done)

	// The nudge fetches again once the first fetch is done.
	client.block <- struct{}{}
	ft.waitFetch(t)
	assert.Equal(t, map[string]any{"pack_delimiter": "_"}, ft.genConfig(t)["options"])
}

func TestConfigFetcherDueFetchDoesNotQueue(t *testing.T) {
	client := &fakeConfigClient{responses: []string{`{"options":{}}`}}
	ft := newFetcherTest(t, client)
	ft.genConfig(t)
	ft.waitFetch(t)

	client.block = make(chan struct{})
	ft.advance(DefaultServerConfigRefresh)
	ft.genConfig(t)
	// osquery calls again while the due fetch is in flight.
	ft.genConfig(t)
	ft.genConfig(t)
	client.block <- struct{}{}
	ft.waitFetch(t)
	ft.noFetch(t)
	assert.Len(t, client.getCalls(), 2)
}

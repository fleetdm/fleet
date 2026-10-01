package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/hostidentity"
	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/osquery_perf"
	"github.com/stretchr/testify/require"
)

func newTestAgent(configTLSETag bool) *agent {
	stats := &osquery_perf.Stats{}
	a := &agent{
		agentIndex:         1,
		serverAddress:      "http://test",
		stats:              stats,
		nodeKey:            "test-node-key",
		configTLSETag:      configTLSETag,
		scheduledQueryData: new(sync.Map),
		hostIdentityClient: hostidentity.NewClient(hostidentity.Config{}, false, 0),
	}
	return a
}

// unchangedBody is the constant server response for a matching etag.
const unchangedBody = `{"etag":"ok"}`

// requestETag extracts the body-carried etag from a config request. nil
// means the agent did not opt in.
func requestETag(r *http.Request) *string {
	var req struct {
		ETag *string `json:"etag"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)
	return req.ETag
}

// withETagKey returns the config body with the validator added under the
// top-level "etag" key, the way the server answers opted-in agents.
func withETagKey(config []byte, etag string) []byte {
	m := make(map[string]any)
	if err := json.Unmarshal(config, &m); err != nil {
		return config
	}
	m["etag"] = etag
	b, err := json.Marshal(m)
	if err != nil {
		return config
	}
	return b
}

func TestConfigETag_FirstFullThenUnchanged(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := "server-computed-etag-123"
	fullBody := withETagKey(configBody, serverETag)

	var requestCount int
	var lastRequestETag *string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		lastRequestETag = requestETag(r)
		if lastRequestETag != nil && *lastRequestETag == serverETag {
			_, _ = w.Write([]byte(unchangedBody))
			return
		}
		_, _ = w.Write(fullBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: opts in with an empty etag, downloads the full config.
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)
	require.NotNil(t, lastRequestETag, "an enabled agent always sends the etag field")
	require.Empty(t, *lastRequestETag)
	// Client stores the server's etag, not a computed hash.
	require.Equal(t, serverETag, a.configETag)
	require.Equal(t, int64(len(fullBody)), a.lastConfigBodyBytes)

	// Second request: echoes the server's etag and gets the unchanged body.
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
	require.NotNil(t, lastRequestETag)
	require.Equal(t, serverETag, *lastRequestETag)
	// The etag and body bytes are unchanged.
	require.Equal(t, serverETag, a.configETag)
	require.Equal(t, int64(len(fullBody)), a.lastConfigBodyBytes)

	// Verify stats
	require.Equal(t, int64(1), a.stats.ConfigFullResponses())
	require.Equal(t, int64(1), a.stats.ConfigNotModified())
	require.Equal(t, int64(1), a.stats.ConfigConditionalRequests())
	// The unchanged response contributed its own (tiny) body to the sent
	// total, and only the difference to the avoided total.
	unchangedLen := int64(len(`{"etag":"ok"}`))
	require.Equal(t, int64(len(fullBody))+unchangedLen, a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(len(fullBody))-unchangedLen, a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_ChangedConfig(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	bodyB := []byte(`{"packs":{"test":{"queries":{"q2":{"query":"select 2","interval":120}}}}}`)
	etagA := "etag-a"
	etagB := "etag-b"

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		e := requestETag(r)
		switch {
		case e != nil && *e == etagA:
			// Config changed: full new body with the new etag.
			_, _ = w.Write(withETagKey(bodyB, etagB))
		case e != nil && *e == etagB:
			_, _ = w.Write([]byte(unchangedBody))
		default:
			_, _ = w.Write(withETagKey(bodyA, etagA))
		}
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: get body A with etagA
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: send etagA, get body B with etagB (config changed)
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagB, a.configETag)

	// Third request: send etagB, get the unchanged body
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagB, a.configETag)
	require.Equal(t, 3, requestCount)
}

func TestConfigETag_Malformed200(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := "etag-a"

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write(withETagKey(bodyA, etagA))
			return
		}
		// Return malformed JSON: no etag can be extracted at all.
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: install body A
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: malformed response carries no receivable etag, so the
	// previous validator remains.
	err = a.config()
	require.Error(t, err)
	require.Equal(t, etagA, a.configETag)
}

func TestConfigETag_Disabled(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)

	var requestCount int
	var gotETagField bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestETag(r) != nil {
			gotETagField = true
		}
		// An agent that does not opt in gets the config with no etag key.
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(false)
	a.serverAddress = server.URL

	// Two requests, neither should carry the etag field.
	for range 2 {
		err := a.config()
		require.NoError(t, err)
	}
	require.Equal(t, 2, requestCount)
	require.False(t, gotETagField)
	require.Empty(t, a.configETag)

	// No conditional requests or avoided bytes recorded
	require.Equal(t, int64(0), a.stats.ConfigConditionalRequests())
	require.Equal(t, int64(0), a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_InvalidUnchangedWithoutHistory(t *testing.T) {
	// A server may only answer {"etag":"ok"} to an agent that echoed one of
	// its validators; answering it to an empty etag is a protocol violation.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(unchangedBody))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config unchanged response")
}

func TestConfigETag_UnchangedRetainsValidator(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagV1 := "etag-v1"

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write(withETagKey(configBody, etagV1))
			return
		}
		// The unchanged body carries no validator; the agent keeps its own.
		_, _ = w.Write([]byte(unchangedBody))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagV1, a.configETag)

	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagV1, a.configETag)
}

func TestConfigETag_ServerETagUsedNotComputed(t *testing.T) {
	// Verify the client uses the server's etag, not a locally computed hash
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := "opaque-server-etag"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(withETagKey(configBody, serverETag))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	// Client stores the server's opaque etag, not the computed hash
	require.Equal(t, serverETag, a.configETag)
	canonical, err := canonicalConfigBody(configBody)
	require.NoError(t, err)
	require.NotEqual(t, sha256Hex(canonical), a.configETag)
}

func TestConfigETag_DriftDetected(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	// Server sends an etag that differs from the hash of the canonical body
	wrongETag := "different-from-hash"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(withETagKey(configBody, wrongETag))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	// Client uses server's etag regardless
	require.Equal(t, wrongETag, a.configETag)
	// Drift counter incremented (diagnostic only)
	require.Equal(t, int64(1), a.stats.ConfigETagDrift())
}

func TestConfigETag_NoDriftWhenMatch(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	// The validator covers the canonical (etag-less) representation.
	canonical, err := canonicalConfigBody(configBody)
	require.NoError(t, err)
	correctETag := sha256Hex(canonical)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(withETagKey(configBody, correctETag))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err = a.config()
	require.NoError(t, err)
	require.Equal(t, correctETag, a.configETag)
	// No drift when the server etag matches the canonical-body hash
	require.Equal(t, int64(0), a.stats.ConfigETagDrift())
}

func TestConfigETag_Gzip(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := "gzip-etag"
	fullBody := withETagKey(configBody, serverETag)

	// Gzip the full body
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write(fullBody)
	_ = gw.Close()
	gzippedBody := buf.Bytes()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		e := requestETag(r)
		if e != nil && *e == serverETag {
			_, _ = w.Write([]byte(unchangedBody))
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(gzippedBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: Go's default transport auto-adds Accept-Encoding: gzip
	// and transparently decompresses, so the body is decompressed before read.
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)
	// Client uses the server's etag regardless of body encoding
	require.Equal(t, serverETag, a.configETag)
}

func TestSHA256Hex(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "empty",
			data: []byte{},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sha256Hex(tt.data)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStats_RecordFullConfigResponse(t *testing.T) {
	stats := &osquery_perf.Stats{}

	stats.RecordFullConfigResponse(1000, false)
	require.Equal(t, int64(1), stats.ConfigFullResponses())
	require.Equal(t, int64(1000), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(0), stats.ConfigConditionalRequests())

	stats.RecordFullConfigResponse(500, true)
	require.Equal(t, int64(2), stats.ConfigFullResponses())
	require.Equal(t, int64(1500), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(1), stats.ConfigConditionalRequests())
}

func TestStats_RecordConfigNotModified(t *testing.T) {
	stats := &osquery_perf.Stats{}

	// The unchanged body is counted as sent; only the difference is avoided.
	stats.RecordConfigNotModified(13, 1000)
	require.Equal(t, int64(1), stats.ConfigNotModified())
	require.Equal(t, int64(1), stats.ConfigConditionalRequests())
	require.Equal(t, int64(13), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(987), stats.ConfigEstimatedSavedBytes())

	stats.RecordConfigNotModified(13, 500)
	require.Equal(t, int64(2), stats.ConfigNotModified())
	require.Equal(t, int64(2), stats.ConfigConditionalRequests())
	require.Equal(t, int64(26), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(1474), stats.ConfigEstimatedSavedBytes())

	// A prior body no larger than the unchanged body avoids nothing, and must
	// never subtract from the running total.
	stats.RecordConfigNotModified(13, 5)
	require.Equal(t, int64(39), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(1474), stats.ConfigEstimatedSavedBytes())
}

func TestStats_ConfigSavingsCalculation(t *testing.T) {
	stats := &osquery_perf.Stats{}

	// 1 full response (1000 bytes) + 9 unchanged responses of 13 bytes each,
	// so 9*987 bytes were actually avoided and 9*13 were still sent.
	stats.RecordFullConfigResponse(1000, false)
	for range 9 {
		stats.RecordConfigNotModified(13, 1000)
	}

	require.Equal(t, int64(1000+9*13), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(9*987), stats.ConfigEstimatedSavedBytes())

	// Sent + avoided must equal the bytes a non-conditional agent would have
	// downloaded; that identity is what makes the reported percentage honest.
	require.Equal(t, int64(10*1000), stats.ConfigResponseBodyBytes()+stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_StatsConcurrency(t *testing.T) {
	stats := &osquery_perf.Stats{}
	var wg sync.WaitGroup

	// Concurrent updates should be race-free
	for range 100 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			stats.RecordFullConfigResponse(100, false)
		}()
		go func() {
			defer wg.Done()
			stats.RecordConfigNotModified(13, 100)
		}()
		go func() {
			defer wg.Done()
			stats.IncrementConfigETagDrift()
		}()
	}
	wg.Wait()

	require.Equal(t, int64(100), stats.ConfigFullResponses())
	require.Equal(t, int64(100), stats.ConfigNotModified())
	// 100 full responses of 100B, plus 100 unchanged responses of 13B sent and
	// 87B avoided each.
	require.Equal(t, int64(100*100+100*13), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(100*87), stats.ConfigEstimatedSavedBytes())
	require.Equal(t, int64(100), stats.ConfigETagDrift())
}

// TestConfigETag_ProcessingFailureDoesNotRefetch pins the store-on-receive
// semantics that mirror the real osquery client: the validator is committed
// as soon as a well-formed response carrying an etag arrives, BEFORE the
// config is processed. A config the agent fails to process is therefore
// confirmed unchanged on subsequent check-ins (13-byte responses) instead of
// being re-downloaded in full every cycle.
func TestConfigETag_ProcessingFailureDoesNotRefetch(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	// Valid JSON with a bad query type (causes a processing failure).
	badBody := []byte(`{"packs":{"test":{"queries":{"q1":"not_a_map"}}}}`)
	etagA := "etag-a"
	etagB := "etag-b"

	var requestCount, fullResponses int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		e := requestETag(r)
		if e != nil && *e == etagB {
			_, _ = w.Write([]byte(unchangedBody))
			return
		}
		fullResponses++
		if requestCount == 1 {
			_, _ = w.Write(withETagKey(bodyA, etagA))
			return
		}
		// The server's config is now the bad one.
		_, _ = w.Write(withETagKey(badBody, etagB))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: install body A.
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: the bad config downloads once, fails to process, and
	// its etag is stored anyway (received, not applied).
	err = a.config()
	require.Error(t, err)
	require.Equal(t, etagB, a.configETag)

	// Subsequent requests echo the bad config's etag and are answered
	// "unchanged" — the payload is never re-downloaded.
	for range 2 {
		err = a.config()
		require.NoError(t, err)
		require.Equal(t, etagB, a.configETag)
	}
	require.Equal(t, 4, requestCount)
	require.Equal(t, 2, fullResponses, "the bad config must be downloaded exactly once")
}

func TestConfigETag_UnchangedDoesNotReplaceScheduledQueryState(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := "etag-a"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := requestETag(r)
		if e != nil && *e == etagA {
			_, _ = w.Write([]byte(unchangedBody))
			return
		}
		_, _ = w.Write(withETagKey(bodyA, etagA))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: install config
	err := a.config()
	require.NoError(t, err)

	// Verify scheduled query data was installed
	var foundQ1 bool
	a.scheduledQueryData.Range(func(key, value any) bool {
		if key.(string) == "test_q1" {
			foundQ1 = true
		}
		return true
	})
	require.True(t, foundQ1)

	// Second request: the unchanged response must not change scheduled query state
	err = a.config()
	require.NoError(t, err)

	// Verify scheduled query data is still there
	var stillFoundQ1 bool
	a.scheduledQueryData.Range(func(key, value any) bool {
		if key.(string) == "test_q1" {
			stillFoundQ1 = true
		}
		return true
	})
	require.True(t, stillFoundQ1)
}

func TestConfigETag_ResponseBodyBytes(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := "etag-a"
	fullBody := withETagKey(bodyA, etagA)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := requestETag(r)
		if e != nil && *e == etagA {
			_, _ = w.Write([]byte(unchangedBody))
			return
		}
		_, _ = w.Write(fullBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: full download
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, int64(len(fullBody)), a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(0), a.stats.ConfigEstimatedSavedBytes())

	// Second request: unchanged
	err = a.config()
	require.NoError(t, err)
	// The unchanged response contributed its own (tiny) body to the sent
	// total, and only the difference to the avoided total.
	unchangedLen := int64(len(`{"etag":"ok"}`))
	require.Equal(t, int64(len(fullBody))+unchangedLen, a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(len(fullBody))-unchangedLen, a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_JsonUnmarshal(t *testing.T) {
	body := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)

	var parsed struct {
		Packs map[string]struct {
			Queries map[string]any `json:"queries"`
		} `json:"packs"`
	}
	err := json.Unmarshal(body, &parsed)
	require.NoError(t, err)
	require.Contains(t, parsed.Packs, "test")
	require.Contains(t, parsed.Packs["test"].Queries, "q1")
}

func TestConfigETag_IoReadAll(t *testing.T) {
	body := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	reader := bytes.NewReader(body)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, body, data)
}

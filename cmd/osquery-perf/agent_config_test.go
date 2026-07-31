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
		agentIndex:           1,
		serverAddress:        "http://test",
		stats:                stats,
		nodeKey:              "test-node-key",
		configTLSETag:        configTLSETag,
		scheduledQueryData:   new(sync.Map),
		hostIdentityClient:   hostidentity.NewClient(hostidentity.Config{}, false, 0),
	}
	return a
}

func TestConfigETag_First200Then304(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := `"server-computed-etag-123"`

	var requestCount int
	var lastIfNoneMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		lastIfNoneMatch = r.Header.Get("If-None-Match")
		if lastIfNoneMatch == serverETag {
			w.Header().Set("ETag", serverETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", serverETag)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: no validator, should get 200
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)
	// Client stores the server's ETag, not a computed hash
	require.Equal(t, serverETag, a.configETag)
	require.Equal(t, int64(len(configBody)), a.lastConfigBodyBytes)

	// Second request: should send the server's ETag and get 304
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
	require.Equal(t, serverETag, lastIfNoneMatch)
	// ETag and body bytes should be unchanged
	require.Equal(t, serverETag, a.configETag)
	require.Equal(t, int64(len(configBody)), a.lastConfigBodyBytes)

	// Verify stats
	require.Equal(t, int64(1), a.stats.ConfigFullResponses())
	require.Equal(t, int64(1), a.stats.ConfigNotModified())
	require.Equal(t, int64(1), a.stats.ConfigConditionalRequests())
	require.Equal(t, int64(len(configBody)), a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(len(configBody)), a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_ChangedConfig(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	bodyB := []byte(`{"packs":{"test":{"queries":{"q2":{"query":"select 2","interval":120}}}}}`)
	etagA := `"etag-a"`
	etagB := `"etag-b"`

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch == etagA {
			// Config changed, return new body with new ETag
			w.Header().Set("ETag", etagB)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bodyB)
			return
		}
		if ifNoneMatch == etagB {
			w.Header().Set("ETag", etagB)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etagA)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bodyA)
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

	// Third request: send etagB, get 304
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagB, a.configETag)
	require.Equal(t, 3, requestCount)
}

func TestConfigETag_Malformed200(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := `"etag-a"`

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("ETag", etagA)
		if requestCount == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bodyA)
			return
		}
		// Return malformed JSON
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: install body A
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: malformed response, previous validator should remain
	err = a.config()
	require.Error(t, err)
	// Previous ETag should still be installed
	require.Equal(t, etagA, a.configETag)
}

func TestConfigETag_Disabled(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)

	var requestCount int
	var gotIfNoneMatch bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("If-None-Match") != "" {
			gotIfNoneMatch = true
		}
		w.Header().Set("ETag", `"some-etag"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(false)
	a.serverAddress = server.URL

	// Two requests, neither should send If-None-Match
	for range 2 {
		err := a.config()
		require.NoError(t, err)
	}
	require.Equal(t, 2, requestCount)
	require.False(t, gotIfNoneMatch)
	require.Empty(t, a.configETag)

	// No conditional requests or avoided bytes recorded
	require.Equal(t, int64(0), a.stats.ConfigConditionalRequests())
	require.Equal(t, int64(0), a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_Invalid304(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request with no validator should treat 304 as error
	err := a.config()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config 304")
}

func TestConfigETag_304UpdatesValidator(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagV1 := `"etag-v1"`
	etagV2 := `"etag-v2"`

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("ETag", etagV1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(configBody)
			return
		}
		// 304 with a new ETag (server rotated the validator)
		w.Header().Set("ETag", etagV2)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: get etagV1
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagV1, a.configETag)

	// Second request: 304 with new ETag should replace stored validator
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagV2, a.configETag)
}

func TestConfigETag_304WithoutETagRetainsValidator(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagV1 := `"etag-v1"`

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("ETag", etagV1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(configBody)
			return
		}
		// 304 without ETag header — client should retain previous validator
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: get etagV1
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagV1, a.configETag)

	// Second request: 304 without ETag should retain etagV1
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagV1, a.configETag)
}

func TestConfigETag_ServerETagUsedNotComputed(t *testing.T) {
	// Verify the client uses the server's ETag, not a locally computed hash
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := `"opaque-server-etag"`
	// The locally computed hash would be different
	_ = quotedSHA256(configBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", serverETag)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	// Client stores the server's opaque ETag, not the computed hash
	require.Equal(t, serverETag, a.configETag)
	require.NotEqual(t, quotedSHA256(configBody), a.configETag)
}

func TestConfigETag_DriftDetected(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	// Server sends an ETag that differs from what we'd compute locally
	wrongETag := `"different-from-hash"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", wrongETag)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	// Client uses server's ETag regardless
	require.Equal(t, wrongETag, a.configETag)
	// Drift counter incremented (diagnostic only)
	require.Equal(t, int64(1), a.stats.ConfigETagDrift())
}

func TestConfigETag_NoDriftWhenMatch(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	correctETag := quotedSHA256(configBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", correctETag)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	err := a.config()
	require.NoError(t, err)
	require.Equal(t, correctETag, a.configETag)
	// No drift when server ETag matches local computation
	require.Equal(t, int64(0), a.stats.ConfigETagDrift())
}

func TestConfigETag_Gzip(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	serverETag := `"gzip-etag"`

	// Gzip the body
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write(configBody)
	_ = gw.Close()
	gzippedBody := buf.Bytes()

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch == serverETag {
			w.Header().Set("ETag", serverETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", serverETag)
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
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
	// Client uses server's ETag regardless of body encoding
	require.Equal(t, serverETag, a.configETag)
}

func TestQuotedSHA256(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "empty",
			data: []byte{},
			want: `"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quotedSHA256(tt.data)
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

	stats.RecordConfigNotModified(1000)
	require.Equal(t, int64(1), stats.ConfigNotModified())
	require.Equal(t, int64(1), stats.ConfigConditionalRequests())
	require.Equal(t, int64(1000), stats.ConfigEstimatedSavedBytes())

	stats.RecordConfigNotModified(500)
	require.Equal(t, int64(2), stats.ConfigNotModified())
	require.Equal(t, int64(2), stats.ConfigConditionalRequests())
	require.Equal(t, int64(1500), stats.ConfigEstimatedSavedBytes())
}

func TestStats_ConfigSavingsCalculation(t *testing.T) {
	stats := &osquery_perf.Stats{}

	// Simulate: 1 full response (1000 bytes) + 9 not-modified (each avoiding 1000 bytes)
	stats.RecordFullConfigResponse(1000, false)
	for range 9 {
		stats.RecordConfigNotModified(1000)
	}

	require.Equal(t, int64(1000), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(9000), stats.ConfigEstimatedSavedBytes())
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
			stats.RecordConfigNotModified(100)
		}()
		go func() {
			defer wg.Done()
			stats.IncrementConfigETagDrift()
		}()
	}
	wg.Wait()

	require.Equal(t, int64(100), stats.ConfigFullResponses())
	require.Equal(t, int64(100), stats.ConfigNotModified())
	require.Equal(t, int64(10000), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(10000), stats.ConfigEstimatedSavedBytes())
	require.Equal(t, int64(100), stats.ConfigETagDrift())
}

func TestConfigETag_ParsingFailurePreservesValidator(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := `"etag-a"`

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("ETag", etagA)
		if requestCount == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bodyA)
			return
		}
		// Return valid JSON but with bad query type (causes type assertion failure)
		badBody := []byte(`{"packs":{"test":{"queries":{"q1":"not_a_map"}}}}`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(badBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: install body A
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: bad query type, previous validator should remain
	err = a.config()
	require.Error(t, err)
	require.Equal(t, etagA, a.configETag)
}

func TestConfigETag_Invalid304WithoutSentETag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL
	a.configETag = ""

	err := a.config()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config 304")
}

func TestConfigETag_304DoesNotReplaceScheduledQueryState(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := `"etag-a"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", etagA)
		if r.Header.Get("If-None-Match") == etagA {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bodyA)
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

	// Second request: 304 should not change scheduled query state
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
	etagA := `"etag-a"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", etagA)
		if r.Header.Get("If-None-Match") == etagA {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bodyA)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: 200
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(0), a.stats.ConfigEstimatedSavedBytes())

	// Second request: 304
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigEstimatedSavedBytes())
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

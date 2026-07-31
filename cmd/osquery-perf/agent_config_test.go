package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	// Server tracks the body it should return
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	expectedETag := quotedSHA256(configBody)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch == expectedETag {
			w.Header().Set("ETag", expectedETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", expectedETag)
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
	require.Equal(t, expectedETag, a.configETag)
	require.Equal(t, int64(len(configBody)), a.lastConfigBodyBytes)

	// Second request: should send If-None-Match and get 304
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
	// ETag and body bytes should be unchanged
	require.Equal(t, expectedETag, a.configETag)
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
	etagA := quotedSHA256(bodyA)
	etagB := quotedSHA256(bodyB)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ifNoneMatch := r.Header.Get("If-None-Match")
		if ifNoneMatch == etagA {
			// Config changed, return new body
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

	// First request: get body A
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, etagA, a.configETag)

	// Second request: send A's tag, get body B (config changed)
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagB, a.configETag)

	// Third request: send B's tag, get 304
	err = a.config()
	require.NoError(t, err)
	require.Equal(t, etagB, a.configETag)
	require.Equal(t, 3, requestCount)
}

func TestConfigETag_Malformed200(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := quotedSHA256(bodyA)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("ETag", etagA)
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
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(configBody)
	}))
	defer server.Close()

	a := newTestAgent(false)
	a.serverAddress = server.URL

	// Two requests, neither should send If-None-Match
	for i := 0; i < 2; i++ {
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

func TestConfigETag_304WithBody(t *testing.T) {
	// Note: httptest.NewServer does not actually send a body for 304 responses,
	// so we cannot directly test this case with httptest. Instead, we verify
	// the logic by checking that a valid 304 (empty body, sent ETag) succeeds.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
		// No body — this is the correct 304 behavior
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL
	a.configETag = `"some-etag"`

	// A valid 304 (empty body, sent ETag) should succeed
	err := a.config()
	require.NoError(t, err)
}

func TestConfigETag_ServerHeaderMismatch(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	correctETag := quotedSHA256(configBody)
	wrongETag := `"wrong-etag-value"`

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
	// Locally calculated validator should be authoritative
	require.Equal(t, correctETag, a.configETag)
	// Mismatch counter should have incremented
	require.Equal(t, int64(1), a.stats.ConfigETagHeaderMismatches())
}

func TestConfigETag_Gzip(t *testing.T) {
	configBody := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	expectedETag := quotedSHA256(configBody)

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
		if ifNoneMatch == expectedETag {
			w.Header().Set("ETag", expectedETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", expectedETag)
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gzippedBody)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL

	// First request: the server sends gzip-encoded body. Go's default transport
	// auto-adds Accept-Encoding: gzip and transparently decompresses, so the
	// ETag is computed from the decompressed bytes (matching the osquery PoC).
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)
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
		{
			name: "simple",
			data: []byte(`{"test": true}`),
			want: fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(`{"test": true}`))),
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
	for i := 0; i < 9; i++ {
		stats.RecordConfigNotModified(1000)
	}

	// Baseline = 1000 (downloaded) + 9000 (avoided) = 10000
	// Savings = 9000 / 10000 = 90%
	require.Equal(t, int64(1000), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(9000), stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_StatsConcurrency(t *testing.T) {
	stats := &osquery_perf.Stats{}
	var wg sync.WaitGroup

	// Concurrent updates should be race-free
	for i := 0; i < 100; i++ {
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
			stats.IncrementConfigETagHeaderMismatches()
		}()
	}
	wg.Wait()

	require.Equal(t, int64(100), stats.ConfigFullResponses())
	require.Equal(t, int64(100), stats.ConfigNotModified())
	require.Equal(t, int64(10000), stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(10000), stats.ConfigEstimatedSavedBytes())
	require.Equal(t, int64(100), stats.ConfigETagHeaderMismatches())
}

func TestConfigETag_ParsingFailurePreservesValidator(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := quotedSHA256(bodyA)

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("ETag", etagA)
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
	// Next request should still send the old validator
}

func TestConfigETag_Invalid304WithoutSentETag(t *testing.T) {
	// A 304 received without sending a conditional request is an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	a := newTestAgent(true)
	a.serverAddress = server.URL
	// Ensure no ETag is set
	a.configETag = ""

	err := a.config()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid config 304")
}

func TestConfigETag_304DoesNotReplaceScheduledQueryState(t *testing.T) {
	bodyA := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	etagA := quotedSHA256(bodyA)

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
	etagA := quotedSHA256(bodyA)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etagA {
			w.Header().Set("ETag", etagA)
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

	// First request: 200
	err := a.config()
	require.NoError(t, err)
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigResponseBodyBytes())
	require.Equal(t, int64(0), a.stats.ConfigEstimatedSavedBytes())

	// Second request: 304
	err = a.config()
	require.NoError(t, err)
	// Body bytes should not change (no new body downloaded)
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigResponseBodyBytes())
	// Saved bytes should equal the last body size
	require.Equal(t, int64(len(bodyA)), a.stats.ConfigEstimatedSavedBytes())
}

func TestConfigETag_JsonUnmarshal(t *testing.T) {
	// Verify that json.Unmarshal works correctly with the body
	body := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)

	var parsed struct {
		Packs map[string]struct {
			Queries map[string]interface{} `json:"queries"`
		} `json:"packs"`
	}
	err := json.Unmarshal(body, &parsed)
	require.NoError(t, err)
	require.Contains(t, parsed.Packs, "test")
	require.Contains(t, parsed.Packs["test"].Queries, "q1")
}

func TestConfigETag_IoReadAll(t *testing.T) {
	// Verify io.ReadAll works correctly
	body := []byte(`{"packs":{"test":{"queries":{"q1":{"query":"select 1","interval":60}}}}}`)
	reader := bytes.NewReader(body)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, body, data)
}

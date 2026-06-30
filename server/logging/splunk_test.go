package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplunkWrite(t *testing.T) {
	ctx := t.Context()

	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, splunkHECPath, r.URL.Path)
		assert.Equal(t, "Splunk test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var err error
		receivedBody, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "main", "fleet", "fleet:json", false, slog.Default())
	require.NoError(t, err)

	err = writer.Write(ctx, logs)
	require.NoError(t, err)
	require.NotEmpty(t, receivedBody)

	// The body should be concatenated JSON objects (one per log entry).
	decoder := json.NewDecoder(bytes.NewReader(receivedBody))
	var events []splunkEvent
	for decoder.More() {
		var evt splunkEvent
		err := decoder.Decode(&evt)
		require.NoError(t, err)
		events = append(events, evt)
	}

	require.Len(t, events, 3)
	for i, evt := range events {
		assert.JSONEq(t, string(logs[i]), string(evt.Event))
		assert.Equal(t, "main", evt.Index)
		assert.Equal(t, "fleet", evt.Source)
		assert.Equal(t, "fleet:json", evt.SourceType)
		assert.NotZero(t, evt.Time)
	}
}

func TestSplunkWriteEmpty(t *testing.T) {
	ctx := t.Context()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Fatal("should not send request for empty logs")
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	err = writer.Write(ctx, []json.RawMessage{})
	require.NoError(t, err)
}

func TestSplunkServerError(t *testing.T) {
	ctx := t.Context()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, `{"text":"Invalid token","code":4}`, http.StatusForbidden)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	err = writer.Write(ctx, logs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "403")
}

func TestSplunkHealthCheckFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.Error(t, err)
	require.Contains(t, err.Error(), "health check")
}

func TestSplunkRecordTooBig(t *testing.T) {
	ctx := t.Context()

	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	// Create one normal log and one oversized log (>1MB)
	normalLog := json.RawMessage(`{"normal":"event"}`)
	bigPayload := make([]byte, splunkMaxSizeOfRecord+1)
	for i := range bigPayload {
		bigPayload[i] = 'x'
	}
	oversizedLog := json.RawMessage(`{"big":"` + string(bigPayload) + `"}`)

	err = writer.Write(ctx, []json.RawMessage{normalLog, oversizedLog})
	require.NoError(t, err)

	// Only the normal event should have been sent; the oversized one should be dropped
	decoder := json.NewDecoder(bytes.NewReader(receivedBody))
	var count int
	for decoder.More() {
		var evt splunkEvent
		err := decoder.Decode(&evt)
		require.NoError(t, err)
		count++
	}
	assert.Equal(t, 1, count, "only the normal-sized event should be sent")
}

func TestSplunkSplitBatchBySize(t *testing.T) {
	ctx := t.Context()

	var batchCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		batchCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	// Create logs that together exceed splunkMaxBatchSize (1MB).
	// Each log wraps to ~10KB after HEC envelope, so ~120 logs should exceed 1MB.
	var largeLogs []json.RawMessage
	payload := make([]byte, 10000)
	for i := range payload {
		payload[i] = 'a'
	}
	for range 120 {
		largeLogs = append(largeLogs, json.RawMessage(`{"data":"`+string(payload)+`"}`))
	}

	err = writer.Write(ctx, largeLogs)
	require.NoError(t, err)
	assert.Greater(t, batchCount, 1, "should split into multiple batches")
}

func TestSplunkRetryOnServiceUnavailable(t *testing.T) {
	ctx := t.Context()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		callCount++
		if callCount <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	err = writer.Write(ctx, logs)
	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "should retry twice then succeed on third attempt")
}

func TestSplunkRetryExhausted(t *testing.T) {
	ctx := t.Context()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splunkHealthPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	writer, err := NewSplunkLogWriter(server.URL, "test-token", "", "", "", false, slog.Default())
	require.NoError(t, err)

	err = writer.Write(ctx, logs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "503")
	// 1 initial attempt + 8 retries = 9 total
	assert.Equal(t, splunkMaxRetries+1, callCount, "should exhaust all retries")
}

func TestSplunkMissingConfig(t *testing.T) {
	_, err := NewSplunkLogWriter("", "token", "", "", "", false, slog.Default())
	require.Error(t, err)
	require.Contains(t, err.Error(), "URL")

	_, err = NewSplunkLogWriter("http://localhost", "", "", "", "", false, slog.Default())
	require.Error(t, err)
	require.Contains(t, err.Error(), "token")
}

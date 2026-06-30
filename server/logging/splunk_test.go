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
		require.Equal(t, splunkHECPath, r.URL.Path)
		require.Equal(t, "Splunk test-token", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var err error
		receivedBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
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

func TestSplunkMissingConfig(t *testing.T) {
	_, err := NewSplunkLogWriter("", "token", "", "", "", false, slog.Default())
	require.Error(t, err)
	require.Contains(t, err.Error(), "URL")

	_, err = NewSplunkLogWriter("http://localhost", "", "", "", "", false, slog.Default())
	require.Error(t, err)
	require.Contains(t, err.Error(), "token")
}

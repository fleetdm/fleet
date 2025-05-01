package logging

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestWebhookSubmission(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	var body struct {
		Timestamp time.Time         `json:"timestamp"`
		Details   []json.RawMessage `json:"details"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
	}))
	writer, err := NewWebhookLogWriter(server.URL, logger)
	require.NoError(t, err)

	logs := []json.RawMessage{
		json.RawMessage(`{"pack":"fruit"}`),
		json.RawMessage(`{"information":213}`),
		json.RawMessage(`{"gordon":"freeman"}`),
	}

	err = writer.Write(ctx, logs)
	require.NoError(t, err)

	require.Len(t, body.Details, 3)
	require.Equal(t, logs, body.Details)
}

func TestWebhookFailure(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	writer, err := NewWebhookLogWriter(server.URL, logger)
	require.NoError(t, err)

	// Should always return no error and not stall, even if bad things happen

	// Bad return code
	err = writer.Write(ctx, []json.RawMessage{json.RawMessage("{}")})
	require.NoError(t, err)

	// No server
	server.Close()
	err = writer.Write(ctx, []json.RawMessage{json.RawMessage("{}")})
	require.NoError(t, err)
}

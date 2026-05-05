package logging

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeCloudRunServiceWriterWithClient(client *http.Client, serviceURL string) *cloudRunServiceLogWriter {
	return &cloudRunServiceLogWriter{
		client: client,
		url:    serviceURL,
		logger: slog.New(slog.DiscardHandler),
	}
}

func TestCloudRunServiceSendsEachRawLog(t *testing.T) {
	ctx := context.Background()
	var bodies []json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		bodies = append(bodies, json.RawMessage(body))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	writer := makeCloudRunServiceWriterWithClient(server.Client(), server.URL)

	err := writer.Write(ctx, logs)
	require.NoError(t, err)
	require.Equal(t, logs, bodies)
}

func TestCloudRunServiceReturnsErrorOnHTTPError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	writer := makeCloudRunServiceWriterWithClient(server.Client(), server.URL)

	err := writer.Write(ctx, []json.RawMessage{json.RawMessage(`{"foo":"bar"}`)})
	require.ErrorContains(t, err, "status 400")
}

func TestCloudRunServiceReturnsErrorOnRequestFailure(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	serviceURL := server.URL
	server.Close()

	writer := makeCloudRunServiceWriterWithClient(server.Client(), serviceURL)

	err := writer.Write(ctx, []json.RawMessage{json.RawMessage(`{"foo":"bar"}`)})
	require.Error(t, err)
}

func TestNewCloudRunServiceLogWriterRequiresURL(t *testing.T) {
	_, err := NewCloudRunServiceLogWriter(context.Background(), "", "", slog.New(slog.DiscardHandler))
	require.ErrorContains(t, err, "Cloud Run service URL missing")
}

func TestNewCloudRunServiceLogWriterRequiresValidURL(t *testing.T) {
	_, err := NewCloudRunServiceLogWriter(context.Background(), "not-a-url", "", slog.New(slog.DiscardHandler))
	require.ErrorContains(t, err, "invalid Cloud Run service URL")
}

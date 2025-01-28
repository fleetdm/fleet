package itunes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetBaseURL(t *testing.T) {
	t.Run("Default URL", func(t *testing.T) {
		os.Setenv("FLEET_DEV_ITUNES_URL", "")
		require.Equal(t, "https://itunes.apple.com/lookup", getBaseURL())
	})

	t.Run("Custom URL", func(t *testing.T) {
		customURL := "http://localhost:8000"
		os.Setenv("FLEET_DEV_ITUNES_URL", customURL)
		require.Equal(t, customURL, getBaseURL())
	})
}

func setupFakeServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	os.Setenv("FLEET_DEV_ITUNES_URL", server.URL)
	t.Cleanup(server.Close)
}

func TestDoRetries(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantCalls int
		wantErr   bool
	}{
		{
			name: "success status code",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "bad requests",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "500 requests retries",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 4,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < tt.wantCalls {
					tt.handler(w, r)
					return
				}
			})

			start := time.Now()
			req, err := http.NewRequest(http.MethodGet, os.Getenv("FLEET_DEV_ITUNES_URL"), nil)
			require.NoError(t, err)
			err = do[any](req, nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantCalls, calls)
			require.WithinRange(t, time.Now(), start, start.Add(time.Duration(tt.wantCalls)*time.Second))
		})
	}
}

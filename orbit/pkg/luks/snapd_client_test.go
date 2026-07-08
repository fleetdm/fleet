//go:build linux

package luks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSnapdClient returns a snapdClient pointed at the given test server,
// bypassing the unix-socket transport.
func newTestSnapdClient(srv *httptest.Server) *snapdClient {
	return &snapdClient{httpClient: srv.Client(), baseURL: srv.URL}
}

func TestSnapdRequestSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/system-volumes", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"recovery-key":"55055-39320"}}`))
	}))
	defer srv.Close()

	var out struct {
		RecoveryKey string `json:"recovery-key"`
	}
	require.NoError(t, newTestSnapdClient(srv).requestSync(context.Background(), http.MethodPost, "/v2/system-volumes", map[string]string{"action": "x"}, &out))
	assert.Equal(t, "55055-39320", out.RecoveryKey)
}

func TestSnapdRequestAsync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/system-volumes":
			_, _ = w.Write([]byte(`{"type":"async","status-code":202,"change":"42"}`))
		case "/v2/changes/42":
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"ready":true,"status":"Done","data":{"recovery-key":"11111-22222"}}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	var out struct {
		RecoveryKey string `json:"recovery-key"`
	}
	require.NoError(t, newTestSnapdClient(srv).requestAsync(context.Background(), http.MethodPost, "/v2/system-volumes", nil, &out))
	assert.Equal(t, "11111-22222", out.RecoveryKey)
}

func TestSnapdRequestAsyncChangeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/system-volumes":
			_, _ = w.Write([]byte(`{"type":"async","status-code":202,"change":"7"}`))
		case "/v2/changes/7":
			_, _ = w.Write([]byte(`{"type":"sync","status-code":200,"result":{"ready":true,"status":"Error","err":"boom"}}`))
		}
	}))
	defer srv.Close()

	err := newTestSnapdClient(srv).requestAsync(context.Background(), http.MethodPost, "/v2/system-volumes", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestSnapdErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"type":"error","status-code":403,"result":{"message":"access denied","kind":"login-required"}}`))
	}))
	defer srv.Close()

	err := newTestSnapdClient(srv).requestSync(context.Background(), http.MethodGet, "/v2/system-volumes", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

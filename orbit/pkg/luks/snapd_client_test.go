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

func TestSnapdAPIErrorIsConflict(t *testing.T) {
	cases := []struct {
		name string
		err  *snapdAPIError
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "409 status", err: &snapdAPIError{StatusCode: http.StatusConflict}, want: true},
		{name: "kind resource-already-exists", err: &snapdAPIError{StatusCode: 400, Kind: "resource-already-exists"}, want: true},
		{name: "kind snap-change-conflict", err: &snapdAPIError{StatusCode: 400, Kind: "snap-change-conflict"}, want: true},
		{
			// Regression: snapd 2.75 returns HTTP 400 with a plural-verb
			// message when a name-only keyslotRef expands to system-data and
			// system-save, which is our add-recovery-key case.
			name: "message plural: key slots ... already exist",
			err: &snapdAPIError{
				StatusCode: 400,
				Message:    `key slots [(container-role: "system-data", name: "fleet-escrow"), (container-role: "system-save", name: "fleet-escrow")] already exist`,
			},
			want: true,
		},
		{
			name: "message singular: key slot ... already exists",
			err:  &snapdAPIError{StatusCode: 400, Message: `key slot "fleet-escrow" already exists`},
			want: true,
		},
		{name: "auth failure is not a conflict", err: &snapdAPIError{StatusCode: 401, Message: "access denied"}, want: false},
		{name: "generic bad request is not a conflict", err: &snapdAPIError{StatusCode: 400, Message: "malformed request"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.err.isConflict())
		})
	}
}

package endpointer

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRemovedPaths(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/fleet/kept", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// A live route with the same shape as the method-limited removed path below, registered first as in the API.
	r.HandleFunc("/api/v1/fleet/partly/{id:[0-9]+}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }).
		Methods(http.MethodGet)
	var logs bytes.Buffer
	RegisterRemovedPaths(r, []string{"v1", "2022-04"}, slog.New(slog.NewJSONHandler(&logs, nil)), []RemovedPath{
		{
			Path:    "/api/_version_/fleet/gone/{id}",
			Message: func(method string) string { return "gone via " + method },
		},
		{
			Path:    "/api/_version_/fleet/partly/{name}",
			Methods: []string{http.MethodDelete},
			Message: func(method string) string { return "partly gone via " + method },
		},
	})

	removedRequests := 0
	for _, c := range []struct {
		name, method, path string
		wantStatus         int
		wantMessage        string
	}{
		{"v1", http.MethodGet, "/api/v1/fleet/gone/1?token=secret", http.StatusGone, "gone via GET"},
		{"dated version", http.MethodPost, "/api/2022-04/fleet/gone/abc", http.StatusGone, "gone via POST"},
		{"head", http.MethodHead, "/api/v1/fleet/gone/1", http.StatusGone, ""},
		{"latest", http.MethodDelete, "/api/latest/fleet/gone/1", http.StatusGone, "gone via DELETE"},
		{"unknown version", http.MethodGet, "/api/v9/fleet/gone/1", http.StatusNotFound, ""},
		{"other path", http.MethodGet, "/api/v1/fleet/kept", http.StatusOK, ""},
		{"limited to its method", http.MethodDelete, "/api/latest/fleet/partly/abc", http.StatusGone, "partly gone via DELETE"},
		{"limited, live route with the same shape", http.MethodGet, "/api/v1/fleet/partly/1", http.StatusOK, ""},
		{"limited, other method", http.MethodGet, "/api/v1/fleet/partly/abc", http.StatusMethodNotAllowed, ""},
		{"limited, other method on a live route's value", http.MethodPut, "/api/v1/fleet/partly/1", http.StatusMethodNotAllowed, ""},
		{"limited, head", http.MethodHead, "/api/v1/fleet/partly/abc", http.StatusMethodNotAllowed, ""},
	} {
		if c.wantStatus == http.StatusGone {
			removedRequests++
		}
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
			assert.Equal(t, c.wantStatus, rec.Code)
			if c.wantMessage == "" {
				return
			}
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
			var body JsonError
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "Gone", body.Message)
			assert.Equal(t, baseError(c.wantMessage), body.Errors)
		})
	}

	// Each request to a removed path, and only those, is logged as a warning, without the query string, which can carry secrets.
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, removedRequests)
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))
	assert.Equal(t, "WARN", entry["level"])
	assert.Equal(t, "/api/v1/fleet/gone/1", entry["uri"])
	assert.Equal(t, "/api/_version_/fleet/gone/{id}", entry["removed_path"])
	assert.NotContains(t, logs.String(), "secret")
}

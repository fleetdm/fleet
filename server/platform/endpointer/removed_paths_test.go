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
	var logs bytes.Buffer
	RegisterRemovedPaths(r, []string{"v1", "2022-04"}, slog.New(slog.NewJSONHandler(&logs, nil)), []RemovedPath{{
		Path:    "/api/_version_/fleet/gone/{id}",
		Message: func(method string) string { return "gone via " + method },
	}})

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

func TestRegisterRemovedPathsWithOverlap(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/fleet/items/{id}/{name}/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }).
		Methods(http.MethodPost)
	var logs bytes.Buffer
	RegisterRemovedPaths(r, []string{"v1"}, slog.New(slog.NewJSONHandler(&logs, nil)), []RemovedPath{{
		Path:    "/api/_version_/fleet/items/{id}/old/{name}",
		Overlap: RemovedPathOverlap{Method: http.MethodPost, Route: "/api/_version_/fleet/items/{id}/{old_segment}/{name}"},
		Message: func(method string) string { return "gone via " + method },
	}})

	removedRequests := 0
	for _, c := range []struct {
		name, method, path string
		wantStatus         int
		wantBody           string
	}{
		{"removed path on the overlapping method", http.MethodPost, "/api/v1/fleet/items/1/old/abc", http.StatusGone, ""},
		{"removed path on another method", http.MethodGet, "/api/v1/fleet/items/1/old/abc", http.StatusGone, ""},
		{"live route wins the overlap", http.MethodPost, "/api/v1/fleet/items/1/old/live", http.StatusOK, ""},
		{"live route", http.MethodPost, "/api/v1/fleet/items/1/abc/live", http.StatusOK, ""},
		{"wrong method on the live route", http.MethodPut, "/api/v1/fleet/items/1/abc/live", http.StatusMethodNotAllowed, ""},
		{"other value answers like an unknown path", http.MethodPost, "/api/v1/fleet/items/1/other/abc", http.StatusNotFound, "404 page not found\n"},
	} {
		if c.wantStatus == http.StatusGone {
			removedRequests++
		}
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
			assert.Equal(t, c.wantStatus, rec.Code)
			if c.wantBody != "" {
				assert.Equal(t, c.wantBody, rec.Body.String())
			}
		})
	}
	assert.Equal(t, removedRequests, strings.Count(logs.String(), "request to a removed path"), "only requests to the removed path are logged")
}

func TestLiteralSegmentsMatch(t *testing.T) {
	t.Parallel()

	const template = "/api/_version_/fleet/hosts/{host_id}/configuration_profiles/resend/{profile_uuid}"
	for _, c := range []struct {
		name        string
		requestPath string
		want        bool
	}{
		{"removed path", "/api/v1/fleet/hosts/1/configuration_profiles/resend/abc", true},
		{"removed path under a URL prefix", "/prefix/api/latest/fleet/hosts/1/configuration_profiles/resend/abc", true},
		{"other value in the literal segment", "/api/v1/fleet/hosts/1/configuration_profiles/foo/abc", false},
		{"live spelling", "/api/v1/fleet/hosts/1/configuration_profiles/abc/resend", false},
		{"too few segments", "/resend/abc", false},
		{"empty", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, literalSegmentsMatch(template, c.requestPath))
		})
	}
}

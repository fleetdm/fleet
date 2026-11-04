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

func TestRegisterRemovedPathsWithOverlap(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	var logs bytes.Buffer
	RegisterRemovedPaths(r, []string{"v1"}, slog.New(slog.NewJSONHandler(&logs, nil)), []RemovedPath{{
		Path: "/api/_version_/fleet/items/{id}/old/{name}",
		Overlap: RemovedPathOverlap{
			Methods:       []string{http.MethodPost, http.MethodDelete},
			RouteTemplate: "/api/_version_/fleet/items/{id}/{old_segment}/{name}",
			LivePaths:     map[string]string{"/api/_version_/fleet/items/{id}/live/{name}": http.MethodGet},
		},
		Message: func(method string) string { return "gone via " + method },
	}})

	for _, c := range []struct {
		name, method, path string
		wantStatus         int
		wantBody           string
	}{
		{"overlapping method", http.MethodPost, "/api/v1/fleet/items/1/old/abc", http.StatusGone, "gone via POST"},
		{"other method", http.MethodGet, "/api/v1/fleet/items/1/old/abc", http.StatusGone, "gone via GET"},
		{"another overlapping method", http.MethodDelete, "/api/v1/fleet/items/1/old/abc", http.StatusGone, "gone via DELETE"},
		{"live path on a method it does not serve", http.MethodPost, "/api/v1/fleet/items/1/live/abc", http.StatusMethodNotAllowed, ""},
		{"other value is an unknown path", http.MethodPost, "/api/v1/fleet/items/1/other/abc", http.StatusNotFound, "404 page not found\n"},
		{"other value on another method is a mismatch", http.MethodPut, "/api/v1/fleet/items/1/other/abc", http.StatusMethodNotAllowed, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
			assert.Equal(t, c.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), c.wantBody)
		})
	}
	assert.Equal(t, 3, strings.Count(logs.String(), "request to a removed path"), "only the 410s are logged")
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
		{"under a URL prefix", "/prefix/api/latest/fleet/hosts/1/configuration_profiles/resend/abc", true},
		{"other literal", "/api/v1/fleet/hosts/1/configuration_profiles/foo/abc", false},
		{"too few segments", "/resend/abc", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, literalSegmentsMatch(template, c.requestPath))
		})
	}
}

func TestRegisterRemovedPathsWithMethodsAndOverlap(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	RegisterRemovedPaths(r, []string{"v1"}, slog.New(slog.DiscardHandler), []RemovedPath{
		{
			// The overlap route takes the only removed method.
			Path:    "/api/_version_/fleet/items/{id}/old/{name}",
			Methods: []string{http.MethodPost},
			Overlap: RemovedPathOverlap{Methods: []string{http.MethodPost}, RouteTemplate: "/api/_version_/fleet/items/{id}/{old_segment}/{name}"},
			Message: func(method string) string { return "gone via " + method },
		},
		{
			// The overlap is for a method that wasn't removed, so it isn't registered.
			Path:    "/api/_version_/fleet/things/{id}/old/{name}",
			Methods: []string{http.MethodDelete},
			Overlap: RemovedPathOverlap{Methods: []string{http.MethodPost}, RouteTemplate: "/api/_version_/fleet/things/{id}/{old_segment}/{name}"},
			Message: func(method string) string { return "gone via " + method },
		},
	})

	for _, c := range []struct {
		name, method, path string
		wantStatus         int
	}{
		{"overlap method", http.MethodPost, "/api/v1/fleet/items/1/old/abc", http.StatusGone},
		{"overlap method, other literal", http.MethodPost, "/api/v1/fleet/items/1/other/abc", http.StatusNotFound},
		{"method that wasn't removed", http.MethodGet, "/api/v1/fleet/items/1/old/abc", http.StatusMethodNotAllowed},
		{"removed method without its overlap", http.MethodDelete, "/api/v1/fleet/things/1/old/abc", http.StatusGone},
		{"overlap for a method that wasn't removed", http.MethodPost, "/api/v1/fleet/things/1/old/abc", http.StatusMethodNotAllowed},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
			assert.Equal(t, c.wantStatus, rec.Code)
		})
	}
}

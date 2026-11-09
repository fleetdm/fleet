package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

// TestRemovedPaths covers every removed path in one place, so removing a path only means adding it to removedPathTemplates.
func TestRemovedPaths(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, removedPathTemplates)
	// removedPaths panics when a path has no removednames entry.
	require.Len(t, removedPaths(), len(removedPathTemplates))

	pathVar := regexp.MustCompile(`\{\w+(:[^}]+)?\}`)
	// Aliases write a variable as {id:[0-9]+} and the table as {id}, and mux treats differently named variables as the same route.
	shape := func(path string) string { return pathVar.ReplaceAllString(path, "{}") }
	var removedShapes []string
	for _, template := range removedPathTemplates {
		removedShapes = append(removedShapes, shape(template))
	}
	for _, alias := range deprecatedPathAliases {
		for _, deprecated := range alias.DeprecatedPaths {
			assert.NotContains(t, removedShapes, shape(deprecated), "a removed path must not also be served as an alias of %s", alias.PrimaryPath)
		}
	}

	// A router of its own, not MakeHandler's: that one instruments every route with global Prometheus counters, which TestAPIRoutesMetrics
	// asserts exactly. TestAPIRoutesConflicts checks these routes against the real router.
	router := mux.NewRouter()
	endpointer.RegisterRemovedPaths(router, []string{"v1", "2022-04"}, slog.New(slog.DiscardHandler), removedPaths())

	// Fill in the path variables with a value that matches any of them.
	for _, template := range removedPathTemplates {
		entry, ok := removedPathEntry(template)
		require.True(t, ok, template)
		for _, version := range []string{"v1", "latest"} {
			path := pathVar.ReplaceAllString(strings.Replace(template, "/_version_/", "/"+version+"/", 1), "1")
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
				http.MethodHead, http.MethodOptions} {
				t.Run(method+" "+path, func(t *testing.T) {
					t.Parallel() // requests with different methods share the route, so -race catches state shared between them
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
					if entry.Context != "" && method != entry.Context {
						// This router holds only the removed paths, so a method the entry didn't remove gets 405, not 410.
						require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
						return
					}
					require.Equal(t, http.StatusGone, rec.Code)
					if method == http.MethodHead {
						return // a response to HEAD has no body
					}

					var body struct {
						Message string              `json:"message"`
						Errors  []map[string]string `json:"errors"`
					}
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
					assert.Equal(t, "Gone", body.Message)
					require.Len(t, body.Errors, 1)
					want := fmt.Sprintf(`The "%s %s" endpoint is not available anymore. What to use instead: %s`,
						method, strings.TrimPrefix(entry.Old, "/api/_version_/fleet"), removednames.GuideURL)
					assert.Equal(t, want, body.Errors[0]["reason"])
				})
			}
		}
	}
}

// TestRemovedPathOverlaps covers the requests that the wildcard of an overlap catches without being requests to the removed path.
func TestRemovedPathOverlaps(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	endpointer.RegisterRemovedPaths(router, []string{"v1"}, slog.New(slog.DiscardHandler), removedPaths())

	for _, c := range []struct {
		method, path string
		wantStatus   int
		wantAllow    string
	}{
		{http.MethodGet, "/api/v1/fleet/scripts/batch/abc/cancel", http.StatusMethodNotAllowed, http.MethodPost},
		{http.MethodGet, "/api/v1/fleet/scripts/batch/abc/foo", http.StatusNotFound, ""},
		{http.MethodPost, "/api/v1/fleet/hosts/1/configuration_profiles/foo/bar", http.StatusNotFound, ""},
	} {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
			assert.Equal(t, c.wantStatus, rec.Code)
			assert.Equal(t, c.wantAllow, rec.Header().Get("Allow"))
		})
	}
}

func TestRemovedPathForEntryWithMethod(t *testing.T) {
	t.Parallel()

	const template = "/api/_version_/fleet/widgets/{name}"
	pinned := removednames.Entry{
		Kind: removednames.KindPath, Scope: removednames.ScopeRequest, Old: template, Context: http.MethodDelete,
		Format: removednames.EndpointFormat{},
	}
	removed := removedPathFor(template, pinned)
	assert.Equal(t, []string{http.MethodDelete}, removed.Methods)
	assert.Equal(t, `The "DELETE /widgets/{name}" endpoint is not available anymore. What to use instead: `+removednames.GuideURL,
		removed.Message(http.MethodDelete))

	unpinned := pinned
	unpinned.Context = ""
	assert.Empty(t, removedPathFor(template, unpinned).Methods)
}

// Send only requests that no route handles, because a handled request would increment the global
// Prometheus request counters and make TestAPIRoutesMetrics fail with more requests than it expects.
func TestRemovedSoftwarePathsNextToLiveRoutes(t *testing.T) {
	svc, _ := newTestService(t, new(mock.Store), nil, nil)
	limitStore, _ := memstore.New(0)
	handler, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)

	for _, c := range []struct {
		path       string
		wantStatus int
	}{
		// request POST-only software routes with GET, each should still return 405
		{"/api/latest/fleet/software/package", http.StatusMethodNotAllowed},
		{"/api/latest/fleet/software/batch", http.StatusMethodNotAllowed},
		{"/api/latest/fleet/software/web_apps", http.StatusMethodNotAllowed},
		// request software paths that are not a number, each should still return 404
		{"/api/latest/fleet/software/foo", http.StatusNotFound},
		{"/api/latest/fleet/software/123abc", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		require.Equal(t, c.wantStatus, rec.Code, c.path)
	}
}

package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemovedPaths covers every removed path in one place, so removing a path only means adding it to removedPathTemplates.
func TestRemovedPaths(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, removedPathTemplates)
	// removedPaths panics when a path has no removednames entry.
	require.Len(t, removedPaths(), len(removedPathTemplates))

	// An entry for one method is only ever for a removed endpoint, so it must have a template, or the path stops answering 410.
	registered := make(map[string]struct{}, len(removedPathTemplates))
	for _, template := range removedPathTemplates {
		registered[removedPathName(template)] = struct{}{}
	}
	for _, entry := range removednames.Filter(removednames.KindPath, 0) {
		if _, ok := registered[entry.Old]; entry.Context != "" && !ok {
			assert.Fail(t, "removed path has no template in removedPathTemplates", "%s %s", entry.Context, entry.Old)
		}
	}

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

	allMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead,
		http.MethodOptions}
	removedMethodsOf := func(template string) []string {
		if methods := removedPathFor(template, removedPathEntries(template)).Methods; len(methods) > 0 {
			return methods
		}
		return allMethods
	}
	message := func(method, template string) string {
		return fmt.Sprintf(`The "%s %s" endpoint is not available anymore. What to use instead: %s`,
			method, strings.TrimPrefix(removedPathName(template), "/api/_version_/fleet"), removednames.GuideURL)
	}
	// The messages each method can get, for a request to a path that didn't remove it but that another removed path matches.
	messagesByMethod := make(map[string][]string)
	for _, template := range removedPathTemplates {
		for _, method := range removedMethodsOf(template) {
			messagesByMethod[method] = append(messagesByMethod[method], message(method, template))
		}
	}

	for _, template := range removedPathTemplates {
		removedMethods := removedMethodsOf(template)
		for _, version := range []string{"v1", "latest"} {
			// Fill in the path variables with a value that matches any of them.
			path := pathVar.ReplaceAllString(strings.Replace(template, "/_version_/", "/"+version+"/", 1), "1")
			for _, method := range allMethods {
				t.Run(method+" "+path, func(t *testing.T) {
					t.Parallel() // requests with different methods share the route, so -race catches state shared between them
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
					removed := slices.Contains(removedMethods, method)
					if !removed && rec.Code != http.StatusGone {
						// This router holds only the removed paths, so a method the entries didn't remove gets 405, not 410.
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
					if !removed {
						// Another removed path matches and removed this method, such as DELETE /mdm/profiles/{profile_uuid} for
						// DELETE /mdm/profiles/summary.
						assert.Contains(t, messagesByMethod[method], body.Errors[0]["reason"])
						return
					}
					assert.Equal(t, message(method, template), body.Errors[0]["reason"])
				})
			}
		}
	}
}

func TestRemovedPathForEntryWithMethod(t *testing.T) {
	t.Parallel()

	const template = "/api/_version_/fleet/widgets/{name}"
	pinned := removednames.Entry{
		Kind: removednames.KindPath, Scope: removednames.ScopeRequest, Old: template, Context: http.MethodDelete,
		Format: removednames.EndpointFormat{},
	}
	removed := removedPathFor(template, []removednames.Entry{pinned})
	assert.Equal(t, []string{http.MethodDelete}, removed.Methods)
	assert.Equal(t, `The "DELETE /widgets/{name}" endpoint is not available anymore. What to use instead: `+removednames.GuideURL,
		removed.Message(http.MethodDelete))

	pinnedGet := pinned
	pinnedGet.Context = http.MethodGet
	removed = removedPathFor(template, []removednames.Entry{pinned, pinnedGet})
	assert.Equal(t, []string{http.MethodDelete, http.MethodGet}, removed.Methods)
	assert.Equal(t, `The "GET /widgets/{name}" endpoint is not available anymore. What to use instead: `+removednames.GuideURL,
		removed.Message(http.MethodGet))

	unpinned := pinned
	unpinned.Context = ""
	assert.Empty(t, removedPathFor(template, []removednames.Entry{unpinned}).Methods)
}

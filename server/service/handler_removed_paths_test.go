package service

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

// TestRemovedPaths covers every removed path in one place, so removing a path only means adding it to removedPathTemplates.
func TestRemovedPaths(t *testing.T) {
	// Not parallel, like the other tests that build the whole router.
	require.NotEmpty(t, removedPathTemplates)
	// removedPaths panics when a path has no removednames entry.
	require.Len(t, removedPaths(), len(removedPathTemplates))

	for _, alias := range deprecatedPathAliases {
		for _, deprecated := range alias.DeprecatedPaths {
			assert.NotContains(t, removedPathTemplates, deprecated, "a removed path must not also be served as an alias of %s", alias.PrimaryPath)
		}
	}

	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)
	router := h.(interface{ Router() *mux.Router }).Router()

	// Fill in the path variables with a value that matches any of them.
	pathVar := regexp.MustCompile(`\{\w+(:[^}]+)?\}`)
	for _, template := range removedPathTemplates {
		entry, ok := removednames.Find(removednames.KindPath, template, "")
		require.True(t, ok, template)
		for _, version := range []string{"v1", "latest"} {
			path := pathVar.ReplaceAllString(strings.Replace(template, "/_version_/", "/"+version+"/", 1), "1")
			for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
				http.MethodHead, http.MethodOptions} {
				t.Run(method+" "+path, func(t *testing.T) {
					// No authentication: the path is gone for everyone.
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
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
					assert.Equal(t, entry.MessageForVersion(version), body.Errors[0]["reason"])
				})
			}
		}
	}
}

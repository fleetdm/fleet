package endpointer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRemovedPaths(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/fleet/kept", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	RegisterRemovedPaths(r, []string{"v1", "2022-04"}, []RemovedPath{{
		Path:    "/api/_version_/fleet/gone/{id}",
		Message: func(version string) string { return "gone in " + version },
	}})

	for _, c := range []struct {
		name, method, path string
		wantStatus         int
		wantMessage        string
	}{
		{"v1", http.MethodGet, "/api/v1/fleet/gone/1", http.StatusGone, "gone in v1"},
		{"dated version", http.MethodPost, "/api/2022-04/fleet/gone/abc", http.StatusGone, "gone in 2022-04"},
		{"latest", http.MethodDelete, "/api/latest/fleet/gone/1", http.StatusGone, "gone in latest"},
		{"unknown version", http.MethodGet, "/api/v9/fleet/gone/1", http.StatusNotFound, ""},
		{"other path", http.MethodGet, "/api/v1/fleet/kept", http.StatusOK, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
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
}

package maintained_apps

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFmaHTTPClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			w.Header().Set("X-Got-Auth", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Run("without token env var", func(t *testing.T) {
		t.Setenv("FLEET_MAINTAINED_APPS_GITHUB_TOKEN", "")

		cli := fmaHTTPClient()
		require.NotNil(t, cli)

		resp, err := cli.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Empty(t, resp.Header.Get("X-Got-Auth"))
	})

	t.Run("with token env var", func(t *testing.T) {
		t.Setenv("FLEET_MAINTAINED_APPS_GITHUB_TOKEN", "ghp_test123")

		cli := fmaHTTPClient()
		require.NotNil(t, cli)

		resp, err := cli.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, resp.Header.Get("X-Got-Auth"), "ghp_test123")
	})
}

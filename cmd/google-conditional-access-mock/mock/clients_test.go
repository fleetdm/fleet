package mock

import (
	"errors"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google_proxy"
	"github.com/stretchr/testify/require"
)

// TestFleetClients drives the Fleet server's own clients against the mock, so the mock and
// the clients can't drift apart on the wire.
func TestFleetClients(t *testing.T) {
	srv := newTestMock(t, Config{Customers: []string{"C0known"}})

	t.Run("proxy", func(t *testing.T) {
		proxy, err := conditional_access_google_proxy.New(srv.URL, func() (string, error) { return "https://fleet.example.com", nil })
		require.NoError(t, err)

		created, err := proxy.Create(t.Context(), testCustomerID)
		require.NoError(t, err)
		require.Len(t, created.FleetServerSecret, 30)
		require.Equal(t, defaultOAuthClientID, created.OAuthClientID)

		_, err = proxy.Create(t.Context(), testCustomerID)
		se, ok := errors.AsType[*conditional_access_google_proxy.StatusError](err)
		require.True(t, ok, err)
		require.Equal(t, http.StatusConflict, se.StatusCode())

		require.True(t, fleet.IsNotFound(proxy.Delete(t.Context(), testCustomerID, "wrong")))
		require.NoError(t, proxy.Delete(t.Context(), testCustomerID, created.FleetServerSecret))
		require.True(t, fleet.IsNotFound(proxy.Delete(t.Context(), testCustomerID, created.FleetServerSecret)))
	})

	t.Run("google delegation check", func(t *testing.T) {
		resp, keyJSON := do(t, http.MethodGet, srv.URL+"/mock/service-account-key", nil, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, keyJSON)

		c, err := conditional_access_google.NewClient(t.Context(), []byte(keyJSON), "fleet-admin@example.com", "C0known", srv.URL+"/")
		require.NoError(t, err)
		require.NoError(t, c.CheckDelegation(t.Context()))

		c, err = conditional_access_google.NewClient(t.Context(), []byte(keyJSON), "fleet-admin@example.com", "C0unknown", srv.URL+"/")
		require.NoError(t, err)
		require.ErrorContains(t, c.CheckDelegation(t.Context()), "403")
	})
}

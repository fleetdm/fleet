package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	api_http "github.com/fleetdm/fleet/v4/server/mdm/acme/api/http"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/service"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	*testutils.TestDB
	ds     *mysql.Datastore
	server *httptest.Server
}

// setupIntegrationTest creates a new test suite with a real database and HTTP server.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	tdb := testutils.SetupTestDB(t, "acme_integration")
	pool := redistest.SetupRedis(t, "acme_integration", false, false, false)
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)

	// Create mocks
	providers := newMockDataProviders(&fleet.AppConfig{
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://example.com", // will update with actual test server URL after it is started
		},
	})

	// Create service
	svc := service.NewService(ds, pool, providers, tdb.Logger)

	// Create router with routes
	router := mux.NewRouter()
	routesFn := service.GetRoutes(svc)
	routesFn(router, nil)

	// Create test server
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	ac, err := providers.AppConfig(t.Context())
	require.NoError(t, err)
	ac.ServerSettings.ServerURL = server.URL

	return &integrationTestSuite{
		TestDB: tdb,
		ds:     ds,
		server: server,
	}
}

// truncateTables clears all test data between tests.
func (s *integrationTestSuite) truncateTables(t *testing.T) {
	t.Helper()
	s.TruncateTables(t)
}

// newNonce makes an HTTP request to new nonce endpoint and returns the parsed response and the raw response.
func (s *integrationTestSuite) newNonce(t *testing.T, httpMethod, pathIdentifier string) (*api_http.GetNewNonceResponse, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/new_nonce", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(httpMethod, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	result := &api_http.GetNewNonceResponse{
		HTTPMethod: resp.Request.Method,
		Nonce:      resp.Header.Get("Replay-Nonce"),
	}
	return result, resp
}

// getDirectory makes an HTTP request to get directory endpoint and returns the parsed response and the raw response.
func (s *integrationTestSuite) getDirectory(t *testing.T, httpMethod, pathIdentifier string) (*api_http.GetDirectoryResponse, *http.Response) {
	t.Helper()
	url := s.server.URL + fmt.Sprintf("/api/mdm/acme/%s/directory", pathIdentifier) //nolint:gosec // test server URL is safe
	req, err := http.NewRequest(httpMethod, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	if resp.StatusCode != http.StatusOK {
		return nil, resp
	}

	var result api_http.GetDirectoryResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	require.NoError(t, err)
	return &result, resp
}

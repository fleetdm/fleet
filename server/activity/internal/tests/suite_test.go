package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	api_http "github.com/fleetdm/fleet/v4/server/activity/api/http"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	"github.com/fleetdm/fleet/v4/server/activity/internal/testutils"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	*testutils.TestDB
	ds           *mysql.Datastore
	server       *httptest.Server
	userProvider *mockUserProvider
}

// setupIntegrationTest creates a new test suite with a real database and HTTP server.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	tdb := testutils.SetupTestDB(t, "activity_integration")
	ds := mysql.NewDatastore(tdb.Conns(), tdb.Logger)

	// Create mocks
	authorizer := &mockAuthorizer{}
	userProvider := newMockUserProvider()

	// Create service
	svc := service.NewService(authorizer, ds, userProvider, tdb.Logger)

	// Create router with routes
	router := mux.NewRouter()
	// Pass-through auth middleware (authzcheck middleware handles creating the authz context)
	authMiddleware := func(e endpoint.Endpoint) endpoint.Endpoint { return e }
	routesFn := service.GetRoutes(svc, authMiddleware)
	routesFn(router, nil)

	// Create test server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.Close()
	})

	return &integrationTestSuite{
		TestDB:       tdb,
		ds:           ds,
		server:       server,
		userProvider: userProvider,
	}
}

// truncateTables clears all test data between tests.
func (s *integrationTestSuite) truncateTables(t *testing.T) {
	t.Helper()
	s.TruncateTables(t)
}

// insertUser creates a user in the database and mock user provider.
func (s *integrationTestSuite) insertUser(t *testing.T, name, email string) uint {
	t.Helper()
	userID := s.TestDB.InsertUser(t, name, email)

	// Also add to mock user provider for enrichment
	s.userProvider.AddUser(&activity.User{
		ID:    userID,
		Name:  name,
		Email: email,
	})

	return userID
}

// getActivities makes an HTTP request to list activities and returns the parsed response.
func (s *integrationTestSuite) getActivities(t *testing.T, queryParams string) (*api_http.ListActivitiesResponse, int) {
	t.Helper()
	url := s.server.URL + "/api/v1/fleet/activities"
	if queryParams != "" {
		url += "?" + queryParams
	}
	resp, err := http.Get(url) //nolint:gosec // test server URL is safe
	require.NoError(t, err)

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	require.NoError(t, err)

	return &result, resp.StatusCode
}

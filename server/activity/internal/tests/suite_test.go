package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	"github.com/fleetdm/fleet/v4/server/activity"
	api_http "github.com/fleetdm/fleet/v4/server/activity/api/http"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// integrationTestSuite holds all dependencies for integration tests.
type integrationTestSuite struct {
	t            *testing.T
	db           *sqlx.DB
	ds           *mysql.Datastore
	server       *httptest.Server
	userProvider *mockUserProvider
}

// setupIntegrationTest creates a new test suite with a real database and HTTP server.
func setupIntegrationTest(t *testing.T) *integrationTestSuite {
	t.Helper()

	// Use UniqueTestName to avoid fragile runtime.Caller stack depth assumptions
	opts := &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "activity_integration_" + t.Name(),
	}
	testName, opts := mysql_testing_utils.ProcessOptions(t, opts)

	// Load schema
	_, thisFile, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(thisFile), "../../../datastore/mysql/schema.sql")
	mysql_testing_utils.LoadSchema(t, testName, opts, schemaPath)

	// Create DB connection
	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	// Create datastore
	logger := log.NewLogfmtLogger(&testutils.TestLogWriter{T: t})
	conns := &common_mysql.DBConnections{Primary: db, Replica: db}
	ds := mysql.NewDatastore(conns, logger)

	// Create mocks
	authorizer := &mockAuthorizer{}
	userProvider := newMockUserProvider()

	// Create service
	svc := service.NewService(authorizer, ds, userProvider, logger)

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
		t:            t,
		db:           db,
		ds:           ds,
		server:       server,
		userProvider: userProvider,
	}
}

// truncateTables clears all test data between tests.
func (s *integrationTestSuite) truncateTables() {
	mysql_testing_utils.TruncateTables(s.t, s.db, log.NewNopLogger(), nil, "activities", "users")
}

// insertUser creates a user in the database and mock user provider.
func (s *integrationTestSuite) insertUser(name, email string) uint {
	ctx := context.Background()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (name, email, password, salt, created_at, updated_at)
		VALUES (?, ?, 'password', 'salt', NOW(), NOW())
	`, name, email)
	require.NoError(s.t, err)

	id, err := result.LastInsertId()
	require.NoError(s.t, err)

	// Also add to mock user provider for enrichment
	s.userProvider.AddUser(&activity.User{
		ID:    uint(id),
		Name:  name,
		Email: email,
	})

	return uint(id)
}

// insertActivity creates an activity in the database.
func (s *integrationTestSuite) insertActivity(userID uint, activityType string, details map[string]any) uint {
	return s.insertActivityWithTime(userID, activityType, details, time.Now().UTC())
}

// insertActivityWithTime creates an activity in the database with a specific timestamp.
func (s *integrationTestSuite) insertActivityWithTime(userID uint, activityType string, details map[string]any, createdAt time.Time) uint {
	ctx := context.Background()

	detailsJSON, err := json.Marshal(details)
	require.NoError(s.t, err)

	var userName, userEmail *string
	if userID > 0 {
		var user struct {
			Name  string `db:"name"`
			Email string `db:"email"`
		}
		err = sqlx.GetContext(ctx, s.db, &user, "SELECT name, email FROM users WHERE id = ?", userID)
		require.NoError(s.t, err)
		userName = &user.Name
		userEmail = &user.Email
	}

	var result any
	if userID > 0 {
		result, err = s.db.ExecContext(ctx, `
			INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
			VALUES (?, ?, ?, ?, ?, ?, false, false)
		`, userID, userName, userEmail, activityType, detailsJSON, createdAt)
	} else {
		result, err = s.db.ExecContext(ctx, `
			INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
			VALUES (NULL, NULL, NULL, ?, ?, ?, false, false)
		`, activityType, detailsJSON, createdAt)
	}
	require.NoError(s.t, err)

	id, err := result.(interface{ LastInsertId() (int64, error) }).LastInsertId()
	require.NoError(s.t, err)
	return uint(id)
}

// getActivities makes an HTTP request to list activities and returns the parsed response.
// It closes the response body immediately after reading.
func (s *integrationTestSuite) getActivities(t *testing.T, queryParams string) (*api_http.ListActivitiesResponse, int) {
	t.Helper()
	url := s.server.URL + "/api/v1/fleet/activities"
	if queryParams != "" {
		url += "?" + queryParams
	}
	resp, err := http.Get(url)
	require.NoError(t, err)

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	require.NoError(t, err)

	return &result, resp.StatusCode
}

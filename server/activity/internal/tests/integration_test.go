package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	"github.com/fleetdm/fleet/v4/server/activity"
	api_http "github.com/fleetdm/fleet/v4/server/activity/api/http"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/internal/service"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for dependencies outside the bounded context

type mockAuthorizer struct{}

func (m *mockAuthorizer) Authorize(ctx context.Context, subject platform_authz.AuthzTyper, action string) error {
	// Mark authorization as checked (like the real authorizer does)
	if authzCtx, ok := authz_ctx.FromContext(ctx); ok {
		authzCtx.SetChecked()
	}
	return nil // Allow all for integration tests
}

type mockUserProvider struct {
	users map[uint]*activity.User
}

func newMockUserProvider() *mockUserProvider {
	return &mockUserProvider{users: make(map[uint]*activity.User)}
}

func (m *mockUserProvider) AddUser(u *activity.User) {
	m.users[u.ID] = u
}

func (m *mockUserProvider) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	var result []*activity.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockUserProvider) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	return nil, nil // Not used in these tests
}

// Test infrastructure

type integrationTestSuite struct {
	t            *testing.T
	db           *sqlx.DB
	ds           *mysql.Datastore
	server       *httptest.Server
	userProvider *mockUserProvider
}

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

func (s *integrationTestSuite) truncateTables() {
	mysql_testing_utils.TruncateTables(s.t, s.db, log.NewNopLogger(), nil, "activities", "users")
}

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

func (s *integrationTestSuite) insertActivity(userID uint, activityType string, details map[string]any) uint {
	return s.insertActivityWithTime(userID, activityType, details, time.Now().UTC())
}

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

// Integration tests

func TestIntegration(t *testing.T) {
	s := setupIntegrationTest(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *integrationTestSuite)
	}{
		{"ListActivities", testListActivities},
		{"ListActivitiesPagination", testListActivitiesPagination},
		{"ListActivitiesCursorPagination", testListActivitiesCursorPagination},
		{"ListActivitiesFilters", testListActivitiesFilters},
		{"ListActivitiesUserEnrichment", testListActivitiesUserEnrichment},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer s.truncateTables()
			c.fn(t, s)
		})
	}
}

func testListActivities(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser("admin", "admin@example.com")

	// Insert activities
	s.insertActivity(userID, "applied_spec_pack", map[string]any{})
	s.insertActivity(userID, "deleted_pack", map[string]any{})
	s.insertActivity(userID, "edited_pack", map[string]any{})

	// Make HTTP request
	resp, err := http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=100")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 3)
	assert.NotNil(t, result.Meta)

	// Verify order (newest first by default)
	assert.Equal(t, "edited_pack", result.Activities[0].Type)
	assert.Equal(t, "deleted_pack", result.Activities[1].Type)
	assert.Equal(t, "applied_spec_pack", result.Activities[2].Type)
}

func testListActivitiesPagination(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser("admin", "admin@example.com")

	// Insert 5 activities
	for i := range 5 {
		s.insertActivity(userID, "test_activity", map[string]any{"index": i})
	}

	// First page
	resp, err := http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=2&order_key=id&order_direction=asc")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 2)
	assert.True(t, result.Meta.HasNextResults)
	assert.False(t, result.Meta.HasPreviousResults)

	// Second page
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=2&page=1&order_key=id&order_direction=asc")
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 2)
	assert.True(t, result.Meta.HasNextResults)
	assert.True(t, result.Meta.HasPreviousResults)

	// Last page
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=2&page=2&order_key=id&order_direction=asc")
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 1)
	assert.False(t, result.Meta.HasNextResults)
	assert.True(t, result.Meta.HasPreviousResults)
}

func testListActivitiesCursorPagination(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser("admin", "admin@example.com")

	// Insert 3 activities
	s.insertActivity(userID, "applied_spec_pack", map[string]any{})
	s.insertActivity(userID, "deleted_pack", map[string]any{})
	s.insertActivity(userID, "edited_pack", map[string]any{})

	// Test cursor-based pagination with after=0 and table alias in order_key
	// This should return the first activity and Meta should be nil (cursor-based pagination
	// doesn't return metadata)
	resp, err := http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=1&order_key=a.id&after=0")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Should return 1 activity
	assert.Len(t, result.Activities, 1)

	// Meta should be nil for cursor-based pagination
	assert.Nil(t, result.Meta)

	// The activity should be the first one (id > 0, ascending order)
	assert.Equal(t, "applied_spec_pack", result.Activities[0].Type)

	// Test cursor pagination to get the next activity
	firstID := result.Activities[0].ID
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=1&order_key=a.id&after=" + strconv.FormatUint(uint64(firstID), 10))
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 1)
	assert.Nil(t, result.Meta)
	assert.Equal(t, "deleted_pack", result.Activities[0].Type)

	// Test descending order with cursor
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=1&order_key=id&order_direction=desc&after=999999")
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 1)
	assert.Nil(t, result.Meta)
	// Descending order, so the newest (edited_pack) should be first
	assert.Equal(t, "edited_pack", result.Activities[0].Type)
}

func testListActivitiesFilters(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser("john_doe", "john@example.com")
	now := time.Now().UTC().Truncate(time.Second)

	// Insert activities with different types and times
	s.insertActivityWithTime(userID, "type_a", map[string]any{}, now.Add(-48*time.Hour))
	s.insertActivityWithTime(userID, "type_a", map[string]any{}, now.Add(-24*time.Hour))
	s.insertActivityWithTime(userID, "type_b", map[string]any{}, now)

	// Filter by type
	resp, err := http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=100&activity_type=type_a")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 2)
	for _, a := range result.Activities {
		assert.Equal(t, "type_a", a.Type)
	}

	// Filter by date range
	startDate := now.Add(-36 * time.Hour).Format(time.RFC3339)
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=100&start_created_at=" + startDate)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 2) // -24h and now

	// Filter by user search query
	resp, err = http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=100&query=john")
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result.Activities, 3) // All activities by john
}

func testListActivitiesUserEnrichment(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser("John Doe", "john@example.com")

	s.insertActivity(userID, "test_activity", map[string]any{})

	resp, err := http.Get(s.server.URL + "/api/v1/fleet/activities?per_page=100")
	require.NoError(t, err)
	defer resp.Body.Close()

	var result api_http.ListActivitiesResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	require.Len(t, result.Activities, 1)

	// Verify user enrichment from mock user provider
	a := result.Activities[0]
	assert.NotNil(t, a.ActorID)
	assert.Equal(t, userID, *a.ActorID)
	assert.NotNil(t, a.ActorFullName)
	assert.Equal(t, "John Doe", *a.ActorFullName)
	assert.NotNil(t, a.ActorEmail)
	assert.Equal(t, "john@example.com", *a.ActorEmail)
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	api_http "github.com/fleetdm/fleet/v4/server/activity/api/http"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService is a mock implementation of api.Service for handler tests
type mockService struct {
	activities     []*api.Activity
	meta           *api.PaginationMetadata
	err            error
	lastOpt        api.ListOptions
	listCallsCount int
}

func (m *mockService) ListActivities(ctx context.Context, opt api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	// Mark authorization as checked (authzcheck middleware requires this)
	if authzCtx, ok := authz_ctx.FromContext(ctx); ok {
		authzCtx.SetChecked()
	}

	m.listCallsCount++
	m.lastOpt = opt
	return m.activities, m.meta, m.err
}

func setupTestRouter(svc api.Service) *mux.Router {
	r := mux.NewRouter()

	// Create a pass-through auth middleware for testing
	authMiddleware := func(e endpoint.Endpoint) endpoint.Endpoint { return e }

	routesFn := GetRoutes(svc, authMiddleware)
	routesFn(r, nil)

	return r
}

func TestHandlerListActivities(t *testing.T) {
	cases := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"Basic", testHandlerListActivitiesBasic},
		{"QueryParams", testHandlerListActivitiesQueryParams},
		{"ServiceError", testHandlerListActivitiesServiceError},
	}
	for _, c := range cases {
		t.Run(c.name, c.fn)
	}
}

func testHandlerListActivitiesBasic(t *testing.T) {
	details := json.RawMessage(`{"key": "value"}`)
	mockSvc := &mockService{
		activities: []*api.Activity{
			{ID: 1, Type: "test_activity", Details: &details},
			{ID: 2, Type: "another_activity"},
		},
		meta: &api.PaginationMetadata{HasNextResults: true},
	}

	r := setupTestRouter(mockSvc)

	req := httptest.NewRequest("GET", "/api/v1/fleet/activities", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response api_http.ListActivitiesResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response.Activities, 2)
	assert.NotNil(t, response.Meta)
	assert.True(t, response.Meta.HasNextResults)

	// Verify defaults were applied
	assert.Equal(t, "created_at", mockSvc.lastOpt.OrderKey)
	assert.Equal(t, api.OrderDesc, mockSvc.lastOpt.OrderDirection)
}

func testHandlerListActivitiesQueryParams(t *testing.T) {
	mockSvc := &mockService{
		activities: []*api.Activity{},
		meta:       &api.PaginationMetadata{},
	}

	r := setupTestRouter(mockSvc)

	req := httptest.NewRequest("GET", "/api/v1/fleet/activities?query=john&activity_type=mdm_enrolled&start_created_at=2024-01-01T00:00:00Z&end_created_at=2024-12-31T23:59:59Z&page=2&per_page=25&order_key=id&order_direction=asc", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify filter params
	assert.Equal(t, "john", mockSvc.lastOpt.MatchQuery)
	assert.Equal(t, "mdm_enrolled", mockSvc.lastOpt.ActivityType)
	assert.Equal(t, "2024-01-01T00:00:00Z", mockSvc.lastOpt.StartCreatedAt)
	assert.Equal(t, "2024-12-31T23:59:59Z", mockSvc.lastOpt.EndCreatedAt)

	// Verify pagination params
	assert.Equal(t, uint(2), mockSvc.lastOpt.Page)
	assert.Equal(t, uint(25), mockSvc.lastOpt.PerPage)
	assert.Equal(t, "id", mockSvc.lastOpt.OrderKey)
	assert.Equal(t, api.OrderAsc, mockSvc.lastOpt.OrderDirection)
}

func testHandlerListActivitiesServiceError(t *testing.T) {
	mockSvc := &mockService{
		err: errors.New("service error"),
	}

	r := setupTestRouter(mockSvc)

	req := httptest.NewRequest("GET", "/api/v1/fleet/activities", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Generic errors return 500 status with JSON error body
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]any
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// The response has an "errors" field (from the error encoder)
	assert.NotNil(t, response["errors"])
}

func TestHandlerAPIVersions(t *testing.T) {
	mockSvc := &mockService{
		activities: []*api.Activity{},
		meta:       &api.PaginationMetadata{},
	}

	r := setupTestRouter(mockSvc)

	// Test v1 endpoint
	req := httptest.NewRequest("GET", "/api/v1/fleet/activities", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Test latest endpoint
	req = httptest.NewRequest("GET", "/api/latest/fleet/activities", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

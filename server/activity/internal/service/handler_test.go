package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	platform_endpointer "github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListActivitiesValidation(t *testing.T) {
	t.Parallel()
	// These tests verify decoder validation logic that returns 400 Bad Request.
	// Happy path and business logic are covered by integration tests.

	cases := []struct {
		name    string
		query   string
		wantErr string
	}{
		{
			name:    "non-integer page",
			query:   "page=abc",
			wantErr: "non-int page value",
		},
		{
			name:    "negative page",
			query:   "page=-1",
			wantErr: "negative page value",
		},
		{
			name:    "non-integer per_page",
			query:   "per_page=abc",
			wantErr: "non-int per_page value",
		},
		{
			name:    "zero per_page",
			query:   "per_page=0",
			wantErr: "invalid per_page value",
		},
		{
			name:    "negative per_page",
			query:   "per_page=-5",
			wantErr: "invalid per_page value",
		},
		{
			name:    "order_direction without order_key",
			query:   "order_direction=desc",
			wantErr: "order_key must be specified with order_direction",
		},
		{
			name:    "invalid order_direction",
			query:   "order_key=id&order_direction=invalid",
			wantErr: "unknown order_direction: invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupTestRouter()

			req := httptest.NewRequest("GET", "/api/v1/fleet/activities?"+tc.query, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)

			var response struct {
				Message string `json:"message"`
				Errors  []struct {
					Name   string `json:"name"`
					Reason string `json:"reason"`
				} `json:"errors"`
			}
			err := json.NewDecoder(rr.Body).Decode(&response)
			require.NoError(t, err)

			// Check that the error message is in the errors array
			require.Len(t, response.Errors, 1)
			assert.Equal(t, "base", response.Errors[0].Name)
			assert.Equal(t, tc.wantErr, response.Errors[0].Reason)
		})
	}
}

// errorEncoder wraps platform_endpointer.EncodeError for use in tests.
func errorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	platform_endpointer.EncodeError(ctx, err, w, nil)
}

func setupTestRouter() *mux.Router {
	r := mux.NewRouter()

	// Mock service that should never be called (validation fails before reaching service)
	mockSvc := &mockService{}

	// Pass-through auth middleware
	authMiddleware := func(e endpoint.Endpoint) endpoint.Endpoint { return e }

	// Server options with proper error encoding
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(errorEncoder),
	}

	routesFn := GetRoutes(mockSvc, authMiddleware)
	routesFn(r, opts)

	return r
}

// mockService implements api.Service for handler tests.
// For validation tests, this should never be called.
type mockService struct{}

func (m *mockService) ListActivities(_ context.Context, _ api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	panic("mockService.ListActivities should not be called in validation tests")
}

func (m *mockService) ListHostPastActivities(_ context.Context, _ uint, _ api.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	panic("mockService.ListHostPastActivities should not be called in validation tests")
}

func (m *mockService) MarkActivitiesAsStreamed(_ context.Context, _ []uint) error {
	panic("mockService.MarkActivitiesAsStreamed should not be called in validation tests")
}

func (m *mockService) StreamActivities(_ context.Context, _ api.JSONLogger, _ uint) error {
	panic("mockService.StreamActivities should not be called in validation tests")
}

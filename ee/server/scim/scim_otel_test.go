package scim

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSCIMOTELMiddleware(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		path         string
		method       string
		expectedSpan string
	}{
		{
			name:         "Users list",
			path:         "Users",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Users",
		},
		{
			name:         "Users list with trailing slash",
			path:         "Users/",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Users",
		},
		{
			name:         "Individual user - hides ID",
			path:         "Users/12345",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Users/{id}",
		},
		{
			name:         "Update user - hides ID",
			path:         "Users/67890",
			method:       "PATCH",
			expectedSpan: "PATCH /api/v1/fleet/scim/Users/{id}",
		},
		{
			name:         "Groups list",
			path:         "Groups",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Groups",
		},
		{
			name:         "Individual group - hides ID",
			path:         "Groups/abc-def-123",
			method:       "PUT",
			expectedSpan: "PUT /api/v1/fleet/scim/Groups/{id}",
		},
		{
			name:         "Schemas",
			path:         "Schemas",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Schemas",
		},
		{
			name:         "Individual schema",
			path:         "Schemas/urn:ietf:params:scim:schemas:core:2.0:User",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/Schemas/{id}",
		},
		{
			name:         "Service provider config",
			path:         "ServiceProviderConfig",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/ServiceProviderConfig",
		},
		{
			name:         "Resource types",
			path:         "ResourceTypes",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/ResourceTypes",
		},
		{
			name:         "Unknown path - uses full path",
			path:         "SomethingElse",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/SomethingElse",
		},
		{
			name:         "Bulk operations endpoint",
			path:         "Bulk",
			method:       "POST",
			expectedSpan: "POST /api/v1/fleet/scim/Bulk",
		},
		{
			name:         "Search endpoint",
			path:         ".search",
			method:       "POST",
			expectedSpan: "POST /api/v1/fleet/scim/.search",
		},
		{
			name:         "Unknown resource with ID - hides ID",
			path:         "CustomResource/abc123",
			method:       "GET",
			expectedSpan: "GET /api/v1/fleet/scim/CustomResource/{id}",
		},
		{
			name:         "Unknown nested path with ID - hides ID",
			path:         "Custom/Resource/123",
			method:       "DELETE",
			expectedSpan: "DELETE /api/v1/fleet/scim/Custom/{id}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test span recorder
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			// Create test configuration with OTEL enabled
			cfg := config.FleetConfig{
				Logging: config.LoggingConfig{
					TracingEnabled: true,
					TracingType:    "opentelemetry",
				},
			}

			// Create a test handler that just returns 200 OK
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with SCIM OTEL middleware
			tracer := tp.Tracer("test")
			wrappedHandler := scimOTELMiddleware(testHandler, "/api/v1/fleet/scim", cfg)

			// Create request - now OTEL runs before StripPrefix so it sees the full path
			req := httptest.NewRequest(tc.method, "/api/v1/fleet/scim/"+tc.path, nil)

			// Add span to context
			ctx, span := tracer.Start(req.Context(), "test-parent-span")
			defer span.End()
			req = req.WithContext(ctx)

			// Execute request
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Force span to end
			span.End()

			// Check spans
			spans := sr.Ended()
			require.GreaterOrEqual(t, len(spans), 2, "Should have at least parent and child spans")

			// Find the SCIM span (should be the second one, after the parent)
			var scimSpan trace.ReadOnlySpan
			for _, s := range spans {
				if s.Name() == tc.expectedSpan {
					scimSpan = s
					break
				}
			}

			require.NotNil(t, scimSpan, "Should find SCIM span with name: %s", tc.expectedSpan)
			assert.Equal(t, tc.expectedSpan, scimSpan.Name())

			// Check that the route tag is set correctly (without exposing IDs)
			attrs := scimSpan.Attributes()
			for _, attr := range attrs {
				if string(attr.Key) == "http.route" {
					// The route should match the pattern, not the actual path
					assert.NotContains(t, attr.Value.AsString(), "123", "Should not expose user ID")
					assert.NotContains(t, attr.Value.AsString(), "67890", "Should not expose user ID")
				}
			}
		})
	}
}

func TestSCIMOTELMiddleware_Disabled(t *testing.T) {
	t.Parallel()
	// Create test configuration with OTEL disabled
	cfg := config.FleetConfig{
		Logging: config.LoggingConfig{
			TracingEnabled: false,
		},
	}

	// Create a test handler
	called := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with SCIM OTEL middleware
	wrappedHandler := scimOTELMiddleware(testHandler, "/api/v1/fleet/scim", cfg)

	// Create request - OTEL sees the full path now
	req := httptest.NewRequest("GET", "/api/v1/fleet/scim/Users", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Should have called the handler without any OTEL instrumentation
	assert.True(t, called, "Handler should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
}

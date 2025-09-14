package ctxerr

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHandleSendsContextToOTEL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		setupContext  func(context.Context) context.Context
		errorMessage  string
		expectedAttrs map[string]any // expected attributes in the exception event
	}{
		{
			name: "with user context",
			setupContext: func(ctx context.Context) context.Context {
				testUser := &fleet.User{
					ID:    123,
					Email: "test@example.com",
				}
				return viewer.NewContext(ctx, viewer.Viewer{User: testUser})
			},
			errorMessage: "test error with user context",
			expectedAttrs: map[string]any{
				"user.id": int64(123),
			},
		},
		{
			name: "with host context",
			setupContext: func(ctx context.Context) context.Context {
				testHost := &fleet.Host{
					ID:       456,
					Hostname: "test-host.example.com",
				}
				return host.NewContext(ctx, testHost)
			},
			errorMessage: "test error with host context",
			expectedAttrs: map[string]any{
				"host.hostname": "test-host.example.com",
				"host.id":       int64(456),
			},
		},
		{
			name: "without additional context",
			setupContext: func(ctx context.Context) context.Context {
				return ctx // no additional context
			},
			errorMessage:  "test error without context",
			expectedAttrs: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test span recorder and tracer provider
			sr := tracetest.NewSpanRecorder()
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

			// Create a context with an active span using the test tracer
			tracer := tp.Tracer("test")
			ctx, span := tracer.Start(t.Context(), "test-span")
			defer span.End()

			// Setup context with test-specific data
			ctx = tc.setupContext(ctx)

			// Create and handle an error
			err := New(ctx, tc.errorMessage)
			Handle(ctx, err)

			// Force span to end so we can check recorded data
			span.End()

			// Check that the exception event was created
			spans := sr.Ended()
			require.Len(t, spans, 1)

			// Find the exception event
			events := spans[0].Events()
			var exceptionEvent *trace.Event
			for i := range events {
				if events[i].Name == "exception" {
					exceptionEvent = &events[i]
					break
				}
			}
			require.NotNil(t, exceptionEvent, "Expected to find an exception event")

			// Check all expected attributes are present
			attributes := make(map[string]any)
			for _, attr := range exceptionEvent.Attributes {
				switch attr.Key {
				case "user.id", "host.id":
					attributes[string(attr.Key)] = attr.Value.AsInt64()
				default:
					attributes[string(attr.Key)] = attr.Value.AsString()
				}
			}

			// Always check for stack trace
			stackTrace, ok := attributes["exception.stacktrace"].(string)
			assert.True(t, ok, "Expected exception.stacktrace attribute")
			assert.Contains(t, stackTrace, "TestHandleSendsContextToOTEL", "Stack trace should contain test function name")
			assert.True(t, strings.Contains(stackTrace, "\n"), "Stack trace should be formatted with newlines")

			// Check for exception message and type
			assert.Equal(t, tc.errorMessage, attributes["exception.message"])
			assert.Equal(t, "*ctxerr.FleetError", attributes["exception.type"])

			// Check test-specific expected attributes
			for expectedKey, expectedValue := range tc.expectedAttrs {
				actualValue, found := attributes[expectedKey]
				assert.True(t, found, "Expected to find attribute %s", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Attribute %s should match", expectedKey)
			}
		})
	}
}

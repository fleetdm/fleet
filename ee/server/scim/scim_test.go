package scim

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// closeTrackingBody wraps an io.Reader and records whether Close has been called.
// It is used to verify that debugPayloadDumpMiddleware closes the original request
// body after consuming it for logging.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (c *closeTrackingBody) Close() error {
	c.closed = true
	return nil
}

func TestDebugPayloadDumpMiddleware(t *testing.T) {
	const samplePayload = `{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[]}`

	t.Run("disabled returns next handler unchanged", func(t *testing.T) {
		var downstreamBody []byte
		var downstreamReadErr error
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamBody, downstreamReadErr = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, false, next)

		// Lock in the zero-overhead promise: when disabled the middleware must return
		// the next handler unchanged, not a wrapped one.
		assert.Equal(t,
			reflect.ValueOf(next).Pointer(),
			reflect.ValueOf(handler).Pointer(),
			"disabled middleware must return next handler unchanged")

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/fleet/scim/Users/1", strings.NewReader(samplePayload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.NoError(t, downstreamReadErr)
		assert.True(t, bytes.Equal([]byte(samplePayload), downstreamBody),
			"downstream handler must see the original body byte-for-byte")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, buf.String(), "no logs should be emitted when middleware is disabled")
	})

	t.Run("enabled logs body and downstream still receives it", func(t *testing.T) {
		var downstreamBody []byte
		var downstreamReadErr error
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamBody, downstreamReadErr = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/fleet/scim/Users/1", strings.NewReader(samplePayload))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.NoError(t, downstreamReadErr)
		assert.True(t, bytes.Equal([]byte(samplePayload), downstreamBody),
			"downstream handler must see the original body byte-for-byte after dump")
		assert.Equal(t, http.StatusOK, rec.Code)

		out := buf.String()
		assert.Contains(t, out, `msg="scim payload dump"`)
		assert.Contains(t, out, "method=PATCH")
		assert.Contains(t, out, "/api/v1/fleet/scim/Users/1")
		// slog's text handler wraps the body in quotes and backslash-escapes inner quotes,
		// so look for distinctive unquoted tokens instead of the raw payload string.
		assert.Contains(t, out, "PatchOp")
		assert.Contains(t, out, "Operations")
	})

	t.Run("enabled with empty body logs without erroring", func(t *testing.T) {
		var downstreamCalled bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamCalled = true
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/fleet/scim/Users", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.True(t, downstreamCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		out := buf.String()
		assert.Contains(t, out, `msg="scim payload dump"`)
		assert.Contains(t, out, "method=GET")
	})

	t.Run("enabled closes the original request body", func(t *testing.T) {
		// The middleware must close the original body after reading it for the log,
		// otherwise the underlying connection / body resource leaks.
		tracker := &closeTrackingBody{Reader: strings.NewReader(samplePayload)}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/scim/Users", nil)
		req.Body = tracker
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, tracker.closed, "original body Close must be called after the middleware reads it")
	})

	t.Run("enabled logs full body even for large payloads", func(t *testing.T) {
		// The middleware reads the entire body and writes it to the log unchanged.
		// Use a marker at the tail to prove we logged past any small head buffer.
		const tailMarker = "TAIL_MARKER_PRESENT_IN_LOG"
		large := strings.Repeat("A", 100*1024) + tailMarker

		var downstreamBody []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/scim/Users", strings.NewReader(large))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, downstreamBody, len(large), "downstream must see the full body")

		out := buf.String()
		assert.Contains(t, out, tailMarker, "log must include the full body, including bytes past any head buffer")
	})
}

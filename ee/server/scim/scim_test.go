package scim

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// closeTrackingBody wraps an io.Reader and records whether Close has been called.
// It is used to verify that debugPayloadDumpMiddleware propagates Close() to the
// original request body even when it stitches the body via io.MultiReader.
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

	t.Run("enabled truncates large body in log but passes full body downstream", func(t *testing.T) {
		// Append a recognizable tail marker after the truncation cap so we can prove the
		// log doesn't include it (truncation actually happened) while the downstream
		// handler still sees the full body including the marker (passthrough preserved).
		const tailMarker = "TAIL_MARKER_NOT_IN_LOG_XYZ"
		large := strings.Repeat("A", maxDumpedSCIMBodyBytes+1024) + tailMarker

		var downstreamBody []byte
		var downstreamReadErr error
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamBody, downstreamReadErr = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/scim/Users", strings.NewReader(large))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.NoError(t, downstreamReadErr)
		assert.Len(t, downstreamBody, len(large), "downstream must still see the full body length")
		assert.Contains(t, string(downstreamBody), tailMarker, "downstream must receive bytes beyond the log cap")
		assert.Equal(t, http.StatusOK, rec.Code)

		out := buf.String()
		assert.Contains(t, out, "size="+strconv.Itoa(maxDumpedSCIMBodyBytes), "log should report the capped size")
		assert.Contains(t, out, "truncated=true", "log should mark the entry as truncated")
		assert.NotContains(t, out, tailMarker, "log must not contain bytes past the cap")
	})

	t.Run("enabled truncated path closes the original body", func(t *testing.T) {
		// Regression: the truncated branch wraps the stitched body via a struct that must
		// propagate Close() to the original http.Request.Body. Previously it used
		// io.NopCloser, which leaked the underlying body/connection.
		body := strings.Repeat("A", maxDumpedSCIMBodyBytes+10)
		tracker := &closeTrackingBody{Reader: strings.NewReader(body)}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.ReadAll(r.Body)
			// The http.Server normally calls Close after the handler returns; emulate that here
			// so we can assert close propagation works end-to-end through our wrapper.
			_ = r.Body.Close()
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
		assert.True(t, tracker.closed, "original body Close must be called when path is truncated")
	})

	t.Run("enabled body equal to cap is not marked truncated", func(t *testing.T) {
		body := strings.Repeat("A", maxDumpedSCIMBodyBytes)

		var downstreamBody []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			downstreamBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := debugPayloadDumpMiddleware(logger, true, next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/fleet/scim/Users", strings.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, downstreamBody, len(body))
		out := buf.String()
		assert.Contains(t, out, "size="+strconv.Itoa(maxDumpedSCIMBodyBytes))
		assert.Contains(t, out, "truncated=false")
	})
}

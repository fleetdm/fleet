package endpointer

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHandlerFunc is a handler function type used for testing.
type testHandlerFunc func(ctx context.Context, request any) (platform_http.Errorer, error)

func TestCustomMiddlewareAfterAuth(t *testing.T) {
	var (
		i                = 0
		beforeIndex      = 0
		authIndex        = 0
		afterFirstIndex  = 0
		afterSecondIndex = 0
	)
	beforeAuthMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			i++
			beforeIndex = i
			return next(ctx, req)
		}
	}

	authMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			i++
			authIndex = i
			if authctx, ok := authz_ctx.FromContext(ctx); ok {
				authctx.SetChecked()
			}
			return next(ctx, req)
		}
	}

	afterAuthMiddlewareFirst := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			i++
			afterFirstIndex = i
			return next(ctx, req)
		}
	}
	afterAuthMiddlewareSecond := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			i++
			afterSecondIndex = i
			return next(ctx, req)
		}
	}

	r := mux.NewRouter()
	ce := &CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (request any, err error) {
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware: authMiddleware,
		CustomMiddleware: []endpoint.Middleware{
			beforeAuthMiddleware,
		},
		CustomMiddlewareAfterAuth: []endpoint.Middleware{
			afterAuthMiddlewareFirst,
			afterAuthMiddlewareSecond,
		},
		Router: r,
	}
	ce.handleEndpoint("/", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		fmt.Printf("handler\n")
		return nopResponse{}, nil
	}, nil, "GET")

	s := httptest.NewServer(r)
	t.Cleanup(func() {
		s.Close()
	})

	req, err := http.NewRequest("GET", s.URL+"/", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		resp.Body.Close()
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, beforeIndex)
	require.Equal(t, 2, authIndex)
	require.Equal(t, 3, afterFirstIndex)
	require.Equal(t, 4, afterSecondIndex)
}

type nopRequest struct{}

type nopResponse struct{}

func (n nopResponse) Error() error {
	return nil
}

type nopEP struct{}

func (n nopEP) CallHandlerFunc(f testHandlerFunc, ctx context.Context, request any, svc any) (platform_http.Errorer, error) {
	return f(ctx, request)
}

func (n nopEP) Service() any {
	return nil
}

// TestHTTPPreAuthMiddlewareRunsBeforeDecode asserts that HTTPPreAuthMiddleware
// short-circuits the request before the body decoder is invoked.
func TestHTTPPreAuthMiddlewareRunsBeforeDecode(t *testing.T) {
	var decodeCalled bool
	var authCalled bool

	authMw := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			authCalled = true
			return next(ctx, req)
		}
	}

	r := mux.NewRouter()
	ce := (&CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (any, error) {
				decodeCalled = true
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware: authMw,
		Router:         r,
	}).WithHTTPPreAuth(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Reject without calling next — decoder and auth must not run.
			w.WriteHeader(http.StatusUnauthorized)
		})
	})

	ce.handleEndpoint("/", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		return nopResponse{}, nil
	}, nil, "POST")

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/", "application/json", strings.NewReader(`{"x":1}`))
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.False(t, decodeCalled, "decoder must not run when pre-auth rejects")
	assert.False(t, authCalled, "auth middleware must not run when pre-auth rejects")
}

// TestHTTPPreAuthMiddlewarePassThrough asserts that when the pre-auth
// middleware calls next, the decoder and auth chain run as normal.
func TestHTTPPreAuthMiddlewarePassThrough(t *testing.T) {
	var decodeCalled bool
	var authCalled bool

	authMw := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			authCalled = true
			if authctx, ok := authz_ctx.FromContext(ctx); ok {
				authctx.SetChecked()
			}
			return next(ctx, req)
		}
	}

	r := mux.NewRouter()
	ce := (&CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (any, error) {
				decodeCalled = true
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware: authMw,
		Router:         r,
	}).WithHTTPPreAuth(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	ce.handleEndpoint("/", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		return nopResponse{}, nil
	}, nil, "POST")

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/", "application/json", strings.NewReader(`{"x":1}`))
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, decodeCalled, "decoder must run when pre-auth passes through")
	assert.True(t, authCalled, "auth middleware must run when pre-auth passes through")
}

func TestRegisterDeprecatedPathAliases(t *testing.T) {
	// Set up a router and register a primary endpoint via CommonEndpointer.
	r := mux.NewRouter()
	registry := NewHandlerRegistry()
	versions := []string{"v1", "2022-04"}

	authMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			if authctx, ok := authz_ctx.FromContext(ctx); ok {
				authctx.SetChecked()
			}
			return next(ctx, req)
		}
	}

	ce := &CommonEndpointer[testHandlerFunc]{
		EP: nopEP{},
		MakeDecoderFn: func(iface any, requestBodySizeLimit int64) kithttp.DecodeRequestFunc {
			return func(ctx context.Context, r *http.Request) (request any, err error) {
				return nopRequest{}, nil
			}
		},
		EncodeFn: func(ctx context.Context, w http.ResponseWriter, i any) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
		AuthMiddleware:  authMiddleware,
		Router:          r,
		Versions:        versions,
		HandlerRegistry: registry,
	}

	// Register the primary endpoint.
	ce.GET("/api/_version_/fleet/fleets", func(ctx context.Context, request any) (platform_http.Errorer, error) {
		return nopResponse{}, nil
	}, nil)

	// Register a deprecated alias for it.
	RegisterDeprecatedPathAliases(r, versions, registry, []DeprecatedPathAlias{
		{
			Method:          "GET",
			PrimaryPath:     "/api/_version_/fleet/fleets",
			DeprecatedPaths: []string{"/api/_version_/fleet/teams"},
		},
	})

	s := httptest.NewServer(r)
	t.Cleanup(s.Close)

	// Both the primary and deprecated paths should return 200.
	for _, path := range []string{"/api/v1/fleet/fleets", "/api/v1/fleet/teams", "/api/latest/fleet/teams"} {
		resp, err := http.Get(s.URL + path)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "path %s should return 200", path)
	}
}

func TestLogDeprecatedPathAlias(t *testing.T) {
	// Without deprecated path info in context, LogDeprecatedPathAlias is a no-op.
	lc := &logging.LoggingContext{}
	ctx := logging.NewContext(context.Background(), lc)
	ctx2 := LogDeprecatedPathAlias(ctx, nil)
	require.Equal(t, ctx, ctx2, "should return same context when no deprecated path info")
	require.Empty(t, lc.Extras)

	// With deprecated path info, it should set warn level and extras.
	ctx = context.WithValue(ctx, deprecatedPathInfoKey{}, deprecatedPathInfo{
		deprecatedPath: "/api/_version_/fleet/teams",
		primaryPath:    "/api/_version_/fleet/fleets",
	})
	LogDeprecatedPathAlias(ctx, nil)

	// Extras is a flat []interface{} of key-value pairs.
	require.Len(t, lc.Extras, 4) // "deprecated_path", value, "deprecation_warning", value
	require.Equal(t, "deprecated_path", lc.Extras[0])
	require.Equal(t, "/api/_version_/fleet/teams", lc.Extras[1])
	require.Equal(t, "deprecation_warning", lc.Extras[2])
	require.Contains(t, lc.Extras[3], "deprecated")

	// ForceLevel should be set to Warn.
	require.NotNil(t, lc.ForceLevel)
	require.Equal(t, slog.LevelWarn, *lc.ForceLevel)
}

func TestRegisterDeprecatedPathAliasesPanicsOnMissing(t *testing.T) {
	r := mux.NewRouter()
	registry := NewHandlerRegistry()
	versions := []string{"v1"}

	require.Panics(t, func() {
		RegisterDeprecatedPathAliases(r, versions, registry, []DeprecatedPathAlias{
			{
				Method:          "GET",
				PrimaryPath:     "/api/_version_/fleet/nonexistent",
				DeprecatedPaths: []string{"/api/_version_/fleet/old"},
			},
		})
	})
}

func defaultJSONUnmarshal(body io.Reader, req any) error {
	return json.NewDecoder(body).Decode(req)
}

type testRequestDecoderType struct {
	Data string `json:"data"`
}

func (d *testRequestDecoderType) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	err := json.NewDecoder(r.Body).Decode(d)
	return d, err
}

// TestMakeDecoderRequestDecoderFalsePositive verifies that a body containing
// malformed JSON that is within the size limit does not produce a
// PayloadTooLargeError (false positive)
func TestMakeDecoderRequestDecoderFalsePositive(t *testing.T) {
	const limit = 50

	makeDecoder := func(limit int64) kithttp.DecodeRequestFunc {
		return MakeDecoder(&testRequestDecoderType{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)
	}

	t.Run("malformed JSON within limit returns decode error, not 413", func(t *testing.T) {
		body := strings.NewReader(`{"data": "truncated`) // malformed, within limit
		r := httptest.NewRequest("POST", "/", body)
		_, err := makeDecoder(limit)(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.False(t, errors.As(err, &ple), "malformed body within limit must not produce PayloadTooLargeError")
	})

	t.Run("body over limit returns 413", func(t *testing.T) {
		big := `{"data":"` + strings.Repeat("x", limit+10) + `"}`
		body := strings.NewReader(big)
		r := httptest.NewRequest("POST", "/", body)
		_, err := makeDecoder(limit)(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "body over limit must produce PayloadTooLargeError, got: %v", err)
	})

	t.Run("malformed JSON exactly at limit returns decode error, not 413", func(t *testing.T) {
		// Build a body of exactly `limit` bytes that is malformed JSON (no closing
		// brace). The MaxBytesReader allows the full read (body fits within limit),
		// so the JSON decoder sees io.ErrUnexpectedEOF — not *http.MaxBytesError.
		// Must not produce PayloadTooLargeError.
		prefix := `{"data":"`
		body := strings.NewReader(prefix + strings.Repeat("x", limit-len(prefix))) // exactly limit bytes, no closing
		r := httptest.NewRequest("POST", "/", body)
		_, err := makeDecoder(limit)(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.False(t, errors.As(err, &ple), "malformed body exactly at limit must not produce PayloadTooLargeError")
	})

	t.Run("body over limit without Content-Length returns 413", func(t *testing.T) {
		// Simulate a chunked request (no Content-Length) whose body exceeds the
		// limit. The peek at the underlying reader finds more data → 413.
		big := `{"data":"` + strings.Repeat("x", limit+10) + `"}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(big))
		r.ContentLength = -1 // strip the Content-Length that httptest set
		_, err := makeDecoder(limit)(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "over-limit body without Content-Length must produce PayloadTooLargeError, got: %v", err)
	})

	t.Run("valid body within limit is decoded successfully", func(t *testing.T) {
		body := strings.NewReader(`{"data":"hello"}`)
		r := httptest.NewRequest("POST", "/", body)
		result, err := makeDecoder(limit)(context.Background(), r)
		require.NoError(t, err)
		rd, ok := result.(*testRequestDecoderType)
		require.True(t, ok)
		assert.Equal(t, "hello", rd.Data)
	})
}

func TestMakeDecoderRequestDecoderGzipBomb(t *testing.T) {
	const limit = 100

	makeDecoder := func() kithttp.DecodeRequestFunc {
		return MakeDecoder(&testRequestDecoderType{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)
	}

	gzipBody := func(data string) *bytes.Buffer {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err := gw.Write([]byte(data))
		require.NoError(t, err)
		require.NoError(t, gw.Close())
		return &buf
	}

	t.Run("gzip bomb exceeding decompressed limit returns 413", func(t *testing.T) {
		big := `{"data":"` + strings.Repeat("x", limit*10) + `"}`
		r := httptest.NewRequest("POST", "/", gzipBody(big))
		r.Header.Set("Content-Encoding", "gzip")
		_, err := makeDecoder()(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "gzip bomb via RequestDecoder must produce PayloadTooLargeError, got: %v", err)
		assert.True(t, ple.Gzipped, "PayloadTooLargeError from gzip bomb must have Gzipped set")
	})

	t.Run("valid gzip body within limit is decoded successfully", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", gzipBody(`{"data":"hi"}`))
		r.Header.Set("Content-Encoding", "gzip")
		result, err := makeDecoder()(context.Background(), r)
		require.NoError(t, err)
		rd, ok := result.(*testRequestDecoderType)
		require.True(t, ok)
		assert.Equal(t, "hi", rd.Data)
	})

	t.Run("Content-Encoding header is cleared after decompression", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", gzipBody(`{"data":"hi"}`))
		r.Header.Set("Content-Encoding", "gzip")
		_, err := makeDecoder()(context.Background(), r)
		require.NoError(t, err)
		assert.Empty(t, r.Header.Get("Content-Encoding"), "Content-Encoding should be cleared after framework decompression")
	})
}

type testGzipRequestType struct {
	Data string `json:"data"`
}

// testRequestDecoderPayloadTooLargeType implements RequestDecoder and returns
// a PayloadTooLargeError directly from DecodeRequest (simulating implementations
// like getHostSoftwareRequest that enforce their own size limits).
type testRequestDecoderPayloadTooLargeType struct {
	Data string `json:"data"`
}

func (d *testRequestDecoderPayloadTooLargeType) DecodeRequest(_ context.Context, r *http.Request) (any, error) {
	return nil, platform_http.PayloadTooLargeError{
		ContentLength:  r.Header.Get("Content-Length"),
		MaxRequestSize: 42,
	}
}

type testGzipBodyDecoderType struct {
	Data string `json:"data"`
}

func TestMakeDecoderGzipBomb(t *testing.T) {
	const limit = 100

	makeDecoder := func() kithttp.DecodeRequestFunc {
		return MakeDecoder(testGzipRequestType{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)
	}

	gzipBody := func(data string) *bytes.Buffer {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err := gw.Write([]byte(data))
		require.NoError(t, err)
		require.NoError(t, gw.Close())
		return &buf
	}

	t.Run("gzip bomb exceeding decompressed limit returns 413", func(t *testing.T) {
		// Compressed payload is small but decompresses well beyond the limit.
		big := `{"data":"` + strings.Repeat("x", limit*10) + `"}`
		r := httptest.NewRequest("POST", "/", gzipBody(big))
		r.Header.Set("Content-Encoding", "gzip")
		_, err := makeDecoder()(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "gzip bomb must produce PayloadTooLargeError, got: %v", err)
		assert.True(t, ple.Gzipped, "PayloadTooLargeError from gzip bomb must have Gzipped set")
	})

	t.Run("valid gzip body within limit is decoded successfully", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", gzipBody(`{"data":"hi"}`))
		r.Header.Set("Content-Encoding", "gzip")
		result, err := makeDecoder()(context.Background(), r)
		require.NoError(t, err)
		rd, ok := result.(*testGzipRequestType)
		require.True(t, ok)
		assert.Equal(t, "hi", rd.Data)
	})

	// Sub-tests for the bodyDecoder (DecodeBody) code path, where isBodyDecoder
	// returns true and decodeBody is called instead of jsonUnmarshal.
	isBodyDecoder := func(v reflect.Value) bool {
		_, ok := v.Interface().(*testGzipBodyDecoderType)
		return ok
	}
	decodeBodyFn := func(_ context.Context, _ *http.Request, v reflect.Value, body io.Reader) error {
		bd := v.Interface().(*testGzipBodyDecoderType)
		return json.NewDecoder(body).Decode(bd)
	}
	makeBodyDecoder := func() kithttp.DecodeRequestFunc {
		return MakeDecoder(testGzipBodyDecoderType{}, defaultJSONUnmarshal, nil, isBodyDecoder, decodeBodyFn, nil, limit)
	}

	t.Run("DecodeBody gzip bomb exceeding decompressed limit returns 413", func(t *testing.T) {
		big := `{"data":"` + strings.Repeat("x", limit*10) + `"}`
		r := httptest.NewRequest("POST", "/", gzipBody(big))
		r.Header.Set("Content-Encoding", "gzip")
		_, err := makeBodyDecoder()(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "gzip bomb via DecodeBody must produce PayloadTooLargeError, got: %v", err)
		assert.True(t, ple.Gzipped, "PayloadTooLargeError from gzip bomb must have Gzipped set")
	})

	t.Run("DecodeBody valid gzip body within limit is decoded successfully", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", gzipBody(`{"data":"hi"}`))
		r.Header.Set("Content-Encoding", "gzip")
		result, err := makeBodyDecoder()(context.Background(), r)
		require.NoError(t, err)
		rd, ok := result.(*testGzipBodyDecoderType)
		require.True(t, ok)
		assert.Equal(t, "hi", rd.Data)
	})

	// Sub-test for the RequestDecoder code path where DecodeRequest itself
	// returns a PayloadTooLargeError (covers lines 545-546).
	t.Run("DecodeRequest returning PayloadTooLargeError preserves inner fields and sets Gzipped", func(t *testing.T) {
		makePayloadDecoder := func() kithttp.DecodeRequestFunc {
			return MakeDecoder(&testRequestDecoderPayloadTooLargeType{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)
		}
		r := httptest.NewRequest("POST", "/", gzipBody(`{"data":"hi"}`))
		r.Header.Set("Content-Encoding", "gzip")
		_, err := makePayloadDecoder()(context.Background(), r)
		require.Error(t, err)
		var ple platform_http.PayloadTooLargeError
		require.True(t, errors.As(err, &ple), "DecodeRequest returning PayloadTooLargeError must propagate, got: %v", err)
		assert.True(t, ple.Gzipped, "Gzipped must be set when the request was gzip-encoded")
		assert.Equal(t, int64(42), ple.MaxRequestSize, "MaxRequestSize from inner error must be preserved")
	})
}

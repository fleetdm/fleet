package wns

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServer wires a token endpoint and a channel endpoint into one httptest server, and records how
// each was called so tests can assert on auth caching, retries, and the client_id format.
type testServer struct {
	srv *httptest.Server

	tokenHits   atomic.Int32
	channelHits atomic.Int32

	// mu guards the captured request fields, which are written by server goroutines.
	mu               sync.Mutex
	lastClientID     string
	lastClientSecret string
	lastChannelAuth  string

	// channelStatus returns the HTTP status the channel endpoint should respond with, given the 1-based
	// call number. Defaults to always 200.
	channelStatus func(call int32) int
	// tokenExpiresIn is the expires_in value returned by the token endpoint.
	tokenExpiresIn int64
}

func (ts *testServer) clientID() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.lastClientID
}

func (ts *testServer) clientSecret() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.lastClientSecret
}

func (ts *testServer) channelAuth() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.lastChannelAuth
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{tokenExpiresIn: 86400}
	ts.channelStatus = func(int32) int { return http.StatusOK }

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		ts.tokenHits.Add(1)
		// Limit the body size (gosec G120) and use assert, not require, since this runs in the server
		// goroutine where a FailNow-style call would be unsafe.
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		assert.NoError(t, r.ParseForm())
		ts.mu.Lock()
		ts.lastClientID = r.PostForm.Get("client_id")
		ts.lastClientSecret = r.PostForm.Get("client_secret")
		ts.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"tok-%d","token_type":"bearer","expires_in":%d}`,
			ts.tokenHits.Load(), ts.tokenExpiresIn)
	})
	mux.HandleFunc("/channel", func(w http.ResponseWriter, r *http.Request) {
		n := ts.channelHits.Add(1)
		ts.mu.Lock()
		ts.lastChannelAuth = r.Header.Get("Authorization")
		ts.mu.Unlock()
		assert.Equal(t, "wns/raw", r.Header.Get("X-WNS-Type"))
		assert.Equal(t, "cache", r.Header.Get("X-WNS-Cache-Policy"))
		w.WriteHeader(ts.channelStatus(n))
	})

	// Use TLS so channel/token URLs are HTTPS, which SendRaw requires for the channel URI.
	ts.srv = httptest.NewTLSServer(mux)
	t.Cleanup(ts.srv.Close)
	return ts
}

func (ts *testServer) newClient(sid, secret string) *Client {
	c := NewClient(sid, secret)
	c.tokenURL = ts.srv.URL + "/token"
	// Trust the test server's self-signed certificate.
	c.httpClient = ts.srv.Client()
	return c
}

func (ts *testServer) channelURI() string { return ts.srv.URL + "/channel" }

func TestSendRaw(t *testing.T) {
	ctx := t.Context()

	t.Run("success and client_id is wrapped in ms-app scheme", func(t *testing.T) {
		ts := newTestServer(t)
		c := ts.newClient("S-1-15-2-123", "secret-value")

		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))

		assert.Equal(t, "ms-app://S-1-15-2-123", ts.clientID())
		assert.Equal(t, "secret-value", ts.clientSecret())
		assert.Equal(t, "Bearer tok-1", ts.channelAuth())
		assert.Equal(t, int32(1), ts.tokenHits.Load())
		assert.Equal(t, int32(1), ts.channelHits.Load())
	})

	t.Run("does not double-wrap a SID that already has the scheme", func(t *testing.T) {
		ts := newTestServer(t)
		c := ts.newClient("ms-app://S-1-15-2-999", "secret")

		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))
		assert.Equal(t, "ms-app://S-1-15-2-999", ts.clientID())
	})

	t.Run("token is cached across sends", func(t *testing.T) {
		ts := newTestServer(t)
		c := ts.newClient("S-1-15-2-123", "secret")

		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))
		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))

		assert.Equal(t, int32(1), ts.tokenHits.Load(), "second send should reuse the cached token")
		assert.Equal(t, int32(2), ts.channelHits.Load())
	})

	t.Run("token is refreshed after expiry", func(t *testing.T) {
		ts := newTestServer(t)
		ts.tokenExpiresIn = 3600
		c := ts.newClient("S-1-15-2-123", "secret")

		now := time.Now()
		c.nowFunc = func() time.Time { return now }

		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))
		assert.Equal(t, int32(1), ts.tokenHits.Load())

		// Advance past the cached token's expiry (3600s minus the 5m safety margin).
		now = now.Add(3600 * time.Second)
		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))
		assert.Equal(t, int32(2), ts.tokenHits.Load(), "expired token should be refetched")
	})

	t.Run("401 triggers a single token refresh and retry", func(t *testing.T) {
		ts := newTestServer(t)
		ts.channelStatus = func(call int32) int {
			if call == 1 {
				return http.StatusUnauthorized
			}
			return http.StatusOK
		}
		c := ts.newClient("S-1-15-2-123", "secret")

		require.NoError(t, c.SendRaw(ctx, ts.channelURI()))
		assert.Equal(t, int32(2), ts.tokenHits.Load(), "401 should force a fresh token")
		assert.Equal(t, int32(2), ts.channelHits.Load())
		assert.Equal(t, "Bearer tok-2", ts.channelAuth())
	})

	t.Run("concurrent sends share one token fetch", func(t *testing.T) {
		ts := newTestServer(t)
		c := ts.newClient("S-1-15-2-123", "secret")

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)
		errs := make([]error, goroutines)
		for i := range goroutines {
			go func() {
				defer wg.Done()
				errs[i] = c.SendRaw(ctx, ts.channelURI())
			}()
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		assert.Equal(t, int32(1), ts.tokenHits.Load(), "concurrent sends should reuse one cached token")
		assert.Equal(t, int32(goroutines), ts.channelHits.Load())
	})

	t.Run("410 returns ErrChannelExpired", func(t *testing.T) {
		ts := newTestServer(t)
		ts.channelStatus = func(int32) int { return http.StatusGone }
		c := ts.newClient("S-1-15-2-123", "secret")

		err := c.SendRaw(ctx, ts.channelURI())
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrChannelExpired), "got %v", err)
	})

	t.Run("other channel error surfaces the status", func(t *testing.T) {
		ts := newTestServer(t)
		ts.channelStatus = func(int32) int { return http.StatusForbidden }
		c := ts.newClient("S-1-15-2-123", "secret")

		err := c.SendRaw(ctx, ts.channelURI())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})
}

func TestFetchTokenErrors(t *testing.T) {
	ctx := t.Context()

	t.Run("non-200 token response is an error", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"invalid_client","error_description":"Invalid client id"}`)
		}))
		t.Cleanup(srv.Close)

		c := NewClient("S-1-15-2-123", "secret")
		c.tokenURL = srv.URL
		c.httpClient = srv.Client()

		err := c.SendRaw(ctx, srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid_client")
	})

	t.Run("missing access_token is an error", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"token_type":"bearer","expires_in":3600}`)
		}))
		t.Cleanup(srv.Close)

		c := NewClient("S-1-15-2-123", "secret")
		c.tokenURL = srv.URL
		c.httpClient = srv.Client()

		err := c.SendRaw(ctx, srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing access_token")
	})

	t.Run("non-HTTPS channel URI is refused before any token is fetched", func(t *testing.T) {
		c := NewClient("S-1-15-2-123", "secret")
		err := c.SendRaw(ctx, "http://evil.example.com/channel")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-HTTPS")
	})
}

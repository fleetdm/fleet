package main

// End-to-end spec for the mock APNS server, exercising the real mux
// (newMux) over HTTP via httptest. The wire contract these tests pin —
// request/response shapes, header semantics, error bodies, SSE stream
// behavior — is documented on pushHandler, parsePushHeaders, apnsPushError,
// and eventsSSEHandler in handlers.go. TestE2EBufordCompatibility verifies
// the contract through the actual buford client `fleet serve` uses in
// production.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	bufordpush "github.com/RobotsAndPencils/buford/push"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
	return newTestServerWithKeepAlive(t, 30*time.Second)
}

func newTestServerWithKeepAlive(t *testing.T, keepAlive time.Duration) *httptest.Server {
	return newTestServerWithTimeouts(t, keepAlive, 10*time.Second)
}

func newTestServerWithTimeouts(t *testing.T, keepAlive, writeTimeout time.Duration) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := httptest.NewServer(newMux(newStore(testTTL, logger), logger, keepAlive, writeTimeout))
	t.Cleanup(srv.Close)
	return srv
}

// --- SSE test client -------------------------------------------------------

type sseEvent struct {
	name    string // value of the "event:" field ("ping" for pushes)
	data    string // value of the "data:" field, or the comment text
	comment bool   // true for ":" keepalive comment lines
}

type sseClient struct {
	events <-chan sseEvent // closed when the stream ends
}

// openEvents performs the raw GET so tests can assert non-200 responses too.
func openEvents(t *testing.T, baseURL, token string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/events?token="+token, nil)
	require.NoError(t, err)
	// No client timeout: SSE connections are long-lived by design.
	resp, err := fleethttp.NewClient().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func sseConnect(t *testing.T, baseURL, token string) *sseClient {
	t.Helper()
	resp := openEvents(t, baseURL, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	events := make(chan sseEvent, 16)
	go func() {
		defer close(events)
		sc := bufio.NewScanner(resp.Body)
		var name string
		var data []string
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, ":"):
				events <- sseEvent{comment: true, data: strings.TrimSpace(line[1:])}
			case strings.HasPrefix(line, "event: "):
				name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data:"):
				// A payload containing newlines spans several data: lines;
				// SSE clients rejoin them with "\n".
				data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			case line == "":
				if name != "" || len(data) > 0 {
					events <- sseEvent{name: name, data: strings.Join(data, "\n")}
					name, data = "", nil
				}
			}
		}
	}()
	return &sseClient{events: events}
}

// nextPing waits for the next non-comment event and returns its payload.
func nextPing(t *testing.T, c *sseClient, timeout time.Duration) string {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				t.Fatal("SSE stream closed while waiting for a ping event")
			}
			if ev.comment {
				continue
			}
			assert.Equal(t, "ping", ev.name, "push events must be named ping")
			return ev.data
		case <-deadline:
			t.Fatal("timed out waiting for a ping event")
		}
	}
}

// expectNoPing asserts no push event arrives within the wait window.
// Keepalive comments are fine; a closed stream delivers nothing, so it
// passes too.
func expectNoPing(t *testing.T, c *sseClient, wait time.Duration) {
	t.Helper()
	deadline := time.After(wait)
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				return
			}
			if ev.comment {
				continue
			}
			t.Fatalf("expected no ping event, got %q with data %q", ev.name, ev.data)
		case <-deadline:
			return
		}
	}
}

func waitStreamClosed(t *testing.T, c *sseClient, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-c.events:
			if !ok {
				return
			}
			// drain whatever was still buffered
		case <-deadline:
			t.Fatal("timed out waiting for the SSE stream to close")
		}
	}
}

// --- HTTP helpers ----------------------------------------------------------

// pushRaw sends a push the way a spec-correct APNS client would, declaring
// apns-push-type: mdm by default. Pass an empty string value in headers to
// omit that header instead (Fleet's buford client sends no apns-* headers at
// all today — TestE2EPushTypeHeader covers that path explicitly).
func pushRaw(t *testing.T, baseURL, token string, payload []byte, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/3/device/"+token, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-push-type", "mdm")
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// requireAPNSErrorBody asserts the response matches the real-APNS error
// shape documented on apnsPushError (JSON reason body, apns-id present on
// errors, timestamp only on 410 Unregistered) and returns the reason.
func requireAPNSErrorBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	var bodyBytes []byte
	bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 64<<10)) // 64KB limit to avoid OOM if the server misbehaves
	assert.Contains(t, resp.Header.Get("Content-Type"), "application/json", string(bodyBytes))
	assert.NotEmpty(t, resp.Header.Get("apns-id"), "real APNS returns apns-id on error responses too")
	var body struct {
		Reason    string `json:"reason"`
		Timestamp *int64 `json:"timestamp"`
	}
	require.NoError(t, json.Unmarshal(bodyBytes, &body),
		"non-200 responses must have a JSON body: buford's parseErrorResponse returns a JSON-decode error to the caller otherwise")
	if resp.StatusCode == http.StatusGone {
		assert.NotNil(t, body.Timestamp, "410 Unregistered must carry the unix-millis timestamp of when the token died")
	} else {
		assert.Nil(t, body.Timestamp, "real APNS only includes timestamp on 410 Unregistered")
	}
	return body.Reason
}

func getStats(t *testing.T, baseURL string) statsResponse {
	t.Helper()
	resp, err := http.Get(baseURL + "/stats")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var st statsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&st))
	return st
}

// waitConnected polls /stats until the expected number of clients is
// connected. Subscription happens after the SSE response headers are sent,
// so a connect immediately followed by a push can race it; tests that need
// the live-delivery path synchronize here.
func waitConnected(t *testing.T, baseURL string, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if getStats(t, baseURL).ActiveConnections == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d connected client(s)", want)
}

// --- Tests -----------------------------------------------------------------

func TestE2EPushDeliveredToConnectedClient(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee01" // nolint:gosec // test token

	c := sseConnect(t, srv.URL, token)
	waitConnected(t, srv.URL, 1)

	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"magic1"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("apns-id"))

	assert.JSONEq(t, `{"mdm":"magic1"}`, nextPing(t, c, 5*time.Second))

	stats := getStats(t, srv.URL)
	assert.Equal(t, 1, stats.TotalPushes)
	assert.Equal(t, 1, stats.DeliveredLive)
	assert.Equal(t, 0, stats.Stored)
}

func TestE2EOfflinePushDeliveredOnConnect(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee02" // nolint:gosec // test token

	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"magic2"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	c := sseConnect(t, srv.URL, token)
	assert.JSONEq(t, `{"mdm":"magic2"}`, nextPing(t, c, 5*time.Second))

	stats := getStats(t, srv.URL)
	assert.Equal(t, 1, stats.Stored)
	assert.Equal(t, 1, stats.DeliveredOnConnect)
}

func TestE2EOfflinePushesCoalesceToLatest(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee03" // nolint:gosec // test token

	for _, magic := range []string{"m1", "m2", "m3"} {
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"`+magic+`"}`), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	c := sseConnect(t, srv.URL, token)
	assert.JSONEq(t, `{"mdm":"m3"}`, nextPing(t, c, 5*time.Second))
	expectNoPing(t, c, 200*time.Millisecond)

	stats := getStats(t, srv.URL)
	assert.Equal(t, 3, stats.TotalPushes)
	assert.Equal(t, 1, stats.Stored)
	assert.Equal(t, 2, stats.Coalesced)
}

func TestE2EPushWithPastExpirationDiscarded(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee04" // nolint:gosec // test token

	// apns-expiration is unix SECONDS (buford's Headers.Expiration marshals
	// seconds); 1 is 1970, long past. Device offline → discard, don't store.
	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"stale"}`), map[string]string{"apns-expiration": "1"})
	require.Equal(t, http.StatusOK, resp.StatusCode, "a discarded push is still a successful push (APNS semantics)")

	c := sseConnect(t, srv.URL, token)
	expectNoPing(t, c, 300*time.Millisecond)

	stats := getStats(t, srv.URL)
	assert.Equal(t, 0, stats.Stored)
	assert.Equal(t, 1, stats.Expired)
}

func TestE2EPushWithFutureExpirationStored(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee05" // nolint:gosec // test token

	exp := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"fresh"}`), map[string]string{"apns-expiration": exp})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	c := sseConnect(t, srv.URL, token)
	assert.JSONEq(t, `{"mdm":"fresh"}`, nextPing(t, c, 5*time.Second))
}

func TestE2EAPNSIDHeader(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee06" // nolint:gosec // test token

	t.Run("generated when absent", func(t *testing.T) {
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		id := resp.Header.Get("apns-id")
		_, err := uuid.Parse(id)
		assert.NoError(t, err, "generated apns-id should be a UUID, got %q", id)
	})

	t.Run("echoed on error responses", func(t *testing.T) {
		// Real APNS returns apns-id on errors too; it is how a push is
		// correlated with its response when dumping raw traffic.
		const reqID = "6f1e3a8d-2b4c-4d1e-8f0a-9b8c7d6e5f40"
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), map[string]string{
			"apns-id":        reqID,
			"apns-push-type": "alert", // rejected in parsePushHeaders, before the id is returned
		})
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, reqID, resp.Header.Get("apns-id"))
	})

	t.Run("echoed when supplied", func(t *testing.T) {
		// Real APNS echoes a request-supplied apns-id back in the response.
		const reqID = "0f0b8e5c-3d5c-4c2e-9f9a-1c2d3e4f5a6b"
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), map[string]string{"apns-id": reqID})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, reqID, resp.Header.Get("apns-id"))
	})
}

func TestE2EPushTypeHeader(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee0c" // nolint:gosec // test token

	t.Run("absent header is accepted", func(t *testing.T) {
		// Fleet's buford path sends no apns-push-type header at all, so
		// header-less pushes must keep working.
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), map[string]string{"apns-push-type": ""})
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("non-mdm push type is rejected", func(t *testing.T) {
		// The mock only models MDM wake-up pushes; a declared non-mdm type is
		// a client bug. Real APNS rejects mismatched push types with
		// InvalidPushType.
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), map[string]string{"apns-push-type": "alert"})
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "InvalidPushType", requireAPNSErrorBody(t, resp))
	})
}

func TestE2EPayloadWithNewlineIsFramedSafely(t *testing.T) {
	// PushMagic comes from the device's TokenUpdate and buford builds the
	// body by string concatenation, so a payload can contain a raw newline.
	// Emitted as-is it would end the SSE event early and corrupt every frame
	// after it; the server must split it across data: lines instead.
	srv := newTestServer(t)
	const token = "aabbccddee0d" // nolint:gosec // test token
	payload := "{\"mdm\":\"line1\nline2\"}"

	c := sseConnect(t, srv.URL, token)
	waitConnected(t, srv.URL, 1)

	resp := pushRaw(t, srv.URL, token, []byte(payload), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, payload, nextPing(t, c, 5*time.Second), "payload must survive the round trip intact")

	// Framing is still intact for the next event.
	resp = pushRaw(t, srv.URL, token, []byte(`{"mdm":"after"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"mdm":"after"}`, nextPing(t, c, 5*time.Second))
}

func TestE2EWriteTimeoutDoesNotKillHealthyStream(t *testing.T) {
	// The per-write deadline exists so a device that stops reading cannot pin
	// its token forever. It must be re-armed on every write, or a stream that
	// simply lives longer than the timeout would be torn down.
	srv := newTestServerWithTimeouts(t, 20*time.Millisecond, 50*time.Millisecond)
	const token = "aabbccddee0e" // nolint:gosec // test token

	c := sseConnect(t, srv.URL, token)
	waitConnected(t, srv.URL, 1)

	time.Sleep(200 * time.Millisecond) // several keepalive intervals, well past one write timeout

	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"still here"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"mdm":"still here"}`, nextPing(t, c, 5*time.Second))
}

func TestE2EInvalidTokenRejected(t *testing.T) {
	srv := newTestServer(t)

	for _, token := range []string{"not-hex-token", "abc"} { // non-hex chars; odd length
		resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "token %q", token)
		assert.Equal(t, "BadDeviceToken", requireAPNSErrorBody(t, resp), "token %q", token)
	}
}

func TestE2EPayloadLimits(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee07" // nolint:gosec // test token

	t.Run("empty payload", func(t *testing.T) {
		resp := pushRaw(t, srv.URL, token, nil, nil)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "PayloadEmpty", requireAPNSErrorBody(t, resp))
	})

	t.Run("payload too large", func(t *testing.T) {
		resp := pushRaw(t, srv.URL, token, bytes.Repeat([]byte("x"), 4098), nil)
		require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
		assert.Equal(t, "PayloadTooLarge", requireAPNSErrorBody(t, resp))
	})

	t.Run("4096 bytes is accepted", func(t *testing.T) {
		resp := pushRaw(t, srv.URL, token, bytes.Repeat([]byte("x"), 4096), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestE2ETokenIsCaseInsensitive(t *testing.T) {
	srv := newTestServer(t)

	// Fleet always sends lowercase hex (hex.EncodeToString), but hex tokens
	// are case-insensitive; normalize instead of relying on the caller.
	c := sseConnect(t, srv.URL, "aabbccddee08")
	waitConnected(t, srv.URL, 1)

	resp := pushRaw(t, srv.URL, "AABBCCDDEE08", []byte(`{"mdm":"m"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"mdm":"m"}`, nextPing(t, c, 5*time.Second))
}

func TestE2EEventsTokenValidation(t *testing.T) {
	srv := newTestServer(t)

	t.Run("missing token", func(t *testing.T) {
		resp := openEvents(t, srv.URL, "")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"validation must happen before the stream starts (before any Flush commits a 200)")
	})

	t.Run("non-hex token", func(t *testing.T) {
		resp := openEvents(t, srv.URL, "not-hex-token")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestE2EReconnectReplacesOlderConnection(t *testing.T) {
	srv := newTestServer(t)
	const token = "aabbccddee09" // nolint:gosec // test token

	oldConn := sseConnect(t, srv.URL, token)
	waitConnected(t, srv.URL, 1)

	newConn := sseConnect(t, srv.URL, token)
	// The replaced handler returns, ending the old stream.
	waitStreamClosed(t, oldConn, 5*time.Second)
	waitConnected(t, srv.URL, 1)

	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"mdm":"m"}`, nextPing(t, newConn, 5*time.Second))
}

func TestE2EHealthz(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestE2EStatsStartAtZero(t *testing.T) {
	srv := newTestServer(t)
	assert.Equal(t, statsResponse{}, getStats(t, srv.URL))
}

func TestE2EKeepalive(t *testing.T) {
	srv := newTestServerWithKeepAlive(t, 50*time.Millisecond)
	const token = "aabbccddee0b" // nolint:gosec // test token

	c := sseConnect(t, srv.URL, token)

	// Keepalives must arrive repeatedly (a ticker, not a one-shot) and be
	// SSE comment lines, which real SSE clients ignore — never ping events.
	seen := 0
	deadline := time.After(5 * time.Second)
	for seen < 3 {
		select {
		case ev, ok := <-c.events:
			require.True(t, ok, "SSE stream closed while waiting for keepalives")
			require.True(t, ev.comment, "expected only keepalive comments, got event %q with data %q", ev.name, ev.data)
			assert.Equal(t, "keepalive", ev.data)
			seen++
		case <-deadline:
			t.Fatalf("timed out: got %d/3 keepalive comments", seen)
		}
	}

	// Keepalives must not corrupt event framing: a ping pushed after several
	// keepalives still parses as a normal event.
	resp := pushRaw(t, srv.URL, token, []byte(`{"mdm":"m"}`), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"mdm":"m"}`, nextPing(t, c, 5*time.Second))
}

// TestE2EDisconnectedStreamIsReapedByKeepalive pins the liveness tradeoff
// eventsSSEHandler makes by hijacking the connection: no goroutine watches
// the socket for a client that vanished, so the keepalive write is what
// discovers it. A stream whose client is gone must unsubscribe itself within
// a keepalive interval or two, or the store leaks a subscriber and every
// later push to that token is coalesced into a connection that can never
// deliver it (see streamEvents).
func TestE2EDisconnectedStreamIsReapedByKeepalive(t *testing.T) {
	srv := newTestServerWithKeepAlive(t, 50*time.Millisecond)
	const token = "aabbccddee0c" // nolint:gosec // test token

	// A raw socket, so the close below is abrupt: no request-context
	// cancellation, nothing but a dead peer for the next write to find.
	conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	require.NoError(t, err)
	_, err = io.WriteString(conn, "GET /events?token="+token+" HTTP/1.1\r\nHost: localhost\r\n\r\n")
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	waitConnected(t, srv.URL, 1)

	require.NoError(t, conn.Close())
	waitConnected(t, srv.URL, 0)
}

// TestE2EBufordCompatibility drives the mock through the actual buford
// client library — the same code path `fleet serve` uses to talk to Apple
// (server/mdm/nanomdm/push/buford wraps bufordpush.Service). If these pass,
// pointing Fleet's push provider at the mock works.
func TestE2EBufordCompatibility(t *testing.T) {
	srv := newTestServer(t)
	svc := bufordpush.NewService(fleethttp.NewClient(), srv.URL)

	t.Run("successful push round-trips to a client", func(t *testing.T) {
		const token = "aabbccddee0a" // nolint:gosec // test token
		id, err := svc.Push(token, nil, []byte(`{"mdm":"pushmagicABC"}`))
		require.NoError(t, err)
		assert.NotEmpty(t, id, "buford returns the apns-id header on success")

		c := sseConnect(t, srv.URL, token)
		assert.JSONEq(t, `{"mdm":"pushmagicABC"}`, nextPing(t, c, 5*time.Second))
	})

	t.Run("invalid token surfaces as BadDeviceToken", func(t *testing.T) {
		_, err := svc.Push("not-hex-token", nil, []byte(`{"mdm":"m"}`))
		var apnsErr *bufordpush.Error
		require.ErrorAs(t, err, &apnsErr,
			"error must decode as a buford push error, not a JSON parse failure — the mock must always send the JSON error body")
		assert.Equal(t, bufordpush.ErrBadDeviceToken, apnsErr.Reason)
		assert.Equal(t, http.StatusBadRequest, apnsErr.Status)
	})
}

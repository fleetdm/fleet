package apnsmock

// Behavioral spec for Client, the SSE subscriber simulated MDM devices use
// to receive wake-up pings from the mock APNS server (cmd/apple-apns-mock).
// The wire format and delivery semantics the client must implement are
// documented on the Client type in client.go.
//
// These tests run against a scripted SSE server so they can stage
// disconnects, failures, and slow consumers deterministically; the real
// server's contract is pinned by cmd/apple-apns-mock's own e2e tests.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "746f6b656e414243" // nolint:gosec // test token

// fastBackoff keeps reconnect tests quick.
var fastBackoff = WithBackoff(5*time.Millisecond, 25*time.Millisecond)

// --- scripted SSE server ----------------------------------------------------

// sseServer counts connections and hands each one, with its 1-based index,
// to the test's script. The script runs on the server's handler goroutine:
// use assert (goroutine-safe), never require/Fatal.
type sseServer struct {
	srv   *httptest.Server
	mu    sync.Mutex
	conns int
}

func newSSEServer(t *testing.T, script func(conn int, w http.ResponseWriter, r *http.Request)) *sseServer {
	t.Helper()
	s := &sseServer{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.conns++
		conn := s.conns
		s.mu.Unlock()
		script(conn, w, r)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *sseServer) URL() string { return s.srv.URL }

func (s *sseServer) connCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conns
}

// startSSE writes the SSE response headers and flushes them, committing the
// 200 so the client sees a successful connect.
func startSSE(w http.ResponseWriter) http.Flusher {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher := w.(http.Flusher)
	flusher.Flush()
	return flusher
}

func sendPing(w http.ResponseWriter, f http.Flusher, payload string) {
	fmt.Fprintf(w, "event: ping\ndata: %s\n\n", payload)
	f.Flush()
}

func sendComment(w http.ResponseWriter, f http.Flusher) {
	fmt.Fprint(w, ": keepalive\n\n")
	f.Flush()
}

// --- receive helpers ---------------------------------------------------------

func recvPing(t *testing.T, ch <-chan Ping, timeout time.Duration) Ping {
	t.Helper()
	select {
	case p, ok := <-ch:
		require.True(t, ok, "pings channel closed while waiting for a ping")
		return p
	case <-time.After(timeout):
		t.Fatal("timed out waiting for a ping")
		return Ping{}
	}
}

func expectNoPing(t *testing.T, ch <-chan Ping, wait time.Duration) {
	t.Helper()
	select {
	case p, ok := <-ch:
		if ok {
			t.Fatalf("expected no ping, got one with magic %q", p.PushMagic)
		}
	case <-time.After(wait):
	}
}

func waitClosed(t *testing.T, ch <-chan Ping, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			// drain whatever is still buffered
		case <-deadline:
			t.Fatal("timed out waiting for the pings channel to close")
		}
	}
}

// --- tests -------------------------------------------------------------------

func TestClientConnectsAndReceivesPing(t *testing.T) {
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/events", r.URL.Path)
		assert.Equal(t, testToken, r.URL.Query().Get("token"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		f := startSSE(w)
		sendPing(w, f, `{"mdm":"pushmagicABC"}`)
		<-r.Context().Done() // hold the stream open
	})

	c := NewClient(srv.URL(), testToken)
	c.Start(t.Context())

	p := recvPing(t, c.Pings(), 5*time.Second)
	assert.Equal(t, "pushmagicABC", p.PushMagic)
	assert.JSONEq(t, `{"mdm":"pushmagicABC"}`, string(p.Raw))
	assert.WithinDuration(t, time.Now(), p.ReceivedAt, 5*time.Second)
}

func TestClientPushMagicParsing(t *testing.T) {
	// The client parses PushMagic as a convenience but must never drop a
	// ping over an unexpected payload shape — Raw is always preserved.
	for _, tc := range []struct {
		name      string
		payload   string
		wantMagic string
	}{
		{"mdm payload", `{"mdm":"pushmagicXYZ"}`, "pushmagicXYZ"},
		{"other json", `{"aps":{"alert":"hi"}}`, ""},
		{"not json", `garbage!`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
				f := startSSE(w)
				sendPing(w, f, tc.payload)
				<-r.Context().Done()
			})

			c := NewClient(srv.URL(), testToken)
			c.Start(t.Context())

			p := recvPing(t, c.Pings(), 5*time.Second)
			assert.Equal(t, tc.wantMagic, p.PushMagic)
			assert.Equal(t, tc.payload, string(p.Raw))
		})
	}
}

func TestClientIgnoresKeepaliveComments(t *testing.T) {
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		f := startSSE(w)
		sendComment(w, f)
		sendComment(w, f)
		sendPing(w, f, `{"mdm":"m"}`)
		sendComment(w, f)
		<-r.Context().Done()
	})

	c := NewClient(srv.URL(), testToken)
	c.Start(t.Context())

	p := recvPing(t, c.Pings(), 5*time.Second)
	assert.Equal(t, "m", p.PushMagic)
	expectNoPing(t, c.Pings(), 200*time.Millisecond)
}

func TestClientReconnectsAfterDisconnect(t *testing.T) {
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		f := startSSE(w)
		switch conn {
		case 1:
			sendPing(w, f, `{"mdm":"first"}`)
			// return: the stream ends, as when the mock server restarts or a
			// newer connection for the token replaces this one
		default:
			sendPing(w, f, `{"mdm":"second"}`)
			<-r.Context().Done()
		}
	})

	c := NewClient(srv.URL(), testToken, fastBackoff)
	c.Start(t.Context())

	assert.Equal(t, "first", recvPing(t, c.Pings(), 5*time.Second).PushMagic)
	assert.Equal(t, "second", recvPing(t, c.Pings(), 5*time.Second).PushMagic)
	assert.GreaterOrEqual(t, srv.connCount(), 2)
}

func TestClientBacksOffAfterCleanStreamEnd(t *testing.T) {
	// A stream that ends cleanly is still a disconnect. Redialing with no
	// delay spins at 100% CPU, and at 300k agents a single mock restart
	// becomes a reconnect storm the server cannot come back up under.
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		startSSE(w) // 200, then immediately end the stream
	})

	c := NewClient(srv.URL(), testToken, WithBackoff(40*time.Millisecond, time.Second))
	c.Start(t.Context())

	time.Sleep(300 * time.Millisecond)

	conns := srv.connCount()
	assert.Positive(t, conns, "client should keep retrying")
	// 300ms of 40ms-and-doubling backoff is ~4 attempts; allow slack for
	// scheduling, but a no-backoff loop would land in the thousands.
	assert.LessOrEqual(t, conns, 12, "reconnects must be throttled by the backoff")
}

func TestClientJoinsMultiLineData(t *testing.T) {
	// The server splits a payload containing newlines across data: lines
	// (SSE frames are newline delimited). The client must rejoin them, not
	// treat the continuation as a protocol error and drop the stream.
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		f := startSSE(w)
		fmt.Fprint(w, "event: ping\ndata: {\"mdm\":\"line1\ndata: line2\"}\n\n")
		f.Flush()
		<-r.Context().Done()
	})

	c := NewClient(srv.URL(), testToken)
	c.Start(t.Context())

	p := recvPing(t, c.Pings(), 5*time.Second)
	assert.Equal(t, "{\"mdm\":\"line1\nline2\"}", string(p.Raw), "the payload must be rejoined byte for byte") // nolint:testifylint // not valid JSON
	// A raw newline inside a JSON string is not valid JSON, so the magic
	// cannot be parsed — the ping is still delivered, Raw intact.
	assert.Empty(t, p.PushMagic)
	assert.Equal(t, 1, srv.connCount(), "a multi-line payload must not tear the stream down")
}

func TestClientRetriesFailedConnections(t *testing.T) {
	// Any connect failure — non-200, network error — is retryable with
	// backoff; the client never gives up until its context is done.
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		if conn <= 2 {
			http.Error(w, `{"reason":"ServiceUnavailable"}`, http.StatusServiceUnavailable)
			return
		}
		f := startSSE(w)
		sendPing(w, f, `{"mdm":"finally"}`)
		<-r.Context().Done()
	})

	c := NewClient(srv.URL(), testToken, fastBackoff)
	c.Start(t.Context())

	assert.Equal(t, "finally", recvPing(t, c.Pings(), 5*time.Second).PushMagic)
	assert.GreaterOrEqual(t, srv.connCount(), 3)
}

func TestClientCoalescesWhenConsumerIsSlow(t *testing.T) {
	// The pings channel is buffered 1 and the client drops pings when it is
	// full — an MDM wake-up carries no unique data, so one queued ping is as
	// good as twenty (same semantics as the server's live-delivery path).
	//
	// Determinism: the client only reconnects after fully consuming the
	// first stream, so once connection 2 is up, all 20 pings were processed
	// while the consumer read nothing — exactly one (the first) can be
	// buffered.
	conn2Up := make(chan struct{})
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		f := startSSE(w)
		switch conn {
		case 1:
			for i := 1; i <= 20; i++ {
				sendPing(w, f, fmt.Sprintf(`{"mdm":"magic%d"}`, i))
			}
			// return: disconnect so the client's reconnect signals "all 20 processed"
		default:
			if conn == 2 {
				close(conn2Up)
			}
			<-r.Context().Done()
		}
	})

	c := NewClient(srv.URL(), testToken, fastBackoff)
	c.Start(t.Context())

	select {
	case <-conn2Up:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the client to reconnect")
	}

	p := recvPing(t, c.Pings(), 5*time.Second)
	assert.Equal(t, "magic1", p.PushMagic, "the queued ping is the first received; later ones are dropped, not queued")
	expectNoPing(t, c.Pings(), 200*time.Millisecond)
}

func TestClientClosesChannelOnContextCancel(t *testing.T) {
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		f := startSSE(w)
		sendPing(w, f, `{"mdm":"m"}`)
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	c := NewClient(srv.URL(), testToken)
	c.Start(ctx)

	recvPing(t, c.Pings(), 5*time.Second)
	cancel()
	// Cancellation must abort the blocking stream read (the request carries
	// the context) and close the channel — the consumer's range loop ends.
	waitClosed(t, c.Pings(), 5*time.Second)
}

func TestClientCancelWhileDisconnectedClosesChannel(t *testing.T) {
	// Cancellation during the backoff wait must also end the loop promptly.
	srv := newSSEServer(t, func(conn int, w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusServiceUnavailable)
	})

	ctx, cancel := context.WithCancel(t.Context())
	c := NewClient(srv.URL(), testToken, WithBackoff(time.Hour, time.Hour)) // parked in backoff
	c.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for srv.connCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	require.Positive(t, srv.connCount(), "client never attempted to connect")

	cancel()
	waitClosed(t, c.Pings(), 5*time.Second)
}

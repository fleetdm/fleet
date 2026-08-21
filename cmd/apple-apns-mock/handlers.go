package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// eventsSSEHandler serves GET /events?token=<hex>, the simulated device's
// stand-in for a real device's APNs courier connection. Pushes arrive as
// `event: ping`, anything stored while the device was away is written on
// connect, and a newer connection for the same token replaces this one.
// Token validation must precede any write — the first write commits a 200.
//
// It hijacks the connection, writes the response head itself, hands the socket
// to one goroutine and returns. Returning lets net/http drop this connection's
// buffers and its disconnect-watching goroutine, ~14 of the ~18KB a blocked
// handler costs.
func eventsSSEHandler(w http.ResponseWriter, r *http.Request, c *coordinator, logger *slog.Logger, keepAlive, writeTimeout time.Duration) {
	token := r.URL.Query().Get("token")
	if token == "" {
		apnsPushError(w, nil, MissingDeviceTokenError())
		return
	}

	if _, err := hex.DecodeString(token); err != nil {
		apnsPushError(w, nil, BadDeviceTokenError())
		return
	}

	logger.DebugContext(r.Context(), "starting SSE stream", "token", token)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		// HTTP/2, httptest recorders, or any middleware that wraps the
		// ResponseWriter without forwarding Hijack.
		streamEventsBuffered(w, r, token, c, logger, keepAlive, writeTimeout)
		return
	}
	// The *bufio.ReadWriter is discarded on purpose: keeping it would pin the
	// buffers hijacking exists to release, and nothing is buffered either way
	// at this point.
	conn, _, err := hijacker.Hijack()
	if err != nil {
		logger.DebugContext(r.Context(), "hijack failed, falling back to buffered stream", "token", token, "error", err)
		streamEventsBuffered(w, r, token, c, logger, keepAlive, writeTimeout)
		return
	}

	if _, err := io.WriteString(conn, sseResponseHead); err != nil {
		logger.DebugContext(r.Context(), "failed to write SSE response head", "token", token, "error", err)
		conn.Close()
		return
	}

	// strings.Clone cuts the token loose from the request's URL string, which
	// would otherwise stay alive for the life of the stream.
	//nolint:gosec // G118: the request context is canceled the moment this handler returns, and returning is exactly what frees the per-connection buffers. The stream has to outlive it.
	go streamEvents(conn, strings.Clone(token), c, logger, keepAlive, writeTimeout)
}

// sseResponseHead is the 200 eventsSSEHandler writes by hand after hijacking.
// A stream that never ends cannot be length-delimited, so it omits
// Content-Length and Transfer-Encoding: HTTP/1.1 reads that as
// read-until-close. Chunked framing would cost a size line and a buffer per
// frame.
const sseResponseHead = "HTTP/1.1 200 OK\r\n" +
	"Content-Type: text/event-stream\r\n" +
	"Cache-Control: no-cache\r\n" +
	"Connection: close\r\n" +
	"\r\n"

// streamEvents owns one hijacked connection for its whole life.
//
// Hijacking gives up immediate notice that the client went away, so keepalive
// writes double as liveness probes: a peer that sent FIN draws RST on the next
// write. With --keep-alive 0 nothing probes and the stream lingers until the
// next push. Each write carries a writeTimeout deadline; the server sets no
// WriteTimeout (SSE must not be reaped), so without one a device that stops
// reading would block this goroutine forever.
func streamEvents(conn net.Conn, token string, c *coordinator, logger *slog.Logger, keepAlive, writeTimeout time.Duration) {
	defer conn.Close()

	write := func(frame string) error {
		if writeTimeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return err
			}
		}
		_, err := io.WriteString(conn, frame)
		return err
	}
	runStream(context.Background(), write, nil, token, c, logger, keepAlive)
}

// streamEventsBuffered is the fallback for ResponseWriters that cannot be
// hijacked. It blocks in the handler, keeping the full net/http footprint, and
// in exchange notices a departing client immediately.
func streamEventsBuffered(w http.ResponseWriter, r *http.Request, token string, c *coordinator, logger *slog.Logger, keepAlive, writeTimeout time.Duration) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	rc := http.NewResponseController(w)
	write := func(frame string) error {
		if writeTimeout > 0 {
			if err := rc.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil &&
				!errors.Is(err, http.ErrNotSupported) {
				return err
			}
		}
		if _, err := io.WriteString(w, frame); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	runStream(r.Context(), write, r.Context().Done(), token, c, logger, keepAlive)
}

// runStream is the SSE loop both paths share: subscribe, flush anything the
// device missed, then forward pushes and keepalives until the stream ends.
// done is nil on the hijacked path, where a nil channel never fires.
//
// A ping drained but not written goes back to Redis via Restore, so a broken
// connection loses no wake-up and no counter drifts.
func runStream(ctx context.Context, write func(string) error, done <-chan struct{}, token string, c *coordinator, logger *slog.Logger, keepAlive time.Duration) {
	sub, pending := c.Subscribe(ctx, token)
	defer c.Unsubscribe(ctx, token, sub)

	if pending != nil {
		// stored ping delivered immediately on connect
		if err := write(pingEvent(pending.payload)); err != nil {
			logger.DebugContext(ctx, "failed to write pending ping", "token", token, "error", err)
			c.Restore(ctx, token, sub, pending, true)
			return
		}
	}

	var keepAliveC <-chan time.Time
	if keepAlive > 0 {
		ticker := time.NewTicker(keepAlive)
		defer ticker.Stop()
		keepAliveC = ticker.C
	}

	for {
		select {
		case p := <-sub.ch: // push arrived while we're live
			if err := write(pingEvent(p.payload)); err != nil {
				logger.DebugContext(ctx, "failed to write ping", "token", token, "error", err)
				c.Restore(ctx, token, sub, p, false)
				return
			}
		case <-sub.replaced: // a newer connection took our token — stand down
			return
		case <-done: // client went away (buffered path only; nil channel never fires)
			return
		case <-keepAliveC: // keepalive
			if err := write(": keepalive\n\n"); err != nil {
				logger.DebugContext(ctx, "failed to write keepalive", "token", token, "error", err)
				return
			}
		}
	}
}

// pingEvent frames one payload as an SSE event. SSE is newline delimited, so
// a payload containing a newline is split across several data: lines (clients
// rejoin them with "\n"); emitting it raw would corrupt every frame after it.
// PushMagic comes from the device's TokenUpdate, so newlines are possible.
func pingEvent(payload []byte) string {
	var b strings.Builder
	b.WriteString("event: ping\n")
	for line := range bytes.SplitSeq(payload, []byte("\n")) {
		b.WriteString("data: ")
		b.Write(bytes.TrimSuffix(line, []byte("\r")))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// pushHandler serves POST /3/device/{token}, the same shape as
// api.push.apple.com. It accepts what Fleet's buford client sends — a raw JSON
// body and no apns-* headers — plus the spec headers parsePushHeaders models,
// and applies APNs payload limits.
//
// The payload is forwarded verbatim, not parsed. Token validation is looser
// than real APNs on purpose: any even-length hex is accepted, because Apple
// also enforces a 32-byte length and mdmtest tokens are variable-length.
func pushHandler(w http.ResponseWriter, r *http.Request, c *coordinator, logger *slog.Logger) {
	// parsePushHeaders returns its headers even on failure so the error
	// response echoes the client's apns-id, like real APNS does.
	headers, err := parsePushHeaders(r)
	if err != nil {
		apnsPushError(w, headers, err)
		return
	}

	token := r.PathValue("token")
	if token == "" {
		apnsPushError(w, headers, MissingDeviceTokenError())
		return
	}

	if _, err := hex.DecodeString(token); err != nil {
		apnsPushError(w, headers, BadDeviceTokenError())
		return
	}

	limitedReader := io.LimitReader(r.Body, 4097)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to read request body", "error", err)
		apnsPushError(w, headers, InternalServerError())
		return
	}
	if len(payload) == 0 {
		apnsPushError(w, headers, PayloadEmptyError())
		return
	}
	if len(payload) > 4096 {
		apnsPushError(w, headers, PayloadTooLargeError())
		return
	}

	expiration := time.Now().Add(c.cfg.DefaultTTL)
	if headers.Expiration != nil {
		expiration = *headers.Expiration
	}
	// A push that cannot be stored or announced was not accepted; saying so
	// lets Fleet's pending-hosts cron retry it.
	if err := c.Push(r.Context(), token, payload, expiration); err != nil {
		apnsPushError(w, headers, ServiceUnavailableError())
		return
	}

	w.Header().Set("apns-id", headers.PushID)
	w.WriteHeader(http.StatusOK)
	logger.DebugContext(r.Context(), "push accepted", "token", token, "apns-id", headers.PushID, "payload_size", len(payload), "expiration", expiration)
}

// pushHeaders holds the apns-* request headers the mock models. Fleet sends
// none of them today, so every field has an "absent" behavior; this struct
// is the extension point for future headers (apns-priority,
// apns-collapse-id, ...).
type pushHeaders struct {
	PushID     string     // apns-id: echoed back if given (else a generated UUID); non-UUID → 400 BadMessageId
	PushType   string     // apns-push-type: absent or "mdm" accepted; anything else → 400 InvalidPushType (this mock only models MDM wake-ups)
	Expiration *time.Time // apns-expiration: unix seconds; 0/past = deliver-now-or-discard, nil = server default TTL
}

// parsePushHeaders validates the modeled apns-* headers. The returned headers
// are non-nil even on error, so the caller can echo the client's apns-id the
// way real APNs does. A malformed apns-id is the exception: the generated one
// stands.
func parsePushHeaders(r *http.Request) (*pushHeaders, error) {
	pushHeaders := &pushHeaders{
		PushID: uuid.NewString(), // default to random UUID, if provided it will be overwritten.
	}

	if pushID := r.Header.Get("apns-id"); pushID != "" {
		if _, err := uuid.Parse(pushID); err != nil {
			return pushHeaders, BadMessageIdError()
		}
		pushHeaders.PushID = pushID
	}

	pushHeaders.PushType = r.Header.Get("apns-push-type")
	if pushHeaders.PushType != "" && pushHeaders.PushType != "mdm" {
		return pushHeaders, InvalidPushTypeError()
	}

	if expiration := r.Header.Get("apns-expiration"); expiration != "" {
		if ts, err := strconv.ParseInt(expiration, 10, 64); err == nil {
			t := time.Unix(ts, 0)
			pushHeaders.Expiration = &t
		} else {
			return pushHeaders, BadExpirationDateError()
		}
	}

	return pushHeaders, nil
}

// healthzHandler serves GET /healthz for infra liveness checks.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// nodeStats are one instance's counters. Stored counts pending keys written,
// and every push writes one, so it tracks pushes rather than offline devices.
// Discarded replaces the old Expired: a ping that ages out of Redis does so on
// a TTL nobody observes.
type nodeStats struct {
	ActiveConnections  int64 `json:"active_connections"`
	TotalPushes        int64 `json:"total_pushes"`
	DeliveredLive      int64 `json:"delivered_live"`
	DeliveredOnConnect int64 `json:"delivered_on_connect"`
	Stored             int64 `json:"stored"`
	Coalesced          int64 `json:"coalesced"`
	Discarded          int64 `json:"discarded"`
	ClaimMisses        int64 `json:"claim_misses"`
	RedisErrors        int64 `json:"redis_errors"`
}

func (s *nodeStats) add(o nodeStats) {
	s.ActiveConnections += o.ActiveConnections
	s.TotalPushes += o.TotalPushes
	s.DeliveredLive += o.DeliveredLive
	s.DeliveredOnConnect += o.DeliveredOnConnect
	s.Stored += o.Stored
	s.Coalesced += o.Coalesced
	s.Discarded += o.Discarded
	s.ClaimMisses += o.ClaimMisses
	s.RedisErrors += o.RedisErrors
}

// statsResponse reports this instance's counters and the totals across every
// instance that has flushed recently.
type statsResponse struct {
	NodeID  string    `json:"node_id"`
	Nodes   int       `json:"nodes"`
	Node    nodeStats `json:"node"`
	Cluster nodeStats `json:"cluster"`
}

// statsHandler serves GET /stats for watching a load test.
func statsHandler(w http.ResponseWriter, r *http.Request, c *coordinator) {
	w.Header().Set("Content-Type", "application/json")
	stats := c.Stats(r.Context())
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(stats)
	if err != nil {
		http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
		return
	}
}

// memStatsResponse reports what the Go runtime knows it is using, the only
// trustworthy input to "what does a connection cost". RSS is not: on macOS,
// pages already handed back stay counted until something else needs them,
// which overstated a 40k-connection run by 3x. Divide by active_connections.
type memStatsResponse struct {
	Goroutines int    `json:"goroutines"`   // ~1 per live SSE stream, plus a handful of runtime and accept goroutines
	HeapBytes  uint64 `json:"heap_bytes"`   // heap objects; includes uncollected garbage unless ?gc=1
	StackBytes uint64 `json:"stack_bytes"`  // goroutine stacks
	InUseBytes uint64 `json:"in_use_bytes"` // everything the runtime holds minus what it has released to the OS
}

func memStatsHandler(w http.ResponseWriter, r *http.Request) {
	samples := []metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(samples)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(memStatsResponse{
		Goroutines: runtime.NumGoroutine(),
		HeapBytes:  samples[0].Value.Uint64(),
		StackBytes: samples[1].Value.Uint64(),
		InUseBytes: samples[2].Value.Uint64() - samples[3].Value.Uint64(),
	}); err != nil {
		http.Error(w, "Failed to encode memstats", http.StatusInternalServerError)
		return
	}
}

// apnsPushError writes the error shape real APNs returns: an apns-id header
// and a {"reason":...} JSON body. The body is load-bearing — buford's
// parseErrorResponse surfaces a JSON-decode error instead of the reason on
// anything else. A 410 Unregistered, if ever modeled, must add a "timestamp"
// field; Apple omits it on every other error.
func apnsPushError(w http.ResponseWriter, headers *pushHeaders, err error) {
	w.Header().Set("Content-Type", "application/json")
	if headers != nil {
		w.Header().Set("apns-id", headers.PushID)
	}
	statusCode := http.StatusBadRequest
	if statuser, ok := err.(Statuser); ok {
		statusCode = statuser.Status()
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(struct {
		Reason string `json:"reason"`
	}{Reason: err.Error()})
}

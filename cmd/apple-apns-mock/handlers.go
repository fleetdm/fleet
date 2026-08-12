package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// eventsSSEHandler serves GET /events?token=<hex> — the simulated device's
// stand-in for a real device's persistent APNS courier connection. Simulated
// clients (osquery-perf/mdmtest, which derive their tokens as
// hex("token"+serial), see pkg/mdm/mdmtest/apple.go TokenUpdate) hold this
// SSE stream open and treat each event as an MDM wake-up.
//
// Contract: pushes arrive as `event: ping` with the data line carrying the
// exact payload Fleet posted; a pending stored ping is delivered immediately
// on connect; `: keepalive` comment lines (ignored by SSE clients) flow on a
// configurable interval so LBs/proxies don't reap idle streams. A newer
// connection for the same token replaces this one (newest wins, matching a
// real device reconnecting) and the replaced stream ends. Token validation
// must happen before the first Flush — flushing commits a 200, making a
// later error status a no-op.
//
// Every write carries a writeTimeout deadline. The server sets no
// WriteTimeout (SSE streams must not be reaped), so without a per-write
// deadline a device that stops reading would block this handler forever:
// unsubscribe would never run, and store.push would keep coalescing wake-ups
// into a connection that can never deliver them instead of storing them.
func eventsSSEHandler(w http.ResponseWriter, r *http.Request, st *store, logger *slog.Logger, keepAlive, writeTimeout time.Duration) {
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	flusher.Flush()

	sub, pending := st.subscribe(token)
	defer st.unsubscribe(token, sub)

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

	if pending != nil {
		// stored ping delivered immediately on connect
		if err := write(pingEvent(pending.payload)); err != nil {
			logger.DebugContext(r.Context(), "failed to write pending ping", "token", token, "error", err)
			st.restore(token, sub, pending, true)
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
				logger.DebugContext(r.Context(), "failed to write ping", "token", token, "error", err)
				st.restore(token, sub, p, false)
				return
			}
		case <-sub.replaced: // a newer connection took our token — stand down
			return
		case <-r.Context().Done(): // client went away
			return
		case <-keepAliveC: // keepalive
			if err := write(": keepalive\n\n"); err != nil {
				logger.DebugContext(r.Context(), "failed to write keepalive", "token", token, "error", err)
				return
			}
		}
	}
}

// pingEvent frames one push payload as an SSE event. SSE is newline
// delimited, so a payload containing a newline is split across several data:
// lines (clients rejoin them with "\n"); emitting it raw would end the event
// early and corrupt every frame after it. The mock forwards payload bytes
// verbatim, and PushMagic reaches it from the device's TokenUpdate, so the
// payload cannot be assumed newline-free.
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

// pushHandler serves POST /3/device/{token} — the same endpoint shape as
// api.push.apple.com. It accepts exactly what Fleet's buford client sends
// today (server/mdm/nanomdm/push/buford): a raw JSON body {"mdm":"<magic>"}
// and NO apns-* headers, responding 200 with an apns-id header and empty
// body. Spec headers are honored when present (see parsePushHeaders), and
// APNS payload limits apply: empty → 400 PayloadEmpty, >4096 bytes → 413
// PayloadTooLarge.
//
// The payload is stored verbatim, not parsed — the mock forwards bytes.
// Token validation is deliberately looser than real APNS: any even-length
// hex is accepted (Apple also enforces the 32-byte token length — a 14-byte
// hex token draws 400 BadDeviceToken from api.push.apple.com — but
// mdmtest/osquery-perf tokens are variable-length hex("token"+serial)).
func pushHandler(w http.ResponseWriter, r *http.Request, st *store, logger *slog.Logger) {
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

	expiration := time.Now().Add(st.defaultTTL)
	if headers.Expiration != nil {
		expiration = *headers.Expiration
	}
	st.push(token, payload, expiration)

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

// parsePushHeaders validates the modeled apns-* headers, mirroring real APNS
// behavior for each (see pushHeaders field comments for per-header
// semantics). The returned headers are always non-nil, including on error, so
// the caller can echo the client's apns-id on the error response — real APNS
// sets apns-id on errors too, and it is how a request is correlated with its
// response. A malformed apns-id is the one exception: it cannot be echoed, so
// the generated one stands.
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

type statsResponse struct {
	ActiveConnections  int `json:"active_connections"`
	TotalPushes        int `json:"total_pushes"`
	DeliveredLive      int `json:"delivered_live"`
	DeliveredOnConnect int `json:"delivered_on_connect"`
	Stored             int `json:"stored"`
	Coalesced          int `json:"coalesced"`
	Expired            int `json:"expired"`
}

// statsHandler serves GET /stats — the store's counters as JSON, for
// watching a load test (connected clients, delivered vs stored vs coalesced
// vs expired pushes).
func statsHandler(w http.ResponseWriter, _ *http.Request, st *store) {
	w.Header().Set("Content-Type", "application/json")
	stats := statsResponse{
		ActiveConnections:  int(st.connected.Load()),
		TotalPushes:        int(st.pushesReceived.Load()),
		DeliveredLive:      int(st.deliveredLive.Load()),
		DeliveredOnConnect: int(st.deliveredOnConnect.Load()),
		Stored:             int(st.stored.Load()),
		Coalesced:          int(st.coalesced.Load()),
		Expired:            int(st.expired.Load()),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err := enc.Encode(stats)
	if err != nil {
		http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
		return
	}
}

// apnsPushError writes an error response in the exact shape real APNS
// returns (captured via tools/mdm/apple/apnspush -direct):
//
//	HTTP/2.0 400 Bad Request
//	Apns-Id: 4FCEA7C9-78CC-0A03-2902-3473E54F9ED4
//	{"reason":"BadDeviceToken"}
//
// The JSON body is load-bearing: buford's parseErrorResponse (the client
// `fleet serve` uses) surfaces a JSON-decode error instead of the reason on
// anything else. apns-id is set on errors too, matching Apple. If the mock
// ever models 410 Unregistered, that response must add a "timestamp" field
// (unix millis when the token died) — Apple omits it on all other errors.
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
	fmt.Fprintf(w, `{"reason":"%s"}`, err.Error())
}

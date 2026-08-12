package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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
func eventsSSEHandler(w http.ResponseWriter, r *http.Request, st *store, logger *slog.Logger, keepAlive time.Duration) {
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

	writeEvent := func(w http.ResponseWriter, payload []byte) {
		fmt.Fprintf(w, "event: ping\n")
		fmt.Fprintf(w, "data: %s\n\n", payload)
	}

	if pending != nil {
		writeEvent(w, pending) // stored ping delivered immediately on connect
		flusher.Flush()
	}

	var keepAliveC <-chan time.Time
	if keepAlive > 0 {
		ticker := time.NewTicker(keepAlive)
		defer ticker.Stop()
		keepAliveC = ticker.C
	}

	for {
		select {
		case payload := <-sub.ch: // push arrived while we're live
			writeEvent(w, payload)
			flusher.Flush()
		case <-sub.replaced: // a newer connection took our token — stand down
			return
		case <-r.Context().Done(): // client went away
			return
		case <-keepAliveC: // keepalive
			if keepAliveC != nil {
				fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}
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
	headers, err := parsePushHeaders(r)
	if err != nil {
		apnsPushError(w, &pushHeaders{
			PushID: uuid.NewString(), // for reporting APNS-ID in the response.
		}, err)
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
// semantics).
func parsePushHeaders(r *http.Request) (*pushHeaders, error) {
	pushHeaders := &pushHeaders{
		PushID: uuid.NewString(), // default to random UUID, if provided it will be overwritten.
	}

	if pushID := r.Header.Get("apns-id"); pushID != "" {
		if _, err := uuid.Parse(pushID); err != nil {
			return nil, BadMessageIdError()
		}
		pushHeaders.PushID = pushID
	}

	pushHeaders.PushType = r.Header.Get("apns-push-type")
	if pushHeaders.PushType != "" && pushHeaders.PushType != "mdm" {
		return nil, InvalidPushTypeError()
	}

	if expiration := r.Header.Get("apns-expiration"); expiration != "" {
		if ts, err := strconv.ParseInt(expiration, 10, 64); err == nil {
			t := time.Unix(ts, 0)
			pushHeaders.Expiration = &t
		} else {
			return nil, BadExpirationDateError()
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

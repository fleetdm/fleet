package apnsmock

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// Package apnsmock provides the client half of Fleet's mock APNS service
// (cmd/apple-apns-mock). Simulated MDM devices (osquery-perf agents, mdmtest
// clients) use Client as their stand-in for a real device's persistent APNS
// courier connection: they subscribe with their device token and receive
// each push Fleet sends as a Ping, which should trigger an MDM check-in

// Ping is one push notification delivered by the mock APNS server. It is a
// pure wake-up signal — like a real APNS MDM push, it carries no data beyond
// proof that the MDM server wants a check-in.
type Ping struct {
	PushMagic  string    // parsed from the {"mdm":"<magic>"} payload; empty for any other shape
	Raw        []byte    // exact payload Fleet posted to /3/device/<token>, verbatim
	ReceivedAt time.Time // when this client received the event
}

// Client is a long-lived SSE subscriber for one device token. It connects to
// the mock server's GET /events?token=<hex> endpoint and delivers each
// `event: ping` on the Pings channel.
//
// Delivery semantics match the server's live path: the channel is buffered 1
// and newer pings are dropped while it is full (a wake-up carries no unique
// data, so one queued ping is as good as many). Lost pings are recovered at
// the system level — the server stores-and-forwards while the client is
// disconnected.
//
// A Client is single-use: call Start exactly once; the run loop reconnects
// with jittered exponential backoff until the context is done, then closes
// the Pings channel.
type Client struct {
	baseURL string // e.g. "http://apns-mock:8378", no trailing slash
	token   string // lowercase hex device token, e.g. hex("token"+serial) for mdmtest clients

	httpClient    *http.Client
	backoffMin    time.Duration
	backoffMax    time.Duration
	initialJitter time.Duration
	logf          func(format string, args ...any)

	pings chan Ping
}

// Option configures a Client.
type Option func(*Client)

// WithBackoff sets the reconnect backoff bounds (default 1s..30s). The delay
// doubles from lo to hi with each consecutive failure and resets to lo after
// a successfully received event.
func WithBackoff(lo, hi time.Duration) Option {
	return func(c *Client) {
		c.backoffMin = lo
		c.backoffMax = hi
	}
}

// WithInitialJitter delays the first connection attempt by a random duration
// in [0, d). A 300k-agent load test starting all clients at once would
// otherwise stampede the mock server with simultaneous connects.
func WithInitialJitter(d time.Duration) Option {
	return func(c *Client) { c.initialJitter = d }
}

// WithLogf routes connection lifecycle logs (connect failures, reconnects)
// somewhere visible; the default discards them, since at load-test scale
// 300k clients logging reconnects would drown everything.
func WithLogf(f func(format string, args ...any)) Option {
	return func(c *Client) { c.logf = f }
}

// NewClient prepares a subscriber for the given mock server base URL and
// hex device token. It does not connect; call Start.
func NewClient(baseURL, deviceToken string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      strings.ToLower(deviceToken),
		httpClient: fleethttp.NewClient(), // deliberately no timeout: SSE streams are long-lived
		backoffMin: time.Second,
		backoffMax: 30 * time.Second,
		logf:       func(string, ...any) {},
		pings:      make(chan Ping, 1),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Pings returns the channel wake-up pings are delivered on. It is closed
// after Start's context is done, so consumers can `for range c.Pings()`.
func (c *Client) Pings() <-chan Ping { return c.pings }

// Start launches the connect/reconnect loop in its own goroutine and returns
// immediately. It never reports an error — every failure is retried with
// backoff until ctx is done, at which point the Pings channel is closed.
func (c *Client) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Client) run(ctx context.Context) {
	defer close(c.pings)

	sleep := func(d time.Duration) {
		if d <= 0 {
			return
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}

	// hold the registration for jitter to avoid stampeding the server
	if c.initialJitter > 0 {
		sleep(time.Duration(rand.IntN(int(c.initialJitter)))) // nolint:gosec // weak rand is fine for jitter
	}

	for backoff := c.backoffMin; ctx.Err() == nil; backoff = min(backoff*2, c.backoffMax) {
		err := c.connectAndStream(ctx)
		if err == nil {
			backoff = c.backoffMin // reset on a successful stream
			continue
		}
		c.logf("apnsmock client: %v", err)
		jitter := time.Duration(0)
		if half := backoff / 2; half > 0 {
			jitter = time.Duration(rand.Int64N(int64(half))) // nolint:gosec // weak rand is fine for jitter
		}
		sleep(backoff + jitter)
	}
}

// connectAndStream opens one SSE connection and consumes it until it breaks,
// delivering each ping. It returns on any failure — connect error, non-200
// (the caller retries; the mock may be restarting), or stream end (EOF when
// a newer connection for the token replaces this one).
func (c *Client) connectAndStream(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/events?token="+c.token, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apnsmock client: GET %s returned %d", httpReq.URL, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 512), 8192)

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, ":"):
			continue // keepalive comment
		case strings.HasPrefix(line, "data: "):
			c.deliver([]byte(line[len("data: "):]))
		case strings.HasPrefix(line, "event:"):
			continue // event type line; pings are the only event
		case line == "":
			continue // event boundary
		default:
			return fmt.Errorf("apnsmock client: unexpected line %q", line)
		}
	}
	return scanner.Err()
}

// deliver hands a ping to the consumer without ever blocking the read loop:
// if the buffer already holds a ping, the new one is dropped (coalescing —
// same rule as the server's live-delivery path).
func (c *Client) deliver(raw []byte) {
	select {
	case c.pings <- Ping{PushMagic: pushMagic(raw), Raw: raw, ReceivedAt: time.Now()}:
	default:
	}
}

// pushMagic extracts the PushMagic from an MDM push payload
// ({"mdm":"<magic>"} — the only payload Fleet ever sends). Any other shape
// yields "" and the ping is still delivered with Raw intact.
func pushMagic(raw []byte) string {
	var p struct {
		MDM string `json:"mdm"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return ""
	}
	return p.MDM
}

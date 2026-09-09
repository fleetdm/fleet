package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// coordinator makes a set of instances behave like one server: Fleet may push
// to any of them and it reaches whichever holds the device's stream.
//
//	push       SET pending:<token> <ping> GET PX <ttl>, then PUBLISH <token>
//	announce   every node: holds it? -> GETDEL pending:<token> -> write it
//	connect    GETDEL pending:<token>
//
// Storing before announcing is what makes "nobody holds this device" a
// non-event: the key waits until it connects somewhere. GETDEL is what keeps
// delivery exactly-once when two nodes briefly hold the same token.
type coordinator struct {
	reg    *registry
	pool   fleet.RedisPool
	logger *slog.Logger
	cfg    coordinatorConfig
}

type coordinatorConfig struct {
	NodeID     string        // this node's label in cluster stats
	KeyPrefix  string        // namespaces every key and the channel
	DefaultTTL time.Duration // retention when the push carries no apns-expiration
}

func newCoordinator(pool fleet.RedisPool, reg *registry, logger *slog.Logger, cfg coordinatorConfig) *coordinator {
	return &coordinator{reg: reg, pool: pool, logger: logger, cfg: cfg}
}

func (c *coordinator) pendingKey(token string) string { return c.cfg.KeyPrefix + "pending:" + token }
func (c *coordinator) statsKey(node string) string    { return c.cfg.KeyPrefix + "stats:" + node }
func (c *coordinator) statsPattern() string           { return c.cfg.KeyPrefix + "stats:*" }
func (c *coordinator) channel() string                { return c.cfg.KeyPrefix + "push" }

// pushMsg announces a push, or a token changing hands.
type pushMsg struct {
	Token string `json:"t"`
	// Inline is set only for a deliver-now-or-discard push, which is never
	// stored and so cannot be claimed.
	Inline []byte `json:"p,omitempty"` // base64 in JSON
	// Owner, when set, means that node has just taken the token: every other
	// node drops its stream for it. Eviction is otherwise registry-local, so
	// without this a device that reconnects elsewhere leaves a live stream
	// behind and pushes split between the two at random. Seq orders takeovers,
	// since announcements can arrive after a later one.
	Owner string `json:"o,omitempty"`
	Seq   int64  `json:"s,omitempty"`
}

// encodePing prefixes the payload with its expiry (8 bytes, big-endian unix
// milliseconds) so a claimed ping can be put back with the time it had left —
// a Redis TTL is opaque to whoever reads the value.
func encodePing(p *ping) []byte {
	buf := make([]byte, 8+len(p.payload))
	binary.BigEndian.PutUint64(buf[:8], uint64(p.expiresAt.UnixMilli())) //nolint:gosec // timestamps are positive
	copy(buf[8:], p.payload)
	return buf
}

func decodePing(b []byte) (*ping, error) {
	if len(b) < 8 {
		return nil, fmt.Errorf("pending value too short: %d bytes", len(b))
	}
	return &ping{
		payload:   b[8:],
		expiresAt: time.UnixMilli(int64(binary.BigEndian.Uint64(b[:8]))), //nolint:gosec // round-trips encodePing
	}, nil
}

// Push stores a push and announces it. An error means it was not accepted, so
// the handler answers 503 and Fleet's pending-hosts cron retries.
func (c *coordinator) Push(ctx context.Context, token string, payload []byte, expiresAt time.Time) error {
	c.reg.pushesReceived.Add(1)

	p := &ping{payload: payload, expiresAt: expiresAt}
	if p.expiresAt.IsZero() {
		p.expiresAt = time.Now().Add(c.cfg.DefaultTTL)
	}

	// Already expired means deliver-now-or-discard, so nothing is stored and
	// the payload rides along for whoever holds the stream.
	if !p.expiresAt.After(time.Now()) {
		c.reg.discarded.Add(1)
		return c.publish(ctx, pushMsg{Token: token, Inline: payload})
	}

	if err := c.storePending(ctx, token, p); err != nil {
		return err
	}
	return c.publish(ctx, pushMsg{Token: token})
}

// storePending writes the token's single pending ping. Overwriting an older
// one is APNs coalescing, and SET's GET option reports it in the same trip.
func (c *coordinator) storePending(ctx context.Context, token string, p *ping) error {
	ttl := time.Until(p.expiresAt)
	if ttl <= 0 {
		return nil // expired in the meantime
	}

	conn := c.pool.Get()
	defer conn.Close()

	old, err := redigo.Bytes(conn.Do("SET", c.pendingKey(token), encodePing(p), "GET", "PX", ttl.Milliseconds()))
	switch {
	case err != nil && !errors.Is(err, redigo.ErrNil):
		c.reg.redisErrors.Add(1)
		c.logger.ErrorContext(ctx, "store pending push", "token", token, "error", err)
		return err
	case old != nil:
		c.reg.coalesced.Add(1)
	default:
		c.reg.stored.Add(1)
	}
	return nil
}

// requeue puts back a ping whose stream ended before it reached the wire, so
// the device gets it wherever it reconnects.
func (c *coordinator) requeue(ctx context.Context, token string, p *ping) error {
	if err := c.storePending(ctx, token, p); err != nil {
		c.logger.ErrorContext(ctx, "requeue undelivered ping", "token", token, "error", err)
		return err
	}
	return nil
}

func (c *coordinator) publish(ctx context.Context, msg pushMsg) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	conn := c.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PUBLISH", c.channel(), body); err != nil {
		c.reg.redisErrors.Add(1)
		c.logger.ErrorContext(ctx, "publish push", "token", msg.Token, "error", err)
		return err
	}
	return nil
}

// claim atomically takes the token's pending ping, so only one node delivers
// it. nil means there was nothing pending.
func (c *coordinator) claim(ctx context.Context, token string) *ping {
	conn := c.pool.Get()
	defer conn.Close()

	b, err := redigo.Bytes(conn.Do("GETDEL", c.pendingKey(token)))
	if err != nil {
		if !errors.Is(err, redigo.ErrNil) {
			c.reg.redisErrors.Add(1)
			c.logger.ErrorContext(ctx, "claim pending push", "token", token, "error", err)
		}
		return nil
	}
	p, err := decodePing(b)
	if err != nil {
		c.logger.ErrorContext(ctx, "decode pending push", "token", token, "error", err)
		return nil
	}
	return p
}

// Subscribe registers a device's stream here and returns whatever was waiting
// for it, wherever that push was received.
func (c *coordinator) Subscribe(ctx context.Context, token string) (*subscriber, *ping) {
	seq := c.nextSeq(ctx)
	sub, evicted := c.reg.subscribe(token, seq)

	// Put back what the replaced connection never wrote, so the claim below
	// picks it up (or a newer push that overwrote it in the meantime).
	if evicted != nil {
		c.reg.deliveredLive.Add(-1)
		_ = c.requeue(ctx, token, evicted)
	}

	p := c.claim(ctx, token)
	if p != nil {
		c.reg.deliveredOnConnect.Add(1)
	}

	// Tell the other instances this token is ours now.
	if err := c.publish(ctx, pushMsg{Token: token, Owner: c.cfg.NodeID, Seq: seq}); err != nil {
		c.logger.ErrorContext(ctx, "announce token ownership", "token", token, "error", err)
	}

	// Counted only now that the claim is done: a push announced while this
	// connect was still claiming can be taken by either path, so a stream
	// that shows up in the gauge has to be one a later push reaches live.
	c.reg.connected.Add(1)
	return sub, p
}

// nextSeq issues a cluster-wide order for a subscription. A local clock would
// not do: takeovers are compared across instances. Zero on failure, which
// makes the announcement a no-op rather than evicting the wrong stream.
func (c *coordinator) nextSeq(ctx context.Context) int64 {
	conn := c.pool.Get()
	defer conn.Close()

	seq, err := redigo.Int64(conn.Do("INCR", c.cfg.KeyPrefix+"seq"))
	if err != nil {
		c.reg.redisErrors.Add(1)
		c.logger.ErrorContext(ctx, "issue subscription sequence", "error", err)
		return 0
	}
	return seq
}

// Unsubscribe releases the stream and puts back a wake-up the device dropped
// before reading.
func (c *coordinator) Unsubscribe(ctx context.Context, token string, sub *subscriber) {
	// Always decrements: the handler calls this exactly once per Subscribe.
	c.reg.connected.Add(-1)
	evicted := c.reg.unsubscribe(token, sub)
	if evicted == nil {
		return
	}
	c.reg.deliveredLive.Add(-1)
	_ = c.requeue(ctx, token, evicted)
}

// Restore puts back a ping the handler took but failed to write, unless a
// newer connection has since taken the token — its own wake-ups supersede it.
func (c *coordinator) Restore(ctx context.Context, token string, sub *subscriber, p *ping, onConnect bool) {
	if !c.reg.isCurrent(token, sub) {
		return
	}
	if onConnect {
		c.reg.deliveredOnConnect.Add(-1)
	} else {
		c.reg.deliveredLive.Add(-1)
	}
	_ = c.requeue(ctx, token, p)
}

// dispatch handles one announcement: a push, or another node taking a token.
func (c *coordinator) dispatch(ctx context.Context, msg pushMsg) {
	if msg.Owner != "" {
		if msg.Owner != c.cfg.NodeID {
			c.evictTo(ctx, msg.Token, msg.Owner, msg.Seq)
		}
		return
	}
	if !c.reg.holds(msg.Token) {
		return
	}

	p := &ping{payload: msg.Inline}
	inline := msg.Inline != nil
	if !inline {
		if p = c.claim(ctx, msg.Token); p == nil {
			c.reg.claimMisses.Add(1)
			return
		}
	}

	if c.reg.deliver(msg.Token, p) == notHeld && !inline {
		// The stream closed between holds and deliver. An inline push is
		// deliver-or-discard, so only a claimed one goes back.
		_ = c.requeue(ctx, msg.Token, p)
	}
}

// evictTo releases a token another node has taken. Anything the dropped
// stream never wrote goes back to Redis and is re-announced, so the new owner
// picks it up instead of waiting for the next push.
func (c *coordinator) evictTo(ctx context.Context, token, owner string, seq int64) {
	evicted := c.reg.evictIfOlder(token, seq)
	if evicted == nil {
		return
	}
	c.logger.DebugContext(ctx, "token taken over by another node", "token", token, "owner", owner)
	c.reg.deliveredLive.Add(-1)
	if err := c.requeue(ctx, token, evicted); err != nil {
		return
	}
	if err := c.publish(ctx, pushMsg{Token: token}); err != nil {
		c.logger.ErrorContext(ctx, "re-announce ping after takeover", "token", token, "error", err)
	}
}

// Run drives the subscription and the stats flush until ctx is cancelled.
func (c *coordinator) Run(ctx context.Context, statsInterval time.Duration) {
	go c.flushStatsLoop(ctx, statsInterval)
	c.subscribeLoop(ctx)
}

// subscribeLoop keeps the broadcast subscription alive for the life of the
// process, reconnecting with backoff. Only one channel is involved, so
// re-establishing it is O(1) however many devices this node holds.
func (c *coordinator) subscribeLoop(ctx context.Context) {
	boff := backoff.NewExponentialBackOff()
	boff.MaxElapsedTime = 0 // never give up

	for ctx.Err() == nil {
		err := c.subscribeOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		c.logger.ErrorContext(ctx, "push subscription dropped, reconnecting", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(boff.NextBackOff()):
		}
	}
}

// subscribeOnce owns one Redis connection for the life of one subscription,
// returning when it dies.
func (c *coordinator) subscribeOnce(ctx context.Context) error {
	conn := c.pool.Get()
	psc := &redigo.PubSubConn{Conn: conn}
	if err := psc.Subscribe(c.channel()); err != nil {
		conn.Close()
		return err
	}
	c.logger.InfoContext(ctx, "subscribed to push channel", "channel", c.channel())

	// Receive blocks, so it needs its own goroutine to stay cancellable.
	msgs := make(chan any, 64)
	stop := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			msg := psc.ReceiveWithTimeout(1 * time.Hour)
			select {
			case msgs <- msg:
			case <-stop:
				return
			}
			switch m := msg.(type) {
			case error:
				return
			case redigo.Subscription:
				if m.Count == 0 {
					return
				}
			}
		}
	}()

	// Closing the connection under a blocked Receive is a data race, so the
	// reader has to be gone first. Unsubscribe is a write, which redigo allows
	// concurrently with a read, and it makes the blocked Receive return.
	defer func() {
		close(stop)
		_ = psc.Unsubscribe(c.channel())
		select {
		case <-readerDone:
		case <-time.After(5 * time.Second):
			c.logger.WarnContext(ctx, "pub/sub reader did not exit; closing anyway")
		}
		_ = conn.Close()
	}()

	c.resync(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case raw, ok := <-msgs:
			if !ok {
				return errors.New("pub/sub receive loop exited")
			}
			switch m := raw.(type) {
			case redigo.Message:
				var msg pushMsg
				if err := json.Unmarshal(m.Data, &msg); err != nil {
					c.logger.ErrorContext(ctx, "malformed push announcement", "error", err)
					continue
				}
				c.dispatch(ctx, msg)
			case error:
				return m
			}
		}
	}
}

// resync claims anything that piled up for this node's devices while the
// subscription was down. Pending keys survive the gap, so it only has to look.
func (c *coordinator) resync(ctx context.Context) {
	tokens := c.reg.tokens()
	if len(tokens) == 0 {
		return
	}
	c.logger.InfoContext(ctx, "resyncing pending pushes after reconnect", "tokens", len(tokens))

	const batch = 1000
	var recovered int
	for start := 0; start < len(tokens); start += batch {
		if ctx.Err() != nil {
			return
		}
		recovered += c.resyncBatch(tokens[start:min(start+batch, len(tokens))])
	}
	if recovered > 0 {
		c.logger.InfoContext(ctx, "recovered pending pushes after reconnect", "count", recovered)
	}
}

func (c *coordinator) resyncBatch(tokens []string) int {
	conn := c.pool.Get()
	defer conn.Close()

	for _, token := range tokens {
		if err := conn.Send("GETDEL", c.pendingKey(token)); err != nil {
			c.reg.redisErrors.Add(1)
			return 0
		}
	}
	if err := conn.Flush(); err != nil {
		c.reg.redisErrors.Add(1)
		return 0
	}

	var recovered int
	for _, token := range tokens {
		b, err := redigo.Bytes(conn.Receive())
		if err != nil {
			if !errors.Is(err, redigo.ErrNil) {
				c.reg.redisErrors.Add(1)
			}
			continue
		}
		p, err := decodePing(b)
		if err != nil {
			continue
		}
		if c.reg.deliver(token, p) == delivered {
			recovered++
		}
	}
	return recovered
}

// flushStatsLoop publishes this node's counters so any instance can report
// cluster-wide totals. The key has a TTL, so a node that goes away drops out.
func (c *coordinator) flushStatsLoop(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.flushStats(ctx, every*4)
		}
	}
}

func (c *coordinator) flushStats(ctx context.Context, ttl time.Duration) {
	body, err := json.Marshal(c.localStats())
	if err != nil {
		return
	}
	conn := c.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("SET", c.statsKey(c.cfg.NodeID), body, "PX", ttl.Milliseconds()); err != nil {
		c.reg.redisErrors.Add(1)
		c.logger.ErrorContext(ctx, "flush stats", "error", err)
	}
}

func (c *coordinator) localStats() nodeStats {
	return nodeStats{
		ActiveConnections:  c.reg.connected.Load(),
		TotalPushes:        c.reg.pushesReceived.Load(),
		DeliveredLive:      c.reg.deliveredLive.Load(),
		DeliveredOnConnect: c.reg.deliveredOnConnect.Load(),
		Stored:             c.reg.stored.Load(),
		Coalesced:          c.reg.coalesced.Load(),
		Discarded:          c.reg.discarded.Load(),
		ClaimMisses:        c.reg.claimMisses.Load(),
		RedisErrors:        c.reg.redisErrors.Load(),
	}
}

// Stats reports this node's counters plus the sum across every node that has
// flushed recently, so a load test can be read off any one instance.
func (c *coordinator) Stats(ctx context.Context) statsResponse {
	resp := statsResponse{NodeID: c.cfg.NodeID, Node: c.localStats()}

	keys, err := redis.ScanKeys(c.pool, c.statsPattern(), 100)
	if err != nil {
		c.logger.ErrorContext(ctx, "scan node stats", "error", err)
		resp.Cluster, resp.Nodes = resp.Node, 1
		return resp
	}

	conn := c.pool.Get()
	defer conn.Close()

	self := c.statsKey(c.cfg.NodeID)
	for _, key := range keys {
		if key == self { // prefer live counters over our own last snapshot
			resp.Cluster.add(resp.Node)
			resp.Nodes++
			continue
		}
		b, err := redigo.Bytes(conn.Do("GET", key))
		if err != nil {
			continue
		}
		var s nodeStats
		if err := json.Unmarshal(b, &s); err != nil {
			continue
		}
		resp.Cluster.add(s)
		resp.Nodes++
	}
	if resp.Nodes == 0 { // our first flush has not landed yet
		resp.Cluster, resp.Nodes = resp.Node, 1
	}
	return resp
}

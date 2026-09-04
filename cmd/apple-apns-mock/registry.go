package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ping is one push awaiting delivery. The payload is whatever Fleet posted,
// kept verbatim; expiresAt rides along so a requeued ping keeps its original
// expiry rather than getting a fresh TTL.
type ping struct {
	payload   []byte
	expiresAt time.Time
}

// subscriber is one live SSE stream.
type subscriber struct {
	ch       chan *ping    // buffered 1: a queued wake-up makes a second redundant
	replaced chan struct{} // closed when a newer connection takes over the token
}

// entry is the per-token state on this node. Pending pushes live in Redis, not
// here, so any node can serve a device that reconnects elsewhere.
type entry struct {
	sub *subscriber
	seq int64 // Redis-issued order of this subscription; see evictIfOlder
}

type shard struct {
	sync.Mutex
	entries map[string]*entry
}

// counters are this node's tallies, reported by GET /stats and flushed to
// Redis for cluster-wide totals.
type counters struct {
	connected          atomic.Int64 // currently connected streams
	pushesReceived     atomic.Int64 // pushes accepted over HTTP
	deliveredLive      atomic.Int64 // pings written to a stream on this node
	deliveredOnConnect atomic.Int64 // pings claimed when a device connected here
	stored             atomic.Int64 // pending keys written
	coalesced          atomic.Int64 // pushes that overwrote a pending one or hit a full buffer
	discarded          atomic.Int64 // pushes that arrived already expired
	claimMisses        atomic.Int64 // announced pushes another node claimed first
	redisErrors        atomic.Int64
}

// registry tracks which device tokens have a live SSE stream on this node.
// Sharded 256 ways because ramp-up is ~100k concurrent map writes.
type registry struct {
	counters
	shards [256]shard
}

// newRegistry allocates the shard maps up front; writing to a nil map panics.
func newRegistry() *registry {
	r := &registry{}
	for i := range r.shards {
		r.shards[i].entries = make(map[string]*entry)
	}
	return r
}

// subscribe registers a stream for the token, evicting any previous one
// (newest wins, matching a device that reconnects after a blip). Returns a
// ping the evicted stream never wrote, if any, for the caller to put back in
// Redis — returning it rather than storing it keeps Redis calls out of the
// shard lock.
func (r *registry) subscribe(token string, seq int64) (*subscriber, *ping) {
	sub := &subscriber{
		ch:       make(chan *ping, 1),
		replaced: make(chan struct{}),
	}

	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil {
		e = &entry{}
		sh.entries[token] = e
	}

	var evicted *ping
	if e.sub != nil {
		close(e.sub.replaced)
		evicted = drain(e.sub)
	}
	e.sub = sub
	e.seq = seq
	return sub, evicted
}

// unsubscribe releases the stream and returns any unwritten ping. It detaches
// only if sub is still the token's current subscriber, so a replaced handler's
// deferred cleanup cannot evict its replacement.
func (r *registry) unsubscribe(token string, sub *subscriber) *ping {
	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil || e.sub != sub {
		return nil // already replaced; subscribe drained us then
	}
	delete(sh.entries, token)
	return drain(sub)
}

// evictIfOlder drops this node's stream for the token because another node
// took it over, returning any ping the stream never wrote. seq is the
// takeover's Redis-issued order: announcements arrive unordered, so an older
// one must not evict a newer stream. The evicted handler's deferred
// unsubscribe finds the entry gone and simply decrements the gauge.
func (r *registry) evictIfOlder(token string, seq int64) *ping {
	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil || e.sub == nil || e.seq >= seq {
		return nil
	}
	close(e.sub.replaced)
	p := drain(e.sub)
	delete(sh.entries, token)
	return p
}

// drain takes the buffered ping, if any. It races the handler's own read of
// sub.ch and exactly one wins, so a ping is written to the wire or handed
// back, never both.
func drain(sub *subscriber) *ping {
	select {
	case p := <-sub.ch:
		return p
	default:
		return nil
	}
}

// deliverResult tells the caller whether it still owns the ping.
type deliverResult int

const (
	delivered  deliverResult = iota // queued for the stream
	notHeld                         // not our token: the caller must put the ping back
	bufferFull                      // a wake-up is already queued, so this one is redundant
)

// deliver hands a ping to the token's stream without blocking.
func (r *registry) deliver(token string, p *ping) deliverResult {
	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil || e.sub == nil {
		return notHeld
	}
	if len(e.sub.ch) == cap(e.sub.ch) {
		r.coalesced.Add(1)
		return bufferFull
	}
	r.deliveredLive.Add(1)
	e.sub.ch <- p // deliver is the only sender and holds the lock, so this cannot block
	return delivered
}

// holds reports whether this node has a live stream for the token. Every
// announced push passes through it, so nodes that don't hold it do no Redis
// work at all.
func (r *registry) holds(token string) bool {
	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	return e != nil && e.sub != nil
}

// isCurrent reports whether sub is still the token's live subscriber.
func (r *registry) isCurrent(token string, sub *subscriber) bool {
	token, sh := r.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	return e != nil && e.sub == sub
}

// tokens lists every token this node holds, for the resync after a pub/sub
// reconnect.
func (r *registry) tokens() []string {
	var out []string
	for i := range r.shards {
		sh := &r.shards[i]
		sh.Lock()
		for token, e := range sh.entries {
			if e.sub != nil {
				out = append(out, token)
			}
		}
		sh.Unlock()
	}
	return out
}

// shardFor returns the token's shard along with the canonical (lowercased) key
// to look it up by: Fleet and the device may disagree on token case.
func (r *registry) shardFor(token string) (string, *shard) {
	token = strings.ToLower(token)
	return token, &r.shards[shardIndex(token)]
}

// shardIndex is a zero-allocation FNV-1a hash mod 256. FNV-1a spreads
// near-identical inputs well, and mdmtest tokens are hex("token"+serial), so
// they share long prefixes.
func shardIndex(token string) int {
	h := uint32(2166136261) // offset basis
	for i := range len(token) {
		h = (h ^ uint32(token[i])) * 16777619 // FNV prime
	}
	return int(h & 0xff)
}

package main

import (
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ping is a single stored push notification awaiting delivery. The payload
// is kept verbatim as Fleet sent it (for MDM always {"mdm":"<PushMagic>"}).
type ping struct {
	payload   []byte
	expiresAt time.Time
}

// subscriber is one live SSE connection for a device token. The channel
// carries the whole ping, not just the payload, so a ping that is buffered
// but never written to the wire can be put back with its original expiry
// (see requeue).
type subscriber struct {
	ch       chan *ping    // buffered 1 — coalescing means one queued ping is enough
	replaced chan struct{} // closed when a newer connection takes over the token
}

// entry is the per-token state: at most one live subscriber and at most one
// pending ping, mirroring APNS, which stores only the most recent
// notification per device.
//
// Invariant: sub != nil implies pending == nil — subscribe always clears
// pending (delivering or expiring it), and push never stores while a
// subscriber is live.
type entry struct {
	sub     *subscriber
	pending *ping
}

type shard struct {
	sync.Mutex
	entries map[string]*entry
}

// store models APNS store-and-forward semantics in memory: pushes to a
// connected device are delivered immediately; pushes to an offline device are
// held (newest wins) until the device connects, the ping expires, or the
// server restarts (real APNS offers no durability guarantee either — Fleet's
// apns_push_to_pending_hosts cron is the system-level retry).
//
// The map is sharded 256 ways because a load test opens ~300k SSE
// connections at ramp-up, each a map write; a single lock would serialize
// them.
type store struct {
	logger             *slog.Logger
	shards             [256]shard // shard = sync.Mutex + map[string]*entry
	defaultTTL         time.Duration
	connected          atomic.Int64 // number of currently connected subscribers
	pushesReceived     atomic.Int64 // number of pushes received (including coalesced)
	deliveredLive      atomic.Int64 // number of pushes delivered to live subscribers
	deliveredOnConnect atomic.Int64 // number of pushes delivered to subscribers on connect (pending)
	stored             atomic.Int64 // number of pushes stored for later delivery (pending)
	coalesced          atomic.Int64 // number of pushes that were coalesced (overwritten) by a later push
	expired            atomic.Int64 // number of pushes that expired before delivery (pending)
}

// newStore initializes all shard maps up front (writing to a nil map
// panics; 256 empty maps cost ~12KB). defaultTTL bounds pending pings when
// no apns-expiration header is given — Fleet's buford client never sends
// one, so in practice this models APNS's ~24h default retention.
func newStore(defaultTTL time.Duration, logger *slog.Logger) *store {
	s := &store{
		defaultTTL: defaultTTL,
		logger:     logger,
	}
	for i := range s.shards {
		s.shards[i].entries = make(map[string]*entry)
	}
	return s
}

// subscribe registers a new subscriber for the token, evicting any previous
// one by closing its replaced channel (newest connection wins — a device
// that reconnects after a network blip owns its token; the old handler's
// deferred unsubscribe cannot evict it, see unsubscribe). It returns the
// pending ping if one exists and is unexpired, clearing it either way
// (expired pings are dropped lazily here, expired++; delivered ones count as
// deliveredOnConnect).
func (s *store) subscribe(token string) (*subscriber, *ping) {
	sub := &subscriber{
		ch:       make(chan *ping, 1),
		replaced: make(chan struct{}),
	}

	token = strings.ToLower(token)
	sh := s.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil {
		e = &entry{}
		sh.entries[token] = e
	}

	// Evict the previous connection, keeping anything it never got to write
	// (requeue runs before the pending hand-off below, so a reconnecting
	// device immediately receives the wake-up its old connection missed).
	if e.sub != nil {
		close(e.sub.replaced)
		s.requeue(e, e.sub)
	}
	e.sub = sub
	s.connected.Add(1)

	// deliver any pending ping
	var pending *ping
	if e.pending != nil {
		if e.pending.expiresAt.After(time.Now()) {
			pending = e.pending
			s.deliveredOnConnect.Add(1)
		} else {
			s.expired.Add(1)
		}
		e.pending = nil
	}
	return sub, pending
}

// requeue puts a ping that was buffered for a subscriber but never written to
// the wire back into the entry's pending slot. push counts a ping as
// deliveredLive the moment it lands in the channel, so if the connection is
// replaced or drops before the handler drains it, the count is wrong and the
// device would never see that wake-up: undo both here.
//
// The receive races the handler's own read of sub.ch, and exactly one of them
// wins, so a ping is either written to the wire or requeued, never both.
// Callers must hold the token's shard lock.
func (s *store) requeue(e *entry, sub *subscriber) {
	select {
	case p := <-sub.ch:
		s.deliveredLive.Add(-1)
		s.storePending(e, p)
	default:
	}
}

// storePending holds a ping for a device that is not connected, applying APNS
// retention rules: a zero expiry means no apns-expiration header was sent and
// gets the server default TTL, while an expiry already in the past means
// deliver-now-or-discard and is dropped (expired++), since there is no
// connection to deliver it to. A ping overwrites any older pending one (APNS
// coalescing, coalesced++), otherwise it counts as stored++. Reports whether
// the ping was kept. Callers must hold the token's shard lock.
func (s *store) storePending(e *entry, p *ping) bool {
	if p.expiresAt.IsZero() {
		p.expiresAt = time.Now().Add(s.defaultTTL)
	} else if !p.expiresAt.After(time.Now()) {
		s.expired.Add(1)
		return false
	}
	if e.pending != nil {
		s.coalesced.Add(1)
	} else {
		s.stored.Add(1)
	}
	e.pending = p
	return true
}

// restore is requeue for a ping the SSE handler already drained but failed to
// write (a broken or stalled connection). It is a no-op once a newer
// connection owns the token, since that connection's own wake-ups supersede
// this one. deliveredOnConnect pings pass onConnect=true so the right counter
// is corrected.
func (s *store) restore(token string, sub *subscriber, p *ping, onConnect bool) {
	token = strings.ToLower(token)
	sh := s.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil || e.sub != sub {
		return
	}
	if onConnect {
		s.deliveredOnConnect.Add(-1)
	} else {
		s.deliveredLive.Add(-1)
	}
	s.storePending(e, p)
}

// push delivers or stores one notification, never blocking and never
// failing — every outcome is a 200 at the APNS protocol level:
//
//   - Live subscriber: synchronous non-blocking send on sub.ch (buffered 1);
//     a full buffer means a wake-up is already queued, so the new ping is
//     dropped (coalesced++). expiresAt is ignored when live — APNS delivers
//     immediately regardless (apns-expiration: 0 means deliver-NOW-or-
//     discard). Delivery is never also stored: APNS does not redeliver
//     already-delivered notifications, and Fleet's pending-hosts cron owns
//     retries for lost wake-ups.
//   - Offline: kept as the token's single pending ping, newest overwriting
//     oldest (APNS coalescing; stored++ for a fresh write, coalesced++ for
//     an overwrite). A zero expiresAt means "no apns-expiration header" →
//     now+defaultTTL; a past expiresAt is discarded without storing
//     (expired++).
func (s *store) push(token string, payload []byte, expiresAt time.Time) {
	token = strings.ToLower(token)
	sh := s.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	s.pushesReceived.Add(1)
	e := sh.entries[token]
	p := &ping{payload: payload, expiresAt: expiresAt}

	// live subscriber, deliver now don't store
	if e != nil && e.sub != nil {
		select {
		case e.sub.ch <- p:
			s.deliveredLive.Add(1)
		default:
			s.coalesced.Add(1)
		}
		return
	}

	// offline device or no subscriber, store as the token's pending ping
	if e == nil {
		e = &entry{}
		sh.entries[token] = e
	}
	if !s.storePending(e, p) && e.sub == nil && e.pending == nil {
		delete(sh.entries, token) // discarded: don't leave an empty entry behind
	}
}

// unsubscribe always decrements the connected gauge (the SSE handler calls
// it exactly once per subscribe), but only detaches the subscriber if it is
// still the token's current one (pointer comparison) — a replaced handler's
// deferred cleanup must not evict its replacement. Entries left with neither
// subscriber nor pending ping are deleted.
func (s *store) unsubscribe(token string, sub *subscriber) {
	token = strings.ToLower(token)

	s.connected.Add(-1)
	sh := s.shardFor(token)
	sh.Lock()
	defer sh.Unlock()

	e := sh.entries[token]
	if e == nil || e.sub != sub {
		// Already replaced: subscribe drained this subscriber when it evicted
		// us, so there is nothing left to requeue.
		return
	}
	e.sub = nil
	s.requeue(e, sub) // the device dropped before reading its last wake-up
	if e.pending == nil {
		delete(sh.entries, token)
	}
}

// sweep drops pending pings that expired before the given time and deletes
// empty entries, bounding memory for tokens that never reconnect (expiry is
// otherwise lazy, on subscribe/push). Entries with a live subscriber always
// survive. Run periodically from main.
func (s *store) sweep(expiresBefore time.Time) {
	for i := range s.shards {
		sh := &s.shards[i]
		sh.Lock()
		for token, e := range sh.entries {
			if e.pending != nil && !e.pending.expiresAt.After(expiresBefore) {
				s.expired.Add(1)
				e.pending = nil
			}
			if e.sub == nil && e.pending == nil {
				delete(sh.entries, token)
			}
		}
		sh.Unlock()
	}
}

func (s *store) shardFor(token string) *shard {
	return &s.shards[shardIndex(token)]
}

func shardIndex(token string) int {
	// zero-allocation FNV-1a hash, then mod by 256 (len(s.shards)) (via &0xff).
	// FNV-1a distributes well even on near-identical inputs — mdmtest tokens
	// are hex("token"+serial), so they share long prefixes.
	h := uint32(2166136261) // FNV-1a offset basis
	for i := range len(token) {
		h = (h ^ uint32(token[i])) * 16777619 // FNV prime
	}
	return int(h & 0xff)
}

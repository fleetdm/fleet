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

// subscriber is one live SSE connection for a device token.
type subscriber struct {
	ch       chan []byte   // buffered 1 — coalescing means one queued ping is enough
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
// pending ping's payload if one exists and is unexpired, clearing it either
// way (expired pings are dropped lazily here, expired++; delivered ones
// count as deliveredOnConnect).
func (s *store) subscribe(token string) (*subscriber, []byte) {
	sub := &subscriber{
		ch:       make(chan []byte, 1),
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

	// evict previous connections
	if e.sub != nil {
		close(e.sub.replaced)
	}
	e.sub = sub
	s.connected.Add(1)

	// deliver any pending ping
	var payload []byte
	if e.pending != nil {
		if e.pending.expiresAt.After(time.Now()) {
			payload = e.pending.payload
			s.deliveredOnConnect.Add(1)
		} else {
			s.expired.Add(1)
		}
		e.pending = nil
	}
	return sub, payload
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
	// live subscriber, deliver now don't store
	if e != nil && e.sub != nil {
		select {
		case e.sub.ch <- payload:
			s.deliveredLive.Add(1)
		default:
			s.coalesced.Add(1)
		}
		return
	}

	// offline device or no subscriber, resolve expiration
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(s.defaultTTL)
	} else if !expiresAt.After(time.Now()) {
		s.expired.Add(1)
		return
	}

	// store or overwrite
	if e == nil {
		e = &entry{}
		sh.entries[token] = e
	}
	if e.pending != nil {
		s.coalesced.Add(1)
	} else {
		s.stored.Add(1)
	}
	e.pending = &ping{
		payload:   payload,
		expiresAt: expiresAt,
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
		return
	}
	e.sub = nil
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
	for i := 0; i < len(token); i++ {
		h = (h ^ uint32(token[i])) * 16777619 // FNV prime
	}
	return int(h & 0xff)
}

package main

// Behavioral spec for the in-memory token store. The store-and-forward,
// coalescing, expiry, and replace-on-reconnect semantics these tests pin are
// documented on the store type and its methods in store.go.

import (
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTTL = time.Hour

// recv asserts a payload is already buffered on the subscriber's channel —
// push delivers synchronously (see store.push), so after push returns there
// is no need to wait.
func recv(t *testing.T, sub *subscriber) []byte {
	t.Helper()
	select {
	case p := <-sub.ch:
		return p
	default:
		t.Fatal("expected a payload on the subscriber channel, got none")
		return nil
	}
}

func expectNone(t *testing.T, sub *subscriber) {
	t.Helper()
	select {
	case p := <-sub.ch:
		t.Fatalf("expected no payload on the subscriber channel, got %q", p)
	default:
	}
}

func isReplaced(sub *subscriber) bool {
	select {
	case <-sub.replaced:
		return true
	default:
		return false
	}
}

func countEntries(s *store) int {
	n := 0
	for i := range s.shards {
		s.shards[i].Lock()
		n += len(s.shards[i].entries)
		s.shards[i].Unlock()
	}
	return n
}

// setPendingExpiry rewrites the stored pending ping's expiry so tests can
// simulate time passing without sleeping.
func setPendingExpiry(t *testing.T, s *store, token string, expiresAt time.Time) {
	t.Helper()
	for i := range s.shards {
		s.shards[i].Lock()
		if e, ok := s.shards[i].entries[token]; ok && e.pending != nil {
			e.pending.expiresAt = expiresAt
			s.shards[i].Unlock()
			return
		}
		s.shards[i].Unlock()
	}
	t.Fatalf("no pending ping stored for token %q", token)
}

func TestPushToLiveSubscriber(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	sub, pending := s.subscribe("aabb01")
	require.Nil(t, pending)

	s.push("aabb01", []byte(`{"mdm":"magic1"}`), time.Time{})

	assert.Equal(t, `{"mdm":"magic1"}`, string(recv(t, sub)))
	assert.EqualValues(t, 1, s.pushesReceived.Load())
	assert.EqualValues(t, 1, s.deliveredLive.Load())
	assert.EqualValues(t, 0, s.stored.Load())
}

func TestPushToLiveSubscriberFullChannelCoalesces(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	sub, _ := s.subscribe("aabb01")

	s.push("aabb01", []byte("p1"), time.Time{})
	s.push("aabb01", []byte("p2"), time.Time{}) // buffer full: dropped

	assert.Equal(t, "p1", string(recv(t, sub)))
	expectNone(t, sub)
	assert.EqualValues(t, 1, s.deliveredLive.Load())
	assert.EqualValues(t, 1, s.coalesced.Load())
	// A coalesced live push must not also be stored as pending.
	assert.EqualValues(t, 0, s.stored.Load())
}

func TestOfflinePushStoredAndDeliveredOnConnect(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	s.push("aabb01", []byte(`{"mdm":"magic1"}`), time.Time{})

	assert.EqualValues(t, 1, s.stored.Load())

	sub, pending := s.subscribe("aabb01")
	require.NotNil(t, pending)
	assert.Equal(t, `{"mdm":"magic1"}`, string(pending))
	// Delivered via the return value, not the channel.
	expectNone(t, sub)
	assert.EqualValues(t, 1, s.deliveredOnConnect.Load())

	// Pending is cleared once delivered: a reconnect gets nothing.
	s.unsubscribe("aabb01", sub)
	_, pending = s.subscribe("aabb01")
	assert.Nil(t, pending)
}

func TestOfflinePushesCoalesceToLatest(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	s.push("aabb01", []byte("p1"), time.Time{})
	s.push("aabb01", []byte("p2"), time.Time{})
	s.push("aabb01", []byte("p3"), time.Time{})

	_, pending := s.subscribe("aabb01")
	assert.Equal(t, "p3", string(pending))
	assert.EqualValues(t, 3, s.pushesReceived.Load())
	assert.EqualValues(t, 1, s.stored.Load())
	assert.EqualValues(t, 2, s.coalesced.Load())
	assert.EqualValues(t, 1, countEntries(s), "coalescing keeps a single entry per token")
}

func TestZeroExpiresAtUsesDefaultTTL(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	before := time.Now()
	s.push("aabb01", []byte("p1"), time.Time{})
	after := time.Now()

	found := false
	for i := range s.shards {
		s.shards[i].Lock()
		if e, ok := s.shards[i].entries["aabb01"]; ok && e.pending != nil {
			found = true
			assert.False(t, e.pending.expiresAt.Before(before.Add(testTTL)))
			assert.False(t, e.pending.expiresAt.After(after.Add(testTTL)))
		}
		s.shards[i].Unlock()
	}
	require.True(t, found, "expected a pending ping with a defaultTTL expiry")
}

func TestExplicitExpiresAtIsKept(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	expiry := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	s.push("aabb01", []byte("p1"), expiry)

	found := false
	for i := range s.shards {
		s.shards[i].Lock()
		if e, ok := s.shards[i].entries["aabb01"]; ok && e.pending != nil {
			found = true
			assert.True(t, e.pending.expiresAt.Equal(expiry))
		}
		s.shards[i].Unlock()
	}
	require.True(t, found)
}

func TestOfflinePushWithPastExpiryDiscarded(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	// apns-expiration: 0 → time.Unix(0, 0): deliver-or-discard, and the
	// device is offline, so discard.
	s.push("aabb01", []byte("p1"), time.Unix(0, 0))

	assert.EqualValues(t, 1, s.pushesReceived.Load())
	assert.EqualValues(t, 0, s.stored.Load())
	assert.EqualValues(t, 1, s.expired.Load())
	assert.EqualValues(t, 0, countEntries(s), "discarded pushes must not leave entries behind")

	_, pending := s.subscribe("aabb01")
	assert.Nil(t, pending)
}

func TestLivePushWithPastExpiryStillDelivered(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	sub, _ := s.subscribe("aabb01")

	// apns-expiration: 0 with a connected device: deliver now.
	s.push("aabb01", []byte("p1"), time.Unix(0, 0))

	assert.Equal(t, "p1", string(recv(t, sub)))
	assert.EqualValues(t, 1, s.deliveredLive.Load())
	assert.EqualValues(t, 0, s.expired.Load())
}

func TestExpiredPendingDroppedLazilyOnSubscribe(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	s.push("aabb01", []byte("p1"), time.Time{})
	setPendingExpiry(t, s, "aabb01", time.Now().Add(-time.Second))

	sub, pending := s.subscribe("aabb01")
	assert.Nil(t, pending)
	expectNone(t, sub)
	assert.EqualValues(t, 1, s.expired.Load())
	assert.EqualValues(t, 0, s.deliveredOnConnect.Load())
}

func TestSweepRemovesExpiredPendingAndEmptyEntries(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	s.push("aaaa01", []byte("p1"), time.Now().Add(time.Hour))
	s.push("bbbb02", []byte("p2"), time.Now().Add(2*time.Hour))
	require.EqualValues(t, 2, countEntries(s))

	s.sweep(time.Now().Add(90 * time.Minute))

	assert.EqualValues(t, 1, countEntries(s), "expired entry should be deleted, unexpired kept")
	assert.EqualValues(t, 1, s.expired.Load())

	_, pending := s.subscribe("aaaa01")
	assert.Nil(t, pending, "swept ping must not be delivered")
	_, pending = s.subscribe("bbbb02")
	assert.Equal(t, "p2", string(pending), "unexpired ping survives the sweep")
}

func TestSweepKeepsLiveSubscribers(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	sub, _ := s.subscribe("aabb01")

	s.sweep(time.Now().Add(48 * time.Hour))

	require.EqualValues(t, 1, countEntries(s), "entries with a live subscriber survive any sweep")
	s.push("aabb01", []byte("p1"), time.Time{})
	assert.Equal(t, "p1", string(recv(t, sub)))
}

func TestReplaceOnReconnect(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	subA, _ := s.subscribe("aabb01")
	require.False(t, isReplaced(subA))

	subB, _ := s.subscribe("aabb01")
	assert.True(t, isReplaced(subA), "older connection must be told it was replaced")
	assert.False(t, isReplaced(subB))

	s.push("aabb01", []byte("p1"), time.Time{})
	assert.Equal(t, "p1", string(recv(t, subB)))
	expectNone(t, subA)
}

func TestStaleUnsubscribeDoesNotEvictReplacement(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	subA, _ := s.subscribe("aabb01")
	subB, _ := s.subscribe("aabb01")
	require.EqualValues(t, 2, s.connected.Load())

	// The replaced handler's deferred unsubscribe fires after the new
	// connection took over the token.
	s.unsubscribe("aabb01", subA)

	assert.EqualValues(t, 1, s.connected.Load(), "gauge tracks connections, so a stale unsubscribe still decrements")
	s.push("aabb01", []byte("p1"), time.Time{})
	assert.Equal(t, "p1", string(recv(t, subB)), "replacement subscriber must survive the stale unsubscribe")
}

func TestUnsubscribeRemovesEmptyEntry(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	sub, _ := s.subscribe("aabb01")
	require.EqualValues(t, 1, s.connected.Load())

	s.unsubscribe("aabb01", sub)

	assert.EqualValues(t, 0, s.connected.Load())
	assert.EqualValues(t, 0, countEntries(s), "no subscriber and no pending ping leaves no entry")

	// The token still works afterwards: an offline push recreates the entry.
	s.push("aabb01", []byte("p1"), time.Time{})
	_, pending := s.subscribe("aabb01")
	assert.Equal(t, "p1", string(pending))
}

func TestTokensAreIndependent(t *testing.T) {
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	subA, _ := s.subscribe("aaaa01")

	s.push("bbbb02", []byte("p1"), time.Time{})

	expectNone(t, subA)
	_, pending := s.subscribe("bbbb02")
	assert.Equal(t, "p1", string(pending))
}

func TestConcurrentAccess(t *testing.T) {
	// No assertions on counters here — this exists to fail under -race and
	// to catch deadlocks between subscribe/push/unsubscribe/sweep.
	s := newStore(testTTL, slog.New(slog.DiscardHandler))
	const tokens = 64
	const rounds = 200

	var wg sync.WaitGroup
	for i := 0; i < tokens; i++ {
		token := fmt.Sprintf("token%02x", i)

		wg.Add(2)
		go func() {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				sub, pending := s.subscribe(token)
				_ = pending
				select {
				case <-sub.ch:
				case <-sub.replaced:
				default:
				}
				s.unsubscribe(token, sub)
			}
		}()
		go func() {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				s.push(token, []byte("p"), time.Time{})
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				s.sweep(time.Now())
			}
		}
	}()
	wg.Wait()
	close(done)

	// Sanity: the store still functions after the stampede.
	sub, _ := s.subscribe("token00")
	s.push("token00", []byte("final"), time.Time{})
	assert.Equal(t, "final", string(recv(t, sub)))
}

package main

// Spec for the node-local half: which tokens this instance holds and how a
// ping reaches their streams. Nothing here needs Redis; the shared half is
// specced in coordinator_test.go.

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recv asserts a ping is already buffered. deliver sends synchronously under
// the shard lock, so there is nothing to wait for.
func recv(t *testing.T, sub *subscriber) string {
	t.Helper()
	select {
	case p := <-sub.ch:
		return string(p.payload)
	default:
		t.Fatal("expected a ping on the subscriber channel, got none")
		return ""
	}
}

func expectNone(t *testing.T, sub *subscriber) {
	t.Helper()
	select {
	case p := <-sub.ch:
		t.Fatalf("expected no ping on the subscriber channel, got %q", p.payload)
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

func countEntries(r *registry) int {
	n := 0
	for i := range r.shards {
		r.shards[i].Lock()
		n += len(r.shards[i].entries)
		r.shards[i].Unlock()
	}
	return n
}

func testPing(payload string) *ping {
	return &ping{payload: []byte(payload), expiresAt: time.Now().Add(time.Hour)}
}

func TestRegistryDeliverToLiveSubscriber(t *testing.T) {
	r := newRegistry()
	sub, evicted := r.subscribe("aabb01", 1)
	require.Nil(t, evicted)

	assert.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))
	assert.Equal(t, "p1", recv(t, sub))
}

func TestRegistryDeliverToUnknownTokenIsNotHeld(t *testing.T) {
	r := newRegistry()

	// The caller still owns the ping here, so this must not read as bufferFull.
	assert.Equal(t, notHeld, r.deliver("aabb01", testPing("p1")))

	sub, _ := r.subscribe("aabb01", 1)
	r.unsubscribe("aabb01", sub)
	assert.Equal(t, notHeld, r.deliver("aabb01", testPing("p1")))
}

func TestRegistryDeliverWithFullBuffer(t *testing.T) {
	r := newRegistry()
	sub, _ := r.subscribe("aabb01", 1)

	require.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))
	// A wake-up is already queued, so the second is redundant, not lost.
	assert.Equal(t, bufferFull, r.deliver("aabb01", testPing("p2")))

	assert.Equal(t, "p1", recv(t, sub))
	expectNone(t, sub)
}

func TestRegistryHoldsAndIsCurrent(t *testing.T) {
	r := newRegistry()
	assert.False(t, r.holds("aabb01"))

	subA, _ := r.subscribe("aabb01", 1)
	assert.True(t, r.holds("aabb01"))
	assert.True(t, r.isCurrent("aabb01", subA))

	subB, _ := r.subscribe("aabb01", 1)
	assert.False(t, r.isCurrent("aabb01", subA), "a replaced connection is no longer current")
	assert.True(t, r.isCurrent("aabb01", subB))

	r.unsubscribe("aabb01", subB)
	assert.False(t, r.holds("aabb01"))
}

func TestRegistryReplaceOnReconnect(t *testing.T) {
	r := newRegistry()
	subA, _ := r.subscribe("aabb01", 1)
	require.False(t, isReplaced(subA))

	subB, _ := r.subscribe("aabb01", 1)
	assert.True(t, isReplaced(subA), "the older connection must be told it was replaced")
	assert.False(t, isReplaced(subB))

	require.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))
	assert.Equal(t, "p1", recv(t, subB))
	expectNone(t, subA)
}

func TestRegistryReplaceReturnsUndeliveredPing(t *testing.T) {
	// A push buffered at the moment the device reconnected was never written,
	// so the caller gets it back to put in Redis.
	r := newRegistry()
	subA, _ := r.subscribe("aabb01", 1)
	require.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))

	_, evicted := r.subscribe("aabb01", 1)

	require.NotNil(t, evicted)
	assert.Equal(t, "p1", string(evicted.payload))
	expectNone(t, subA)
}

func TestRegistryUnsubscribeReturnsUndeliveredPing(t *testing.T) {
	r := newRegistry()
	sub, _ := r.subscribe("aabb01", 1)
	require.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))

	evicted := r.unsubscribe("aabb01", sub)

	require.NotNil(t, evicted)
	assert.Equal(t, "p1", string(evicted.payload))
	assert.Equal(t, 0, countEntries(r), "an unsubscribed token leaves nothing behind")
}

func TestRegistryStaleUnsubscribeDoesNotEvictReplacement(t *testing.T) {
	r := newRegistry()
	subA, _ := r.subscribe("aabb01", 1)
	subB, _ := r.subscribe("aabb01", 1)

	// The replaced handler's deferred unsubscribe fires after the takeover.
	evicted := r.unsubscribe("aabb01", subA)

	assert.Nil(t, evicted, "subscribe already drained this subscriber when it evicted it")
	assert.True(t, r.isCurrent("aabb01", subB), "the replacement must survive")
	require.Equal(t, delivered, r.deliver("aabb01", testPing("p1")))
	assert.Equal(t, "p1", recv(t, subB))
}

func TestRegistryTokensAreIndependent(t *testing.T) {
	r := newRegistry()
	subA, _ := r.subscribe("aaaa01", 1)

	require.Equal(t, notHeld, r.deliver("bbbb02", testPing("p1")))
	expectNone(t, subA)
}

func TestRegistryTokenIsCaseInsensitive(t *testing.T) {
	// Hex tokens are case-insensitive; the registry must not rely on the
	// caller normalizing them.
	r := newRegistry()
	sub, _ := r.subscribe("aabbccddee01", 1)

	assert.True(t, r.holds("AABBCCDDEE01"))
	require.Equal(t, delivered, r.deliver("AABBCCDDEE01", testPing("p1")))
	assert.Equal(t, "p1", recv(t, sub))
}

func TestRegistryTokensListsHeldStreams(t *testing.T) {
	// tokens() drives the resync after a reconnect, so it must list exactly
	// the streams this node can still deliver to.
	r := newRegistry()
	subA, _ := r.subscribe("aaaa01", 1)
	r.subscribe("bbbb02", 1)
	r.unsubscribe("aaaa01", subA)

	assert.ElementsMatch(t, []string{"bbbb02"}, r.tokens())
}

func TestRegistryConcurrentAccess(t *testing.T) {
	// No assertions: this exists to fail under -race and catch deadlocks.
	r := newRegistry()
	const tokens = 64
	const rounds = 200

	var wg sync.WaitGroup
	for i := range tokens {
		token := fmt.Sprintf("token%02x", i)

		wg.Go(func() {
			for range rounds {
				sub, evicted := r.subscribe(token, 1)
				_ = evicted
				select {
				case <-sub.ch:
				case <-sub.replaced:
				default:
				}
				r.unsubscribe(token, sub)
			}
		})
		wg.Go(func() {
			for range rounds {
				r.deliver(token, testPing("p"))
			}
		})
		wg.Go(func() {
			for range rounds {
				r.holds(token)
				_ = r.tokens()
			}
		})
	}
	wg.Wait()

	// Still working after the stampede.
	sub, _ := r.subscribe("token00", 1)
	require.Equal(t, delivered, r.deliver("token00", testPing("final")))
	assert.Equal(t, "final", recv(t, sub))
}

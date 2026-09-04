package main

// Spec for the shared half: store-and-forward through Redis, the claim that
// keeps delivery exactly-once, and routing a push to whichever instance holds
// the device. Needs REDIS_TEST=1 and a Redis on :6379.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCoordinator(t *testing.T, env testRedisEnv) *coordinator {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	coord := newCoordinator(env.pool, newRegistry(), logger, coordinatorConfig{
		NodeID:     fmt.Sprintf("node%d", env.nextNode()),
		KeyPrefix:  env.prefix,
		DefaultTTL: testTTL,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go coord.Run(ctx, 50*time.Millisecond)
	waitSubscribed(t, coord)
	return coord
}

// waitPing waits for a ping to reach the stream. Delivery goes out through
// Redis and back, so unlike the registry it is not synchronous.
func waitPing(t *testing.T, sub *subscriber, timeout time.Duration) string {
	t.Helper()
	select {
	case p := <-sub.ch:
		return string(p.payload)
	case <-time.After(timeout):
		t.Fatal("timed out waiting for a ping")
		return ""
	}
}

func expectNoPingOn(t *testing.T, sub *subscriber, wait time.Duration) {
	t.Helper()
	select {
	case p := <-sub.ch:
		t.Fatalf("expected no ping, got %q", p.payload)
	case <-time.After(wait):
	}
}

// waitAnnouncementsHandled blocks until this node's subscription has handled
// every announcement published so far: a marker ping for a token it holds
// cannot arrive ahead of them, because Redis delivers to a subscriber in
// publish order.
//
// A test that pushes to an offline device and then subscribes needs it.
// Otherwise the push's own announcement can be dispatched in the window
// between Subscribe registering the stream and its claim, and dispatch wins
// the GETDEL — the device still gets the ping, down the live stream, but not
// by the path under test.
func waitAnnouncementsHandled(t *testing.T, c *coordinator) {
	t.Helper()
	const marker = "ffffff"
	sub, _ := c.reg.subscribe(marker, 0)

	// Inline, so the marker needs no pending key. Wait for it to reach the stream so we
	// know this node's subscription has processed all earlier announcements (Redis delivers
	// to a subscriber in publish order).
	require.NoError(t, c.publish(t.Context(), pushMsg{Token: marker, Inline: []byte(`{"mdm":"marker"}`)}))
	waitPing(t, sub, 5*time.Second)

	c.reg.unsubscribe(marker, sub)
	c.reg.deliveredLive.Add(-1) // the marker is not a delivery under test
}

func pendingValue(t *testing.T, c *coordinator, token string) []byte {
	t.Helper()
	conn := c.pool.Get()
	defer conn.Close()
	b, err := redigo.Bytes(conn.Do("GET", c.pendingKey(token)))
	if err != nil {
		require.ErrorIs(t, err, redigo.ErrNil)
		return nil
	}
	return b
}

func TestCoordinatorPingEncodingRoundTrips(t *testing.T) {
	// The expiry rides in the value because a Redis TTL is opaque to whoever
	// reads it, and a requeued ping has to keep the time it had left.
	want := &ping{payload: []byte(`{"mdm":"magic"}`), expiresAt: time.Now().Add(time.Hour).Truncate(time.Millisecond)}

	got, err := decodePing(encodePing(want))

	require.NoError(t, err)
	assert.Equal(t, string(want.payload), string(got.payload))
	assert.True(t, want.expiresAt.Equal(got.expiresAt))

	_, err = decodePing([]byte("short"))
	assert.Error(t, err)
}

func TestCoordinatorPushDeliversToLocalStream(t *testing.T) {
	c := newTestCoordinator(t, testRedis(t))
	sub, pending := c.Subscribe(t.Context(), "aabb01")
	require.Nil(t, pending)

	require.NoError(t, c.Push(t.Context(), "aabb01", []byte(`{"mdm":"m"}`), time.Time{}))

	assert.JSONEq(t, `{"mdm":"m"}`, waitPing(t, sub, 5*time.Second))
	assert.Empty(t, pendingValue(t, c, "aabb01"), "a delivered push must not stay pending")
	assert.EqualValues(t, 1, c.reg.deliveredLive.Load())
}

func TestCoordinatorPushCrossesInstances(t *testing.T) {
	// The whole point: Fleet load-balances its pushes, so the instance that
	// takes the push is almost never the one holding the stream.
	env := testRedis(t)
	holder := newTestCoordinator(t, env)
	pusher := newTestCoordinator(t, env)

	sub, _ := holder.Subscribe(t.Context(), "aabb01")

	require.NoError(t, pusher.Push(t.Context(), "aabb01", []byte(`{"mdm":"crossed"}`), time.Time{}))

	assert.JSONEq(t, `{"mdm":"crossed"}`, waitPing(t, sub, 5*time.Second))
	assert.EqualValues(t, 1, holder.reg.deliveredLive.Load(), "the holder delivered it")
	assert.EqualValues(t, 0, pusher.reg.deliveredLive.Load(), "the pusher holds no stream")
	assert.EqualValues(t, 1, pusher.reg.stored.Load(), "the pusher wrote the pending key")
}

func TestCoordinatorOfflinePushClaimedOnConnect(t *testing.T) {
	env := testRedis(t)
	pusher := newTestCoordinator(t, env)
	holder := newTestCoordinator(t, env)

	require.NoError(t, pusher.Push(t.Context(), "aabb01", []byte(`{"mdm":"waited"}`), time.Time{}))
	require.NotEmpty(t, pendingValue(t, pusher, "aabb01"), "nobody holds it, so it waits in Redis")
	waitAnnouncementsHandled(t, holder)

	// The device turns up on a different instance than the one pushed to.
	_, pending := holder.Subscribe(t.Context(), "aabb01")

	require.NotNil(t, pending)
	assert.JSONEq(t, `{"mdm":"waited"}`, string(pending.payload))
	assert.EqualValues(t, 1, holder.reg.deliveredOnConnect.Load())
	assert.Empty(t, pendingValue(t, holder, "aabb01"), "claiming clears it")
}

func TestCoordinatorOfflinePushesCoalesce(t *testing.T) {
	c := newTestCoordinator(t, testRedis(t))

	for _, m := range []string{"m1", "m2", "m3"} {
		require.NoError(t, c.Push(t.Context(), "aabb01", []byte(`{"mdm":"`+m+`"}`), time.Time{}))
	}
	waitAnnouncementsHandled(t, c)

	_, pending := c.Subscribe(t.Context(), "aabb01")
	require.NotNil(t, pending)
	assert.JSONEq(t, `{"mdm":"m3"}`, string(pending.payload), "newest wins, as APNs coalesces")
	assert.EqualValues(t, 3, c.reg.pushesReceived.Load())
	assert.EqualValues(t, 1, c.reg.stored.Load())
	assert.EqualValues(t, 2, c.reg.coalesced.Load())
}

func TestCoordinatorExpiredPushIsNeverStored(t *testing.T) {
	// apns-expiration: 0 is deliver-now-or-discard, so it rides inline in the
	// announcement instead of being stored.
	t.Run("delivered when connected", func(t *testing.T) {
		env := testRedis(t)
		holder := newTestCoordinator(t, env)
		pusher := newTestCoordinator(t, env)
		sub, _ := holder.Subscribe(t.Context(), "aabb01")

		require.NoError(t, pusher.Push(t.Context(), "aabb01", []byte(`{"mdm":"now"}`), time.Unix(0, 0)))

		assert.JSONEq(t, `{"mdm":"now"}`, waitPing(t, sub, 5*time.Second))
		assert.EqualValues(t, 1, pusher.reg.discarded.Load())
		assert.EqualValues(t, 0, pusher.reg.stored.Load())
	})

	t.Run("dropped when offline", func(t *testing.T) {
		c := newTestCoordinator(t, testRedis(t))

		require.NoError(t, c.Push(t.Context(), "aabb01", []byte(`{"mdm":"gone"}`), time.Unix(0, 0)))

		assert.Empty(t, pendingValue(t, c, "aabb01"))
		_, pending := c.Subscribe(t.Context(), "aabb01")
		assert.Nil(t, pending, "a device that reconnects must not receive it later")
	})
}

func TestCoordinatorClaimIsExactlyOnce(t *testing.T) {
	// Two instances can briefly hold the same token while a device reconnects.
	// Both see the announcement; GETDEL decides which one delivers.
	env := testRedis(t)
	c1 := newTestCoordinator(t, env)
	c2 := newTestCoordinator(t, env)

	// Register on both registries directly: separate processes never see each
	// other's eviction.
	sub1, _ := c1.reg.subscribe("aabb01", 1)
	sub2, _ := c2.reg.subscribe("aabb01", 1)

	require.NoError(t, c1.Push(t.Context(), "aabb01", []byte(`{"mdm":"once"}`), time.Time{}))

	var got int
	deadline := time.After(5 * time.Second)
	for got == 0 {
		select {
		case p := <-sub1.ch:
			assert.JSONEq(t, `{"mdm":"once"}`, string(p.payload))
			got++
		case p := <-sub2.ch:
			assert.JSONEq(t, `{"mdm":"once"}`, string(p.payload))
			got++
		case <-deadline:
			t.Fatal("neither instance delivered the push")
		}
	}

	expectNoPingOn(t, sub1, 200*time.Millisecond)
	expectNoPingOn(t, sub2, 200*time.Millisecond)
	waitFor(t, func() bool { return c1.reg.claimMisses.Load()+c2.reg.claimMisses.Load() == 1 })
	assert.EqualValues(t, 1, c1.reg.claimMisses.Load()+c2.reg.claimMisses.Load(),
		"the instance that lost the race records a claim miss")
}

func TestCoordinatorReconnectElsewhereMovesToken(t *testing.T) {
	// Eviction is registry-local, so a device that reconnects to another
	// instance would otherwise leave a live stream behind and pushes would
	// split between the two at random. Subscribing announces the takeover.
	env := testRedis(t)
	first := newTestCoordinator(t, env)
	second := newTestCoordinator(t, env)

	oldSub, _ := first.Subscribe(t.Context(), "aabb01")
	newSub, _ := second.Subscribe(t.Context(), "aabb01")

	waitFor(t, func() bool { return !first.reg.holds("aabb01") })
	select {
	case <-oldSub.replaced:
	default:
		t.Fatal("the superseded stream was never told to stand down")
	}

	require.NoError(t, first.Push(t.Context(), "aabb01", []byte(`{"mdm":"followed"}`), time.Time{}))

	assert.JSONEq(t, `{"mdm":"followed"}`, waitPing(t, newSub, 5*time.Second))
	assert.EqualValues(t, 1, second.reg.connected.Load())
}

func TestCoordinatorTakeoverCarriesUndeliveredPing(t *testing.T) {
	// A ping buffered on the old instance when the device moved was never
	// written, so it is re-announced for the new owner rather than lost.
	env := testRedis(t)
	first := newTestCoordinator(t, env)
	second := newTestCoordinator(t, env)

	first.Subscribe(t.Context(), "aabb01")
	require.NoError(t, first.Push(t.Context(), "aabb01", []byte(`{"mdm":"inflight"}`), time.Time{}))
	waitFor(t, func() bool { return first.reg.deliveredLive.Load() == 1 })

	newSub, pending := second.Subscribe(t.Context(), "aabb01")

	if pending != nil {
		assert.JSONEq(t, `{"mdm":"inflight"}`, string(pending.payload))
		return
	}
	assert.JSONEq(t, `{"mdm":"inflight"}`, waitPing(t, newSub, 5*time.Second))
}

func TestCoordinatorUnsubscribeRequeuesToRedis(t *testing.T) {
	// A ping buffered when the device dropped was never written, so it goes
	// back to Redis for whichever instance the device reconnects to.
	c := newTestCoordinator(t, testRedis(t))
	sub, _ := c.Subscribe(t.Context(), "aabb01")
	require.NoError(t, c.Push(t.Context(), "aabb01", []byte(`{"mdm":"unread"}`), time.Time{}))
	waitFor(t, func() bool { return c.reg.deliveredLive.Load() == 1 })

	c.Unsubscribe(t.Context(), "aabb01", sub)

	assert.EqualValues(t, 0, c.reg.deliveredLive.Load(), "it never reached the wire")
	_, pending := c.Subscribe(t.Context(), "aabb01")
	require.NotNil(t, pending)
	assert.JSONEq(t, `{"mdm":"unread"}`, string(pending.payload))
}

func TestCoordinatorSubscribeRequeuesEvictedPing(t *testing.T) {
	// Same race on reconnect: the new connection gets the wake-up the old one
	// never wrote.
	c := newTestCoordinator(t, testRedis(t))
	c.Subscribe(t.Context(), "aabb01")
	require.NoError(t, c.Push(t.Context(), "aabb01", []byte(`{"mdm":"missed"}`), time.Time{}))
	waitFor(t, func() bool { return c.reg.deliveredLive.Load() == 1 })

	_, pending := c.Subscribe(t.Context(), "aabb01")

	require.NotNil(t, pending)
	assert.JSONEq(t, `{"mdm":"missed"}`, string(pending.payload))
}

func TestCoordinatorRestoreOnlyWhenCurrent(t *testing.T) {
	c := newTestCoordinator(t, testRedis(t))
	subA, _ := c.Subscribe(t.Context(), "aabb01")

	c.Restore(t.Context(), "aabb01", subA, &ping{payload: []byte("mine"), expiresAt: time.Now().Add(time.Hour)}, false)
	assert.NotEmpty(t, pendingValue(t, c, "aabb01"), "the current stream's failed write goes back")

	subB, _ := c.Subscribe(t.Context(), "aabb01") // claims it, clearing the key
	require.NotNil(t, subB)
	require.Empty(t, pendingValue(t, c, "aabb01"))

	c.Restore(t.Context(), "aabb01", subA, &ping{payload: []byte("stale"), expiresAt: time.Now().Add(time.Hour)}, false)
	assert.Empty(t, pendingValue(t, c, "aabb01"), "a replaced stream cannot resurrect its ping")
}

func TestCoordinatorResyncRecoversMissedPushes(t *testing.T) {
	// Announcements published while the subscription was down are missed, but
	// their pending keys survive, so resync picks them up.
	c := newTestCoordinator(t, testRedis(t))
	sub, _ := c.Subscribe(t.Context(), "aabb01")

	// Write the key without announcing it: the same state as a dropped
	// subscription.
	require.NoError(t, c.storePending(t.Context(), "aabb01", &ping{
		payload:   []byte(`{"mdm":"missed"}`),
		expiresAt: time.Now().Add(time.Hour),
	}))
	expectNoPingOn(t, sub, 200*time.Millisecond)

	c.resync(t.Context())

	assert.JSONEq(t, `{"mdm":"missed"}`, waitPing(t, sub, 5*time.Second))
}

func TestCoordinatorStatsAggregateAcrossInstances(t *testing.T) {
	env := testRedis(t)
	c1 := newTestCoordinator(t, env)
	c2 := newTestCoordinator(t, env)

	c1.Subscribe(t.Context(), "aaaa01")
	c2.Subscribe(t.Context(), "bbbb02")
	require.NoError(t, c1.Push(t.Context(), "cccc03", []byte(`{"mdm":"m"}`), time.Time{}))

	// Both instances flush on a ticker; wait for the other one to appear.
	var stats statsResponse
	waitFor(t, func() bool {
		stats = c1.Stats(t.Context())
		return stats.Nodes == 2
	})

	assert.Equal(t, c1.cfg.NodeID, stats.NodeID)
	assert.EqualValues(t, 1, stats.Node.ActiveConnections, "node reports only its own stream")
	assert.EqualValues(t, 2, stats.Cluster.ActiveConnections, "cluster totals both")
	assert.EqualValues(t, 1, stats.Cluster.TotalPushes)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
}

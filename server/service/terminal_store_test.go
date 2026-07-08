package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore returns a fresh store that does NOT start the background
// sweeper goroutine (gcOnce is package-level and may already be spent).
func newTestStore() *terminalSessionStore {
	return &terminalSessionStore{sessions: make(map[string]*terminalSession)}
}

// addSession inserts a session directly into the store without triggering
// gcOnce, so tests remain isolated from the package-level GC goroutine.
func addSession(s *terminalSessionStore, hostID uint, name string) (string, *terminalSession) {
	id := uuid.New().String()
	sess := &terminalSession{
		hostID:          hostID,
		hostDisplayName: name,
		createdAt:       time.Now(),
		fromBrowser:     make(chan []byte, 256),
		toBrowser:       make(chan []byte, 256),
		orbitConnected:  make(chan struct{}),
		done:            make(chan struct{}),
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return id, sess
}

// ── markBrowserClaimed ────────────────────────────────────────────────────────

func TestMarkBrowserClaimed(t *testing.T) {
	s := newTestStore()

	t.Run("returns true and sets flag for existing session", func(t *testing.T) {
		id, _ := addSession(s, 1, "host-a")
		ok := s.markBrowserClaimed(id)
		require.True(t, ok)

		s.mu.Lock()
		claimed := s.sessions[id].browserClaimed
		s.mu.Unlock()
		assert.True(t, claimed)
	})

	t.Run("returns false for nonexistent session", func(t *testing.T) {
		ok := s.markBrowserClaimed("does-not-exist")
		assert.False(t, ok)
	})
}

// ── claim ─────────────────────────────────────────────────────────────────────

func TestClaim(t *testing.T) {
	s := newTestStore()

	t.Run("fails without browser claim", func(t *testing.T) {
		id, _ := addSession(s, 1, "host-a")
		// Browser has NOT authenticated — orbit should not be able to claim.
		sess, ok := s.claim(id, 1)
		assert.False(t, ok)
		assert.Nil(t, sess)
	})

	t.Run("succeeds after browser claim with correct host", func(t *testing.T) {
		id, _ := addSession(s, 2, "host-b")
		require.True(t, s.markBrowserClaimed(id))

		sess, ok := s.claim(id, 2)
		require.True(t, ok)
		require.NotNil(t, sess)
		assert.True(t, sess.connected)
	})

	t.Run("fails for wrong host ID", func(t *testing.T) {
		id, _ := addSession(s, 3, "host-c")
		require.True(t, s.markBrowserClaimed(id))

		sess, ok := s.claim(id, 99)
		assert.False(t, ok)
		assert.Nil(t, sess)
	})

	t.Run("fails for nonexistent session", func(t *testing.T) {
		sess, ok := s.claim("no-such-id", 1)
		assert.False(t, ok)
		assert.Nil(t, sess)
	})

	t.Run("duplicate claim is rejected", func(t *testing.T) {
		id, _ := addSession(s, 4, "host-d")
		require.True(t, s.markBrowserClaimed(id))

		_, ok1 := s.claim(id, 4)
		require.True(t, ok1, "first claim must succeed")

		_, ok2 := s.claim(id, 4)
		assert.False(t, ok2, "second claim must be rejected")
	})
}

// ── pendingForHost ────────────────────────────────────────────────────────────

func TestPendingForHost(t *testing.T) {
	s := newTestStore()

	t.Run("excludes sessions without browser claim", func(t *testing.T) {
		addSession(s, 10, "host-x") // not browser-claimed
		ids := s.pendingForHost(10)
		assert.Empty(t, ids, "unclaimed sessions must not appear as pending")
	})

	t.Run("includes browser-claimed, not-yet-connected sessions", func(t *testing.T) {
		s2 := newTestStore()
		id, _ := addSession(s2, 10, "host-x")
		require.True(t, s2.markBrowserClaimed(id))

		ids := s2.pendingForHost(10)
		assert.Equal(t, []string{id}, ids)
	})

	t.Run("excludes connected sessions", func(t *testing.T) {
		s3 := newTestStore()
		id, _ := addSession(s3, 10, "host-x")
		require.True(t, s3.markBrowserClaimed(id))
		_, ok := s3.claim(id, 10)
		require.True(t, ok)

		ids := s3.pendingForHost(10)
		assert.Empty(t, ids, "connected sessions must not appear as pending")
	})

	t.Run("only returns sessions for requested host", func(t *testing.T) {
		s4 := newTestStore()
		idA, _ := addSession(s4, 10, "host-a")
		idB, _ := addSession(s4, 20, "host-b")
		require.True(t, s4.markBrowserClaimed(idA))
		require.True(t, s4.markBrowserClaimed(idB))

		ids := s4.pendingForHost(10)
		assert.Equal(t, []string{idA}, ids)
		assert.NotContains(t, ids, idB)
	})
}

// ── sweep ─────────────────────────────────────────────────────────────────────

func TestSweep(t *testing.T) {
	t.Run("removes expired pending sessions and closes done channel", func(t *testing.T) {
		s := newTestStore()
		id, sess := addSession(s, 1, "host")

		// Backdate so it looks expired.
		s.mu.Lock()
		sess.createdAt = time.Now().Add(-sessionTTL - time.Second)
		s.mu.Unlock()

		s.sweep()

		_, ok := s.get(id)
		assert.False(t, ok, "expired pending session should be removed")

		select {
		case <-sess.done:
			// done channel must be closed
		default:
			t.Fatal("done channel must be closed after sweep")
		}
	})

	t.Run("removes expired browser-claimed (but not orbit-connected) sessions", func(t *testing.T) {
		s := newTestStore()
		id, sess := addSession(s, 1, "host")
		require.True(t, s.markBrowserClaimed(id))

		s.mu.Lock()
		sess.createdAt = time.Now().Add(-sessionTTL - time.Second)
		s.mu.Unlock()

		s.sweep()

		_, ok := s.get(id)
		assert.False(t, ok, "expired browser-claimed session should be removed")
	})

	t.Run("preserves active (connected) sessions regardless of age", func(t *testing.T) {
		s := newTestStore()
		id, sess := addSession(s, 1, "host")
		require.True(t, s.markBrowserClaimed(id))
		_, ok := s.claim(id, 1)
		require.True(t, ok)

		// Backdate way past TTL.
		s.mu.Lock()
		sess.createdAt = time.Now().Add(-sessionTTL * 10)
		s.mu.Unlock()

		s.sweep()

		_, ok = s.get(id)
		assert.True(t, ok, "active session must survive sweep")
	})

	t.Run("does not remove unexpired sessions", func(t *testing.T) {
		s := newTestStore()
		id, _ := addSession(s, 1, "host")
		// createdAt defaults to time.Now() — well within TTL.
		s.sweep()

		_, ok := s.get(id)
		assert.True(t, ok, "fresh session must not be removed")
	})
}

// ── remove ────────────────────────────────────────────────────────────────────

func TestRemove(t *testing.T) {
	s := newTestStore()

	id, sess := addSession(s, 1, "host")
	s.remove(id)

	_, ok := s.get(id)
	assert.False(t, ok, "session should be gone after remove")

	select {
	case <-sess.done:
		// done channel must be closed
	default:
		t.Fatal("done channel must be closed after remove")
	}

	// Double-remove must not panic.
	assert.NotPanics(t, func() { s.remove(id) })
}

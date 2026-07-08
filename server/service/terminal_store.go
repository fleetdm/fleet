package service

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// terminalSession holds the state for one pending or active web terminal
// session. It is kept entirely in memory because sessions are transient—they
// disappear when the Fleet server restarts, which is acceptable for an
// interactive terminal.
type terminalSession struct {
	hostID          uint
	hostDisplayName string
	createdAt       time.Time

	// browserClaimed is set to true after a browser WebSocket has successfully
	// authenticated for this session.  Orbit only receives session IDs when
	// browserClaimed is true, so no shell is ever started without a verified
	// browser connection.  Protected by the store's mutex.
	browserClaimed bool

	// connected is set to true once an orbit agent has successfully claimed the
	// session via claim().  Protected by the store's mutex.
	connected bool

	// fromBrowser buffers bytes from the browser WebSocket headed to the orbit
	// agent (keyboard input, resize events).
	fromBrowser chan []byte

	// toBrowser buffers bytes from the orbit agent headed to the browser
	// (PTY output).
	toBrowser chan []byte

	// orbitConnected is closed by the orbit-side WebSocket handler once the
	// orbit agent has dialled in and is ready to relay data.
	orbitConnected chan struct{}

	// done is closed when the session is torn down from either side.
	done chan struct{}
}

// terminalStore is the package-level singleton that owns all live sessions.
var terminalStore = &terminalSessionStore{
	sessions: make(map[string]*terminalSession),
}

// gcOnce ensures the background TTL sweeper is started at most once.
var gcOnce sync.Once

type terminalSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*terminalSession
}

// sessionTTL is the maximum lifetime of a session that has not yet been
// connected by an orbit agent.  Active (connected) sessions are not swept;
// they are cleaned up when the browser or orbit handler returns.
const sessionTTL = 5 * time.Minute

// create allocates a new session for hostID and returns its ID.
// It also starts the background TTL sweeper on first call.
func (s *terminalSessionStore) create(hostID uint, hostDisplayName string) (string, *terminalSession) {
	// Start GC goroutine once, the first time any session is created.
	gcOnce.Do(func() { go s.sweepLoop() })

	id := uuid.New().String()
	sess := &terminalSession{
		hostID:          hostID,
		hostDisplayName: hostDisplayName,
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

// markBrowserClaimed records that a browser has authenticated for this session.
// Returns false if the session no longer exists (e.g. expired and swept).
// After a successful call the session becomes visible to orbit via pendingForHost.
func (s *terminalSessionStore) markBrowserClaimed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return false
	}
	sess.browserClaimed = true
	return true
}

// claim atomically validates host ownership, rejects duplicates, and marks the
// session as connected.  All three checks run under the store's mutex so two
// concurrent orbit goroutines cannot both pass the duplicate check and then
// both attempt to close orbitConnected (which would panic).
//
// It only succeeds when the session exists, belongs to hostID, has been claimed
// by a browser (browserClaimed == true), and has not already been connected.
func (s *terminalSessionStore) claim(id string, hostID uint) (*terminalSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, false // session not found
	}
	if sess.hostID != hostID {
		return nil, false // wrong host
	}
	if !sess.browserClaimed {
		return nil, false // browser has not authenticated yet
	}
	if sess.connected {
		return nil, false // duplicate orbit connection
	}
	sess.connected = true
	return sess, true
}

// sweepLoop runs in the background and purges sessions older than sessionTTL.
func (s *terminalSessionStore) sweepLoop() {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for range t.C {
		s.sweep()
	}
}

// sweep removes non-connected sessions older than sessionTTL.
// Active (connected) sessions are never swept — they are removed when the
// orbit or browser handler returns.
func (s *terminalSessionStore) sweep() {
	cutoff := time.Now().Add(-sessionTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess.connected {
			continue // live terminal — leave it alone
		}
		if sess.createdAt.Before(cutoff) {
			select {
			case <-sess.done:
			default:
				close(sess.done)
			}
			delete(s.sessions, id)
		}
	}
}

func (s *terminalSessionStore) get(id string) (*terminalSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

// remove closes the session's done channel and removes it from the store.
func (s *terminalSessionStore) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		select {
		case <-sess.done:
		default:
			close(sess.done)
		}
		delete(s.sessions, id)
	}
}

// pendingForHost returns IDs of sessions where the browser has authenticated
// (browserClaimed == true) but orbit has not yet connected (connected == false).
// Sessions where the browser has not yet authenticated are excluded so that
// orbit cannot start a shell for a session with no live browser.
func (s *terminalSessionStore) pendingForHost(hostID uint) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for id, sess := range s.sessions {
		if sess.hostID == hostID && sess.browserClaimed && !sess.connected {
			ids = append(ids, id)
		}
	}
	return ids
}

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
	// creatorUserID is the Fleet user ID of the global admin who created the
	// session via POST /hosts/{id}/terminal.  markBrowserClaimed enforces that
	// only the same user can attach a browser WebSocket, preventing any other
	// admin from hijacking a session UUID they obtained out-of-band.
	creatorUserID uint
	createdAt     time.Time

	// browserClaimed is set to true after the creating user's browser WebSocket
	// has successfully authenticated for this session.  It is single-use: a
	// second call to markBrowserClaimed for the same session is rejected.
	// Orbit only receives session IDs when browserClaimed is true.
	// Protected by the store's mutex.
	browserClaimed bool

	// connected is set to true once an orbit agent has successfully claimed the
	// session via claim().  Protected by the store's mutex.
	connected bool

	// connectedAt records when orbit claimed the session.  The sweeper uses
	// this to enforce maxSessionDuration on active (connected) sessions.
	connectedAt time.Time

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

const (
	// sessionTTL is the maximum lifetime of a session that has not yet been
	// connected by an orbit agent.
	sessionTTL = 5 * time.Minute

	// maxSessionDuration is the absolute wall-clock limit on an active
	// (connected) terminal session.  The sweeper removes sessions that exceed
	// this limit so that forgotten or half-open shells cannot live forever.
	// The orbit handler also enforces this with context.WithTimeout so the
	// WS relay exits promptly rather than waiting for the next sweep cycle.
	maxSessionDuration = 8 * time.Hour
)

// create allocates a new session for hostID, bound to creatorUserID, and
// returns its ID.  It also starts the background TTL sweeper on first call.
func (s *terminalSessionStore) create(hostID uint, hostDisplayName string, creatorUserID uint) (string, *terminalSession) {
	// Start GC goroutine once, the first time any session is created.
	gcOnce.Do(func() { go s.sweepLoop() })

	id := uuid.New().String()
	sess := &terminalSession{
		hostID:          hostID,
		hostDisplayName: hostDisplayName,
		creatorUserID:   creatorUserID,
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

// markBrowserClaimed records that the creating user's browser has authenticated
// for this session.  It enforces two invariants:
//
//  1. Single-use: a second call (e.g. a racing browser tab) is rejected, so
//     multiple browser WebSockets cannot concurrently read the same channels.
//  2. Creator-bound: only the user who called CreateTerminalSession can attach.
//
// Returns false if the session no longer exists, was already claimed, or the
// claiming user does not match the creator.
// On success the session becomes visible to orbit via pendingForHost.
func (s *terminalSessionStore) markBrowserClaimed(id string, userID uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return false
	}
	if sess.browserClaimed {
		return false // single-use: reject any subsequent attempt
	}
	if sess.creatorUserID != userID {
		return false // only the creating admin may attach
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
	sess.connectedAt = time.Now()
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

// sweep removes stale sessions:
//   - Non-connected sessions older than sessionTTL (pre-connection timeout).
//   - Connected sessions whose connectedAt exceeds maxSessionDuration (absolute
//     timeout).  Closing sess.done signals the browser/orbit handlers to exit.
func (s *terminalSessionStore) sweep() {
	now := time.Now()
	cutoff := now.Add(-sessionTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess.connected {
			// Apply absolute wall-clock limit to active sessions.
			if !sess.connectedAt.IsZero() && now.Sub(sess.connectedAt) > maxSessionDuration {
				select {
				case <-sess.done:
				default:
					close(sess.done)
				}
				delete(s.sessions, id)
			}
			continue
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

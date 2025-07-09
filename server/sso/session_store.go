package sso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// Session stores state for the lifetime of a single sign on session.
type Session struct {
	// RequestID is the SAMLRequest ID that must match "InResponseTo" in the SAMLResponse.
	RequestID string `json:"request_id"`
	// Metadata is the IdP's Metadata used to validate the response.
	Metadata string `json:"metadata"`
	// OriginalURL is the resource being accessed when login request was triggered
	OriginalURL string `json:"original_url"`
}

// SessionStore persists state of a sso session across process boundries and
// method calls by associating the state of the sign on session with a unique
// token created by the user agent (browser SPA).  The lifetime of the state object
// is constrained in the backing store (Redis) so if the sso process is not completed in
// a reasonable amount of time, it automatically expires and is removed.
type SessionStore interface {
	create(sessionID, requestID, originalURL, metadata string, lifetimeSecs uint) error
	get(sessionID string) (*Session, error)
	expire(sessionID string) error
	// Fullfill loads a session with the given session ID, deletes it and returns it.
	Fullfill(sessionID string) (*Session, error)
}

// NewSessionStore creates a SessionStore
func NewSessionStore(pool fleet.RedisPool) SessionStore {
	return &store{pool}
}

type store struct {
	pool fleet.RedisPool
}

func (s *store) create(sessionID, requestID, originalURL, metadata string, lifetimeSecs uint) error {
	if len(sessionID) < 8 {
		return errors.New("request id must be 8 or more characters in length")
	}
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()

	session := Session{
		RequestID:   requestID,
		Metadata:    metadata,
		OriginalURL: originalURL,
	}
	var writer bytes.Buffer
	err := json.NewEncoder(&writer).Encode(session)
	if err != nil {
		return err
	}
	_, err = conn.Do("SETEX", sessionID, lifetimeSecs, writer.String())
	return err
}

func (s *store) get(sessionID string) (*Session, error) {
	// not reading from a replica here as this gets called in close succession
	// in the auth flow, with initiate SSO writing and callback SSO having to
	// read that write.
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	val, err := redigo.String(conn.Do("GET", sessionID))
	if err != nil {
		if err == redigo.ErrNil {
			return nil, fleet.NewAuthRequiredError("session not found")
		}
		return nil, err
	}

	var sess Session
	reader := bytes.NewBufferString(val)
	err = json.NewDecoder(reader).Decode(&sess)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *store) expire(sessionID string) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("DEL", sessionID)
	return err
}

func (s *store) Fullfill(sessionID string) (*Session, error) {
	session, err := s.get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("sso request invalid: %w", err)
	}
	// Remove session so that it can't be reused before it expires.
	err = s.expire(sessionID)
	if err != nil {
		return nil, fmt.Errorf("remove sso request: %w", err)
	}
	return session, nil
}

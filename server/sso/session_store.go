package sso

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// Session stores state for the lifetime of a single sign on session
type Session struct {
	// OriginalURL is the resource being accessed when login request was triggered
	OriginalURL string `json:"original_url"`
	// UserName is only assigned from the IDP auth response, if present it
	// indicates the that user has authenticated against the IDP.
	UserName string `json:"user_name"`
	// ExpiresAt session will be removed after this time.
	ExpiresAt time.Time `json:"expires_at"`
	Metadata  string    `json:"metadata"`
}

// SessionStore persists state of a sso session across process boundries and
// method calls by associating the state of the sign on session with a unique
// token created by the user agent (browser SPA).  The lifetime of the state object
// is constrained in the backing store (Redis) so if the sso process is not completed in
// a reasonable amount of time, it automatically expires and is removed.
type SessionStore interface {
	create(requestID, originalURL, metadata string, lifetimeSecs uint) error
	get(requestID string) (*Session, error)
	expire(requestID string) error
	Fullfill(requestID string) (*Session, *Metadata, error)
}

// NewSessionStore creates a SessionStore
func NewSessionStore(pool fleet.RedisPool) SessionStore {
	return &store{pool}
}

type store struct {
	pool fleet.RedisPool
}

func (s *store) create(requestID, originalURL, metadata string, lifetimeSecs uint) error {
	if len(requestID) < 8 {
		return errors.New("request id must be 8 or more characters in length")
	}
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	sess := Session{OriginalURL: originalURL, Metadata: metadata}
	var writer bytes.Buffer
	err := json.NewEncoder(&writer).Encode(sess)
	if err != nil {
		return err
	}
	_, err = conn.Do("SETEX", requestID, lifetimeSecs, writer.String())
	return err
}

func (s *store) get(requestID string) (*Session, error) {
	// not reading from a replica here as this gets called in close succession
	// in the auth flow, with initiate SSO writing and callback SSO having to
	// read that write.
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	val, err := redigo.String(conn.Do("GET", requestID))
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

func (s *store) expire(requestID string) error {
	conn := redis.ConfigureDoer(s.pool, s.pool.Get())
	defer conn.Close()
	_, err := conn.Do("DEL", requestID)
	return err
}

func (s *store) Fullfill(requestID string) (*Session, *Metadata, error) {
	session, err := s.get(requestID)
	if err != nil {
		return nil, nil, fmt.Errorf("sso request invalid: %w", err)
	}

	// Remove session so that it can't be reused before it expires.
	err = s.expire(requestID)
	if err != nil {
		return nil, nil, fmt.Errorf("remove sso request: %w", err)
	}

	var metadata *Metadata
	if err := xml.Unmarshal([]byte(session.Metadata), &metadata); err != nil {
		return nil, nil, fmt.Errorf("unmarshal sso request metadata: %w", err)
	}

	return session, metadata, nil
}

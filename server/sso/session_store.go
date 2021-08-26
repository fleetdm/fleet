package sso

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
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
	create(requestID, originalURL, x509Cert string, lifetimeSecs uint) error
	Get(requestID string) (*Session, error)
	Expire(requestID string) error
}

// NewSessionStore creates a SessionStore
func NewSessionStore(pool *redisc.Cluster) SessionStore {
	return &store{pool}
}

type store struct {
	pool *redisc.Cluster
}

func (s *store) create(requestID, originalURL, metadata string, lifetimeSecs uint) error {
	if len(requestID) < 8 {
		return errors.New("request id must be 8 or more characters in length")
	}
	conn := s.pool.Get()
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

func (s *store) Get(requestID string) (*Session, error) {
	conn := s.pool.Get()
	defer conn.Close()
	val, err := redis.String(conn.Do("GET", requestID))
	if err != nil {
		if err == redis.ErrNil {
			return nil, ErrSessionNotFound
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

var ErrSessionNotFound = errors.New("session not found")

func (s *store) Expire(requestID string) error {
	conn := s.pool.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", requestID)
	return err
}

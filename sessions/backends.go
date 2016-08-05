package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/jinzhu/gorm"
)

////////////////////////////////////////////////////////////////////////////////
// Session Backend API
////////////////////////////////////////////////////////////////////////////////

// SessionBackend is the abstract interface that all session backends must
// conform to. SessionBackend instances are only expected to exist within the
// context of a single request.
type SessionBackend interface {
	// Given a session key, find and return a session object or an error if one
	// could not be found for the given key
	FindKey(key string) (*Session, error)

	// Given a session id, find and return a session object or an error if one
	// could not be found for the given id
	FindID(id uint) (*Session, error)

	// Find all of the active sessions for a given user
	FindAllForUser(id uint) ([]*Session, error)

	// Create a session object tied to the given user ID
	Create(userID uint) (*Session, error)

	// Destroy the currently tracked session
	Destroy(session *Session) error

	// Destroy all of the sessions for a given user
	DestroyAllForUser(id uint) error

	// Mark the currently tracked session as access to extend expiration
	MarkAccessed(session *Session) error
}

////////////////////////////////////////////////////////////////////////////////
// Session Backend Plugins
////////////////////////////////////////////////////////////////////////////////

// GormSessionBackend stores sessions using a pre-instantiated gorm database
// object
type GormSessionBackend struct {
	DB *gorm.DB
}

func (s *GormSessionBackend) validate(session *Session) error {
	if time.Since(session.AccessedAt).Seconds() >= Lifespan {
		err := s.DB.Delete(session).Error
		if err != nil {
			return err
		}
		return ErrSessionExpired
	}

	err := s.MarkAccessed(session)
	if err != nil {
		return err
	}

	return nil
}

func (s *GormSessionBackend) FindID(id uint) (*Session, error) {
	session := &Session{
		ID: id,
	}

	err := s.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = s.validate(session)
	if err != nil {
		return nil, err
	}

	return session, nil

}

func (s *GormSessionBackend) FindKey(key string) (*Session, error) {
	session := &Session{
		Key: key,
	}

	err := s.DB.Where(session).First(session).Error
	if err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, ErrNoActiveSession
		default:
			return nil, err
		}
	}

	err = s.validate(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *GormSessionBackend) FindAllForUser(id uint) ([]*Session, error) {
	var sessions []*Session
	err := s.DB.Where("user_id = ?", id).Find(&sessions).Error
	return sessions, err
}

func (s *GormSessionBackend) Create(userID uint) (*Session, error) {
	key := make([]byte, SessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	session := &Session{
		UserID: userID,
		Key:    base64.StdEncoding.EncodeToString(key),
	}

	err = s.DB.Create(session).Error
	if err != nil {
		return nil, err
	}

	err = s.MarkAccessed(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *GormSessionBackend) Destroy(session *Session) error {
	err := s.DB.Delete(session).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *GormSessionBackend) DestroyAllForUser(id uint) error {
	return s.DB.Delete(&Session{}, "user_id = ?", id).Error
}

func (s *GormSessionBackend) MarkAccessed(session *Session) error {
	session.AccessedAt = time.Now().UTC()
	return s.DB.Save(session).Error
}

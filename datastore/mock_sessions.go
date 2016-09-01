package datastore

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/kolide/kolide-ose/kolide"
)

func (orm *mockDB) FindSessionByKey(key string) (*kolide.Session, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	for _, session := range orm.sessions {
		if session.Key == key {
			return session, nil
		}
	}
	return nil, ErrNotFound
}

func (orm *mockDB) FindSessionByID(id uint) (*kolide.Session, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	if session, ok := orm.sessions[id]; ok {
		return session, nil
	}
	return nil, ErrNotFound
}

func (orm *mockDB) FindAllSessionsForUser(id uint) ([]*kolide.Session, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()

	var sessions []*kolide.Session
	for _, session := range orm.sessions {
		if session.UserID == id {
			sessions = append(sessions, session)
		}
	}
	if len(sessions) == 0 {
		return nil, ErrNotFound
	}
	return sessions, nil
}

func (orm *mockDB) CreateSessionForUserID(userID uint) (*kolide.Session, error) {
	orm.mtx.Lock()
	defer orm.mtx.Unlock()
	key := make([]byte, orm.sessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	session := &kolide.Session{
		UserID: userID,
		Key:    base64.StdEncoding.EncodeToString(key),
	}

	session.ID = uint(len(orm.sessions))
	orm.sessions[session.ID] = session
	if err = orm.MarkSessionAccessed(session); err != nil {
		return nil, err
	}

	return session, nil

}

func (orm *mockDB) DestroySession(session *kolide.Session) error {
	if _, ok := orm.sessions[session.ID]; !ok {
		return ErrNotFound
	}
	delete(orm.sessions, session.ID)
	return nil
}

func (orm *mockDB) DestroyAllSessionsForUser(id uint) error {
	for _, session := range orm.sessions {
		if session.UserID == id {
			delete(orm.sessions, session.ID)
		}
	}
	return nil
}

func (orm *mockDB) MarkSessionAccessed(session *kolide.Session) error {
	session.AccessedAt = time.Now().UTC()
	if _, ok := orm.sessions[session.ID]; !ok {
		return ErrNotFound
	}
	orm.sessions[session.ID] = session
	return nil
}

// TODO test session validation(expiration)

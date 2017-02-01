package inmem

import (
	"time"

	"github.com/kolide/kolide/server/kolide"
)

func (d *Datastore) SessionByKey(key string) (*kolide.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, session := range d.sessions {
		if session.Key == key {
			return session, nil
		}
	}
	return nil, notFound("Session")
}

func (d *Datastore) SessionByID(id uint) (*kolide.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if session, ok := d.sessions[id]; ok {
		return session, nil
	}
	return nil, notFound("Session").WithID(id)
}

func (d *Datastore) ListSessionsForUser(id uint) ([]*kolide.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var sessions []*kolide.Session
	for _, session := range d.sessions {
		if session.UserID == id {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (d *Datastore) NewSession(session *kolide.Session) (*kolide.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	session.ID = d.nextID(session)
	d.sessions[session.ID] = session
	if err := d.MarkSessionAccessed(session); err != nil {
		return nil, err
	}

	return session, nil

}

func (d *Datastore) DestroySession(session *kolide.Session) error {
	if _, ok := d.sessions[session.ID]; !ok {
		return notFound("Session").WithID(session.ID)
	}
	delete(d.sessions, session.ID)
	return nil
}

func (d *Datastore) DestroyAllSessionsForUser(id uint) error {
	for _, session := range d.sessions {
		if session.UserID == id {
			delete(d.sessions, session.ID)
		}
	}
	return nil
}

func (d *Datastore) MarkSessionAccessed(session *kolide.Session) error {
	session.AccessedAt = time.Now().UTC()
	if _, ok := d.sessions[session.ID]; !ok {
		return notFound("Session").WithID(session.ID)
	}
	d.sessions[session.ID] = session
	return nil
}

// TODO test session validation(expiration)

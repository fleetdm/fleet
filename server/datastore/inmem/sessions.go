package inmem

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (d *Datastore) SessionByKey(key string) (*fleet.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	for _, session := range d.sessions {
		if session.Key == key {
			return session, nil
		}
	}
	return nil, notFound("Session")
}

func (d *Datastore) SessionByID(id uint) (*fleet.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if session, ok := d.sessions[id]; ok {
		return session, nil
	}
	return nil, notFound("Session").WithID(id)
}

func (d *Datastore) ListSessionsForUser(id uint) ([]*fleet.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var sessions []*fleet.Session
	for _, session := range d.sessions {
		if session.UserID == id {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (d *Datastore) NewSession(session *fleet.Session) (*fleet.Session, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	session.ID = d.nextID(session)
	d.sessions[session.ID] = session
	if err := d.MarkSessionAccessed(session); err != nil {
		return nil, err
	}

	return session, nil

}

func (d *Datastore) DestroySession(session *fleet.Session) error {
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

func (d *Datastore) MarkSessionAccessed(session *fleet.Session) error {
	session.AccessedAt = time.Now().UTC()
	if _, ok := d.sessions[session.ID]; !ok {
		return notFound("Session").WithID(session.ID)
	}
	d.sessions[session.ID] = session
	return nil
}

// TODO test session validation(expiration)

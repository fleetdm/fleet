package mysql

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (d *Datastore) SessionByKey(key string) (*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
			WHERE ` + "`key`" + ` = ? LIMIT 1
	`
	session := &fleet.Session{}
	err := d.db.Get(session, sqlStatement, key)
	if err != nil {
		return nil, errors.Wrap(err, "selecting sessions")
	}

	return session, nil
}

func (d *Datastore) SessionByID(id uint) (*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE id = ?
		LIMIT 1
	`
	session := &fleet.Session{}
	err := d.db.Get(session, sqlStatement, id)
	if err != nil {
		return nil, errors.Wrap(err, "selecting session by id")
	}

	return session, nil
}

func (d *Datastore) ListSessionsForUser(id uint) ([]*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE user_id = ?
	`
	sessions := []*fleet.Session{}
	err := d.db.Select(&sessions, sqlStatement, id)
	if err != nil {
		return nil, errors.Wrap(err, "selecting sessions for user")
	}

	return sessions, nil

}

func (d *Datastore) NewSession(session *fleet.Session) (*fleet.Session, error) {
	sqlStatement := `
		INSERT INTO sessions (
			user_id,
			` + "`key`" + `
		)
		VALUES(?,?)
	`
	result, err := d.db.Exec(sqlStatement, session.UserID, session.Key)
	if err != nil {
		return nil, errors.Wrap(err, "inserting session")
	}

	id, _ := result.LastInsertId()
	session.ID = uint(id)
	return session, nil
}

func (d *Datastore) DestroySession(session *fleet.Session) error {
	err := d.deleteEntity("sessions", session.ID)
	if err != nil {
		return errors.Wrap(err, "deleting session")
	}

	return nil
}

func (d *Datastore) DestroyAllSessionsForUser(id uint) error {
	sqlStatement := `
		DELETE FROM sessions WHERE user_id = ?
	`
	_, err := d.db.Exec(sqlStatement, id)
	if err != nil {
		return errors.Wrap(err, "deleting sessions for user")
	}

	return nil
}

func (d *Datastore) MarkSessionAccessed(session *fleet.Session) error {
	sqlStatement := `
		UPDATE sessions SET
		accessed_at = ?
		WHERE id = ?
	`
	results, err := d.db.Exec(sqlStatement, d.clock.Now(), session.ID)
	if err != nil {
		return errors.Wrap(err, "updating mark session as accessed")
	}
	rows, err := results.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected updating mark session accessed")
	}
	if rows == 0 {
		return notFound("Session").WithID(session.ID)
	}
	return nil
}

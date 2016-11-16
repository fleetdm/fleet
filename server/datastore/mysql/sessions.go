package mysql

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) SessionByKey(key string) (*kolide.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
			WHERE key = ? LIMIT 1
	`
	session := &kolide.Session{}
	err := d.db.Get(session, sqlStatement, key)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return session, nil
}

func (d *Datastore) SessionByID(id uint) (*kolide.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE id = ?
		LIMIT 1
	`
	session := &kolide.Session{}
	err := d.db.Get(session, sqlStatement, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return session, nil
}

func (d *Datastore) ListSessionsForUser(id uint) ([]*kolide.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE user_id = ?
	`
	sessions := []*kolide.Session{}
	err := d.db.Select(&sessions, sqlStatement, id)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return sessions, nil

}

func (d *Datastore) NewSession(session *kolide.Session) (*kolide.Session, error) {
	sqlStatement := `
		INSERT INTO sessions (
			user_id,
			key
		)
		VALUES(?,?,?,?)
	`
	result, err := d.db.Exec(sqlStatement, session.UserID, session.Key)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	session.ID = uint(id)
	return session, nil
}

func (d *Datastore) DestroySession(session *kolide.Session) error {
	sqlStatement := `
		DELETE FROM sessions WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, session.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) DestroyAllSessionsForUser(id uint) error {
	sqlStatement := `
		DELETE FROM sessions WHERE user_id = ?
	`
	_, err := d.db.Exec(sqlStatement, id)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) MarkSessionAccessed(session *kolide.Session) error {
	sqlStatement := `
		UPDATE sessions SET
		accessed_at = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(sqlStatement, d.clock.Now(), session.ID)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

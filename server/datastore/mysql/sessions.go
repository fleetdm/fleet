package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (d *Datastore) SessionByKey(ctx context.Context, key string) (*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
			WHERE ` + "`key`" + ` = ? LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, d.reader, session, sqlStatement, key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions")
	}

	return session, nil
}

func (d *Datastore) SessionByID(ctx context.Context, id uint) (*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE id = ?
		LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, d.reader, session, sqlStatement, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting session by id")
	}

	return session, nil
}

func (d *Datastore) ListSessionsForUser(ctx context.Context, id uint) ([]*fleet.Session, error) {
	sqlStatement := `
		SELECT * FROM sessions
		WHERE user_id = ?
	`
	sessions := []*fleet.Session{}
	err := sqlx.SelectContext(ctx, d.reader, &sessions, sqlStatement, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions for user")
	}

	return sessions, nil
}

func (d *Datastore) NewSession(ctx context.Context, session *fleet.Session) (*fleet.Session, error) {
	sqlStatement := `
		INSERT INTO sessions (
			user_id,
			` + "`key`" + `
		)
		VALUES(?,?)
	`
	result, err := d.writer.ExecContext(ctx, sqlStatement, session.UserID, session.Key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting session")
	}

	id, _ := result.LastInsertId()
	session.ID = uint(id)
	return session, nil
}

func (d *Datastore) DestroySession(ctx context.Context, session *fleet.Session) error {
	err := d.deleteEntity(ctx, sessionsTable, session.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting session")
	}

	return nil
}

func (d *Datastore) DestroyAllSessionsForUser(ctx context.Context, id uint) error {
	sqlStatement := `
		DELETE FROM sessions WHERE user_id = ?
	`
	_, err := d.writer.ExecContext(ctx, sqlStatement, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting sessions for user")
	}

	return nil
}

func (d *Datastore) MarkSessionAccessed(ctx context.Context, session *fleet.Session) error {
	sqlStatement := `
		UPDATE sessions SET
		accessed_at = ?
		WHERE id = ?
	`
	results, err := d.writer.ExecContext(ctx, sqlStatement, d.clock.Now(), session.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating mark session as accessed")
	}
	rows, err := results.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected updating mark session accessed")
	}
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("Session").WithID(session.ID))
	}
	return nil
}

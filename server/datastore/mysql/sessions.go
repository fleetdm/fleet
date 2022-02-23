package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SessionByKey(ctx context.Context, key string) (*fleet.Session, error) {
	sqlStatement := `
		SELECT s.*, u.api_only FROM sessions s
		LEFT JOIN users u
		ON s.user_id = u.id
		WHERE ` + "s.`key`" + ` = ? LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, ds.reader, session, sqlStatement, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Session").WithName("<key redacted>"))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions")
	}

	return session, nil
}

func (ds *Datastore) SessionByID(ctx context.Context, id uint) (*fleet.Session, error) {
	sqlStatement := `
		SELECT s.*, u.api_only FROM sessions s
		LEFT JOIN users u
		ON s.user_id = u.id
		WHERE s.id = ?
		LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, ds.reader, session, sqlStatement, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Session").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting session by id")
	}

	return session, nil
}

func (ds *Datastore) ListSessionsForUser(ctx context.Context, id uint) ([]*fleet.Session, error) {
	sqlStatement := `
		SELECT s.*, u.api_only FROM sessions s
		INNER JOIN users u
		ON s.user_id = u.id
		WHERE s.user_id = ?
	`
	sessions := []*fleet.Session{}
	err := sqlx.SelectContext(ctx, ds.reader, &sessions, sqlStatement, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions for user")
	}

	return sessions, nil
}

func (ds *Datastore) NewSession(ctx context.Context, session *fleet.Session) (*fleet.Session, error) {
	sqlStatement := `
		INSERT INTO sessions (
			user_id,
			` + "`key`" + `
		)
		VALUES(?,?)
	`
	result, err := ds.writer.ExecContext(ctx, sqlStatement, session.UserID, session.Key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting session")
	}

	id, _ := result.LastInsertId()
	session.ID = uint(id)
	return session, nil
}

func (ds *Datastore) DestroySession(ctx context.Context, session *fleet.Session) error {
	err := ds.deleteEntity(ctx, sessionsTable, session.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting session")
	}

	return nil
}

func (ds *Datastore) DestroyAllSessionsForUser(ctx context.Context, id uint) error {
	sqlStatement := `
		DELETE FROM sessions WHERE user_id = ?
	`
	_, err := ds.writer.ExecContext(ctx, sqlStatement, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting sessions for user")
	}

	return nil
}

func (ds *Datastore) MarkSessionAccessed(ctx context.Context, session *fleet.Session) error {
	sqlStatement := `
		UPDATE sessions SET
		accessed_at = ?
		WHERE id = ?
	`
	results, err := ds.writer.ExecContext(ctx, sqlStatement, ds.clock.Now(), session.ID)
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

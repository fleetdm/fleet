package mysql

import (
	"context"
	"database/sql"
	"errors"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const mfaTokenEntropyInBytes = 32

func (ds *Datastore) SessionByMFAToken(ctx context.Context, token string, sessionKeySize int) (*fleet.Session, *fleet.User, error) {
	var userID uint
	err := sqlx.GetContext(
		ctx,
		ds.reader(ctx),
		&userID,
		"SELECT user_id FROM verification_tokens WHERE token = ? AND created_at >= NOW() - INTERVAL ? SECOND",
		token,
		fleet.MFALinkTTL.Seconds(),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ctxerr.Wrap(ctx, notFound("Verification Token"))
		}
		return nil, nil, err
	}

	user, err := ds.UserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	var session *fleet.Session
	err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if session, err = ds.makeSessionInTransaction(ctx, tx, user.ID, sessionKeySize); err != nil {
			return err
		}

		// only delete token once we've successfully consumed it
		if _, err = tx.ExecContext(ctx, "DELETE FROM verification_tokens WHERE token = ?", token); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return session, user, nil
}

func (ds *Datastore) NewMFAToken(ctx context.Context, userID uint) (string, error) {
	token, err := server.GenerateRandomURLSafeText(mfaTokenEntropyInBytes)
	if err != nil {
		return "", err
	}

	_, err = ds.writer(ctx).ExecContext(
		ctx,
		`INSERT INTO verification_tokens (user_id, token) VALUES (?, ?)`,
		userID,
		token,
	)

	if err != nil {
		return "", err
	}

	return token, nil
}

func (ds *Datastore) SessionByKey(ctx context.Context, key string) (*fleet.Session, error) {
	sqlStatement := `
		SELECT s.*, u.api_only FROM sessions s
		LEFT JOIN users u
		ON s.user_id = u.id
		WHERE ` + "s.`key`" + ` = ? LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), session, sqlStatement, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Session").WithName("<key redacted>"))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions")
	}

	return session, nil
}

func (ds *Datastore) SessionByID(ctx context.Context, id uint) (*fleet.Session, error) {
	return ds.sessionByID(ctx, ds.reader(ctx), id)
}

func (ds *Datastore) sessionByID(ctx context.Context, q sqlx.QueryerContext, id uint) (*fleet.Session, error) {
	sqlStatement := `
		SELECT s.*, u.api_only FROM sessions s
		LEFT JOIN users u
		ON s.user_id = u.id
		WHERE s.id = ?
		LIMIT 1
	`
	session := &fleet.Session{}
	err := sqlx.GetContext(ctx, q, session, sqlStatement, id)
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
	var sessions []*fleet.Session
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &sessions, sqlStatement, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting sessions for user")
	}

	return sessions, nil
}

func (ds *Datastore) NewSession(ctx context.Context, userID uint, sessionKeySize int) (*fleet.Session, error) {
	return ds.makeSessionInTransaction(ctx, ds.writer(ctx), userID, sessionKeySize)
}

func (ds *Datastore) makeSessionInTransaction(ctx context.Context, tx sqlx.ExtContext, userID uint, sessionKeySize int) (*fleet.Session, error) {
	sessionKey, err := server.GenerateRandomText(sessionKeySize)
	if err != nil {
		return nil, err
	}

	sqlStatement := `
		INSERT INTO sessions (
			user_id,
			` + "`key`" + `
		)
		VALUES(?,?)
	`
	result, err := tx.ExecContext(ctx, sqlStatement, userID, sessionKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving session")
	}

	id, _ := result.LastInsertId()           // cannot fail with the mysql driver
	return ds.sessionByID(ctx, tx, uint(id)) //nolint:gosec // dismiss G115
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
	_, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, id)
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
	results, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, ds.clock.Now(), session.ID)
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

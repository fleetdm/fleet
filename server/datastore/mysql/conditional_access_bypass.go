package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ConditionalAccessBypassDevice(ctx context.Context, hostID uint) error {
	const stmt = `
	INSERT INTO
		host_conditional_access (host_id, bypassed_at)
	VALUES
		(?, NOW())
	ON DUPLICATE KEY UPDATE
		bypassed_at = NOW()`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting host conditional bypass")
	}

	return nil
}

func (ds *Datastore) ConditionalAccessConsumeBypass(ctx context.Context, hostID uint) (*time.Time, error) {
	const selectStmt = `
		SELECT
			bypassed_at
		FROM
			host_conditional_access
		WHERE
			host_id = ?
		FOR UPDATE SKIP LOCKED
	`

	const deleteStmt = `
		DELETE FROM
			host_conditional_access
		WHERE
			host_id = ?
	`

	var bypassedAt *time.Time

	if err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var t time.Time
		err := tx.QueryRowxContext(ctx, selectStmt, hostID).Scan(&t)
		if errors.Is(err, sql.ErrNoRows) {
			// There is no conditional access bypass, no rows is not an error
			return nil
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "reading conditional access bypass")
		}

		bypassedAt = &t

		_, err = tx.ExecContext(ctx, deleteStmt, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting conditional access bypass")
		}

		return nil
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "consuming conditional access bypass")
	}

	return bypassedAt, nil
}

func (ds *Datastore) ConditionalAccessClearBypasses(ctx context.Context) error {
	const stmt = `DELETE FROM host_conditional_access`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning all conditional access bypasses")
	}

	return nil
}

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ConditionalAccessBypassDevice(ctx context.Context, hostID uint) error {
	const checkStmt = `
	SELECT
		COUNT(*)
	FROM
		policy_membership pm
	INNER JOIN
		policies p ON pm.policy_id = p.id
	WHERE
		pm.host_id = ?
		AND p.conditional_access_bypass_enabled = 0
		AND pm.passes = 0
	`
	const insertStmt = `
	INSERT INTO
		host_conditional_access (host_id, bypassed_at)
	VALUES
		(?, NOW(6))
	ON DUPLICATE KEY UPDATE
		bypassed_at = NOW(6)`

	var blockCount uint

	if err := sqlx.GetContext(ctx, ds.writer(ctx), &blockCount, checkStmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "checking failing policy count")
	}

	if blockCount != 0 {
		return &fleet.BadRequestError{Message: "host has failing non-bypassable policies"}
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, hostID); err != nil {
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
		err := sqlx.GetContext(ctx, tx, &t, selectStmt, hostID)
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

func (ds *Datastore) ConditionalAccessBypassedAt(ctx context.Context, hostID uint) (*time.Time, error) {
	const stmt = `
		SELECT
			bypassed_at
		FROM
			host_conditional_access
		WHERE
			host_id = ?`

	var bypassedAt time.Time
	err := sqlx.GetContext(ctx, ds.reader(ctx), &bypassedAt, stmt, hostID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host bypass time")
	}

	return &bypassedAt, nil
}

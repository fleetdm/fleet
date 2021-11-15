package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func (d *Datastore) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	lockObtainers := []func(context.Context, string, string, time.Duration) (sql.Result, error){
		d.extendLockIfAlreadyAcquired,
		d.overwriteLockIfExpired,
		d.createLock,
	}

	for _, lockFunc := range lockObtainers {
		res, err := lockFunc(ctx, name, owner, expiration)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "lock")
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "rows affected")
		}
		if rowsAffected > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (d *Datastore) createLock(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.writer.ExecContext(ctx,
		`INSERT IGNORE INTO locks (name, owner, expires_at) VALUES (?, ?, ?)`,
		name, owner, time.Now().Add(expiration),
	)
}

func (d *Datastore) extendLockIfAlreadyAcquired(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.writer.ExecContext(ctx,
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE name = ? and owner = ?`,
		name, owner, time.Now().Add(expiration), name, owner,
	)
}

func (d *Datastore) overwriteLockIfExpired(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.writer.ExecContext(ctx,
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE expires_at < CURRENT_TIMESTAMP and name = ?`,
		name, owner, time.Now().Add(expiration), name,
	)
}

func (d *Datastore) Unlock(ctx context.Context, name string, owner string) error {
	_, err := d.writer.ExecContext(ctx, `DELETE FROM locks WHERE name = ? and owner = ?`, name, owner)
	return err
}

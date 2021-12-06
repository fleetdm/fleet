package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (d *Datastore) DBLocks(ctx context.Context) ([]*fleet.DBLock, error) {
	stmt := `
    SELECT
      r.trx_id              waiting_trx_id,
      r.trx_mysql_thread_id waiting_thread,
      r.trx_query           waiting_query,
      b.trx_id              blocking_trx_id,
      b.trx_mysql_thread_id blocking_thread,
      b.trx_query           blocking_query
    FROM       information_schema.innodb_lock_waits w
    INNER JOIN information_schema.innodb_trx b
      ON b.trx_id = w.blocking_trx_id
    INNER JOIN information_schema.innodb_trx r
      ON r.trx_id = w.requesting_trx_id`

	var locks []*fleet.DBLock
	// Even though this is a Read, use the writer as we want the db locks from
	// the primary database (the read replica should have little to no trx locks).
	if err := d.writer.SelectContext(ctx, &locks, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select locking information")
	}
	return locks, nil
}

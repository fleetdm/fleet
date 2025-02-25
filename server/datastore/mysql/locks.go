package mysql

import (
	"context"
	"database/sql"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

var innodbLockWaitsTableExists atomic.Int64 // Initializes to 0. 0 means we haven't checked yet.

func (ds *Datastore) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	lockObtainers := []func(context.Context, string, string, time.Duration) (sql.Result, error){
		ds.extendLockIfAlreadyAcquired,
		ds.overwriteLockIfExpired,
		ds.createLock,
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

func (ds *Datastore) createLock(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return ds.writer(ctx).ExecContext(ctx,
		`INSERT IGNORE INTO locks (name, owner, expires_at) VALUES (?, ?, ?)`,
		name, owner, time.Now().Add(expiration),
	)
}

func (ds *Datastore) extendLockIfAlreadyAcquired(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return ds.writer(ctx).ExecContext(ctx,
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE name = ? and owner = ?`,
		name, owner, time.Now().Add(expiration), name, owner,
	)
}

func (ds *Datastore) overwriteLockIfExpired(ctx context.Context, name string, owner string, expiration time.Duration) (sql.Result, error) {
	return ds.writer(ctx).ExecContext(ctx,
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE expires_at < CURRENT_TIMESTAMP and name = ?`,
		name, owner, time.Now().Add(expiration), name,
	)
}

func (ds *Datastore) Unlock(ctx context.Context, name string, owner string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE locks SET expires_at = CURRENT_TIMESTAMP WHERE name = ? AND owner = ?`, name, owner)

	return err
}

func (ds *Datastore) DBLocks(ctx context.Context) ([]*fleet.DBLock, error) {
	// information_schema.innodb_lock_waits has been deprecated in MySQL 8, so we need to check if it exists.
	// We only need to check once.
	localInnodbLockWaitsTableExists := innodbLockWaitsTableExists.Load()
	if localInnodbLockWaitsTableExists == 0 {
		var exists bool
		existsStmt := `
		SELECT EXISTS (SELECT *
			FROM information_schema.tables
			WHERE table_schema = 'information_schema'
  			AND table_name = 'innodb_lock_waits')`
		if err := ds.writer(ctx).GetContext(ctx, &exists, existsStmt); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "check for existence of innodb_lock_waits table")
		}
		if exists {
			localInnodbLockWaitsTableExists = 1
		} else {
			localInnodbLockWaitsTableExists = -1
		}
		innodbLockWaitsTableExists.Store(localInnodbLockWaitsTableExists)
	}
	var stmt string
	if localInnodbLockWaitsTableExists == 1 {
		stmt = `
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
	} else {
		// Mapping from information_schema.innodb_lock_waits to performance_schema.data_lock_waits columns:
		//
		// INNODB_LOCK_WAITS data_lock_waits
		// ----------------- ----------------
		// REQUESTING_TRX_ID REQUESTING_ENGINE_TRANSACTION_ID
		// REQUESTED_LOCK_ID REQUESTING_ENGINE_LOCK_ID
		// BLOCKING_TRX_ID   BLOCKING_ENGINE_TRANSACTION_ID
		// BLOCKING_LOCK_ID  BLOCKING_ENGINE_LOCK_ID
		stmt = `
		SELECT
		  r.trx_id              waiting_trx_id,
		  r.trx_mysql_thread_id waiting_thread,
		  r.trx_query           waiting_query,
		  b.trx_id              blocking_trx_id,
		  b.trx_mysql_thread_id blocking_thread,
		  b.trx_query           blocking_query
		FROM       performance_schema.data_lock_waits w
		INNER JOIN information_schema.innodb_trx b
		  ON b.trx_id = w.blocking_engine_transaction_id
		INNER JOIN information_schema.innodb_trx r
		  ON r.trx_id = w.requesting_engine_transaction_id`
	}

	var locks []*fleet.DBLock
	// Even though this is a Read, use the writer as we want the db locks from
	// the primary database (the read replica should have little to no trx locks).
	if err := ds.writer(ctx).SelectContext(ctx, &locks, stmt); err != nil {
		// To read innodb tables, the DB user must have PROCESS and SELECT privileges.
		//
		// This can be set by a DB admin by running:
		//	GRANT PROCESS,SELECT ON *.* TO 'fleet'@'%';
		//	FLUSH PRIVILEGES;
		// Make sure to restart fleet after running the commands above.
		if isMySQLAccessDenied(err) {
			return nil, &accessDeniedError{
				Message:     "select locking information: DB user must have global PROCESS and SELECT privilege",
				InternalErr: err,
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "select locking information")
	}
	return locks, nil
}

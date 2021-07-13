package mysql

import (
	"database/sql"
	"time"
)

func (d *Datastore) Lock(name string, owner string, expiration time.Duration) (bool, error) {
	lockObtainers := []func(string, string, time.Duration) (sql.Result, error){
		d.extendLockIfAlreadyAcquired,
		d.overwriteLockIfExpired,
		d.createLock,
	}

	for _, lockFunc := range lockObtainers {
		res, err := lockFunc(name, owner, expiration)
		if err != nil {
			return false, err
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return false, err
		}
		if rowsAffected > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (d *Datastore) createLock(name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.db.Exec(
		`INSERT IGNORE INTO locks (name, owner, expires_at) VALUES (?, ?, ?)`,
		name, owner, time.Now().Add(expiration),
	)
}

func (d *Datastore) extendLockIfAlreadyAcquired(name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.db.Exec(
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE name = ? and owner = ?`,
		name, owner, time.Now().Add(expiration), name, owner,
	)
}

func (d *Datastore) overwriteLockIfExpired(name string, owner string, expiration time.Duration) (sql.Result, error) {
	return d.db.Exec(
		`UPDATE locks SET name = ?, owner = ?, expires_at = ? WHERE expires_at < CURRENT_TIMESTAMP and name = ?`,
		name, owner, time.Now().Add(expiration), name,
	)
}

func (d *Datastore) Unlock(name string, owner string) error {
	_, err := d.db.Exec(`DELETE FROM locks WHERE name = ? and owner = ?`, name, owner)
	return err
}

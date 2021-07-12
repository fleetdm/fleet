package mysql

import (
	"time"
)

func (d *Datastore) Lock(name string, owner string, expiration time.Duration) (bool, error) {
	lockObtainers := []func(string, string, time.Duration) (int64, error){
		d.extendLockIfAlreadyAcquired,
		d.overwriteLockIfExpired,
		d.createLock,
	}

	for _, lock := range lockObtainers {
		rowsAffected, err := lock(name, owner, expiration)
		if err != nil {
			return false, err
		}
		if rowsAffected > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (d *Datastore) createLock(name string, owner string, expiration time.Duration) (int64, error) {
	res, err := d.db.Exec(
		`INSERT IGNORE INTO locks (name, owner, expires_at) VALUES (?, ?, ?)`,
		name, owner, time.Now().Add(expiration),
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (d *Datastore) extendLockIfAlreadyAcquired(name string, owner string, expiration time.Duration) (int64, error) {
	res, err := d.db.Exec(`
		UPDATE locks SET name = ?, owner = ?, expires_at = ? 
		WHERE name = ? and owner = ?`,
		name, owner, time.Now().Add(expiration), name, owner,
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (d *Datastore) overwriteLockIfExpired(name string, owner string, expiration time.Duration) (int64, error) {
	res, err := d.db.Exec(`
		UPDATE locks SET name = ?, owner = ?, expires_at = ? 
		WHERE expires_at < CURRENT_TIMESTAMP and name = ?`,
		name, owner, time.Now().Add(expiration), name,
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (d *Datastore) Unlock(name string, owner string) error {
	_, err := d.db.Exec(`DELETE FROM locks WHERE name = ? and owner = ?`, name, owner)
	return err
}

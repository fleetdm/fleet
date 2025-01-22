package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/go-kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type encryptionKey struct {
	Base      string `db:"base64_encrypted"`
	Salt      string `db:"base64_encrypted_salt"`
	KeySlot   *uint
	CreatedAt time.Time
	NotFound  bool
}

func (ds *Datastore) SetOrUpdateHostDiskEncryptionKey(ctx context.Context, host *fleet.Host, encryptedBase64Key, clientError string,
	decryptable *bool) error {

	existingKey, err := ds.getExistingHostDiskEncryptionKey(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting existing key, if present")
	}

	// TODO: add salt here
	var keySlot *uint
	var encryptedBase64Salt string
	// We use the same timestamp for base and archive tables so that it can be used as an additional debug tool if needed.
	createdAt := time.Now().UTC()
	var incomingKey = encryptionKey{Base: encryptedBase64Key, Salt: encryptedBase64Salt, KeySlot: keySlot, CreatedAt: createdAt}
	err = ds.archiveHostDiskEncryptionKey(ctx, host, incomingKey, existingKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "archiving key")
	}

	if existingKey.NotFound {
		_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, client_error, decryptable, created_at)
VALUES
  (?, ?, ?, ?, ?)`, host.ID, encryptedBase64Key, clientError, decryptable, createdAt)
		if err == nil {
			return nil
		}
		var mysqlErr *mysql.MySQLError
		switch {
		case errors.As(err, &mysqlErr) && mysqlErr.Number == 1062:
			level.Error(ds.logger).Log("msg", "Primary key already exists in host_disk_encryption_keys. Falling back to update", "host_id",
				host)
			// This should never happen unless there is a bug in the code or an infra issue (like huge replication lag).
		default:
			return ctxerr.Wrap(ctx, err, "inserting key")
		}
	}

	_, err = ds.writer(ctx).ExecContext(ctx, `
UPDATE host_disk_encryption_keys SET
  /* if the key has changed, set decrypted to its initial value so it can be calculated again if necessary (if null) */
  decryptable = IF(
    base64_encrypted = ? AND base64_encrypted != '',
    decryptable,
    ?
  ),
  base64_encrypted = ?,
  client_error = ?
WHERE host_id = ?
`, encryptedBase64Key, decryptable, encryptedBase64Key, clientError, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting key")
	}
	return nil
}

func (ds *Datastore) getExistingHostDiskEncryptionKey(ctx context.Context, host *fleet.Host) (encryptionKey, error) {
	getExistingKeyStmt := `SELECT base64_encrypted, base64_encrypted_salt FROM host_disk_encryption_keys WHERE host_id = ?`
	var existingKey encryptionKey
	err := sqlx.GetContext(ctx, ds.reader(ctx), &existingKey, getExistingKeyStmt, host.ID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// no existing key, proceed to insert
		existingKey.NotFound = true
	case err != nil:
		return encryptionKey{}, ctxerr.Wrap(ctx, err, "getting existing key")
	}
	return existingKey, nil
}

func (ds *Datastore) archiveHostDiskEncryptionKey(ctx context.Context, host *fleet.Host, incomingKey encryptionKey,
	existingKey encryptionKey) error {
	// We archive only valid and different keys to reduce noise.
	if (incomingKey.Base != "" && existingKey.Base != incomingKey.Base) ||
		(incomingKey.Salt != "" && existingKey.Salt != incomingKey.Salt) {
		const insertKeyIntoArchiveStmt = `
INSERT INTO host_disk_encryption_keys_archive (host_id, hardware_serial, base64_encrypted, base64_encrypted_salt, key_slot, created_at)
VALUES (?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, insertKeyIntoArchiveStmt, host.ID, host.HardwareSerial, incomingKey.Base,
			incomingKey.Salt,
			incomingKey.KeySlot, incomingKey.CreatedAt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting key into archive")
		}
	}
	return nil
}

func (ds *Datastore) SaveLUKSData(ctx context.Context, host *fleet.Host, encryptedBase64Passphrase string, encryptedBase64Salt string,
	keySlot uint) error {
	if encryptedBase64Passphrase == "" || encryptedBase64Salt == "" { // should have been caught at service level
		return errors.New("passphrase and salt must be set")
	}

	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, base64_encrypted_salt, key_slot, client_error, decryptable)
VALUES
  (?, ?, ?, ?, '', TRUE)
ON DUPLICATE KEY UPDATE
  decryptable = TRUE,
  base64_encrypted = VALUES(base64_encrypted),
  base64_encrypted_salt = VALUES(base64_encrypted_salt),
  key_slot = VALUES(key_slot),
  client_error = ''
`, host.ID, encryptedBase64Passphrase, encryptedBase64Salt, keySlot)
	return err
}

func (ds *Datastore) IsHostPendingEscrow(ctx context.Context, hostID uint) bool {
	var pendingEscrowCount uint
	_ = sqlx.GetContext(ctx, ds.reader(ctx), &pendingEscrowCount, `
          SELECT COUNT(*) FROM host_disk_encryption_keys WHERE host_id = ? AND reset_requested = TRUE`, hostID)
	return pendingEscrowCount > 0
}

func (ds *Datastore) ClearPendingEscrow(ctx context.Context, hostID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE host_disk_encryption_keys SET reset_requested = FALSE WHERE host_id = ?`, hostID)
	return err
}

func (ds *Datastore) ReportEscrowError(ctx context.Context, hostID uint, errorMessage string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, client_error) VALUES (?, '', ?) ON DUPLICATE KEY UPDATE client_error = VALUES(client_error)
`, hostID, errorMessage)
	return err
}

func (ds *Datastore) QueueEscrow(ctx context.Context, hostID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, reset_requested) VALUES (?, '', TRUE) ON DUPLICATE KEY UPDATE reset_requested = TRUE
`, hostID)
	return err
}

func (ds *Datastore) AssertHasNoEncryptionKeyStored(ctx context.Context, hostID uint) error {
	var hasKeyCount uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hasKeyCount, `
          SELECT COUNT(*) FROM host_disk_encryption_keys WHERE host_id = ? AND base64_encrypted != ''`, hostID)
	if hasKeyCount > 0 {
		return &fleet.BadRequestError{Message: "Key has already been escrowed for this host"}
	}

	return err
}

func (ds *Datastore) GetUnverifiedDiskEncryptionKeys(ctx context.Context) ([]fleet.HostDiskEncryptionKey, error) {
	// NOTE(mna): currently we only verify encryption keys for macOS,
	// Windows/bitlocker uses a different approach where orbit sends the
	// encryption key and we encrypt it server-side with the WSTEP certificate,
	// so it is always decryptable once received.
	//
	// To avoid sending Windows-related keys to verify as part of this call, we
	// only return rows that have a non-empty encryption key (for Windows, the
	// key is blanked if an error occurred trying to retrieve it on the host).
	var keys []fleet.HostDiskEncryptionKey
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &keys, `
          SELECT
            base64_encrypted,
            host_id,
            updated_at
          FROM
            host_disk_encryption_keys
          WHERE
            decryptable IS NULL AND
            base64_encrypted != ''
	`)
	return keys, err
}

func (ds *Datastore) SetHostsDiskEncryptionKeyStatus(
	ctx context.Context,
	hostIDs []uint,
	decryptable bool,
	threshold time.Time,
) error {
	if len(hostIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(
		"UPDATE host_disk_encryption_keys SET decryptable = ? WHERE host_id IN (?) AND updated_at <= ?",
		decryptable, hostIDs, threshold,
	)
	if err != nil {
		return err
	}
	_, err = ds.writer(ctx).ExecContext(ctx, query, args...)
	return err
}

func (ds *Datastore) GetHostDiskEncryptionKey(ctx context.Context, hostID uint) (*fleet.HostDiskEncryptionKey, error) {
	var key fleet.HostDiskEncryptionKey
	err := sqlx.GetContext(ctx, ds.reader(ctx), &key, `
          SELECT
            host_id, base64_encrypted, decryptable, updated_at, client_error
          FROM
            host_disk_encryption_keys
          WHERE host_id = ?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := fmt.Sprintf("for host %d", hostID)
			return nil, ctxerr.Wrap(ctx, notFound("HostDiskEncryptionKey").WithMessage(msg))
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from host_disk_encryption_keys for host_id %d", hostID)
	}
	return &key, nil
}

func (ds *Datastore) CleanupDiskEncryptionKeysOnTeamChange(ctx context.Context, hostIDs []uint, newTeamID *uint) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return cleanupDiskEncryptionKeysOnTeamChangeDB(ctx, tx, hostIDs, newTeamID)
	})
}

func cleanupDiskEncryptionKeysOnTeamChangeDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, newTeamID *uint) error {
	// We are using Apple's encryption profile to determine if any hosts, including Windows and Linux, are encrypted.
	// This is a safe assumption since encryption is enabled for the whole team.
	_, err := getMDMAppleConfigProfileByTeamAndIdentifierDB(ctx, tx, newTeamID, mobileconfig.FleetFileVaultPayloadIdentifier)
	if err != nil {
		if fleet.IsNotFound(err) {
			// the new team does not have a filevault profile so we need to delete the existing ones
			if err := bulkDeleteHostDiskEncryptionKeysDB(ctx, tx, hostIDs); err != nil {
				return ctxerr.Wrap(ctx, err, "reconcile filevault profiles on team change bulk delete host disk encryption keys")
			}
		} else {
			return ctxerr.Wrap(ctx, err, "reconcile filevault profiles on team change get profile")
		}
	}
	return nil
}

func bulkDeleteHostDiskEncryptionKeysDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	deleteStmt, deleteArgs, err := sqlx.In("DELETE FROM host_disk_encryption_keys WHERE host_id IN (?)", hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query")
	}

	_, err = tx.ExecContext(ctx, deleteStmt, deleteArgs...)
	return err
}

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (ds *Datastore) SetOrUpdateHostDiskEncryptionKey(
	ctx context.Context,
	host *fleet.Host,
	encryptedBase64Key,
	clientError string,
	decryptable *bool,
) (bool, error) {
	existingKey, err := ds.getExistingHostDiskEncryptionKey(ctx, host)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "getting existing key, if present")
	}

	// We use the same timestamp for base and archive tables so that it can be used as an additional debug tool if needed.
	incomingKey := encryptionKey{Base: encryptedBase64Key, CreatedAt: time.Now().UTC()}
	archived, err := ds.archiveHostDiskEncryptionKey(ctx, host, incomingKey, existingKey)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "archiving key")
	}

	if existingKey.NotFound {
		_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, client_error, decryptable, created_at)
VALUES
  (?, ?, ?, ?, ?)`, host.ID, incomingKey.Base, clientError, decryptable, incomingKey.CreatedAt)
		if err == nil {
			return archived, nil
		}
		var mysqlErr *mysql.MySQLError
		switch {
		case errors.As(err, &mysqlErr) && mysqlErr.Number == 1062:
			ds.logger.ErrorContext(ctx, "Primary key already exists in host_disk_encryption_keys. Falling back to update", "host_id", host.ID)
			// This should never happen unless there is a bug in the code or an infra issue (like huge replication lag).
		default:
			return false, ctxerr.Wrap(ctx, err, "inserting key")
		}
	}

	// An agent reporting a failure sends no key, and overwriting the stored one with that empty value would take the
	// only recovery key Fleet can show an admin away from a host that is still encrypted. Record the error on its own
	// and leave the key, and its decryptable flag, alone.
	if incomingKey.Base == "" && clientError != "" {
		_, err = ds.writer(ctx).ExecContext(ctx, `
UPDATE host_disk_encryption_keys SET client_error = ? WHERE host_id = ?`, clientError, host.ID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "updating key client error")
		}
		return archived, nil
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
`, incomingKey.Base, decryptable, incomingKey.Base, clientError, host.ID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updating key")
	}
	return archived, nil
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

// archiveHostDiskEncryptionKey archives the existing key into the archive table.
// If the incoming key is different from the existing key, it is archived.
// If the incoming key is the same as the existing key, it is not archived.
// If the incoming key is empty, it is not archived.
// Returns whether the key was archived.
func (ds *Datastore) archiveHostDiskEncryptionKey(
	ctx context.Context,
	host *fleet.Host,
	incomingKey encryptionKey,
	existingKey encryptionKey,
) (bool, error) {
	// We archive only valid and different keys to reduce noise.
	if (incomingKey.Base != "" && existingKey.Base != incomingKey.Base) ||
		(incomingKey.Salt != "" && existingKey.Salt != incomingKey.Salt) {
		const insertKeyIntoArchiveStmt = `
INSERT INTO host_disk_encryption_keys_archive (host_id, hardware_serial, base64_encrypted, base64_encrypted_salt, key_slot, created_at)
VALUES (?, ?, ?, ?, ?, ?)`
		_, err := ds.writer(ctx).ExecContext(ctx, insertKeyIntoArchiveStmt, host.ID, host.HardwareSerial, incomingKey.Base,
			incomingKey.Salt,
			incomingKey.KeySlot, incomingKey.CreatedAt)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "inserting key into archive")
		}
		return true, nil
	}
	return false, nil
}

func (ds *Datastore) DeleteLUKSData(ctx context.Context, hostID, keySlot uint) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `
DELETE FROM host_disk_encryption_keys WHERE host_id = ? AND key_slot = ?`, hostID, keySlot)
		return err
	})
}

func (ds *Datastore) SaveLUKSData(
	ctx context.Context,
	host *fleet.Host,
	encryptedBase64Passphrase string,
	encryptedBase64Salt string,
	keySlot *uint,
) (bool, error) {
	// Salt and key slot are empty/nil for TPM-backed FDE recovery keys, where
	// snapd owns the LUKS key slots; only the passphrase/recovery key itself is
	// guaranteed to be present.
	if encryptedBase64Passphrase == "" { // should have been caught at service level
		return false, errors.New("passphrase must be set")
	}

	existingKey, err := ds.getExistingHostDiskEncryptionKey(ctx, host)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "getting existing LUKS key, if present")
	}

	// We use the same timestamp for base and archive tables so that it can be used as an additional debug tool if needed.
	incomingKey := encryptionKey{
		Base: encryptedBase64Passphrase, Salt: encryptedBase64Salt, KeySlot: keySlot,
		CreatedAt: time.Now().UTC(),
	}
	archived, err := ds.archiveHostDiskEncryptionKey(ctx, host, incomingKey, existingKey)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "archiving LUKS key")
	}

	if existingKey.NotFound {
		_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, base64_encrypted_salt, key_slot, decryptable, created_at)
VALUES
  (?, ?, ?, ?, TRUE, ?)`, host.ID, incomingKey.Base, incomingKey.Salt, incomingKey.KeySlot, incomingKey.CreatedAt)
		if err == nil {
			return archived, nil
		}
		var mysqlErr *mysql.MySQLError
		switch {
		case errors.As(err, &mysqlErr) && mysqlErr.Number == 1062:
			ds.logger.ErrorContext(ctx, "Primary key already exists in LUKS host_disk_encryption_keys. Falling back to update",
				"host_id",
				host)
			// This should never happen unless there is a bug in the code or an infra issue (like huge replication lag).
		default:
			return false, ctxerr.Wrap(ctx, err, "inserting LUKS key")
		}
	}

	_, err = ds.writer(ctx).ExecContext(ctx, `
UPDATE host_disk_encryption_keys SET
  /* if the key has changed, set decrypted to its initial value so it can be calculated again if necessary (if null) */
  decryptable = TRUE,
  base64_encrypted = ?,
  base64_encrypted_salt = ?,
  key_slot = ?,
  client_error = '',
  escrow_sent_at = NULL,
  /* a request still pending once a key exists is a stale duplicate: none are accepted while a key is stored */
  reset_requested = FALSE
WHERE host_id = ?
`, incomingKey.Base, incomingKey.Salt, incomingKey.KeySlot, host.ID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "updating LUKS key")
	}
	return archived, nil
}

func (ds *Datastore) GetHostEscrowState(ctx context.Context, hostID uint) (*fleet.HostEscrowState, error) {
	// client_error is ignored on purpose: a stale error must not hide a retry in flight.
	var row struct {
		Pending     bool   `db:"reset_requested"`
		SinceMicros *int64 `db:"since_micros"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &row, `
SELECT reset_requested, TIMESTAMPDIFF(MICROSECOND, escrow_sent_at, NOW(6)) AS since_micros
FROM host_disk_encryption_keys WHERE host_id = ?`, hostID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return &fleet.HostEscrowState{}, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting host escrow state")
	}
	state := &fleet.HostEscrowState{Pending: row.Pending}
	if row.SinceMicros != nil {
		since := time.Duration(*row.SinceMicros) * time.Microsecond
		state.SinceLastActivity = &since
	}
	return state, nil
}

func (ds *Datastore) MarkEscrowSentToAgent(ctx context.Context, hostID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
UPDATE host_disk_encryption_keys SET reset_requested = FALSE, escrow_sent_at = NOW(6) WHERE host_id = ?`, hostID)
	return err
}

func (ds *Datastore) SetEscrowInFlight(ctx context.Context, hostID uint, inFlight bool) error {
	stmt := `UPDATE host_disk_encryption_keys SET escrow_sent_at = NULL WHERE host_id = ?`
	if inFlight {
		// only a host still in flight is refreshed, so a late report cannot revive a finished escrow
		stmt = `UPDATE host_disk_encryption_keys SET escrow_sent_at = NOW(6) WHERE host_id = ? AND escrow_sent_at IS NOT NULL`
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting in-flight host escrow")
	}
	return nil
}

func (ds *Datastore) ReportEscrowError(ctx context.Context, hostID uint, errorMessage string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO host_disk_encryption_keys
  (host_id, base64_encrypted, client_error) VALUES (?, '', ?)
ON DUPLICATE KEY UPDATE client_error = VALUES(client_error), escrow_sent_at = NULL
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
	host_id, 
	base64_encrypted, 
	base64_encrypted_salt,
	key_slot,
	decryptable, 
	updated_at, 
	client_error
FROM host_disk_encryption_keys
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

func (ds *Datastore) GetHostArchivedDiskEncryptionKey(ctx context.Context, host *fleet.Host) (*fleet.HostArchivedDiskEncryptionKey, error) {
	// TODO: Are we sure that host id is the right way to find the archived key? Are we concerned
	// about cases where host with the same hardware serial has been deleted and recreated? If we
	// learn that this is a real world concern, we should consider using the hardware serial as the primary
	// key (or part of a composite index) for finding archived keys.
	sqlFmt := `
SELECT
	host_id, 
	base64_encrypted, 
	base64_encrypted_salt,
	key_slot,
	created_at
FROM host_disk_encryption_keys_archive
%s
ORDER BY created_at DESC
LIMIT 1`

	var key fleet.HostArchivedDiskEncryptionKey
	err := sqlx.GetContext(ctx, ds.reader(ctx), &key, fmt.Sprintf(sqlFmt, `WHERE host_id = ?`), host.ID)
	if err == sql.ErrNoRows && host.HardwareSerial != "" {
		// If we didn't find a key by host ID, try to find it by hardware serial.
		ds.logger.DebugContext(ctx, "get archived disk encryption key by host serial", "serial", host.HardwareSerial, "host_id", host.ID)
		err = sqlx.GetContext(ctx, ds.reader(ctx), &key, fmt.Sprintf(sqlFmt, `WHERE hardware_serial = ?`), host.HardwareSerial)
	}

	msg := fmt.Sprintf("for host %d with serial %s", host.ID, host.HardwareSerial)
	switch {
	case err == sql.ErrNoRows:
		return nil, ctxerr.Wrap(ctx, notFound("HostDiskEncryptionKey").WithMessage(msg))
	case err != nil:
		return nil, ctxerr.Wrapf(ctx, err, "get archived disk encryption key %s", msg)
	default:
		return &key, nil
	}
}

func (ds *Datastore) IsHostDiskEncryptionKeyArchived(ctx context.Context, hostID uint) (bool, error) {
	// TODO: Are we sure that host id is the right way to find the archived key? Are we concerned
	// about cases where host with the same hardware serial has been deleted and recreated? If we
	// learn that this is a real world concern, we should consider using the hardware serial as the primary
	// key (or part of a composite index) for finding archived keys.
	var exists bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists, `SELECT EXISTS(SELECT 1 FROM host_disk_encryption_keys_archive WHERE host_id = ?)`, hostID); err != nil {
		return false, ctxerr.Wrap(ctx, err, "checking if host disk encryption key is archived")
	}
	return exists, nil
}

func (ds *Datastore) CleanupDiskEncryptionKeysOnTeamChange(ctx context.Context, hostIDs []uint, newTeamID *uint) error {
	diskEncryption, err := ds.GetConfigEnableDiskEncryption(ctx, newTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get destination fleet disk encryption settings")
	}
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return cleanupDiskEncryptionKeysOnTeamChangeDB(ctx, tx, hostIDs, diskEncryption)
	})
}

// cleanupDiskEncryptionKeysOnTeamChangeDB drops the escrowed keys of moved hosts
// whose platform no longer escrows to Fleet in the destination fleet. Each
// platform has its own setting.
func cleanupDiskEncryptionKeysOnTeamChangeDB(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, diskEncryption fleet.DiskEncryptionConfig) error {
	if len(hostIDs) == 0 {
		return nil
	}

	stmt, args, err := sqlx.In(`SELECT id, platform FROM hosts WHERE id IN (?)`, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building host platform query")
	}
	var hosts []struct {
		ID       uint   `db:"id"`
		Platform string `db:"platform"`
	}
	if err := sqlx.SelectContext(ctx, tx, &hosts, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting host platforms")
	}

	toDelete := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		// FleetPlatform() rather than a platform list in SQL, so this agrees with
		// every other per-platform decision in the codebase
		host := fleet.Host{Platform: h.Platform}
		var keep bool
		switch host.FleetPlatform() {
		case "darwin":
			keep = diskEncryption.MacOSEscrowEnabled
		case "windows":
			keep = diskEncryption.WindowsEnabled
		case "linux":
			keep = diskEncryption.LinuxEscrowEnabled
		}
		if !keep {
			toDelete = append(toDelete, h.ID)
		}
	}

	if err := bulkDeleteHostDiskEncryptionKeysDB(ctx, tx, toDelete); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk delete host disk encryption keys on fleet change")
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

/////////////////////////////////////////////////////////////////////////////////
// BitLocker startup PIN handoff
/////////////////////////////////////////////////////////////////////////////////

// setBitLockerPINPendingFlag keeps mdm_windows_enrollments.bitlocker_pin_request_pending in step with
// host_bitlocker_pin_requests. The flag exists so the orbit config check-in can answer "is a PIN waiting?" from the enrollment
// row it already reads every poll.
func setBitLockerPINPendingFlag(ctx context.Context, tx sqlx.ExtContext, hostUUID string, pending bool) error {
	if hostUUID == "" {
		return ctxerr.Wrap(ctx, errors.New("missing host UUID"), "set bitlocker pin request pending flag")
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE mdm_windows_enrollments SET bitlocker_pin_request_pending = ?
WHERE host_uuid = ? ORDER BY created_at DESC, id DESC LIMIT 1`, pending, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set bitlocker pin request pending flag")
	}
	return nil
}

// QueueBitLockerPINRequest stores the end user's encrypted BitLocker startup PIN for the agent to collect.
func (ds *Datastore) QueueBitLockerPINRequest(ctx context.Context, host *fleet.Host, encryptedPIN string) error {
	const stmt = `
INSERT INTO host_bitlocker_pin_requests (host_id, request_uuid, pin_encrypted, status, client_error, created_at, updated_at)
VALUES (?, ?, ?, ?, '', NOW(6), NOW(6))
ON DUPLICATE KEY UPDATE
	request_uuid = VALUES(request_uuid),
	pin_encrypted = VALUES(pin_encrypted),
	status = VALUES(status),
	client_error = '',
	created_at = NOW(6),
	updated_at = NOW(6)`
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// A fresh id per submission is what lets a late outcome for a superseded PIN be recognized and ignored.
		requestID := uuid.NewV7()
		if _, err := tx.ExecContext(ctx, stmt, host.ID, requestID[:], encryptedPIN, fleet.BitLockerPINRequestPending); err != nil {
			return ctxerr.Wrap(ctx, err, "queue bitlocker pin request")
		}
		return setBitLockerPINPendingFlag(ctx, tx, host.UUID, true)
	})
}

// GetBitLockerPINRequest returns where a host's PIN submission stands, for the My device page to poll. It never returns the PIN
// itself, and reports notFound when the host has no submission.
func (ds *Datastore) GetBitLockerPINRequest(ctx context.Context, hostID uint) (*fleet.HostBitLockerPINRequest, error) {
	var req fleet.HostBitLockerPINRequest
	err := sqlx.GetContext(ctx, ds.reader(ctx), &req, `
SELECT status, client_error, created_at, updated_at FROM host_bitlocker_pin_requests WHERE host_id = ?`, hostID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("BitLockerPINRequest").WithID(hostID))
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "get bitlocker pin request")
	}
	return &req, nil
}

// TakeBitLockerPINRequest hands the encrypted PIN to the agent exactly once, then clears it.
func (ds *Datastore) TakeBitLockerPINRequest(ctx context.Context, host *fleet.Host) (string, string, error) {
	var (
		encryptedPIN string
		requestUUID  string
		collected    bool
	)
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var row struct {
			PIN         string `db:"pin_encrypted"`
			RequestUUID []byte `db:"request_uuid"`
		}
		err := sqlx.GetContext(ctx, tx, &row, `
SELECT pin_encrypted, request_uuid
FROM host_bitlocker_pin_requests
WHERE host_id = ?
	AND status = ?
	AND pin_encrypted IS NOT NULL
	AND created_at > DATE_SUB(NOW(6), INTERVAL ? SECOND)
FOR UPDATE`, host.ID, fleet.BitLockerPINRequestPending, int(fleet.BitLockerPINRequestTTL.Seconds()))
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Nothing to collect. Commit the cleared flag rather than returning an error.
			return setBitLockerPINPendingFlag(ctx, tx, host.UUID, false)
		case err != nil:
			return ctxerr.Wrap(ctx, err, "select bitlocker pin request for update")
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE host_bitlocker_pin_requests SET status = ?, pin_encrypted = NULL WHERE host_id = ?`,
			fleet.BitLockerPINRequestDelivered, host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "mark bitlocker pin request delivered")
		}
		if err := setBitLockerPINPendingFlag(ctx, tx, host.UUID, false); err != nil {
			return err
		}

		encryptedPIN, requestUUID, collected = row.PIN, uuid.UUID(row.RequestUUID).String(), true
		return nil
	})
	switch {
	case err != nil:
		return "", "", err
	case !collected:
		return "", "", ctxerr.Wrap(ctx, notFound("BitLockerPINRequest").WithID(host.ID))
	}
	return encryptedPIN, requestUUID, nil
}

// SetBitLockerPINRequestOutcome records what the agent did with the PIN it collected. The row is kept on success
// rather than deleted, so the waiting page has a positive signal to poll for.
func (ds *Datastore) SetBitLockerPINRequestOutcome(
	ctx context.Context, host *fleet.Host, requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string,
) error {
	// The age guard matches HostBitLockerPINRequest.Expired, so an outcome the My device page has already reported as timed
	// out cannot be accepted between the timeout and the hourly cleanup and flip the page back.
	const stmt = `
UPDATE host_bitlocker_pin_requests
SET status = ?, client_error = ?, pin_encrypted = NULL
WHERE host_id = ? AND request_uuid = ? AND status = ?
	AND updated_at > DATE_SUB(NOW(6), INTERVAL ? SECOND)`
	// The id comes from the agent. Make sure it is valid.
	requestID, err := uuid.Parse(requestUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, notFound("BitLockerPINRequest").WithID(host.ID))
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt, outcome, clientError, host.ID, requestID[:], fleet.BitLockerPINRequestDelivered,
			int(fleet.BitLockerPINResultTimeout.Seconds()))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "set bitlocker pin request outcome")
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "rows affected setting bitlocker pin request outcome")
		}
		if affected == 0 {
			return ctxerr.Wrap(ctx, notFound("BitLockerPINRequest").WithID(host.ID))
		}
		// Both outcomes are terminal, so there is nothing left for the agent to collect.
		return setBitLockerPINPendingFlag(ctx, tx, host.UUID, false)
	})
}

// bitLockerPINRequestRetention is how long a finished PIN submission is kept so the My device page can show its outcome.
const bitLockerPINRequestRetention = 24 * time.Hour

// CleanupExpiredBitLockerPINRequests runs on the hourly cleanups cron. It retires submissions the agent never collected, and
// ones it collected but never reported on, and clears the enrollment flag that pointed at them. It also deletes finished
// submissions a day after they finish.
func (ds *Datastore) CleanupExpiredBitLockerPINRequests(ctx context.Context) error {
	// Collecting a PIN sets status to delivered, which moves updated_at to the collection time.
	const expireStmt = `
UPDATE host_bitlocker_pin_requests
SET status = ?, pin_encrypted = NULL, client_error = ?
WHERE (status = ? AND created_at <= DATE_SUB(NOW(6), INTERVAL ? SECOND))
	OR (status = ? AND updated_at <= DATE_SUB(NOW(6), INTERVAL ? SECOND))`
	if _, err := ds.writer(ctx).ExecContext(ctx, expireStmt, fleet.BitLockerPINRequestFailed, fleet.BitLockerPINRequestTimedOutError,
		fleet.BitLockerPINRequestPending, int(fleet.BitLockerPINRequestTTL.Seconds()),
		fleet.BitLockerPINRequestDelivered, int(fleet.BitLockerPINResultTimeout.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "expire unfinished bitlocker pin requests")
	}

	// Retiring a submission above does not touch the enrollment row, so clear the flag for any host whose submission is no
	// longer collectable.
	const clearFlagStmt = `
UPDATE host_bitlocker_pin_requests r
JOIN hosts h ON h.id = r.host_id
JOIN mdm_windows_enrollments e ON e.host_uuid = h.uuid
SET e.bitlocker_pin_request_pending = 0
WHERE r.status != ? AND e.bitlocker_pin_request_pending = 1`
	if _, err := ds.writer(ctx).ExecContext(ctx, clearFlagStmt, fleet.BitLockerPINRequestPending); err != nil {
		return ctxerr.Wrap(ctx, err, "clear stale bitlocker pin pending flags")
	}

	const reapStmt = `
DELETE FROM host_bitlocker_pin_requests
WHERE status IN (?, ?) AND updated_at <= DATE_SUB(NOW(6), INTERVAL ? SECOND)`
	if _, err := ds.writer(ctx).ExecContext(ctx, reapStmt, fleet.BitLockerPINRequestSet, fleet.BitLockerPINRequestFailed,
		int(bitLockerPINRequestRetention.Seconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "reap finished bitlocker pin requests")
	}
	return nil
}

// DeleteBitLockerPINRequest drops a host's PIN submission.
func (ds *Datastore) DeleteBitLockerPINRequest(ctx context.Context, host *fleet.Host) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM host_bitlocker_pin_requests WHERE host_id = ?`, host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "delete bitlocker pin request")
		}
		return setBitLockerPINPendingFlag(ctx, tx, host.UUID, false)
	})
}

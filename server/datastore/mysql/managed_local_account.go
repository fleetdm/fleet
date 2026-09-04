package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SaveHostManagedLocalAccount(ctx context.Context, hostUUID, plaintextPassword, commandUUID string) error {
	encrypted, err := encrypt([]byte(plaintextPassword), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypting managed local account password")
	}

	const stmt = `
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password),
			command_uuid = VALUES(command_uuid),
			status = NULL,
			account_uuid = NULL,
			pending_encrypted_password = NULL,
			pending_command_uuid = NULL,
			auto_rotate_at = NULL,
			client_error = '',
			deleted = 0
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, encrypted, commandUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "save host managed local account")
	}
	return nil
}

// SaveHostManagedLocalAccountFromEscrow stores a device-generated password for hosts where fleetd creates the account (Windows).
func (ds *Datastore) SaveHostManagedLocalAccountFromEscrow(ctx context.Context, hostUUID, plaintextPassword string) error {
	encrypted, err := encrypt([]byte(plaintextPassword), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypting managed local account password")
	}

	const stmt = `
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, NULL, ?)
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password),
			command_uuid = NULL,
			status = VALUES(status),
			account_uuid = NULL,
			pending_encrypted_password = NULL,
			pending_command_uuid = NULL,
			auto_rotate_at = NULL,
			client_error = '',
			deleted = 0
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, encrypted, fleet.MDMDeliveryVerified); err != nil {
		return ctxerr.Wrap(ctx, err, "save host managed local account from escrow")
	}
	return nil
}

// ReportManagedLocalAccountEscrowError records a device-reported failure to create the managed local account, mirroring
// ReportEscrowError for disk encryption keys.
func (ds *Datastore) ReportManagedLocalAccountEscrowError(ctx context.Context, hostUUID, clientError string) error {
	const stmt = `
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status, client_error)
		VALUES (?, NULL, NULL, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			client_error = VALUES(client_error),
			-- Revive a soft-deleted row so the failure is visible instead of the host merely looking unresponsive.
			-- The retained password is deliberately NOT cleared: soft delete exists to keep it recoverable.
			deleted = 0
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, fleet.MDMDeliveryFailed, clientError); err != nil {
		return ctxerr.Wrap(ctx, err, "report managed local account escrow error")
	}
	return nil
}

// softDeleteManagedLocalAccountPasswordDB retires the escrowed password for a host without destroying it
func softDeleteManagedLocalAccountPasswordDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	const stmt = `
		UPDATE host_managed_local_account_passwords
		SET deleted = 1,
		    pending_encrypted_password = NULL,
		    pending_command_uuid = NULL,
		    auto_rotate_at = NULL
		WHERE host_uuid = ? AND deleted = 0
	`
	if _, err := tx.ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "soft delete managed local account password")
	}
	return nil
}

func (ds *Datastore) GetHostManagedLocalAccountPassword(ctx context.Context, hostUUID string) (*fleet.HostManagedLocalAccountPassword, error) {
	const stmt = `SELECT encrypted_password, updated_at FROM host_managed_local_account_passwords WHERE host_uuid = ? AND deleted = 0`

	var row struct {
		EncryptedPassword []byte    `db:"encrypted_password"`
		UpdatedAt         time.Time `db:"updated_at"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccountPassword").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting managed local account password")
	}

	// Treat a missing password as no record at all.
	if len(row.EncryptedPassword) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccountPassword").
			WithMessage(fmt.Sprintf("for host %s", hostUUID)))
	}

	decrypted, err := decrypt(row.EncryptedPassword, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting managed local account password")
	}

	return &fleet.HostManagedLocalAccountPassword{
		Username:  fleet.ManagedLocalAccountUsername,
		Password:  string(decrypted),
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (ds *Datastore) GetHostManagedLocalAccountStatus(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
	const stmt = `
		SELECT
			status,
			client_error,
			encrypted_password IS NOT NULL AS has_password,
			pending_encrypted_password IS NOT NULL AS pending_rotation,
			auto_rotate_at,
			-- Windows stages nothing server-side (fleetd generates the password on the device), so its in-flight
			-- rotation lives on the enrollment row instead. Correlated rather than joined so re-enrolled hosts, which
			-- keep one row per enrollment, resolve to the current one; NULL for every non-Windows host.
			(
				SELECT e.managed_local_account_rotation_requested
				FROM mdm_windows_enrollments e
				WHERE e.host_uuid = host_managed_local_account_passwords.host_uuid
				ORDER BY e.created_at DESC, e.id DESC
				LIMIT 1
			) AS rotation_requested,
			-- LEFT so a managed local account row is never hidden by a missing hosts row; platform only decides how a
			-- failed rotation is presented, and NULL falls through to the stricter macOS handling below.
			h.platform
		FROM host_managed_local_account_passwords
		LEFT JOIN hosts h ON h.uuid = host_managed_local_account_passwords.host_uuid
		WHERE host_uuid = ? AND deleted = 0
	`

	var row struct {
		Status            *string    `db:"status"`
		ClientError       string     `db:"client_error"`
		HasPassword       bool       `db:"has_password"`
		PendingRotation   bool       `db:"pending_rotation"`
		AutoRotateAt      *time.Time `db:"auto_rotate_at"`
		RotationRequested *bool      `db:"rotation_requested"`
		Platform          *string    `db:"platform"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting managed local account status")
	}

	// NULL in DB means the AccountConfiguration command is still pending (not yet acked).
	// Once any rotation lifecycle event has fired, the column carries a real status string.
	status := "pending"
	if row.Status != nil {
		status = *row.Status
	}
	// password_available is decoupled from rotation lifecycle: a viewed-and-waiting row
	// still has a usable password even though status='pending'.
	//
	// Windows follows the recovery lock password: having a password is enough. A rotation that fails on the device left
	// the old password in place, so it still logs the admin in, and hiding it defeats the point of a break-glass
	// account. macOS still hides a failed row; a separate story covers bringing it in line.
	passwordAvailable := row.HasPassword
	isWindows := row.Platform != nil && fleet.IsWindowsPlatform(*row.Platform)
	if !isWindows && status == string(fleet.MDMDeliveryFailed) {
		passwordAvailable = false
	}
	// Either platform's in-flight rotation reads as pending to callers: macOS stages a pending password, Windows carries
	// an outstanding request on its enrollment.
	pendingRotation := row.PendingRotation || (row.RotationRequested != nil && *row.RotationRequested)
	return &fleet.HostMDMManagedLocalAccount{
		Status:            &status,
		Detail:            row.ClientError,
		PasswordAvailable: passwordAvailable,
		AutoRotateAt:      row.AutoRotateAt,
		PendingRotation:   pendingRotation,
	}, nil
}

func (ds *Datastore) SetHostManagedLocalAccountStatus(ctx context.Context, hostUUID string, status fleet.MDMDeliveryStatus) error {
	const stmt = `UPDATE host_managed_local_account_passwords SET status = ? WHERE host_uuid = ? AND deleted = 0`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, status, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set managed local account status")
	}
	return nil
}

func (ds *Datastore) GetManagedLocalAccountUUID(ctx context.Context, hostUUID string) (*string, error) {
	const stmt = `SELECT account_uuid FROM host_managed_local_account_passwords WHERE host_uuid = ? AND deleted = 0`

	var accountUUID *string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &accountUUID, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("ManagedLocalAccount").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "get managed local account uuid")
	}
	return accountUUID, nil
}

func (ds *Datastore) SetManagedLocalAccountUUID(ctx context.Context, hostUUID, accountUUID string) error {
	const stmt = `
		UPDATE host_managed_local_account_passwords
		SET account_uuid = ?
		WHERE host_uuid = ? AND deleted = 0 AND (account_uuid IS NULL OR account_uuid <> ?)`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, accountUUID, hostUUID, accountUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set managed local account uuid")
	}
	return nil
}

func (ds *Datastore) GetManagedLocalAccountByCommandUUID(ctx context.Context, commandUUID string) (*fleet.Host, error) {
	return ds.lookupManagedLocalAccountHost(ctx, "command_uuid", commandUUID)
}

func (ds *Datastore) GetManagedLocalAccountByPendingCommandUUID(ctx context.Context, commandUUID string) (*fleet.Host, error) {
	return ds.lookupManagedLocalAccountHost(ctx, "pending_command_uuid", commandUUID)
}

// lookupManagedLocalAccountHost shares the join-to-hosts lookup used by both the
// AccountConfiguration ack (matches command_uuid) and the SetAutoAdminPassword ack
// (matches pending_command_uuid). The column name is interpolated, not parameterized,
// because callers pass a fixed identifier — never untrusted input.
func (ds *Datastore) lookupManagedLocalAccountHost(ctx context.Context, column, commandUUID string) (*fleet.Host, error) {
	stmt := fmt.Sprintf(`SELECT host_uuid FROM host_managed_local_account_passwords WHERE %s = ? AND deleted = 0`, column)

	var hostUUID string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostUUID, stmt, commandUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("ManagedLocalAccount").
				WithMessage(fmt.Sprintf("for command %s", commandUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting managed local account by command uuid")
	}

	const hostStmt = `SELECT id FROM hosts WHERE uuid = ?`

	var hostID uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostID, hostStmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host id by host uuid")
	}
	host, err := ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host")
	}
	return host, nil
}

// MarkManagedLocalAccountPasswordViewed sets the auto-rotation deadline on first view.
// The conditional UPDATE only fires when auto_rotate_at IS NULL, so subsequent views
// inside the window do not extend the timer. The pre-existing rotateAt is read back
// in either case so callers can show the deadline to the user.
func (ds *Datastore) MarkManagedLocalAccountPasswordViewed(ctx context.Context, hostUUID string) (time.Time, error) {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET status = '%s',
		    auto_rotate_at = NOW(6) + INTERVAL 65 MINUTE,
		    initiated_by_fleet = 1
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND auto_rotate_at IS NULL
		  AND encrypted_password IS NOT NULL
		  AND (status IS NULL OR status <> '%s')
		  AND pending_encrypted_password IS NULL
	`, fleet.MDMDeliveryPending, fleet.MDMDeliveryFailed)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return time.Time{}, ctxerr.Wrap(ctx, err, "mark managed local account password viewed")
	}

	// Read back the (possibly pre-existing) auto_rotate_at. If the row is ineligible
	// (no password, status=failed, rotation pending, or no row at all) the read
	// returns either NULL or sql.ErrNoRows — both surface as notFound to the caller.
	const selectStmt = `
		SELECT auto_rotate_at
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND encrypted_password IS NOT NULL
		  AND (status IS NULL OR status <> ?)
	`
	var rotateAt *time.Time
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &rotateAt, selectStmt, hostUUID, fleet.MDMDeliveryFailed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return time.Time{}, ctxerr.Wrap(ctx, err, "read managed local account auto_rotate_at")
	}
	if rotateAt == nil {
		// Update was a no-op AND the row didn't have a pre-existing auto_rotate_at —
		// the only way to land here is if pending_encrypted_password IS NOT NULL
		// (a rotation is in flight). Treat that as not-eligible-for-view.
		return time.Time{}, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").
			WithMessage(fmt.Sprintf("for host %s", hostUUID)))
	}
	return *rotateAt, nil
}

// InitiateManagedLocalAccountRotation stages a rotation by writing the encrypted
// pending password and the command UUID. Returns typed errors when the row is not
// eligible so callers (manual request-based vs auto-rotation cron) can react differently.
func (ds *Datastore) InitiateManagedLocalAccountRotation(ctx context.Context, hostUUID, pendingPlaintextPassword, cmdUUID string) error {
	encryptedPassword, err := encrypt([]byte(pendingPlaintextPassword), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypt pending managed local account password")
	}

	// auto_rotate_at is cleared once the command is staged: the API surfaces
	// auto_rotate_at as a "rotation will fire at" hint, but once a rotation is in
	// flight the hint is stale (the row is now waiting on the device ack instead
	// of the cron). Complete/Fail also clear auto_rotate_at; this just covers the
	// pending-but-unacked window between enqueue and ack.
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET pending_encrypted_password = ?,
		    pending_command_uuid = ?,
		    auto_rotate_at = NULL,
		    status = '%s'
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND encrypted_password IS NOT NULL
		  AND account_uuid IS NOT NULL
		  AND (status IS NULL OR status <> '%s')
		  AND pending_encrypted_password IS NULL
	`, fleet.MDMDeliveryPending, fleet.MDMDeliveryFailed)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, encryptedPassword, cmdUUID, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "initiate managed local account rotation")
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		return nil
	}

	// Diagnose the cause to give the caller a typed error.
	var dest struct {
		HasPassword bool           `db:"has_password"`
		HasUUID     bool           `db:"has_uuid"`
		HasPending  bool           `db:"has_pending"`
		Status      sql.NullString `db:"status"`
	}
	const checkStmt = `
		SELECT
			encrypted_password IS NOT NULL AS has_password,
			account_uuid IS NOT NULL AS has_uuid,
			pending_encrypted_password IS NOT NULL AS has_pending,
			status
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?
		  AND deleted = 0
	`
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &dest, checkStmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return ctxerr.Wrap(ctx, err, "check managed local account rotation eligibility")
	}

	if dest.HasPending {
		return ctxerr.Wrap(ctx, fleet.ErrManagedLocalAccountRotationPending, fmt.Sprintf("host %s", hostUUID))
	}
	return ctxerr.Wrap(ctx, fleet.ErrManagedLocalAccountNotEligible, fmt.Sprintf("host %s (status=%v has_password=%v has_uuid=%v)",
		hostUUID, dest.Status.String, dest.HasPassword, dest.HasUUID))
}

// MarkManagedLocalAccountRotationDeferred records a manual rotation that couldn't
// be enqueued because account_uuid was not yet captured. auto_rotate_at=NOW(6) makes
// the cron pick it up on the next tick (after the UUID is captured by osquery), and
// initiated_by_fleet=0 tells the cron *not* to re-log the activity (the manual path
// already logged it with the user as actor at click time).
func (ds *Datastore) MarkManagedLocalAccountRotationDeferred(ctx context.Context, hostUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET status = '%s',
		    auto_rotate_at = NOW(6),
		    initiated_by_fleet = 0
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND encrypted_password IS NOT NULL
		  AND (status IS NULL OR status <> '%s')
		  AND pending_encrypted_password IS NULL
	`, fleet.MDMDeliveryPending, fleet.MDMDeliveryFailed)

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "mark managed local account rotation deferred")
	}
	return nil
}

// ClearManagedLocalAccountRotation undoes the pending columns set by Initiate; the
// caller uses this on a non-APNs commander failure (the command was never enqueued).
// Status is left as 'pending' because at this point we can't safely call it 'verified'
// without re-reading the row — the next view or rotation will reset it.
func (ds *Datastore) ClearManagedLocalAccountRotation(ctx context.Context, hostUUID string) error {
	const stmt = `
		UPDATE host_managed_local_account_passwords
		SET pending_encrypted_password = NULL,
		    pending_command_uuid = NULL
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND pending_encrypted_password IS NOT NULL
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "clear managed local account rotation")
	}
	return nil
}

// CompleteManagedLocalAccountRotation moves pending → current and clears all
// rotation lifecycle columns. The cmdUUID match guards against an ack landing on
// a row that has since started a different rotation (defense in depth — the unique
// pending_command_uuid should make this impossible in practice).
func (ds *Datastore) CompleteManagedLocalAccountRotation(ctx context.Context, hostUUID, cmdUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET encrypted_password = pending_encrypted_password,
		    command_uuid = pending_command_uuid,
		    pending_encrypted_password = NULL,
		    pending_command_uuid = NULL,
		    status = '%s',
		    auto_rotate_at = NULL,
		    initiated_by_fleet = 0
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND pending_encrypted_password IS NOT NULL
		  AND pending_command_uuid = ?
	`, fleet.MDMDeliveryVerified)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, cmdUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "complete managed local account rotation")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("ManagedLocalAccountPendingRotation").
			WithMessage(fmt.Sprintf("for host %s command %s", hostUUID, cmdUUID)))
	}
	return nil
}

// FailManagedLocalAccountRotation marks the rotation failed and clears pending columns.
// encrypted_password (the still-known-good password) stays in place so the user can
// continue to view it; auto_rotate_at is cleared so we don't keep retrying a failed
// rotation on the cron.
func (ds *Datastore) FailManagedLocalAccountRotation(ctx context.Context, hostUUID, cmdUUID, errorMessage string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET pending_encrypted_password = NULL,
		    pending_command_uuid = NULL,
		    status = '%s',
		    auto_rotate_at = NULL,
		    initiated_by_fleet = 0
		WHERE host_uuid = ?
		  AND deleted = 0
		  AND pending_command_uuid = ?
	`, fleet.MDMDeliveryFailed)

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, cmdUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fail managed local account rotation")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("ManagedLocalAccountPendingRotation").
			WithMessage(fmt.Sprintf("for host %s command %s (error=%q)", hostUUID, cmdUUID, errorMessage)))
	}
	return nil
}

// GetManagedLocalAccountsForAutoRotation returns rows the cron should rotate.
// Eligibility:
//   - auto_rotate_at has elapsed
//   - account_uuid captured (we need it to address the specific account)
//   - encrypted_password present (we need a "current" password to rotate from)
//   - no pending rotation already
//   - status is not 'failed' — note we DO accept 'pending' (a viewed-and-waiting row)
//
// initiated_by_fleet is returned alongside so the cron can skip activity logging
// for deferred manual rotations (which were logged at click time).
func (ds *Datastore) GetManagedLocalAccountsForAutoRotation(ctx context.Context) ([]fleet.HostManagedLocalAccountAutoRotationInfo, error) {
	stmt := fmt.Sprintf(`
		SELECT
			hmlap.host_uuid,
			h.id AS host_id,
			COALESCE(NULLIF(h.computer_name, ''), h.hostname) AS display_name,
			hmlap.account_uuid,
			hmlap.initiated_by_fleet
		FROM host_managed_local_account_passwords hmlap
		JOIN hosts h ON h.uuid = hmlap.host_uuid
		WHERE hmlap.deleted = 0
		  AND hmlap.auto_rotate_at IS NOT NULL
		  AND hmlap.auto_rotate_at <= NOW(6)
		  AND hmlap.account_uuid IS NOT NULL
		  AND hmlap.encrypted_password IS NOT NULL
		  AND hmlap.pending_encrypted_password IS NULL
		  AND (hmlap.status IS NULL OR hmlap.status <> '%s')
		LIMIT 100
	`, fleet.MDMDeliveryFailed)

	var hosts []fleet.HostManagedLocalAccountAutoRotationInfo
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get managed local accounts for auto rotation")
	}
	return hosts, nil
}

// GetWindowsManagedLocalAccountsForAutoRotation returns the Windows rows the cron should rotate. It is separate from
// GetManagedLocalAccountsForAutoRotation because the eligibility differs on both ends: Windows rows never capture an
// account_uuid (nothing addresses the account by id), and "already rotating" is an outstanding request on the enrollment
// rather than a staged pending password.
//
// Eligibility:
//   - auto_rotate_at has elapsed
//   - encrypted_password present, so there is a current password to replace
//   - the host is Windows and has a Windows MDM enrollment to carry the notification
//   - no rotation already requested on that enrollment
//   - status is not 'failed'; 'pending' is accepted, being a viewed-and-waiting row
func (ds *Datastore) GetWindowsManagedLocalAccountsForAutoRotation(ctx context.Context) ([]fleet.HostManagedLocalAccountWindowsRotationInfo, error) {
	stmt := fmt.Sprintf(`
		SELECT
			hmlap.host_uuid,
			h.id AS host_id,
			COALESCE(NULLIF(h.computer_name, ''), h.hostname) AS display_name,
			hmlap.initiated_by_fleet
		FROM host_managed_local_account_passwords hmlap
		JOIN hosts h ON h.uuid = hmlap.host_uuid AND h.platform = 'windows'
		JOIN mdm_windows_enrollments e ON e.host_uuid = hmlap.host_uuid
		WHERE hmlap.deleted = 0
		  AND hmlap.auto_rotate_at IS NOT NULL
		  AND hmlap.auto_rotate_at <= NOW(6)
		  AND hmlap.encrypted_password IS NOT NULL
		  AND (hmlap.status IS NULL OR hmlap.status <> '%s')
		  AND e.managed_local_account_rotation_requested = 0
		  AND e.id = (
			SELECT e2.id FROM mdm_windows_enrollments e2
			WHERE e2.host_uuid = hmlap.host_uuid
			ORDER BY e2.created_at DESC, e2.id DESC
			LIMIT 1
		  )
		LIMIT 100
	`, fleet.MDMDeliveryFailed)

	var hosts []fleet.HostManagedLocalAccountWindowsRotationInfo
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get windows managed local accounts for auto rotation")
	}
	return hosts, nil
}

// InitiateWindowsManagedLocalAccountRotation asks the device to re-provision its managed local account, the Windows
// counterpart to InitiateManagedLocalAccountRotation. There is no pending password to stage: fleetd generates one on the
// device, so all that is recorded is the request itself, plus clearing auto_rotate_at so the cron does not pick the row
// up again while the device works through it.
//
// Both writes land in one transaction: a request the device never sees, or a cleared timer with no request behind it,
// would each strand the row. Returns ErrManagedLocalAccountRotationPending when a rotation is already outstanding, and
// notFound when the host has no Windows MDM enrollment or no managed local account row.
func (ds *Datastore) InitiateWindowsManagedLocalAccountRotation(ctx context.Context, hostUUID string) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Eligibility is read rather than inferred from an UPDATE's row count: MySQL reports rows *changed*, so a row
		// already sitting at status='pending' with a spent timer would look ineligible when it is merely unchanged.
		var acct struct {
			HasPassword bool           `db:"has_password"`
			Status      sql.NullString `db:"status"`
		}
		switch err := sqlx.GetContext(ctx, tx, &acct, `
			SELECT encrypted_password IS NOT NULL AS has_password, status
			FROM host_managed_local_account_passwords
			WHERE host_uuid = ? AND deleted = 0
		`, hostUUID); {
		case errors.Is(err, sql.ErrNoRows):
			return ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		case err != nil:
			return ctxerr.Wrap(ctx, err, "check windows managed local account rotation eligibility")
		}
		// A previous failure does not block a fresh request. Windows keeps a failed row's password visible, so the admin
		// can see it and ask again; refusing here would leave them an enabled button that always errors. Only the
		// auto-rotation cron still skips failed rows, so a standing cause is not retried every tick unasked.
		if !acct.HasPassword {
			return ctxerr.Wrap(ctx, fleet.ErrManagedLocalAccountNotEligible,
				fmt.Sprintf("host %s (has_password=false status=%v)", hostUUID, acct.Status.String))
		}

		res, err := tx.ExecContext(ctx,
			`UPDATE mdm_windows_enrollments SET managed_local_account_rotation_requested = 1
			 WHERE host_uuid = ? AND managed_local_account_rotation_requested = 0
			 ORDER BY created_at DESC, id DESC LIMIT 1`,
			hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "request windows managed local account rotation")
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			// Nothing changed: either a rotation is already outstanding, or there is no enrollment to ask.
			var requested bool
			switch err := sqlx.GetContext(ctx, tx, &requested,
				`SELECT managed_local_account_rotation_requested FROM mdm_windows_enrollments
				 WHERE host_uuid = ? ORDER BY created_at DESC, id DESC LIMIT 1`, hostUUID); {
			case errors.Is(err, sql.ErrNoRows):
				return ctxerr.Wrap(ctx, notFound("MDMWindowsEnrolledDevice").WithMessage(hostUUID))
			case err != nil:
				return ctxerr.Wrap(ctx, err, "check windows managed local account rotation request")
			}
			return ctxerr.Wrap(ctx, fleet.ErrManagedLocalAccountRotationPending, fmt.Sprintf("host %s", hostUUID))
		}

		// Hand the row to the device: status reflects "waiting on the host", and the timer is spent. client_error is
		// cleared too, so a previous failure's message does not sit next to a rotation that is now under way.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE host_managed_local_account_passwords
			SET status = '%s', auto_rotate_at = NULL, client_error = ''
			WHERE host_uuid = ? AND deleted = 0
		`, fleet.MDMDeliveryPending), hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "mark windows managed local account rotation pending")
		}
		return nil
	})
}

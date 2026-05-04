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
			account_uuid = NULL
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, encrypted, commandUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "save host managed local account")
	}
	return nil
}

func (ds *Datastore) GetHostManagedLocalAccountPassword(ctx context.Context, hostUUID string) (*fleet.HostManagedLocalAccountPassword, error) {
	const stmt = `SELECT encrypted_password, updated_at FROM host_managed_local_account_passwords WHERE host_uuid = ?`

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
			encrypted_password IS NOT NULL AS has_password,
			pending_encrypted_password IS NOT NULL AS pending_rotation,
			auto_rotate_at
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?
	`

	var row struct {
		Status          *string    `db:"status"`
		HasPassword     bool       `db:"has_password"`
		PendingRotation bool       `db:"pending_rotation"`
		AutoRotateAt    *time.Time `db:"auto_rotate_at"`
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
	// still has a usable password even though status='pending'. Only 'failed' (and the
	// initial-config NULL — encrypted_password not yet stored) hides the password.
	passwordAvailable := row.HasPassword && status != string(fleet.MDMDeliveryFailed)
	return &fleet.HostMDMManagedLocalAccount{
		Status:            &status,
		PasswordAvailable: passwordAvailable,
		AutoRotateAt:      row.AutoRotateAt,
		PendingRotation:   row.PendingRotation,
	}, nil
}

func (ds *Datastore) SetHostManagedLocalAccountStatus(ctx context.Context, hostUUID string, status fleet.MDMDeliveryStatus) error {
	const stmt = `UPDATE host_managed_local_account_passwords SET status = ? WHERE host_uuid = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, status, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set managed local account status")
	}
	return nil
}

func (ds *Datastore) GetManagedLocalAccountUUID(ctx context.Context, hostUUID string) (*string, error) {
	const stmt = `SELECT account_uuid FROM host_managed_local_account_passwords WHERE host_uuid = ?`

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
		WHERE host_uuid = ? AND (account_uuid IS NULL OR account_uuid <> ?)`

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
	stmt := fmt.Sprintf(`SELECT host_uuid FROM host_managed_local_account_passwords WHERE %s = ?`, column)

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
//
// The 65 minute window is the spec'd value (1h5m, providing a buffer over the 1h
// shown in the UI so the cron has time to actually fire before the user expects it).
func (ds *Datastore) MarkManagedLocalAccountPasswordViewed(ctx context.Context, hostUUID string) (time.Time, error) {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET status = '%s',
		    auto_rotate_at = NOW(6) + INTERVAL 65 MINUTE,
		    initiated_by_fleet = 1
		WHERE host_uuid = ?
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
// eligible so callers (manual EE service vs auto-rotation cron) can react differently.
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
//
// Idempotent — repeated calls with the row already in the deferred state are no-ops.
func (ds *Datastore) MarkManagedLocalAccountRotationDeferred(ctx context.Context, hostUUID string) error {
	stmt := fmt.Sprintf(`
		UPDATE host_managed_local_account_passwords
		SET status = '%s',
		    auto_rotate_at = NOW(6),
		    initiated_by_fleet = 0
		WHERE host_uuid = ?
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
		WHERE hmlap.auto_rotate_at IS NOT NULL
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

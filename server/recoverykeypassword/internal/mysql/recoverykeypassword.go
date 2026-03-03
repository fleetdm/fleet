// Package mysql provides the MySQL datastore implementation for recovery key passwords.
package mysql

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the recovery key password datastore.
type Datastore struct {
	primary          *sqlx.DB
	replica          *sqlx.DB
	serverPrivateKey string
	logger           *slog.Logger
}

// NewDatastore creates a new MySQL datastore for recovery key passwords.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{
		primary:          conns.Primary,
		replica:          conns.Replica,
		serverPrivateKey: conns.Options.PrivateKey,
		logger:           logger,
	}
}

// Ensure Datastore implements the interface
var _ recoverykeypassword.Datastore = (*Datastore)(nil)

// SetHostRecoveryKeyPassword generates a new recovery key password,
// encrypts it, and stores it for the given host.
func (ds *Datastore) SetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (string, error) {
	password, err := recoverykeypassword.GeneratePassword()
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generating recovery key password")
	}

	encrypted, err := encrypt([]byte(password), ds.serverPrivateKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "encrypting recovery key password")
	}

	const stmt = `
		INSERT INTO host_recovery_key_passwords (host_id, encrypted_password)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password)
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, hostID, encrypted); err != nil {
		return "", ctxerr.Wrap(ctx, err, "storing recovery key password")
	}

	return password, nil
}

// GetHostRecoveryKeyPassword retrieves and decrypts the recovery key password.
func (ds *Datastore) GetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (*recoverykeypassword.HostRecoveryKeyPassword, error) {
	const stmt = `SELECT encrypted_password, updated_at FROM host_recovery_key_passwords WHERE host_id = ?`

	var row struct {
		EncryptedPassword []byte    `db:"encrypted_password"`
		UpdatedAt         time.Time `db:"updated_at"`
	}
	if err := sqlx.GetContext(ctx, ds.replica, &row, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, platform_mysql.NotFound("HostRecoveryKeyPassword").
				WithMessage(fmt.Sprintf("for host %d", hostID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting recovery key password")
	}

	decrypted, err := decrypt(row.EncryptedPassword, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting recovery key password")
	}

	return &recoverykeypassword.HostRecoveryKeyPassword{
		Password:  string(decrypted),
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func encrypt(plainText []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

func decrypt(encrypted []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

// GetHostsForRecoveryLockAction returns hosts that need recovery lock password action.
func (ds *Datastore) GetHostsForRecoveryLockAction(ctx context.Context) ([]recoverykeypassword.HostRecoveryLockAction, error) {
	// Query hosts that:
	// - Have enable_recovery_lock_password = true (from team config or appconfig for no-team hosts)
	// - Are macOS 11.5+ (version check via operating_systems table)
	// - Are MDM enrolled (enabled = 1 and device enrollment type)
	// - Have no recovery lock password record yet
	// Note: hosts with existing records (pending, verifying, verified, failed) are NOT included
	const stmt = `
		SELECT h.id, h.uuid, h.team_id, rkp.status
		FROM hosts h
		JOIN nano_enrollments ne ON ne.device_id = h.uuid
		JOIN host_operating_system hos ON hos.host_id = h.id
		JOIN operating_systems os ON os.id = hos.os_id
		LEFT JOIN teams t ON t.id = h.team_id
		CROSS JOIN app_config_json ac
		LEFT JOIN host_recovery_key_passwords rkp ON rkp.host_id = h.id
		WHERE os.platform = 'darwin'
		  AND ne.enabled = 1
		  AND ne.type IN ('Device', 'User Enrollment (Device)')
		  AND (
		      -- Team hosts: check team config
		      (h.team_id IS NOT NULL AND JSON_EXTRACT(t.config, '$.mdm.enable_recovery_lock_password') = true)
		      OR
		      -- No-team hosts: check appconfig
		      (h.team_id IS NULL AND JSON_EXTRACT(ac.json_value, '$.mdm.enable_recovery_lock_password') = true)
		  )
		  AND (
		      -- macOS 11.5+ version check (os.version is e.g., "15.7", "11.5.2")
		      CAST(SUBSTRING_INDEX(os.version, '.', 1) AS UNSIGNED) > 11
		      OR (
		          CAST(SUBSTRING_INDEX(os.version, '.', 1) AS UNSIGNED) = 11
		          AND CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(os.version, '.', 2), '.', -1) AS UNSIGNED) >= 5
		      )
		  )
		  AND rkp.host_id IS NULL
		LIMIT 500
	`

	var results []struct {
		HostID   uint                     `db:"id"`
		HostUUID string                   `db:"uuid"`
		TeamID   *uint                    `db:"team_id"`
		Status   *fleet.MDMDeliveryStatus `db:"status"`
	}

	if err := sqlx.SelectContext(ctx, ds.replica, &results, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	hosts := make([]recoverykeypassword.HostRecoveryLockAction, len(results))
	for i, r := range results {
		hosts[i] = recoverykeypassword.HostRecoveryLockAction{
			HostID:   r.HostID,
			HostUUID: r.HostUUID,
			TeamID:   r.TeamID,
			Status:   r.Status,
		}
	}

	return hosts, nil
}

// SetRecoveryLockPending sets the recovery lock status to pending with the given set command UUID.
func (ds *Datastore) SetRecoveryLockPending(ctx context.Context, hostID uint, setCommandUUID string) error {
	const stmt = `
		UPDATE host_recovery_key_passwords
		SET status = 'pending',
		    set_command_uuid = ?,
		    verify_command_uuid = NULL,
		    set_command_ack_at = NULL,
		    error_message = NULL
		WHERE host_id = ?
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, setCommandUUID, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock pending")
	}

	return nil
}

// SetRecoveryLockVerifying marks the SetRecoveryLock command as acknowledged and updates
// status to verifying with the given verify command UUID.
func (ds *Datastore) SetRecoveryLockVerifying(ctx context.Context, hostID uint, verifyCommandUUID string) error {
	const stmt = `
		UPDATE host_recovery_key_passwords
		SET status = 'verifying',
		    verify_command_uuid = ?,
		    set_command_ack_at = CURRENT_TIMESTAMP(6)
		WHERE host_id = ?
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, verifyCommandUUID, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock verifying")
	}

	return nil
}

// SetRecoveryLockVerified marks the recovery lock as verified.
func (ds *Datastore) SetRecoveryLockVerified(ctx context.Context, hostID uint) error {
	const stmt = `
		UPDATE host_recovery_key_passwords
		SET status = 'verified',
		    error_message = NULL
		WHERE host_id = ?
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock verified")
	}

	return nil
}

// SetRecoveryLockFailed marks the recovery lock as failed with the given error message.
func (ds *Datastore) SetRecoveryLockFailed(ctx context.Context, hostID uint, errorMsg string) error {
	const stmt = `
		UPDATE host_recovery_key_passwords
		SET status = 'failed',
		    error_message = ?
		WHERE host_id = ?
	`

	if _, err := ds.primary.ExecContext(ctx, stmt, errorMsg, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "set recovery lock failed")
	}

	return nil
}

// GetHostIDByVerifyCommandUUID returns the host ID associated with a VerifyRecoveryLock command UUID.
func (ds *Datastore) GetHostIDByVerifyCommandUUID(ctx context.Context, verifyCommandUUID string) (uint, error) {
	const stmt = `SELECT host_id FROM host_recovery_key_passwords WHERE verify_command_uuid = ?`

	var hostID uint
	if err := sqlx.GetContext(ctx, ds.replica, &hostID, stmt, verifyCommandUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, platform_mysql.NotFound("HostRecoveryKeyPassword").
				WithMessage(fmt.Sprintf("for verify_command_uuid %s", verifyCommandUUID)))
		}
		return 0, ctxerr.Wrap(ctx, err, "get host id by verify command uuid")
	}

	return hostID, nil
}

// GetPendingRecoveryLockHosts returns hosts with status='pending' along with
// the SetRecoveryLock command result status from nano_command_results.
func (ds *Datastore) GetPendingRecoveryLockHosts(ctx context.Context) ([]recoverykeypassword.HostPendingRecoveryLock, error) {
	const stmt = `
		SELECT
			rkp.host_id,
			h.uuid AS host_uuid,
			rkp.set_command_uuid,
			COALESCE(ncr.status, '') AS set_command_status,
			COALESCE(ncr.result, '') AS set_command_error_info
		FROM host_recovery_key_passwords rkp
		JOIN hosts h ON h.id = rkp.host_id
		LEFT JOIN nano_command_results ncr ON ncr.command_uuid = rkp.set_command_uuid AND ncr.id = h.uuid
		WHERE rkp.status = 'pending'
	`

	var results []struct {
		HostID              uint   `db:"host_id"`
		HostUUID            string `db:"host_uuid"`
		SetCommandUUID      string `db:"set_command_uuid"`
		SetCommandStatus    string `db:"set_command_status"`
		SetCommandErrorInfo string `db:"set_command_error_info"`
	}

	if err := sqlx.SelectContext(ctx, ds.replica, &results, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get pending recovery lock hosts")
	}

	hosts := make([]recoverykeypassword.HostPendingRecoveryLock, len(results))
	for i, r := range results {
		hosts[i] = recoverykeypassword.HostPendingRecoveryLock{
			HostID:              r.HostID,
			HostUUID:            r.HostUUID,
			SetCommandUUID:      r.SetCommandUUID,
			SetCommandStatus:    r.SetCommandStatus,
			SetCommandErrorInfo: r.SetCommandErrorInfo,
		}
	}

	return hosts, nil
}

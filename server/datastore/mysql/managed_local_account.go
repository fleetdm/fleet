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
	const stmt = `SELECT status FROM host_managed_local_account_passwords WHERE host_uuid = ?`

	var dbStatus *string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dbStatus, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostManagedLocalAccount").
				WithMessage(fmt.Sprintf("for host %s", hostUUID)))
		}
		return nil, ctxerr.Wrap(ctx, err, "getting managed local account status")
	}

	// NULL in DB means the command is pending (not yet acknowledged).
	status := "pending"
	if dbStatus != nil {
		status = *dbStatus
	}
	return &fleet.HostMDMManagedLocalAccount{
		Status:            &status,
		PasswordAvailable: status == string(fleet.MDMDeliveryVerified),
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
	const stmt = `SELECT host_uuid FROM host_managed_local_account_passwords WHERE command_uuid = ?`

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

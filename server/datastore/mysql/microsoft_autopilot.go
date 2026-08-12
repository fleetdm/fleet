package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// microsoftGraphCredentialRow mirrors the mdm_microsoft_graph_credentials table. The secret is read as the raw encrypted
// blob and decrypted into fleet.MicrosoftGraphCredential.ClientSecret, so the encrypted form never escapes this file.
type microsoftGraphCredentialRow struct {
	TenantID          string     `db:"tenant_id"`
	ClientID          string     `db:"client_id"`
	ClientSecret      []byte     `db:"client_secret"`
	CredentialInvalid bool       `db:"credential_invalid"`
	LastSyncedAt      *time.Time `db:"last_synced_at"`
	LastSyncError     *string    `db:"last_sync_error"`
}

func (r microsoftGraphCredentialRow) toCredential(secret string) *fleet.MicrosoftGraphCredential {
	return &fleet.MicrosoftGraphCredential{
		TenantID:          r.TenantID,
		ClientID:          r.ClientID,
		ClientSecret:      secret,
		CredentialInvalid: r.CredentialInvalid,
		LastSyncedAt:      r.LastSyncedAt,
		LastSyncError:     r.LastSyncError,
	}
}

// ListMicrosoftGraphCredentials returns every stored Graph credential with its client secret decrypted.
func (ds *Datastore) ListMicrosoftGraphCredentials(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
	const stmt = `
SELECT tenant_id, client_id, client_secret, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
ORDER BY tenant_id`

	var rows []microsoftGraphCredentialRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credentials")
	}

	creds := make([]*fleet.MicrosoftGraphCredential, 0, len(rows))
	for _, row := range rows {
		secret, err := decrypt(row.ClientSecret, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decrypt microsoft graph client secret for tenant %s", row.TenantID)
		}
		creds = append(creds, row.toCredential(string(secret)))
	}
	return creds, nil
}

// ListMicrosoftGraphCredentialMetadata returns the stored credentials without their client secrets.
func (ds *Datastore) ListMicrosoftGraphCredentialMetadata(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
	const stmt = `
SELECT tenant_id, client_id, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
ORDER BY tenant_id`

	var creds []*fleet.MicrosoftGraphCredential
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &creds, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credential metadata")
	}
	return creds, nil
}

// ReplaceMicrosoftGraphCredentials reconciles the stored credentials in one transaction: every credential in upsert is
// stored, and every tenant in deleteTenantIDs is removed.
func (ds *Datastore) ReplaceMicrosoftGraphCredentials(
	ctx context.Context,
	upsert []*fleet.MicrosoftGraphCredential,
	deleteTenantIDs []string,
) error {
	if len(upsert) == 0 && len(deleteTenantIDs) == 0 {
		return nil
	}

	// Encrypt before opening the transaction so a bad private key fails without holding one open.
	encryptedSecrets := make([][]byte, len(upsert))
	for i, cred := range upsert {
		encrypted, err := encrypt([]byte(cred.ClientSecret), ds.serverPrivateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "encrypt microsoft graph client secret with datastore.serverPrivateKey")
		}
		encryptedSecrets[i] = encrypted
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Storing a credential resets all of its sync state.
		const upsertStmt = `
INSERT INTO mdm_microsoft_graph_credentials (tenant_id, client_id, client_secret)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
	client_id = VALUES(client_id),
	client_secret = VALUES(client_secret),
	last_synced_at = NULL,
	credential_invalid = 0,
	last_sync_error = NULL`

		for i, cred := range upsert {
			if _, err := tx.ExecContext(ctx, upsertStmt, cred.TenantID, cred.ClientID, encryptedSecrets[i]); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert microsoft graph credential")
			}
		}

		if len(deleteTenantIDs) > 0 {
			deleteStmt, args, err := sqlx.In(`DELETE FROM mdm_microsoft_graph_credentials WHERE tenant_id IN (?)`, deleteTenantIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build delete microsoft graph credentials statement")
			}
			if _, err := tx.ExecContext(ctx, deleteStmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete microsoft graph credentials")
			}
		}

		return nil
	})
}

// SetMicrosoftGraphCredentialInvalid flips the per-tenant flag reporting that a credential needs an admin's attention.
func (ds *Datastore) SetMicrosoftGraphCredentialInvalid(ctx context.Context, tenantID string, invalid bool) (bool, error) {
	const stmt = `UPDATE mdm_microsoft_graph_credentials SET credential_invalid = ? WHERE tenant_id = ? AND credential_invalid != ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, invalid, tenantID, invalid)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "update mdm_microsoft_graph_credentials credential_invalid")
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// RecordMicrosoftGraphSyncResult stamps the outcome of a sync pass for a tenant. A nil syncErr records a success and
// clears any previous error; a non-nil one records the message for display alongside the credential.
func (ds *Datastore) RecordMicrosoftGraphSyncResult(ctx context.Context, tenantID string, syncErr *string) error {
	const stmt = `UPDATE mdm_microsoft_graph_credentials SET last_synced_at = NOW(6), last_sync_error = ? WHERE tenant_id = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, syncErr, tenantID); err != nil {
		return ctxerr.Wrap(ctx, err, "record microsoft graph sync result")
	}
	return nil
}

const hostAutopilotDeviceColumns = 6

// hostAutopilotDeviceBatchSize is how many devices go into one INSERT statement. At 6 placeholders each that is 6k per
// statement, well under MySQL's 65535 limit. A var so tests can shrink it and exercise the batch boundary.
var hostAutopilotDeviceBatchSize = 1000

// BatchUpsertHostAutopilotDevices stores the Autopilot metadata for many hosts. Re-inserting after a soft delete clears
// deleted_at, so a device that leaves and later rejoins the Autopilot list becomes live again rather than staying
// invisible behind a stale tombstone.
func (ds *Datastore) BatchUpsertHostAutopilotDevices(ctx context.Context, devices []*fleet.HostAutopilotDevice) error {
	const stmt = `
INSERT INTO host_autopilot_devices
	(host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id)
VALUES %s
ON DUPLICATE KEY UPDATE
	autopilot_device_id = VALUES(autopilot_device_id),
	entra_device_id = VALUES(entra_device_id),
	group_tag = VALUES(group_tag),
	hardware_serial = VALUES(hardware_serial),
	tenant_id = VALUES(tenant_id),
	deleted_at = NULL`

	err := common_mysql.BatchProcessSimple(devices, hostAutopilotDeviceBatchSize, func(batch []*fleet.HostAutopilotDevice) error {
		args := make([]any, 0, len(batch)*hostAutopilotDeviceColumns)
		for _, dev := range batch {
			args = append(args, dev.HostID, dev.AutopilotDeviceID, dev.EntraDeviceID, dev.GroupTag, dev.HardwareSerial, dev.TenantID)
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(batch)), ",")

		if _, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(stmt, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})

	return ctxerr.Wrap(ctx, err, "batch upsert host autopilot devices")
}

// BatchSoftDeleteHostAutopilotDevices tombstones the Autopilot records for the given hosts, marking the devices as no
// longer present in the tenant's Autopilot registry. The host row itself is untouched: a device that is deregistered
// from Autopilot stops being a pending host, but an already-enrolled host keeps reporting in.
func (ds *Datastore) BatchSoftDeleteHostAutopilotDevices(ctx context.Context, hostIDs []uint) error {
	const stmt = `
UPDATE host_autopilot_devices
SET deleted_at = NOW(6)
WHERE deleted_at IS NULL AND host_id IN (?)`

	err := common_mysql.BatchProcessSimple(hostIDs, hostAutopilotDeviceBatchSize, func(batch []uint) error {
		expanded, args, err := sqlx.In(stmt, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build IN clause")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, expanded, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec batch")
		}
		return nil
	})

	return ctxerr.Wrap(ctx, err, "batch soft delete host autopilot devices")
}

// ListHostAutopilotDevices returns the live (not soft-deleted) Autopilot records for a tenant.
func (ds *Datastore) ListHostAutopilotDevices(ctx context.Context, tenantID string) ([]*fleet.HostAutopilotDevice, error) {
	const stmt = `
SELECT host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id
FROM host_autopilot_devices
WHERE tenant_id = ? AND deleted_at IS NULL
ORDER BY host_id`

	var devices []*fleet.HostAutopilotDevice
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &devices, stmt, tenantID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host autopilot devices")
	}
	return devices, nil
}

// GetHostAutopilotDevice returns the live Autopilot record for a host, or a not-found error. A soft-deleted record
// reads as not found: the device has left Autopilot, and only the host survives.
func (ds *Datastore) GetHostAutopilotDevice(ctx context.Context, hostID uint) (*fleet.HostAutopilotDevice, error) {
	const stmt = `
SELECT host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id
FROM host_autopilot_devices
WHERE host_id = ? AND deleted_at IS NULL`

	var device fleet.HostAutopilotDevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &device, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostAutopilotDevice").WithID(hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host autopilot device")
	}
	return &device, nil
}

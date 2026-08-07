package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

// GetMicrosoftGraphCredential returns the credential for a tenant, with its client secret decrypted.
func (ds *Datastore) GetMicrosoftGraphCredential(ctx context.Context, tenantID string) (*fleet.MicrosoftGraphCredential, error) {
	const stmt = `
SELECT tenant_id, client_id, client_secret, credential_invalid, last_synced_at, last_sync_error
FROM mdm_microsoft_graph_credentials
WHERE tenant_id = ?`

	var row microsoftGraphCredentialRow
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MicrosoftGraphCredential").WithName(tenantID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get microsoft graph credential")
	}

	secret, err := decrypt(row.ClientSecret, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypt microsoft graph client secret")
	}
	return row.toCredential(string(secret)), nil
}

// UpsertMicrosoftGraphCredential stores a credential, encrypting the client secret with the server private key.
func (ds *Datastore) UpsertMicrosoftGraphCredential(ctx context.Context, cred *fleet.MicrosoftGraphCredential) error {
	encryptedSecret, err := encrypt([]byte(cred.ClientSecret), ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypt microsoft graph client secret with datastore.serverPrivateKey")
	}

	const stmt = `
INSERT INTO mdm_microsoft_graph_credentials (tenant_id, client_id, client_secret)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
	client_id = VALUES(client_id),
	client_secret = VALUES(client_secret)`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, cred.TenantID, cred.ClientID, encryptedSecret); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert microsoft graph credential")
	}
	return nil
}

// DeleteMicrosoftGraphCredential removes the credential for a tenant.
func (ds *Datastore) DeleteMicrosoftGraphCredential(ctx context.Context, tenantID string) error {
	const stmt = `DELETE FROM mdm_microsoft_graph_credentials WHERE tenant_id = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, tenantID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete microsoft graph credential")
	}
	return nil
}

// SetMicrosoftGraphCredentialInvalid flips the flag that drives the app-wide "credential needs attention" banner.
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

// UpsertHostAutopilotDevice stores the Autopilot metadata for a host. Re-inserting after a soft delete clears
// deleted_at, so a device that leaves and later rejoins the Autopilot list becomes live again rather than staying
// invisible behind a stale tombstone.
func (ds *Datastore) UpsertHostAutopilotDevice(ctx context.Context, dev *fleet.HostAutopilotDevice) error {
	const stmt = `
INSERT INTO host_autopilot_devices
	(host_id, autopilot_device_id, entra_device_id, group_tag, hardware_serial, tenant_id)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	autopilot_device_id = VALUES(autopilot_device_id),
	entra_device_id = VALUES(entra_device_id),
	group_tag = VALUES(group_tag),
	hardware_serial = VALUES(hardware_serial),
	tenant_id = VALUES(tenant_id),
	deleted_at = NULL`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		dev.HostID, dev.AutopilotDeviceID, dev.EntraDeviceID, dev.GroupTag, dev.HardwareSerial, dev.TenantID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host autopilot device")
	}
	return nil
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

package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// SetOrUpdatePSSODevice replaces (or creates) a host's PSSO registration in a
// single transaction: upserts the device row, deletes any stale KeyID rows for
// the host, then inserts the two new KeyID rows (signing + encryption).
func (ds *Datastore) SetOrUpdatePSSODevice(
	ctx context.Context,
	device fleet.PSSODevice,
	signKeyID fleet.PSSOKeyID,
	encKeyID fleet.PSSOKeyID,
) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		const upsertDevice = `
			INSERT INTO mdm_apple_psso_devices
				(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				device_uuid        = VALUES(device_uuid),
				signing_key_pem    = VALUES(signing_key_pem),
				encryption_key_pem = VALUES(encryption_key_pem),
				key_exchange_key   = VALUES(key_exchange_key)
		`
		if _, err := tx.ExecContext(ctx, upsertDevice,
			device.HostID,
			device.DeviceUUID,
			device.SigningKeyPEM,
			device.EncryptionKeyPEM,
			device.KeyExchangeKey,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert psso device")
		}

		if _, err := tx.ExecContext(ctx,
			`DELETE FROM mdm_apple_psso_key_ids WHERE host_id = ?`,
			device.HostID,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "clear existing psso key_ids")
		}

		const insertKeyID = `
			INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
			VALUES (?, ?, ?, ?)
		`
		for _, k := range []fleet.PSSOKeyID{signKeyID, encKeyID} {
			if _, err := tx.ExecContext(ctx, insertKeyID, k.KID, k.HostID, k.KeyType, k.PEM); err != nil {
				return ctxerr.Wrap(ctx, err, "insert psso key_id")
			}
		}
		return nil
	})
}

// GetPSSODeviceByKeyID resolves a kid back to its owning device and the
// specific KeyID row that matched (so callers know whether they're holding the
// signing or encryption side of the device's keypair).
func (ds *Datastore) GetPSSODeviceByKeyID(ctx context.Context, kid string) (*fleet.PSSODevice, *fleet.PSSOKeyID, error) {
	type joined struct {
		// device columns
		HostID           uint   `db:"host_id"`
		DeviceUUID       string `db:"device_uuid"`
		SigningKeyPEM    string `db:"signing_key_pem"`
		EncryptionKeyPEM string `db:"encryption_key_pem"`
		KeyExchangeKey   []byte `db:"key_exchange_key"`
		DeviceCreatedAt  []byte `db:"device_created_at"`
		DeviceUpdatedAt  []byte `db:"device_updated_at"`
		// key_id columns
		KID         string             `db:"kid"`
		KeyType     fleet.PSSOKeyType  `db:"key_type"`
		PEM         string             `db:"pem"`
		KIDCreated  []byte             `db:"kid_created_at"`
	}

	const stmt = `
		SELECT
			d.host_id            AS host_id,
			d.device_uuid        AS device_uuid,
			d.signing_key_pem    AS signing_key_pem,
			d.encryption_key_pem AS encryption_key_pem,
			d.key_exchange_key   AS key_exchange_key,
			d.created_at         AS device_created_at,
			d.updated_at         AS device_updated_at,
			k.kid                AS kid,
			k.key_type           AS key_type,
			k.pem                AS pem,
			k.created_at         AS kid_created_at
		FROM mdm_apple_psso_key_ids k
		JOIN mdm_apple_psso_devices d ON d.host_id = k.host_id
		WHERE k.kid = ?
	`

	var row joined
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, kid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ctxerr.Wrap(ctx, notFound("PSSOKeyID").WithName(kid))
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get psso device by kid")
	}

	device := &fleet.PSSODevice{
		HostID:           row.HostID,
		DeviceUUID:       row.DeviceUUID,
		SigningKeyPEM:    row.SigningKeyPEM,
		EncryptionKeyPEM: row.EncryptionKeyPEM,
		KeyExchangeKey:   row.KeyExchangeKey,
	}
	keyID := &fleet.PSSOKeyID{
		KID:     row.KID,
		HostID:  row.HostID,
		KeyType: row.KeyType,
		PEM:     row.PEM,
	}
	return device, keyID, nil
}

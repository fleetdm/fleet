package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// SetOrUpdatePSSODevice upserts a host's PSSO registration: the device row
// plus the given key rows in a single transaction. Keys are upserted by kid;
// keys from earlier registrations are left in place so they keep working.
func (ds *Datastore) SetOrUpdatePSSODevice(ctx context.Context, hostUUID string, keys []fleet.PSSOKey) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		const upsertDevice = `
			INSERT INTO mdm_apple_psso_devices (host_uuid)
			VALUES (?)
			ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP(6)
		`
		if _, err := tx.ExecContext(ctx, upsertDevice, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert psso device")
		}

		// kid is the global primary key of mdm_apple_psso_keys and the token
		// endpoint resolves a device's host from it, so a kid must never move
		// between hosts. Reject a registration reusing a kid another host owns
		// instead of letting the upsert silently reassign it. FOR UPDATE gap-locks
		// the kid so a concurrent registration can't insert it between this check
		// and the upsert below.
		const ownerOfKID = `SELECT host_uuid FROM mdm_apple_psso_keys WHERE kid = ? FOR UPDATE`
		for _, k := range keys {
			var owner string
			switch err := sqlx.GetContext(ctx, tx, &owner, ownerOfKID, k.KID); {
			case errors.Is(err, sql.ErrNoRows):
				// Unclaimed kid: safe to insert.
			case err != nil:
				return ctxerr.Wrap(ctx, err, "check psso key owner")
			case owner != hostUUID:
				return ctxerr.Wrap(ctx, &fleet.ConflictError{Message: "psso key id is already registered to another host"})
			}
		}

		const upsertKey = `
			INSERT INTO mdm_apple_psso_keys (kid, host_uuid, key_type, pem)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				host_uuid = VALUES(host_uuid),
				key_type  = VALUES(key_type),
				pem       = VALUES(pem)
		`
		for _, k := range keys {
			if _, err := tx.ExecContext(ctx, upsertKey, k.KID, hostUUID, k.KeyType, k.PEM); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert psso key")
			}
		}
		return nil
	})
}

func (ds *Datastore) GetPSSODevice(ctx context.Context, hostUUID string) (*fleet.PSSODevice, error) {
	const stmt = `
		SELECT host_uuid, created_at, updated_at
		FROM mdm_apple_psso_devices
		WHERE host_uuid = ?
	`
	var device fleet.PSSODevice
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &device, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("PSSODevice").WithName(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get psso device")
	}
	return &device, nil
}

func (ds *Datastore) GetPSSOKey(ctx context.Context, kid string) (*fleet.PSSOKey, error) {
	const stmt = `
		SELECT kid, host_uuid, key_type, pem, created_at, updated_at
		FROM mdm_apple_psso_keys
		WHERE kid = ?
	`
	var key fleet.PSSOKey
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &key, stmt, kid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("PSSOKey").WithName(kid))
		}
		return nil, ctxerr.Wrap(ctx, err, "get psso key")
	}
	return &key, nil
}

func (ds *Datastore) ListPSSOKeys(ctx context.Context, hostUUID string) ([]*fleet.PSSOKey, error) {
	const stmt = `
		SELECT kid, host_uuid, key_type, pem, created_at, updated_at
		FROM mdm_apple_psso_keys
		WHERE host_uuid = ?
		ORDER BY created_at DESC, kid
	`
	var keys []*fleet.PSSOKey
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &keys, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list psso keys")
	}
	return keys, nil
}

// DeletePSSODevice clears a host's PSSO registration; the keys cascade.
func (ds *Datastore) DeletePSSODevice(ctx context.Context, hostUUID string) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM mdm_apple_psso_devices WHERE host_uuid = ?`, hostUUID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delete psso device")
	}
	return nil
}

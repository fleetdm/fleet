package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/jmoiron/sqlx"
)

// Most of the code for the host identity feature is located at ./ee/server/service/hostidentity

func (ds *Datastore) GetHostIdentityCertBySerialNumber(ctx context.Context, serialNumber uint64) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, fmt.Sprintf(`
		SELECT serial, host_id, name, not_valid_after, public_key_raw
		FROM host_identity_scep_certificates
		WHERE serial = %d
			AND not_valid_after > NOW()
			AND revoked = 0`, serialNumber))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("host identity certificate")
	case err != nil:
		return nil, err
	}
	return &hostIdentityCert, nil
}

func (ds *Datastore) UpdateHostIdentityCertHostIDBySerial(ctx context.Context, serialNumber uint64, hostID uint) error {
	return common_mysql.WithRetryTxx(ctx, ds.writer(ctx), func(tx sqlx.ExtContext) error {
		return updateHostIdentityCertHostIDBySerial(ctx, tx, hostID, serialNumber)
	}, ds.logger)
}

func updateHostIdentityCertHostIDBySerial(ctx context.Context, tx sqlx.ExtContext, hostID uint, serialNumber uint64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE host_identity_scep_certificates
		SET host_id = ?
		WHERE serial = ?`, hostID, serialNumber)
	return err
}

func (ds *Datastore) GetHostIdentityCertByName(ctx context.Context, name string) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, `
		SELECT serial, host_id, name, not_valid_after, public_key_raw, created_at
		FROM host_identity_scep_certificates
		WHERE name = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("host identity certificate")
	case err != nil:
		return nil, err
	}
	return &hostIdentityCert, nil
}

func (ds *Datastore) GetHostIdentityCertByPublicKey(ctx context.Context, publicKeyDER []byte) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, `
		SELECT serial, host_id, name, not_valid_after, public_key_raw, created_at
		FROM host_identity_scep_certificates
		WHERE public_key_raw = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, publicKeyDER)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("host identity certificate")
	case err != nil:
		return nil, err
	}
	return &hostIdentityCert, nil
}

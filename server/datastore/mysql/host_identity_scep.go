package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetHostIdentityCertBySerialNumber(ctx context.Context, serialNumber uint64) (*types.HostIdentityCertificate, error) {
	var hostIdentityCert types.HostIdentityCertificate
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostIdentityCert, `
		SELECT serial, host_id, name, not_valid_after, public_key_raw
		FROM host_identity_scep_certificates
		WHERE serial = ?
			AND not_valid_after > NOW()
			AND revoked = 0`, serialNumber)
	if err != nil {
		return nil, err
	}
	return &hostIdentityCert, nil
}

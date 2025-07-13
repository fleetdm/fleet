package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
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
	if err != nil {
		return nil, err
	}
	return &hostIdentityCert, nil
}

func (ds *Datastore) UpdateHostIdentityCertHostIDBySerial(ctx context.Context, serialNumber uint64, hostID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_identity_scep_certificates
		SET host_id = ?
		WHERE serial = ?`, hostID, serialNumber)
	return err
}

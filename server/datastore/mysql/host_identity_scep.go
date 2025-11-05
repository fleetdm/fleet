package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

// RevokeOldHostIdentityCerts revokes old certificates for hosts that have a newer certificate.
// It only revokes certificates where the newer certificate is older than the grace period.
// This prevents authentication failures during certificate rotation.
// Returns the number of certificates revoked.
func (ds *Datastore) RevokeOldHostIdentityCerts(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	// Find and revoke old certificates where a newer certificate exists for the same host,
	// and the newer certificate is older than the grace period (to ensure it's stable)
	//
	// Explanation:
	// 1. Find the newest "stable" cert for each host (stable = issued before grace period)
	// 2. Revoke all certs with serial < newest stable serial for that host
	stmt := `
		UPDATE host_identity_scep_certificates old_certs
		INNER JOIN (
			SELECT host_id, MAX(serial) as newest_stable_serial
			FROM host_identity_scep_certificates
			WHERE not_valid_before < DATE_SUB(NOW(6), INTERVAL ? SECOND)
			  AND revoked = 0
			GROUP BY host_id
		) stable_certs ON old_certs.host_id = stable_certs.host_id
		SET old_certs.revoked = 1, old_certs.updated_at = NOW(6)
		WHERE old_certs.serial < stable_certs.newest_stable_serial
		  AND old_certs.revoked = 0
	`

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, int(gracePeriod.Seconds()))
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

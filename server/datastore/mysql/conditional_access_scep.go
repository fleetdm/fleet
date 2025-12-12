package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// GetConditionalAccessCertHostIDBySerialNumber retrieves the host_id for a valid certificate by serial number.
// This is a lightweight method optimized for authentication flows.
func (ds *Datastore) GetConditionalAccessCertHostIDBySerialNumber(ctx context.Context, serial uint64) (uint, error) {
	stmt := `
		SELECT host_id
		FROM conditional_access_scep_certificates
		WHERE serial = ? AND revoked = 0 AND not_valid_after > NOW()
	`
	var hostID uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostID, stmt, serial)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, notFound("ConditionalAccessCertificate"))
		}
		return 0, ctxerr.Wrap(ctx, err, "get conditional access cert host_id by serial")
	}
	return hostID, nil
}

// GetConditionalAccessCertCreatedAtByHostID retrieves the created_at timestamp of the most recent certificate for a host.
// This is a lightweight method for rate limiting checks.
func (ds *Datastore) GetConditionalAccessCertCreatedAtByHostID(ctx context.Context, hostID uint) (*time.Time, error) {
	stmt := `
		SELECT created_at
		FROM conditional_access_scep_certificates
		WHERE host_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	var createdAt time.Time
	err := sqlx.GetContext(ctx, ds.reader(ctx), &createdAt, stmt, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("ConditionalAccessCertificate"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get conditional access cert created_at by host ID")
	}
	return &createdAt, nil
}

// RevokeOldConditionalAccessCerts revokes old certificates for hosts that have a newer certificate.
// It only revokes certificates where the newer certificate is older than the grace period.
// This prevents authentication failures during certificate rotation.
// Returns the number of certificates revoked.
func (ds *Datastore) RevokeOldConditionalAccessCerts(ctx context.Context, gracePeriod time.Duration) (int64, error) {
	// Find and revoke old certificates where a newer certificate exists for the same host,
	// and the newer certificate is older than the grace period (to ensure it's stable)
	//
	// Explanation:
	// 1. Find the newest "stable" cert for each host (stable = issued before grace period)
	// 2. Revoke all certs with serial < newest stable serial for that host
	stmt := `
		UPDATE conditional_access_scep_certificates old_certs
		INNER JOIN (
			SELECT host_id, MAX(serial) as newest_stable_serial
			FROM conditional_access_scep_certificates
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
		return 0, ctxerr.Wrap(ctx, err, "revoke old conditional access certificates")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get rows affected")
	}

	return rowsAffected, nil
}

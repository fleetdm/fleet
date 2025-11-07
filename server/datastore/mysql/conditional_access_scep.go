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

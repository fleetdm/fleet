package pgsql

import (
	"context"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// Executes SQL statements that return a single COUNT(*) of rows.
func (s *PgSQLStorage) queryRowContextRowExists(ctx context.Context, query string, args ...interface{}) (bool, error) {
	var ct int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&ct)
	return ct > 0, err
}

func (s *PgSQLStorage) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE id = $1;`,
		r.ID,
	)
}

func (s *PgSQLStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE sha256 = $1;`,
		strings.ToLower(hash),
	)
}

func (s *PgSQLStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE id = $1 AND sha256 = $2;`,
		r.ID, strings.ToLower(hash),
	)
}

// AssociateCertHash "DO NOTHING" on duplicated keys
func (s *PgSQLStorage) AssociateCertHash(r *mdm.Request, hash string, _ time.Time) error {
	_, err := s.db.ExecContext(
		r.Context, `
INSERT INTO cert_auth_associations (id, sha256)
VALUES ($1, $2)
ON CONFLICT ON CONSTRAINT cert_auth_associations_pkey DO UPDATE SET updated_at=now();`,
		r.ID,
		strings.ToLower(hash),
	)
	return err
}

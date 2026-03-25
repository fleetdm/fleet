package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/google/uuid"
)

// UpsertACMEEnrollment upserts the acme_enrollments table with the given
// host_identifier. It generates a new path_identifier for the row and returns
// it. If a row already exists for the host, its path_identifier is refreshed;
// otherwise a new row is inserted.
func (ds *Datastore) UpsertACMEEnrollment(ctx context.Context, hostIdentifier string) (string, error) {
	ctx, span := tracer.Start(ctx, "acme.mysql.UpsertACMEEnrollment")
	defer span.End()

	pathIdentifier := uuid.NewString()

	result, err := ds.writer(ctx).ExecContext(ctx, `
UPDATE acme_enrollments
SET path_identifier = ?
WHERE host_identifier = ?
`, pathIdentifier, hostIdentifier)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "updating ACME enrollment path identifier")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "checking rows affected for ACME enrollment upsert")
	}

	if rows == 0 {
		_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO acme_enrollments (path_identifier, host_identifier)
VALUES (?, ?)
`, pathIdentifier, hostIdentifier)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "inserting ACME enrollment")
		}
	}

	return pathIdentifier, nil
}

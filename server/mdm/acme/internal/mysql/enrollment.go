package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/google/uuid"
)

// NewEnrollment creates a new row in the acme_enrollments table with the given
// host_identifier. It generates a new path_identifier for the row and returns
// it.
func (ds *Datastore) NewEnrollment(ctx context.Context, hostIdentifier string) (string, error) {
	ctx, span := tracer.Start(ctx, "acme.mysql.NewEnrollment")
	defer span.End()

	pathIdentifier := uuid.NewString()

	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO acme_enrollments (path_identifier, host_identifier)
VALUES (?, ?)
`, pathIdentifier, hostIdentifier)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "inserting ACME enrollment")
	}

	return pathIdentifier, nil
}

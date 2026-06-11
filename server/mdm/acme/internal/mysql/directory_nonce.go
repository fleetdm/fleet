package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetACMEEnrollment(ctx context.Context, pathIdentifier string) (*types.Enrollment, error) {
	ctx, span := tracer.Start(ctx, "acme.mysql.GetACMEEnrollment")
	defer span.End()

	const stmt = `
SELECT
	id,
	path_identifier,
	host_identifier,
	not_valid_after,
	revoked
FROM
	acme_enrollments
WHERE
	path_identifier = ?
LIMIT 1
`
	var enrollment types.Enrollment
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enrollment, stmt, pathIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = types.EnrollmentNotFoundError(fmt.Sprintf("ACME enrollment with path identifier %s not found", pathIdentifier))
			return nil, ctxerr.Wrap(ctx, err)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting ACME enrollment")
	}
	return &enrollment, nil
}

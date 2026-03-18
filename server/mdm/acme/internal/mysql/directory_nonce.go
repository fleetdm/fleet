package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetACMEEnrollment(ctx context.Context, pathIdentifier string) (*types.ACMEEnrollment, error) {
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
	var enrollment types.ACMEEnrollment
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enrollment, stmt, pathIdentifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("ACME enrollment").WithName(pathIdentifier))
		}
		return nil, err
	}
	return &enrollment, nil
}

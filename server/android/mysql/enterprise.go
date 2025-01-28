package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateEnterprise(ctx context.Context) (uint, error) {
	stmt := `INSERT INTO android_enterprises (signup_name) VALUES ('')`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		// Note: ctxerr is dependent on the fleet package -- the fleet package should be factored out or android should have its own version of ctxerr
		return 0, ctxerr.Wrap(ctx, err, "inserting enterprise")
	}
	id, _ := res.LastInsertId()
	return uint(id), nil // nolint:gosec // dismiss G115
}

func (ds *Datastore) GetEnterpriseByID(ctx context.Context, id uint) (*android.Enterprise, error) {
	stmt := `SELECT id, signup_name, enterprise_id FROM android_enterprises WHERE id = ?`
	var enterprise android.Enterprise
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enterprise, stmt, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("Android enterprise").WithID(id)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "selecting enterprise")
	}
	return &enterprise, nil
}

func (ds *Datastore) UpdateEnterprise(ctx context.Context, enterprise *android.Enterprise) error {
	if enterprise == nil || enterprise.ID == 0 {
		return errors.New("missing enterprise ID")
	}
	stmt := `UPDATE android_enterprises
    SET signup_name = ?,
        enterprise_id = ?
	WHERE id = ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, enterprise.SignupName, enterprise.EnterpriseID, enterprise.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting enterprise")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return notFound("Android enterprise").WithID(enterprise.ID)
	}
	return nil
}

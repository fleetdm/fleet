package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateEnterprise(ctx context.Context, userID uint) (uint, error) {
	// android_enterprises user_id is only set when the row is created
	stmt := `INSERT INTO android_enterprises (signup_name, user_id) VALUES ('', ?)`
	res, err := ds.Writer(ctx).ExecContext(ctx, stmt, userID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "inserting enterprise")
	}
	id, _ := res.LastInsertId()
	return uint(id), nil // nolint:gosec // dismiss G115
}

func (ds *Datastore) GetEnterpriseByID(ctx context.Context, id uint) (*android.EnterpriseDetails, error) {
	stmt := `SELECT id, signup_name, enterprise_id, pubsub_topic_id, signup_token, user_id FROM android_enterprises WHERE id = ?`
	var enterprise android.EnterpriseDetails
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enterprise, stmt, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android enterprise").WithID(id)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise by id")
	}
	return &enterprise, nil
}

func (ds *Datastore) GetEnterpriseBySignupToken(ctx context.Context, signupToken string) (*android.EnterpriseDetails, error) {
	stmt := `SELECT id, signup_name, enterprise_id, pubsub_topic_id, signup_token, user_id FROM android_enterprises WHERE signup_token = ?`
	var enterprise android.EnterpriseDetails
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enterprise, stmt, signupToken)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android enterprise")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise by signup token")
	}
	return &enterprise, nil
}

func (ds *Datastore) GetEnterprise(ctx context.Context) (*android.Enterprise, error) {
	stmt := `SELECT id, enterprise_id FROM android_enterprises WHERE enterprise_id != '' LIMIT 1`
	var enterprise android.Enterprise
	err := sqlx.GetContext(ctx, ds.reader(ctx), &enterprise, stmt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android enterprise")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting active enterprise")
	}
	return &enterprise, nil
}

func (ds *Datastore) UpdateEnterprise(ctx context.Context, enterprise *android.EnterpriseDetails) error {
	if enterprise == nil || enterprise.ID == 0 {
		return errors.New("missing enterprise ID")
	}
	stmt := `UPDATE android_enterprises
    SET signup_name = ?,
        enterprise_id = ?,
        pubsub_topic_id = ?,
        signup_token = ?
	WHERE id = ?`
	res, err := ds.Writer(ctx).ExecContext(ctx, stmt, enterprise.SignupName, enterprise.EnterpriseID, enterprise.TopicID,
		enterprise.SignupToken, enterprise.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting enterprise")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return common_mysql.NotFound("Android enterprise").WithID(enterprise.ID)
	}
	return nil
}

func (ds *Datastore) DeleteOtherEnterprises(ctx context.Context, id uint) error {
	stmt := `DELETE FROM android_enterprises WHERE id != ?`
	_, err := ds.Writer(ctx).ExecContext(ctx, stmt, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting other enterprises")
	}
	return nil
}

func (ds *Datastore) DeleteAllEnterprises(ctx context.Context) error {
	stmt := `DELETE FROM android_enterprises`
	_, err := ds.Writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting all enterprises")
	}
	return nil
}

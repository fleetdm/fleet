package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewMDMAppleEnrollment(
	ctx context.Context, enrollment fleet.MDMAppleEnrollmentPayload,
) (*fleet.MDMAppleEnrollment, error) {
	res, err := ds.appleMDMWriter.ExecContext(ctx,
		`INSERT INTO mdm_apple_enrollments (name, config, dep_config) VALUES (?, ?, ?)`,
		enrollment.Name, enrollment.Config, enrollment.DEPConfig,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleEnrollment{
		ID:        uint(id),
		Name:      enrollment.Name,
		Config:    enrollment.Config,
		DEPConfig: enrollment.DEPConfig,
	}, nil
}

func (ds *Datastore) MDMAppleEnrollment(ctx context.Context, enrollmentID uint) (*fleet.MDMAppleEnrollment, error) {
	var enrollment fleet.MDMAppleEnrollment
	if err := sqlx.GetContext(ctx, ds.appleMDMWriter,
		&enrollment,
		`SELECT id, name, config, dep_config FROM mdm_apple_enrollments WHERE id = ?`,
		enrollmentID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleEnrollment").WithID(enrollmentID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment by id")
	}
	return &enrollment, nil
}

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
		`INSERT INTO mdm_apple_enrollments (name, dep_config) VALUES (?, ?)`,
		enrollment.Name, enrollment.DEPConfig,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleEnrollment{
		ID:        uint(id),
		Name:      enrollment.Name,
		DEPConfig: enrollment.DEPConfig,
	}, nil
}

func (ds *Datastore) MDMAppleEnrollment(ctx context.Context, enrollmentID uint) (*fleet.MDMAppleEnrollment, error) {
	var enrollment fleet.MDMAppleEnrollment
	if err := sqlx.GetContext(ctx, ds.appleMDMWriter,
		&enrollment,
		`SELECT id, name, dep_config FROM mdm_apple_enrollments WHERE id = ?`,
		enrollmentID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleEnrollment").WithID(enrollmentID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment by id")
	}
	return &enrollment, nil
}

func (ds *Datastore) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
	query := `
SELECT
    id,
    command_uuid,
    status,
    result
FROM
    command_results
WHERE
    command_uuid = ?
`

	var results []*fleet.MDMAppleCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.appleMDMWriter,
		&results,
		query,
		commandUUID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get command results")
	}

	resultsMap := make(map[string]*fleet.MDMAppleCommandResult, len(results))
	for _, result := range results {
		resultsMap[result.ID] = result
	}

	return resultsMap, nil
}

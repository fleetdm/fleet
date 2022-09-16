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
	res, err := ds.writer.ExecContext(ctx,
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

func (ds *Datastore) ListMDMAppleEnrollments(ctx context.Context) ([]fleet.MDMAppleEnrollment, error) {
	var enrollments []fleet.MDMAppleEnrollment
	if err := sqlx.SelectContext(
		ctx,
		ds.writer,
		&enrollments,
		`SELECT id, name, dep_config FROM mdm_apple_enrollments`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list devices")
	}
	return enrollments, nil
}

func (ds *Datastore) MDMAppleEnrollment(ctx context.Context, enrollmentID uint) (*fleet.MDMAppleEnrollment, error) {
	var enrollment fleet.MDMAppleEnrollment
	if err := sqlx.GetContext(ctx, ds.writer,
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
    nano_command_results
WHERE
    command_uuid = ?
`

	var results []*fleet.MDMAppleCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.writer,
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

func (ds *Datastore) NewMDMAppleInstaller(ctx context.Context, name string, size int64, manifest string, installer []byte, urlToken string) (*fleet.MDMAppleInstaller, error) {
	res, err := ds.writer.ExecContext(
		ctx,
		`INSERT INTO mdm_apple_installers (name, size, manifest, installer, url_token) VALUES (?, ?, ?, ?, ?)`,
		name, size, manifest, installer, urlToken,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleInstaller{
		ID:        uint(id),
		Size:      size,
		Name:      name,
		Manifest:  manifest,
		Installer: installer,
		URLToken:  urlToken,
	}, nil
}

func (ds *Datastore) MDMAppleInstaller(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer,
		&installer,
		`SELECT id, name, size, manifest, installer, url_token FROM mdm_apple_installers WHERE url_token = ?`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithName(token))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer by token")
	}
	return &installer, nil
}

func (ds *Datastore) MDMAppleInstallerDetailsByID(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer,
		&installer,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers WHERE id = ?`,
		id,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer details by id")
	}
	return &installer, nil
}

func (ds *Datastore) MDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer,
		&installer,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers WHERE url_token = ?`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithName(token))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer details by id")
	}
	return &installer, nil
}

func (ds *Datastore) ListMDMAppleInstallers(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
	var installers []fleet.MDMAppleInstaller
	if err := sqlx.SelectContext(ctx, ds.writer,
		&installers,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list installers")
	}
	return installers, nil
}

func (ds *Datastore) MDMAppleListDevices(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
	var devices []fleet.MDMAppleDevice
	if err := sqlx.SelectContext(
		ctx,
		ds.writer,
		&devices,
		`
SELECT
    d.id,
    d.serial_number,
    e.enabled
FROM
    nano_devices d
    JOIN nano_enrollments e ON d.id = e.device_id
WHERE
    type = "Device"
`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list devices")
	}
	return devices, nil
}

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
)

func (ds *Datastore) NewMDMAppleEnrollmentProfile(
	ctx context.Context,
	payload fleet.MDMAppleEnrollmentProfilePayload,
) (*fleet.MDMAppleEnrollmentProfile, error) {
	res, err := ds.writer.ExecContext(ctx,
		`
INSERT INTO
    mdm_apple_enrollment_profiles (token, type, dep_profile)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    token = VALUES(token),
    type = VALUES(type),
    dep_profile = VALUES(dep_profile)
`,
		payload.Token, payload.Type, payload.DEPProfile,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleEnrollmentProfile{
		ID:         uint(id),
		Token:      payload.Token,
		Type:       payload.Type,
		DEPProfile: payload.DEPProfile,
	}, nil
}

func (ds *Datastore) ListMDMAppleEnrollmentProfiles(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
	var enrollmentProfiles []*fleet.MDMAppleEnrollmentProfile
	if err := sqlx.SelectContext(
		ctx,
		ds.writer,
		&enrollmentProfiles,
		`
SELECT
    id,
    token,
    type,
    dep_profile,
    created_at,
    updated_at
FROM
    mdm_apple_enrollment_profiles
`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list enrollment profiles")
	}
	return enrollmentProfiles, nil
}

func (ds *Datastore) GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
	var enrollment fleet.MDMAppleEnrollmentProfile
	if err := sqlx.GetContext(ctx, ds.writer,
		&enrollment,
		`
SELECT
    id,
    token,
    type,
    dep_profile,
    created_at,
    updated_at
FROM
    mdm_apple_enrollment_profiles
WHERE
    token = ?
`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleEnrollmentProfile"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment profile by token")
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

func (ds *Datastore) DeleteMDMAppleInstaller(ctx context.Context, id uint) error {
	if _, err := ds.writer.ExecContext(ctx, `DELETE FROM mdm_apple_installers WHERE id = ?`, id); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
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

func (ds *Datastore) IngestMDMAppleDeviceFromCheckin(ctx context.Context, mdmHost fleet.MDMAppleHostDetails) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ingestMDMAppleDeviceFromCheckinDB(ctx, tx, mdmHost)
	})
}

func ingestMDMAppleDeviceFromCheckinDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	mdmHost fleet.MDMAppleHostDetails,
) error {
	if mdmHost.SerialNumber == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected device serial number but got empty string")
	}
	if mdmHost.UDID == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected unique device id but got empty string")
	}

	stmt := `SELECT id, uuid, hardware_serial FROM hosts WHERE uuid = ? OR hardware_serial = ?`

	var foundHost fleet.Host
	err := sqlx.GetContext(ctx, tx, &foundHost, stmt, mdmHost.UDID, mdmHost.SerialNumber)
	switch {
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return ctxerr.Wrap(ctx, err, "get mdm apple host by serial number or udid")

	case errors.Is(err, sql.ErrNoRows):
		return insertMDMAppleHostDB(ctx, tx, mdmHost)

	case foundHost.HardwareSerial != mdmHost.SerialNumber || foundHost.UUID != mdmHost.UDID:
		return updateMDMAppleHostDB(ctx, tx, foundHost.ID, mdmHost)

	default:
		// ok, nothing to do here
		return nil
	}
}

func updateMDMAppleHostDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, mdmHost fleet.MDMAppleHostDetails) error {
	updateStmt := `
		UPDATE hosts SET
			hardware_serial = ?,
			uuid = ?,
			hardware_model = ?,
			platform =  ?,
			refetch_requested = ?,
			osquery_host_id = ?
		WHERE id = ?`

	if _, err := tx.ExecContext(
		ctx,
		updateStmt,
		mdmHost.SerialNumber,
		mdmHost.UDID,
		mdmHost.Model,
		"darwin",
		1,
		mdmHost.UDID,
		hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update mdm apple host")
	}

	return nil
}

func insertMDMAppleHostDB(ctx context.Context, tx sqlx.ExtContext, mdmHost fleet.MDMAppleHostDetails) error {
	insertStmt := `
		INSERT INTO hosts (
			hardware_serial,
			uuid, 
			hardware_model, 
			platform, 
			last_enrolled_at, 
			detail_updated_at, 
			osquery_host_id,
			refetch_requested
		) VALUES (?,?,?,?,?,?,?,?)`

	res, err := tx.ExecContext(
		ctx,
		insertStmt,
		mdmHost.SerialNumber,
		mdmHost.UDID,
		mdmHost.Model,
		"darwin",
		"2000-01-01 00:00:00",
		"2000-01-01 00:00:00",
		mdmHost.UDID,
		1,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "insert mdm apple host")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "last insert id mdm apple host")
	}
	if id < 1 {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host unexpected last insert id")
	}

	if err := upsertMDMAppleHostDisplayNamesDB(ctx, tx, uint(id)); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert related tables")
	}

	if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, uint(id)); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert related tables")
	}

	return nil
}

func (ds *Datastore) IngestMDMAppleDevicesFromDEPSync(ctx context.Context, devices []godep.Device) (int64, error) {
	if len(devices) < 1 {
		return 0, nil
	}
	filteredDevices := filterMDMAppleDevices(devices)
	if len(filteredDevices) < 1 {
		return 0, nil
	}

	var resCount int64
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		us, args := unionSelectDevices(filteredDevices)

		stmt := fmt.Sprintf(`
		INSERT INTO hosts (hardware_serial, hardware_model, platform, last_enrolled_at, detail_updated_at, osquery_host_id, refetch_requested) (
			SELECT
				us.hardware_serial,
				COALESCE(GROUP_CONCAT(DISTINCT us.hardware_model), ''),
				'darwin' AS platform,
				'2000-01-01 00:00:00' AS last_enrolled_at,
				'2000-01-01 00:00:00' AS detail_updated_at,
				NULL AS osquery_host_id,
				1 AS refetch_requested
			FROM (%s) us
			LEFT JOIN hosts h ON us.hardware_serial = h.hardware_serial
		WHERE
			h.id IS NULL
		GROUP BY
			us.hardware_serial)`,
			us,
		)

		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple hosts from dep sync insert")
		}

		n, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple hosts from dep sync rows affected")
		}
		resCount = n

		// get new host ids
		args = []interface{}{}
		parts := []string{}
		for _, d := range filteredDevices {
			args = append(args, d.SerialNumber)
			parts = append(parts, "?")
		}
		var hostIDs []uint
		err = sqlx.SelectContext(ctx, tx, &hostIDs, fmt.Sprintf(`
			SELECT id FROM hosts WHERE hardware_serial IN(%s)`,
			strings.Join(parts, ",")),
			args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host get host ids")
		}

		if err := upsertMDMAppleHostDisplayNamesDB(ctx, tx, hostIDs...); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert related tables")
		}

		if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, hostIDs...); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert related tables")
		}

		return nil
	})

	return resCount, err
}

func upsertMDMAppleHostDisplayNamesDB(ctx context.Context, tx sqlx.ExtContext, hostIDs ...uint) error {
	args := []interface{}{}
	parts := []string{}
	for _, id := range hostIDs {
		args = append(args, id)
		parts = append(parts, "(?, '')")
	}

	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO host_display_names (host_id, display_name) VALUES %s
			ON DUPLICATE KEY UPDATE display_name = VALUES(display_name)`, strings.Join(parts, ",")),
		args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host display names")
	}

	return nil
}

func upsertMDMAppleHostLabelMembershipDB(ctx context.Context, tx sqlx.ExtContext, hostIDs ...uint) error {
	// Builtin label memberships are usually inserted when the first distributed query results are
	// received; however, we want to insert pending MDM hosts now because it may still be some time
	// before osquery is running on these devices. Because these are Apple devices, we're adding
	// them to the "All Hosts" and "macOS" labels.
	labelIDs := []uint{}
	err := sqlx.SelectContext(ctx, tx, &labelIDs, `SELECT id FROM labels WHERE label_type = 1 AND (name = 'All Hosts' OR name = 'macOS')`)
	switch {
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get builtin labels")
	case len(labelIDs) != 2:
		// Builtin labels can get deleted so it is important that we check that they still exist
		// before we continue.
		return ctxerr.New(ctx, fmt.Sprintf("expected 2 builtin labels but got %d", len(labelIDs)))
	default:
		// continue
	}

	parts := []string{}
	args := []interface{}{}
	for _, id := range hostIDs {
		parts = append(parts, "(?,?),(?,?)")
		args = append(args, id, labelIDs[0], id, labelIDs[1])
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO label_membership (host_id, label_id) VALUES %s 
			ON DUPLICATE KEY UPDATE host_id = host_id`, strings.Join(parts, ",")), args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert label membership")
	}

	return nil
}

func filterMDMAppleDevices(devices []godep.Device) []godep.Device {
	var filtered []godep.Device
	for _, device := range devices {
		// We currently only support macOS devices so we screen out iOS and tvOS.
		if strings.ToLower(device.OS) != "osx" {
			continue
		}
		// We currently only listen for an op_type of "added", the other
		// op_types are ambiguous and it would be needless to ingest the device
		// every single time we get an update.
		if strings.ToLower(device.OpType) == "added" ||
			// The op_type field is only applicable with the SyncDevices API call,
			// Empty op_type come from the first call to FetchDevices without a cursor,
			// and we do want to assign profiles to them.
			strings.ToLower(device.OpType) == "" {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

func unionSelectDevices(devices []godep.Device) (stmt string, args []interface{}) {
	for i, d := range devices {
		if i == 0 {
			stmt = "SELECT ? hardware_serial, ? hardware_model"
		} else {
			stmt += " UNION SELECT ?, ?"
		}
		args = append(args, d.SerialNumber, d.Model)
	}

	return stmt, args
}

package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
	"github.com/micromdm/nanodep/godep"
)

func (ds *Datastore) NewMDMAppleConfigProfile(ctx context.Context, cp fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
	stmt := `
INSERT INTO
    mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
VALUES (?, ?, ?, ?)`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	res, err := ds.writer.ExecContext(ctx, stmt, teamID, cp.Identifier, cp.Name, cp.Mobileconfig)
	if err != nil {
		switch {
		case isDuplicate(err):
			return nil, ctxerr.Wrap(ctx, formatErrorDuplicateConfigProfile(err, &cp))
		default:
			return nil, ctxerr.Wrap(ctx, err, "creating new mdm config profile")
		}
	}

	id, _ := res.LastInsertId()

	return &fleet.MDMAppleConfigProfile{
		ProfileID:    uint(id),
		Identifier:   cp.Identifier,
		Name:         cp.Name,
		Mobileconfig: cp.Mobileconfig,
		TeamID:       cp.TeamID,
	}, nil
}

func formatErrorDuplicateConfigProfile(err error, cp *fleet.MDMAppleConfigProfile) error {
	switch {
	case strings.Contains(err.Error(), "idx_mdm_apple_config_prof_team_identifier"):
		return &existsError{
			ResourceType: "MDMAppleConfigProfile.PayloadIdentifier",
			Identifier:   cp.Identifier,
			TeamID:       cp.TeamID,
		}
	case strings.Contains(err.Error(), "idx_mdm_apple_config_prof_team_name"):
		return &existsError{
			ResourceType: "MDMAppleConfigProfile.PayloadDisplayName",
			Identifier:   cp.Name,
			TeamID:       cp.TeamID,
		}
	default:
		return err
	}
}

func (ds *Datastore) ListMDMAppleConfigProfiles(ctx context.Context, teamID *uint) ([]*fleet.MDMAppleConfigProfile, error) {
	stmt := `
SELECT
	profile_id,
	team_id,
	name,
	identifier,
	mobileconfig,
	created_at,
	updated_at
FROM
	mdm_apple_configuration_profiles
WHERE
	team_id=?`

	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	var res []*fleet.MDMAppleConfigProfile
	err := sqlx.SelectContext(ctx, ds.reader, &res, stmt, teamID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ds *Datastore) GetMDMAppleConfigProfile(ctx context.Context, profileID uint) (*fleet.MDMAppleConfigProfile, error) {
	stmt := `
SELECT
	profile_id,
	team_id,
	name,
	identifier,
	mobileconfig,
	created_at,
	updated_at
FROM
	mdm_apple_configuration_profiles
WHERE
	profile_id=?`

	var res fleet.MDMAppleConfigProfile
	err := sqlx.GetContext(ctx, ds.reader, &res, stmt, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithID(profileID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get mdm apple config profile")
	}

	return &res, nil
}

func (ds *Datastore) DeleteMDMAppleConfigProfile(ctx context.Context, profileID uint) error {
	res, err := ds.writer.ExecContext(ctx, `DELETE FROM mdm_apple_configuration_profiles WHERE profile_id=?`, profileID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching delete mdm config profile query rows affected")
	}
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithID(profileID))
	}

	return nil
}

func (ds *Datastore) DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx context.Context, teamID *uint, profileIdentifier string) error {
	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	res, err := ds.writer.ExecContext(ctx, `DELETE FROM mdm_apple_configuration_profiles WHERE team_id = ? AND identifier = ?`, teamID, profileIdentifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if deleted, _ := res.RowsAffected(); deleted == 0 {
		message := fmt.Sprintf("identifier: %s, team_id: %d", profileIdentifier, teamID)
		return ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithMessage(message))
	}

	return nil
}

func (ds *Datastore) GetHostMDMProfiles(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
	stmt := fmt.Sprintf(`
SELECT
	profile_id,
	profile_name AS name,
	profile_identifier AS identifier,
	status,
	COALESCE(operation_type, '') AS operation_type,
	COALESCE(detail, '') AS detail
FROM
	host_mdm_apple_profiles
WHERE
	host_uuid = ? AND NOT (operation_type = '%s' AND status = '%s')`,
		fleet.MDMAppleOperationTypeRemove,
		fleet.MDMAppleDeliveryApplied,
	)

	var profiles []fleet.HostMDMAppleProfile
	if err := sqlx.SelectContext(ctx, ds.reader, &profiles, stmt, hostUUID); err != nil {
		return nil, err
	}
	return profiles, nil
}

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

func (ds *Datastore) GetMDMAppleCommandRequestType(ctx context.Context, commandUUID string) (string, error) {
	var rt string
	err := sqlx.GetContext(ctx, ds.reader, &rt, `SELECT request_type FROM nano_commands WHERE command_uuid = ?`, commandUUID)
	return rt, err
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
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host get app config")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ingestMDMAppleDeviceFromCheckinDB(ctx, tx, mdmHost, ds.logger, appCfg)
	})
}

func ingestMDMAppleDeviceFromCheckinDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	mdmHost fleet.MDMAppleHostDetails,
	logger log.Logger,
	appCfg *fleet.AppConfig,
) error {
	if mdmHost.SerialNumber == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected device serial number but got empty string")
	}
	if mdmHost.UDID == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected unique device id but got empty string")
	}

	// MDM is necessarily enabled if this gets called, always pass true for that
	// parameter.
	matchID, _, err := matchHostDuringEnrollment(ctx, tx, true, "", mdmHost.UDID, mdmHost.SerialNumber)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return insertMDMAppleHostDB(ctx, tx, mdmHost, logger, appCfg)

	case err != nil:
		return ctxerr.Wrap(ctx, err, "get mdm apple host by serial number or udid")

	default:
		return updateMDMAppleHostDB(ctx, tx, matchID, mdmHost, appCfg)
	}
}

func updateMDMAppleHostDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	mdmHost fleet.MDMAppleHostDetails,
	appCfg *fleet.AppConfig,
) error {
	updateStmt := `
		UPDATE hosts SET
			hardware_serial = ?,
			uuid = ?,
			hardware_model = ?,
			platform =  ?,
			refetch_requested = ?,
			osquery_host_id = COALESCE(NULLIF(osquery_host_id, ''), ?)
		WHERE id = ?`

	if _, err := tx.ExecContext(
		ctx,
		updateStmt,
		mdmHost.SerialNumber,
		mdmHost.UDID,
		mdmHost.Model,
		"darwin",
		1,
		// Set osquery_host_id to the device UUID only if it is not already set.
		mdmHost.UDID,
		hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update mdm apple host")
	}

	if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, appCfg.ServerSettings, false, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
	}

	return nil
}

func insertMDMAppleHostDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	mdmHost fleet.MDMAppleHostDetails,
	logger log.Logger,
	appCfg *fleet.AppConfig,
) error {
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
	host := fleet.Host{ID: uint(id), HardwareModel: mdmHost.Model, HardwareSerial: mdmHost.SerialNumber}

	if err := upsertMDMAppleHostDisplayNamesDB(ctx, tx, host); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert display names")
	}

	if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, logger, host); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert label membership")
	}

	if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, appCfg.ServerSettings, false, host.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
	}
	return nil
}

func (ds *Datastore) IngestMDMAppleDevicesFromDEPSync(ctx context.Context, devices []godep.Device) (int64, error) {
	if len(devices) < 1 {
		level.Debug(ds.logger).Log("msg", "ingesting devices from DEP received < 1 device, skipping", "len(devices)", len(devices))
		return 0, nil
	}
	filteredDevices := filterMDMAppleDevices(devices, ds.logger)
	if len(filteredDevices) < 1 {
		level.Debug(ds.logger).Log("msg", "ingesting devices from DEP filtered all devices, skipping", "len(devices)", len(devices))
		return 0, nil
	}

	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "ingest mdm apple host get app config")
	}

	args := []interface{}{nil}
	if name := appCfg.MDM.AppleBMDefaultTeam; name != "" {
		team, err := ds.TeamByName(ctx, name)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			level.Debug(ds.logger).Log(
				"msg",
				"ingesting devices from DEP: unable to find default team assigned in config, the devices won't be assigned to a team",
				"team_name",
				name,
			)
			// If the team doesn't exist, we still ingest the device, but it won't
			// belong to any team.
		case err != nil:
			return 0, ctxerr.Wrap(ctx, err, "ingest mdm apple host get team by name")
		default:
			args[0] = team.ID
		}
	}

	var resCount int64
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		us, unionArgs := unionSelectDevices(filteredDevices)
		args = append(args, unionArgs...)

		stmt := fmt.Sprintf(`
		INSERT INTO hosts (
			hardware_serial,
			hardware_model,
			platform,
			last_enrolled_at,
			detail_updated_at,
			osquery_host_id,
			refetch_requested,
			team_id
		) (
			SELECT
				us.hardware_serial,
				COALESCE(GROUP_CONCAT(DISTINCT us.hardware_model), ''),
				'darwin' AS platform,
				'2000-01-01 00:00:00' AS last_enrolled_at,
				'2000-01-01 00:00:00' AS detail_updated_at,
				NULL AS osquery_host_id,
				1 AS refetch_requested,
				? AS team_id
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
		var hosts []fleet.Host
		err = sqlx.SelectContext(ctx, tx, &hosts, fmt.Sprintf(`
			SELECT id, hardware_model, hardware_serial FROM hosts WHERE hardware_serial IN(%s)`,
			strings.Join(parts, ",")),
			args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host get host ids")
		}

		if err := upsertMDMAppleHostDisplayNamesDB(ctx, tx, hosts...); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert display names")
		}

		if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, ds.logger, hosts...); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert label membership")
		}

		var ids []uint
		for _, h := range hosts {
			ids = append(ids, h.ID)
		}
		if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, appCfg.ServerSettings, true, ids...); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
		}

		return nil
	})

	return resCount, err
}

func upsertMDMAppleHostDisplayNamesDB(ctx context.Context, tx sqlx.ExtContext, hosts ...fleet.Host) error {
	args := []interface{}{}
	parts := []string{}
	for _, h := range hosts {
		args = append(args, h.ID, h.DisplayName())
		parts = append(parts, "(?, ?)")
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

func upsertMDMAppleHostMDMInfoDB(ctx context.Context, tx sqlx.ExtContext, serverSettings fleet.ServerSettings, fromSync bool, hostIDs ...uint) error {
	serverURL, err := apple_mdm.ResolveAppleMDMURL(serverSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolve Fleet MDM URL")
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO mobile_device_management_solutions (name, server_url) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE server_url = VALUES(server_url)`,
		fleet.WellKnownMDMFleet, serverURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert mdm solution")
	}

	var mdmID int64
	if insertOnDuplicateDidInsert(result) {
		mdmID, _ = result.LastInsertId()
	} else {
		stmt := `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`
		if err := sqlx.GetContext(ctx, tx, &mdmID, stmt, fleet.WellKnownMDMFleet, serverURL); err != nil {
			return ctxerr.Wrap(ctx, err, "query mdm solution id")
		}
	}

	// if the device is coming from the DEP sync, we don't consider it
	// enrolled yet.
	enrolled := !fromSync

	args := []interface{}{}
	parts := []string{}
	for _, id := range hostIDs {
		args = append(args, enrolled, serverURL, fromSync, mdmID, false, id)
		parts = append(parts, "(?, ?, ?, ?, ?, ?)")
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, is_server, host_id) VALUES %s
		ON DUPLICATE KEY UPDATE enrolled = VALUES(enrolled)`, strings.Join(parts, ",")), args...)

	return ctxerr.Wrap(ctx, err, "upsert host mdm info")
}

func upsertMDMAppleHostLabelMembershipDB(ctx context.Context, tx sqlx.ExtContext, logger log.Logger, hosts ...fleet.Host) error {
	// Builtin label memberships are usually inserted when the first distributed
	// query results are received; however, we want to insert pending MDM hosts
	// now because it may still be some time before osquery is running on these
	// devices. Because these are Apple devices, we're adding them to the "All
	// Hosts" and "macOS" labels.
	labelIDs := []uint{}
	err := sqlx.SelectContext(ctx, tx, &labelIDs, `SELECT id FROM labels WHERE label_type = 1 AND (name = 'All Hosts' OR name = 'macOS')`)
	switch {
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get builtin labels")
	case len(labelIDs) != 2:
		// Builtin labels can get deleted so it is important that we check that
		// they still exist before we continue.
		level.Error(logger).Log("err", fmt.Sprintf("expected 2 builtin labels but got %d", len(labelIDs)))
		return nil
	default:
		// continue
	}

	parts := []string{}
	args := []interface{}{}
	for _, h := range hosts {
		parts = append(parts, "(?,?),(?,?)")
		args = append(args, h.ID, labelIDs[0], h.ID, labelIDs[1])
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO label_membership (host_id, label_id) VALUES %s
			ON DUPLICATE KEY UPDATE host_id = host_id`, strings.Join(parts, ",")), args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert label membership")
	}

	return nil
}

func (ds *Datastore) UpdateHostTablesOnMDMUnenroll(ctx context.Context, uuid string) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `
			DELETE FROM host_mdm
			WHERE host_id = (SELECT id FROM hosts WHERE uuid = ?)`, uuid)
		return err
	})
}

func filterMDMAppleDevices(devices []godep.Device, logger log.Logger) []godep.Device {
	var filtered []godep.Device
	for _, device := range devices {
		// We currently only listen for an op_type of "added", the
		// other op_types are ambiguous and it would be needless to
		// ingest the device every single time we get an update.
		if strings.ToLower(device.OpType) == "added" ||
			// The op_type field is only applicable with the SyncDevices
			// API call, Empty op_type come from the first call to
			// FetchDevices without a cursor.
			strings.ToLower(device.OpType) == "" {
			level.Debug(logger).Log("msg", "filterMDMAppleDevices: adding device", "serial", device.SerialNumber, "op_type", device.OpType, "os", device.OS)
			filtered = append(filtered, device)
			continue
		}
		level.Debug(logger).Log("msg", "filterMDMAppleDevices: skipping device", "serial", device.SerialNumber, "op_type", device.OpType, "os", device.OS)
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

func (ds *Datastore) GetNanoMDMEnrollmentStatus(ctx context.Context, id string) (bool, error) {
	var enabled bool
	err := sqlx.GetContext(ctx, ds.reader, &enabled, `SELECT enabled FROM nano_enrollments WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrapf(ctx, err, "getting data from nano_enrollments for id %s", id)
	}

	return enabled, nil
}

func (ds *Datastore) BatchSetMDMAppleProfiles(ctx context.Context, tmID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
	const loadExistingProfiles = `
SELECT
  identifier,
  mobileconfig
FROM
  mdm_apple_configuration_profiles
WHERE
  team_id = ? AND
  identifier IN (?)
`

	const deleteProfilesNotInList = `
DELETE FROM
  mdm_apple_configuration_profiles
WHERE
  team_id = ? AND
  identifier NOT IN (?)
`

	const insertNewOrEditedProfile = `
INSERT INTO
  mdm_apple_configuration_profiles (
    team_id, identifier, name, mobileconfig
  )
VALUES
  ( ?, ?, ?, ? )
`

	// use a profile team id of 0 if no-team
	var profTeamID uint
	if tmID != nil {
		profTeamID = *tmID
	}

	// build a list of identifiers for the incoming profiles, will keep the
	// existing ones if there's a match and no change
	incomingIdents := make([]string, len(profiles))
	// at the same time, index the incoming profiles keyed by identifier for ease
	// or processing
	incomingProfs := make(map[string]*fleet.MDMAppleConfigProfile, len(profiles))
	for i, p := range profiles {
		incomingIdents[i] = p.Identifier
		incomingProfs[p.Identifier] = p
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var existingProfiles []*fleet.MDMAppleConfigProfile

		if len(incomingIdents) > 0 {
			// load existing profiles that match the incoming profiles by identifiers
			stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, incomingIdents)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build query to load existing profiles")
			}
			if err := sqlx.SelectContext(ctx, tx, &existingProfiles, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "load existing profiles")
			}
		}

		// match the existing profiles to the incoming ones and keep the existing
		// ones that have not changed
		keepIdents := make([]string, 0, len(incomingIdents))
		for _, p := range existingProfiles {
			if newP := incomingProfs[p.Identifier]; newP != nil {
				if bytes.Equal(newP.Mobileconfig, p.Mobileconfig) {
					// the profile has not changed, keep the existing one
					keepIdents = append(keepIdents, p.Identifier)
					delete(incomingProfs, p.Identifier)
				}
			}
		}

		// profiles that are managed and delivered by Fleet
		fleetIdents := apple_mdm.ProfilesManagedByFleet()

		var (
			stmt string
			args []interface{}
			err  error
		)
		// delete the obsolete profiles (all those that are not in keepIdents or delivered by Fleet)
		stmt, args, err = sqlx.In(deleteProfilesNotInList, profTeamID, append(keepIdents, fleetIdents...))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete obsolete profiles")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete profiles")
		}

		// insert the new profiles and the ones that have changed
		for _, p := range incomingProfs {
			if _, err := tx.ExecContext(ctx, insertNewOrEditedProfile, profTeamID, p.Identifier, p.Name, p.Mobileconfig); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited profile with identifier %q", p.Identifier)
			}
		}
		return nil
	})
}

func (ds *Datastore) ListMDMAppleProfilesToInstall(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the desired state, can be obtained from a JOIN between
	// mdm_apple_configuration_profiles and hosts.
	// - Set B, the current state given by host_mdm_apple_profiles.
	//
	// A - B gives us the profiles that need to be installed.
	query := `
          SELECT ds.profile_id, ds.host_uuid, ds.profile_identifier, ds.profile_name
          FROM (
            SELECT
              macp.profile_id,
              h.uuid as host_uuid,
              macp.identifier as profile_identifier,
			  macp.name as profile_name
            FROM mdm_apple_configuration_profiles macp
            JOIN hosts h ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
            JOIN nano_enrollments ne ON ne.device_id = h.uuid
            WHERE h.platform = 'darwin' AND ne.enabled = 1
          ) as ds
          LEFT JOIN host_mdm_apple_profiles hmap
            ON hmap.profile_id = ds.profile_id AND hmap.host_uuid = ds.host_uuid
          WHERE
          hmap.profile_id IS NULL
          AND hmap.host_uuid IS NULL
          AND hmap.status != 'pending' OR hmap.status IS NULL
          AND hmap.operation_type != 'install' OR hmap.operation_type IS NULL
          -- accounts for the edge case of having profiles but not having hosts
          AND ds.host_uuid IS NOT NULL
	`

	var profiles []*fleet.MDMAppleProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader, &profiles, query)
	return profiles, err
}

func (ds *Datastore) ListMDMAppleProfilesToRemove(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
	// The query below is a set difference between:
	//
	// - Set A (ds), the desired state, can be obtained from a JOIN between
	// mdm_apple_configuration_profiles and hosts.
	// - Set B, the current state given by host_mdm_apple_profiles.
	//
	// B - A gives us the profiles that need to be removed.
	query := `
          SELECT hmap.profile_id, hmap.profile_identifier, hmap.profile_name, hmap.host_uuid
          FROM (
            SELECT h.uuid, macp.profile_id
            FROM mdm_apple_configuration_profiles macp
            JOIN hosts h ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
            JOIN nano_enrollments ne ON ne.device_id = h.uuid
            WHERE h.platform = 'darwin' AND ne.enabled = 1
          ) as ds
          RIGHT JOIN host_mdm_apple_profiles hmap
            ON hmap.profile_id = ds.profile_id AND hmap.host_uuid = ds.uuid
          WHERE ds.profile_id IS NULL AND ds.uuid IS NULL
          AND hmap.operation_type != 'remove'
	`

	var profiles []*fleet.MDMAppleProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader, &profiles, query)
	return profiles, err
}

func (ds *Datastore) GetMDMAppleProfilesContents(ctx context.Context, ids []uint) (map[uint]fleet.Mobileconfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	stmt := `
          SELECT profile_id, mobileconfig as mobileconfig
          FROM mdm_apple_configuration_profiles WHERE profile_id IN (?)
	`
	query, args, err := sqlx.In(stmt, ids)
	if err != nil {
		return nil, err
	}
	rows, err := ds.reader.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make(map[uint]fleet.Mobileconfig)
	for rows.Next() {
		var id uint
		var mobileconfig fleet.Mobileconfig
		if err := rows.Scan(&id, &mobileconfig); err != nil {
			return nil, err
		}
		results[id] = mobileconfig
	}
	return results, nil
}

func (ds *Datastore) BulkUpsertMDMAppleHostProfiles(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
	if len(payload) == 0 {
		return nil
	}

	var args []any
	var sb strings.Builder

	for _, p := range payload {
		args = append(args, p.ProfileID, p.ProfileIdentifier, p.ProfileName, p.HostUUID, p.Status, p.OperationType, p.CommandUUID)
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?),")
	}

	stmt := fmt.Sprintf(`
	    INSERT INTO host_mdm_apple_profiles (
              profile_id,
              profile_identifier,
              profile_name,
              host_uuid,
              status,
              operation_type,
              command_uuid
            )
            VALUES %s
	    ON DUPLICATE KEY UPDATE
              status = VALUES(status),
              operation_type = VALUES(operation_type),
              command_uuid = VALUES(command_uuid)`,
		strings.TrimSuffix(sb.String(), ","),
	)

	_, err := ds.writer.ExecContext(ctx, stmt, args...)
	return err
}

func (ds *Datastore) UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
	if profile.OperationType == fleet.MDMAppleOperationTypeRemove &&
		profile.Status != nil && *profile.Status == fleet.MDMAppleDeliveryApplied {
		_, err := ds.writer.ExecContext(ctx, `
          DELETE FROM host_mdm_apple_profiles
          WHERE host_uuid = ? AND command_uuid = ?
        `, profile.HostUUID, profile.CommandUUID)
		return err
	}

	_, err := ds.writer.ExecContext(ctx, `
          UPDATE host_mdm_apple_profiles
          SET status = ?, operation_type = ?, detail = ?
          WHERE host_uuid = ? AND command_uuid = ?
        `, profile.Status, profile.OperationType, profile.Detail, profile.HostUUID, profile.CommandUUID)
	return err
}

func (ds *Datastore) GetMDMAppleHostsProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleHostsProfilesSummary, error) {
	// TODO(sarah): add cases to handle Fleet-managed profiles (e.g., disk encryption)
	sqlFmt := `
SELECT
	count(
		CASE WHEN EXISTS (
			SELECT
				1 FROM host_mdm_apple_profiles hmap
			WHERE
				h.uuid = hmap.host_uuid
				AND hmap.status = 'failed') THEN
			1
		END) AS failed,
	count(
		CASE WHEN EXISTS (
			SELECT
				1 FROM host_mdm_apple_profiles hmap
			WHERE
				h.uuid = hmap.host_uuid
				AND hmap.status = 'pending')
			AND NOT EXISTS (
				SELECT
					1 FROM host_mdm_apple_profiles hmap
				WHERE
					h.uuid = hmap.host_uuid
					AND hmap.status = 'failed') THEN
			1
		END) AS pending,
	count(
		CASE WHEN EXISTS (
			SELECT
				1 FROM host_mdm_apple_profiles hmap
			WHERE
				h.uuid = hmap.host_uuid
				AND hmap.status = 'applied')
			AND NOT EXISTS (
				SELECT
					1 FROM host_mdm_apple_profiles hmap
				WHERE
					h.uuid = hmap.host_uuid
					AND(hmap.status = 'failed'
						OR hmap.status = 'pending')) THEN
			1
		END) AS applied
FROM
	hosts h
WHERE
	%s`

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = fmt.Sprintf("h.team_id = %d", *teamID)
	}

	var res fleet.MDMAppleHostsProfilesSummary
	err := sqlx.GetContext(ctx, ds.reader, &res, fmt.Sprintf(sqlFmt, teamFilter))
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (ds *Datastore) InsertMDMIdPAccount(ctx context.Context, account *fleet.MDMIdPAccount) error {
	stmt := `
      INSERT INTO mdm_idp_accounts
        (uuid, username, salt, entropy, iterations)
      VALUES
        (?, ?, ?, ?, ?)
      ON DUPLICATE KEY UPDATE
        username   = VALUES(username),
        salt       = VALUES(salt),
        entropy    = VALUES(entropy),
        iterations = VALUES(iterations)`

	_, err := ds.writer.ExecContext(ctx, stmt, account.UUID, account.Username, account.Salt, account.Entropy, account.Iterations)
	return ctxerr.Wrap(ctx, err, "creating new MDM IdP account")
}

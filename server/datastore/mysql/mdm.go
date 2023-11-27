package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error) {
	stmt := `
SELECT CASE
	WHEN EXISTS (SELECT 1 FROM nano_commands WHERE command_uuid = ?) THEN 'darwin'
	WHEN EXISTS (SELECT 1 FROM windows_mdm_commands WHERE command_uuid = ?) THEN 'windows'
	ELSE ''
END AS platform
`

	var p string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &p, stmt, commandUUID, commandUUID); err != nil {
		return "", err
	}
	if p == "" {
		return "", ctxerr.Wrap(ctx, notFound("MDMCommand").WithName(commandUUID))
	}

	return p, nil
}

func (ds *Datastore) ListMDMCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {
	appleStmt := `
SELECT
    nvq.id as host_uuid,
    nvq.command_uuid,
    COALESCE(NULLIF(nvq.status, ''), 'Pending') as status,
    COALESCE(nvq.result_updated_at, nvq.created_at) as updated_at,
    nvq.request_type,
    h.hostname,
    h.team_id
FROM
    nano_view_queue nvq
INNER JOIN
    hosts h
ON
    nvq.id = h.uuid
WHERE
   nvq.active = 1
`

	windowsStmt := `
SELECT
    mwe.host_uuid,
    wmc.command_uuid,
    COALESCE(NULLIF(wmcr.status_code, ''), 'Pending') as status,
    COALESCE(wmc.updated_at, wmc.created_at) as updated_at,
    wmc.target_loc_uri as request_type,
    h.hostname,
    h.team_id
FROM windows_mdm_commands wmc
LEFT JOIN windows_mdm_command_queue wmcq ON wmcq.command_uuid = wmc.command_uuid
LEFT JOIN windows_mdm_command_results wmcr ON wmc.command_uuid = wmcr.command_uuid
INNER JOIN mdm_windows_enrollments mwe ON wmcq.enrollment_id = mwe.id OR wmcr.enrollment_id = mwe.id
INNER JOIN hosts h ON h.uuid = mwe.host_uuid
`

	jointStmt := fmt.Sprintf(
		`SELECT * FROM ((%s) UNION ALL (%s)) as combined_commands WHERE %s`,
		appleStmt, windowsStmt, ds.whereFilterHostsByTeams(tmFilter, "h"),
	)
	jointStmt, params := appendListOptionsWithCursorToSQL(jointStmt, nil, &listOpts.ListOptions)
	var results []*fleet.MDMCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, jointStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}

func (ds *Datastore) BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if err := ds.batchSetMDMWindowsProfilesDB(ctx, tx, tmID, winProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set windows profiles")
		}

		if err := ds.batchSetMDMAppleProfilesDB(ctx, tx, tmID, macProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set apple profiles")
		}

		return nil
	})
}

func (ds *Datastore) ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {

	var profs []*fleet.MDMConfigProfilePayload

	const selectStmt = `
SELECT
	profile_id,
	team_id,
	name,
	platform,
	identifier,
	checksum,
	created_at,
	updated_at
FROM (
	SELECT
		CONVERT(profile_id, CHAR) as profile_id,
		team_id,
		name,
		'darwin' as platform,
		identifier,
		checksum,
		created_at,
		updated_at
	FROM
		mdm_apple_configuration_profiles
	WHERE
		team_id = ? AND
		identifier NOT IN (?)

	UNION

	SELECT
		profile_uuid as profile_id,
		team_id,
		name,
		'windows' as platform,
		'' as identifier,
		'' as checksum,
		created_at,
		updated_at
	FROM
		mdm_windows_configuration_profiles
	WHERE
		team_id = ?
) as combined_profiles
`

	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	fleetIdentsMap := mobileconfig.FleetPayloadIdentifiers()
	fleetIdentifiers := make([]string, 0, len(fleetIdentsMap))
	for k := range fleetIdentsMap {
		fleetIdentifiers = append(fleetIdentifiers, k)
	}

	args := []any{globalOrTeamID, fleetIdentifiers, globalOrTeamID}
	stmt, args := appendListOptionsWithCursorToSQL(selectStmt, args, &opt)

	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "sqlx.In ListMDMConfigProfiles")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profs, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select profiles")
	}

	var metaData *fleet.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(profs) > int(opt.PerPage) {
			metaData.HasNextResults = true
			profs = profs[:len(profs)-1]
		}
	}
	return profs, metaData, nil
}

// Note that team ID 0 is used for profiles that apply to hosts in no team
// (i.e. pass 0 in that case as part of the teamIDs slice). Only one of the
// slice arguments can have values.
func (ds *Datastore) BulkSetPendingMDMHostProfiles(
	ctx context.Context,
	hostIDs, teamIDs, profileIDs []uint,
	profileUUIDs, hostUUIDs []string,
) error {
	var countArgs int
	if len(hostIDs) > 0 {
		countArgs++
	}
	if len(teamIDs) > 0 {
		countArgs++
	}
	if len(profileIDs) > 0 {
		countArgs++
	}
	if len(profileUUIDs) > 0 {
		countArgs++
	}
	if len(hostUUIDs) > 0 {
		countArgs++
	}
	if countArgs > 1 {
		return errors.New("only one of hostIDs, teamIDs, profileIDs, profileUUIDs or hostUUIDs can be provided")
	}
	if countArgs == 0 {
		return nil
	}

	var (
		hosts    []fleet.Host
		args     []any
		uuidStmt string
	)

	switch {
	case len(hostUUIDs) > 0:
		// TODO: if a very large number (~65K) of uuids was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE uuid IN (?)`
		args = append(args, hostUUIDs)

	case len(hostIDs) > 0:
		// TODO: if a very large number (~65K) of uuids was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE id IN (?)`
		args = append(args, hostIDs)

	case len(teamIDs) > 0:
		// TODO: if a very large number (~65K) of team IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `SELECT uuid, platform FROM hosts WHERE `
		if len(teamIDs) == 1 && teamIDs[0] == 0 {
			uuidStmt += `team_id IS NULL`
		} else {
			uuidStmt += `team_id IN (?)`
			args = append(args, teamIDs)
			for _, tmID := range teamIDs {
				if tmID == 0 {
					uuidStmt += ` OR team_id IS NULL`
					break
				}
			}
		}

	case len(profileIDs) > 0:
		// TODO: if a very large number (~65K) of profile IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_apple_configuration_profiles macp
	ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
WHERE
	macp.profile_id IN (?) AND h.platform = 'darwin'`
		args = append(args, profileIDs)

	case len(profileUUIDs) > 0:
		// TODO: if a very large number (~65K) of profile IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_windows_configuration_profiles mawp
	ON h.team_id = mawp.team_id OR (h.team_id IS NULL AND mawp.team_id = 0)
WHERE
	mawp.profile_uuid IN (?) AND h.platform = 'windows'`
		args = append(args, profileUUIDs)

	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// TODO: this could be optimized to avoid querying for platform when
		// profileIDs or profileUUIDs are provided.
		if len(hosts) == 0 {
			uuidStmt, args, err := sqlx.In(uuidStmt, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "prepare query to load host UUIDs")
			}
			if err := sqlx.SelectContext(ctx, tx, &hosts, uuidStmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "execute query to load host UUIDs")
			}
		}

		var macHosts []string
		var winHosts []string
		for _, h := range hosts {
			switch h.Platform {
			case "darwin":
				macHosts = append(macHosts, h.UUID)
			case "windows":
				winHosts = append(winHosts, h.UUID)
			default:
				level.Debug(ds.logger).Log(
					"msg", "tried to set profile status for a host with unsupported platform",
					"platform", h.Platform,
					"host_uuid", h.UUID,
				)
			}
		}

		if err := ds.bulkSetPendingMDMAppleHostProfilesDB(ctx, tx, macHosts); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending apple host profiles")
		}

		if err := ds.bulkSetPendingMDMWindowsHostProfilesDB(ctx, tx, winHosts); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending windows host profiles")
		}

		return nil
	})
}

func (ds *Datastore) UpdateHostMDMProfilesVerification(ctx context.Context, host *fleet.Host, toVerify, toFail, toRetry []string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := setMDMProfilesVerifiedDB(ctx, tx, host, toVerify); err != nil {
			return err
		}
		if err := setMDMProfilesFailedDB(ctx, tx, host, toFail); err != nil {
			return err
		}
		if err := setMDMProfilesRetryDB(ctx, tx, host, toRetry); err != nil {
			return err
		}
		return nil
	})
}

// setMDMProfilesRetryDB sets the status of the given identifiers to retry (nil) and increments the retry count
func setMDMProfilesRetryDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	status = NULL,
	detail = '',
	retries = retries + 1
WHERE
	host_uuid = ?
	AND operation_type = ?
	AND %s IN(?)`

	args := []interface{}{
		host.UUID,
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}

	var stmt string
	switch host.Platform {
	case "darwin":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set retry host profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting retry host profiles")
	}
	return nil
}

// setMDMProfilesFailedDB sets the status of the given identifiers to failed if the current status
// is verifying or verified. It also sets the detail to a message indicating that the profile was
// either verifying or verified. Only profiles with the install operation type are updated.
func setMDMProfilesFailedDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	detail = if(status = ?, ?, ?),
	status = ?
WHERE
	host_uuid = ?
	AND status IN(?)
	AND operation_type = ?
	AND %s IN(?)`

	var stmt string
	switch host.Platform {
	case "darwin":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}

	args := []interface{}{
		fleet.MDMDeliveryVerifying,
		fleet.HostMDMProfileDetailFailedWasVerifying,
		fleet.HostMDMProfileDetailFailedWasVerified,
		fleet.MDMDeliveryFailed,
		host.UUID,
		[]interface{}{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerified},
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set failed host profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting failed host profiles")
	}
	return nil
}

// setMDMProfilesVerifiedDB sets the status of the given identifiers to verified if the current
// status is verifying. Only profiles with the install operation type are updated.
func setMDMProfilesVerifiedDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host, identifiersOrNames []string) error {
	if len(identifiersOrNames) == 0 {
		return nil
	}

	const baseStmt = `
UPDATE
	%s
SET
	detail = '',
	status = ?
WHERE
	host_uuid = ?
	AND status IN(?)
	AND operation_type = ?
	AND %s IN(?)`

	var stmt string
	switch host.Platform {
	case "darwin":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_apple_profiles", "profile_identifier")
	case "windows":
		stmt = fmt.Sprintf(baseStmt, "host_mdm_windows_profiles", "profile_name")
	default:
		return fmt.Errorf("unsupported platform %s", host.Platform)
	}

	args := []interface{}{
		fleet.MDMDeliveryVerified,
		host.UUID,
		[]interface{}{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryFailed},
		fleet.MDMOperationTypeInstall,
		identifiersOrNames,
	}
	stmt, args, err := sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building sql statement to set verified host macOS profiles")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "setting verified host profiles")
	}
	return nil
}

func (ds *Datastore) GetHostMDMProfilesExpectedForVerification(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}

	switch host.Platform {
	case "darwin":
		return ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID)
	case "windows":
		return ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", host.Platform)
	}
}

func (ds *Datastore) getHostMDMWindowsProfilesExpectedForVerification(ctx context.Context, teamID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
  SELECT name, syncml as raw_profile, updated_at as earliest_install_date
  FROM mdm_windows_configuration_profiles mwcp
  WHERE mwcp.team_id = ?
  `

	var profiles []*fleet.ExpectedMDMProfile
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, teamID)
	if err != nil {
		return nil, err
	}

	byName := make(map[string]*fleet.ExpectedMDMProfile, len(profiles))
	for _, r := range profiles {
		byName[r.Name] = r
	}

	return byName, nil
}

func (ds *Datastore) getHostMDMAppleProfilesExpectedForVerification(ctx context.Context, teamID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
SELECT
	identifier,
	earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(updated_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY
			checksum) cs
	ON macp.checksum = cs.checksum
WHERE
	macp.team_id = ?`

	var rows []*fleet.ExpectedMDMProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting expected profiles for host in team %d", teamID))
	}

	byIdentifier := make(map[string]*fleet.ExpectedMDMProfile, len(rows))
	for _, r := range rows {
		byIdentifier[r.Identifier] = r
	}

	return byIdentifier, nil
}

func (ds *Datastore) GetHostMDMProfilesRetryCounts(ctx context.Context, host *fleet.Host) ([]fleet.HostMDMProfileRetryCount, error) {
	const darwinStmt = `
SELECT
	profile_identifier,
	retries
FROM
	host_mdm_apple_profiles hmap
WHERE
	hmap.host_uuid = ?`

	const windowsStmt = `
SELECT
	profile_name,
	retries
FROM
	host_mdm_windows_profiles hmwp
WHERE
	hmwp.host_uuid = ?`

	var stmt string
	switch host.Platform {
	case "darwin":
		stmt = darwinStmt
	case "windows":
		stmt = windowsStmt
	default:
		return nil, fmt.Errorf("unsupported platform %s", host.Platform)
	}

	var dest []fleet.HostMDMProfileRetryCount
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt, host.UUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting retry counts for host %s", host.UUID))
	}

	return dest, nil
}

func (ds *Datastore) GetHostMDMProfileRetryCountByCommandUUID(ctx context.Context, host *fleet.Host, cmdUUID string) (fleet.HostMDMProfileRetryCount, error) {
	const darwinStmt = `
SELECT
	profile_identifier, retries
FROM
	host_mdm_apple_profiles hmap
WHERE
	hmap.host_uuid = ?
	AND hmap.command_uuid = ?`

	const windowsStmt = `
SELECT
	profile_uuid, retries
FROM
	host_mdm_windows_profiles hmwp
WHERE
	hmwp.host_uuid = ?
	AND hmwp.command_uuid = ?`

	var stmt string
	switch host.Platform {
	case "darwin":
		stmt = darwinStmt
	case "windows":
		stmt = windowsStmt
	default:
		return fleet.HostMDMProfileRetryCount{}, fmt.Errorf("unsupported platform %s", host.Platform)
	}

	var dest fleet.HostMDMProfileRetryCount
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, host.UUID, cmdUUID); err != nil {
		if err == sql.ErrNoRows {
			return dest, notFound("HostMDMCommand").WithMessage(fmt.Sprintf("command uuid %s not found for host uuid %s", cmdUUID, host.UUID))
		}
		return dest, ctxerr.Wrap(ctx, err, fmt.Sprintf("getting retry count for host %s command uuid %s", host.UUID, cmdUUID))
	}

	return dest, nil
}

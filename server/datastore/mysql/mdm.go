package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
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

func getCombinedMDMCommandsQuery() string {
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

	return fmt.Sprintf(
		`SELECT * FROM ((%s) UNION ALL (%s)) as combined_commands WHERE `,
		appleStmt, windowsStmt,
	)
}

func (ds *Datastore) ListMDMCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {

	jointStmt := getCombinedMDMCommandsQuery() + ds.whereFilterHostsByTeams(tmFilter, "h")
	jointStmt, params := appendListOptionsWithCursorToSQL(jointStmt, nil, &listOpts.ListOptions)
	var results []*fleet.MDMCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, jointStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}

func (ds *Datastore) getMDMCommand(ctx context.Context, q sqlx.QueryerContext, cmdUUID string) (*fleet.MDMCommand, error) {
	stmt := getCombinedMDMCommandsQuery() + "command_uuid = ?"

	var cmd fleet.MDMCommand
	if err := sqlx.GetContext(ctx, q, &cmd, stmt, cmdUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get mdm command by UUID")
	}
	return &cmd, nil
}

func (ds *Datastore) BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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
	// this lists custom profiles, it explicitly filters out the fleet-reserved
	// ones (reserved identifiers for Apple profiles, reserved names for Windows).

	var profs []*fleet.MDMConfigProfilePayload

	const selectStmt = `
SELECT
	profile_uuid,
	team_id,
	name,
	platform,
	identifier,
	checksum,
	created_at,
	uploaded_at
FROM (
	SELECT
		profile_uuid,
		team_id,
		name,
		'darwin' as platform,
		identifier,
		checksum,
		created_at,
		uploaded_at
	FROM
		mdm_apple_configuration_profiles
	WHERE
		team_id = ? AND
		identifier NOT IN (?)

	UNION

	SELECT
		profile_uuid,
		team_id,
		name,
		'windows' as platform,
		'' as identifier,
		'' as checksum,
		created_at,
		uploaded_at
	FROM
		mdm_windows_configuration_profiles
	WHERE
		team_id = ? AND
		name NOT IN (?)
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
	fleetNamesMap := mdm.FleetReservedProfileNames()
	fleetNames := make([]string, 0, len(fleetNamesMap))
	for k := range fleetNamesMap {
		fleetNames = append(fleetNames, k)
	}

	args := []any{globalOrTeamID, fleetIdentifiers, globalOrTeamID, fleetNames}
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

	// load the labels associated with those profiles
	var winProfUUIDs, macProfUUIDs []string
	for _, prof := range profs {
		if prof.Platform == "windows" {
			winProfUUIDs = append(winProfUUIDs, prof.ProfileUUID)
		} else {
			macProfUUIDs = append(macProfUUIDs, prof.ProfileUUID)
		}
	}
	labels, err := ds.listProfileLabelsForProfiles(ctx, winProfUUIDs, macProfUUIDs)
	if err != nil {
		return nil, nil, err
	}

	// match the labels with their profiles
	profMap := make(map[string]*fleet.MDMConfigProfilePayload, len(profs))
	for _, prof := range profs {
		profMap[prof.ProfileUUID] = prof
	}
	for _, label := range labels {
		if prof, ok := profMap[label.ProfileUUID]; ok {
			prof.Labels = append(prof.Labels, label)
		}
	}

	return profs, metaData, nil
}

func (ds *Datastore) listProfileLabelsForProfiles(ctx context.Context, winProfUUIDs, macProfUUIDs []string) ([]fleet.ConfigurationProfileLabel, error) {
	// load the labels associated with those profiles
	const labelsStmt = `
SELECT
	COALESCE(apple_profile_uuid, windows_profile_uuid) as profile_uuid,
	label_name,
	COALESCE(label_id, 0) as label_id,
	IF(label_id IS NULL, 1, 0) as broken
FROM
	mdm_configuration_profile_labels mcpl
WHERE
	mcpl.apple_profile_uuid IN (?) OR
	mcpl.windows_profile_uuid IN (?)
ORDER BY
	profile_uuid, label_name
`

	// ensure there's at least one (non-matching) value in the slice so the IN
	// clause is valid
	if len(winProfUUIDs) == 0 {
		winProfUUIDs = []string{"-"}
	}
	if len(macProfUUIDs) == 0 {
		macProfUUIDs = []string{"-"}
	}

	stmt, args, err := sqlx.In(labelsStmt, macProfUUIDs, winProfUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In to list labels for profiles")
	}

	var labels []fleet.ConfigurationProfileLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select profiles labels")
	}
	return labels, nil
}

// Note that team ID 0 is used for profiles that apply to hosts in no team
// (i.e. pass 0 in that case as part of the teamIDs slice). Only one of the
// slice arguments can have values.
func (ds *Datastore) BulkSetPendingMDMHostProfiles(
	ctx context.Context,
	hostIDs, teamIDs []uint,
	profileUUIDs, hostUUIDs []string,
) error {
	var (
		countArgs    int
		macProfUUIDs []string
		winProfUUIDs []string
	)

	if len(hostIDs) > 0 {
		countArgs++
	}
	if len(teamIDs) > 0 {
		countArgs++
	}
	if len(profileUUIDs) > 0 {
		countArgs++

		// split into mac and win profiles
		for _, puid := range profileUUIDs {
			if strings.HasPrefix(puid, "a") {
				macProfUUIDs = append(macProfUUIDs, puid)
			} else {
				winProfUUIDs = append(winProfUUIDs, puid)
			}
		}
	}
	if len(hostUUIDs) > 0 {
		countArgs++
	}
	if countArgs > 1 {
		return errors.New("only one of hostIDs, teamIDs, profileUUIDs or hostUUIDs can be provided")
	}
	if countArgs == 0 {
		return nil
	}
	if len(macProfUUIDs) > 0 && len(winProfUUIDs) > 0 {
		return errors.New("profile uuids must all be Apple or Windows profiles")
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

	case len(macProfUUIDs) > 0:
		// TODO: if a very large number (~65K) of profile UUIDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_apple_configuration_profiles macp
	ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
WHERE
	macp.profile_uuid IN (?) AND h.platform = 'darwin'`
		args = append(args, macProfUUIDs)

	case len(winProfUUIDs) > 0:
		// TODO: if a very large number (~65K) of profile IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_windows_configuration_profiles mawp
	ON h.team_id = mawp.team_id OR (h.team_id IS NULL AND mawp.team_id = 0)
WHERE
	mawp.profile_uuid IN (?) AND h.platform = 'windows'`
		args = append(args, winProfUUIDs)

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
		[]interface{}{
			fleet.MDMDeliveryVerifying,
			fleet.MDMDeliveryVerified,
		},
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
		[]interface{}{
			fleet.MDMDeliveryPending,
			fleet.MDMDeliveryVerifying,
			fleet.MDMDeliveryFailed,
		},
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
		return ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID, host.ID)
	case "windows":
		return ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID, host.ID)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", host.Platform)
	}
}

func (ds *Datastore) getHostMDMWindowsProfilesExpectedForVerification(ctx context.Context, teamID, hostID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
SELECT
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	0 AS count_profile_labels,
	0 AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
WHERE
	mwcp.team_id = ?
	AND NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.apple_profile_uuid = mwcp.profile_uuid)
GROUP BY  name, syncml
	UNION
	SELECT
		name,
		syncml AS raw_profile,
		min(mwcp.uploaded_at) AS earliest_install_date,
		COUNT(*) AS count_profile_labels,
		COUNT(lm.label_id) AS count_host_labels
	FROM
		mdm_windows_configuration_profiles mwcp
		JOIN mdm_configuration_profile_labels mcpl ON mcpl.windows_profile_uuid = mwcp.profile_uuid
		LEFT OUTER JOIN label_membership lm ON lm.label_id = mcpl.label_id
			AND lm.host_id = ?
	WHERE
		mwcp.team_id = ?
	GROUP BY
		name, syncml
	HAVING
		count_profile_labels > 0
		AND count_host_labels = count_profile_labels

  `

	var profiles []*fleet.ExpectedMDMProfile
	// Note: teamID provided twice
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, teamID, hostID, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "running query for windows profiles")
	}

	byName := make(map[string]*fleet.ExpectedMDMProfile, len(profiles))
	for _, r := range profiles {
		byName[r.Name] = r
	}

	return byName, nil
}

func (ds *Datastore) getHostMDMAppleProfilesExpectedForVerification(ctx context.Context, teamID, hostID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
SELECT
	macp.identifier AS identifier,
	0 AS count_profile_labels,
	0 AS count_host_labels,
	earliest_install_date
FROM
	mdm_apple_configuration_profiles macp
	JOIN (
		SELECT
			checksum,
			min(uploaded_at) AS earliest_install_date
		FROM
			mdm_apple_configuration_profiles
		GROUP BY
			checksum) cs ON macp.checksum = cs.checksum
WHERE
	macp.team_id = ?
	AND NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.apple_profile_uuid = macp.profile_uuid)
	UNION
	-- label-based profiles where the host is a member of all the labels
	SELECT
		macp.identifier AS identifier,
		COUNT(*) AS count_profile_labels,
		COUNT(lm.label_id) AS count_host_labels,
		min(earliest_install_date) AS earliest_install_date
	FROM
		mdm_apple_configuration_profiles macp
		JOIN (
			SELECT
				checksum,
				min(uploaded_at) AS earliest_install_date
			FROM
				mdm_apple_configuration_profiles
			GROUP BY
				checksum) cs ON macp.checksum = cs.checksum
		JOIN mdm_configuration_profile_labels mcpl ON mcpl.apple_profile_uuid = macp.profile_uuid
		LEFT OUTER JOIN label_membership lm ON lm.label_id = mcpl.label_id
			AND lm.host_id = ?
	WHERE
		macp.team_id = ?
	GROUP BY
		identifier
	HAVING
		count_profile_labels > 0
		AND count_host_labels = count_profile_labels
	`

	var rows []*fleet.ExpectedMDMProfile
	// Note: teamID provided twice
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, teamID, hostID, teamID); err != nil {
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

func batchSetProfileLabelAssociationsDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	profileLabels []fleet.ConfigurationProfileLabel,
	platform string,
) error {
	if len(profileLabels) == 0 {
		return nil
	}

	var platformPrefix string
	switch platform {
	case "darwin":
		// map "darwin" to "apple" to be consistent with other
		// "platform-agnostic" datastore methods. We initially used "darwin"
		// because that's what hosts use (as the data is reported by osquery)
		// and sometimes we want to dynamically select a table based on host
		// data.
		platformPrefix = "apple"
	case "windows":
		platformPrefix = "windows"
	default:
		return fmt.Errorf("unsupported platform %s", platform)
	}

	// delete any profile+label tuple that is NOT in the list of provided tuples
	// but are associated with the provided profiles (so we don't delete
	// unrelated profile+label tuples)
	deleteStmt := `
	  DELETE FROM mdm_configuration_profile_labels
	  WHERE (%s_profile_uuid, label_id) NOT IN (%s) AND
	  %s_profile_uuid IN (?)
	`

	upsertStmt := `
	  INSERT INTO mdm_configuration_profile_labels
              (%s_profile_uuid, label_id, label_name)
          VALUES
              %s
          ON DUPLICATE KEY UPDATE
              label_id = VALUES(label_id)
	`

	var (
		insertBuilder strings.Builder
		deleteBuilder strings.Builder
		insertParams  []any
		deleteParams  []any

		setProfileUUIDs = make(map[string]struct{})
	)
	for i, pl := range profileLabels {
		if i > 0 {
			insertBuilder.WriteString(",")
			deleteBuilder.WriteString(",")
		}
		insertBuilder.WriteString("(?, ?, ?)")
		deleteBuilder.WriteString("(?, ?)")
		insertParams = append(insertParams, pl.ProfileUUID, pl.LabelID, pl.LabelName)
		deleteParams = append(deleteParams, pl.ProfileUUID, pl.LabelID)

		setProfileUUIDs[pl.ProfileUUID] = struct{}{}
	}

	_, err := tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, platformPrefix, insertBuilder.String()), insertParams...)
	if err != nil {
		if isChildForeignKeyError(err) {
			// one of the provided labels doesn't exist
			return foreignKey("mdm_configuration_profile_labels", fmt.Sprintf("(profile, label)=(%v)", insertParams))
		}

		return ctxerr.Wrap(ctx, err, "setting label associations for profile")
	}

	deleteStmt = fmt.Sprintf(deleteStmt, platformPrefix, deleteBuilder.String(), platformPrefix)

	profUUIDs := make([]string, 0, len(setProfileUUIDs))
	for k := range setProfileUUIDs {
		profUUIDs = append(profUUIDs, k)
	}
	deleteArgs := append(deleteParams, profUUIDs)

	deleteStmt, args, err := sqlx.In(deleteStmt, deleteArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In delete labels for profiles")
	}
	if _, err := tx.ExecContext(ctx, deleteStmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting labels for profiles")
	}

	return nil
}

func (ds *Datastore) MDMGetEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	// Currently, there can only be one EULA in the database, and we're
	// hardcoding it's id to be 1 in order to enforce this restriction.
	stmt := "SELECT name, created_at, token FROM eulas WHERE id = 1"
	var eula fleet.MDMEULA
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &eula, stmt); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMEULA"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get EULA metadata")
	}
	return &eula, nil
}

func (ds *Datastore) MDMGetEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	stmt := "SELECT name, bytes FROM eulas WHERE token = ?"
	var eula fleet.MDMEULA
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &eula, stmt, token); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMEULA"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get EULA bytes")
	}
	return &eula, nil
}

func (ds *Datastore) MDMInsertEULA(ctx context.Context, eula *fleet.MDMEULA) error {
	// We're intentionally hardcoding the id to be 1 because we only want to
	// allow one EULA.
	stmt := `
          INSERT INTO eulas (id, name, bytes, token)
	  VALUES (1, ?, ?, ?)
	`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, eula.Name, eula.Bytes, eula.Token)
	if err != nil {
		if isDuplicate(err) {
			return ctxerr.Wrap(ctx, alreadyExists("MDMEULA", eula.Token))
		}
		return ctxerr.Wrap(ctx, err, "create EULA")
	}

	return nil
}

func (ds *Datastore) MDMDeleteEULA(ctx context.Context, token string) error {
	stmt := "DELETE FROM eulas WHERE token = ?"
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, token)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete EULA")
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMEULA"))
	}
	return nil
}

func (ds *Datastore) GetHostCertAssociationsToExpire(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityAssociation, error) {
	// TODO(roberto): this is not good because we don't have any indexes on
	// h.uuid, due to time constraints, I'm assuming that this
	// function is called with a relatively low amount of shas
	//
	// Note that we use GROUP BY because we can't guarantee unique entries
	// based on uuid in the hosts table.
	stmt, args, err := sqlx.In(
		`SELECT
			h.uuid as host_uuid,
			ncaa.sha256 as sha256,
			COALESCE(MAX(hm.fleet_enroll_ref), '') as enroll_reference
		 FROM
			nano_cert_auth_associations ncaa
			LEFT JOIN hosts h ON h.uuid = ncaa.id
			LEFT JOIN host_mdm hm ON hm.host_id = h.id
		 WHERE
			cert_not_valid_after BETWEEN '0000-00-00' AND DATE_ADD(CURDATE(), INTERVAL ? DAY)
			AND renew_command_uuid IS NULL
		GROUP BY
			host_uuid, ncaa.sha256, cert_not_valid_after
		ORDER BY cert_not_valid_after ASC
		LIMIT ?
		`, expiryDays, limit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building sqlx.In query")
	}

	var uuids []fleet.SCEPIdentityAssociation
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &uuids, stmt, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get identity certs close to expiry")
	}
	return uuids, nil
}

func (ds *Datastore) SetCommandForPendingSCEPRenewal(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
	if len(assocs) == 0 {
		return nil
	}

	var sb strings.Builder
	args := make([]any, len(assocs)*3)
	for i, assoc := range assocs {
		sb.WriteString("(?, ?, ?),")
		args[i*3] = assoc.HostUUID
		args[i*3+1] = assoc.SHA256
		args[i*3+2] = cmdUUID
	}

	stmt := fmt.Sprintf(`
		INSERT INTO nano_cert_auth_associations (id, sha256, renew_command_uuid) VALUES %s
		ON DUPLICATE KEY UPDATE
			renew_command_uuid = VALUES(renew_command_uuid)
	`, strings.TrimSuffix(sb.String(), ","))

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return fmt.Errorf("failed to update cert associations: %w", err)
		}

		// NOTE: we can't use insertOnDuplicateDidInsert because the
		// LastInsertId check only works tables that have an
		// auto-incrementing primary key. See notes in that function
		// and insertOnDuplicateDidUpdate to understand the mechanism.
		affected, _ := res.RowsAffected()
		if affected == 1 {
			return errors.New("this function can only be used to update existing associations")
		}

		return nil
	})
}

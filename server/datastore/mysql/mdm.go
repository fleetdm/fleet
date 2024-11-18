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
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/go-kit/log/level"
	"github.com/google/go-cmp/cmp"
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

func getCombinedMDMCommandsQuery(ds *Datastore, hostFilter string) (string, []interface{}) {
	appleStmt := `
SELECT
    nvq.id as host_uuid,
    nvq.command_uuid,
    COALESCE(NULLIF(nvq.status, ''), 'Pending') as status,
    COALESCE(nvq.result_updated_at, nvq.created_at) as updated_at,
    nvq.request_type as request_type,
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
WHERE TRUE
`

	var params []interface{}
	appleStmtWithFilter, params := ds.whereFilterHostsByIdentifier(hostFilter, appleStmt, params)
	windowsStmtWithFilter, params := ds.whereFilterHostsByIdentifier(hostFilter, windowsStmt, params)

	stmt := fmt.Sprintf(
		`SELECT * FROM ((%s) UNION ALL (%s)) as combined_commands WHERE `,
		appleStmtWithFilter, windowsStmtWithFilter,
	)

	return stmt, params
}

func (ds *Datastore) ListMDMCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {
	jointStmt, params := getCombinedMDMCommandsQuery(ds, listOpts.Filters.HostIdentifier)
	jointStmt += ds.whereFilterHostsByTeams(tmFilter, "h")
	jointStmt, params = addRequestTypeFilter(jointStmt, &listOpts.Filters, params)
	jointStmt, params = appendListOptionsWithCursorToSQL(jointStmt, params, &listOpts.ListOptions)
	var results []*fleet.MDMCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, jointStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}

func addRequestTypeFilter(stmt string, filter *fleet.MDMCommandFilters, params []interface{}) (string, []interface{}) {
	if filter.RequestType != "" {
		stmt += " AND request_type = ?"
		params = append(params, filter.RequestType)
	}

	return stmt, params
}

func (ds *Datastore) getMDMCommand(ctx context.Context, q sqlx.QueryerContext, cmdUUID string) (*fleet.MDMCommand, error) {
	stmt, _ := getCombinedMDMCommandsQuery(ds, "")
	stmt += "command_uuid = ?"

	var cmd fleet.MDMCommand
	if err := sqlx.GetContext(ctx, q, &cmd, stmt, cmdUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get mdm command by UUID")
	}
	return &cmd, nil
}

func (ds *Datastore) BatchSetMDMProfiles(ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile,
	winProfiles []*fleet.MDMWindowsConfigProfile, macDeclarations []*fleet.MDMAppleDeclaration) (updates fleet.MDMProfilesUpdates,
	err error,
) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		if updates.WindowsConfigProfile, err = ds.batchSetMDMWindowsProfilesDB(ctx, tx, tmID, winProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set windows profiles")
		}

		if updates.AppleConfigProfile, err = ds.batchSetMDMAppleProfilesDB(ctx, tx, tmID, macProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set apple profiles")
		}

		if _, updates.AppleDeclaration, err = ds.batchSetMDMAppleDeclarations(ctx, tx, tmID, macDeclarations); err != nil {
			return ctxerr.Wrap(ctx, err, "batch set apple declarations")
		}

		return nil
	})
	return updates, err
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

	UNION ALL

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

	UNION ALL

	SELECT
		declaration_uuid AS profile_uuid,
		team_id,
		name,
		'darwin' AS platform,
		identifier,
		checksum AS checksum,
		created_at,
		uploaded_at
	FROM mdm_apple_declarations
	WHERE team_id = ? AND
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

	args := []any{globalOrTeamID, fleetIdentifiers, globalOrTeamID, fleetNames, globalOrTeamID, fleetNames}
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
		if len(profs) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			profs = profs[:len(profs)-1]
		}
	}

	// load the labels associated with those profiles
	var winProfUUIDs, macProfUUIDs, macDeclUUIDs []string
	for _, prof := range profs {
		if prof.Platform == "windows" {
			winProfUUIDs = append(winProfUUIDs, prof.ProfileUUID)
		} else {
			if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
				macDeclUUIDs = append(macDeclUUIDs, prof.ProfileUUID)
				continue
			}

			macProfUUIDs = append(macProfUUIDs, prof.ProfileUUID)
		}
	}
	labels, err := ds.listProfileLabelsForProfiles(ctx, winProfUUIDs, macProfUUIDs, macDeclUUIDs)
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
			switch {
			case label.Exclude && label.RequireAll:
				// this should never happen so log it for debugging
				level.Debug(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all",
					"profile_uuid", label.ProfileUUID,
					"label_name", label.LabelName,
				)
			case label.Exclude && !label.RequireAll:
				prof.LabelsExcludeAny = append(prof.LabelsExcludeAny, label)
			case !label.Exclude && !label.RequireAll:
				prof.LabelsIncludeAny = append(prof.LabelsIncludeAny, label)
			default:
				// default include all
				prof.LabelsIncludeAll = append(prof.LabelsIncludeAll, label)
			}
		}
	}

	return profs, metaData, nil
}

func (ds *Datastore) listProfileLabelsForProfiles(ctx context.Context, winProfUUIDs, macProfUUIDs, macDeclUUIDs []string) ([]fleet.ConfigurationProfileLabel, error) {
	// load the labels associated with those profiles
	const labelsStmt = `
SELECT
	COALESCE(apple_profile_uuid, windows_profile_uuid) as profile_uuid,
	label_name,
	COALESCE(label_id, 0) as label_id,
	IF(label_id IS NULL, 1, 0) as broken,
	exclude,
	require_all
FROM
	mdm_configuration_profile_labels mcpl
WHERE
	mcpl.apple_profile_uuid IN (?) OR
	mcpl.windows_profile_uuid IN (?)
UNION ALL
SELECT
	apple_declaration_uuid as profile_uuid,
	label_name,
	COALESCE(label_id, 0) as label_id,
	IF(label_id IS NULL, 1, 0) as broken,
	exclude,
	require_all
FROM
	mdm_declaration_labels mdl
WHERE
	mdl.apple_declaration_uuid IN (?)
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
	if len(macDeclUUIDs) == 0 {
		macDeclUUIDs = []string{"-"}
	}

	stmt, args, err := sqlx.In(labelsStmt, macProfUUIDs, winProfUUIDs, macDeclUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In to list labels for profiles")
	}

	var labels []fleet.ConfigurationProfileLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select profiles labels")
	}
	return labels, nil
}

func (ds *Datastore) BulkSetPendingMDMHostProfiles(
	ctx context.Context,
	hostIDs, teamIDs []uint,
	profileUUIDs, hostUUIDs []string,
) (updates fleet.MDMProfilesUpdates, err error) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		updates, err = ds.bulkSetPendingMDMHostProfilesDB(ctx, tx, hostIDs, teamIDs, profileUUIDs, hostUUIDs)
		return err
	})
	return updates, err
}

// Note that team ID 0 is used for profiles that apply to hosts in no team
// (i.e. pass 0 in that case as part of the teamIDs slice). Only one of the
// slice arguments can have values.
func (ds *Datastore) bulkSetPendingMDMHostProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostIDs, teamIDs []uint,
	profileUUIDs, hostUUIDs []string,
) (updates fleet.MDMProfilesUpdates, err error) {
	var (
		countArgs     int
		macProfUUIDs  []string
		winProfUUIDs  []string
		hasAppleDecls bool
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
			if strings.HasPrefix(puid, fleet.MDMAppleProfileUUIDPrefix) { //nolint:gocritic // ignore ifElseChain
				macProfUUIDs = append(macProfUUIDs, puid)
			} else if strings.HasPrefix(puid, fleet.MDMAppleDeclarationUUIDPrefix) {
				hasAppleDecls = true
			} else {
				// Note: defaulting to windows profiles without checking the prefix as
				// many tests fail otherwise and it's a whole rabbit hole that I can't
				// address at the moment.
				winProfUUIDs = append(winProfUUIDs, puid)
			}
		}
	}
	if len(hostUUIDs) > 0 {
		countArgs++
	}
	if countArgs > 1 {
		return updates, errors.New("only one of hostIDs, teamIDs, profileUUIDs or hostUUIDs can be provided")
	}
	if countArgs == 0 {
		return updates, nil
	}

	var countProfUUIDs int
	if len(macProfUUIDs) > 0 {
		countProfUUIDs++
	}
	if len(winProfUUIDs) > 0 {
		countProfUUIDs++
	}
	if hasAppleDecls {
		countProfUUIDs++
	}
	if countProfUUIDs > 1 {
		return updates, errors.New("profile uuids must be all Apple profiles, all Apple declarations, or all Windows profiles")
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
		// TODO: if a very large number (~65K/2) of profile UUIDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_apple_configuration_profiles macp
	ON h.team_id = macp.team_id OR (h.team_id IS NULL AND macp.team_id = 0)
LEFT JOIN host_mdm_apple_profiles hmap
	ON h.uuid = hmap.host_uuid
WHERE
	macp.profile_uuid IN (?) AND (h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')
OR
	hmap.profile_uuid IN (?) AND (h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')`
		args = append(args, macProfUUIDs, macProfUUIDs)

	case len(winProfUUIDs) > 0:
		// TODO: if a very large number (~65K/2) of profile IDs was provided, could
		// result in too many placeholders (not an immediate concern).
		uuidStmt = `
SELECT DISTINCT h.uuid, h.platform
FROM hosts h
JOIN mdm_windows_configuration_profiles mawp
	ON h.team_id = mawp.team_id OR (h.team_id IS NULL AND mawp.team_id = 0)
LEFT JOIN host_mdm_windows_profiles hmwp
	ON h.uuid = hmwp.host_uuid
WHERE
	mawp.profile_uuid IN (?) AND h.platform = 'windows'
OR
	hmwp.profile_uuid IN (?) AND h.platform = 'windows'`
		args = append(args, winProfUUIDs, winProfUUIDs)

	}

	// TODO: this could be optimized to avoid querying for platform when
	// profileIDs or profileUUIDs are provided.
	if len(hosts) == 0 && !hasAppleDecls {
		uuidStmt, args, err := sqlx.In(uuidStmt, args...)
		if err != nil {
			return updates, ctxerr.Wrap(ctx, err, "prepare query to load host UUIDs")
		}
		if err := sqlx.SelectContext(ctx, tx, &hosts, uuidStmt, args...); err != nil {
			return updates, ctxerr.Wrap(ctx, err, "execute query to load host UUIDs")
		}
	}

	var appleHosts []string
	var winHosts []string
	for _, h := range hosts {
		switch h.Platform {
		case "darwin", "ios", "ipados":
			appleHosts = append(appleHosts, h.UUID)
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

	updates.AppleConfigProfile, err = ds.bulkSetPendingMDMAppleHostProfilesDB(ctx, tx, appleHosts, profileUUIDs)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending apple host profiles")
	}

	updates.WindowsConfigProfile, err = ds.bulkSetPendingMDMWindowsHostProfilesDB(ctx, tx, winHosts, profileUUIDs)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending windows host profiles")
	}

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}
	// TODO(roberto): this method currently sets the state of all
	// declarations for all hosts. I don't see an immediate concern
	// (and my hunch is that we could even do the same for
	// profiles) but this could be optimized to use only a provided
	// set of host uuids.
	_, updates.AppleDeclaration, err = mdmAppleBatchSetHostDeclarationStateDB(ctx, tx, batchSize, nil)
	if err != nil {
		return updates, ctxerr.Wrap(ctx, err, "bulk set pending apple declarations")
	}

	return updates, nil
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
	case "darwin", "ios", "ipados":
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
	case "darwin", "ios", "ipados":
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
	case "darwin", "ios", "ipados":
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
	case "darwin", "ios", "ipados":
		return ds.getHostMDMAppleProfilesExpectedForVerification(ctx, teamID, host.ID)
	case "windows":
		return ds.getHostMDMWindowsProfilesExpectedForVerification(ctx, teamID, host.ID)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", host.Platform)
	}
}

func (ds *Datastore) getHostMDMWindowsProfilesExpectedForVerification(ctx context.Context, teamID, hostID uint) (map[string]*fleet.ExpectedMDMProfile, error) {
	stmt := `
-- profiles without labels
SELECT
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	0 AS count_profile_labels,
	0 AS count_non_broken_labels,
	0 AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
WHERE
	mwcp.team_id = ? AND
	NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.windows_profile_uuid = mwcp.profile_uuid
	)
GROUP BY name, syncml

UNION

-- label-based profiles where the host is a member of all the labels (include-all).
-- by design, "include" labels cannot match if they are broken (the host cannot be
-- a member of a deleted label).
SELECT
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) as count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 0
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	mwcp.team_id = ?
GROUP BY
	name, syncml
HAVING
	count_profile_labels > 0 AND
	count_host_labels = count_profile_labels

UNION

-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
-- explicitly ignore profiles with broken excluded labels so that they are never applied.
SELECT
	name,
	syncml AS raw_profile,
	min(mwcp.uploaded_at) AS earliest_install_date,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) as count_non_broken_labels,
	COUNT(lm.label_id) AS count_host_labels
FROM
	mdm_windows_configuration_profiles mwcp
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.windows_profile_uuid = mwcp.profile_uuid AND mcpl.exclude = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	mwcp.team_id = ?
GROUP BY
	name, syncml
HAVING
	-- considers only the profiles with labels, without any broken label, and with the host not in any label
	count_profile_labels > 0 AND
	count_profile_labels = count_non_broken_labels AND
	count_host_labels = 0
`
	var profiles []*fleet.ExpectedMDMProfile
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, teamID, hostID, teamID, hostID, teamID)
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
-- profiles without labels
SELECT
	macp.identifier AS identifier,
	0 AS count_profile_labels,
	0 AS count_non_broken_labels,
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
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
WHERE
	macp.team_id = ? AND
	NOT EXISTS (
		SELECT
			1
		FROM
			mdm_configuration_profile_labels mcpl
		WHERE
			mcpl.apple_profile_uuid = macp.profile_uuid
	)

UNION

-- label-based profiles where the host is a member of all the labels (include-all)
-- by design, "include" labels cannot match if they are broken (the host cannot be
-- a member of a deleted label).
SELECT
	macp.identifier AS identifier,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) AS count_non_broken_labels,
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
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.apple_profile_uuid = macp.profile_uuid AND mcpl.exclude = 0
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	macp.team_id = ?
GROUP BY
	identifier
HAVING
	count_profile_labels > 0 AND
	count_host_labels = count_profile_labels

UNION

-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
-- explicitly ignore profiles with broken excluded labels so that they are never applied.
SELECT
	macp.identifier AS identifier,
	COUNT(*) AS count_profile_labels,
	COUNT(mcpl.label_id) AS count_non_broken_labels,
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
		GROUP BY checksum
	) cs ON macp.checksum = cs.checksum
	JOIN mdm_configuration_profile_labels mcpl
		ON mcpl.apple_profile_uuid = macp.profile_uuid AND mcpl.exclude = 1
	LEFT OUTER JOIN label_membership lm
		ON lm.label_id = mcpl.label_id AND lm.host_id = ?
WHERE
	macp.team_id = ?
GROUP BY
	identifier
HAVING
	-- considers only the profiles with labels, without any broken label, and with the host not in any label
	count_profile_labels > 0 AND
	count_profile_labels = count_non_broken_labels AND
	count_host_labels = 0
`

	var rows []*fleet.ExpectedMDMProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, teamID, hostID, teamID, hostID, teamID); err != nil {
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
	case "darwin", "ios", "ipados":
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
	case "darwin", "ios", "ipados":
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
) (updatedDB bool, err error) {
	if len(profileLabels) == 0 {
		// FIXME: At what point are we deleting all labels for a profile (e.g., the user might
		// remove all labels from an existing profile)?
		return false, nil
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
		return false, fmt.Errorf("unsupported platform %s", platform)
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
              (%s_profile_uuid, label_id, label_name, exclude, require_all)
          VALUES
              %s
          ON DUPLICATE KEY UPDATE
              label_id = VALUES(label_id),
              exclude = VALUES(exclude),
			  require_all = VALUES(require_all)
	`

	selectStmt := `
		SELECT %s_profile_uuid as profile_uuid, label_id, label_name, exclude, require_all FROM mdm_configuration_profile_labels
		WHERE (%s_profile_uuid, label_name) IN (%s)
	`

	var (
		insertBuilder         strings.Builder
		selectOrDeleteBuilder strings.Builder
		selectParams          []any
		insertParams          []any
		deleteParams          []any

		setProfileUUIDs = make(map[string]struct{})
	)
	labelsToInsert := make(map[string]*fleet.ConfigurationProfileLabel, len(profileLabels))
	for i, pl := range profileLabels {
		labelsToInsert[fmt.Sprintf("%s\n%s", pl.ProfileUUID, pl.LabelName)] = &profileLabels[i]
		if i > 0 {
			insertBuilder.WriteString(",")
			selectOrDeleteBuilder.WriteString(",")
		}
		insertBuilder.WriteString("(?, ?, ?, ?, ?)")
		selectOrDeleteBuilder.WriteString("(?, ?)")
		selectParams = append(selectParams, pl.ProfileUUID, pl.LabelName)
		insertParams = append(insertParams, pl.ProfileUUID, pl.LabelID, pl.LabelName, pl.Exclude, pl.RequireAll)
		deleteParams = append(deleteParams, pl.ProfileUUID, pl.LabelID)

		setProfileUUIDs[pl.ProfileUUID] = struct{}{}
	}

	// Determine if we need to update the database
	var existingProfileLabels []fleet.ConfigurationProfileLabel
	err = sqlx.SelectContext(ctx, tx, &existingProfileLabels,
		fmt.Sprintf(selectStmt, platformPrefix, platformPrefix, selectOrDeleteBuilder.String()), selectParams...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "selecting existing profile labels")
	}

	updateNeeded := false
	if len(existingProfileLabels) == len(labelsToInsert) {
		for _, existing := range existingProfileLabels {
			toInsert, ok := labelsToInsert[fmt.Sprintf("%s\n%s", existing.ProfileUUID, existing.LabelName)]
			// The fleet.ConfigurationProfileLabel struct has no pointers, so we can use standard cmp.Equal
			if !ok || !cmp.Equal(existing, *toInsert) {
				updateNeeded = true
				break
			}
		}
	} else {
		updateNeeded = true
	}

	if updateNeeded {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, platformPrefix, insertBuilder.String()), insertParams...)
		if err != nil {
			if isChildForeignKeyError(err) {
				// one of the provided labels doesn't exist
				return false, foreignKey("mdm_configuration_profile_labels", fmt.Sprintf("(profile, label)=(%v)", insertParams))
			}

			return false, ctxerr.Wrap(ctx, err, "setting label associations for profile")
		}
		updatedDB = true
	}

	deleteStmt = fmt.Sprintf(deleteStmt, platformPrefix, selectOrDeleteBuilder.String(), platformPrefix)

	profUUIDs := make([]string, 0, len(setProfileUUIDs))
	for k := range setProfileUUIDs {
		profUUIDs = append(profUUIDs, k)
	}
	deleteArgs := deleteParams
	deleteArgs = append(deleteArgs, profUUIDs)

	deleteStmt, args, err := sqlx.In(deleteStmt, deleteArgs...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "sqlx.In delete labels for profiles")
	}
	var result sql.Result
	if result, err = tx.ExecContext(ctx, deleteStmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "deleting labels for profiles")
	}
	if result != nil {
		rows, _ := result.RowsAffected()
		updatedDB = updatedDB || rows > 0
	}

	return updatedDB, nil
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
		if IsDuplicate(err) {
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
	stmt, args, err := sqlx.In(`
SELECT
    h.uuid AS host_uuid,
    ncaa.sha256 AS sha256,
    COALESCE(MAX(hm.fleet_enroll_ref), '') AS enroll_reference,
    ne.enrolled_from_migration
FROM (
    -- grab only the latest certificate associated with this device
    SELECT
        n1.id,
	n1.sha256,
	n1.cert_not_valid_after,
	n1.renew_command_uuid
    FROM
        nano_cert_auth_associations n1
    WHERE
        n1.sha256 = (
            SELECT
                n2.sha256
            FROM
                nano_cert_auth_associations n2
            WHERE
                n1.id = n2.id
            ORDER BY
                n2.created_at DESC,
                n2.sha256 ASC
            LIMIT 1
        )
) ncaa
JOIN
    hosts h ON h.uuid = ncaa.id
LEFT JOIN
    host_mdm hm ON hm.host_id = h.id
LEFT JOIN
    nano_enrollments ne ON ne.id = ncaa.id
WHERE
    ncaa.cert_not_valid_after BETWEEN '0000-00-00' AND DATE_ADD(CURDATE(), INTERVAL ? DAY)
    AND ncaa.renew_command_uuid IS NULL
    AND ne.enabled = 1
GROUP BY
    host_uuid, ncaa.sha256, ncaa.cert_not_valid_after
ORDER BY
    cert_not_valid_after ASC
LIMIT ?`, expiryDays, limit)
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

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

func (ds *Datastore) CleanSCEPRenewRefs(ctx context.Context, hostUUID string) error {
	stmt := `
	UPDATE nano_cert_auth_associations
	SET renew_command_uuid = NULL
	WHERE id = ?
	ORDER BY created_at desc
	LIMIT 1`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning SCEP renew references")
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		return ctxerr.Errorf(ctx, "nano association for host.uuid %s doesn't exist", hostUUID)
	}

	return nil
}

func (ds *Datastore) GetHostMDMProfileInstallStatus(ctx context.Context, hostUUID string, profUUID string) (fleet.MDMDeliveryStatus, error) {
	table, column, err := getTableAndColumnNameForHostMDMProfileUUID(profUUID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting table and column")
	}

	selectStmt := fmt.Sprintf(`
SELECT
	COALESCE(status, ?) as status
	FROM
	%s
WHERE
	operation_type = ?
	AND host_uuid = ?
	AND %s = ?
`, table, column)

	var status fleet.MDMDeliveryStatus
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &status, selectStmt, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall, hostUUID, profUUID); err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("HostMDMProfile").WithMessage("unable to match profile to host")
		}
		return "", ctxerr.Wrap(ctx, err, "get MDM profile status")
	}
	return status, nil
}

func (ds *Datastore) ResendHostMDMProfile(ctx context.Context, hostUUID string, profUUID string) error {
	table, column, err := getTableAndColumnNameForHostMDMProfileUUID(profUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting table and column")
	}

	// update the status to NULL to trigger resending on the next cron run
	updateStmt := fmt.Sprintf(`UPDATE %s SET status = NULL WHERE host_uuid = ? AND %s = ?`, table, column)

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, updateStmt, hostUUID, profUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resending host MDM profile")
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			// this should never happen, log for debugging
			level.Debug(ds.logger).Log("msg", "resend profile status not updated", "host_uuid", hostUUID, "profile_uuid", profUUID)
		}

		return nil
	})
}

func getTableAndColumnNameForHostMDMProfileUUID(profUUID string) (table, column string, err error) {
	switch {
	case strings.HasPrefix(profUUID, fleet.MDMAppleDeclarationUUIDPrefix):
		return "host_mdm_apple_declarations", "declaration_uuid", nil
	case strings.HasPrefix(profUUID, fleet.MDMAppleProfileUUIDPrefix):
		return "host_mdm_apple_profiles", "profile_uuid", nil
	case strings.HasPrefix(profUUID, fleet.MDMWindowsProfileUUIDPrefix):
		return "host_mdm_windows_profiles", "profile_uuid", nil
	default:
		return "", "", fmt.Errorf("invalid profile UUID prefix %s", profUUID)
	}
}

func (ds *Datastore) AreHostsConnectedToFleetMDM(ctx context.Context, hosts []*fleet.Host) (map[string]bool, error) {
	var (
		appleUUIDs []any
		winUUIDs   []any
	)

	res := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		switch h.Platform {
		case "darwin", "ipados", "ios":
			appleUUIDs = append(appleUUIDs, h.UUID)
		case "windows":
			winUUIDs = append(winUUIDs, h.UUID)
		}
		res[h.UUID] = false
	}

	setConnectedUUIDs := func(stmt string, uuids []any, mp map[string]bool) error {
		var res []string

		if len(uuids) > 0 {
			stmt, args, err := sqlx.In(stmt, uuids)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "building sqlx.In statement")
			}
			err = sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "retrieving hosts connected to fleet")
			}
		}

		for _, uuid := range res {
			mp[uuid] = true
		}

		return nil
	}

	// NOTE: if you change any of the conditions in this query, please
	// update the `hostMDMSelect` constant too, which has a
	// `connected_to_fleet` condition, and any relevant filters.
	const appleStmt = `
	  SELECT ne.id
	  FROM nano_enrollments ne
	    JOIN hosts h ON h.uuid = ne.id
	    JOIN host_mdm hm ON hm.host_id = h.id
	  WHERE ne.id IN (?)
	    AND ne.enabled = 1
	    AND ne.type = 'Device'
	    AND hm.enrolled = 1
	`
	if err := setConnectedUUIDs(appleStmt, appleUUIDs, res); err != nil {
		return nil, err
	}

	// NOTE: if you change any of the conditions in this query, please
	// update the `hostMDMSelect` constant too, which has a
	// `connected_to_fleet` condition, and any relevant filters.
	const winStmt = `
	  SELECT mwe.host_uuid
	  FROM mdm_windows_enrollments mwe
	    JOIN hosts h ON h.uuid = mwe.host_uuid
	    JOIN host_mdm hm ON hm.host_id = h.id
	  WHERE mwe.host_uuid IN (?)
	    AND mwe.device_state = '` + microsoft_mdm.MDMDeviceStateEnrolled + `'
	    AND hm.enrolled = 1
	`
	if err := setConnectedUUIDs(winStmt, winUUIDs, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (ds *Datastore) IsHostConnectedToFleetMDM(ctx context.Context, host *fleet.Host) (bool, error) {
	mp, err := ds.AreHostsConnectedToFleetMDM(ctx, []*fleet.Host{host})
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "finding if host is connected to Fleet MDM")
	}
	return mp[host.UUID], nil
}

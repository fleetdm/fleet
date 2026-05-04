package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) EnqueueSetupExperienceItems(ctx context.Context, hostPlatform, hostPlatformLike, hostUUID string, teamID uint) (bool, error) {
	return ds.enqueueSetupExperienceItems(ctx, hostPlatform, hostPlatformLike, hostUUID, teamID, false)
}

func (ds *Datastore) ResetSetupExperienceItemsAfterFailure(ctx context.Context, hostPlatform, hostPlatformLike, hostUUID string, teamID uint) (bool, error) {
	return ds.enqueueSetupExperienceItems(ctx, hostPlatform, hostPlatformLike, hostUUID, teamID, true)
}

func (ds *Datastore) enqueueSetupExperienceItems(ctx context.Context, hostPlatform, hostPlatformLike, hostUUID string, teamID uint, resetFailedSetupSteps bool) (bool, error) {
	// NOTE: there are 3 different "platform" values in play here: host platform,
	// host platform-like and fleet-platform-like.
	//
	// The host platform is the most specific, e.g. "darwin", "windows", "ios",
	// "ubuntu", "arch", "fedora", etc.
	//
	// Platform-like is the "generic platform" to which the specific platform belongs,
	// e.g. "debian" for "ubuntu", "rhel" for "fedora", etc. For Apple or Windows, it
	// is typically the same as platform. It may be empty in some cases (e.g. for "arch"
	// as it doesn't have a "ID_LIKE" set in /etc/os-release by default, but also "ios").
	//
	// Fleet-platform-like is the even-more-generic platform, and is implemented in
	// fleet.PlatformFromHost: "windows", "darwin", "linux", "ios", etc.
	//
	// So for many platforms, all three are the same, but for linux distros, those can be
	// 3 different values. There is no harm - at least in this function - in filling
	// hostPlatformLike to hostPlatform if it is empty (e.g. for "ios" or "arch").
	//
	// From my tests enrolling such hosts, results are:
	// - host platform - host platform like - fleet platform like -
	//   ios             <empty>              ios
	//   darwin          darwin               darwin
	//   arch            <empty>              linux
	//   ubuntu          debian               linux
	//   windows         windows              windows
	if hostPlatformLike == "" {
		hostPlatformLike = hostPlatform
	}

	if hostPlatformLike != "darwin" && hostPlatformLike != "ios" && hostPlatformLike != "ipados" {
		// Find the host with the given UUID and platform. If it's already been enrolled for > the cutoff,
		// don't enqueue any items. This handles the edge case where an enrolled host upgrades from an
		// Orbit version that didn't support setup experience to one that does.
		// See https://github.com/fleetdm/fleet/issues/35717
		stmtHost := `
		SELECT
			last_enrolled_at
		FROM
			hosts
		WHERE uuid = ? AND platform = ?
		`
		var lastEnrolledAt sql.NullTime
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &lastEnrolledAt, stmtHost, hostUUID, hostPlatform); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// This shouldn't happen but we don't check for it elsewhere,
				// so we'll log a warning and continue.
				ds.logger.WarnContext(ctx, "Host not found while enqueueing setup experience items", "host_uuid", hostUUID, "platform_like", hostPlatformLike, "platform", hostPlatform)
			} else {
				return false, ctxerr.Wrap(ctx, err, "finding host for enqueueing setup experience items")
			}
		}
		// If the host was enrolled more than 24 hours ago, don't enqueue any items.
		// Note: if the last enroll date is our "zero date" (1/1/2000), treat it as if it's never enrolled.
		if lastEnrolledAt.Valid && lastEnrolledAt.Time.Before(time.Now().Add(-24*time.Hour)) && lastEnrolledAt.Time.After(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)) {
			// On Windows, the 24h-old-host guard races with last_enrolled_at on re-Autopilot:
			// orbit calls SetupExperienceInit before the new last_enrolled_at lands, so a
			// previously-enrolled host that's mid-Autopilot-OOBE looks "old" and gets skipped
			// even though it IS in ESP and we DO want setup-experience to run. Fall back to a
			// direct check of mdm_windows_enrollments.awaiting_configuration: if the host is
			// in Pending/Active, it's actively in ESP, bypass the age guard.
			if hostPlatform == "windows" {
				var awaiting fleet.WindowsMDMAwaitingConfiguration
				stmtAwaiting := `
				SELECT awaiting_configuration
				FROM mdm_windows_enrollments
				WHERE host_uuid = ?
				ORDER BY created_at DESC, id DESC
				LIMIT 1
				`
				if err := sqlx.GetContext(ctx, ds.reader(ctx), &awaiting, stmtAwaiting, hostUUID); err != nil && !errors.Is(err, sql.ErrNoRows) {
					return false, ctxerr.Wrap(ctx, err, "checking windows awaiting_configuration for setup experience age guard")
				} else if err == nil && awaiting != fleet.WindowsMDMAwaitingConfigurationNone {
					ds.logger.DebugContext(ctx, "Windows host enrolled >24h ago but is in awaiting_configuration; running setup experience for re-Autopilot",
						"host_uuid", hostUUID, "awaiting_configuration", awaiting)
					// fall through to enqueue
				} else {
					ds.logger.DebugContext(ctx, "Host enrolled more than 24 hours ago, skipping enqueueing setup experience items", "host_uuid", hostUUID, "platform_like", hostPlatformLike, "last_enrolled_at", lastEnrolledAt.Time)
					return false, nil
				}
			} else {
				ds.logger.DebugContext(ctx, "Host enrolled more than 24 hours ago, skipping enqueueing setup experience items", "host_uuid", hostUUID, "platform_like", hostPlatformLike, "last_enrolled_at", lastEnrolledAt.Time)
				return false, nil
			}
		}
	}

	// NOTE: currently, the Android platform does not use the "enqueue setup experience items" flow as it
	// doesn't support any on-device UI (such as the screen showing setup progress) nor any
	// ordering of installs - all software to install is provided as part of the Android policy
	// when the host enrolls in Fleet.
	// See https://github.com/fleetdm/fleet/issues/33761#issuecomment-3548996114

	stmtClearSetupStatus := `
DELETE FROM setup_experience_status_results
WHERE host_uuid = ? AND %s`
	if resetFailedSetupSteps {
		stmtClearSetupStatus = fmt.Sprintf(stmtClearSetupStatus, "status != 'success'")
	} else {
		stmtClearSetupStatus = fmt.Sprintf(stmtClearSetupStatus, "TRUE")
	}

	// Build combined software query (installers + VPP apps) before the transaction.
	fleetPlatform := fleet.PlatformFromHost(hostPlatformLike)

	var softwareUnionParts []string
	var softwareArgs []any

	includeSoftwareInstallers := fleetPlatform != "ios" && fleetPlatform != "ipados"
	includeVPPApps := fleetPlatform == "darwin" || fleetPlatform == "ios" || fleetPlatform == "ipados"

	if includeSoftwareInstallers {
		installerSelect := `
SELECT
	? AS host_uuid,
	st.name AS name,
	'pending' AS status,
	si.id AS software_installer_id,
	NULL AS vpp_app_team_id,
	COALESCE(stdn.display_name, st.name) AS sort_name
FROM software_installers si
INNER JOIN software_titles st
	ON si.title_id = st.id
LEFT JOIN software_title_display_names stdn
	ON stdn.software_title_id = st.id AND stdn.team_id = ?
WHERE install_during_setup = true
AND global_or_team_id = ?
AND si.is_active = TRUE
AND (
	-- installer platform matches the host's fleet platform (darwin, linux or windows)
	si.platform = ?
	AND
	(
		-- platform is 'darwin' or 'windows', so nothing else to check.
		(si.platform = 'darwin' OR si.platform = 'windows')
		-- platform is 'linux', so we must check if the installer is compatible with the linux distribution.
		OR
		(
			-- tar.gz and sh can be installed on any Linux distribution
			(si.extension = 'tar.gz' OR si.extension = 'sh')
			OR
			(
				-- deb packages can only be installed on Debian-based hosts.
				(si.extension = 'deb' AND ? = 'debian')
				OR
				-- rpm packages can only be installed on RHEL-based hosts.
				(si.extension = 'rpm' AND ? = 'rhel')
			)
		)
	)
)
AND %s`
		if resetFailedSetupSteps {
			installerSelect = fmt.Sprintf(installerSelect, "si.id NOT IN (SELECT software_installer_id FROM setup_experience_status_results WHERE host_uuid = ? AND status = 'success' AND software_installer_id IS NOT NULL)")
		} else {
			installerSelect = fmt.Sprintf(installerSelect, "TRUE")
		}
		softwareUnionParts = append(softwareUnionParts, installerSelect)
		softwareArgs = append(softwareArgs, hostUUID, teamID, teamID, fleetPlatform, hostPlatformLike, hostPlatformLike)
		if resetFailedSetupSteps {
			softwareArgs = append(softwareArgs, hostUUID)
		}
	}

	if includeVPPApps {
		vppSelect := `
SELECT
	? AS host_uuid,
	st.name AS name,
	'pending' AS status,
	NULL AS software_installer_id,
	vat.id AS vpp_app_team_id,
	COALESCE(stdn.display_name, st.name) AS sort_name
FROM vpp_apps va
INNER JOIN vpp_apps_teams vat
	ON vat.adam_id = va.adam_id
	AND vat.platform = va.platform
INNER JOIN software_titles st
	ON va.title_id = st.id
LEFT JOIN software_title_display_names stdn
	ON stdn.software_title_id = st.id AND stdn.team_id = ?
WHERE vat.install_during_setup = true
AND vat.global_or_team_id = ?
AND va.platform = ?
AND %s`
		if resetFailedSetupSteps {
			vppSelect = fmt.Sprintf(vppSelect, "vat.id NOT IN (SELECT vpp_app_team_id FROM setup_experience_status_results WHERE host_uuid = ? AND status = 'success' AND vpp_app_team_id IS NOT NULL)")
		} else {
			vppSelect = fmt.Sprintf(vppSelect, "TRUE")
		}
		softwareUnionParts = append(softwareUnionParts, vppSelect)
		softwareArgs = append(softwareArgs, hostUUID, teamID, teamID, fleetPlatform)
		if resetFailedSetupSteps {
			softwareArgs = append(softwareArgs, hostUUID)
		}
	}

	var stmtSoftwareCombined string
	if len(softwareUnionParts) > 0 {
		stmtSoftwareCombined = fmt.Sprintf(`
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	software_installer_id,
	vpp_app_team_id
)
SELECT host_uuid, name, status, software_installer_id, vpp_app_team_id FROM (
	%s
) AS combined
ORDER BY sort_name ASC, COALESCE(software_installer_id, vpp_app_team_id, 0)`, strings.Join(softwareUnionParts, " UNION ALL "))
	}

	stmtSetupScripts := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	setup_experience_script_id
) SELECT
	?,
	name,
	'pending',
	id
FROM setup_experience_scripts
WHERE global_or_team_id = ?`

	var totalInsertions uint
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		totalInsertions = 0 // reset for each attempt

		// Clean out old statuses for the host
		if _, err := tx.ExecContext(ctx, stmtClearSetupStatus, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "removing stale setup experience entries")
		}

		// Combined software (installers + VPP apps)
		if stmtSoftwareCombined != "" {
			res, err := tx.ExecContext(ctx, stmtSoftwareCombined, softwareArgs...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "inserting setup experience software items")
			}
			inserts, err := res.RowsAffected()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "retrieving number of inserted software items")
			}
			totalInsertions += uint(inserts) // nolint: gosec
		}

		// Scripts
		if fleetPlatform == "darwin" {
			res, err := tx.ExecContext(ctx, stmtSetupScripts, hostUUID, teamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "inserting setup experience scripts")
			}
			inserts, err := res.RowsAffected()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "retrieving number of inserted setup experience scripts")
			}
			totalInsertions += uint(inserts) // nolint: gosec
		}

		// Set setup experience on Apple hosts only if they have something configured.
		if fleetPlatform == "darwin" || fleetPlatform == "ios" || fleetPlatform == "ipados" {
			if totalInsertions > 0 {
				if err := setHostAwaitingConfiguration(ctx, tx, hostUUID, true); err != nil {
					return ctxerr.Wrap(ctx, err, "setting host awaiting configuration to true")
				}
			}
		}

		return nil
	}); err != nil {
		return false, ctxerr.Wrap(ctx, err, "enqueue setup experience")
	}

	return totalInsertions > 0, nil
}

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, platform string, teamID uint, titleIDs []uint) error {
	switch platform {
	case string(fleet.MacOSPlatform),
		string(fleet.IOSPlatform),
		string(fleet.IPadOSPlatform),
		string(fleet.AndroidPlatform),
		"windows",
		"linux":
		// ok, valid platform
	default:
		return ctxerr.Errorf(ctx, "platform %q is not supported, only %q, %q, %q, %q, \"windows\", or \"linux\" platforms are supported",
			platform, fleet.MacOSPlatform, fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.AndroidPlatform)
	}

	titleIDQuestionMarks := strings.Join(slices.Repeat([]string{"?"}, len(titleIDs)), ",")

	stmtSelectInstallersIDs := fmt.Sprintf(`
SELECT
	st.id AS title_id,
	si.id,
	st.name,
	si.platform
FROM
	software_titles st
LEFT JOIN
	software_installers si
	ON st.id = si.title_id
WHERE
	si.global_or_team_id = ?
AND
	si.is_active = TRUE
AND
	st.id IN (%s)
`, titleIDQuestionMarks)

	stmtSelectVPPAppsTeamsID := fmt.Sprintf(`
SELECT
	st.id AS title_id,
	vat.id,
	st.name,
	vat.platform
FROM
	software_titles st
LEFT JOIN
	vpp_apps va
	ON st.id = va.title_id
LEFT JOIN
	vpp_apps_teams vat
	ON va.adam_id = vat.adam_id AND va.platform = vat.platform
WHERE
	vat.global_or_team_id = ?
AND
	st.id IN (%s)
AND va.platform IN ('darwin', 'ios', 'ipados', 'android')
`, titleIDQuestionMarks)

	stmtUnsetInstallers := `
UPDATE software_installers
SET install_during_setup = false
WHERE platform = ? AND global_or_team_id = ?`

	stmtUnsetVPPAppsTeams := `
UPDATE vpp_apps_teams vat
SET install_during_setup = false
WHERE platform = ? AND global_or_team_id = ?`

	stmtSetInstallers := `
UPDATE software_installers
SET install_during_setup = true
WHERE id IN (%s)`

	stmtSetVPPAppsTeams := `
UPDATE vpp_apps_teams
SET install_during_setup = true
WHERE id IN (%s)`

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var softwareIDPlatforms []idPlatformTuple
		var softwareIDs []any
		var vppIDPlatforms []idPlatformTuple
		var vppAppTeamIDs []any
		// List of title IDs that were sent but aren't in the
		// database. We add everything and then remove them
		// from the list when we validate them below
		missingTitleIDs := make(map[uint]struct{})
		// Arguments used for queries that select vpp apps/installers
		titleIDAndTeam := []any{teamID}
		for _, id := range titleIDs {
			missingTitleIDs[id] = struct{}{}
			titleIDAndTeam = append(titleIDAndTeam, id)
		}

		// Select requested software installers
		if platform != string(fleet.IOSPlatform) && platform != string(fleet.IPadOSPlatform) && platform != string(fleet.AndroidPlatform) {
			if len(titleIDs) > 0 {
				if err := sqlx.SelectContext(ctx, tx, &softwareIDPlatforms, stmtSelectInstallersIDs, titleIDAndTeam...); err != nil {
					return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
				}
			}

			// Validate software titles match the expected platform.
			for _, tuple := range softwareIDPlatforms {
				delete(missingTitleIDs, tuple.TitleID)
				if tuple.Platform != platform {
					return ctxerr.Wrap(ctx, &fleet.BadRequestError{
						Message: fmt.Sprintf("invalid platform for requested software installer: %d (%s, %s), vs. expected %s", tuple.ID, tuple.Name, tuple.Platform, platform),
					})
				}
				softwareIDs = append(softwareIDs, tuple.ID)
			}
		}

		// Select requested VPP apps
		if platform == string(fleet.MacOSPlatform) || platform == string(fleet.IOSPlatform) || platform == string(fleet.IPadOSPlatform) ||
			platform == string(fleet.AndroidPlatform) {
			if len(titleIDs) > 0 {
				if err := sqlx.SelectContext(ctx, tx, &vppIDPlatforms, stmtSelectVPPAppsTeamsID, titleIDAndTeam...); err != nil {
					return ctxerr.Wrap(ctx, err, "selecting vpp app team IDs using title IDs")
				}
			}

			// Validate VPP app platforms
			for _, tuple := range vppIDPlatforms {
				delete(missingTitleIDs, tuple.TitleID)
				if tuple.Platform != platform {
					return ctxerr.Wrap(ctx, &fleet.BadRequestError{
						Message: fmt.Sprintf("invalid platform for requested AppStoreApp title: %d (%s, %s), vs. expected %s", tuple.ID, tuple.Name, tuple.Platform, platform),
					})
				}
				vppAppTeamIDs = append(vppAppTeamIDs, tuple.ID)
			}
		}

		// If we have any missing titles, return error
		if len(missingTitleIDs) > 0 {
			var keys []string
			for k := range missingTitleIDs {
				keys = append(keys, fmt.Sprintf("%d", k))
			}
			err := &fleet.BadRequestError{
				Message: "at least one selected software title does not exist or is not available for setup experience",
			}
			return ctxerr.Wrapf(ctx, err, "title IDs not available: %s", strings.Join(keys, ","))
		}

		// Unset all installers
		if _, err := tx.ExecContext(ctx, stmtUnsetInstallers, platform, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting software installers")
		}

		// Unset all vpp apps
		if platform == string(fleet.MacOSPlatform) || platform == string(fleet.IOSPlatform) ||
			platform == string(fleet.IPadOSPlatform) || platform == string(fleet.AndroidPlatform) {
			if _, err := tx.ExecContext(ctx, stmtUnsetVPPAppsTeams, platform, teamID); err != nil {
				return ctxerr.Wrap(ctx, err, "unsetting vpp app teams")
			}
		}

		if len(softwareIDs) > 0 {
			stmtSetInstallersLoop := fmt.Sprintf(stmtSetInstallers, questionMarks(len(softwareIDs)))
			if _, err := tx.ExecContext(ctx, stmtSetInstallersLoop, softwareIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "setting software installers")
			}
		}

		if (platform == string(fleet.MacOSPlatform) || platform == string(fleet.IOSPlatform) ||
			platform == string(fleet.IPadOSPlatform) || platform == string(fleet.AndroidPlatform)) && len(vppAppTeamIDs) > 0 {
			stmtSetVPPAppsTeamsLoop := fmt.Sprintf(stmtSetVPPAppsTeams, questionMarks(len(vppAppTeamIDs)))
			if _, err := tx.ExecContext(ctx, stmtSetVPPAppsTeamsLoop, vppAppTeamIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "setting vpp app teams")
			}
		}

		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience software")
	}

	return nil
}

func (ds *Datastore) GetSetupExperienceCount(ctx context.Context, platform string, teamID *uint) (*fleet.SetupExperienceCount, error) {
	stmt := `
		SELECT
		(
			SELECT COUNT(*)
			FROM software_installers
			WHERE team_id = ?
			AND install_during_setup = 1
			AND platform = ?
		) AS installers,
		(
			SELECT COUNT(*)
			FROM vpp_apps_teams
			WHERE team_id = ?
			AND platform = ?
			AND install_during_setup = 1
		) AS vpp,
		(
			SELECT COUNT(*)
			FROM setup_experience_scripts
			WHERE team_id = ?
		) AS scripts`

	sec := &fleet.SetupExperienceCount{}
	if err := sqlx.GetContext(
		ctx, ds.reader(ctx), sec, stmt,
		teamID, platform,
		teamID, platform,
		teamID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting setup experience counts")
	}

	// Only macOS supports scripts during setup experience currently
	if platform != string(fleet.MacOSPlatform) {
		sec.Scripts = 0
	}

	return sec, nil
}

func (ds *Datastore) ListSetupExperienceSoftwareTitles(ctx context.Context, platform string, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	// I believe this can be removed, as the platforms are validated before this function
	for p := range strings.SplitSeq(strings.ReplaceAll(platform, "macos", "darwin"), ",") {
		switch p {
		case string(fleet.MacOSPlatform),
			string(fleet.IOSPlatform),
			string(fleet.IPadOSPlatform),
			string(fleet.AndroidPlatform),
			"windows",
			"linux":
			// ok, valid platform
		default:
			return nil, 0, nil, ctxerr.Errorf(ctx, "platform %q is not supported, only %q, %q, %q, %q, \"windows\", or \"linux\" platforms are supported",
				p, fleet.MacOSPlatform, fleet.IOSPlatform, fleet.AndroidPlatform, fleet.IPadOSPlatform)
		}
	}

	opts.IncludeMetadata = true
	opts.After = ""

	titles, count, meta, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		TeamID:              &teamID,
		ListOptions:         opts,
		Platform:            platform,
		AvailableForInstall: true,
		ForSetupExperience:  true,
	}, fleet.TeamFilter{
		IncludeObserver: true,
		TeamID:          &teamID,
	})
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "calling list software titles")
	}

	return titles, count, meta, nil
}

type idPlatformTuple struct {
	ID       uint   `db:"id"`
	TitleID  uint   `db:"title_id"`
	Name     string `db:"name"`
	Platform string `db:"platform"`
}

func questionMarks(number int) string {
	return strings.Join(slices.Repeat([]string{"?"}, number), ",")
}

func (ds *Datastore) ListSetupExperienceResultsByHostUUID(ctx context.Context, hostUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
	const stmt = `
SELECT
	sesr.id,
	sesr.host_uuid,
	sesr.name,
	sesr.status,
	sesr.software_installer_id,
	sesr.host_software_installs_execution_id,
	sesr.vpp_app_team_id,
	sesr.nano_command_uuid,
	sesr.setup_experience_script_id,
	sesr.script_execution_id,
	NULLIF(va.adam_id, '') AS vpp_app_adam_id,
	NULLIF(va.platform, '') AS vpp_app_platform,
	ses.script_content_id,
	COALESCE(si.title_id, COALESCE(va.title_id, NULL)) AS software_title_id,
	COALESCE(
		(SELECT source FROM software_titles WHERE id = si.title_id),
		(SELECT source FROM software_titles WHERE id = va.title_id)
	) AS source,
    CASE
        WHEN hsi.execution_status = 'failed_install' THEN
            CASE
                WHEN post_install_script_exit_code IS NOT NULL AND post_install_script_exit_code != 0 THEN COALESCE(post_install_script_output, 'Unknown error in post-install script')
                WHEN install_script_exit_code IS NOT NULL AND install_script_exit_code != 0 THEN COALESCE(install_script_output, 'Unknown error in install script')
                WHEN pre_install_query_output IS NULL OR pre_install_query_output = '' THEN 'Pre-install query failed'
                ELSE 'Installation failed'
            END
        WHEN hsr.exit_code IS NOT NULL AND hsr.exit_code != 0 THEN COALESCE(hsr.output, 'Unknown error in script')
        ELSE sesr.error
    END AS error
FROM setup_experience_status_results sesr
LEFT JOIN setup_experience_scripts ses ON ses.id = sesr.setup_experience_script_id
LEFT JOIN software_installers si ON si.id = sesr.software_installer_id AND si.is_active = TRUE
LEFT JOIN host_software_installs hsi ON hsi.execution_id = sesr.host_software_installs_execution_id
LEFT JOIN host_script_results hsr ON hsr.execution_id = sesr.script_execution_id
LEFT JOIN vpp_apps_teams vat ON vat.id = sesr.vpp_app_team_id
LEFT JOIN vpp_apps va ON vat.adam_id = va.adam_id AND vat.platform = va.platform
WHERE host_uuid = ?
ORDER BY sesr.id
	`
	var results []*fleet.SetupExperienceStatusResult
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select setup experience status results by host uuid")
	}

	titleIDs := make([]uint, 0, len(results))
	byTitleID := make(map[uint]*fleet.SetupExperienceStatusResult, len(results))
	for _, res := range results {
		if res.SoftwareTitleID != nil {
			titleIDs = append(titleIDs, *res.SoftwareTitleID)
			byTitleID[*res.SoftwareTitleID] = res
		}
	}

	// load custom display name and custom icon for the software installers, if any
	if len(titleIDs) > 0 {
		icons, err := ds.GetSoftwareIconsByTeamAndTitleIds(ctx, teamID, titleIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get software icons by team and title IDs")
		}

		displayNames, err := ds.getDisplayNamesByTeamAndTitleIds(ctx, teamID, titleIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get software display names by team and title IDs")
		}

		for titleID, icon := range icons {
			if res := byTitleID[titleID]; res != nil {
				res.IconURL = icon.IconUrl()
			}
		}

		for titleID, name := range displayNames {
			if res := byTitleID[titleID]; res != nil {
				res.DisplayName = name
			}
		}
	}

	return results, nil
}

func (ds *Datastore) UpdateSetupExperienceStatusResult(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
	const stmt = `
UPDATE setup_experience_status_results
SET
	host_uuid = ?,
	name = ?,
	status = ?,
	software_installer_id = ?,
	host_software_installs_execution_id = ?,
	vpp_app_team_id = ?,
	nano_command_uuid = ?,
	setup_experience_script_id = ?,
	script_execution_id = ?,
	error = LEFT(?, 255)
WHERE id = ?
`
	if err := status.IsValid(); err != nil {
		return ctxerr.Wrap(ctx, err, "invalid status update")
	}

	if _, err := ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		status.HostUUID,
		status.Name,
		status.Status,
		status.SoftwareInstallerID,
		status.HostSoftwareInstallsExecutionID,
		status.VPPAppTeamID,
		status.NanoCommandUUID,
		status.SetupExperienceScriptID,
		status.ScriptExecutionID,
		status.Error,
		status.ID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "updating setup experience status result")
	}

	return nil
}

func (ds *Datastore) GetSetupExperienceScript(ctx context.Context, teamID *uint) (*fleet.Script, error) {
	return ds.getSetupExperienceScript(ctx, ds.reader(ctx), teamID)
}

func (ds *Datastore) getSetupExperienceScript(ctx context.Context, q sqlx.QueryerContext, teamID *uint) (*fleet.Script, error) {
	query := `
SELECT
  id,
  team_id,
  name,
  script_content_id,
  created_at,
  updated_at
FROM
  setup_experience_scripts
WHERE
  global_or_team_id = ?
`
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var script fleet.Script
	if err := sqlx.GetContext(ctx, q, &script, query, globalOrTeamID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SetupExperienceScript"), "get setup experience script")
		}
		return nil, ctxerr.Wrap(ctx, err, "get setup experience script")
	}

	return &script, nil
}

func (ds *Datastore) GetSetupExperienceScriptByID(ctx context.Context, scriptID uint) (*fleet.Script, error) {
	query := `
SELECT
  id,
  team_id,
  name,
  script_content_id,
  created_at,
  updated_at
FROM
  setup_experience_scripts
WHERE
  id = ?
`

	var script fleet.Script
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &script, query, scriptID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SetupExperienceScript"), "get setup experience script by id")
		}
		return nil, ctxerr.Wrap(ctx, err, "get setup experience script by id")
	}

	return &script, nil
}

func (ds *Datastore) SetSetupExperienceScript(ctx context.Context, script *fleet.Script) error {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		// first insert script contents
		scRes, err := insertScriptContents(ctx, tx, script.ScriptContents)
		if err != nil {
			return err
		}
		id, _ := scRes.LastInsertId()

		// This clause allows for PUT semantics. The basic idea is:
		// - no existing setup script -> go through the usual insert logic
		// - existing setup script with different content -> delete(with all side effects) and re-insert
		// - existing setup script with same content -> no-op
		gotSetupExperienceScript, err := ds.getSetupExperienceScript(ctx, tx, script.TeamID)
		if err != nil && !fleet.IsNotFound(err) {
			return err
		}
		// We will fall through on a notFound err - nothing to do here
		if err == nil {
			if gotSetupExperienceScript.ScriptContentID != uint(id) { // nolint:gosec // dismiss G115 - low risk here
				err = ds.deleteSetupExperienceScript(ctx, tx, script.TeamID)
				if err != nil {
					return err
				}
			} else {
				// no change
				return nil
			}
		}

		// then create the script entity
		_, err = insertSetupExperienceScript(ctx, tx, script, uint(id)) // nolint: gosec
		return err
	})

	return err
}

func insertSetupExperienceScript(ctx context.Context, tx sqlx.ExtContext, script *fleet.Script, scriptContentsID uint) (sql.Result, error) {
	const insertStmt = `
INSERT INTO
  setup_experience_scripts (
    team_id, global_or_team_id, name, script_content_id
  )
VALUES
  (?, ?, ?, ?)
`
	var globalOrTeamID uint
	if script.TeamID != nil {
		globalOrTeamID = *script.TeamID
	}
	res, err := tx.ExecContext(ctx, insertStmt,
		script.TeamID, globalOrTeamID, script.Name, scriptContentsID)
	if err != nil {

		if IsDuplicate(err) {
			// already exists for this team/no team
			err = &existsError{ResourceType: "SetupExperienceScript", TeamID: &globalOrTeamID}
		} else if isChildForeignKeyError(err) {
			// team does not exist
			err = foreignKey("setup_experience_scripts", fmt.Sprintf("team_id=%v", script.TeamID))
		}
		return nil, ctxerr.Wrap(ctx, err, "insert setup experience script")
	}

	return res, nil
}

func (ds *Datastore) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	return ds.deleteSetupExperienceScript(ctx, ds.writer(ctx), teamID)
}

func (ds *Datastore) deleteSetupExperienceScript(ctx context.Context, tx sqlx.ExtContext, teamID *uint) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	_, err := tx.ExecContext(ctx, `DELETE FROM setup_experience_scripts WHERE global_or_team_id = ?`, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: CleanupUnusedScriptContents is responsible for removing any orphaned script_contents
	// for setup experience scripts.

	return nil
}

func (ds *Datastore) SetHostAwaitingConfiguration(ctx context.Context, hostUUID string, awaitingConfiguration bool) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return setHostAwaitingConfiguration(ctx, tx, hostUUID, awaitingConfiguration)
	})
}

func setHostAwaitingConfiguration(ctx context.Context, tx sqlx.ExtContext, hostUUID string, awaitingConfiguration bool) error {
	const stmt = `
INSERT INTO host_mdm_apple_awaiting_configuration (host_uuid, awaiting_configuration)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
	awaiting_configuration = VALUES(awaiting_configuration)
	`

	_, err := tx.ExecContext(ctx, stmt, hostUUID, awaitingConfiguration)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting host awaiting configuration")
	}

	return nil
}

func (ds *Datastore) GetHostAwaitingConfiguration(ctx context.Context, hostUUID string) (bool, error) {
	const stmt = `
SELECT
	awaiting_configuration
FROM host_mdm_apple_awaiting_configuration
WHERE host_uuid = ?
	`
	var awaitingConfiguration bool

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &awaitingConfiguration, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, notFound("HostAwaitingConfiguration")
		}

		return false, ctxerr.Wrap(ctx, err, "getting host awaiting configuration")
	}

	return awaitingConfiguration, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceVPPStatus(ctx context.Context, hostUUID string, nanoCommandUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	stmt := `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND nano_command_uuid = ? AND status NOT IN (?, ?, ?)`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, status, hostUUID, nanoCommandUUID, fleet.SetupExperienceStatusSuccess, fleet.SetupExperienceStatusFailure, fleet.SetupExperienceStatusCancelled)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	stmt := `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND host_software_installs_execution_id = ? AND status NOT IN (?, ?, ?)`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, status, hostUUID, executionID, fleet.SetupExperienceStatusSuccess, fleet.SetupExperienceStatusFailure, fleet.SetupExperienceStatusCancelled)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceScriptStatus(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	stmt := `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND script_execution_id = ? AND status NOT IN (?, ?, ?)`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, status, hostUUID, executionID, fleet.SetupExperienceStatusSuccess, fleet.SetupExperienceStatusFailure, fleet.SetupExperienceStatusCancelled)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (ds *Datastore) CancelPendingSetupExperienceSteps(ctx context.Context, hostUUID string) error {
	cancelStmt := "UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND status NOT IN (?, ?)"
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, cancelStmt, fleet.SetupExperienceStatusCancelled, hostUUID, fleet.SetupExperienceStatusSuccess, fleet.SetupExperienceStatusFailure)
		return err
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cancelling pending setup experience steps")
	}
	return nil
}

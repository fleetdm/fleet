package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) insertInHouseApp(ctx context.Context, payload *fleet.InHouseAppPayload) (uint, uint, error) {
	selectStmt := `SELECT COUNT(id) FROM in_house_apps WHERE global_or_team_id = ? AND (bundle_identifier = ? OR filename = ?)`

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	titleName, _ := strings.CutSuffix(payload.Filename, ".ipa")
	titleIDipad, err := ds.getOrGenerateInHouseAppTitleID(ctx, titleName, payload.BundleID, "ipados_apps")
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}
	titleIDios, err := ds.getOrGenerateInHouseAppTitleID(ctx, titleName, payload.BundleID, "ios_apps")
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}

	var installerID uint
	var count uint
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		row := tx.QueryRowxContext(ctx, selectStmt, globalOrTeamID, payload.BundleID, payload.Filename)
		if err := row.Scan(&count); err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}
		if count > 0 {
			// ios or ipados version of this installer exists
			err = alreadyExists("insertInHouseApp", payload.Filename)
		}

		argsIos := []any{
			tid,
			globalOrTeamID,
			payload.Filename,
			payload.StorageID,
			payload.Version,
			payload.BundleID,
			titleIDios,
			"ios",
			payload.SelfService,
		}
		argsIpad := []any{
			tid,
			globalOrTeamID,
			payload.Filename,
			payload.StorageID,
			payload.Version,
			payload.BundleID,
			titleIDipad,
			"ipados",
			payload.SelfService,
		}

		_, err := ds.insertInHouseAppDB(ctx, tx, payload, argsIpad)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		installerID, err = ds.insertInHouseAppDB(ctx, tx, payload, argsIos)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		return nil
	})

	return installerID, titleIDios, ctxerr.Wrap(ctx, err, "insertInHouseApp")
}

func (ds *Datastore) getOrGenerateInHouseAppTitleID(ctx context.Context, name string, bundleID string, source string) (uint, error) {
	selectStmt := `SELECT id FROM software_titles WHERE (bundle_identifier = ? AND source = ?) OR (name = ? AND source = ?)`
	selectArgs := []any{bundleID, source, name, source}
	insertStmt := `INSERT INTO software_titles (name, source, bundle_identifier, extension_for) VALUES (?, ?, ?, '')`
	insertArgs := []any{name, source, bundleID}

	titleID, err := ds.optimisticGetOrInsert(ctx,
		&parameterizedStmt{
			Statement: selectStmt,
			Args:      selectArgs,
		},
		&parameterizedStmt{
			Statement: insertStmt,
			Args:      insertArgs,
		},
	)
	if err != nil {
		return 0, err
	}
	return titleID, nil
}

func (ds *Datastore) insertInHouseAppDB(ctx context.Context, tx sqlx.ExtContext, payload *fleet.InHouseAppPayload, args []any) (uint, error) {
	stmt := `
	INSERT INTO in_house_apps (
		team_id,
		global_or_team_id,
		filename,
		storage_id,
		version,
		bundle_identifier,
		title_id,
		platform,
		self_service
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		if IsDuplicate(err) {
			err = alreadyExists("insertInHouseAppDB", payload.Filename)
		}
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseAppDB")
	}
	id64, err := res.LastInsertId()
	installerID := uint(id64) //nolint:gosec // dismiss G115
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseAppDB")
	}

	if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels, softwareTypeInHouseApp); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseAppDB")
	}
	return installerID, nil
}

// hihsiAlias is the table alias to use as prefix for the
// host_in_house_software_installs column names, no prefix used if empty.
// ncrAlias is the table alias to use as prefix for the nano_command_results
// column names, no prefix used if empty.
// colAlias is the name to be assigned to the computed status column, pass
// empty to have the value only, no column alias set.
func inHouseAppHostStatusNamedQuery(hihsiAlias, ncrAlias, colAlias string) string {
	if hihsiAlias != "" {
		hihsiAlias += "."
	}
	if ncrAlias != "" {
		ncrAlias += "."
	}
	if colAlias != "" {
		colAlias = " AS " + colAlias
	}

	return fmt.Sprintf(`
	CASE
		WHEN %sverification_at IS NOT NULL THEN
			:software_status_installed
		WHEN %sverification_failed_at IS NOT NULL THEN
			:software_status_failed
		WHEN %sstatus = :mdm_status_error OR %sstatus = :mdm_status_format_error THEN
			:software_status_failed
		ELSE
			:software_status_pending
	END %s
	`, hihsiAlias, hihsiAlias, ncrAlias, ncrAlias, colAlias)
}

func (ds *Datastore) GetInHouseAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
  iha.id,
  iha.team_id,
  iha.title_id,
  iha.filename,
  iha.platform,
  iha.storage_id,
  iha.version,
  iha.created_at AS uploaded_at,
  st.bundle_identifier AS bundle_identifier,
  COALESCE(st.name, '') AS software_title,
	iha.self_service
FROM
  in_house_apps iha
  JOIN software_titles st ON st.id = iha.title_id
WHERE
  iha.title_id = ? AND iha.global_or_team_id = ?`

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var dest fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, titleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("InHouseApp"), "get in house app metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get in house app metadata")
	}
	dest.Extension = "ipa"

	labels, err := ds.getSoftwareInstallerLabels(ctx, dest.InstallerID, softwareTypeInHouseApp)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get in house app labels")
	}
	var exclAny, inclAny []fleet.SoftwareScopeLabel
	for _, l := range labels {
		if l.Exclude {
			exclAny = append(exclAny, l)
		} else {
			inclAny = append(inclAny, l)
		}
	}

	if len(inclAny) > 0 && len(exclAny) > 0 {
		level.Warn(ds.logger).Log("msg", "in house app has both include and exclude labels", "installer_id", dest.InstallerID, "include", fmt.Sprintf("%v", inclAny), "exclude", fmt.Sprintf("%v", exclAny))
	}
	dest.LabelsExcludeAny = exclAny
	dest.LabelsIncludeAny = inclAny

	return &dest, nil
}

func (ds *Datastore) SaveInHouseAppUpdates(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) error {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `UPDATE in_house_apps SET
			storage_id = ?,
			filename = ?,
			version = ?,
			-- keep current value if provided arg is nil
			self_service = COALESCE(?, self_service)
	 WHERE id = ?`

		args := []any{
			payload.StorageID,
			payload.Filename,
			payload.Version,
			payload.SelfService,
			payload.InstallerID,
		}

		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update in house app")
		}

		if payload.ValidatedLabels != nil {
			if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, payload.InstallerID, *payload.ValidatedLabels, softwareTypeInHouseApp); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert in house app labels")
			}
		}

		return nil
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update in house app")
	}

	return nil
}

func (ds *Datastore) DeleteInHouseApp(ctx context.Context, id uint) error {
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		err := ds.RemovePendingInHouseAppInstalls(ctx, id)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "delete in house app: remove pending in house app installs")
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM in_house_apps WHERE id = ?`, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete in house app")
		}
		return err
	})
	return err
}

func (ds *Datastore) RemovePendingInHouseAppInstalls(ctx context.Context, inHouseAppID uint) error {
	type ipaInstall struct {
		HostID      uint   `db:"host_id"`
		ExecutionID string `db:"command_uuid"`
	}
	var installs []ipaInstall
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &installs, `
		SELECT
			host_id,
			command_uuid
		FROM
			host_in_house_software_installs
		WHERE
			in_house_app_id = ? AND
			canceled = 0 AND
			verification_at IS NULL AND
			verification_failed_at IS NULL
`, inHouseAppID)
	if err != nil {
		return err
	}

	for _, in := range installs {
		_, err := ds.CancelHostUpcomingActivity(ctx, in.HostID, in.ExecutionID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *Datastore) GetSummaryHostInHouseAppInstalls(ctx context.Context, teamID *uint, inHouseAppID uint) (*fleet.VPPAppStatusSummary, error) {
	var dest fleet.VPPAppStatusSummary // Using the vpp struct since it is more appropriate for ipa
	stmt := `
WITH
-- select most recent upcoming activities for each host
upcoming AS (
	SELECT
		ua.host_id,
		:software_status_pending AS status
	FROM
		upcoming_activities ua
		JOIN in_house_app_upcoming_activities ihaua ON ua.id = ihaua.upcoming_activity_id
		JOIN hosts h ON host_id = h.id
		LEFT JOIN (
			upcoming_activities ua2
			INNER JOIN in_house_app_upcoming_activities ihaua2
				ON ua2.id = ihaua2.upcoming_activity_id
		) ON ua.host_id = ua2.host_id AND
			ihaua.in_house_app_id = ihaua2.in_house_app_id AND
			ua.activity_type = ua2.activity_type AND
			(ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
	WHERE
		ua.activity_type = 'in_house_app_install'
		AND ua2.id IS NULL
		AND ihaua.in_house_app_id = :in_house_app_id
		AND (h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0))
),

-- select most recent past activities for each host
past AS (
	SELECT
		hihsi.host_id,
		CASE
			WHEN ncr.status = :mdm_status_acknowledged THEN
				:software_status_installed
			WHEN ncr.status = :mdm_status_error OR ncr.status = :mdm_status_format_error THEN
				:software_status_failed
			ELSE
				NULL -- either pending or not installed
		END AS status
	FROM
		host_in_house_software_installs hihsi
		JOIN hosts h ON host_id = h.id
		JOIN nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hihsi.command_uuid
		LEFT JOIN host_in_house_software_installs hihsi2
			ON hihsi.host_id = hihsi2.host_id AND
				 hihsi.in_house_app_id = hihsi2.in_house_app_id AND
				 hihsi2.removed = 0 AND
				 hihsi2.canceled = 0 AND
				 (hihsi.created_at < hihsi2.created_at OR (hihsi.created_at = hihsi2.created_at AND hihsi.id < hihsi2.id))
	WHERE
		hihsi2.id IS NULL
		AND hihsi.in_house_app_id = :in_house_app_id
		AND (h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0))
		AND hihsi.host_id NOT IN (SELECT host_id FROM upcoming) -- antijoin to exclude hosts with upcoming activities
		AND hihsi.removed = 0
		AND hihsi.canceled = 0
)

-- count each status
SELECT
	COALESCE(SUM( IF(status = :software_status_pending, 1, 0)), 0) AS pending,
	COALESCE(SUM( IF(status = :software_status_failed, 1, 0)), 0) AS failed,
	COALESCE(SUM( IF(status = :software_status_installed, 1, 0)), 0) AS installed
FROM (

-- union most recent past and upcoming activities after joining to get statuses for most recent activities
SELECT
	past.host_id,
	past.status
FROM past
UNION
SELECT
	upcoming.host_id,
	upcoming.status
FROM upcoming
) t`

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	query, args, err := sqlx.Named(stmt, map[string]any{
		"in_house_app_id":           inHouseAppID,
		"team_id":                   tmID,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
		"software_status_pending":   fleet.SoftwarePending,
		"software_status_failed":    fleet.SoftwareFailed,
		"software_status_installed": fleet.SoftwareInstalled,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host in house app installs: named query")
	}

	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host in house install status")
	}
	return &dest, nil
}

func (ds *Datastore) IsInHouseAppLabelScoped(ctx context.Context, inHouseAppID, hostID uint) (bool, error) {
	return ds.isSoftwareLabelScoped(ctx, inHouseAppID, hostID, softwareTypeInHouseApp)
}

func (ds *Datastore) InsertHostInHouseAppInstall(ctx context.Context, hostID uint, inHouseAppID, softwareTitleID uint, commandUUID string, opts fleet.HostSoftwareInstallOptions) error {
	const (
		insertUAStmt = `
INSERT INTO upcoming_activities
		(host_id, priority, user_id, fleet_initiated, activity_type, execution_id, payload)
VALUES
		(?, ?, ?, ?, 'in_house_app_install', ?,
			JSON_OBJECT(
				'self_service', ?,
				'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = ?)
			)
		)`

		insertIHAUAStmt = `
INSERT INTO in_house_app_upcoming_activities
		(upcoming_activity_id, in_house_app_id, software_title_id)
VALUES
		(?, ?, ?)`

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return notFound("Host").WithID(hostID)
		}

		return ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertUAStmt,
			hostID,
			opts.Priority(),
			userID,
			opts.IsFleetInitiated(),
			commandUUID,
			opts.SelfService,
			userID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert in house app install request")
		}

		activityID, _ := res.LastInsertId()
		_, err = tx.ExecContext(ctx, insertIHAUAStmt,
			activityID,
			inHouseAppID,
			softwareTitleID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert in house app install request join table")
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity")
		}
		return nil
	})
	return err
}

func (ds *Datastore) SetInHouseAppInstallAsVerified(ctx context.Context, hostID uint, installUUID, verificationUUID string) error {
	stmt := `
UPDATE host_in_house_software_installs
SET verification_at = CURRENT_TIMESTAMP(6),
verification_command_uuid = ?
WHERE command_uuid = ?
	`

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, stmt, verificationUUID, installUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "set in house app install as verified")
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, installUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity from in house app install verify")
		}

		return nil
	})
}

func (ds *Datastore) SetInHouseAppInstallAsFailed(ctx context.Context, hostID uint, installUUID, verificationUUID string) error {
	stmt := `
UPDATE host_in_house_software_installs
SET verification_failed_at = CURRENT_TIMESTAMP(6),
verification_command_uuid = ?
WHERE command_uuid = ?
	`

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, stmt, verificationUUID, installUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "set in house app install as failed")
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, installUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity from in house app install failed")
		}

		return nil
	})
}

func (ds *Datastore) ReplaceInHouseAppInstallVerificationUUID(ctx context.Context, oldVerifyUUID, verifyCommandUUID string) error {
	stmt := `
UPDATE host_in_house_software_installs
SET verification_command_uuid = ?
WHERE verification_command_uuid = ?
	`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, verifyCommandUUID, oldVerifyUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "update in-house app install verification command")
	}

	return nil
}

func (ds *Datastore) GetUnverifiedInHouseAppInstallsForHost(ctx context.Context, hostUUID string) ([]*fleet.HostVPPSoftwareInstall, error) {
	stmt := `
SELECT
		hihsi.host_id AS host_id,
		hihsi.command_uuid AS command_uuid,
		ncr.updated_at AS ack_at,
		ncr.status AS install_command_status,
		iha.bundle_identifier AS bundle_identifier
FROM nano_command_results ncr
JOIN host_in_house_software_installs hihsi ON hihsi.command_uuid = ncr.command_uuid
JOIN in_house_apps iha ON iha.id = hihsi.in_house_app_id AND iha.platform = hihsi.platform
WHERE ncr.id = ?
AND ncr.status = 'Acknowledged'
AND hihsi.verification_at IS NULL
AND hihsi.verification_failed_at IS NULL
		`

	var result []*fleet.HostVPPSoftwareInstall
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get unverified in-house app installs for host")
	}

	return result, nil
}

func (ds *Datastore) GetPastActivityDataForInHouseAppInstall(ctx context.Context, commandResults *mdm.CommandResults) (*fleet.User, *fleet.ActivityTypeInstalledSoftware, error) {
	if commandResults == nil {
		return nil, nil, nil
	}

	stmt := `
SELECT
	u.name AS user_name,
	u.id AS user_id,
	u.email as user_email,
	hihsi.host_id AS host_id,
	hdn.display_name AS host_display_name,
	st.name AS software_title,
	hihsi.command_uuid AS command_uuid,
	hihsi.self_service AS self_service
FROM
	host_in_house_software_installs hihsi
	LEFT OUTER JOIN users u ON hihsi.user_id = u.id
	LEFT OUTER JOIN host_display_names hdn ON hdn.host_id = hihsi.host_id
	LEFT OUTER JOIN in_house_apps iha ON hihsi.in_house_app_id = iha.id
	LEFT OUTER JOIN software_titles st ON st.id = iha.title_id
WHERE
	hihsi.command_uuid = :command_uuid AND
	hihsi.canceled = 0
	`

	type result struct {
		HostID          uint    `db:"host_id"`
		HostDisplayName string  `db:"host_display_name"`
		SoftwareTitle   string  `db:"software_title"`
		CommandUUID     string  `db:"command_uuid"`
		UserName        *string `db:"user_name"`
		UserID          *uint   `db:"user_id"`
		UserEmail       *string `db:"user_email"`
		SelfService     bool    `db:"self_service"`
	}

	listStmt, args, err := sqlx.Named(stmt, map[string]any{
		"command_uuid":              commandResults.CommandUUID,
		"software_status_failed":    string(fleet.SoftwareInstallFailed),
		"software_status_installed": string(fleet.SoftwareInstalled),
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build list query from named args")
	}

	var res result
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, listStmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, notFound("install_command")
		}

		return nil, nil, ctxerr.Wrap(ctx, err, "select past activity data for in-house app install")
	}

	var user *fleet.User
	if res.UserID != nil {
		user = &fleet.User{
			ID:    *res.UserID,
			Name:  *res.UserName,
			Email: *res.UserEmail,
		}
	}

	var status string
	switch commandResults.Status {
	case fleet.MDMAppleStatusAcknowledged:
		status = string(fleet.SoftwareInstalled)
	case fleet.MDMAppleStatusCommandFormatError, fleet.MDMAppleStatusError:
		status = string(fleet.SoftwareInstallFailed)
	default:
		// This case shouldn't happen (we should only be doing this check if the command is in a
		// "terminal" state, but adding it so we have a default
		status = string(fleet.SoftwareInstallPending)
	}

	act := &fleet.ActivityTypeInstalledSoftware{
		HostID:          res.HostID,
		HostDisplayName: res.HostDisplayName,
		SoftwareTitle:   res.SoftwareTitle,
		CommandUUID:     res.CommandUUID,
		Status:          status,
		SelfService:     res.SelfService,
	}

	return user, act, nil
}

func (ds *Datastore) BatchSetInHouseAppsInstallers(ctx context.Context, tmID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
	const upsertSoftwareTitles = `
INSERT INTO software_titles
  (name, source, extension_for, bundle_identifier)
VALUES
  %s
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  source = VALUES(source),
  extension_for = VALUES(extension_for),
  bundle_identifier = VALUES(bundle_identifier)
`

	const loadSoftwareTitles = `
SELECT
  id
FROM
  software_titles
WHERE (unique_identifier, source, extension_for) IN (%s)
`

	const cancelAllPendingInHouseInstalls = `
UPDATE
	host_in_house_software_installs
SET
	canceled = 1
WHERE
	verification_at IS NULL AND
	verification_failed_at IS NULL AND
	in_house_app_id IN (
		SELECT id FROM in_house_apps WHERE global_or_team_id = ?
	)
`

	const cancelAllPendingInHouseNanoCmds = `
UPDATE
	nano_enrollment_queue
SET
	active = 0
WHERE
	command_uuid IN (
		SELECT command_uuid
		FROM host_in_house_software_installs hihsi
			INNER JOIN in_house_apps iha ON hihsi.in_house_app_id = iha.id
		WHERE
			hihsi.verification_at IS NULL AND
			hihsi.verification_failed_at IS NULL AND
			iha.global_or_team_id = ?
	)
`
	const loadAffectedHostsPendingInHouseInstallsUA = `
		SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
		INNER JOIN in_house_app_upcoming_activities ihua
			ON ua.id = ihua.upcoming_activity_id
		WHERE
			ua.activity_type = 'in_house_app_install' AND
			ua.activated_at IS NOT NULL AND
			ihua.in_house_app_id IN (
				SELECT id FROM in_house_apps WHERE global_or_team_id = ?
		)
`

	const deleteAllPendingInHouseInstallsUA = `
		DELETE FROM upcoming_activities
		USING upcoming_activities
		INNER JOIN in_house_app_upcoming_activities ihua
			ON upcoming_activities.id = ihua.upcoming_activity_id
		WHERE
			activity_type = 'in_house_app_install' AND
			ihua.in_house_app_id IN (
				SELECT id FROM in_house_apps WHERE global_or_team_id = ?
		)
`
	const markAllInHouseInstallsAsRemoved = `
		UPDATE host_in_house_software_installs SET removed = TRUE
		WHERE in_house_app_id IN (
			SELECT id FROM in_house_apps WHERE global_or_team_id = ?
		)
`

	const deleteAllInHouseInstallersInTeam = `
DELETE FROM
	in_house_apps
WHERE
  global_or_team_id = ?
`

	const cancelPendingInHouseInstallsNotInList = `
UPDATE
	host_in_house_software_installs
SET
	canceled = 1
WHERE
	verification_at IS NULL AND
	verification_failed_at IS NULL AND
	in_house_app_id IN (
		SELECT id FROM in_house_apps WHERE global_or_team_id = ? AND title_id NOT IN (?)
	)
`

	const cancelPendingInHouseNanoCmdsNotInList = `
UPDATE
	nano_enrollment_queue
SET
	active = 0
WHERE
	command_uuid IN (
		SELECT command_uuid
		FROM host_in_house_software_installs hihsi
			INNER JOIN in_house_apps iha ON hihsi.in_house_app_id = iha.id
		WHERE
			hihsi.verification_at IS NULL AND
			hihsi.verification_failed_at IS NULL AND
			iha.global_or_team_id = ? AND
			iha.title_id NOT IN (?)
	)
`

	const loadAffectedHostsPendingInHouseInstallsNotInListUA = `
		SELECT
			DISTINCT host_id
		FROM
			upcoming_activities ua
		INNER JOIN in_house_app_upcoming_activities ihua
			ON ua.id = ihua.upcoming_activity_id
		WHERE
			ua.activity_type = 'in_house_app_install' AND
			ua.activated_at IS NOT NULL AND
			ihua.in_house_app_id IN (
				SELECT id FROM in_house_apps WHERE global_or_team_id = ? AND title_id NOT IN (?)
		)
`

	const deletePendingInHouseInstallsNotInListUA = `
		DELETE FROM upcoming_activities
		USING upcoming_activities
		INNER JOIN in_house_app_upcoming_activities ihua
			ON upcoming_activities.id = ihua.upcoming_activity_id
		WHERE
			activity_type = 'in_house_app_install' AND
			ihua.in_house_app_id IN (
				SELECT id FROM in_house_apps WHERE global_or_team_id = ? AND title_id NOT IN (?)
		)
`

	const markInHouseInstallsNotInListAsRemoved = `
		UPDATE host_in_house_software_installs SET removed = TRUE
		WHERE in_house_app_id IN (
			SELECT id FROM in_house_apps WHERE global_or_team_id = ? AND title_id NOT IN (?)
		)
`

	const deleteInHouseInstallersNotInList = `
DELETE FROM
	in_house_apps
WHERE
  global_or_team_id = ? AND
	title_id NOT IN (?)
`

	const checkExistingInstaller = `
SELECT
	id,
	storage_id != ? is_package_modified
FROM
	in_house_apps
WHERE
	global_or_team_id = ?	AND
	title_id IN (SELECT id FROM software_titles WHERE unique_identifier = ? AND source = ? AND extension_for = '')
`

	const insertNewOrEditedInstaller = `
INSERT INTO software_installers (
	title_id,
	team_id,
	global_or_team_id,
	filename,
	version,
	storage_id,
	platform,
	bundle_identifier,
	self_service,
	url
) VALUES (
  (SELECT id FROM software_titles WHERE unique_identifier = ? AND source = ? AND extension_for = ''),
  ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON DUPLICATE KEY UPDATE
  filename = VALUES(filename),
  version = VALUES(version),
  storage_id = VALUES(storage_id),
  platform = VALUES(platform),
  bundle_identifier = VALUES(bundle_identifier),
  self_service = VALUES(self_service),
  url = VALUES(url)
`

	const loadInHouseInstallerID = `
SELECT
	id
FROM
	in_house_apps
WHERE
	-- this is guaranteed to select a single in-house installer, due to unique index
	global_or_team_id = ?	AND
	filename = ? AND
	platform = ?
`

	const deleteInHouseLabelsNotInList = `
DELETE FROM
	in_house_app_labels
WHERE
	in_house_app_id = ? AND
	label_id NOT IN (?)
`

	const deleteAllInHouseLabels = `
DELETE FROM
	in_house_app_labels
WHERE
	in_house_app_id = ?
`

	const upsertInHouseLabels = `
INSERT INTO
	in_house_app_labels (
		in_house_app_id,
		label_id,
		exclude
	)
VALUES
	%s
ON DUPLICATE KEY UPDATE
	exclude = VALUES(exclude)
`

	const loadExistingInHouseLabels = `
SELECT
	label_id,
	exclude
FROM
	in_house_app_labels
WHERE
	in_house_app_id = ?
`

	// use a team id of 0 if no-team
	var globalOrTeamID uint
	if tmID != nil {
		globalOrTeamID = *tmID
	}

	// NOTE: at the time of implementation, in-house apps do not support install
	// during setup, automatic install (via policies), categories, and
	// uninstalls, so the related validations and updates that are done in
	// BatchSetSoftwareInstallers are removed here.

	var activateAffectedHostIDs []uint

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// if no installers are provided, just delete whatever was in the table
		if len(installers) == 0 {
			if _, err := tx.ExecContext(ctx, cancelAllPendingInHouseInstalls, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "cancel all pending host in-house install records")
			}
			if _, err := tx.ExecContext(ctx, cancelAllPendingInHouseNanoCmds, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "cancel all pending in-house nano commands")
			}

			var affectedHostIDs []uint
			if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs,
				loadAffectedHostsPendingInHouseInstallsUA, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "load affected hosts for upcoming in-house installs")
			}
			activateAffectedHostIDs = affectedHostIDs

			if _, err := tx.ExecContext(ctx, deleteAllPendingInHouseInstallsUA, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all upcoming pending in-house install records")
			}

			if _, err := tx.ExecContext(ctx, markAllInHouseInstallsAsRemoved, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "mark all host in-house installs as removed")
			}

			if _, err := tx.ExecContext(ctx, deleteAllInHouseInstallersInTeam, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete obsolete in-house installers")
			}

			return nil
		}

		var args []any
		for _, installer := range installers {
			args = append(
				args,
				strings.TrimSuffix(installer.Filename, ".ipa"),
				installer.Source,
				"",
				func() *string {
					if strings.TrimSpace(installer.BundleIdentifier) != "" {
						return &installer.BundleIdentifier
					}
					return nil
				}(),
			)
		}

		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?),", len(installers)), ",")
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(upsertSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert new/edited software titles")
		}

		var titleIDs []uint
		args = []any{}
		for _, installer := range installers {
			args = append(
				args,
				BundleIdentifierOrName(installer.BundleIdentifier, strings.TrimSuffix(installer.Filename, ".ipa")),
				installer.Source,
				"",
			)
		}
		values = strings.TrimSuffix(strings.Repeat("(?,?,?),", len(installers)), ",")

		if err := sqlx.SelectContext(ctx, tx, &titleIDs, fmt.Sprintf(loadSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "load existing titles")
		}

		stmt, args, err := sqlx.In(cancelPendingInHouseInstallsNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to cancel pending in-house installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "cancel obsolete pending host in-house install records")
		}
		stmt, args, err = sqlx.In(cancelPendingInHouseNanoCmdsNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to cancel pending in-house nano commands")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "cancel obsolete pending host in-house install nano commands")
		}

		stmt, args, err = sqlx.In(loadAffectedHostsPendingInHouseInstallsNotInListUA, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to load affected hosts for upcoming in-house installs")
		}
		var affectedHostIDs []uint
		if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "load affected hosts for upcoming in-house installs")
		}
		activateAffectedHostIDs = affectedHostIDs

		stmt, args, err = sqlx.In(deletePendingInHouseInstallsNotInListUA, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete upcoming pending in-house installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete upcoming pending host in-house install records")
		}

		stmt, args, err = sqlx.In(markInHouseInstallsNotInListAsRemoved, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to mark obsolete host in-house installs as removed")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "mark obsolete host in-house installs as removed")
		}

		stmt, args, err = sqlx.In(deleteInHouseInstallersNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete obsolete in-house installers")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete in-house installers")
		}

		for _, installer := range installers {
			if installer.ValidatedLabels == nil {
				return ctxerr.Errorf(ctx, "labels have not been validated for in-house app with name %s", installer.Filename)
			}

			wasUpdatedArgs := []interface{}{
				// package update
				installer.StorageID,
				// WHERE clause
				globalOrTeamID,
				BundleIdentifierOrName(installer.BundleIdentifier, strings.TrimSuffix(installer.Filename, ".ipa")),
				installer.Source,
			}

			// pull existing installer state if it exists so we can diff for side effects post-update
			type existingInstallerUpdateCheckResult struct {
				InstallerID        uint `db:"id"`
				IsPackageModified  bool `db:"is_package_modified"`
				IsMetadataModified bool
			}
			var existing []existingInstallerUpdateCheckResult
			err = sqlx.SelectContext(ctx, tx, &existing, checkExistingInstaller, wasUpdatedArgs...)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "checking for existing installer with name %q", installer.Filename)
			}

			args := []interface{}{
				BundleIdentifierOrName(installer.BundleIdentifier, strings.TrimSuffix(installer.Filename, ".ipa")),
				installer.Source,
				tmID,
				globalOrTeamID,
				installer.Filename,
				installer.Version,
				installer.StorageID,
				installer.Platform,
				installer.BundleIdentifier,
				installer.SelfService,
				installer.URL,
			}
			upsertQuery := insertNewOrEditedInstaller
			if len(existing) > 0 && existing[0].IsPackageModified { // update uploaded_at for updated installer package
				upsertQuery = fmt.Sprintf("%s, updated_at = NOW()", upsertQuery)
			}

			if _, err := tx.ExecContext(ctx, upsertQuery, args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited in-house app with name %q", installer.Filename)
			}

			// now that the software installer is created/updated, load its installer
			// ID (cannot use res.LastInsertID due to the upsert statement, won't
			// give the id in case of update)
			var installerID uint
			if err := sqlx.GetContext(ctx, tx, &installerID, loadInHouseInstallerID, globalOrTeamID, installer.Filename, installer.Platform); err != nil {
				return ctxerr.Wrapf(ctx, err, "load id of new/edited in-house app with name %q", installer.Filename)
			}

			// process the labels associated with that in-house installer
			if len(installer.ValidatedLabels.ByName) == 0 {
				// no label to apply, so just delete all existing labels if any
				res, err := tx.ExecContext(ctx, deleteAllInHouseLabels, installerID)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete in-house labels for %s", installer.Filename)
				}

				if n, _ := res.RowsAffected(); n > 0 && len(existing) > 0 {
					// if it did delete a row, then the target changed so pending
					// installs/uninstalls must be deleted
					existing[0].IsMetadataModified = true
				}
			} else {
				// there are new labels to apply, delete only the obsolete ones
				labelIDs := make([]uint, 0, len(installer.ValidatedLabels.ByName))
				for _, lbl := range installer.ValidatedLabels.ByName {
					labelIDs = append(labelIDs, lbl.LabelID)
				}
				stmt, args, err := sqlx.In(deleteInHouseLabelsNotInList, installerID, labelIDs)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build statement to delete in-house labels not in list")
				}

				res, err := tx.ExecContext(ctx, stmt, args...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete in-house labels not in list for %s", installer.Filename)
				}
				if n, _ := res.RowsAffected(); n > 0 && len(existing) > 0 {
					// if it did delete a row, then the target changed so pending
					// installs/uninstalls must be deleted
					existing[0].IsMetadataModified = true
				}

				excludeLabels := installer.ValidatedLabels.LabelScope == fleet.LabelScopeExcludeAny
				if len(existing) > 0 && !existing[0].IsMetadataModified {
					// load the remaining labels for that installer, so that we can detect
					// if any label changed (if the counts differ, then labels did change,
					// otherwise if the exclude bool changed, the target did change).
					var existingLabels []struct {
						LabelID uint `db:"label_id"`
						Exclude bool `db:"exclude"`
					}
					if err := sqlx.SelectContext(ctx, tx, &existingLabels, loadExistingInHouseLabels, installerID); err != nil {
						return ctxerr.Wrapf(ctx, err, "load existing labels for in-house with name %q", installer.Filename)
					}

					if len(existingLabels) != len(labelIDs) {
						existing[0].IsMetadataModified = true
					}
					if len(existingLabels) > 0 && existingLabels[0].Exclude != excludeLabels {
						// same labels are provided, but the include <-> exclude changed
						existing[0].IsMetadataModified = true
					}
				}

				// upsert the new labels now that obsolete ones have been deleted
				var upsertLabelArgs []any
				for _, lblID := range labelIDs {
					upsertLabelArgs = append(upsertLabelArgs, installerID, lblID, excludeLabels)
				}
				upsertLabelValues := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(installer.ValidatedLabels.ByName)), ",")

				_, err = tx.ExecContext(ctx, fmt.Sprintf(upsertInHouseLabels, upsertLabelValues), upsertLabelArgs...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "insert new/edited labels for in-house with name %q", installer.Filename)
				}
			}

			// perform side effects if this was an update (related to pending install requests)
			if len(existing) > 0 {
				affectedHostIDs, err := ds.runInHouseUpdateSideEffectsInTransaction(
					ctx,
					tx,
					existing[0].InstallerID,
					existing[0].IsMetadataModified,
					existing[0].IsPackageModified,
				)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "processing side-effects for in-house with name %q", installer.Filename)
				}
				activateAffectedHostIDs = append(activateAffectedHostIDs, affectedHostIDs...)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return ds.activateNextUpcomingActivityForBatchOfHosts(ctx, activateAffectedHostIDs)
}

func (ds *Datastore) runInHouseUpdateSideEffectsInTransaction(ctx context.Context, tx sqlx.ExtContext, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) (affectedHostIDs []uint, err error) {
	if wasMetadataUpdated || wasPackageUpdated { // cancel pending installs
		const cancelInHouseInstalls = `
UPDATE
	host_in_house_software_installs
SET
	canceled = 1
WHERE
	verification_at IS NULL AND
	verification_failed_at IS NULL AND
	in_house_app_id IN = ?
`
		_, err = tx.ExecContext(ctx, cancelInHouseInstalls, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "cancel pending host in-house installs")
		}

		const cancelInHouseCmds = `
UPDATE
	nano_enrollment_queue
SET
	active = 0
WHERE
	command_uuid IN (
		SELECT command_uuid
		FROM host_in_house_software_installs 
		WHERE
			verification_at IS NULL AND
			verification_failed_at IS NULL AND
			in_house_app_id = ?
	)
`
		_, err = tx.ExecContext(ctx, cancelInHouseCmds, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "cancel pending host in-house commands")
		}

		const loadAffectedHosts = `
SELECT
	DISTINCT host_id
FROM
	upcoming_activities ua
INNER JOIN in_house_app_upcoming_activities ihua
	ON ua.id = ihua.upcoming_activity_id
WHERE
	ua.activity_type = 'in_house_app_install' AND
	ua.activated_at IS NOT NULL AND
	ihua.in_house_app_id = ?
`
		if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs, loadAffectedHosts, installerID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select affected host IDs for in-house installs")
		}

		const deleteUpcomingInHouse = `
DELETE FROM upcoming_activities
USING upcoming_activities
INNER JOIN in_house_app_upcoming_activities ihua
	ON upcoming_activities.id = ihua.upcoming_activity_id
WHERE
	activity_type = 'in_house_app_install' AND
	ihua.in_house_app_id = ?
`

		_, err = tx.ExecContext(ctx, deleteUpcomingInHouse, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete upcoming host in-house installs")
		}
	}

	if wasPackageUpdated { // hide existing install counts
		const markInHouseRemoved = `
UPDATE host_in_house_software_installs SET removed = TRUE
WHERE in_house_app_id = ?
`
		_, err := tx.ExecContext(ctx, markInHouseRemoved, installerID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "hide existing install counts")
		}
	}

	return affectedHostIDs, nil
}

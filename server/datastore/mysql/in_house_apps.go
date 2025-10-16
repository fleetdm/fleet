package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) insertInHouseApp(ctx context.Context, payload *fleet.InHouseAppPayload) (uint, uint, error) {
	selectStmt := `SELECT COUNT(id) FROM in_house_apps WHERE global_or_team_id = ? AND (bundle_identifier = ? OR name = ?)`

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	titleIDipad, err := ds.getOrGenerateInHouseAppTitleID(ctx, payload.Name, payload.BundleID, "ipados_apps")
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}
	titleIDios, err := ds.getOrGenerateInHouseAppTitleID(ctx, payload.Name, payload.BundleID, "ios_apps")
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}

	var installerID uint
	var count uint
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		row := tx.QueryRowxContext(ctx, selectStmt, globalOrTeamID, payload.BundleID, payload.Name)
		if err := row.Scan(&count); err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp: ")
		}
		if count > 0 {
			// ios or ipados version of this installer exists
			err = alreadyExists("insertInHouseApp", payload.Name)
		}

		argsIos := []any{
			tid,
			globalOrTeamID,
			payload.Name,
			payload.StorageID,
			payload.Version,
			payload.BundleID,
			titleIDios,
			"ios",
		}
		argsIpad := []any{
			tid,
			globalOrTeamID,
			payload.Name,
			payload.StorageID,
			payload.Version,
			payload.BundleID,
			titleIDipad,
			"ipados",
		}

		_, err := ds.insertInHouseInstaller(ctx, tx, payload, argsIpad)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp: ")
		}

		installerID, err = ds.insertInHouseInstaller(ctx, tx, payload, argsIos)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp: ")
		}

		return nil
	})

	return installerID, titleIDios, ctxerr.Wrap(ctx, err, "insertInHouseApp")
}

func (ds *Datastore) getOrGenerateInHouseAppTitleID(ctx context.Context, name string, bundleID string, source string) (uint, error) {
	selectStmt := `SELECT id FROM software_titles WHERE bundle_identifier = ? AND source = ? OR (name = ? AND source = ?)`
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

func (ds *Datastore) insertInHouseInstaller(ctx context.Context, tx sqlx.ExtContext, payload *fleet.InHouseAppPayload, args []any) (uint, error) {
	stmt := `
	INSERT INTO in_house_apps (
		team_id,
		global_or_team_id,
		name,
		storage_id,
		version,
		bundle_identifier,
		title_id,
		platform
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		if IsDuplicate(err) {
			err = alreadyExists("insertInHouseApp", payload.Name)
		}
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}
	id64, err := res.LastInsertId()
	installerID := uint(id64) //nolint:gosec // dismiss G115
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}

	if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels, softwareTypeInHouseApp); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}
	return installerID, nil
}

func (ds *Datastore) GetInHouseAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
  iha.id,
  iha.team_id,
  iha.title_id,
  COALESCE(iha.name, '') AS software_title,
  iha.platform,
  iha.storage_id,
  st.bundle_identifier AS bundle_identifier,
  iha.version
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
                    name = ?,
                    version = ?
                 WHERE id = ?`

		// Avoid updating platform from .ipa file for now
		// ext := "ipa"
		// if i := strings.LastIndex(ext, "."); i != -1 {
		// 	ext = ext[i+1:]
		// }
		// platform, _ := fleet.SoftwareInstallerPlatformFromExtension(ext)

		args := []any{
			payload.StorageID,
			payload.Filename,
			payload.Version,
			// platform,
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
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &installs, `SELECT host_id, command_uuid FROM host_in_house_software_installs WHERE in_house_app_id = ?`, inHouseAppID)
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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) insertInHouseApp(ctx context.Context, payload *fleet.InHouseAppPayload) (uint, uint, error) {
	stmt := `
	INSERT INTO in_house_apps (
		team_id,
		title_id,
		global_or_team_id,
		name,
		storage_id,
		platform,
		version
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	titleID, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, &fleet.UploadSoftwareInstallerPayload{
		TeamID:           tid,
		Title:            payload.Name,
		BundleIdentifier: payload.BundleID,
		Source:           "ios_apps"}, // TODO: what about iPad apps
	)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}

	var installerID uint
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		args := []any{
			tid,
			titleID,
			globalOrTeamID,
			payload.Name,
			payload.StorageID,
			payload.Platform,
			payload.Version,
		}

		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			if IsDuplicate(err) {
				// already exists for this team/no team
				err = alreadyExists("InHouseApp", payload.Name)
			}
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		id64, err := res.LastInsertId()
		installerID = uint(id64) //nolint:gosec // dismiss G115
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels, softwareTypeInHouseApp); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert in house app labels")
		}

		return nil
	})

	return installerID, titleID, ctxerr.Wrap(ctx, err, "insertInHouseApp")
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
                    version = ?,
                    platform = ?
                 WHERE id = ?`

		ext := "ipa"
		if i := strings.LastIndex(ext, "."); i != -1 {
			ext = ext[i+1:]
		}
		platform, _ := fleet.SoftwareInstallerPlatformFromExtension(ext)

		args := []any{
			payload.StorageID,
			payload.Filename,
			payload.Version,
			platform,
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
		if err != nil {
			return ctxerr.Wrap(ctx, err, "remove pending in house app installs")
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

func (ds *Datastore) GetSummaryInHouseAppInstalls(ctx context.Context, teamID *uint, inHouseAppID uint) (*fleet.SoftwareInstallerStatusSummary, error) {
	var dest fleet.SoftwareInstallerStatusSummary // Using this struct for in house apps for now
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
	COALESCE(SUM( IF(status = :software_status_pending, 1, 0)), 0) AS pending_install,
	COALESCE(SUM( IF(status = :software_status_failed, 1, 0)), 0) AS failed_install,
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
		"software_status_pending":   fleet.SoftwareInstallPending,
		"software_status_failed":    fleet.SoftwareInstallFailed,
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

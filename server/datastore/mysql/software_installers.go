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
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListPendingSoftwareInstalls(ctx context.Context, hostID uint) ([]string, error) {
	const stmt = `
  SELECT
    execution_id
  FROM
    host_software_installs
  WHERE
    host_id = ?
  AND
    install_script_exit_code IS NULL
  AND
    pre_install_query_output IS NULL
  ORDER BY
    created_at ASC
`
	var results []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending software installs")
	}
	return results, nil
}

func (ds *Datastore) GetSoftwareInstallDetails(ctx context.Context, executionId string) (*fleet.SoftwareInstallDetails, error) {
	const stmt = `
  SELECT
    hsi.host_id AS host_id,
    hsi.execution_id AS execution_id,
    hsi.software_installer_id AS installer_id,
    hsi.self_service AS self_service,
    COALESCE(si.pre_install_query, '') AS pre_install_condition,
    inst.contents AS install_script,
    COALESCE(pisnt.contents, '') AS post_install_script
  FROM
    host_software_installs hsi
  INNER JOIN
    software_installers si
    ON hsi.software_installer_id = si.id
  LEFT OUTER JOIN
    script_contents inst
    ON inst.id = si.install_script_content_id
  LEFT OUTER JOIN
    script_contents pisnt
    ON pisnt.id = si.post_install_script_content_id
  WHERE
    hsi.execution_id = ?`

	result := &fleet.SoftwareInstallDetails{}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), result, stmt, executionId); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstallerDetails").WithName(executionId), "get software installer details")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software install details")
	}
	return result, nil
}

func (ds *Datastore) MatchOrCreateSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
	//
	// TODO(lucas): Check if labels exist first.
	//

	titleID, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, payload)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get or generate software installer title ID")
	}

	if err := ds.addSoftwareTitleToMatchingSoftware(ctx, titleID, payload); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "add software title to matching software")
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.InstallScript)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get or generate install script contents ID")
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != "" {
		sid, err := ds.getOrGenerateScriptContentsID(ctx, payload.PostInstallScript)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "get or generate post-install script contents ID")
		}
		postInstallScriptID = &sid
	}

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	stmt := `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	title_id,
	storage_id,
	filename,
	version,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	platform,
    self_service,
	install_type
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		tid,
		globalOrTeamID,
		titleID,
		payload.StorageID,
		payload.Filename,
		payload.Version,
		installScriptID,
		payload.PreInstallQuery,
		postInstallScriptID,
		payload.Platform,
		payload.SelfService,
		payload.InstallType,
	}

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		if IsDuplicate(err) {
			// already exists for this team/no team
			err = alreadyExists("SoftwareInstaller", payload.Title)
		}
		return 0, ctxerr.Wrap(ctx, err, "insert software installer")
	}

	id, _ := res.LastInsertId()

	// TODO(lucas): Make this a transaction, otherwise we may end up with automatic software being applied to all hosts
	// if the above succeeds and the below fails.

	// Insert associated labels for the created installer.
	if len(payload.LabelsExcludeAny) > 0 || len(payload.LabelsIncludeAny) > 0 {
		if err := ds.insertSoftwareInstallerLabels(ctx, uint(id), payload.LabelsIncludeAny, payload.LabelsExcludeAny); err != nil {
			return 0, ctxerr.Wrap(ctx, err, "insert software installer labels")
		}
	}

	return uint(id), nil
}

func (ds *Datastore) insertSoftwareInstallerLabels(ctx context.Context, softwareInstallerID uint, labelsIncludeAny []string, labelsExcludeAny []string) error {
	exclude := len(labelsExcludeAny) > 0 // only one of labelsIncludeAny/labelsExcludeAny can be set at this point.
	labelNames := labelsExcludeAny
	if len(labelsIncludeAny) > 0 {
		labelNames = labelsIncludeAny
	}

	values := strings.TrimSuffix(
		strings.Repeat("(?, (SELECT id FROM labels WHERE name = ?), ?),", len(labelNames)),
		",",
	)
	stmt := fmt.Sprintf(`
INSERT INTO software_installer_labels (
	software_installer_id,
	label_id,
	exclude
) VALUES %s`, values)

	args := make([]interface{}, 0, 3*len(labelNames))
	for _, labelName := range labelNames {
		args = append(args, softwareInstallerID, labelName, exclude)
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert software installer")
	}
	return nil
}

func (ds *Datastore) getOrGenerateSoftwareInstallerTitleID(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
	selectStmt := `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`
	selectArgs := []any{payload.Title, payload.Source}
	insertStmt := `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, '')`
	insertArgs := []any{payload.Title, payload.Source}

	if payload.BundleIdentifier != "" {
		selectStmt = `SELECT id FROM software_titles WHERE bundle_identifier = ?`
		selectArgs = []any{payload.BundleIdentifier}
		insertStmt = `INSERT INTO software_titles (name, source, bundle_identifier, browser) VALUES (?, ?, ?, '')`
		insertArgs = append(insertArgs, payload.BundleIdentifier)
	}

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

func (ds *Datastore) addSoftwareTitleToMatchingSoftware(ctx context.Context, titleID uint, payload *fleet.UploadSoftwareInstallerPayload) error {
	whereClause := "WHERE (s.name, s.source, s.browser) = (?, ?, '')"
	whereArgs := []any{payload.Title, payload.Source}
	if payload.BundleIdentifier != "" {
		whereClause = "WHERE s.bundle_identifier = ?"
		whereArgs = []any{payload.BundleIdentifier}
	}

	args := make([]any, 0, len(whereArgs))
	args = append(args, titleID)
	args = append(args, whereArgs...)
	updateSoftwareStmt := fmt.Sprintf(`
		    UPDATE software s
		    SET s.title_id = ?
		    %s`, whereClause)
	_, err := ds.writer(ctx).ExecContext(ctx, updateSoftwareStmt, args...)
	return ctxerr.Wrap(ctx, err, "adding fk reference in software to software_titles")
}

func (ds *Datastore) GetSoftwareInstallerMetadataByID(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
	si.id,
	si.team_id,
	si.title_id,
	si.storage_id,
	si.filename,
	si.version,
	si.install_script_content_id,
	si.pre_install_query,
	si.post_install_script_content_id,
	si.uploaded_at,
	COALESCE(st.name, '') AS software_title
FROM
	software_installers si
	LEFT OUTER JOIN software_titles st ON st.id = si.title_id
WHERE
	si.id = ?`

	var dest fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstaller").WithID(id), "get software installer metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software installer metadata")
	}

	return &dest, nil
}

func (ds *Datastore) GetSoftwareInstallerMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
	var scriptContentsSelect, scriptContentsFrom string
	if withScriptContents {
		scriptContentsSelect = ` , inst.contents AS install_script, COALESCE(pisnt.contents, '') AS post_install_script `
		scriptContentsFrom = ` LEFT OUTER JOIN script_contents inst ON inst.id = si.install_script_content_id
		LEFT OUTER JOIN script_contents pisnt ON pisnt.id = si.post_install_script_content_id `
	}

	query := fmt.Sprintf(`
SELECT
  si.id,
  si.team_id,
  si.title_id,
  si.storage_id,
  si.filename,
  si.version,
  si.install_script_content_id,
  si.pre_install_query,
  si.post_install_script_content_id,
  si.uploaded_at,
  si.self_service,
  COALESCE(st.name, '') AS software_title,
  si.install_type
  %s
FROM
  software_installers si
  JOIN software_titles st ON st.id = si.title_id
  %s
WHERE
  si.title_id = ? AND si.global_or_team_id = ?`,
		scriptContentsSelect, scriptContentsFrom)

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var dest fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, titleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareInstaller"), "get software installer metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software installer metadata")
	}

	type softwareInstallerLabel struct {
		LabelID   uint   `db:"label_id"`
		LabelName string `db:"label_name"`
		Exclude   bool   `db:"exclude"`
	}
	var softwareInstallerLabels []softwareInstallerLabel
	query = `SELECT 
		sil.label_id, l.name AS label_name, sil.exclude 
		FROM software_installer_labels sil 
		JOIN labels l ON sil.label_id=l.id
		WHERE software_installer_id = ?`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &softwareInstallerLabels, query, dest.InstallerID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer labels")
	}
	var (
		labelsExcludeAny []fleet.SoftwareInstallerLabel
		labelsIncludeAny []fleet.SoftwareInstallerLabel
	)
	for _, softwareInstallerLabel := range softwareInstallerLabels {
		item := fleet.SoftwareInstallerLabel{
			ID:   softwareInstallerLabel.LabelID,
			Name: softwareInstallerLabel.LabelName,
		}
		if softwareInstallerLabel.Exclude {
			labelsExcludeAny = append(labelsExcludeAny, item)
		} else {
			labelsIncludeAny = append(labelsIncludeAny, item)
		}
	}
	dest.LabelsExcludeAny = labelsExcludeAny
	dest.LabelsIncludeAny = labelsIncludeAny

	return &dest, nil
}

func (ds *Datastore) DeleteSoftwareInstaller(ctx context.Context, id uint) error {
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM software_installers WHERE id = ?`, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete software installer")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return notFound("SoftwareInstaller").WithID(id)
	}

	return nil
}

func (ds *Datastore) InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareInstallerID uint, selfService bool) (string, error) {
	const (
		insertStmt = `
		  INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id, user_id, self_service)
		  VALUES
		    (?, ?, ?, ?, ?)
		    `

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("Host").WithID(hostID)
		}

		return "", ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}
	installID := uuid.NewString()
	_, err = ds.writer(ctx).ExecContext(ctx, insertStmt,
		installID,
		hostID,
		softwareInstallerID,
		userID,
		selfService,
	)

	return installID, ctxerr.Wrap(ctx, err, "inserting new install software request")
}

func (ds *Datastore) GetSoftwareInstallResults(ctx context.Context, resultsUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	query := fmt.Sprintf(`
SELECT
	hsi.execution_id AS execution_id,
	hsi.pre_install_query_output,
	hsi.post_install_script_output,
	hsi.install_script_output,
	hsi.host_id AS host_id,
	st.name AS software_title,
	st.id AS software_title_id,
	COALESCE(%s, '') AS status,
	si.filename AS software_package,
	hsi.user_id AS user_id,
	hsi.post_install_script_exit_code,
	hsi.install_script_exit_code,
    hsi.self_service,
    hsi.host_deleted_at
FROM
	host_software_installs hsi
	JOIN software_installers si ON si.id = hsi.software_installer_id
	JOIN software_titles st ON si.title_id = st.id
LEFT JOIN (
	SELECT hs.host_id, s.title_id, s.version AS installed_version
	FROM host_software hs
	INNER JOIN software s ON hs.software_id = s.id
) isth ON isth.title_id = st.id AND isth.host_id = hsi.host_id
WHERE
	hsi.execution_id = :execution_id
	`, softwareInstallerHostStatusNamedQuery("hsi", "isth", "si", ""))

	stmt, args, err := sqlx.Named(query, map[string]any{
		"execution_id":              resultsUUID,
		"software_status_pending":   fleet.SoftwareInstallerPending,
		"software_status_blocked":   fleet.SoftwareInstallerBlocked,
		"software_status_verifying": fleet.SoftwareInstallerVerifying,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_verified":  fleet.SoftwareInstallerVerified,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build named query for get software install results")
	}

	var dest fleet.HostSoftwareInstallerResult
	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostSoftwareInstallerResult"), "get host software installer results")
		}
		return nil, ctxerr.Wrap(ctx, err, "get host software installer results")
	}

	return &dest, nil
}

func (ds *Datastore) GetSummaryHostSoftwareInstalls(ctx context.Context, installerID uint) (*fleet.SoftwareInstallerStatusSummary, error) {
	var dest fleet.SoftwareInstallerStatusSummary

	stmt := fmt.Sprintf(`
SELECT
	COALESCE(SUM( IF(status = :software_status_pending, 1, 0)), 0) AS pending,
	COALESCE(SUM( IF(status = :software_status_blocked, 1, 0)), 0) AS blocked,
	COALESCE(SUM( IF(status = :software_status_verifying, 1, 0)), 0) AS verifying,
	COALESCE(SUM( IF(status = :software_status_failed, 1, 0)), 0) AS failed,
	COALESCE(SUM( IF(status = :software_status_verified, 1, 0)), 0) AS verified
FROM (
SELECT
	software_installer_id,
	%s
FROM
	host_software_installs hsi
JOIN
	software_installers si ON hsi.software_installer_id = si.id
LEFT JOIN (
	SELECT s.title_id, s.version AS installed_version
	FROM host_software hs
	INNER JOIN software s ON hs.software_id = s.id
) isth ON isth.title_id = si.title_id
WHERE
	software_installer_id = :installer_id
	AND hsi.id IN(
		SELECT
			max(id) -- ensure we use only the most recently created install attempt for each host
			FROM host_software_installs
		WHERE
			software_installer_id = :installer_id
			AND host_deleted_at IS NULL
		GROUP BY
			host_id)) s`, softwareInstallerHostStatusNamedQuery("hsi", "isth", "si", "status"))

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"installer_id":              installerID,
		"software_status_pending":   fleet.SoftwareInstallerPending,
		"software_status_blocked":   fleet.SoftwareInstallerBlocked,
		"software_status_verifying": fleet.SoftwareInstallerVerifying,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_verified":  fleet.SoftwareInstallerVerified,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host software installs: named query")
	}

	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host software install status")
	}

	return &dest, nil
}

func (ds *Datastore) vppAppJoin(appID fleet.VPPAppID, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	stmt := fmt.Sprintf(`JOIN (
SELECT
	host_id
FROM
	host_vpp_software_installs hvsi
LEFT OUTER JOIN
	nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
WHERE
	adam_id = :adam_id AND platform = :platform
	AND hvsi.id IN(
		SELECT
			max(id) -- ensure we use only the most recent install attempt for each host
			FROM host_vpp_software_installs
		WHERE
			adam_id = :adam_id AND platform = :platform
		GROUP BY
			host_id, adam_id)
	AND (%s) = :status) hss ON hss.host_id = h.id
`, vppAppHostStatusNamedQuery("hvsi", "ncr", ""))

	return sqlx.Named(stmt, map[string]interface{}{
		"status":                    status,
		"adam_id":                   appID.AdamID,
		"platform":                  appID.Platform,
		"software_status_verifying": fleet.SoftwareInstallerVerifying,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_pending":   fleet.SoftwareInstallerPending,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
	})
}

func (ds *Datastore) softwareInstallerJoin(installerID uint, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	stmt := fmt.Sprintf(`JOIN (
SELECT
	host_id
FROM
	host_software_installs hsi
JOIN software_installers si ON hsi.installer_id = si.id
LEFT JOIN (
	SELECT s.title_id, s.version AS installed_version
	FROM host_software hs
	INNER JOIN software s ON hs.software_id = s.id
) isth ON isth.title_id = si.title_id
WHERE
	software_installer_id = :installer_id
	AND hsi.id IN(
		SELECT
			max(id) -- ensure we use only the most recent install attempt for each host
			FROM host_software_installs
		WHERE
			software_installer_id = :installer_id
		GROUP BY
			host_id, software_installer_id)
	AND (%s) = :status) hss ON hss.host_id = h.id
`, softwareInstallerHostStatusNamedQuery("hsi", "isth", "si", ""))

	return sqlx.Named(stmt, map[string]interface{}{
		"status":                    status,
		"installer_id":              installerID,
		"software_status_verifying": fleet.SoftwareInstallerVerifying,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_pending":   fleet.SoftwareInstallerPending,
	})
}

func (ds *Datastore) CleanupUnusedSoftwareInstallers(ctx context.Context, softwareInstallStore fleet.SoftwareInstallerStore) error {
	if softwareInstallStore == nil {
		// no-op in this case, possible if not running with a Premium license
		return nil
	}

	// get the list of software installers hashes that are in use
	var storageIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &storageIDs, `SELECT DISTINCT storage_id FROM software_installers`); err != nil {
		return ctxerr.Wrap(ctx, err, "get list of software installers in use")
	}

	_, err := softwareInstallStore.Cleanup(ctx, storageIDs)
	return ctxerr.Wrap(ctx, err, "cleanup unused software installers")
}

func (ds *Datastore) BatchSetSoftwareInstallers(ctx context.Context, tmID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
	const upsertSoftwareTitles = `
INSERT INTO software_titles
  (name, source, browser)
VALUES
  %s
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  source = VALUES(source),
  browser = VALUES(browser)
`

	const loadSoftwareTitles = `
SELECT
  id
FROM
  software_titles
WHERE (name, source, browser) IN (%s)
`
	const deleteAllInstallersInTeam = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ?
`

	const deleteInstallersNotInList = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ? AND
  title_id NOT IN (?)
`

	const insertNewOrEditedInstaller = `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	storage_id,
	filename,
	version,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	platform,
	self_service,
	title_id
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
  (SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = '')
)
ON DUPLICATE KEY UPDATE
  install_script_content_id = VALUES(install_script_content_id),
  post_install_script_content_id = VALUES(post_install_script_content_id),
  storage_id = VALUES(storage_id),
  filename = VALUES(filename),
  version = VALUES(version),
  pre_install_query = VALUES(pre_install_query),
  platform = VALUES(platform),
  self_service = VALUES(self_service)
`

	// use a team id of 0 if no-team
	var globalOrTeamID uint
	if tmID != nil {
		globalOrTeamID = *tmID
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// if no installers are provided, just delete whatever was in
		// the table
		if len(installers) == 0 {
			_, err := tx.ExecContext(ctx, deleteAllInstallersInTeam, globalOrTeamID)
			return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
		}

		var args []any
		for _, installer := range installers {
			args = append(args, installer.Title, installer.Source, "")
		}

		values := strings.TrimSuffix(
			strings.Repeat("(?,?,?),", len(installers)),
			",",
		)
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(upsertSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert new/edited software title")
		}

		var titleIDs []uint
		if err := sqlx.SelectContext(ctx, tx, &titleIDs, fmt.Sprintf(loadSoftwareTitles, values), args...); err != nil {
			return ctxerr.Wrap(ctx, err, "load existing titles")
		}

		stmt, args, err := sqlx.In(deleteInstallersNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete obsolete installers")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
		}

		for _, installer := range installers {
			isRes, err := insertScriptContents(ctx, tx, installer.InstallScript)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting install script contents for software installer with name %q", installer.Filename)
			}
			installScriptID, _ := isRes.LastInsertId()

			var postInstallScriptID *int64
			if installer.PostInstallScript != "" {
				pisRes, err := insertScriptContents(ctx, tx, installer.PostInstallScript)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "inserting post-install script contents for software installer with name %q", installer.Filename)
				}

				insertID, _ := pisRes.LastInsertId()
				postInstallScriptID = &insertID
			}

			args := []interface{}{
				tmID,
				globalOrTeamID,
				installer.StorageID,
				installer.Filename,
				installer.Version,
				installScriptID,
				installer.PreInstallQuery,
				postInstallScriptID,
				installer.Platform,
				installer.SelfService,
				installer.Title,
				installer.Source,
			}

			if _, err := tx.ExecContext(ctx, insertNewOrEditedInstaller, args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited installer with name %q", installer.Filename)
			}
		}
		return nil
	})
}

func (ds *Datastore) HasSelfServiceSoftwareInstallers(ctx context.Context, hostPlatform string, hostTeamID *uint) (bool, error) {
	if fleet.IsLinux(hostPlatform) {
		hostPlatform = "linux"
	}
	stmt := `SELECT 1 FROM software_installers WHERE self_service = 1 AND platform = ? AND global_or_team_id = ?`
	var globalOrTeamID uint
	if hostTeamID != nil {
		globalOrTeamID = *hostTeamID
	}
	args := []interface{}{hostPlatform, globalOrTeamID}
	var hasInstallers bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hasInstallers, stmt, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, ctxerr.Wrap(ctx, err, "check for self-service software installers")
	}
	return hasInstallers, nil
}

func (ds *Datastore) TriggerHostSoftwareInstallations(ctx context.Context, hostID uint, hostTeamID *uint, hostPlatform string, hostInstalledSoftware []fleet.Software) error {
	//
	// Get the available for install automatic software for this host.
	//
	// TODO(lucas): Check if we can probably used a more optimized version than re-using this.
	automaticSoftware, _, err := ds.ListHostSoftware(ctx, &fleet.Host{
		ID:       hostID,
		TeamID:   hostTeamID,
		Platform: hostPlatform,
	}, fleet.HostSoftwareTitleListOptions{
		OnlyAvailableForInstall: true,
		InstallType:             string(fleet.SoftwareInstallerInstallTypeAutomatic),
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host automatic software installers")
	}

	//
	// Filter out automatic software that has a status (Status == nil
	// means the software is available but wasn't actioned yet).
	//
	// TODO(lucas): What do we do with pending/failed/blocked statuses here? Probably ignore them.
	var availableAutomaticSoftware []*fleet.HostSoftwareWithInstaller
	for i := range automaticSoftware {
		if automaticSoftware[i].Status == nil {
			availableAutomaticSoftware = append(availableAutomaticSoftware, automaticSoftware[i])
		}
	}

	//
	// Prepare a map of installed software.
	//
	installedSoftwareMap := make(map[string]fleet.Software, len(hostInstalledSoftware))
	for _, hostInstalledSoftwareItem := range hostInstalledSoftware {
		key := hostInstalledSoftwareItem.Name + hostInstalledSoftwareItem.BundleIdentifier + hostInstalledSoftwareItem.Version
		installedSoftwareMap[key] = hostInstalledSoftwareItem
	}

	//
	// For each automatic software that is un-actioned and is not installed, trigger an install request.
	//
	for _, automaticSoftwareItem := range availableAutomaticSoftware {
		bundleIdentifier := ""
		if automaticSoftwareItem.BundleIdentifier != nil {
			bundleIdentifier = *automaticSoftwareItem.BundleIdentifier
		}
		// Name and bundle identifier come from software_titles
		// Version comes from the software_installers.
		key := automaticSoftwareItem.Name + bundleIdentifier + *&automaticSoftwareItem.SoftwarePackage.Version
		if _, ok := installedSoftwareMap[key]; !ok {
			// An automatic software installer is not present on this host, thus we trigger an installation (if there isn't one).
			if _, err := ds.InsertSoftwareInstallRequest(
				ctx, hostID, automaticSoftwareItem.SoftwarePackage.SoftwareInstallerID, *automaticSoftwareItem.SoftwarePackage.SelfService,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "insert software install request for automatic software")
			}
		}
	}

	return nil
}

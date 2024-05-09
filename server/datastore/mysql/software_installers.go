package mysql

import (
	"context"
	"database/sql"
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
    si.pre_install_query AS pre_install_condition,
    inst.contents AS install_script,
    pisnt.contents AS post_install_script
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
		return nil, ctxerr.Wrap(ctx, err, "list pending software installs")
	}
	return result, nil
}

func (ds *Datastore) MatchOrCreateSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
	titleID, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, payload.Title, payload.Source)
	if err != nil {
		return 0, err
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.InstallScript)
	if err != nil {
		return 0, err
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != "" {
		sid, err := ds.getOrGenerateScriptContentsID(ctx, payload.PostInstallScript)
		if err != nil {
			return 0, err
		}
		postInstallScriptID = &sid
	}

	var tid uint
	if payload.TeamID != nil {
		tid = *payload.TeamID
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
	post_install_script_content_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		payload.TeamID,
		tid,
		titleID,
		payload.StorageID,
		payload.Filename,
		payload.Version,
		installScriptID,
		payload.PreInstallQuery,
		postInstallScriptID,
	}

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		if isDuplicate(err) {
			// already exists for this team/no team
			err = alreadyExists("SoftwareInstaller", payload.Title)
		}
		return 0, ctxerr.Wrap(ctx, err, "insert software installer")
	}

	id, _ := res.LastInsertId()

	return uint(id), nil
}

func (ds *Datastore) getOrGenerateSoftwareInstallerTitleID(ctx context.Context, name, source string) (uint, error) {
	titleID, err := ds.optimisticGetOrInsert(ctx,
		&parameterizedStmt{
			Statement: `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`,
			Args:      []interface{}{name, source},
		},
		&parameterizedStmt{
			Statement: `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)`,
			Args:      []interface{}{name, source, ""},
		},
	)
	if err != nil {
		return 0, err
	}

	return titleID, nil
}

func (ds *Datastore) GetSoftwareInstallerMetadata(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
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

func (ds *Datastore) GetSoftwareInstallerMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
	id,
	team_id,
	title_id,
	storage_id,
	filename,
	version,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	uploaded_at
FROM 
	software_installers
WHERE 
	title_id = ? AND global_or_team_id = ?`

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

func (ds *Datastore) GetSoftwareInstallerForTitle(ctx context.Context, softwareTitleID uint, teamID *uint) (*fleet.SoftwareInstaller, error) {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	const getInstallerIDStmt = `
SELECT
	id,
	team_id,
	title_id,
	storage_id,
	filename,
	version,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	uploaded_at
FROM
	software_installers
WHERE
	title_id = ? AND global_or_team_id = ?`

	var installer fleet.SoftwareInstaller
	err := sqlx.GetContext(ctx, ds.reader(ctx), &installer, getInstallerIDStmt, softwareTitleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("SoftwareInstaller")
		}

		return nil, ctxerr.Wrap(ctx, err, "finding software installer by title")
	}

	return &installer, nil
}

func (ds *Datastore) InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareInstallerID uint) error {
	const (
		insertStmt = `
		  INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id, user_id)
		  VALUES
		    (?, ?, ?, ?)
		    `

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
	_, err = ds.writer(ctx).ExecContext(ctx, insertStmt,
		uuid.NewString(),
		hostID,
		softwareInstallerID,
		userID,
	)

	return ctxerr.Wrap(ctx, err, "inserting new install software request")
}

func (ds *Datastore) GetSoftwareInstallResults(ctx context.Context, resultsUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	query := `
SELECT
	hsi.execution_id AS execution_id,
	COALESCE(hsi.pre_install_query_output, '') AS pre_install_query_output,
	COALESCE(hsi.post_install_script_output, '') AS post_install_script_output,
	COALESCE(hsi.install_script_output, '') AS install_script_output,
	hsi.host_id AS host_id,
	h.computer_name AS host_display_name,
	st.name AS software_title,
	st.id AS software_title_id,
	COALESCE(CASE
		WHEN hsi.post_install_script_exit_code IS NOT NULL AND
			hsi.post_install_script_exit_code = 0 THEN ? -- installed
		WHEN hsi.post_install_script_exit_code IS NOT NULL AND
			hsi.post_install_script_exit_code != 0 THEN ? -- failed
		WHEN hsi.install_script_exit_code IS NOT NULL AND
			hsi.install_script_exit_code = 0 THEN ? -- installed
		WHEN hsi.install_script_exit_code IS NOT NULL AND
			hsi.install_script_exit_code != 0 THEN ? -- failed
		WHEN hsi.pre_install_query_output IS NOT NULL AND
			hsi.pre_install_query_output = '' THEN ? -- failed
		WHEN hsi.host_id IS NOT NULL THEN ? -- pending
		ELSE NULL -- not installed from Fleet installer
	END, '') AS status,
	si.filename AS software_package,
	h.team_id AS host_team_id,
	hsi.user_id AS user_id
FROM
	host_software_installs hsi
	JOIN hosts h ON h.id = hsi.host_id
	JOIN software_installers si ON si.id = hsi.software_installer_id
	JOIN software_titles st ON si.title_id = st.id
WHERE
	hsi.execution_id = ?
	`

	var dest fleet.HostSoftwareInstallerResult
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, fleet.SoftwareInstallerInstalled, fleet.SoftwareInstallerFailed, fleet.SoftwareInstallerInstalled, fleet.SoftwareInstallerFailed, fleet.SoftwareInstallerFailed, fleet.SoftwareInstallerPending, resultsUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostSoftwareInstallerResult"), "get host software installer results")
		}
		return nil, ctxerr.Wrap(ctx, err, "get host software installer results")
	}

	return &dest, nil
}

func tmplNamedSQLCaseHostSoftwareInstallStatus(alias string) string {
	return fmt.Sprintf(`
	CASE WHEN %[1]s.post_install_script_exit_code IS NOT NULL
		AND %[1]s.post_install_script_exit_code = 0 THEN
		:installed
	WHEN %[1]s.post_install_script_exit_code IS NOT NULL
		AND %[1]s.post_install_script_exit_code != 0 THEN
		:failed
	WHEN %[1]s.install_script_exit_code IS NOT NULL
		AND %[1]s.install_script_exit_code = 0 THEN
		:installed
	WHEN %[1]s.install_script_exit_code IS NOT NULL
		AND %[1]s.install_script_exit_code != 0 THEN
		:failed
	WHEN %[1]s.pre_install_query_output IS NOT NULL
		AND %[1]s.pre_install_query_output = '' THEN
		:failed
	WHEN %[1]s.host_id IS NOT NULL THEN
		:pending
	ELSE
		NULL -- not installed from Fleet installer
	END`, alias)
}

func (ds *Datastore) GetSummaryHostSoftwareInstalls(ctx context.Context, installerID uint) (*fleet.SoftwareInstallerStatusSummary, error) {
	var dest fleet.SoftwareInstallerStatusSummary

	stmt := fmt.Sprintf(`
SELECT
	COALESCE(SUM( IF(status = :pending, 1, 0)), 0) AS pending,
	COALESCE(SUM( IF(status = :failed, 1, 0)), 0) AS failed,
	COALESCE(SUM( IF(status = :installed, 1, 0)), 0) AS installed
FROM (
SELECT
	software_installer_id,
	%s AS status
FROM
	host_software_installs hsi
WHERE
	software_installer_id = :installer_id
	AND id IN(
		SELECT
			max(id) -- ensure we use only the most recently created install attempt for each host
			FROM host_software_installs
		WHERE 
			software_installer_id = :installer_id
		GROUP BY
			host_id)) s`, tmplNamedSQLCaseHostSoftwareInstallStatus("hsi"))

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"installer_id": installerID,
		"pending":      fleet.SoftwareInstallerPending,
		"failed":       fleet.SoftwareInstallerFailed,
		"installed":    fleet.SoftwareInstallerInstalled,
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

func (ds *Datastore) softwareInstallerJoin(installerID uint, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	stmt := fmt.Sprintf(`JOIN (
SELECT
	host_id
FROM
	host_software_installs hsi
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
`, tmplNamedSQLCaseHostSoftwareInstallStatus("hsi"))

	return sqlx.Named(stmt, map[string]interface{}{
		"status":       status,
		"installer_id": installerID,
		"installed":    fleet.SoftwareInstallerInstalled,
		"failed":       fleet.SoftwareInstallerFailed,
		"pending":      fleet.SoftwareInstallerPending,
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
	title_id
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?,
  (SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = '')
)
ON DUPLICATE KEY UPDATE
  install_script_content_id = VALUES(install_script_content_id),
  post_install_script_content_id = VALUES(post_install_script_content_id),
  storage_id = VALUES(storage_id),
  filename = VALUES(filename),
  version = VALUES(version),
  pre_install_query = VALUES(pre_install_query)
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
			isRes, err := insertScriptContents(ctx, installer.InstallScript, tx)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting install script contents for software installer with name %q", installer.Filename)
			}
			installScriptID, _ := isRes.LastInsertId()

			var postInstallScriptID *int64
			if installer.PostInstallScript != "" {
				pisRes, err := insertScriptContents(ctx, installer.PostInstallScript, tx)
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

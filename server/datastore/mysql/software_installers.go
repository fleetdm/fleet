package mysql

import (
	"context"
	"database/sql"

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
	id = ?`

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

func (ds *Datastore) InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareTitleID uint, teamID *uint) error {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	const (
		insertStmt = `
		  INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id)
		  VALUES
		    (?, ?, ?)
		    `

		getInstallerIDStmt = `SELECT id FROM software_installers WHERE title_id = ? AND global_or_team_id = ?`

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return notFound("Host").WithID(hostID)
		}

		return ctxerr.Wrap(ctx, err, "inserting new install software request")
	}

	var installerID uint
	err = sqlx.GetContext(ctx, ds.reader(ctx), &installerID, getInstallerIDStmt, softwareTitleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return notFound("SoftwareInstaller")
		}

		return ctxerr.Wrap(ctx, err, "inserting new install software request")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, insertStmt,
		uuid.NewString(),
		hostID,
		installerID,
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
	h.team_id AS host_team_id
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

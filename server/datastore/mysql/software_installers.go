package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

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
		hostID,
		uuid.NewString(),
		softwareTitleID,
	)

	return ctxerr.Wrap(ctx, err, "inserting new install software request")
}

func tmplNamedSQLCaseHostSoftwareInstallStatus(alias string) string {
	return fmt.Sprintf(`
	CASE WHEN %[1]s.post_install_script_exit_code IS NOT NULL AND %[1]s.post_install_script_exit_code = 0 THEN
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

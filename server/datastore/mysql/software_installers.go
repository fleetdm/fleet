package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
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
	status = ?
  ORDER BY
    created_at ASC
`
	var results []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostID, fleet.SoftwareInstallPending); err != nil {
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
    uninst.contents AS uninstall_script,
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
    script_contents uninst
    ON uninst.id = si.uninstall_script_content_id
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

func (ds *Datastore) MatchOrCreateSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (installerID, titleID uint, err error) {
	if payload.ValidatedLabels == nil {
		// caller must ensure this is not nil; if caller intends no labels to be created,
		// payload.ValidatedLabels should point to an empty struct.
		return 0, 0, errors.New("validated labels must not be nil")
	}

	titleID, err = ds.getOrGenerateSoftwareInstallerTitleID(ctx, payload)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate software installer title ID")
	}

	if err := ds.addSoftwareTitleToMatchingSoftware(ctx, titleID, payload); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "add software title to matching software")
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.InstallScript)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate install script contents ID")
	}

	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.UninstallScript)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != "" {
		sid, err := ds.getOrGenerateScriptContentsID(ctx, payload.PostInstallScript)
		if err != nil {
			return 0, 0, ctxerr.Wrap(ctx, err, "get or generate post-install script contents ID")
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

	if err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	title_id,
	storage_id,
	filename,
	extension,
	version,
	package_ids,
	install_script_content_id,
	pre_install_query,
	post_install_script_content_id,
    uninstall_script_content_id,
	platform,
    self_service,
	user_id,
	user_name,
	user_email,
	fleet_library_app_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (SELECT name FROM users WHERE id = ?), (SELECT email FROM users WHERE id = ?), ?)`

		args := []interface{}{
			tid,
			globalOrTeamID,
			titleID,
			payload.StorageID,
			payload.Filename,
			payload.Extension,
			payload.Version,
			strings.Join(payload.PackageIDs, ","),
			installScriptID,
			payload.PreInstallQuery,
			postInstallScriptID,
			uninstallScriptID,
			payload.Platform,
			payload.SelfService,
			payload.UserID,
			payload.UserID,
			payload.UserID,
			payload.FleetLibraryAppID,
		}

		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			if IsDuplicate(err) {
				// already exists for this team/no team
				err = alreadyExists("SoftwareInstaller", payload.Title)
			}
			return err
		}

		id, _ := res.LastInsertId()
		installerID = uint(id) //nolint:gosec // dismiss G115

		if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert software installer labels")
		}

		return nil
	}); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insert software installer")
	}

	return installerID, titleID, nil
}

func (ds *Datastore) getOrGenerateSoftwareInstallerTitleID(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, error) {
	selectStmt := `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`
	selectArgs := []any{payload.Title, payload.Source}
	insertStmt := `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, '')`
	insertArgs := []any{payload.Title, payload.Source}

	if payload.BundleIdentifier != "" {
		// match by bundle identifier first, or standard matching if we don't have a bundle identifier match
		selectStmt = `SELECT id FROM software_titles WHERE bundle_identifier = ? OR (name = ? AND source = ? AND browser = '') ORDER BY bundle_identifier = ? DESC LIMIT 1`
		selectArgs = []any{payload.BundleIdentifier, payload.Title, payload.Source, payload.BundleIdentifier}
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

// setOrUpdateSoftwareInstallerLabelsDB sets or updates the label associations for the specified software
// installer. If no labels are provided, it will remove all label associations with the software installer.
func setOrUpdateSoftwareInstallerLabelsDB(ctx context.Context, tx sqlx.ExtContext, installerID uint, labels fleet.LabelIdentsWithScope) error {
	labelIds := make([]uint, 0, len(labels.ByName))
	for _, label := range labels.ByName {
		labelIds = append(labelIds, label.LabelID)
	}

	// remove existing labels
	delArgs := []interface{}{installerID}
	delStmt := `DELETE FROM software_installer_labels WHERE software_installer_id = ?`
	if len(labelIds) > 0 {
		inStmt, args, err := sqlx.In(` AND label_id NOT IN (?)`, labelIds)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build delete existing software installer labels query")
		}
		delArgs = append(delArgs, args...)
		delStmt += inStmt
	}
	_, err := tx.ExecContext(ctx, delStmt, delArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing software installer labels")
	}

	// insert new labels
	if len(labelIds) > 0 {
		var exclude bool
		switch labels.LabelScope {
		case fleet.LabelScopeIncludeAny:
			exclude = false
		case fleet.LabelScopeExcludeAny:
			exclude = true
		default:
			// this should never happen
			return ctxerr.New(ctx, "invalid label scope")
		}

		stmt := `INSERT INTO software_installer_labels (software_installer_id, label_id, exclude) VALUES %s ON DUPLICATE KEY UPDATE exclude = VALUES(exclude)`
		var placeholders string
		var insertArgs []interface{}
		for _, lid := range labelIds {
			placeholders += "(?, ?, ?),"
			insertArgs = append(insertArgs, installerID, lid, exclude)
		}
		placeholders = strings.TrimSuffix(placeholders, ",")

		_, err = tx.ExecContext(ctx, fmt.Sprintf(stmt, placeholders), insertArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert software installer label")
		}
	}

	return nil
}

func (ds *Datastore) UpdateInstallerSelfServiceFlag(ctx context.Context, selfService bool, id uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE software_installers SET self_service = ? WHERE id = ?`, selfService, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer")
	}

	return nil
}

func (ds *Datastore) SaveInstallerUpdates(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) error {
	if payload.InstallScript == nil || payload.UninstallScript == nil || payload.PreInstallQuery == nil || payload.SelfService == nil {
		return ctxerr.Wrap(ctx, errors.New("missing installer update payload fields"), "update installer record")
	}

	installScriptID, err := ds.getOrGenerateScriptContentsID(ctx, *payload.InstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate install script contents ID")
	}

	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, *payload.UninstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}

	var postInstallScriptID *uint
	if payload.PostInstallScript != nil && *payload.PostInstallScript != "" { // pointer because optional
		sid, err := ds.getOrGenerateScriptContentsID(ctx, *payload.PostInstallScript)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get or generate post-install script contents ID")
		}
		postInstallScriptID = &sid
	}

	touchUploaded := ""
	if payload.InstallerFile != nil {
		touchUploaded = ", uploaded_at = NOW()"
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := fmt.Sprintf(`UPDATE software_installers SET
	storage_id = ?,
	filename = ?,
	version = ?,
	package_ids = ?,
	install_script_content_id = ?,
	pre_install_query = ?,
	post_install_script_content_id = ?,
    uninstall_script_content_id = ?,
    self_service = ?,
	user_id = ?,
	user_name = (SELECT name FROM users WHERE id = ?),
	user_email = (SELECT email FROM users WHERE id = ?) %s
	WHERE id = ?`, touchUploaded)

		args := []interface{}{
			payload.StorageID,
			payload.Filename,
			payload.Version,
			strings.Join(payload.PackageIDs, ","),
			installScriptID,
			*payload.PreInstallQuery,
			postInstallScriptID,
			uninstallScriptID,
			*payload.SelfService,
			payload.UserID,
			payload.UserID,
			payload.UserID,
			payload.InstallerID,
		}

		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update software installer")
		}

		if payload.ValidatedLabels != nil {
			if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, payload.InstallerID, *payload.ValidatedLabels); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert software installer labels")
			}
		}

		return nil
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer")
	}

	return nil
}

func (ds *Datastore) ValidateOrbitSoftwareInstallerAccess(ctx context.Context, hostID uint, installerID uint) (bool, error) {
	query := `
    SELECT 1
    FROM
      host_software_installs
    WHERE
      software_installer_id = ?
    AND
      host_id = ?
    AND
      install_script_exit_code IS NULL
`
	var access bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &access, query, installerID, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check software installer association to host")
	}
	return true, nil
}

func (ds *Datastore) GetSoftwareInstallerMetadataByID(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
	query := `
SELECT
	si.id,
	si.team_id,
	si.title_id,
	si.storage_id,
	si.package_ids,
	si.filename,
	si.extension,
	si.version,
	si.install_script_content_id,
	si.pre_install_query,
	si.post_install_script_content_id,
	si.uninstall_script_content_id,
	si.uploaded_at,
	COALESCE(st.name, '') AS software_title,
	si.platform,
	si.fleet_library_app_id
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
		scriptContentsSelect = ` , inst.contents AS install_script, COALESCE(pinst.contents, '') AS post_install_script, uninst.contents AS uninstall_script `
		scriptContentsFrom = ` LEFT OUTER JOIN script_contents inst ON inst.id = si.install_script_content_id
		LEFT OUTER JOIN script_contents pinst ON pinst.id = si.post_install_script_content_id
		LEFT OUTER JOIN script_contents uninst ON uninst.id = si.uninstall_script_content_id`
	}

	query := fmt.Sprintf(`
SELECT
  si.id,
  si.team_id,
  si.title_id,
  si.storage_id,
  si.package_ids,
  si.filename,
  si.extension,
  si.version,
  si.install_script_content_id,
  si.pre_install_query,
  si.post_install_script_content_id,
  si.uninstall_script_content_id,
  si.uploaded_at,
  si.self_service,
  COALESCE(st.name, '') AS software_title
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

	// TODO: do we want to include labels on other queries that return software installer metadata
	// (e.g., GetSoftwareInstallerMetadataByID)?
	labels, err := ds.getSoftwareInstallerLabels(ctx, dest.InstallerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer labels")
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
		// there's a bug somewhere
		level.Debug(ds.logger).Log("msg", "software installer has both include and exclude labels", "installer_id", dest.InstallerID, "include", fmt.Sprintf("%v", inclAny), "exclude", fmt.Sprintf("%v", exclAny))
	}
	dest.LabelsExcludeAny = exclAny
	dest.LabelsIncludeAny = inclAny

	policies, err := ds.getPoliciesBySoftwareTitleIDs(ctx, []uint{titleID}, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies by software title ID")
	}
	dest.AutomaticInstallPolicies = policies

	return &dest, nil
}

func (ds *Datastore) getSoftwareInstallerLabels(ctx context.Context, installerID uint) ([]fleet.SoftwareScopeLabel, error) {
	query := `
SELECT
	label_id,
	exclude,
	l.name as label_name,
	si.title_id
FROM
	software_installer_labels sil
	JOIN software_installers si ON si.id = sil.software_installer_id
	JOIN labels l ON l.id = sil.label_id
WHERE
	software_installer_id = ?`

	var labels []fleet.SoftwareScopeLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, query, installerID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installer labels")
	}

	return labels, nil
}

var (
	errDeleteInstallerWithAssociatedPolicy = &fleet.ConflictError{Message: "Couldn't delete. Policy automation uses this software. Please disable policy automation for this software and try again."}
	errDeleteInstallerInstalledDuringSetup = &fleet.ConflictError{Message: "Couldn't delete. This software is installed when new Macs boot. Please remove software in Controls > Setup experience and try again."}
)

func (ds *Datastore) DeleteSoftwareInstaller(ctx context.Context, id uint) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		err := ds.runInstallerUpdateSideEffectsInTransaction(ctx, tx, id, true, true)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clean up related installs and uninstalls")
		}

		// allow delete only if install_during_setup is false
		res, err := tx.ExecContext(ctx, `DELETE FROM software_installers WHERE id = ? AND install_during_setup = 0`, id)
		if err != nil {
			if isMySQLForeignKey(err) {
				// Check if the software installer is referenced by a policy automation.
				var count int
				if err := sqlx.GetContext(ctx, tx, &count, `SELECT COUNT(*) FROM policies WHERE software_installer_id = ?`, id); err != nil {
					return ctxerr.Wrapf(ctx, err, "getting reference from policies")
				}
				if count > 0 {
					return errDeleteInstallerWithAssociatedPolicy
				}
			}
			return ctxerr.Wrap(ctx, err, "delete software installer")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			// could be that the software installer does not exist, or it is installed
			// during setup, do additional check.
			var installDuringSetup bool
			if err := sqlx.GetContext(ctx, tx, &installDuringSetup,
				`SELECT install_during_setup FROM software_installers WHERE id = ?`, id); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, err, "check if software installer is installed during setup")
			}
			if installDuringSetup {
				return errDeleteInstallerInstalledDuringSetup
			}
			return notFound("SoftwareInstaller").WithID(id)
		}

		return nil
	})
}

// deletePendingSoftwareInstallsForPolicy should be called after a policy is deleted to remove any pending software installs
func (ds *Datastore) deletePendingSoftwareInstallsForPolicy(ctx context.Context, teamID *uint, policyID uint) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	const deleteStmt = `
		DELETE FROM
			host_software_installs
		WHERE
			policy_id = ? AND
			status = ? AND
			software_installer_id IN (
				SELECT id FROM software_installers WHERE global_or_team_id = ?
			)
	`
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, policyID, fleet.SoftwareInstallPending, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete pending software installs for policy")
	}

	return nil
}

func (ds *Datastore) InsertSoftwareInstallRequest(ctx context.Context, hostID uint, softwareInstallerID uint, selfService bool, policyID *uint) (string, error) {
	const (
		getInstallerStmt = `SELECT filename, "version", title_id, COALESCE(st.name, '[deleted title]') title_name
			FROM software_installers si LEFT JOIN software_titles st ON si.title_id = st.id WHERE si.id = ?`
		insertStmt = `
		  INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id, user_id, self_service, policy_id, installer_filename, version, software_title_id, software_title_name)
		  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

	var installerDetails struct {
		Filename  string  `db:"filename"`
		Version   string  `db:"version"`
		TitleID   *uint   `db:"title_id"`
		TitleName *string `db:"title_name"`
	}
	if err = sqlx.GetContext(ctx, ds.reader(ctx), &installerDetails, getInstallerStmt, softwareInstallerID); err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("SoftwareInstaller").WithID(softwareInstallerID)
		}

		return "", ctxerr.Wrap(ctx, err, "getting installer data")
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
		policyID,
		installerDetails.Filename,
		installerDetails.Version,
		installerDetails.TitleID,
		installerDetails.TitleName,
	)

	return installID, ctxerr.Wrap(ctx, err, "inserting new install software request")
}

func (ds *Datastore) ProcessInstallerUpdateSideEffects(ctx context.Context, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return ds.runInstallerUpdateSideEffectsInTransaction(ctx, tx, installerID, wasMetadataUpdated, wasPackageUpdated)
	})
}

func (ds *Datastore) runInstallerUpdateSideEffectsInTransaction(ctx context.Context, tx sqlx.ExtContext, installerID uint, wasMetadataUpdated bool, wasPackageUpdated bool) error {
	if wasMetadataUpdated || wasPackageUpdated { // cancel pending installs/uninstalls
		// TODO make this less naive; this assumes that installs/uninstalls execute and report back immediately
		_, err := tx.ExecContext(ctx, `DELETE FROM host_script_results WHERE execution_id IN (
				SELECT execution_id FROM host_software_installs WHERE software_installer_id = ? AND status = 'pending_uninstall'
			)`, installerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete pending uninstall scripts")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM host_software_installs
			   WHERE software_installer_id = ? AND status IN('pending_install', 'pending_uninstall')`, installerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete pending host software installs/uninstalls")
		}
	}

	if wasPackageUpdated { // hide existing install counts
		_, err := tx.ExecContext(ctx, `UPDATE host_software_installs SET removed = TRUE
	  			WHERE software_installer_id = ? AND status IS NOT NULL AND host_deleted_at IS NULL`, installerID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "hide existing install counts")
		}
	}

	return nil
}

func (ds *Datastore) InsertSoftwareUninstallRequest(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint) error {
	const (
		getInstallerStmt = `SELECT title_id, COALESCE(st.name, '[deleted title]') title_name
			FROM software_installers si LEFT JOIN software_titles st ON si.title_id = st.id WHERE si.id = ?`
		insertStmt = `
		  INSERT INTO host_software_installs
		    (execution_id, host_id, software_installer_id, user_id, uninstall, installer_filename, software_title_id, software_title_name, version)
		  VALUES (?, ?, ?, ?, 1, '', ?, ?, 'unknown')
		    `
		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notFound("Host").WithID(hostID)
		}
		return ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var installerDetails struct {
		TitleID   *uint   `db:"title_id"`
		TitleName *string `db:"title_name"`
	}
	if err = sqlx.GetContext(ctx, ds.reader(ctx), &installerDetails, getInstallerStmt, softwareInstallerID); err != nil {
		if err == sql.ErrNoRows {
			return notFound("SoftwareInstaller").WithID(softwareInstallerID)
		}

		return ctxerr.Wrap(ctx, err, "getting installer data")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}
	_, err = ds.writer(ctx).ExecContext(ctx, insertStmt,
		executionID,
		hostID,
		softwareInstallerID,
		userID,
		installerDetails.TitleID,
		installerDetails.TitleName,
	)

	return ctxerr.Wrap(ctx, err, "inserting new uninstall software request")
}

func (ds *Datastore) GetSoftwareInstallResults(ctx context.Context, resultsUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	query := `
SELECT
	hsi.execution_id AS execution_id,
	hsi.pre_install_query_output,
	hsi.post_install_script_output,
	hsi.install_script_output,
	hsi.host_id AS host_id,
	COALESCE(st.name, hsi.software_title_name) AS software_title,
	hsi.software_title_id,
	COALESCE(hsi.execution_status, '') AS status,
	hsi.installer_filename AS software_package,
	hsi.user_id AS user_id,
	hsi.post_install_script_exit_code,
	hsi.install_script_exit_code,
	hsi.self_service,
	hsi.host_deleted_at,
	hsi.policy_id,
	hsi.created_at as created_at,
	hsi.updated_at as updated_at
FROM
	host_software_installs hsi
	LEFT JOIN software_titles st ON hsi.software_title_id = st.id
WHERE
	hsi.execution_id = :execution_id
	`

	stmt, args, err := sqlx.Named(query, map[string]any{
		"execution_id": resultsUUID,
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

	stmt := `
SELECT
	COALESCE(SUM( IF(status = :software_status_pending_install, 1, 0)), 0) AS pending_install,
	COALESCE(SUM( IF(status = :software_status_failed_install, 1, 0)), 0) AS failed_install,
	COALESCE(SUM( IF(status = :software_status_pending_uninstall, 1, 0)), 0) AS pending_uninstall,
	COALESCE(SUM( IF(status = :software_status_failed_uninstall, 1, 0)), 0) AS failed_uninstall,
	COALESCE(SUM( IF(status = :software_status_installed, 1, 0)), 0) AS installed
FROM (
SELECT
	software_installer_id,
	status
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
			AND host_deleted_at IS NULL
			AND removed = 0
		GROUP BY
			host_id)) s`

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"installer_id":                      installerID,
		"software_status_pending_install":   fleet.SoftwareInstallPending,
		"software_status_failed_install":    fleet.SoftwareInstallFailed,
		"software_status_pending_uninstall": fleet.SoftwareUninstallPending,
		"software_status_failed_uninstall":  fleet.SoftwareUninstallFailed,
		"software_status_installed":         fleet.SoftwareInstalled,
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
	// Since VPP does not have uninstaller yet, we map the generic pending/failed statuses to the install statuses
	switch status {
	case fleet.SoftwarePending:
		status = fleet.SoftwareInstallPending
	case fleet.SoftwareFailed:
		status = fleet.SoftwareInstallFailed
	default:
		// no change
	}
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
		"software_status_installed": fleet.SoftwareInstalled,
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_pending":   fleet.SoftwareInstallPending,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
	})
}

func (ds *Datastore) softwareInstallerJoin(installerID uint, status fleet.SoftwareInstallerStatus) (string, []interface{}, error) {
	statusFilter := "hsi.status = :status"
	var status2 fleet.SoftwareInstallerStatus
	switch status {
	case fleet.SoftwarePending:
		status = fleet.SoftwareInstallPending
		status2 = fleet.SoftwareUninstallPending
	case fleet.SoftwareFailed:
		status = fleet.SoftwareInstallFailed
		status2 = fleet.SoftwareUninstallFailed
	default:
		// no change
	}
	if status2 != "" {
		statusFilter = "hsi.status IN (:status, :status2)"
	}
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
			AND removed = 0
		GROUP BY
			host_id, software_installer_id)
	AND %s) hss ON hss.host_id = h.id
`, statusFilter)

	return sqlx.Named(stmt, map[string]interface{}{
		"status":       status,
		"status2":      status2,
		"installer_id": installerID,
	})
}

func (ds *Datastore) GetHostLastInstallData(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
	stmt := `
		SELECT execution_id, hsi.status
		FROM host_software_installs hsi
		WHERE hsi.id = (
			SELECT
				MAX(id)
			FROM host_software_installs
			WHERE
				software_installer_id = :installer_id AND host_id = :host_id
			GROUP BY
				host_id, software_installer_id)
`

	stmt, args, err := sqlx.Named(stmt, map[string]interface{}{
		"host_id":      hostID,
		"installer_id": installerID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build named query to get host last install data")
	}

	var hostLastInstall fleet.HostLastInstallData
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &hostLastInstall, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get host last install data")
	}
	return &hostLastInstall, nil
}

func (ds *Datastore) CleanupUnusedSoftwareInstallers(ctx context.Context, softwareInstallStore fleet.SoftwareInstallerStore, removeCreatedBefore time.Time) error {
	if softwareInstallStore == nil {
		// no-op in this case, possible if not running with a Premium license
		return nil
	}

	// get the list of software installers hashes that are in use
	var storageIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &storageIDs, `SELECT DISTINCT storage_id FROM software_installers`); err != nil {
		return ctxerr.Wrap(ctx, err, "get list of software installers in use")
	}

	_, err := softwareInstallStore.Cleanup(ctx, storageIDs, removeCreatedBefore)
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

	const unsetAllInstallersFromPolicies = `
UPDATE
  policies
SET
  software_installer_id = NULL
WHERE
  team_id = ?
`

	const deleteAllPendingUninstallScriptExecutions = `
		DELETE FROM host_script_results WHERE execution_id IN (
			SELECT execution_id FROM host_software_installs WHERE status = 'pending_uninstall'
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ?
			   )
		)
`
	const deleteAllPendingSoftwareInstalls = `
		DELETE FROM host_software_installs
		   WHERE status IN('pending_install', 'pending_uninstall')
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ?
			   )
`
	const markAllSoftwareInstallsAsRemoved = `
		UPDATE host_software_installs SET removed = TRUE
			WHERE status IS NOT NULL AND host_deleted_at IS NULL
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ?
			   )
`

	const deleteAllInstallersInTeam = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ?
`

	const deletePendingUninstallScriptExecutionsNotInList = `
		DELETE FROM host_script_results WHERE execution_id IN (
			SELECT execution_id FROM host_software_installs WHERE status = 'pending_uninstall'
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			   )
		)
`
	const deletePendingSoftwareInstallsNotInList = `
		DELETE FROM host_software_installs
		   WHERE status IN('pending_install', 'pending_uninstall')
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			   )
`
	const markSoftwareInstallsNotInListAsRemoved = `
		UPDATE host_software_installs SET removed = TRUE
			WHERE status IS NOT NULL AND host_deleted_at IS NULL
				AND software_installer_id IN (
					SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id NOT IN (?)
			   )
`

	const unsetInstallersNotInListFromPolicies = `
UPDATE
  policies
SET
  software_installer_id = NULL
WHERE
  software_installer_id IN (
    SELECT id FROM software_installers
    WHERE global_or_team_id = ? AND
    title_id NOT IN (?)
  )
`

	const countInstallDuringSetupNotInList = `
SELECT
  COUNT(*)
FROM
  software_installers
WHERE
  global_or_team_id = ? AND
  title_id NOT IN (?) AND
  install_during_setup = 1
`

	const deleteInstallersNotInList = `
DELETE FROM
  software_installers
WHERE
  global_or_team_id = ? AND
  title_id NOT IN (?)
`

	const checkExistingInstaller = `
SELECT id,
storage_id != ? is_package_modified,
install_script_content_id != ? OR uninstall_script_content_id != ? OR pre_install_query != ? OR
COALESCE(post_install_script_content_id != ? OR
	(post_install_script_content_id IS NULL AND ? IS NOT NULL) OR
	(? IS NULL AND post_install_script_content_id IS NOT NULL)
, FALSE) is_metadata_modified FROM software_installers
WHERE global_or_team_id = ?	AND title_id IN (SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = '')
`

	const insertNewOrEditedInstaller = `
INSERT INTO software_installers (
	team_id,
	global_or_team_id,
	storage_id,
	filename,
	extension,
	version,
	install_script_content_id,
	uninstall_script_content_id,
	pre_install_query,
	post_install_script_content_id,
	platform,
	self_service,
	title_id,
	user_id,
	user_name,
	user_email,
	url,
	package_ids,
	install_during_setup
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
  (SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''),
  ?, (SELECT name FROM users WHERE id = ?), (SELECT email FROM users WHERE id = ?), ?, ?, COALESCE(?, false)
)
ON DUPLICATE KEY UPDATE
  install_script_content_id = VALUES(install_script_content_id),
  uninstall_script_content_id = VALUES(uninstall_script_content_id),
  post_install_script_content_id = VALUES(post_install_script_content_id),
  storage_id = VALUES(storage_id),
  filename = VALUES(filename),
  extension = VALUES(extension),
  version = VALUES(version),
  pre_install_query = VALUES(pre_install_query),
  platform = VALUES(platform),
  self_service = VALUES(self_service),
  user_id = VALUES(user_id),
  user_name = VALUES(user_name),
  user_email = VALUES(user_email),
  url = VALUES(url),
  install_during_setup = COALESCE(?, install_during_setup)
`

	const loadSoftwareInstallerID = `
SELECT
	id
FROM
	software_installers
WHERE
	global_or_team_id = ?	AND
	-- this is guaranteed to select a single title_id, due to unique index
	title_id IN (SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = '')
`

	const deleteInstallerLabelsNotInList = `
DELETE FROM
	software_installer_labels
WHERE
	software_installer_id = ? AND
	label_id NOT IN (?)
`

	const deleteAllInstallerLabels = `
DELETE FROM 
	software_installer_labels
WHERE
	software_installer_id = ?
`

	const upsertInstallerLabels = `
INSERT INTO
	software_installer_labels (
		software_installer_id,
		label_id,
		exclude
	)
VALUES
	%s
ON DUPLICATE KEY UPDATE
	exclude = VALUES(exclude)
`

	const loadExistingInstallerLabels = `
SELECT
	label_id,
	exclude
FROM
	software_installer_labels
WHERE
	software_installer_id = ?
`

	// use a team id of 0 if no-team
	var globalOrTeamID uint
	if tmID != nil {
		globalOrTeamID = *tmID
	}

	// if we're batch-setting installers and replacing the ones installed during
	// setup in the same go, no need to validate that we don't delete one marked
	// as install during setup (since we're overwriting those). This is always
	// called from fleetctl gitops, so it should always be the case anyway.
	var replacingInstallDuringSetup bool
	if len(installers) == 0 || installers[0].InstallDuringSetup != nil {
		replacingInstallDuringSetup = true
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// if no installers are provided, just delete whatever was in
		// the table
		if len(installers) == 0 {
			if _, err := tx.ExecContext(ctx, unsetAllInstallersFromPolicies, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "unset all obsolete installers in policies")
			}

			if _, err := tx.ExecContext(ctx, deleteAllPendingUninstallScriptExecutions, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all pending uninstall script executions")
			}

			if _, err := tx.ExecContext(ctx, deleteAllPendingSoftwareInstalls, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete all pending host software install records")
			}

			if _, err := tx.ExecContext(ctx, markAllSoftwareInstallsAsRemoved, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "mark all host software installs as removed")
			}

			if _, err := tx.ExecContext(ctx, deleteAllInstallersInTeam, globalOrTeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
			}

			return nil
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

		stmt, args, err := sqlx.In(unsetInstallersNotInListFromPolicies, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to unset obsolete installers from policies")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "unset obsolete software installers from policies")
		}

		// check if any in the list are install_during_setup, fail if there is one
		if !replacingInstallDuringSetup {
			stmt, args, err = sqlx.In(countInstallDuringSetupNotInList, globalOrTeamID, titleIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build statement to check installers install_during_setup")
			}
			var countInstallDuringSetup int
			if err := sqlx.GetContext(ctx, tx, &countInstallDuringSetup, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "check installers installed during setup")
			}
			if countInstallDuringSetup > 0 {
				return errDeleteInstallerInstalledDuringSetup
			}
		}

		stmt, args, err = sqlx.In(deletePendingUninstallScriptExecutionsNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete pending uninstall script executions")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete pending uninstall script executions")
		}

		stmt, args, err = sqlx.In(deletePendingSoftwareInstallsNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete pending software installs")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete pending host software install records")
		}

		stmt, args, err = sqlx.In(markSoftwareInstallsNotInListAsRemoved, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to mark obsolete host software installs as removed")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "mark obsolete host software installs as removed")
		}

		stmt, args, err = sqlx.In(deleteInstallersNotInList, globalOrTeamID, titleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build statement to delete obsolete installers")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete software installers")
		}

		for _, installer := range installers {
			if installer.ValidatedLabels == nil {
				return ctxerr.Errorf(ctx, "labels have not been validated for installer with name %s", installer.Filename)
			}

			isRes, err := insertScriptContents(ctx, tx, installer.InstallScript)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting install script contents for software installer with name %q", installer.Filename)
			}
			installScriptID, _ := isRes.LastInsertId()

			uisRes, err := insertScriptContents(ctx, tx, installer.UninstallScript)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting uninstall script contents for software installer with name %q", installer.Filename)
			}
			uninstallScriptID, _ := uisRes.LastInsertId()

			var postInstallScriptID *int64
			if installer.PostInstallScript != "" {
				pisRes, err := insertScriptContents(ctx, tx, installer.PostInstallScript)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "inserting post-install script contents for software installer with name %q", installer.Filename)
				}

				insertID, _ := pisRes.LastInsertId()
				postInstallScriptID = &insertID
			}

			wasUpdatedArgs := []interface{}{
				// package update
				installer.StorageID,
				// metadata update
				installScriptID,
				uninstallScriptID,
				installer.PreInstallQuery,
				postInstallScriptID,
				postInstallScriptID,
				postInstallScriptID,
				// WHERE clause
				globalOrTeamID,
				installer.Title,
				installer.Source,
			}

			// pull existing installer state if it exists so we can diff for side effects post-update
			type existingInstallerUpdateCheckResult struct {
				InstallerID        uint `db:"id"`
				IsPackageModified  bool `db:"is_package_modified"`
				IsMetadataModified bool `db:"is_metadata_modified"`
			}
			var existing []existingInstallerUpdateCheckResult
			err = sqlx.SelectContext(ctx, tx, &existing, checkExistingInstaller, wasUpdatedArgs...)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return ctxerr.Wrapf(ctx, err, "checking for existing installer with name %q", installer.Filename)
				}
			}

			args := []interface{}{
				tmID,
				globalOrTeamID,
				installer.StorageID,
				installer.Filename,
				installer.Extension,
				installer.Version,
				installScriptID,
				uninstallScriptID,
				installer.PreInstallQuery,
				postInstallScriptID,
				installer.Platform,
				installer.SelfService,
				installer.Title,
				installer.Source,
				installer.UserID,
				installer.UserID,
				installer.UserID,
				installer.URL,
				strings.Join(installer.PackageIDs, ","),
				installer.InstallDuringSetup,
				installer.InstallDuringSetup,
			}
			upsertQuery := insertNewOrEditedInstaller
			if len(existing) > 0 && existing[0].IsPackageModified { // update uploaded_at for updated installer package
				upsertQuery = fmt.Sprintf("%s, uploaded_at = NOW()", upsertQuery)
			}

			if _, err := tx.ExecContext(ctx, upsertQuery, args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited installer with name %q", installer.Filename)
			}

			// now that the software installer is created/updated, load its installer
			// ID (cannot use res.LastInsertID due to the upsert statement, won't
			// give the id in case of update)
			var installerID uint
			if err := sqlx.GetContext(ctx, tx, &installerID, loadSoftwareInstallerID, globalOrTeamID, installer.Title, installer.Source); err != nil {
				return ctxerr.Wrapf(ctx, err, "load id of new/edited installer with name %q", installer.Filename)
			}

			// process the labels associated with that software installer
			if len(installer.ValidatedLabels.ByName) == 0 {
				// no label to apply, so just delete all existing labels if any
				res, err := tx.ExecContext(ctx, deleteAllInstallerLabels, installerID)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer labels for %s", installer.Filename)
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
				stmt, args, err := sqlx.In(deleteInstallerLabelsNotInList, installerID, labelIDs)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build statement to delete installer labels not in list")
				}

				res, err := tx.ExecContext(ctx, stmt, args...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "delete installer labels not in list for %s", installer.Filename)
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
					if err := sqlx.SelectContext(ctx, tx, &existingLabels, loadExistingInstallerLabels, installerID); err != nil {
						return ctxerr.Wrapf(ctx, err, "load existing labels for installer with name %q", installer.Filename)
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

				_, err = tx.ExecContext(ctx, fmt.Sprintf(upsertInstallerLabels, upsertLabelValues), upsertLabelArgs...)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "insert new/edited labels for installer with name %q", installer.Filename)
				}
			}

			// perform side effects if this was an update (related to pending (un)install requests)
			if len(existing) > 0 {
				if err := ds.runInstallerUpdateSideEffectsInTransaction(
					ctx,
					tx,
					existing[0].InstallerID,
					existing[0].IsMetadataModified,
					existing[0].IsPackageModified,
				); err != nil {
					return ctxerr.Wrapf(ctx, err, "processing installer with name %q", installer.Filename)
				}
			}
		}

		return nil
	})
}

func (ds *Datastore) HasSelfServiceSoftwareInstallers(ctx context.Context, hostPlatform string, hostTeamID *uint) (bool, error) {
	if fleet.IsLinux(hostPlatform) {
		hostPlatform = "linux"
	}
	stmt := `SELECT 1
		WHERE EXISTS (
			SELECT 1
			FROM software_installers
			WHERE self_service = 1 AND platform = ? AND global_or_team_id = ?
		) OR EXISTS (
			SELECT 1
			FROM vpp_apps_teams
			WHERE self_service = 1 AND platform = ? AND global_or_team_id = ?
		)`
	var globalOrTeamID uint
	if hostTeamID != nil {
		globalOrTeamID = *hostTeamID
	}
	args := []interface{}{hostPlatform, globalOrTeamID, hostPlatform, globalOrTeamID}
	var hasInstallers bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hasInstallers, stmt, args...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, ctxerr.Wrap(ctx, err, "check for self-service software installers")
	}
	return hasInstallers, nil
}

func (ds *Datastore) GetSoftwareTitleNameFromExecutionID(ctx context.Context, executionID string) (string, error) {
	stmt := `
	SELECT name
	FROM software_titles st
	INNER JOIN software_installers si ON si.title_id = st.id
	INNER JOIN host_software_installs hsi ON hsi.software_installer_id = si.id
	WHERE hsi.execution_id = ?
	`
	var name string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &name, stmt, executionID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get software title name from execution ID")
	}
	return name, nil
}

func (ds *Datastore) GetSoftwareInstallersWithoutPackageIDs(ctx context.Context) (map[uint]string, error) {
	query := `
		SELECT id, storage_id FROM software_installers WHERE package_ids = ''
	`
	type result struct {
		ID        uint   `db:"id"`
		StorageID string `db:"storage_id"`
	}

	var results []result
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installers without package ID")
	}
	if len(results) == 0 {
		return nil, nil
	}
	idMap := make(map[uint]string, len(results))
	for _, r := range results {
		idMap[r.ID] = r.StorageID
	}
	return idMap, nil
}

func (ds *Datastore) UpdateSoftwareInstallerWithoutPackageIDs(ctx context.Context, id uint,
	payload fleet.UploadSoftwareInstallerPayload,
) error {
	uninstallScriptID, err := ds.getOrGenerateScriptContentsID(ctx, payload.UninstallScript)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get or generate uninstall script contents ID")
	}
	query := `
		UPDATE software_installers
		SET package_ids = ?, uninstall_script_content_id = ?, extension = ?
		WHERE id = ?
	`
	_, err = ds.writer(ctx).ExecContext(ctx, query, strings.Join(payload.PackageIDs, ","), uninstallScriptID, payload.Extension, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software installer without package ID")
	}
	return nil
}

func (ds *Datastore) GetSoftwareInstallers(ctx context.Context, teamID uint) ([]fleet.SoftwarePackageResponse, error) {
	const loadInsertedSoftwareInstallers = `
SELECT
  team_id,
  title_id,
  url
FROM
  software_installers
WHERE global_or_team_id = ?
`
	var softwarePackages []fleet.SoftwarePackageResponse
	// Using ds.writer(ctx) on purpose because this method is to be called after applying software.
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &softwarePackages, loadInsertedSoftwareInstallers, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software installers")
	}
	return softwarePackages, nil
}

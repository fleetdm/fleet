package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250926123048, Down_20250926123048)
}

func Up_20250926123048(tx *sql.Tx) error {
	// Find incorrect/correct title
	titleRows, err := tx.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('corp.sap.privileges.pkg', 'corp.sap.privileges')
	`)
	if err != nil {
		return err
	}
	defer titleRows.Close()

	bundleIdToTitleId := map[string]string{}
	for titleRows.Next() {
		var id, bundleIdentifier string
		if err := titleRows.Scan(&id, &bundleIdentifier); err != nil {
			return err
		}
		bundleIdToTitleId[bundleIdentifier] = id
	}
	if err := titleRows.Err(); err != nil {
		return err
	}

	if len(bundleIdToTitleId) == 0 {
		// No "Privileges" titles, nothing to do
		return nil
	}

	// If the "Privileges" app has not been indexed by osquery,
	// we will not have the correct title/bundle
	// so we need to insert it
	if _, ok := bundleIdToTitleId["corp.sap.privileges"]; !ok {
		res, err := tx.Exec(`
			INSERT INTO software_titles (name, source, bundle_identifier) VALUES
			('Privileges', 'apps', 'corp.sap.privileges')
		`)
		if err != nil {
			return err
		}

		lastInsertId, err := res.LastInsertId()
		if err != nil {
			return err
		}
		bundleIdToTitleId["corp.sap.privileges"] = fmt.Sprintf("%d", lastInsertId)
	}

	// Find software installers with incorrect title
	installerRows, err := tx.Query(`
		SELECT id
		FROM software_installers
		WHERE title_id = ?
		AND extension = 'pkg'
	`, bundleIdToTitleId["corp.sap.privileges.pkg"])
	if err != nil {
		return err
	}
	defer installerRows.Close()

	var softwareInstallerIds []string
	for installerRows.Next() {
		var id string
		if err := installerRows.Scan(&id); err != nil {
			return err
		}
		softwareInstallerIds = append(softwareInstallerIds, id)
	}
	if err := installerRows.Err(); err != nil {
		return err
	}

	// Update software installers to point to correct title
	for _, softwareInstallerId := range softwareInstallerIds {
		if _, err := tx.Exec(`
			UPDATE software_installers
			SET title_id = ?
			WHERE id = ?
		`, bundleIdToTitleId["corp.sap.privileges"], softwareInstallerId); err != nil {
			return err
		}
	}

	// Delete incorrect title if exists
	if incorrectTitleId, ok := bundleIdToTitleId["corp.sap.privileges.pkg"]; ok {
		if _, err := tx.Exec(`
			DELETE FROM software_titles
			WHERE id = ?
		`, incorrectTitleId); err != nil {
			return err
		}
	}

	return nil
}

func Down_20250926123048(tx *sql.Tx) error {
	return nil
}

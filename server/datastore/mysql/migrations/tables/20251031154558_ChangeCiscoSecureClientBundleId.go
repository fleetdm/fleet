package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251031154558, Down_20251031154558)
}

func Up_20251031154558(tx *sql.Tx) error {
	// Note: this is almost the same migration as 20250926123048_ChangePrivilegesInstallerBundleId
	// Since it is the same problem of an installer making a software title with
	// the wrong bundle id (used the pkg-ref instead)

	// Find incorrect/correct title
	titleRows, err := tx.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('com.cisco.pkg.anyconnect.vpn', 'com.cisco.secureclient.gui')
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
		return nil
	}

	// If the "Cisco Secure Client" app has not been indexed by osquery,
	// we will not have the correct title/bundle
	// so we need to insert it
	if _, ok := bundleIdToTitleId["com.cisco.secureclient.gui"]; !ok {
		res, err := tx.Exec(`
			INSERT INTO software_titles (name, source, bundle_identifier) VALUES
			('Cisco Secure Client', 'apps', 'com.cisco.secureclient.gui')
		`)
		if err != nil {
			return err
		}

		lastInsertId, err := res.LastInsertId()
		if err != nil {
			return err
		}
		bundleIdToTitleId["com.cisco.secureclient.gui"] = fmt.Sprintf("%d", lastInsertId)
	}

	// Find software installers with incorrect title
	installerRows, err := tx.Query(`
		SELECT id
		FROM software_installers
		WHERE title_id = ?
		AND extension = 'pkg'
	`, bundleIdToTitleId["com.cisco.pkg.anyconnect.vpn"])
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
		`, bundleIdToTitleId["com.cisco.secureclient.gui"], softwareInstallerId); err != nil {
			return err
		}
	}

	// Delete incorrect title if exists
	if incorrectTitleId, ok := bundleIdToTitleId["com.cisco.pkg.anyconnect.vpn"]; ok {
		if _, err := tx.Exec(`
			DELETE FROM software_titles
			WHERE id = ?
		`, incorrectTitleId); err != nil {
			return err
		}
	}
	return nil
}

func Down_20251031154558(tx *sql.Tx) error {
	return nil
}

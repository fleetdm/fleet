package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260326210603, Down_20260326210603)
}

func Up_20260326210603(tx *sql.Tx) error {
	// Update software_titles to use FMA canonical names where there's a matching
	// bundle_identifier. This fixes existing titles that were created with
	// osquery-reported names (e.g., "Code") instead of the FMA name
	// (e.g., "Microsoft Visual Studio Code").
	//
	// software_titles.bundle_identifier already has an index (idx_software_titles_bundle_identifier).
	// A later migration adds idx_software_bundle_identifier on software.bundle_identifier
	// so the hourly FMA sync UPDATE below (and the runtime equivalent in
	// UpsertMaintainedApp) is an indexed lookup instead of a full-table scan.
	_, err := tx.Exec(`
		UPDATE software_titles st
		JOIN fleet_maintained_apps fma
			ON st.bundle_identifier = fma.unique_identifier
			AND fma.platform = 'darwin'
		SET st.name = fma.name
		WHERE st.bundle_identifier IS NOT NULL
			AND st.bundle_identifier != ''
			AND st.name != fma.name
	`)
	if err != nil {
		return err
	}

	// Also update software entries to match their software_titles names.
	// This ensures consistency when navigating from software_titles to software versions.
	_, err = tx.Exec(`
		UPDATE software s
		JOIN fleet_maintained_apps fma
			ON s.bundle_identifier = fma.unique_identifier
			AND fma.platform = 'darwin'
		SET s.name = fma.name
		WHERE s.bundle_identifier IS NOT NULL
			AND s.bundle_identifier != ''
			AND s.name != fma.name
	`)
	return err
}

func Down_20260326210603(tx *sql.Tx) error {
	// Down migration is a no-op because we cannot reliably restore the original
	// osquery-reported names. The FMA names are the canonical/correct names anyway.
	return nil
}

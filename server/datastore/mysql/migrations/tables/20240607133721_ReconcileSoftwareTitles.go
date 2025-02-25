package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240607133721, Down_20240607133721)
}

func Up_20240607133721(tx *sql.Tx) error {

	// For users that are not running vulnerabilities job, we need to ensure that software_titles are up-to-date

	upsertTitlesStmt := `
INSERT INTO software_titles (name, source, browser)
SELECT DISTINCT
	name,
	source,
	browser
FROM
	software s
WHERE
	NOT EXISTS (SELECT 1 FROM software_titles st WHERE (s.name, s.source, s.browser) = (st.name, st.source, st.browser))
ON DUPLICATE KEY UPDATE software_titles.id = software_titles.id`

	_, err := tx.Exec(upsertTitlesStmt)
	if err != nil {
		return fmt.Errorf("failed to upsert software titles: %w", err)
	}

	// update title ids for software table entries
	updateSoftwareStmt := `
UPDATE
	software s,
	software_titles st
SET
	s.title_id = st.id
WHERE
	(s.name, s.source, s.browser) = (st.name, st.source, st.browser)
	AND (s.title_id IS NULL OR s.title_id != st.id)`

	_, err = tx.Exec(updateSoftwareStmt)
	if err != nil {
		return fmt.Errorf("failed to update software title_id: %w", err)
	}
	return nil
}

func Down_20240607133721(tx *sql.Tx) error {
	return nil
}

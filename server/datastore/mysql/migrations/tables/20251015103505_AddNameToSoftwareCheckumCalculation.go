package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251015103505, Down_20251015103505)
}

func Up_20251015103505(tx *sql.Tx) error {
	softwareStmt := `
		UPDATE software SET
			checksum = UNHEX(
			MD5(
				-- concatenate with separator \x00
				CONCAT_WS(CHAR(0),
					version,
					source,
					bundle_identifier,
					` + "`release`" + `,
					arch,
					vendor,
					browser,
					extension_id,
					name
				)
			)
		)
		WHERE source = 'apps'
		  AND bundle_identifier IS NOT NULL
		  AND bundle_identifier != ''
		`
	_, err := tx.Exec(softwareStmt)
	if err != nil {
		return fmt.Errorf("updating software checksums %w", err)
	}

	return nil
}

func Down_20251015103505(tx *sql.Tx) error {
	return nil
}

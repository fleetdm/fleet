package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251015103505, Down_20251015103505)
}

func Up_20251015103505(tx *sql.Tx) error {
	var minID, maxID sql.NullInt64
	err := tx.QueryRow(`
		SELECT MIN(id), MAX(id)
		FROM software
		WHERE source = 'apps'
		  AND bundle_identifier IS NOT NULL
		  AND bundle_identifier != ''
	`).Scan(&minID, &maxID)
	if err != nil {
		return fmt.Errorf("getting ID range: %w", err)
	}

	if !minID.Valid || !maxID.Valid {
		return nil
	}

	const batchSize = 10000
	for startID := minID.Int64; startID <= maxID.Int64; startID += batchSize {
		endID := startID + batchSize - 1
		if endID > maxID.Int64 {
			endID = maxID.Int64
		}

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
		  AND id >= ? AND id <= ?
		`
		_, err = tx.Exec(softwareStmt, startID, endID)
		if err != nil {
			return fmt.Errorf("updating software checksums (batch %d-%d): %w", startID, endID, err)
		}
	}

	return nil
}

func Down_20251015103505(tx *sql.Tx) error {
	return nil
}

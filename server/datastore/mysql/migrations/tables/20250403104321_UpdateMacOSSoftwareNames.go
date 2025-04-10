package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20250403104321, Down_20250403104321)
}

func Up_20250403104321(tx *sql.Tx) error {
	titleStmt := `UPDATE software_titles SET name = TRIM( TRAILING '.app' FROM name ) WHERE source = 'apps'`
	_, err := tx.Exec(titleStmt)
	if err != nil {
		return fmt.Errorf("updating software_titles.name: %w", err)
	}

	dupeIDsStmt := `SELECT
					s1.id AS id, s1.bundle_identifier FROM software s1
					JOIN software s2 ON s1.bundle_identifier = s2.bundle_identifier
						AND s1.version = s2.version
						AND s1.title_id = s2.title_id
						AND s1.source = s2.source
						AND s1.` + "`release`= s2.`release`" +
		`AND s1.arch = s2.arch
						AND s1.vendor = s1.vendor
						AND s1.browser = s2.browser
						AND s1.extension_id = s2.extension_id
				WHERE
					s1.source = 'apps'
				GROUP BY
					id
				HAVING
					COUNT(*) > 1`

	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	selectedIDs := make(map[string]uint)
	excludedIDs := make(map[string][]uint)
	var softwareIDs []struct {
		ID               uint   `db:"id"`
		BundleIdentifier string `db:"bundle_identifier"`
	}
	if err := txx.Select(&softwareIDs, dupeIDsStmt); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("selecting duplicate software rows: %w", err)
		}
	}

	for _, s := range softwareIDs {
		if _, ok := selectedIDs[s.BundleIdentifier]; !ok {
			selectedIDs[s.BundleIdentifier] = s.ID
			continue
		}

		excludedIDs[s.BundleIdentifier] = append(excludedIDs[s.BundleIdentifier], s.ID)
	}

	fmt.Printf("selectedIDs: %v\n", selectedIDs)
	fmt.Printf("excludedIDs: %v\n", excludedIDs)

	getRecordToUpdateStmt := `
SELECT
	hs1.host_id, hs1.software_id
FROM
	host_software hs1
WHERE
	hs1.software_id IN (?)
	AND NOT EXISTS (
		SELECT
			*
		FROM
			host_software hs2
		WHERE
			hs2.software_id = ?
			AND hs2.host_id = hs1.host_id) ORDER BY hs1.last_opened_at DESC LIMIT 1;`

	var allExcludedIDs []uint
	for bid, excluded := range excludedIDs {
		var hs struct {
			HostID     uint `db:"host_id"`
			SoftwareID uint `db:"software_id"`
		}
		selectedID, ok := selectedIDs[bid]
		if !ok {
			return fmt.Errorf("%s had excluded IDs but no selected ID", bid)
		}

		stmt, args, err := sqlx.In(getRecordToUpdateStmt, excluded, selectedID)
		if err != nil {
			return fmt.Errorf("sqlx.In for getting host software record to update for bundle_id %s: %w", bid, err)
		}

		if err := txx.Get(&hs, stmt, args...); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// if there are no rows, this means the host is already pointed at the selected software
				// ID, so no update needed
				continue
			}
			return fmt.Errorf("getting host software record to update for bundle_id %s: %w", bid, err)
		}

		_, err = tx.Exec(`UPDATE host_software SET software_id = ? WHERE host_id = ? AND software_id = ?`, selectedID, hs.HostID, hs.SoftwareID)
		if err != nil {
			return fmt.Errorf("updating host_software.software_id for bundle_id %s: %w", bid, err)
		}

		allExcludedIDs = append(allExcludedIDs, excluded...)
	}

	// at this point, every host that needs one has a pointer to the selected ID, so we can delete
	// all the records with the excluded IDs.
	deleteHostSoftwareStmt := "DELETE FROM host_software WHERE software_id IN (?)"
	deleteHostSoftwareStmt, args, err := sqlx.In(deleteHostSoftwareStmt, allExcludedIDs)
	if err != nil {
		return fmt.Errorf("sqlx.In for deleting excluded ids from host_software: %w", err)
	}

	if _, err := tx.Exec(deleteHostSoftwareStmt, args...); err != nil {
		return fmt.Errorf("deleting excluded ids from host_software")
	}

	// now we ca safely delete duplicate rows from software table
	deleteSoftwareDupesStmt := `
WITH DupSoftware AS (
	SELECT
		id,
		ROW_NUMBER() OVER (PARTITION BY bundle_identifier,
			version,
			title_id,
			source,
			` + "`release`" + `,
			arch,
			vendor,
			browser,
			extension_id ORDER BY id DESC) AS row_num
	FROM
		software
) DELETE FROM software
WHERE id IN(
	SELECT
		id FROM DupSoftware
	WHERE
		row_num > 1 AND source = 'apps')`

	if _, err := tx.Exec(deleteSoftwareDupesStmt); err != nil {
		return fmt.Errorf("deleting duplicates from software: %w", err)
	}

	// now we can update the software entries to use the new name
	softwareStmt := `
	UPDATE software SET 
		name = TRIM( TRAILING '.app' FROM name ),
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
				extension_id
			)
		)
	)
		WHERE source = 'apps'
		AND bundle_identifier IS NOT NULL
	`
	_, err = tx.Exec(softwareStmt)
	if err != nil {
		return fmt.Errorf("updating software name and checksum: %w", err)
	}

	newColStmt := `ALTER TABLE software ADD COLUMN name_source enum('basic', 'bundle_4.67') DEFAULT 'basic' NOT NULL`
	_, err = tx.Exec(newColStmt)
	if err != nil {
		return fmt.Errorf("adding name_source column to software: %w", err)
	}

	return nil
}

func Down_20250403104321(tx *sql.Tx) error {
	return nil
}

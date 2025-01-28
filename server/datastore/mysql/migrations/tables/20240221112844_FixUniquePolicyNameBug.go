package tables

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20240221112844, Down_20240221112844)
}

func Up_20240221112844(tx *sql.Tx) error {
	// The bug was that the checksum was not updating. So we need to update the checksum for existing rows.
	const updateStmt = `
	UPDATE
		policies
	SET
		checksum = UNHEX(
			MD5(
				-- concatenate with separator \x00
				CONCAT_WS(CHAR(0),
					COALESCE(team_id, ''),
					name
				)
			)
		)
	`
	const updateNameStmt = `
	UPDATE
		policies
	SET
		name = CONCAT(name, '%d'),
		checksum = UNHEX(
			MD5(
				-- concatenate with separator \x00
				CONCAT_WS(CHAR(0),
					COALESCE(team_id, ''),
					name
				)
			)
		)
	WHERE id = ?
	`
	_, err := tx.Exec(updateStmt)
	isDuplicate := func(err error) bool {
		var driverErr *mysql.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == mysqlerr.ER_DUP_ENTRY {
			return true
		}
		return false
	}

	duplicateExists := false
	if isDuplicate(err) {
		duplicateExists = true
	} else if err != nil {
		return fmt.Errorf("failed to update policies table to fill the checksum column: %w", err)
	}

	// Since there is a duplicate name somewhere in the table (should be rare), we update table 1 row at a time
	if duplicateExists {
		rows, err := tx.Query("SELECT id FROM policies")
		if err != nil {
			return fmt.Errorf("failed to query policies table: %w", err)
		}
		var ids []uint64
		defer rows.Close()
		for rows.Next() {
			var id uint64
			if err := rows.Scan(&id); err != nil {
				return fmt.Errorf("failed to scan policies table: %w", err)
			}
			ids = append(ids, id)
		}
		if rows.Err() != nil {
			return fmt.Errorf("failed during row iteration of policies table: %w", rows.Err())
		}
		for _, id := range ids {
			_, err = tx.Exec(updateStmt+" WHERE id = ?", id)
			if isDuplicate(err) {
				for i := 2; i < 10000; i++ {
					_, err = tx.Exec(fmt.Sprintf(updateNameStmt, i), id)
					if isDuplicate(err) {
						continue
					} else if err != nil {
						// We do not update this row -- it can be updated next time the policy is modified. This should be very rare.
						// Lack of update can happen if duplicate name is 255 characters, or all of the nearly 10000 names we tried are already taken.
						logger.Warn.Printf("failed to update policy id %d", id)
					}
					break
				}
			} else if err != nil {
				return fmt.Errorf("failed to update policies table to fill the checksum column on id %d: %w", id, err)
			}

		}
	}
	return nil
}

func Down_20240221112844(tx *sql.Tx) error {
	return nil
}

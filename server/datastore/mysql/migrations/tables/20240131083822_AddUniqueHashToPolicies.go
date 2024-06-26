package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240131083822, Down_20240131083822)
}

func Up_20240131083822(tx *sql.Tx) error {
	// binary(16) is the efficient way to store md5 hashes,
	// see https://dev.mysql.com/doc/refman/8.0/en/encryption-functions.html
	// We store it using UNHEX(MD5(<the string value to hash>)).
	//
	// We use md5 for consistency as we already use it in the software table and
	// for configuration profiles. So instead of using different hashing
	// algorithms, we'll stick to md5.
	//
	// This approach closely matches the one used in the software table.
	_, err := tx.Exec(`ALTER TABLE policies ADD COLUMN checksum BINARY(16) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add checksum column to policies table: %w", err)
	}

	// fill the checksum for existing rows - order of column used to generate the
	// checksum is important, we will need to use the same everywhere. The logic
	// of that computed checksum is captured in
	// mysql.policiesChecksumComputedColumn, but we don't use it here because if
	// the function's implementation changes in the future, it should not affect
	// this DB migration (e.g. the function might use columns that don't exist at
	// the point in time when this migration is run).
	_, err = tx.Exec(
		`
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
	`,
	)
	if err != nil {
		return fmt.Errorf("failed to update policies table to fill the checksum column: %w", err)
	}

	// now that every row has a checksum, make it non-nullable and unique
	_, err = tx.Exec(
		`ALTER TABLE policies
		CHANGE COLUMN checksum checksum BINARY(16) NOT NULL,
		ADD UNIQUE INDEX idx_policies_checksum (checksum)`,
	)
	if err != nil {
		return fmt.Errorf("failed to make checksum column NOT NULL and UNIQUE in policies table: %w", err)
	}

	// remove the old unique index on (name)
	_, err = tx.Exec(`ALTER TABLE policies DROP INDEX idx_policies_unique_name`)
	if err != nil {
		return fmt.Errorf("failed to drop unique index on name column in policies table: %w", err)
	}
	return nil
}

func Down_20240131083822(tx *sql.Tx) error {
	return nil
}

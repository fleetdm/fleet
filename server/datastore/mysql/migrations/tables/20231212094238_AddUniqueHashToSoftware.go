package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231212094238, Down_20231212094238)
}

func Up_20231212094238(tx *sql.Tx) error {
	// binary(16) is the efficient way to store md5 hashes,
	// see https://dev.mysql.com/doc/refman/8.0/en/encryption-functions.html
	// We store it using UNHEX(MD5(<the string value to hash>)).
	//
	// We use md5 even though it is insecure because it is used only to
	// deduplicate software entries, not for any security purpose (i.e. a
	// maliciously-crafted collision would not cause serions issues, and the
	// source of input is osquery-extracted software data), and we already use it
	// for configuration profiles so instead of using different hashing
	// algorithms, we'll stick to md5.
	_, err := tx.Exec(`ALTER TABLE software ADD COLUMN checksum BINARY(16) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add checksum column to software table: %w", err)
	}

	// fill the checksum for existing rows - order of column used to generate the
	// checksum is important, we will need to use the same everywhere. The logic
	// of that computed checksum is captured in
	// mysql.softwareChecksumComputedColumn, but we don't use it here because if
	// the function's implementation changes in the future, it should not affect
	// this DB migration (e.g. the function might use columns that don't exist at
	// the point in time when this migration is run).
	_, err = tx.Exec(`
UPDATE
	software
SET
	checksum = UNHEX(
		MD5(
			-- concatenate with separator \x00
			CONCAT_WS(CHAR(0),
				name,
				version,
				source,
				COALESCE(bundle_identifier, ''),
				` + "`release`" + `,
				arch,
				vendor,
				browser,
				extension_id
			)
		)
	)
`)
	if err != nil {
		return fmt.Errorf("failed to update software table to fill the checksum column: %w", err)
	}

	// now that every row has a checksum, make it non-nullable and unique
	_, err = tx.Exec(`ALTER TABLE software
		CHANGE COLUMN checksum checksum BINARY(16) NOT NULL,
		ADD UNIQUE INDEX idx_software_checksum (checksum)`)
	if err != nil {
		return fmt.Errorf("failed to make checksum column NOT NULL and UNIQUE in software table: %w", err)
	}
	return nil
}

func Down_20231212094238(tx *sql.Tx) error {
	return nil
}

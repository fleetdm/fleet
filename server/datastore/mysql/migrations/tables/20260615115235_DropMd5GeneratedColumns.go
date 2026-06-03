package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260615115235, Down_20260615115235)
}

// Up_20260615115235 converts the three STORED generated columns that used the
// SQL MD5() function into plain BINARY(16) columns. MySQL 9.6/9.7 LTS removed
// MD5()/SHA1() and forbid them in generated columns, so a fresh install can no
// longer create these columns. MODIFY COLUMN drops the GENERATED expression
// while MySQL keeps the existing stored bytes, so no host is flagged for
// profile re-delivery and no DDM device re-syncs on upgrade. Going forward the
// values are computed in Go at every write site.
func Up_20260615115235(tx *sql.Tx) error {
	stmts := []string{
		`ALTER TABLE mdm_apple_declarations MODIFY COLUMN token BINARY(16)`,
		`ALTER TABLE mdm_windows_configuration_profiles MODIFY COLUMN checksum BINARY(16)`,
		`ALTER TABLE mdm_android_configuration_profiles MODIFY COLUMN checksum BINARY(16)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("dropping md5 generated column expression: %q: %w", stmt, err)
		}
	}
	return nil
}

func Down_20260615115235(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20250227143904, Down_20250227143904)
}

// changeCollation changes the default collation set of the database and all
// table to the provided collation
//
// This is based on the changeCollation function that's included in this
// module and part of the 20230315104937_EnsureUniformCollation migration.
func changeCollation2025(tx *sql.Tx, charset string, collation string) (err error) {
	_, err = tx.Exec(fmt.Sprintf("ALTER DATABASE DEFAULT CHARACTER SET `%s` COLLATE `%s`", charset, collation))
	if err != nil {
		return fmt.Errorf("alter database: %w", err)
	}

	txx := sqlx.Tx{Tx: tx}
	var names []string
	err = txx.Select(&names, `
          SELECT table_name
          FROM information_schema.TABLES AS T, information_schema.COLLATION_CHARACTER_SET_APPLICABILITY AS C
          WHERE C.collation_name = T.table_collation
          AND T.table_schema = (SELECT database())
          AND (C.CHARACTER_SET_NAME != ? OR C.COLLATION_NAME != ?)
	  -- exclude tables that have columns with specific collations
	  AND table_name NOT IN ('hosts', 'enroll_secrets')`, charset, collation)
	if err != nil {
		return fmt.Errorf("selecting tables: %w", err)
	}

	// disable foreign checks before changing the collations, otherwise the
	// migration might fail. These are re-enabled after we're done.
	defer func() {
		if _, execErr := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); execErr != nil {
			err = fmt.Errorf("re-enabling foreign key checks: %w", err)
		}
	}()
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	for _, name := range names {
		_, err = tx.Exec(fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET `%s` COLLATE `%s`", name, charset, collation))
		if err != nil {
			return fmt.Errorf("alter table %s: %w", name, err)
		}
	}

	return nil
}

func Up_20250227143904(tx *sql.Tx) error {
	return changeCollation2025(tx, "utf8mb4", "utf8mb4_unicode_ci")
}

func Down_20250227143904(tx *sql.Tx) error {
	return nil
}

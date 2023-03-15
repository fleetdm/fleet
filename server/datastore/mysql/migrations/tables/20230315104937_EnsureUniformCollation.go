package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20230315104937, Down_20230315104937)
}

// changeCollation changes the default collation set of the database and all
// table to the provided collation
//
// This is based on the changeCharacterSet function that's included in this
// module and part of the 20170306075207_UseUTF8MB migration.
func changeCollation(tx *sql.Tx, charset string, collation string) error {
	_, err := tx.Exec(fmt.Sprintf("ALTER DATABASE DEFAULT CHARACTER SET `%s` COLLATE `%s`", charset, collation))
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
          AND (C.CHARACTER_SET_NAME != ? OR C.COLLATION_NAME != ?)`, charset, collation)
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
	return err
}

func Up_20230315104937(tx *sql.Tx) error {
	// while newer versions of MySQL default to
	// utf8mb4_0900_ai_ci, we still need to support 5.7, which
	// defaults to utf8mb4_general_ci
	return changeCollation(tx, "utf8mb4", "utf8mb4_general_ci")
}

func Down_20230315104937(tx *sql.Tx) error {
	return nil
}

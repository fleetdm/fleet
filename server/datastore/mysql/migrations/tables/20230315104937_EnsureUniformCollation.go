package tables

import (
	"database/sql"
	"fmt"
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
	_, err := tx.Exec(fmt.Sprintf("ALTER DATABASE DEFAULT CHARACTER SET %s COLLATE %s", charset, collation))
	if err != nil {
		return fmt.Errorf("alter database: %w", err)
	}

	rows, err := tx.Query(`
          SELECT table_name
          FROM information_schema.TABLES AS T, information_schema.COLLATION_CHARACTER_SET_APPLICABILITY AS C
          WHERE C.collation_name = T.table_collation
          AND T.table_schema = (SELECT database())
          AND (C.CHARACTER_SET_NAME != ? OR C.COLLATION_NAME != ?)`, charset, collation)
	if err != nil {
		return fmt.Errorf("selecting tables: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scanning ID: %w", err)
		}
		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("scanning rows: %w", err)
	}

	// disable foreign checks before changing the collations, otherwise the
	// migration might fail. These are re-enabled afterwards.
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	for _, name := range names {
		_, err = tx.Exec(fmt.Sprintf("ALTER TABLE %s CONVERT TO CHARACTER SET %s COLLATE %s", name, charset, collation))
		if err != nil {
			return fmt.Errorf("alter table %s: %w", name, err)
		}
	}
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return fmt.Errorf("re-enabling foreign key checks: %w", err)
	}
	return nil
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

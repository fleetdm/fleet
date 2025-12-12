package tables

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20170306075207, Down_20170306075207)
}

// changeCharacterSet changes the default character set of the database and all
// table to the provided character set
func changeCharacterSet(tx *sql.Tx, charset string) error {
	// This env var should only be set during TestCollation.
	if v := os.Getenv("FLEET_TEST_DISABLE_COLLATION_UPDATES"); v != "" {
		return nil
	}

	_, err := tx.Exec(fmt.Sprintf("ALTER DATABASE DEFAULT CHARACTER SET %s", charset))
	if err != nil {
		return errors.Wrap(err, "alter database")
	}

	rows, err := tx.Query(`
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = (SELECT database())
`)
	if err != nil {
		return errors.Wrap(err, "selecting tables")
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return errors.Wrap(err, "scanning ID")
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "scanning rows")
	}
	rows.Close()

	for _, name := range names {
		_, err = tx.Exec(fmt.Sprintf("ALTER TABLE %s CONVERT TO CHARACTER SET %s COLLATE utf8mb4_unicode_ci", name, charset))
		if err != nil {
			return errors.Wrap(err, "alter table "+name)
		}
	}
	return nil
}

func Up_20170306075207(tx *sql.Tx) error {
	return changeCharacterSet(tx, "utf8mb4")
}

func Down_20170306075207(tx *sql.Tx) error {
	return changeCharacterSet(tx, "utf8")
}

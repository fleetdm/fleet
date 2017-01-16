package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20170117025759, Down_20170117025759)
}

func Up_20170117025759(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `decorators` MODIFY `query` TEXT NOT NULL;",
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"ALTER TABLE `queries` MODIFY `query` TEXT NOT NULL;",
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"ALTER TABLE `labels` MODIFY `query` TEXT NOT NULL;",
	)
	if err != nil {
		return err
	}
	return nil
}

func Down_20170117025759(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `decorators` MODIFY `query` VARCHAR(255) NOT NULL;",
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"ALTER TABLE `queries` MODIFY `query` VARCHAR(255) NOT NULL;",
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		"ALTER TABLE `labels` MODIFY `query` VARCHAR(255) NOT NULL;",
	)
	if err != nil {
		return err
	}
	return nil
}

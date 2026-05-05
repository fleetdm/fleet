package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250815130115, Down_20250815130115)
}

func Up_20250815130115(tx *sql.Tx) error {
	stmt := `ALTER TABLE host_dep_assignments
	ADD COLUMN mdm_migration_deadline TIMESTAMP(6) DEFAULT NULL,
	ADD COLUMN mdm_migration_completed TIMESTAMP(6) DEFAULT NULL`
	_, err := tx.Exec(stmt)
	return err
}

func Down_20250815130115(tx *sql.Tx) error {
	return nil
}

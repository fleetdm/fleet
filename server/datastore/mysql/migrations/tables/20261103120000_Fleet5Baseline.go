package tables

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261103120000, Down_20261103120000)
}

const Fleet5BaselineVersion = 20261103120000

// baselineSchema is the full Fleet 5 schema and seed data: a mysqldump of a
// database after every Fleet 4 table and data migration, without the
// migration status tables. It must never change; schema changes go in new
// migrations.
//
// MySQL commits DDL implicitly, so a failed run can't be rolled back. Every
// statement is idempotent (CREATE TABLE IF NOT EXISTS, INSERT IGNORE, DROP
// VIEW IF EXISTS) so the migration can be retried instead.
//
//go:embed baseline_schema.sql
var baselineSchema string

func Up_20261103120000(tx *sql.Tx) error {
	// Fleet 4 databases are recognizable by their migration records; a failed
	// baseline run leaves none.
	txx := sqlx.Tx{Tx: tx}
	var legacyMigrations []int64
	if err := txx.Select(&legacyMigrations, `SELECT DISTINCT version_id FROM migration_status_tables WHERE version_id > 0 AND version_id < ? AND is_applied`, Fleet5BaselineVersion); err != nil {
		return fmt.Errorf("loading Fleet 4 table migrations: %w", err)
	}
	var legacyDataTable int
	if err := txx.Get(&legacyDataTable, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'migration_status_data'`); err != nil {
		return fmt.Errorf("checking for Fleet 4 data migrations table: %w", err)
	}
	if len(legacyMigrations) > 0 || legacyDataTable > 0 {
		// A 4.95.x database already has the baseline schema. Running the
		// baseline on any other Fleet 4 database would fill in missing tables
		// and leave a mix of both schemas.
		if !isFleet495Database(legacyMigrations) {
			return errors.New("this database is not on Fleet 4.95; upgrade to the latest Fleet 4.95.x before upgrading to Fleet 5")
		}
		return nil
	}

	if _, err := tx.Exec("ALTER DATABASE DEFAULT CHARACTER SET `utf8mb4` COLLATE `utf8mb4_unicode_ci`"); err != nil {
		return fmt.Errorf("setting database default collation: %w", err)
	}

	// Tables are dumped in name order, so foreign keys can reference tables
	// created later. FOREIGN_KEY_CHECKS is a session variable and the
	// connection goes back to the pool, so it must be restored.
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	if err := execBaselineStatements(tx); err != nil {
		_, _ = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
		return err
	}
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return fmt.Errorf("enabling foreign key checks: %w", err)
	}
	return nil
}

// isFleet495Database reports whether applied has every Fleet 4.95.0 migration
// and nothing newer from Fleet 4, such as a 4.96 release.
func isFleet495Database(applied []int64) bool {
	known := make(map[int64]struct{}, len(Fleet4Migrations))
	for _, v := range Fleet4Migrations {
		known[v] = struct{}{}
	}
	for _, v := range applied {
		if _, ok := known[v]; !ok {
			return false
		}
		delete(known, v)
	}
	return len(known) == 0
}

func execBaselineStatements(tx *sql.Tx) error {
	// mysqldump escapes newlines inside values, so ";\n" only ever ends a statement.
	for i, stmt := range strings.Split(baselineSchema, ";\n") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("executing baseline statement %d: %w", i+1, err)
		}
	}
	return nil
}

func Down_20261103120000(tx *sql.Tx) error {
	return nil
}

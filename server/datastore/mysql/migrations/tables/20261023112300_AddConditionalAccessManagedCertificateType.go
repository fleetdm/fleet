package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20261023112300, Down_20261023112300)
}

func Up_20261023112300(tx *sql.Tx) error {
	// Keep NULL DEFAULT NULL: certificate ingestion creates rows without a type.
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm_managed_certificates
		MODIFY COLUMN type ENUM('digicert', 'custom_scep_proxy', 'ndes', 'smallstep', 'ejbca', 'conditional_access')
		CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
		NULL DEFAULT NULL,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding conditional_access to host_mdm_managed_certificates.type: %w", err)
	}

	insStmt := `
	INSERT IGNORE INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL', 0, :created_at)
	`
	// use a constant time so that the generated schema is deterministic
	createdAt := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert for conditional access SCEP fleet variable: %w", err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("failed to insert conditional access SCEP fleet variable into fleet_variables: %w", err)
	}
	return nil
}

func Down_20261023112300(tx *sql.Tx) error {
	return nil
}

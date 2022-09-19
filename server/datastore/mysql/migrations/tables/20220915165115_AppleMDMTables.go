package tables

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed 20220915165115_AppleMDMTables_scep.sql
var scepSchema string

//go:embed 20220915165115_AppleMDMTables_nanomdm.sql
var nanoSchema string

//go:embed 20220915165115_AppleMDMTables_nanodep.sql
var depSchema string

func init() {
	MigrationClient.AddMigration(Up_20220915165115, Down_20220915165115)
}

func Up_20220915165115(tx *sql.Tx) error {
	// Apply MDM SCEP schema.
	_, err := tx.Exec(scepSchema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM SCEP schema: %w", err)
	}

	// Apply MDM Core schema.
	_, err = tx.Exec(nanoSchema)
	if err != nil {
		return fmt.Errorf("failed to apply nanomdm schema: %w", err)
	}

	// Apply MDM DEP schema.
	_, err = tx.Exec(depSchema)
	if err != nil {
		return fmt.Errorf("failed to apply nanomdm schema: %w", err)
	}

	// Add Fleet domain tables.
	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS mdm_apple_enrollments (
    id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL DEFAULT '',
    -- dep_config is NULL for manual enrollments
    dep_config JSON DEFAULT NULL,

    PRIMARY KEY (id)
);`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_apple_enrollments table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS mdm_apple_installers (
    id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL DEFAULT '',
    size BIGINT NOT NULL,
    manifest TEXT NOT NULL,
    installer LONGBLOB DEFAULT NULL,
    url_token VARCHAR(36) DEFAULT NULL,

    PRIMARY KEY (id)
);`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_apple_installers table: %w", err)
	}

	return nil
}

func Down_20220915165115(tx *sql.Tx) error {
	return nil
}

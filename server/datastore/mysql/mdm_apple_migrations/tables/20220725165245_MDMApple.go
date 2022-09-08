package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	nanodep_mysql "github.com/micromdm/nanodep/storage/mysql"
	nanomdm_mysql "github.com/micromdm/nanomdm/storage/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20220725165245, Down_20220725165245)
}

func Up_20220725165245(tx *sql.Tx) error {
	// (1) Apply MDM SCEP schema.
	_, err := tx.Exec(scep_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM SCEP schema: %w", err)
	}

	// (2) Apply MDM Core schema.
	_, err = tx.Exec(nanomdm_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM core schema: %w", err)
	}

	// (3) Apply extra tables.
	//
	// TODO(lucas): Adding them here now, but these are Fleet Apple MDM related tables.
	//
	// TODO(lucas): Does it make sense to have two tables? `mdm_apple_automatic_enrollments`
	// and `mdm_apple_manual_enrollments`
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS mdm_apple_enrollments(
		id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL DEFAULT '',
		-- dep_config is NULL for manual enrollments
		dep_config JSON DEFAULT NULL,

		PRIMARY KEY (id)
	);`)
	if err != nil {
		return fmt.Errorf("failed to create apple_enrollments: %w", err)
	}

	// (4) Apply MDM DEP schema.
	_, err = tx.Exec(nanodep_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM core schema: %w", err)
	}
	return nil
}

func Down_20220725165245(tx *sql.Tx) error {
	return nil
}

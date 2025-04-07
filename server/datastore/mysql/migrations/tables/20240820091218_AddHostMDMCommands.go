package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240820091218, Down_20240820091218)
}

func Up_20240820091218(tx *sql.Tx) error {
	if tableExists(tx, "host_mdm_commands") {
		return nil
	}

	_, err := tx.Exec(`
-- This table is used to track the MDM commands that have been sent to a host.
-- For example, if 'refetch apps' command was already sent to a host, we don't want
-- to send it again.
CREATE TABLE host_mdm_commands (
	host_id int unsigned NOT NULL,
	command_type VARCHAR(31) COLLATE utf8mb4_unicode_ci NOT NULL,
	created_at TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6),
	updated_at TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_id, command_type)
)`)
	if err != nil {
		return fmt.Errorf("failed to create table host_mdm_commands: %w", err)
	}

	return nil
}

func Down_20240820091218(_ *sql.Tx) error {
	return nil
}

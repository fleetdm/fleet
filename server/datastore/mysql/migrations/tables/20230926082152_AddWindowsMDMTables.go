package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230926082152, Down_20230926082152)
}

func Up_20230926082152(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE windows_mdm_commands (
    -- command uuid is fleet-generated, so that a command has a unique id as
    -- soon as it is enqueued. Note that a UUID is 36 chars, but for
    -- consistency with nano_mdm_commands we use 127 here.
    command_uuid    VARCHAR(127) NOT NULL,
    device_id       VARCHAR(255) NOT NULL,

    -- verb is e.g. Add, Exec, Get, etc.
    cmd_verb        VARCHAR(20)  NOT NULL,
    setting_uri     VARCHAR(255) NOT NULL,
    setting_value   VARCHAR(255) NOT NULL,
    priority        SMALLINT UNSIGNED NOT NULL DEFAULT 0,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (command_uuid, device_id)
)`)
	if err != nil {
		return fmt.Errorf("failed to create windows_mdm_commands table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE windows_mdm_command_results (
    command_uuid    VARCHAR(127) NOT NULL,
    device_id       VARCHAR(255) NOT NULL,
    session_id      VARCHAR(255) NOT NULL,
    message_id      VARCHAR(255) NOT NULL,
    -- this is the MS-MDM protocol-specific command ID, not the Fleet command UUID.
    command_id      VARCHAR(255) NOT NULL,

    cmd_verb        VARCHAR(20)  NOT NULL,
    setting_uri     VARCHAR(255) NOT NULL,
    setting_value   VARCHAR(255) NOT NULL,
    priority        SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    state           INT UNSIGNED NOT NULL DEFAULT 0,
    status          VARCHAR(31) NOT NULL,
    result_input    TEXT NULL,
    result_output   TEXT NULL,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (command_uuid, device_id),
    FOREIGN KEY fk_windows_mdm_command_results_command_uuid_device_id (command_uuid, device_id) REFERENCES windows_mdm_commands (command_uuid, device_id) ON DELETE CASCADE ON UPDATE CASCADE
)`)
	if err != nil {
		return fmt.Errorf("failed to create windows_mdm_command_results table: %w", err)
	}

	return nil
}

func Down_20230926082152(tx *sql.Tx) error {
	return nil
}

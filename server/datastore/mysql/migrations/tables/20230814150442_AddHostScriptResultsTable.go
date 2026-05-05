package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230814150442, Down_20230814150442)
}

func Up_20230814150442(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_script_results (
    id              INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
    host_id         INT(10) UNSIGNED NOT NULL,

    -- execution_id is a unique identifier (e.g. UUID) generated for each
    -- execution of a script.
    execution_id    VARCHAR(255) NOT NULL,

    -- in the future, we may have a concept of "saved scripts" and in that case
    -- the host_script_results may be associated with a script_id instead of
    -- the actual script contents. If that's the case, it may be best to allow
    -- this field to be NULL (if a saved script is used) but for now we don't
    -- support this so I'm making it NOT NULL.
    script_contents TEXT NOT NULL,

    -- output is the combination of stdout and stderr from the script execution.
    output          TEXT NOT NULL,

    -- runtime is the execution time of the script in seconds, rounded.
    runtime         INT(10) UNSIGNED NOT NULL DEFAULT 0,

    -- the exit code of the script execution, large enough to not assume too
    -- much about the possible range (e.g. https://stackoverflow.com/a/328423/1094941)
    -- It can be NULL to represent that the script results have not been received
    -- yet, and -1 if the script executed but was terminated abruptly (e.g. due to
    -- a signal/timeout, same as how Go reports this: https://pkg.go.dev/os#ProcessState.ExitCode).
    exit_code       INT(10) NULL,

    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    -- this index can be used to lookup results for a specific
    -- execution (execution ids, e.g. when updating the row for results)
    UNIQUE KEY idx_host_script_results_execution_id (execution_id),

    -- this index can be used to lookup results for a host, to check if a host is currently
		-- executing a script (by host_id and with exit_code = NULL), and an created_at condition
		-- can be added to dismiss a pending execution that's been running for too long (e.g. host
    -- was offline and never sent results, we should eventually start accepting a new
    -- script execution).
    KEY idx_host_script_results_host_exit_created (host_id, exit_code, created_at)
) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create host_script_results table: %w", err)
	}

	return nil
}

func Down_20230814150442(tx *sql.Tx) error {
	return nil
}

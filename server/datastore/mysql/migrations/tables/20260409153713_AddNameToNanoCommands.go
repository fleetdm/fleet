package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260409153713, Down_20260409153713)
}

func Up_20260409153713(tx *sql.Tx) error {
	if !columnExists(tx, "nano_commands", "name") {
		_, err := tx.Exec(`
ALTER TABLE nano_commands ADD COLUMN name varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
		if err != nil {
			return fmt.Errorf("failed to add nano_commands.name column: %w", err)
		}
	}

	// Recreate the view to include the new name column
	_, err := tx.Exec(`
		CREATE OR REPLACE SQL SECURITY INVOKER VIEW nano_view_queue AS
SELECT
    q.id COLLATE utf8mb4_unicode_ci AS id,
    q.created_at,
    q.active,
    q.priority,
    c.command_uuid COLLATE utf8mb4_unicode_ci AS command_uuid,
    c.request_type COLLATE utf8mb4_unicode_ci AS request_type,
    c.command COLLATE utf8mb4_unicode_ci AS command,
    c.name AS name,
    r.updated_at AS result_updated_at,
    r.status COLLATE utf8mb4_unicode_ci AS status,
    r.result COLLATE utf8mb4_unicode_ci AS result
FROM
    nano_enrollment_queue AS q

        INNER JOIN nano_commands AS c
        ON q.command_uuid = c.command_uuid

        LEFT JOIN nano_command_results r
        ON r.command_uuid = q.command_uuid AND r.id = q.id
ORDER BY
    q.priority DESC,
    q.created_at;
	`)
	if err != nil {
		return fmt.Errorf("failed to recreate nano_view_queue with name column: %w", err)
	}

	return nil
}

func Down_20260409153713(_ *sql.Tx) error {
	return nil
}

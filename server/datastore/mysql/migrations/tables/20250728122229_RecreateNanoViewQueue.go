package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250728122229, Down_20250728122229)
}

func Up_20250728122229(tx *sql.Tx) error {
	// c.command_uuid COLLATE utf8mb4_0900_ai_ci AS command_uuid,
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
		return fmt.Errorf("failed to recreate nano_view_queue: %w", err)
	}
	return nil
}

func Down_20250728122229(tx *sql.Tx) error {
	return nil
}

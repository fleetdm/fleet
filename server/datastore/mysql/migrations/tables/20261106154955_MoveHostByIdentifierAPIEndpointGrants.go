package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261106154955, Down_20261106154955)
}

// Up_20261106154955 moves the API-only user grants of the removed GET /hosts/identifier/:identifier to GET /hosts/:id, which now takes
// the same identifiers. INSERT IGNORE skips a user who already has both grants.
func Up_20261106154955(tx *sql.Tx) error {
	const (
		oldPath = "/api/v1/fleet/hosts/identifier/:identifier"
		newPath = "/api/v1/fleet/hosts/:id"
	)
	if _, err := tx.Exec(`
		INSERT IGNORE INTO user_api_endpoints (user_id, path, method, created_at)
		SELECT user_id, ?, method, created_at FROM user_api_endpoints WHERE path = ? AND method = 'GET'`, newPath, oldPath); err != nil {
		return fmt.Errorf("granting %s: %w", newPath, err)
	}
	if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE path = ? AND method = 'GET'`, oldPath); err != nil {
		return fmt.Errorf("removing the grants of %s: %w", oldPath, err)
	}
	return nil
}

func Down_20261106154955(tx *sql.Tx) error {
	return nil
}

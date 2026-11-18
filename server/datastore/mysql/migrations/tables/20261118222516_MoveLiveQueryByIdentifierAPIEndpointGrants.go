package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261118222516, Down_20261118222516)
}

// Up_20261118222516 moves the API-only user grants of the removed POST /hosts/identifier/:identifier/query to POST /hosts/:id/query,
// which now takes the same identifiers. INSERT IGNORE skips a user who already has both grants.
func Up_20261118222516(tx *sql.Tx) error {
	const (
		oldPath = "/api/v1/fleet/hosts/identifier/:identifier/query"
		newPath = "/api/v1/fleet/hosts/:id/query"
	)
	if _, err := tx.Exec(`
		INSERT IGNORE INTO user_api_endpoints (user_id, path, method, created_at)
		SELECT user_id, ?, method, created_at FROM user_api_endpoints WHERE path = ? AND method = 'POST'`, newPath, oldPath); err != nil {
		return fmt.Errorf("granting %s: %w", newPath, err)
	}
	if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE path = ? AND method = 'POST'`, oldPath); err != nil {
		return fmt.Errorf("removing the grants of %s: %w", oldPath, err)
	}
	return nil
}

func Down_20261118222516(tx *sql.Tx) error {
	return nil
}

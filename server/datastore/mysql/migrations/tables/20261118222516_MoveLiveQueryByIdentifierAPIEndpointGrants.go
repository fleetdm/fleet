package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration("Move live report by host identifier API endpoint grants to the host ID endpoint", Up_20261118222516, Down_20261118222516)
}

// Up_20261118222516 moves the API-only user grants of the removed POST /hosts/identifier/:identifier/query to POST /hosts/:id/query,
// which now takes the same identifiers. Grants are matched by fingerprint because the server accepts any placeholder spelling.
func Up_20261118222516(tx *sql.Tx) error {
	const (
		oldPath = "/api/v1/fleet/hosts/identifier/:identifier/query"
		newPath = "/api/v1/fleet/hosts/:id/query"
	)
	oldFingerprint := fleet.NewAPIEndpointFromTpl("POST", oldPath).Fingerprint()

	type grant struct {
		UserID uint   `db:"user_id"`
		Method string `db:"method"`
		Path   string `db:"path"`
	}
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var grants []grant
	if err := txx.Select(&grants, `SELECT user_id, method, path FROM user_api_endpoints WHERE method = 'POST'`); err != nil {
		return fmt.Errorf("select POST api endpoint grants: %w", err)
	}
	for _, g := range grants {
		if fleet.NewAPIEndpointFromTpl(g.Method, g.Path).Fingerprint() != oldFingerprint {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE user_id = ? AND method = ? AND path = ?`,
			g.UserID, g.Method, g.Path); err != nil {
			return fmt.Errorf("removing the grant of %s for user %d: %w", g.Path, g.UserID, err)
		}
		// The new path is the catalog spelling because the UI matches grants by exact path. IGNORE skips a user who already has it.
		if _, err := tx.Exec(`INSERT IGNORE INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, 'POST')`,
			g.UserID, newPath); err != nil {
			return fmt.Errorf("granting %s to user %d: %w", newPath, g.UserID, err)
		}
	}
	return nil
}

func Down_20261118222516(tx *sql.Tx) error {
	return nil
}

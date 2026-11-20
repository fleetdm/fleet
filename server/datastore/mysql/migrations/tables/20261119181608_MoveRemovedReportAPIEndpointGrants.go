package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261119181608, Down_20261119181608)
}

// Up_20261119181608 moves the API-only user grants of removed report endpoints to the endpoints that replace them. Dropping a grant
// instead could leave a user with no grants, which means full access. Grants are matched by fingerprint because a grant may name its
// path placeholders differently, and rewritten to the exact catalog path because the UI matches grants by exact path.
func Up_20261119181608(tx *sql.Tx) error {
	moves := []struct{ method, oldPath, newPath string }{
		{"GET", "/api/v1/fleet/queries/run", "/api/v1/fleet/reports/run"},
		{"DELETE", "/api/v1/fleet/reports/id/:id", "/api/v1/fleet/reports/:id"},
	}
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	for _, m := range moves {
		oldFingerprint := fleet.NewAPIEndpointFromTpl(m.method, m.oldPath).Fingerprint()
		type grant struct {
			UserID uint   `db:"user_id"`
			Path   string `db:"path"`
		}
		var grants, matches []grant
		if err := txx.Select(&grants, `SELECT user_id, path FROM user_api_endpoints WHERE method = ?`, m.method); err != nil {
			return fmt.Errorf("select %s api endpoint grants: %w", m.method, err)
		}
		for _, g := range grants {
			if fleet.NewAPIEndpointFromTpl(m.method, g.Path).Fingerprint() == oldFingerprint {
				matches = append(matches, g)
			}
		}

		for _, g := range matches {
			if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE user_id = ? AND method = ? AND path = ?`,
				g.UserID, m.method, g.Path); err != nil {
				return fmt.Errorf("removing grant %s %s of user %d: %w", m.method, g.Path, g.UserID, err)
			}
			if _, err := tx.Exec(`INSERT IGNORE INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, ?)`,
				g.UserID, m.newPath, m.method); err != nil {
				return fmt.Errorf("granting %s %s to user %d: %w", m.method, m.newPath, g.UserID, err)
			}
		}
	}
	return nil
}

func Down_20261119181608(tx *sql.Tx) error {
	return nil
}

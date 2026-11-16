package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261109155711, Down_20261109155711)
}

func Up_20261109155711(tx *sql.Tx) error {
	const (
		oldByNamePath = "/api/v1/fleet/reports/:name"
		oldByIDPath   = "/api/v1/fleet/reports/id/:id"
		newByIDPath   = "/api/v1/fleet/reports/:id"
	)
	fingerprint := func(path string) string {
		return fleet.NewAPIEndpointFromTpl("DELETE", path).Fingerprint()
	}
	// Delete by name has no replacement but delete by ID. Dropping the grant instead could leave a user with no
	// grants, which means full access. Grants are rewritten to the exact catalog paths because the UI matches them
	// by exact path, while the server ignores placeholder names.
	replacements := map[string][]string{
		fingerprint(oldByNamePath): {newByIDPath},
		fingerprint(oldByIDPath):   {oldByIDPath, newByIDPath},
	}

	type grant struct {
		UserID uint   `db:"user_id"`
		Method string `db:"method"`
		Path   string `db:"path"`
	}
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var grants []grant
	if err := txx.Select(&grants, `SELECT user_id, method, path FROM user_api_endpoints WHERE method = 'DELETE'`); err != nil {
		return fmt.Errorf("select DELETE api endpoint grants: %w", err)
	}
	var toDelete, toInsert []grant
	for _, g := range grants {
		targets, ok := replacements[fingerprint(g.Path)]
		if !ok {
			continue
		}
		toDelete = append(toDelete, g)
		for _, p := range targets {
			toInsert = append(toInsert, grant{UserID: g.UserID, Method: "DELETE", Path: p})
		}
	}

	// Delete everything before inserting: a by-name grant may have been saved with an
	// ":id" placeholder, which is the exact string of the new by-ID path.
	for _, g := range toDelete {
		if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE user_id = ? AND method = ? AND path = ?`,
			g.UserID, g.Method, g.Path); err != nil {
			return fmt.Errorf("delete old delete-report grant for user %d: %w", g.UserID, err)
		}
	}
	for _, g := range toInsert {
		if _, err := tx.Exec(`INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE user_id = user_id`,
			g.UserID, g.Path, g.Method); err != nil {
			return fmt.Errorf("insert delete-report grant for user %d: %w", g.UserID, err)
		}
	}
	return nil
}

func Down_20261109155711(tx *sql.Tx) error {
	return nil
}

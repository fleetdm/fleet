package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261106010303, Down_20261106010303)
}

func Up_20261106010303(tx *sql.Tx) error {
	const (
		oldByNamePath = "/api/v1/fleet/reports/:name"
		oldByIDPath   = "/api/v1/fleet/reports/id/:id"
		newByNamePath = "/api/v1/fleet/reports/name/:name"
		newByIDPath   = "/api/v1/fleet/reports/:id"
	)
	fingerprint := func(path string) string {
		return fleet.NewAPIEndpointFromTpl("DELETE", path).Fingerprint()
	}
	// Grants match on placeholder-normalized paths, so an untouched old by-name grant
	// would silently start granting the new by-ID path instead.
	replacements := map[string][]string{
		fingerprint(oldByNamePath): {newByNamePath},
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

func Down_20261106010303(tx *sql.Tx) error {
	return nil
}

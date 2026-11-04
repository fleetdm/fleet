package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20261104130317, Down_20261104130317)
}

func Up_20261104130317(tx *sql.Tx) error {
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
		userID uint
		method string
		path   string
	}
	rows, err := tx.Query(`SELECT user_id, method, path FROM user_api_endpoints WHERE method = 'DELETE'`)
	if err != nil {
		return fmt.Errorf("select DELETE api endpoint grants: %w", err)
	}
	var toDelete, toInsert []grant
	for rows.Next() {
		var g grant
		if err := rows.Scan(&g.userID, &g.method, &g.path); err != nil {
			rows.Close()
			return fmt.Errorf("scan api endpoint grant: %w", err)
		}
		targets, ok := replacements[fingerprint(g.path)]
		if !ok {
			continue
		}
		toDelete = append(toDelete, g)
		for _, p := range targets {
			toInsert = append(toInsert, grant{userID: g.userID, method: "DELETE", path: p})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate api endpoint grants: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close api endpoint grants rows: %w", err)
	}

	// Delete everything before inserting: a by-name grant may have been saved with an
	// ":id" placeholder, which is the exact string of the new by-ID path.
	for _, g := range toDelete {
		if _, err := tx.Exec(`DELETE FROM user_api_endpoints WHERE user_id = ? AND method = ? AND path = ?`,
			g.userID, g.method, g.path); err != nil {
			return fmt.Errorf("delete old delete-report grant for user %d: %w", g.userID, err)
		}
	}
	for _, g := range toInsert {
		if _, err := tx.Exec(`INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE user_id = user_id`,
			g.userID, g.path, g.method); err != nil {
			return fmt.Errorf("insert delete-report grant for user %d: %w", g.userID, err)
		}
	}
	return nil
}

func Down_20261104130317(tx *sql.Tx) error {
	return nil
}

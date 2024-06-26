package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240130115133, Down_20240130115133)
}

func Up_20240130115133(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE operating_systems
		ADD COLUMN os_version_id INT UNSIGNED DEFAULT NULL
		`
	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to add os_version_id column: %w", err)
	}

	// Step 2: Retrieve distinct name-version combinations and assign IDs
	type NameVersion struct {
		Name    string
		Version string
	}
	var nameVersions []NameVersion
	query := `SELECT DISTINCT name, version FROM operating_systems`
	rows, err := tx.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query distinct name and version: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var nv NameVersion
		if err := rows.Scan(&nv.Name, &nv.Version); err != nil {
			return fmt.Errorf("failed to scan name and version: %w", err)
		}
		nameVersions = append(nameVersions, nv)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over rows: %w", err)
	}

	// Step 3: Update the operating_systems table with os_version_id
	for id, nv := range nameVersions {
		updateStmt := `
			UPDATE operating_systems
			SET os_version_id = ?
			WHERE name = ? AND version = ?
		`
		if _, err := tx.Exec(updateStmt, id+1, nv.Name, nv.Version); err != nil {
			return fmt.Errorf("failed to update os_version_id for %s %s: %w", nv.Name, nv.Version, err)
		}
	}

	return nil
}

func Down_20240130115133(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20220524102918, Down_20220524102918)
}

func Up_20220524102918(tx *sql.Tx) error {
	delStmt := `
    DELETE pm
    FROM policy_membership pm
    LEFT JOIN policies p ON p.id = pm.policy_id
    LEFT JOIN hosts h ON h.id = pm.host_id
    WHERE p.team_id IS NOT NULL
      AND (p.team_id != h.team_id
           OR h.team_id IS NULL)
  `

	if _, err := tx.Exec(delStmt); err != nil {
		return fmt.Errorf("deleting orphaned policy memberships: %w", err)
	}

	return nil
}

func Down_20220524102918(tx *sql.Tx) error {
	return nil
}

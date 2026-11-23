package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261103222114, Down_20261103222114)
}

// Up_20261103222114 renames the activity types that had "team" in their name. The activity_type index keeps each UPDATE selective, and
// batching would not help because the migration runs in a single transaction.
func Up_20261103222114(tx *sql.Tx) error {
	renamed := [][2]string{
		{"created_team", "created_fleet"},
		{"deleted_team", "deleted_fleet"},
		{"applied_spec_team", "applied_spec_fleet"},
		{"changed_user_team_role", "changed_user_fleet_role"},
		{"deleted_user_team_role", "deleted_user_fleet_role"},
	}
	for _, types := range renamed {
		oldType, newType := types[0], types[1]
		if _, err := tx.Exec(`UPDATE activity_past SET activity_type = ? WHERE activity_type = ?`, newType, oldType); err != nil {
			return fmt.Errorf("renaming activity type %s to %s: %w", oldType, newType, err)
		}
	}
	return nil
}

func Down_20261103222114(tx *sql.Tx) error {
	return nil
}

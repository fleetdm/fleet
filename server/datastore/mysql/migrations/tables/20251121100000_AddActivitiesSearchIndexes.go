package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251121100000, Down_20251121100000)
}

func Up_20251121100000(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE INDEX idx_activities_user_name ON activities (user_name);
        CREATE INDEX idx_activities_user_email ON activities (user_email);
		-- ^ Individual indexes for OR conditions
		
		CREATE INDEX idx_activities_activity_type ON activities (activity_type);
		-- ^ Individual index for filtering, ORDER BY will use it's own index which already exists.

        CREATE INDEX idx_activities_type_created ON activities (activity_type, created_at);
		-- ^ Composite index for AND conditions 

		-- User table indexes
		-- Email index comes from the unique key constraint on the table
		CREATE INDEX idx_users_name ON users (name);
	`)
	return err
}

func Down_20251121100000(tx *sql.Tx) error {
	return nil
}

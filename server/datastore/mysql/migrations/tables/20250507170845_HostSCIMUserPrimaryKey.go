package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250507170845, Down_20250507170845)
}

func Up_20250507170845(tx *sql.Tx) error {
	// Step 1: Create a temporary table to store the rows we want to keep
	// (for each host_id, keep only the row with the smallest scim_user_id)
	_, err := tx.Exec(`
	CREATE TEMPORARY TABLE host_scim_user_temp AS
	SELECT host_id, MIN(scim_user_id) as scim_user_id, MIN(created_at) as created_at
	FROM host_scim_user
	GROUP BY host_id;
	`)
	if err != nil {
		return fmt.Errorf("failed to create temporary table: %s", err)
	}

	// Step 2: Drop the constraints
	_, err = tx.Exec(`
	ALTER TABLE host_scim_user
	DROP FOREIGN KEY fk_host_scim_scim_user_id,
	DROP PRIMARY KEY;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop constraints: %s", err)
	}

	// Step 3: Delete all rows from the original table
	_, err = tx.Exec(`
	DELETE FROM host_scim_user;
	`)
	if err != nil {
		return fmt.Errorf("failed to delete rows from host_scim_user: %s", err)
	}

	// Step 4: Insert the rows we want to keep back into the original table
	_, err = tx.Exec(`
	INSERT INTO host_scim_user (host_id, scim_user_id, created_at)
	SELECT host_id, scim_user_id, created_at FROM host_scim_user_temp;
	`)
	if err != nil {
		return fmt.Errorf("failed to insert rows back into host_scim_user: %s", err)
	}

	// Step 5: Add the new primary key (host_id only) and add back the foreign key constraint
	_, err = tx.Exec(`
	ALTER TABLE host_scim_user
	ADD PRIMARY KEY (host_id),
	ADD CONSTRAINT fk_host_scim_scim_user_id FOREIGN KEY (scim_user_id) REFERENCES scim_users(id) ON DELETE CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("failed to add constraints: %s", err)
	}

	// Step 6: Drop the temporary table
	_, err = tx.Exec(`
	DROP TEMPORARY TABLE IF EXISTS host_scim_user_temp;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop temporary table: %s", err)
	}

	return nil
}

func Down_20250507170845(tx *sql.Tx) error {
	// This migration cannot be safely reversed as it potentially removes data
	// However, we can restore the original schema structure

	// Step 1: Drop the foreign key constraint
	_, err := tx.Exec(`
	ALTER TABLE host_scim_user
	DROP FOREIGN KEY fk_host_scim_scim_user_id;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop foreign key constraint: %s", err)
	}

	// Step 2: Drop the primary key
	_, err = tx.Exec(`
	ALTER TABLE host_scim_user
	DROP PRIMARY KEY;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop primary key: %s", err)
	}

	// Step 3: Add back the original composite primary key
	_, err = tx.Exec(`
	ALTER TABLE host_scim_user
	ADD PRIMARY KEY (host_id, scim_user_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to add original primary key: %s", err)
	}

	// Step 4: Add back the foreign key constraint
	_, err = tx.Exec(`
	ALTER TABLE host_scim_user
	ADD CONSTRAINT fk_host_scim_scim_user_id FOREIGN KEY (scim_user_id) REFERENCES scim_users (id) ON DELETE CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("failed to add foreign key constraint: %s", err)
	}

	return nil
}

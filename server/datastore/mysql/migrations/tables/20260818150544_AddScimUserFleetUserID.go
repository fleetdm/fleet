package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260818150544, Down_20260818150544)
}

func Up_20260818150544(tx *sql.Tx) error {
	// Durable link from a SCIM user to its matching Fleet user, so
	// deprovisioning resolves the Fleet user by a stable id rather than by the
	// mutable userName/emails, which a SCIM client could change to evade
	// deprovisioning.
	if _, err := tx.Exec(`
		ALTER TABLE scim_users
			ADD COLUMN fleet_user_id INT UNSIGNED NULL,
			ADD CONSTRAINT fk_scim_users_fleet_user_id
				FOREIGN KEY (fleet_user_id) REFERENCES users (id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("add fleet_user_id to scim_users: %w", err)
	}

	// Backfill existing rows by matching current identifiers to a Fleet user by
	// email. userName is used first (it is the email in most IdP configs);
	// users.email is unique, so this matches at most one Fleet user.
	if _, err := tx.Exec(`
		UPDATE scim_users su
		JOIN users u ON u.email = su.user_name
		SET su.fleet_user_id = u.id
		WHERE su.fleet_user_id IS NULL AND su.user_name LIKE '%@%'`); err != nil {
		return fmt.Errorf("backfill scim_users.fleet_user_id by user_name: %w", err)
	}
	// Then by the emails list, but only when all matching emails resolve to a
	// single distinct Fleet user, to avoid a non-deterministic link when a SCIM
	// user's emails point at different Fleet accounts.
	if _, err := tx.Exec(`
		UPDATE scim_users su
		JOIN (
			SELECT e.scim_user_id, MIN(u.id) AS fleet_user_id
			FROM scim_user_emails e
			JOIN users u ON u.email = e.email
			GROUP BY e.scim_user_id
			HAVING COUNT(DISTINCT u.id) = 1
		) m ON m.scim_user_id = su.id
		SET su.fleet_user_id = m.fleet_user_id
		WHERE su.fleet_user_id IS NULL`); err != nil {
		return fmt.Errorf("backfill scim_users.fleet_user_id by email: %w", err)
	}

	return nil
}

func Down_20260818150544(tx *sql.Tx) error {
	return nil
}

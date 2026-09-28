package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260928170000, Down_20260928170000)
}

func Up_20260928170000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS idp_connections (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			is_default TINYINT(1) NOT NULL DEFAULT 0,
			scim_token_hash BINARY(32) NULL,
			created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			UNIQUE KEY idx_idp_connections_name (name),
			KEY idx_idp_connections_token_hash (scim_token_hash)
		) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create idp_connections: %w", err)
	}

	if _, err := tx.Exec(`INSERT IGNORE INTO idp_connections (name, is_default) VALUES ('default', 1)`); err != nil {
		return fmt.Errorf("insert default idp connection: %w", err)
	}

	var connectionID int
	if err := tx.QueryRow(`SELECT id FROM idp_connections WHERE name = 'default' LIMIT 1`).Scan(&connectionID); err != nil {
		return fmt.Errorf("lookup default idp connection: %w", err)
	}

	if err := addSCIMConnectionColumn(tx, "scim_users", connectionID); err != nil {
		return err
	}
	if err := addSCIMConnectionColumn(tx, "scim_groups", connectionID); err != nil {
		return err
	}

	// Assign updated_at explicitly so ON UPDATE NOW(6) does not mark every
	// existing row as freshly modified.
	if _, err := tx.Exec(`
		UPDATE scim_users
		SET idp_connection_id = ?, updated_at = updated_at
		WHERE idp_connection_id IS NULL`, connectionID); err != nil {
		return fmt.Errorf("backfill scim_users.idp_connection_id: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE scim_groups
		SET idp_connection_id = ?, updated_at = updated_at
		WHERE idp_connection_id IS NULL`, connectionID); err != nil {
		return fmt.Errorf("backfill scim_groups.idp_connection_id: %w", err)
	}

	if err := tightenSCIMConnectionColumn(tx, "scim_users", connectionID); err != nil {
		return err
	}
	if err := tightenSCIMConnectionColumn(tx, "scim_groups", connectionID); err != nil {
		return err
	}

	if indexExistsTx(tx, "scim_users", "idx_scim_users_user_name") {
		if _, err := tx.Exec(`ALTER TABLE scim_users DROP INDEX idx_scim_users_user_name`); err != nil {
			return fmt.Errorf("drop scim_users user_name unique index: %w", err)
		}
	}
	if !indexExistsTx(tx, "scim_users", "idx_scim_users_connection_user_name") {
		if _, err := tx.Exec(`ALTER TABLE scim_users ADD UNIQUE KEY idx_scim_users_connection_user_name (idp_connection_id, user_name)`); err != nil {
			return fmt.Errorf("add scim_users connection user_name unique index: %w", err)
		}
	}
	if indexExistsTx(tx, "scim_groups", "idx_scim_groups_display_name") {
		if _, err := tx.Exec(`ALTER TABLE scim_groups DROP INDEX idx_scim_groups_display_name`); err != nil {
			return fmt.Errorf("drop scim_groups display_name unique index: %w", err)
		}
	}
	if !indexExistsTx(tx, "scim_groups", "idx_scim_groups_connection_display_name") {
		if _, err := tx.Exec(`ALTER TABLE scim_groups ADD UNIQUE KEY idx_scim_groups_connection_display_name (idp_connection_id, display_name)`); err != nil {
			return fmt.Errorf("add scim_groups connection display_name unique index: %w", err)
		}
	}
	return nil
}

func addSCIMConnectionColumn(tx *sql.Tx, table string, connectionID int) error {
	if !columnExists(tx, table, "idp_connection_id") {
		// Nullable until the backfill below. The default keeps inserts that omit the
		// column on the org directory.
		stmt := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN idp_connection_id INT UNSIGNED NULL DEFAULT %d`, table, connectionID)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("add %s.idp_connection_id: %w", table, err)
		}
	}
	fkName := fmt.Sprintf("fk_%s_idp_connection", table)
	if !fkExists(tx, table, fkName) {
		fk := fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (idp_connection_id) REFERENCES idp_connections (id)`, table, fkName)
		if _, err := tx.Exec(fk); err != nil {
			return fmt.Errorf("add %s idp connection foreign key: %w", table, err)
		}
	}
	return nil
}

func tightenSCIMConnectionColumn(tx *sql.Tx, table string, connectionID int) error {
	stmt := fmt.Sprintf(`ALTER TABLE %s MODIFY idp_connection_id INT UNSIGNED NOT NULL DEFAULT %d`, table, connectionID)
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("require %s.idp_connection_id: %w", table, err)
	}
	return nil
}

func Down_20260928170000(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260512173249, Down_20260512173249)
}

func Up_20260512173249(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE vpp_client_users (
			id                INT UNSIGNED NOT NULL AUTO_INCREMENT,
			vpp_token_id      INT UNSIGNED NOT NULL,
			managed_apple_id  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			client_user_id    VARCHAR(36)  COLLATE utf8mb4_unicode_ci NOT NULL,
			apple_user_id     VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			status            ENUM('pending','registered','retired') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
			created_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_vpp_client_users_token_apple_id (vpp_token_id, managed_apple_id),
			UNIQUE KEY idx_vpp_client_users_token_client_user_id (vpp_token_id, client_user_id),
			CONSTRAINT fk_vpp_client_users_vpp_token_id FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating vpp_client_users table: %w", err)
	}
	return nil
}

func Down_20260512173249(tx *sql.Tx) error {
	return nil
}

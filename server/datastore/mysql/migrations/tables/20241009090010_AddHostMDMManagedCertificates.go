package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241009090010, Down_20241009090010)
}

func Up_20241009090010(tx *sql.Tx) error {
	_, err := tx.Exec(
		`CREATE TABLE host_mdm_managed_certificates (
			host_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			profile_uuid varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			challenge_retrieved_at TIMESTAMP(6) NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_uuid,profile_uuid)
		)`)
	if err != nil {
		return fmt.Errorf("failed to CREATE TABLE host_mdm_managed_certificates: %w", err)
	}
	return nil
}

func Down_20241009090010(_ *sql.Tx) error {
	return nil
}

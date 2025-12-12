package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250519170000, Down_20250519170000)
}

func Up_20250519170000(tx *sql.Tx) error {
	// microsoft_compliance_partner_integrations stores the Microsoft Compliance Partner integrations.
	// On the first version this table will only contain one row (one tenant supported for all devices in Fleet).
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS microsoft_compliance_partner_integrations (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,

		tenant_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		proxy_server_secret VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		setup_done BOOLEAN NOT NULL DEFAULT FALSE,

		created_at DATETIME(6) NULL DEFAULT NOW(6),
		updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6),

		UNIQUE KEY idx_microsoft_compliance_partner_tenant_id (tenant_id)
	)`); err != nil {
		return fmt.Errorf("failed to create microsoft_compliance_partner table: %w", err)
	}

	// microsoft_compliance_partner_host_statuses is used to track the "Device ID" and "User Principal Name"
	// of the host in Entra and the last "managed" and "compliant" statuses reported to Microsoft Intune servers.
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS microsoft_compliance_partner_host_statuses (
		host_id INT UNSIGNED NOT NULL PRIMARY KEY,

		device_id VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		user_principal_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,

		managed BOOLEAN NULL,
		compliant BOOLEAN NULL,

		created_at DATETIME(6) NULL DEFAULT NOW(6),
		updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6)
	)`); err != nil {
		return fmt.Errorf("failed to create microsoft_compliance_partner_host_statuses table: %w", err)
	}

	// Adding a new field to policies to enable/disable them for conditional access.
	_, err := tx.Exec(`ALTER TABLE policies ADD COLUMN conditional_access_enabled TINYINT(1) UNSIGNED NOT NULL DEFAULT '0'`)
	if err != nil {
		return fmt.Errorf("failed to add conditional_access_enabled to policies: %w", err)
	}

	return nil
}

func Down_20250519170000(tx *sql.Tx) error {
	return nil
}

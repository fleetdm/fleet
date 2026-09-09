package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260812134345, Down_20260812134345)
}

func Up_20260812134345(tx *sql.Tx) error {
	// mdm_microsoft_graph_credentials stores the Entra app-registration credential Fleet authenticates with when
	// calling Microsoft Graph to read Windows Autopilot device identities.
	//
	// tenant_id is UNIQUE because the Autopilot device registry is scoped to an Entra tenant and not to an
	// application: two credentials for the same tenant read an identical device list
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS mdm_microsoft_graph_credentials (
	    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	    tenant_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	    client_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	    client_secret BLOB NOT NULL,
	    -- Set by the sync when the credential fails to authenticate or is denied, cleared on the next success.
	    credential_invalid TINYINT(1) NOT NULL DEFAULT '0',
	    last_synced_at DATETIME(6) NULL DEFAULT NULL,
	    last_sync_error TEXT COLLATE utf8mb4_unicode_ci,
	    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	    PRIMARY KEY (id),
	    UNIQUE KEY idx_mdm_microsoft_graph_credentials_tenant_id (tenant_id)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_microsoft_graph_credentials table: %s", err)
	}

	// host_autopilot_devices stores the Windows-Autopilot-only per-device metadata for a host, keyed by host_id so the
	// group tag survives the pending -> enrolled transition (the host row is reused when the device enrolls). Modeled
	// on host_dep_assignments, Fleet's precedent for per-device pending metadata.
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS host_autopilot_devices (
	    host_id INT UNSIGNED NOT NULL,
	    autopilot_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    entra_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    group_tag VARCHAR(2048) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    hardware_serial VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    tenant_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	    deleted_at DATETIME(6) NULL DEFAULT NULL,
	    PRIMARY KEY (host_id),
	    KEY idx_host_autopilot_hardware_serial (hardware_serial)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("failed to create host_autopilot_devices table: %s", err)
	}

	return nil
}

func Down_20260812134345(tx *sql.Tx) error {
	return nil
}

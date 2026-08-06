package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260806212645, Down_20260806212645)
}

func Up_20260806212645(tx *sql.Tx) error {
	// mdm_microsoft_graph_credentials stores the Entra app-registration credential Fleet authenticates with when
	// calling Microsoft Graph to read Windows Autopilot device identities. It is a dedicated table rather than a field
	// on app_config_json because client_secret is a secret and must be encrypted at rest, and rather than an entry in
	// mdm_config_assets because that table is keyed UNIQUE (name, deletion_uuid) and so holds a single row per
	// credential name. ABM and VPP tokens both had to make this same move once they became multi-instance;
	// abm_tokens and vpp_tokens are the shape followed here.
	//
	// tenant_id is UNIQUE because the Autopilot device registry is scoped to an Entra tenant and not to an
	// application: two credentials for the same tenant read an identical device list, while doubling calls against
	// Graph's throttling limits and making per-tenant sync status ambiguous.
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS mdm_microsoft_graph_credentials (
	    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	    tenant_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	    client_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	    -- Encrypted with the server private key, as abm_tokens.token is.
	    client_secret BLOB NOT NULL,
	    -- Set by the sync when the credential fails to authenticate or is denied, cleared on the next success. Drives
	    -- the app-wide banner, as abm_tokens.token_invalid does.
	    credential_invalid TINYINT(1) NOT NULL DEFAULT '0',
	    last_synced_at TIMESTAMP NULL DEFAULT NULL,
	    last_sync_error TEXT COLLATE utf8mb4_unicode_ci,
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	    PRIMARY KEY (id),
	    UNIQUE KEY idx_mdm_microsoft_graph_credentials_tenant_id (tenant_id)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_microsoft_graph_credentials table: %s", err)
	}

	// host_autopilot_devices stores the Windows-Autopilot-only per-device metadata for a host, keyed by host_id so the
	// group tag survives the pending -> enrolled transition (the host row is reused when the device enrolls). Modeled
	// on host_dep_assignments, Fleet's precedent for per-device pending metadata, which is the Apple counterpart of
	// this table and likewise carries no mdm_ infix.
	//
	// autopilot_device_id is the Graph resource id of the Autopilot registration and is distinct from
	// azure_ad_device_id (the Entra device object). Reconciliation keys on autopilot_device_id because it is unique
	// and stable for the life of the registration, whereas serial numbers are neither: Graph itself paginates this
	// collection on serial, and placeholder serials such as "Default string" ship on real hardware.
	//
	// group_tag is VARCHAR(2048) because that is Intune's maximum group tag length; a narrower column would truncate
	// silently. It is deliberately NOT indexed: at utf8mb4 that is 8192 bytes, well over InnoDB's 3072-byte index key
	// limit. Filtering by group tag would need a prefix index.
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS host_autopilot_devices (
	    host_id INT UNSIGNED NOT NULL,
	    autopilot_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    azure_ad_device_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    group_tag VARCHAR(2048) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    hardware_serial VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    tenant_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	    deleted_at TIMESTAMP NULL DEFAULT NULL,
	    PRIMARY KEY (host_id),
	    KEY idx_host_autopilot_hardware_serial (hardware_serial),
	    KEY idx_host_autopilot_tenant_id (tenant_id),
	    CONSTRAINT fk_host_autopilot_devices_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("failed to create host_autopilot_devices table: %s", err)
	}

	return nil
}

func Down_20260806212645(tx *sql.Tx) error {
	return nil
}

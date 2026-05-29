package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260529084900, Down_20260529084900)
}

func Up_20260529084900(tx *sql.Tx) error {
	// host_google_cloud_identity_clientstates tracks the last-known Google Cloud
	// Identity ClientState that Fleet has PATCHed for a (host, signed-in Workspace
	// deviceUser) pair. Cardinality is per-deviceUser (not per-host) because a
	// single host can have multiple Workspace identities signed into Endpoint
	// Verification.
	//
	// The diff-against-last-reported-state semantics mirror host_conditional_access
	// (Microsoft Entra): on every osquery distributed-query result that updates
	// policy compliance, the integration loads the row, compares desired vs.
	// last_* fields, and only PATCHes Google when something changed.
	// Resolution shape: the canonical `devices/{deviceId}/deviceUsers/{deviceUserId}`
	// name is filled lazily by the sync layer's first
	// `devices.list?filter=serial_number:"{serial}" → deviceUsers.list →
	// match-by-email` flow. The osquery ingest layer only stages
	// `workspace_email` (one row per EV-resolved signed-in Workspace
	// identity on the host); the (host_id, workspace_email, partner_suffix)
	// triple uniquely identifies the ClientState Fleet emits.
	if _, err := tx.Exec(`
		CREATE TABLE host_google_cloud_identity_clientstates (
			id                      INT UNSIGNED NOT NULL AUTO_INCREMENT,
			host_id                 INT UNSIGNED NOT NULL,
			workspace_email         VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			partner_suffix          VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			device_user_resource    VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			last_compliant          TINYINT(1) DEFAULT NULL,
			last_managed            TINYINT(1) DEFAULT NULL,
			last_score_reason       VARCHAR(1024) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			last_etag               VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			last_synced_at          TIMESTAMP(6) NULL DEFAULT NULL,
			created_at              TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at              TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_hgcic_host_email_suffix (host_id, workspace_email, partner_suffix),
			KEY idx_hgcic_host (host_id),
			CONSTRAINT fk_hgcic_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("creating host_google_cloud_identity_clientstates table: %w", err)
	}
	return nil
}

func Down_20260529084900(tx *sql.Tx) error {
	return nil
}

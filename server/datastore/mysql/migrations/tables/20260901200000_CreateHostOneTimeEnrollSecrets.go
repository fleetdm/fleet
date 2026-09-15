package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260901200000, Down_20260901200000)
}

// host_one_time_enroll_secrets holds per-device, single-use enroll secrets
// A secret is bound to the device identifiers captured at mint time. A host
// has at most one unconsumed secret at a time (enforced by the minting code,
// re-deliveries of the profile hand out that same secret until it is consumed.
// Rows are retained after use (until superseded by a newer secret for the same
// host or the host is deleted) so that a later attempt
// with a spent secret can be recognized and reported rather than treated as an
// unknown secret.
func Up_20260901200000(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS host_one_time_enroll_secrets (
			id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			secret            VARCHAR(255) COLLATE utf8mb4_bin NOT NULL,
			host_id           INT UNSIGNED NULL,
			team_id           INT UNSIGNED NULL,
			platform          VARCHAR(255) NOT NULL,
			hardware_uuid     VARCHAR(255) NOT NULL,
			hardware_serial   VARCHAR(255) NOT NULL,
			created_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			consumed_at       TIMESTAMP(6) NULL,
			orbit_used_at     TIMESTAMP(6) NULL,
			osquery_used_at   TIMESTAMP(6) NULL,
			PRIMARY KEY (id),
			UNIQUE KEY idx_hotes_secret (secret),
			KEY idx_hotes_host_id (host_id),
			CONSTRAINT fk_hotes_team_id FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("creating host_one_time_enroll_secrets: %w", err)
	}
	return nil
}

func Down_20260901200000(tx *sql.Tx) error {
	return nil
}

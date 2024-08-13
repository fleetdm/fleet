package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240813103800, Down_20240813103800)
}

func Up_20240813103800(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE abm_tokens (
	id                     int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	organization_name      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	apple_id               varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	terms_expired          tinyint(1) NOT NULL DEFAULT '0',
	renew_at               timestamp NOT NULL,

	-- those team_id fields are the default teams where devices from this ABM
	-- will be enrolled during the DEP process, based on the device's platform
	-- (NULL means "no team").
	macos_default_team_id  int(10) UNSIGNED DEFAULT NULL,
	ios_default_team_id    int(10) UNSIGNED DEFAULT NULL,
	ipados_default_team_id int(10) UNSIGNED DEFAULT NULL,

	created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at             TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id),

	UNIQUE KEY idx_abm_tokens_organization_name (organization_name),

	CONSTRAINT fk_abm_tokens_macos_default_team_id
		FOREIGN KEY (macos_default_team_id) REFERENCES teams (id) ON DELETE SET NULL,
	CONSTRAINT fk_abm_tokens_ios_default_team_id
		FOREIGN KEY (ios_default_team_id) REFERENCES teams (id) ON DELETE SET NULL,
	CONSTRAINT fk_abm_tokens_ipados_default_team_id
		FOREIGN KEY (ipados_default_team_id) REFERENCES teams (id) ON DELETE SET NULL
)`)
	if err != nil {
		return fmt.Errorf("failed to create table abm_tokens: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_dep_assignments
	ADD COLUMN abm_token_id int(10) UNSIGNED NULL,
	ADD CONSTRAINT fk_host_dep_assignments_abm_token_id
		FOREIGN KEY (abm_token_id) REFERENCES abm_tokens(id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter table host_dep_assignments: %w", err)
	}
	return nil
}

func Down_20240813103800(tx *sql.Tx) error {
	return nil
}

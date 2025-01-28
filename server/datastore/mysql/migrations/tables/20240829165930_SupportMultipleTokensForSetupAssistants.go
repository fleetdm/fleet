package tables

import (
	"database/sql"
	"errors"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240829165930, Down_20240829165930)
}

func Up_20240829165930(tx *sql.Tx) error {
	// mdm_apple_default_setup_assistants will now track profile_uuids per team
	// AND ABM token.
	const alterDefaultStmt = `
	ALTER TABLE mdm_apple_default_setup_assistants
		ADD COLUMN abm_token_id int unsigned DEFAULT NULL,
		DROP KEY idx_mdm_default_setup_assistant_global_or_team_id,
		ADD CONSTRAINT idx_mdm_default_setup_assistant_global_or_team_id_abm_token_id
			UNIQUE (global_or_team_id, abm_token_id),
		ADD CONSTRAINT fk_mdm_default_setup_assistant_abm_token_id
			FOREIGN KEY (abm_token_id) REFERENCES abm_tokens(id) ON DELETE CASCADE
`
	if _, err := tx.Exec(alterDefaultStmt); err != nil {
		return fmt.Errorf("alter mdm_apple_default_setup_assistants to track per ABM token: %w", err)
	}

	var abmTokenID uint
	if err := tx.QueryRow("SELECT id FROM abm_tokens LIMIT 1").Scan(&abmTokenID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("get existing ABM token ID: %w", err)
		}
	}

	if abmTokenID > 0 {
		// there should only be one ABM token (or none) when this migration runs.
		// Timestamp is important in the code logic (it triggers a full DEP sync if
		// recently changed), so we ensure it stays the same.
		const updateDefaultStmt = `
	UPDATE mdm_apple_default_setup_assistants SET
		abm_token_id = ?,
		updated_at = updated_at
`
		if _, err := tx.Exec(updateDefaultStmt, abmTokenID); err != nil {
			return fmt.Errorf("update mdm_apple_default_setup_assistants to set abm token id: %w", err)
		}
	}
	// TODO: should we delete entries without an ABM token ID at this point? And
	// make the column NOT NULL?

	const createCustomStmt = `
CREATE TABLE mdm_apple_setup_assistant_profiles (
	id int unsigned NOT NULL AUTO_INCREMENT,

	-- the corresponding custom setup assistant in mdm_apple_setup_assistants,
	-- which is already associated with a team.
	setup_assistant_id int unsigned NOT NULL,

	-- the ABM token used to define this profile in the Apple API.
	abm_token_id int unsigned NOT NULL,

	-- the profile UUID returned by the Apple API when defined with this ABM
	-- token.
	profile_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

	created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id),
	UNIQUE KEY idx_mdm_apple_setup_assistant_profiles_asst_id_tok_id (setup_assistant_id, abm_token_id),
	CONSTRAINT fk_mdm_apple_setup_assistant_profiles_setup_assistant_id
		FOREIGN KEY (setup_assistant_id) REFERENCES mdm_apple_setup_assistants (id) ON DELETE CASCADE,
	CONSTRAINT fk_mdm_apple_setup_assistant_profiles_abm_token_id
		FOREIGN KEY (abm_token_id) REFERENCES abm_tokens (id) ON DELETE CASCADE
)
`
	if _, err := tx.Exec(createCustomStmt); err != nil {
		return fmt.Errorf("create mdm_apple_setup_assistant_profiles table: %w", err)
	}

	if abmTokenID > 0 {
		// migrate any existing profile_uuid to the new table
		const insertCustomStmt = `
		INSERT INTO mdm_apple_setup_assistant_profiles (
			setup_assistant_id,
			abm_token_id,
			profile_uuid,
			updated_at
		)
		SELECT
			mas.id,
			?,
			mas.profile_uuid,
			mas.updated_at
		FROM
			mdm_apple_setup_assistants mas
`
		if _, err := tx.Exec(insertCustomStmt, abmTokenID); err != nil {
			return fmt.Errorf("create mdm_apple_setup_assistant_profiles table: %w", err)
		}
	}

	const alterCustomStmt = `
	ALTER TABLE mdm_apple_setup_assistants
		DROP COLUMN profile_uuid
`
	if _, err := tx.Exec(alterCustomStmt); err != nil {
		return fmt.Errorf("alter mdm_apple_setup_assistants to drop profile_uuid: %w", err)
	}

	return nil
}

func Down_20240829165930(tx *sql.Tx) error {
	return nil
}

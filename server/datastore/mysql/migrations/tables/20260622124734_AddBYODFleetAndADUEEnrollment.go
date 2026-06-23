package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260622124734, Down_20260622124734)
}

func Up_20260622124734(tx *sql.Tx) error {
	// First we modify the abm_tokens table to add the column for BYOD default team id, and a unique token for ADUE enrollments
	_, err := tx.Exec(`ALTER TABLE abm_tokens
    ADD COLUMN byod_default_team_id INT UNSIGNED NULL,
    ADD COLUMN enrollment_url_token VARBINARY(64) NOT NULL DEFAULT '',
    ADD CONSTRAINT abm_tokens_byod_team_fk
        FOREIGN KEY (byod_default_team_id) REFERENCES teams(id) ON DELETE SET NULL;`)
	if err != nil {
		return fmt.Errorf("altering abm_tokens for BYOD and ADUE: %w", err)
	}
	var abmTokens []struct {
		ID uint `db:"id"`
	}
	rows, err := tx.Query(`SELECT id FROM abm_tokens`)
	if err != nil {
		return fmt.Errorf("selecting abm_tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var token struct {
			ID uint `db:"id"`
		}
		if err := rows.Scan(&token.ID); err != nil {
			return fmt.Errorf("scanning abm_tokens: %w", err)
		}
		abmTokens = append(abmTokens, token)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating abm_tokens: %w", err)
	}

	for _, token := range abmTokens {
		// Generate a unique token for ADUE enrollment
		urlEncodedToken, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
		if err != nil {
			return fmt.Errorf("generating enrollment URL token: %w", err)
		}

		_, err = tx.Exec(`UPDATE abm_tokens SET enrollment_url_token = ? WHERE id = ?`, urlEncodedToken, token.ID)
		if err != nil {
			return fmt.Errorf("updating abm_tokens with enrollment URL token: %w", err)
		}
	}

	// Drop the default, and force uniqueness so we always force a new enrollment URL token when adding an ABM token
	// We add a small constraint check to ensure the length is more than 32 bytes.
	_, err = tx.Exec(`ALTER TABLE abm_tokens 
	ALTER COLUMN enrollment_url_token DROP DEFAULT,
	ADD UNIQUE KEY idx_abm_tokens_enrollment_url_token (enrollment_url_token),
	ADD CONSTRAINT abm_tokens_enroll_url_length CHECK (LENGTH(enrollment_url_token) > 32);`)
	if err != nil {
		return fmt.Errorf("dropping default for enrollment_url_token: %w", err)
	}

	// Create ADUE enrollments table to track challenges and accompanying information
	// such as IdP account and ABM token (for default team enrollment)
	_, err = tx.Exec(`CREATE TABLE mdm_adue_enrollment_challenges (
		id               INT UNSIGNED NOT NULL AUTO_INCREMENT,
		challenge        VARBINARY(64)   NOT NULL,
		idp_account_uuid VARCHAR(255)  COLLATE utf8mb4_unicode_ci  NOT NULL,
		abm_token_id     INT UNSIGNED    NULL,
		expires_at       TIMESTAMP(6)        NOT NULL,
		used_at          TIMESTAMP(6)        NULL,
		created_at       TIMESTAMP(6)        NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		PRIMARY KEY (id),
		UNIQUE KEY idx_mdm_adue_challenge (challenge),
		KEY idx_mdm_adue_expires (expires_at),
		CONSTRAINT mdm_adue_abm_token_fk
			FOREIGN KEY (abm_token_id) REFERENCES abm_tokens(id) ON DELETE CASCADE,
		CONSTRAINT mdm_adue_idp_account_fk
			FOREIGN KEY (idp_account_uuid) REFERENCES mdm_idp_accounts(uuid) ON DELETE CASCADE
	);`)
	if err != nil {
		return fmt.Errorf("creating mdm_adue_enrollment_challenges table: %w", err)
	}

	return nil
}

func Down_20260622124734(tx *sql.Tx) error {
	return nil
}

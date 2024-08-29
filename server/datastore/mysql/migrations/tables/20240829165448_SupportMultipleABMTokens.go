package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240829165448, Down_20240829165448)
}

func Up_20240829165448(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE abm_tokens (
	id                     int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	organization_name      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	apple_id               varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	terms_expired          tinyint(1) NOT NULL DEFAULT '0',
	renew_at               timestamp NOT NULL,
	-- encrypted token, encrypted with the ABM cert and key and encrypted again
	-- with the FleetConfig.Server.PrivateKey
	token                  blob NOT NULL,

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

	// migrate the existing ABM token (if any) to the new abm_tokens table
	const getABM = `
SELECT
	value
FROM
	mdm_config_assets
WHERE
	name = ? AND deletion_uuid = ''
`
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var token []byte
	if err := txx.Get(&token, getABM, fleet.MDMAssetABMTokenDeprecated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// nothing to migrate, exit early
			return nil
		}
		return fmt.Errorf("selecting existing ABM token: %w", err)
	}

	// get the current ABM configuration from the app config
	const getABMCfg = `
SELECT
	json_value->"$.mdm"
FROM
	app_config_json
LIMIT 1
`
	var raw sql.Null[json.RawMessage]
	if err := txx.Get(&raw, getABMCfg); err != nil {
		return fmt.Errorf("select MDM config from app_config_json: %w", err)
	}

	var (
		abmTermsExpired bool
		abmDefaultTeam  string
	)
	// decode if we did get an object
	if raw.Valid && len(raw.V) > 0 && raw.V[0] == '{' {
		var config map[string]interface{}
		if err := json.Unmarshal(raw.V, &config); err != nil {
			return fmt.Errorf("unmarshal appconfig: %w", err)
		}

		if s, ok := config["apple_bm_default_team"].(string); ok {
			abmDefaultTeam = s
		}
		if b, ok := config["apple_bm_terms_expired"].(bool); ok {
			abmTermsExpired = b
		}
	}

	var defaultTeamID *uint
	if abmDefaultTeam != "" {
		// get the default team id
		var id uint
		if err := txx.Get(&id, "SELECT id FROM teams WHERE name = ?", abmDefaultTeam); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("select default ABM team id: %w", err)
			}
		}
		// only use it if the team exists
		if id > 0 {
			defaultTeamID = &id
		}
	}

	// NOTE: we don't know the organization_name, apple_id and renew_at of the
	// existing token - the only way to know is to make an Apple API call and to
	// decrypt the token, two operations that are not safe to do in a DB
	// migration. Instead, insert with empty values and we have a check in the
	// cron job ("apple_mdm_dep_profile_assigner") that runs regularly (and
	// ~immediately at Fleet startup) to ensure the token's information is
	// filled.
	// https://github.com/fleetdm/fleet/pull/21287#discussion_r1715891448

	// insert the token in the new table
	const insABM = `
INSERT INTO abm_tokens
	(
		organization_name,
		apple_id,
		terms_expired,
		renew_at,
		token,
		macos_default_team_id,
		ios_default_team_id,
		ipados_default_team_id
	)
VALUES
	('', '', ?, DATE('2000-01-01'), ?, ?, ?, ?)
`
	res, err := tx.Exec(insABM, abmTermsExpired, token, defaultTeamID, defaultTeamID, defaultTeamID)
	if err != nil {
		return fmt.Errorf("insert existing ABM token into abm_tokens: %w", err)
	}
	tokenID, _ := res.LastInsertId()

	// soft-delete the token from the deprecated storage
	const delABM = `
UPDATE
	mdm_config_assets
SET
	deleted_at = CURRENT_TIMESTAMP(),
	deletion_uuid = ?
WHERE
	name = ? AND deletion_uuid = ''
`
	deletionUUID := uuid.New().String()
	if _, err = tx.Exec(delABM, deletionUUID, fleet.MDMAssetABMTokenDeprecated); err != nil {
		return fmt.Errorf("delete ABM token from mdm_config_assets: %w", err)
	}

	// associate all existing host DEP enrollments with the existing token
	const updDEP = `
UPDATE
	host_dep_assignments
SET
	abm_token_id = ?
WHERE
	deleted_at IS NULL
`
	if _, err = tx.Exec(updDEP, tokenID); err != nil {
		return fmt.Errorf("update ABM token link in host_dep_assignments: %w", err)
	}

	return nil
}

func Down_20240829165448(tx *sql.Tx) error {
	return nil
}

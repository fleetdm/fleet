package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240829165605, Down_20240829165605)
}

func Up_20240829165605(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE vpp_tokens (
	id                     int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	organization_name      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	location               varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	renew_at               timestamp NOT NULL,
	-- encrypted token, encrypted with the FleetConfig.Server.PrivateKey value
	token                  blob NOT NULL,

	-- Note that as per a Slack discussion [1], we opted not to enforce the constraint
	-- of team uniqueness in the DB as it quickly got hairy - we'd need a distinct
	-- global_or_team_id NOT NULL to be able to enforce it along with the enum, and even
	-- then we want to allow multiple entries for a NULL team id with the enum set to 'none'
	-- (inactive tokens), and we need to handle the team deletion which only sets team_id
	-- automatically via the FK ON DELETE clause. We thought about using a trigger but
	-- decided against it given the impacts [2]. We'll instead enforce the constraint in
	-- the Go code.
	-- [1]: https://fleetdm.slack.com/archives/C03C41L5YEL/p1724073772972669
	-- [2]: https://www.percona.com/blog/how-triggers-may-significantly-affect-the-amount-of-memory-allocated-to-your-mysql-server/
	team_id                int(10) UNSIGNED DEFAULT NULL,

	-- null_team_type indicates the special team represented when team_id is NULL,
	-- which can be none (VPP token is inactive), "all teams" or the "no team" team.
	-- It is important, when setting a non-NULL team_id, to always update this field
	-- to "none" so that if the team is deleted, via the ON DELETE SET NULL constraint,
	-- the VPP token automatically becomes inactive (and not, e.g. "all teams" which
	-- would introduce data inconsistency).
	null_team_type         ENUM('none', 'allteams', 'noteam') DEFAULT 'none',

	created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at             TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id),

	UNIQUE KEY idx_vpp_tokens_location (location),

	CONSTRAINT fk_vpp_tokens_team_id
		FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL
)`)
	if err != nil {
		return fmt.Errorf("failed to create table vpp_tokens: %w", err)
	}

	_, err = tx.Exec(`
ALTER TABLE host_vpp_software_installs
	ADD COLUMN vpp_token_id int(10) UNSIGNED NULL,
	ADD CONSTRAINT fk_host_vpp_software_installs_vpp_token_id
		FOREIGN KEY (vpp_token_id) REFERENCES vpp_tokens(id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter table host_vpp_software_installs: %w", err)
	}

	// migrate the existing VPP token (if any) to the new vpp_tokens table
	const getVPP = `
SELECT
	value
FROM
	mdm_config_assets
WHERE
	name = ? AND deletion_uuid = ''
`
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var token []byte
	if err := txx.Get(&token, getVPP, fleet.MDMAssetVPPTokenDeprecated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// nothing to migrate, exit early
			return nil
		}
		return fmt.Errorf("selecting existing VPP token: %w", err)
	}

	// insert the token in the new table, defaulting to "all teams"
	const insVPP = `
INSERT INTO vpp_tokens
	(
		organization_name,
		location,
		renew_at,
		token,
		team_id,
		null_team_type
	)
VALUES
	('', '', DATE('2000-01-01'), ?, NULL, 'allteams')
`
	res, err := tx.Exec(insVPP, token)
	if err != nil {
		return fmt.Errorf("insert existing VPP token into vpp_tokens: %w", err)
	}
	tokenID, _ := res.LastInsertId()

	// soft-delete the token from the deprecated storage
	const delVPP = `
UPDATE
	mdm_config_assets
SET
	deleted_at = CURRENT_TIMESTAMP(),
	deletion_uuid = ?
WHERE
	name = ? AND deletion_uuid = ''
`
	deletionUUID := uuid.New().String()
	if _, err = tx.Exec(delVPP, deletionUUID, fleet.MDMAssetVPPTokenDeprecated); err != nil {
		return fmt.Errorf("delete VPP token from mdm_config_assets: %w", err)
	}

	// associate all existing host VPP install requests with the existing token
	const updHost = `
UPDATE
	host_vpp_software_installs
SET
	vpp_token_id = ?,
	updated_at = updated_at
`
	if _, err = tx.Exec(updHost, tokenID); err != nil {
		return fmt.Errorf("update VPP token link in host_vpp_software_installs: %w", err)
	}

	// NOTE: we can't add the token's metadata during migration because we don't
	// have the Server.PrivateKey value to decrypt the data from
	// mdm_config_assets (the VPP token, once decrypted, contains the metadata we
	// need). Enqueue a worker job to complete the migration, back-filling the
	// metadata for the migrated token.
	const (
		jobName        = "db_migration"
		taskName       = "migrate_vpp_token"
		jobStateQueued = "queued"
	)

	type migrateArgs struct {
		Task string `json:"task"`
	}
	argsJSON, err := json.Marshal(migrateArgs{Task: taskName})
	if err != nil {
		return fmt.Errorf("failed to JSON marshal the job arguments: %w", err)
	}

	// hard-coded timestamps are used so that schema.sql is stable
	const query = `
INSERT INTO jobs (
    name,
    args,
    state,
		error,
    not_before,
		created_at,
		updated_at
)
VALUES (?, ?, ?, '', ?, ?, ?)
`
	ts := time.Date(2024, 8, 19, 0, 0, 0, 0, time.UTC)
	if _, err := tx.Exec(query, jobName, argsJSON, jobStateQueued, ts, ts, ts); err != nil {
		return fmt.Errorf("failed to insert worker job: %w", err)
	}
	return nil
}

func Down_20240829165605(tx *sql.Tx) error {
	return nil
}

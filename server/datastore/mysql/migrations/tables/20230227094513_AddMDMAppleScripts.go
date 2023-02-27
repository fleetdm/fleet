package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230227094513, Down_20230227094513)
}

func Up_20230227094513(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_apple_scripts (
	id          INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	-- team_id is zero for scripts that are not associated with any team
	team_id     INT(10) UNSIGNED NOT NULL DEFAULT 0,
	name        VARCHAR(255) NOT NULL,
	-- scripts should be plain text but given the wide variety of script
	-- types, storing as blobs to be safe (they may not all be utf8 either)
	script      BLOB NOT NULL,
	created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at  TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (id),
	UNIQUE KEY idx_mdm_apple_scripts_name (team_id, name)
);`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}
	return nil
}

func Down_20230227094513(tx *sql.Tx) error {
	return nil
}

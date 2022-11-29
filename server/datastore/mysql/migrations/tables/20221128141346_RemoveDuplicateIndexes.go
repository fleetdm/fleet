package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221128141346, Down_20221128141346)
}

func Up_20221128141346(tx *sql.Tx) error {
	dropIndexes := []string{
		"ALTER TABLE app_config_json DROP INDEX id;",
		"ALTER TABLE host_users DROP INDEX idx_uid_username;",
		"ALTER TABLE policy_membership DROP INDEX idx_policy_membership_policy_id;",
		"ALTER TABLE queries DROP INDEX constraint_query_name_unique;",
		"ALTER TABLE software DROP INDEX software_listing_idx, ADD INDEX software_listing_idx (`name`);",
		"ALTER TABLE software_cve DROP INDEX software_cve_software_id;",
	}
	for _, alter := range dropIndexes {
		_, err := tx.Exec(alter)
		if err != nil {
			return errors.Wrapf(err, "create table")
		}
	}
	return nil
}

func Down_20221128141346(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230408084104, Down_20230408084104)
}

func Up_20230408084104(tx *sql.Tx) error {
	_, err := tx.Exec(
		`ALTER TABLE mdm_apple_configuration_profiles ADD COLUMN checksum BINARY(16) NOT NULL;
		 ALTER TABLE host_mdm_apple_profiles ADD COLUMN checksum BINARY(16) NOT NULL;
		 UPDATE mdm_apple_configuration_profiles SET checksum = UNHEX(MD5(mobileconfig));
		 UPDATE host_mdm_apple_profiles hmap SET checksum = (SELECT checksum FROM mdm_apple_configuration_profiles macp WHERE macp.profile_id = hmap.profile_id);`)
	return errors.Wrap(err, "add checksum column")
}

func Down_20230408084104(tx *sql.Tx) error {
	return nil
}

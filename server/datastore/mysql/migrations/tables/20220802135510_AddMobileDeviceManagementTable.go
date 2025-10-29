package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220802135510, Down_20220802135510)
}

func Up_20220802135510(tx *sql.Tx) error {
	// An mdm ID identifies a Vendor + Server URL combination (e.g. Jamf + https://company.jamfcloud.com)
	// A distinct server URL for the same vendor results in different MDM ID.
	_, err := tx.Exec(`
CREATE TABLE mobile_device_management_solutions (
  id            INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(100) NOT NULL,
  server_url    VARCHAR(255) NOT NULL,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY idx_mobile_device_management_solutions_name (name, server_url)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	// adding as NULLable to prevent costly migration for users with many hosts,
	// the mdm_id will be lazily populated as MDM query results get returned by
	// hosts.
	_, err = tx.Exec(`ALTER TABLE host_mdm ADD COLUMN mdm_id INT(10) UNSIGNED NULL;`)
	if err != nil {
		return errors.Wrapf(err, "alter table")
	}

	_, err = tx.Exec(`CREATE INDEX host_mdm_mdm_id_idx ON host_mdm (mdm_id);`)
	if err != nil {
		return errors.Wrapf(err, "create mdm id index")
	}

	// those are boolean fields, but indexing is still likely to speed things up
	// significantly, because a) we will filter on a combination of both booleans and
	// b) it's unlikely that enrolled/unenrolled ratio will be close to 50%.
	// see https://stackoverflow.com/questions/10524651/is-there-any-performance-gain-in-indexing-a-boolean-field
	_, err = tx.Exec(`CREATE INDEX host_mdm_enrolled_installed_from_dep_idx ON host_mdm (enrolled, installed_from_dep);`)
	if err != nil {
		return errors.Wrapf(err, "create enrollment status index")
	}

	return nil
}

func Down_20220802135510(tx *sql.Tx) error {
	return nil
}

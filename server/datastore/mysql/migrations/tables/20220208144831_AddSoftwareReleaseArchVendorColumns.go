package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220208144831, Down_20220208144831)
}

func Up_20220208144831(tx *sql.Tx) error {
	// NOTE(lucas): I'm using short lengths for the new varchar columns
	// due to constraints on the size of the key added below. Using 255
	// for the three new fields would fail with:
	// "Error 1071: Specified key was too long; max key length is 3072 bytes".
	//
	// We need to use "NOT NULL" because these new columns are to be included in the KEY.
	if _, err := tx.Exec("ALTER TABLE software " +
		"ADD COLUMN `release` VARCHAR(64) NOT NULL DEFAULT '', " +
		"ADD COLUMN vendor VARCHAR(32) NOT NULL DEFAULT '', " +
		"ADD COLUMN arch VARCHAR(16) NOT NULL DEFAULT ''"); err != nil {
		return errors.Wrap(err, "add new software columns")
	}

	// Delete current identifier for a software.
	currIndexName, err := indexNameByColumnName(tx, "software", "name")
	if err != nil {
		return errors.Wrap(err, "fetch current software index")
	}
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE software DROP KEY %s", currIndexName)); err != nil {
		return errors.Wrap(err, "add new software columns")
	}

	// A software piece was originally identified by name, version and source.
	//
	// We now add "vendor", "release" and "arch":
	// - release is the version of the OS this software was released on (e.g. "30.el7" for a CentOS package).
	// - vendor is the supplier of the software (e.g. "CentOS").
	// - arch is the target architecture of the software (e.g. "x86_64").
	if _, err := tx.Exec("ALTER TABLE software ADD UNIQUE KEY (name, version, source, `release`, vendor, arch)"); err != nil {
		return errors.Wrap(err, "add new index")
	}

	// Remove all software with source rpm_packages, as we will be ingesting them with new osquery
	// fields.
	//
	// Due to foreign keys, the following statement also deletes the corresponding
	// entries in `software_cpe` and `software_cve`.
	if _, err := tx.Exec("DELETE FROM software WHERE source = 'rpm_packages'"); err != nil {
		return errors.Wrap(err, "delete existing software for rpm_packages")
	}

	// Adding index to optimize software listing by source and vendor for vulnerability post-processing.
	if _, err := tx.Exec("CREATE INDEX software_source_vendor_idx ON software (source, vendor)"); err != nil {
		return errors.Wrap(err, "creating source+vendor index")
	}

	return nil
}

func Down_20220208144831(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20200707120000, Down_20200707120000)
}

func Up_20200707120000(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE `decorators`")
	if err != nil {
		return errors.Wrap(err, "drop decorators table")
	}

	_, err = tx.Exec("DROP TABLE `yara_file_paths`")
	if err != nil {
		return errors.Wrap(err, "drop yara_file_paths table")
	}

	_, err = tx.Exec("DROP TABLE `yara_signature_paths`")
	if err != nil {
		return errors.Wrap(err, "drop yara_signature_paths table")
	}

	_, err = tx.Exec("DROP TABLE `yara_signatures`")
	if err != nil {
		return errors.Wrap(err, "drop yara_signatures table")
	}

	_, err = tx.Exec("DROP TABLE `file_integrity_monitoring_files`")
	if err != nil {
		return errors.Wrap(err, "drop file_integrity_monitoring_files table")
	}

	_, err = tx.Exec("DROP TABLE `file_integrity_monitorings`")
	if err != nil {
		return errors.Wrap(err, "drop file_integrity_monitorings table")
	}

	_, err = tx.Exec("DROP TABLE `options`")
	if err != nil {
		return errors.Wrap(err, "drop file_integrity_monitorings table")
	}

	return nil
}

func Down_20200707120000(tx *sql.Tx) error {
	return nil
}

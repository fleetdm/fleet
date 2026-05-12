package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260507160833, Down_20260507160833)
}

func Up_20260507160833(tx *sql.Tx) error {
	// Allow NULL and remove the 'ndes' default on host_mdm_managed_certificates.type
	// so rows created from cert ingestion (PR 2.2) — for non-proxied flows where
	// Fleet isn't in the issuance path and doesn't know the CA type — can be
	// inserted without forcing a misleading type value. Existing rows are
	// unaffected; new INSERTs that don't specify type will get NULL instead of
	// 'ndes'. All existing INSERT call sites specify type explicitly, so removing
	// the default is safe.
	_, err := tx.Exec(`
		ALTER TABLE host_mdm_managed_certificates
		MODIFY COLUMN type ENUM('digicert', 'custom_scep_proxy', 'ndes', 'smallstep')
		CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
		NULL DEFAULT NULL
	`)
	if err != nil {
		return errors.Wrap(err, "alter host_mdm_managed_certificates.type to allow NULL")
	}
	return nil
}

func Down_20260507160833(tx *sql.Tx) error {
	return nil
}

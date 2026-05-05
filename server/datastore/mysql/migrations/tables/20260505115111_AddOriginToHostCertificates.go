package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260505115111, Down_20260505115111)
}

func Up_20260505115111(tx *sql.Tx) error {
	// Add an `origin` column tracking which ingestion source created the
	// host_certificates row. This scopes deletion semantics so each ingestion
	// source only soft-deletes rows it owns: an osquery sync that omits a row
	// inserted via MDM `CertificateList` will not delete that row, and vice
	// versa. The column is internal — not exposed in the public API.
	//
	// Existing rows default to 'osquery' since osquery has been the only
	// ingestion source until this change.
	_, err := tx.Exec(`
		ALTER TABLE host_certificates
		ADD COLUMN origin ENUM('osquery', 'mdm') NOT NULL DEFAULT 'osquery'
	`)
	if err != nil {
		return errors.Wrap(err, "add origin column to host_certificates")
	}
	return nil
}

func Down_20260505115111(tx *sql.Tx) error {
	return nil
}

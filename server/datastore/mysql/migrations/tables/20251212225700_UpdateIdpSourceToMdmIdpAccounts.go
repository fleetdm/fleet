package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251212225700, Down_20251212225700)
}

func Up_20251212225700(tx *sql.Tx) error {
	// Update existing host_emails records with source='idp' to source='mdm_idp_accounts'
	// This fixes the inconsistency where manually updated human-device mappings
	// were stored with source='idp' instead of 'mdm_idp_accounts'
	// See: https://github.com/fleetdm/fleet/issues/37168
	_, err := tx.Exec(`UPDATE host_emails SET source = 'mdm_idp_accounts' WHERE source = 'idp'`)
	if err != nil {
		return errors.Wrap(err, "update idp source to mdm_idp_accounts")
	}
	return nil
}

func Down_20251212225700(tx *sql.Tx) error {
	return nil
}

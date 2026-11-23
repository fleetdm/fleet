package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20261111220456, Down_20261111220456)
}

func Up_20261111220456(tx *sql.Tx) error {
	// Android unlock PINs are generated to fit the host's password policy, so they can be longer than macOS's 6 digits.
	var maxLength int
	if err := tx.QueryRow(`
SELECT CHARACTER_MAXIMUM_LENGTH
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'host_mdm_actions' AND COLUMN_NAME = 'unlock_pin'
`).Scan(&maxLength); err != nil {
		return errors.Wrap(err, "get host_mdm_actions.unlock_pin length")
	}
	if maxLength >= 32 {
		return nil
	}

	_, err := tx.Exec(`
ALTER TABLE host_mdm_actions
	MODIFY COLUMN unlock_pin VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
	return errors.Wrap(err, "widen host_mdm_actions.unlock_pin")
}

func Down_20261111220456(tx *sql.Tx) error {
	return nil
}

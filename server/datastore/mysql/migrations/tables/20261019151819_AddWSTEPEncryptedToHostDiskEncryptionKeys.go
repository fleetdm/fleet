package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261019151819, Down_20261019151819)
}

func Up_20261019151819(tx *sql.Tx) error {
	// host_disk_encryption_keys.updated_at auto-updates and drives the host's disk encryption status, so the backfill
	// pins it; the archive has no updated_at.
	for _, t := range []struct{ table, keepUpdatedAt string }{
		{"host_disk_encryption_keys", ", k.updated_at = k.updated_at"},
		{"host_disk_encryption_keys_archive", ""},
	} {
		// 1 marks a legacy Windows recovery key, encrypted to the WSTEP identity certificate by an older release. Keys
		// written from now on use the server private key and get the default. Rows of hosts that no longer exist stay
		// at the default because nothing reads them.
		if !columnExists(tx, t.table, "wstep_encrypted") {
			if _, err := tx.Exec(fmt.Sprintf(`
				ALTER TABLE %s
				ADD COLUMN wstep_encrypted TINYINT(1) NOT NULL DEFAULT 0
			`, t.table)); err != nil {
				return fmt.Errorf("adding wstep_encrypted to %s: %w", t.table, err)
			}
		}
		// Runs on every attempt so a retry after a partial failure still marks the rows; it is idempotent.
		if _, err := tx.Exec(fmt.Sprintf(`
			UPDATE %s k
			JOIN hosts h ON h.id = k.host_id
			SET k.wstep_encrypted = 1%s
			WHERE h.platform = 'windows' AND k.base64_encrypted != ''
		`, t.table, t.keepUpdatedAt)); err != nil {
			return fmt.Errorf("marking Windows rows in %s: %w", t.table, err)
		}
	}
	// The archive holds many rows per host and the hourly job looks for marked ones long after the last is gone.
	if !indexExistsTx(tx, "host_disk_encryption_keys_archive", "idx_host_disk_encryption_keys_archive_wstep_encrypted") {
		if _, err := tx.Exec(`
			ALTER TABLE host_disk_encryption_keys_archive
			ADD INDEX idx_host_disk_encryption_keys_archive_wstep_encrypted (wstep_encrypted, id)
		`); err != nil {
			return fmt.Errorf("adding wstep_encrypted index to host_disk_encryption_keys_archive: %w", err)
		}
	}
	return nil
}

func Down_20261019151819(tx *sql.Tx) error {
	return nil
}

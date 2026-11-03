package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261020152240, Down_20261020152240)
}

func Up_20261020152240(tx *sql.Tx) error {
	// host_disk_encryption_keys.updated_at auto-updates and drives the host's disk encryption status, so the backfill
	// pins it; the archive has no updated_at. An archived key outlives its host and can still be served to a host that
	// re-enrolls with the same hardware serial, so archive rows without a host are marked by shape: DER from
	// pkcs7.Encrypt starts with a two-byte-length SEQUENCE, whose base64 starts with "MII". A live row without a host is
	// never read and keeps the default.
	for _, t := range []struct{ table, idColumn, keepUpdatedAt, where string }{
		{"host_disk_encryption_keys", "host_id", ", k.updated_at = k.updated_at", "h.platform = 'windows'"},
		{"host_disk_encryption_keys_archive", "id", "", "(h.platform = 'windows' OR (h.id IS NULL AND k.base64_encrypted LIKE 'MII%'))"},
	} {
		// 1 marks a legacy Windows recovery key, encrypted to the WSTEP identity certificate by an older release. Keys
		// written from now on use the server private key and get the default.
		if !columnExists(tx, t.table, "wstep_encrypted") {
			if _, err := tx.Exec(fmt.Sprintf(`
				ALTER TABLE %s
				ADD COLUMN wstep_encrypted TINYINT(1) NOT NULL DEFAULT 0
			`, t.table)); err != nil {
				return fmt.Errorf("adding wstep_encrypted to %s: %w", t.table, err)
			}
		}
		// Runs on every attempt so a retry after a partial failure still marks the rows; it is idempotent. The archive
		// has several rows per host, so both backfills walk the primary key in ranges instead of one statement.
		var minID, maxID sql.NullInt64
		if err := tx.QueryRow(fmt.Sprintf(`SELECT MIN(%[1]s), MAX(%[1]s) FROM %[2]s`, t.idColumn, t.table)).Scan(&minID, &maxID); err != nil {
			return fmt.Errorf("reading %s range in %s: %w", t.idColumn, t.table, err)
		}
		const batch = 20000
		for from := minID.Int64; minID.Valid && from <= maxID.Int64; from += batch {
			if _, err := tx.Exec(fmt.Sprintf(`
				UPDATE %s k
				LEFT JOIN hosts h ON h.id = k.host_id
				SET k.wstep_encrypted = 1%s
				WHERE k.%s BETWEEN ? AND ? AND %s AND k.base64_encrypted != ''
			`, t.table, t.keepUpdatedAt, t.idColumn, t.where), from, from+batch-1); err != nil {
				return fmt.Errorf("marking Windows rows in %s: %w", t.table, err)
			}
		}
	}
	// The hourly job keeps looking for marked rows long after the last one is gone, so the lookup must cost an index
	// probe, not a scan.
	for _, idx := range []struct{ table, name, columns string }{
		{"host_disk_encryption_keys", "idx_host_disk_encryption_keys_wstep_encrypted", "wstep_encrypted, host_id"},
		{"host_disk_encryption_keys_archive", "idx_host_disk_encryption_keys_archive_wstep_encrypted", "wstep_encrypted, id"},
	} {
		if indexExistsTx(tx, idx.table, idx.name) {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s ADD INDEX %s (%s)`, idx.table, idx.name, idx.columns)); err != nil {
			return fmt.Errorf("adding %s to %s: %w", idx.name, idx.table, err)
		}
	}
	return nil
}

func Down_20261020152240(tx *sql.Tx) error {
	return nil
}

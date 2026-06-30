package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260630100331, Down_20260630100331)
}

// Up_20260630100331 soft-deletes existing osquery-origin Windows host certificate
// rows so they are re-ingested with their distinguished name (subject/issuer)
// parsed from osquery's keyed subject2/issuer2 columns.
//
// Before #31294, Windows certificates were ingested from the legacy
// subject/issuer columns and parseWindowsDN dumped the whole raw string into the
// common name, leaving country/organization/organizational unit empty.
// UpdateHostCertificates skips re-inserting a certificate whose SHA-1 and
// validity dates are unchanged, so already-stored rows would otherwise keep
// their degraded fields until the certificate is renewed. Soft-deleting them
// here makes the next ingestion treat them as new and parse them correctly.
//
// Only osquery-origin Windows rows are affected: MDM-origin certificates are
// parsed directly from the certificate (not via parseWindowsDN), and other
// platforms were never affected by the Windows DN gap. A logged-off user's
// user-scoped certificates are hidden until that user next logs in and osquery
// re-reports them (one-time transition cost). The work is done in id-keyed
// batches with progress reporting since the table can be large.
func Up_20260630100331(tx *sql.Tx) error {
	step := incrementalMigrationStep(countWindowsHostCertsToReparse, softDeleteWindowsHostCertsForReparse)
	if err := step(tx); err != nil {
		return fmt.Errorf("soft-deleting windows host certificates for re-parse: %w", err)
	}
	return nil
}

func countWindowsHostCertsToReparse(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COUNT(*)
		FROM host_certificates hc
		JOIN hosts h ON h.id = hc.host_id
		WHERE h.platform = 'windows' AND hc.origin = 'osquery' AND hc.deleted_at IS NULL`).Scan(&total)
	return total, err
}

// softDeleteWindowsHostCertsForReparse walks the osquery-origin Windows host
// certificates in id-keyed batches and soft-deletes each one, calling increment
// per row so progress is reported. The deleted_at IS NULL filter (plus the
// strictly increasing id cursor) makes the walk safe and resumable.
func softDeleteWindowsHostCertsForReparse(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	const batchSize = 1000
	var lastID uint64
	for {
		var ids []uint64
		if err := txx.Select(&ids, `
			SELECT hc.id
			FROM host_certificates hc
			JOIN hosts h ON h.id = hc.host_id
			WHERE h.platform = 'windows' AND hc.origin = 'osquery' AND hc.deleted_at IS NULL AND hc.id > ?
			ORDER BY hc.id
			LIMIT ?`, lastID, batchSize); err != nil {
			return fmt.Errorf("selecting windows host certs batch after id %d: %w", lastID, err)
		}
		if len(ids) == 0 {
			return nil
		}

		query, args, err := sqlx.In(`UPDATE host_certificates SET deleted_at = NOW(6) WHERE id IN (?)`, ids)
		if err != nil {
			return fmt.Errorf("building soft-delete query: %w", err)
		}
		if _, err := txx.Exec(query, args...); err != nil {
			return fmt.Errorf("soft-deleting windows host certs batch after id %d: %w", lastID, err)
		}

		for range ids {
			increment()
		}
		lastID = ids[len(ids)-1]
	}
}

func Down_20260630100331(tx *sql.Tx) error {
	return nil
}

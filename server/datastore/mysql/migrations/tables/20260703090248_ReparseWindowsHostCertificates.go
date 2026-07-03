package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260703090248, Down_20260703090248)
}

// Up_20260630100331 soft-deletes existing osquery-origin Windows host certificate rows so they are re-ingested with
// their distinguished name (subject/issuer) parsed from osquery's keyed subject2/issuer2 columns.
func Up_20260703090248(tx *sql.Tx) error {
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

// softDeleteWindowsHostCertsForReparse walks the osquery-origin Windows host certificates in id-keyed batches and
// soft-deletes each one, calling increment per row so progress is reported.
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

func Down_20260703090248(tx *sql.Tx) error {
	return nil
}

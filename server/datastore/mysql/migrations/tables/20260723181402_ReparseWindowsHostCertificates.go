package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260723181402, Down_20260723181402)
}

// Up_20260723181402 soft-deletes existing osquery-origin Windows host certificate rows so they are re-ingested with
// their distinguished name (subject/issuer) parsed from osquery's keyed subject2/issuer2 columns.
func Up_20260723181402(tx *sql.Tx) error {
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

// softDeleteWindowsHostCertsForReparse collects Windows host IDs once upfront,
// then soft-deletes their certificate rows in batches using only the
// host_certificates primary key. This avoids re-joining the hosts table on
// every batch, which was causing ~14 minutes of sustained full DB load at
// 100K hosts (#50228).
func softDeleteWindowsHostCertsForReparse(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Step 1: collect all Windows host IDs in one query (bounded by host count, not cert count).
	var windowsHostIDs []uint64
	if err := txx.Select(&windowsHostIDs, `SELECT id FROM hosts WHERE platform = 'windows'`); err != nil {
		return fmt.Errorf("selecting windows host ids: %w", err)
	}
	if len(windowsHostIDs) == 0 {
		return nil
	}

	// Step 2: soft-delete certs in batches of host IDs. Each batch UPDATE
	// hits the host_certificates index on (host_id) without joining hosts.
	const hostBatchSize = 500
	for i := 0; i < len(windowsHostIDs); i += hostBatchSize {
		end := i + hostBatchSize
		if end > len(windowsHostIDs) {
			end = len(windowsHostIDs)
		}
		batch := windowsHostIDs[i:end]

		query, args, err := sqlx.In(`
			UPDATE host_certificates
			SET deleted_at = NOW(6)
			WHERE host_id IN (?) AND origin = 'osquery' AND deleted_at IS NULL`, batch)
		if err != nil {
			return fmt.Errorf("building soft-delete query for host batch: %w", err)
		}
		result, err := txx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("soft-deleting certs for host batch starting at index %d: %w", i, err)
		}
		affected, _ := result.RowsAffected()
		for range affected {
			increment()
		}
	}
	return nil
}

func Down_20260723181402(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260826000000, Down_20260826000000)
}

// canonicalSoftwareChecksum is fleet.Software.ComputeRawChecksum expressed in SQL.
const canonicalSoftwareChecksum = "UNHEX(MD5(CONCAT_WS(CHAR(0), " +
	"version, source, COALESCE(bundle_identifier, ''), `release`, arch, vendor, " +
	"extension_for, extension_id, name, " +
	"NULLIF(COALESCE(application_id, ''), ''), NULLIF(COALESCE(upgrade_code, ''), ''))))"

// Vars so tests can lower them to exercise the batching.
var (
	dedupeSoftwareBatchSize    = 1000
	recomputeSoftwareBatchSize = uint64(50000)
)

func Up_20260826000000(tx *sql.Tx) error {
	// v4.76.0 moved `name` from the front to the back of the software checksum. Rows
	// created before that upgrade kept their old checksum, so when a host reported the
	// same software again it no longer matched and a second `software` row was inserted:
	// the same software twice, with the same CVEs but split host counts. Merge those
	// duplicates, then recompute every checksum so pre-upgrade rows that have not
	// duplicated yet cannot duplicate later.
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// A window function maps duplicates to their survivor in one pass; a GROUP BY would
	// have to join its result back, which has no index. Partitioning by the canonical
	// checksum rather than the columns it hashes sorts a 16-byte binary key instead of
	// 11 utf8mb4 strings (~8x faster), and matches exactly the equivalence the recompute
	// below must not collide under (CONCAT_WS skips NULLs, so distinct column tuples can
	// share a checksum).
	var mappings []struct {
		StaleID    uint64 `db:"stale_id"`
		SurvivorID uint64 `db:"survivor_id"`
	}
	if err := txx.Select(&mappings, `
		SELECT stale_id, survivor_id FROM (
			SELECT id AS stale_id,
				MIN(id) OVER (PARTITION BY `+canonicalSoftwareChecksum+`) AS survivor_id
			FROM software
		) m WHERE stale_id <> survivor_id`); err != nil {
		return fmt.Errorf("selecting duplicate software rows: %w", err)
	}

	// Instances with no duplicates skip this entirely rather than scan the large host
	// software tables for nothing.
	for start := 0; start < len(mappings); start += dedupeSoftwareBatchSize {
		end := min(start+dedupeSoftwareBatchSize, len(mappings))
		batch := mappings[start:end]

		// One CASE per batch, so each table below is touched once instead of once per
		// duplicate.
		var whens strings.Builder
		caseArgs := make([]any, 0, len(batch)*2)
		staleIDs := make([]any, 0, len(batch))
		for _, m := range batch {
			whens.WriteString(" WHEN ? THEN ?")
			caseArgs = append(caseArgs, m.StaleID, m.SurvivorID)
			staleIDs = append(staleIDs, m.StaleID)
		}
		mapCase := func(col string) string { return "CASE " + col + whens.String() + " END" }

		// Give the survivor a link for every host on a duplicate row. INSERT IGNORE skips
		// hosts already linked to it; the duplicate links are deleted below.
		stmt, args, err := sqlx.In(
			"INSERT IGNORE INTO host_software (host_id, software_id, last_opened_at) "+
				"SELECT host_id, "+mapCase("software_id")+", last_opened_at "+
				"FROM host_software WHERE software_id IN (?)",
			append(append([]any{}, caseArgs...), staleIDs)...)
		if err != nil {
			return fmt.Errorf("building host_software repoint: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("repointing host_software onto surviving software rows: %w", err)
		}

		// host_software_installed_paths is only indexed on (host_id, software_id), so
		// filtering it by software_id alone scans this large table. Drive off host_software
		// instead, which is indexed on software_id and still holds the duplicate links
		// here, to reach each path row by the full key. STRAIGHT_JOIN and FORCE INDEX pin
		// that plan, since mid-migration index statistics are unreliable.
		//
		// This skips a path row whose host_software link is already missing, which only
		// happens when a host's paths were not reconciled after its software was (separate
		// transactions). Such a row is already stale: nothing joining software can see it,
		// and the host's next detail query deletes it.
		stmt, args, err = sqlx.In(
			"UPDATE host_software hs "+
				"STRAIGHT_JOIN host_software_installed_paths hsip FORCE INDEX (host_id_software_id_idx) "+
				"  ON hsip.host_id = hs.host_id AND hsip.software_id = hs.software_id "+
				"SET hsip.software_id = "+mapCase("hs.software_id")+
				" WHERE hs.software_id IN (?)",
			append(append([]any{}, caseArgs...), staleIDs)...)
		if err != nil {
			return fmt.Errorf("building host_software_installed_paths repoint: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("repointing host_software_installed_paths onto surviving software rows: %w", err)
		}

		// The counts are recomputed by the software host-count crons, and deleting a
		// software row cascades software_cpe.
		for _, del := range []struct{ desc, query string }{
			{"host_software links", "DELETE FROM host_software WHERE software_id IN (?)"},
			{"kernel host counts", "DELETE FROM kernel_host_counts WHERE software_id IN (?)"},
			{"software CVEs", "DELETE FROM software_cve WHERE software_id IN (?)"},
			{"software host counts", "DELETE FROM software_host_counts WHERE software_id IN (?)"},
			{"software rows", "DELETE FROM software WHERE id IN (?)"},
		} {
			stmt, args, err := sqlx.In(del.query, staleIDs)
			if err != nil {
				return fmt.Errorf("building duplicate %s delete: %w", del.desc, err)
			}
			if _, err := tx.Exec(stmt, args...); err != nil {
				return fmt.Errorf("deleting duplicate %s: %w", del.desc, err)
			}
		}
	}

	// The merge made canonical checksums unique, so this cannot collide on
	// idx_software_checksum.
	// Batched by primary key to keep each statement bounded on this large table.
	var maxID uint64
	if err := tx.QueryRow("SELECT COALESCE(MAX(id), 0) FROM software").Scan(&maxID); err != nil {
		return fmt.Errorf("getting max software id: %w", err)
	}
	for start := uint64(0); start <= maxID; start += recomputeSoftwareBatchSize {
		if _, err := tx.Exec(
			"UPDATE software SET checksum = "+canonicalSoftwareChecksum+
				" WHERE id >= ? AND id < ? AND checksum <> "+canonicalSoftwareChecksum,
			start, start+recomputeSoftwareBatchSize,
		); err != nil {
			return fmt.Errorf("recomputing software checksums: %w", err)
		}
	}

	return nil
}

func Down_20260826000000(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260810160000, Down_20260810160000)
}

// canonicalSoftwareChecksum is fleet.Software.ComputeRawChecksum expressed in SQL.
const canonicalSoftwareChecksum = "UNHEX(MD5(CONCAT_WS(CHAR(0), " +
	"version, source, COALESCE(bundle_identifier, ''), `release`, arch, vendor, " +
	"extension_for, extension_id, name, " +
	"NULLIF(COALESCE(application_id, ''), ''), NULLIF(COALESCE(upgrade_code, ''), ''))))"

// dedupeSoftwareScope limits the repair to the software the affected customer reported
// duplicated (giflib from Homebrew, all versions). Other rows the checksum reorder may
// have left stale are deliberately out of scope.
const dedupeSoftwareScope = "name = 'giflib' AND source = 'homebrew_packages'"

// softwareChecksumMapping is a duplicate software row and the row it merges into.
type softwareChecksumMapping struct {
	StaleID    uint64 `db:"stale_id"`
	SurvivorID uint64 `db:"survivor_id"`
}

// Var so tests can lower it to exercise the batching.
var dedupeSoftwareBatchSize = 1000

func Up_20260810160000(tx *sql.Tx) error {
	// v4.76.0 moved `name` from the front to the back of the software checksum. Rows
	// created before that upgrade kept their old checksum, so when a host reported the
	// same software again it no longer matched and a second `software` row was inserted:
	// the same software twice, with the same CVEs but split host counts. Merge those
	// duplicates, then recompute the scoped rows' checksums so ones that have not
	// duplicated yet cannot duplicate later.
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Map each duplicate row to its survivor. The scope filter is served by
	// idx_sw_name_source_browser, so only the handful of matching rows are read; FORCE
	// INDEX pins that plan, since mid-migration index statistics are unreliable and
	// `software` is too large to risk a fallback scan. Rows are duplicates when the
	// canonical checksum formula agrees, which is exactly the equivalence the recompute
	// below must not collide under (CONCAT_WS skips NULLs, so distinct column tuples can
	// share a checksum).
	var mappings []softwareChecksumMapping
	if err := txx.Select(&mappings, `
		SELECT stale_id, survivor_id FROM (
			SELECT id AS stale_id,
				MIN(id) OVER (PARTITION BY `+canonicalSoftwareChecksum+`) AS survivor_id
			FROM software FORCE INDEX (idx_sw_name_source_browser)
			WHERE `+dedupeSoftwareScope+`
		) m WHERE stale_id <> survivor_id`); err != nil {
		return fmt.Errorf("selecting duplicate software rows: %w", err)
	}

	logger.Info.Printf("deduplicating software: found %d duplicate software rows\n", len(mappings))

	if err := mergeDuplicateSoftware(tx, mappings); err != nil {
		return err
	}

	// Recompute the scoped rows' checksums to the current formula. The merge made
	// canonical checksums unique within the scope, and the formula hashes the name, so
	// this cannot collide on idx_software_checksum.
	res, err := tx.Exec(
		"UPDATE software FORCE INDEX (idx_sw_name_source_browser) SET checksum = " + canonicalSoftwareChecksum +
			" WHERE " + dedupeSoftwareScope + " AND checksum <> " + canonicalSoftwareChecksum)
	if err != nil {
		return fmt.Errorf("recomputing software checksums: %w", err)
	}
	recomputed, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("counting recomputed software checksums: %w", err)
	}
	logger.Info.Printf("deduplicating software: recomputed %d software checksums\n", recomputed)

	return nil
}

// mergeDuplicateSoftware repoints every host reference from each duplicate row onto its
// survivor, then deletes the duplicates. Instances with no duplicates never touch the
// large host software tables.
func mergeDuplicateSoftware(tx *sql.Tx, mappings []softwareChecksumMapping) error {
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
	return nil
}

func Down_20260810160000(tx *sql.Tx) error {
	return nil
}

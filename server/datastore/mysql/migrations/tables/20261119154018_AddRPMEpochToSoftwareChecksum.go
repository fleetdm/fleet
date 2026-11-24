package tables

import (
	"crypto/md5" //nolint:gosec // software checksums are a lookup key, not a security control
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20261119154018, Down_20261119154018)
}

var rpmEpochChecksumBatchSize = 1000

// Up_20261119154018 adds the stored RPM epoch to the checksum of the software rows that have one,
// matching what ingestion now computes, so hosts keep linking to the same software IDs. The new
// checksum is computed from the stored one and the epoch, so no other column is read.
func Up_20261119154018(tx *sql.Tx) error {
	var lastID uint64
	var updated int
	for {
		batch, err := selectSoftwareWithEpoch20261119154018(tx, lastID)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}

		caseArgs := make([]any, 0, len(batch)*2)
		ids := make([]any, 0, len(batch))
		for _, r := range batch {
			caseArgs = append(caseArgs, r.id, softwareChecksumWithEpoch20261119154018(r.checksum, r.epoch))
			ids = append(ids, r.id)
		}
		//nolint:gosec // only placeholders are formatted in, no input
		stmt := fmt.Sprintf(`UPDATE software SET checksum = CASE id %sEND WHERE id IN (%s)`,
			strings.Repeat("WHEN ? THEN ? ", len(ids)), strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","))
		if _, err := tx.Exec(stmt, append(caseArgs, ids...)...); err != nil {
			return fmt.Errorf("update software checksums: %w", err)
		}
		updated += len(ids)
		lastID = batch[len(batch)-1].id
	}

	if updated > 0 {
		logger.Info.Printf("Added the RPM epoch to the checksum of %d software rows", updated)
	}
	return nil
}

type softwareWithEpoch20261119154018 struct {
	id       uint64
	checksum []byte
	epoch    uint32
}

func selectSoftwareWithEpoch20261119154018(tx *sql.Tx, afterID uint64) ([]softwareWithEpoch20261119154018, error) {
	rows, err := tx.Query(
		"SELECT s.id, s.checksum, sm.epoch "+
			"FROM software_metadata sm STRAIGHT_JOIN software s ON s.id = sm.software_id "+
			"WHERE sm.software_id > ? AND sm.epoch > 0 ORDER BY sm.software_id LIMIT ?",
		afterID, rpmEpochChecksumBatchSize)
	if err != nil {
		return nil, fmt.Errorf("select software with an epoch: %w", err)
	}
	defer rows.Close()

	var batch []softwareWithEpoch20261119154018
	for rows.Next() {
		var r softwareWithEpoch20261119154018
		if err := rows.Scan(&r.id, &r.checksum, &r.epoch); err != nil {
			return nil, fmt.Errorf("scan software with an epoch: %w", err)
		}
		batch = append(batch, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate software with an epoch: %w", err)
	}
	return batch, nil
}

// softwareChecksumWithEpoch20261119154018 is how fleet.Software.ComputeRawChecksum adds the epoch
// to the checksum of the software's other fields as of this migration.
func softwareChecksumWithEpoch20261119154018(checksum []byte, epoch uint32) []byte {
	sum := md5.Sum([]byte(hex.EncodeToString(checksum) + "epoch:" + strconv.FormatUint(uint64(epoch), 10))) //nolint:gosec // see import
	return sum[:]
}

func Down_20261119154018(tx *sql.Tx) error {
	return nil
}

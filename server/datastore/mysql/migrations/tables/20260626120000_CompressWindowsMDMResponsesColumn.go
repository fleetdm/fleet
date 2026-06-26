package tables

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260626120000, Down_20260626120000)
}

// Up_20260626120000 converts windows_mdm_responses.raw_response (a MEDIUMTEXT holding the plaintext SyncML envelope) into a new
// raw_response_gz MEDIUMBLOB that stores the envelope gzip-compressed. This shrinks the row and the redo-log/commit-quorum pressure of the
// Windows MDM check-in hot path (issue #44188). The text column could not hold raw gzip bytes (charset-constrained), which previously forced
// a base64 wrapper; the blob column removes that overhead entirely.
//
// The new column is added and the old one dropped via ALGORITHM=INSTANT (metadata-only, supported on Fleet's MySQL 8.0.44+ floor), so the
// only work proportional to table size is the backfill. The backfill runs in id-keyed batches and reports progress so operators are not
// staring at a frozen console; the server is down for migrations, so there is no concurrent write contention. Each step is guarded by a
// column-existence check so a migration that is interrupted (DDL implicitly commits in MySQL) resumes cleanly on the next run.
func Up_20260626120000(tx *sql.Tx) error {
	if !columnExists(tx, "windows_mdm_responses", "raw_response_gz") {
		if _, err := tx.Exec(`ALTER TABLE windows_mdm_responses ADD COLUMN raw_response_gz MEDIUMBLOB NULL, ALGORITHM=INSTANT`); err != nil {
			return fmt.Errorf("adding raw_response_gz column: %w", err)
		}
	}

	// Only backfill and drop while the legacy column is still present. Once it is gone the conversion is complete, so a re-run is a no-op.
	if columnExists(tx, "windows_mdm_responses", "raw_response") {
		backfill := incrementalMigrationStep(
			func(tx *sql.Tx) (uint64, error) {
				var total uint64
				err := tx.QueryRow(`SELECT COUNT(*) FROM windows_mdm_responses WHERE raw_response_gz IS NULL`).Scan(&total)
				return total, err
			},
			backfillWindowsMDMResponsesGz,
		)
		if err := backfill(tx); err != nil {
			return fmt.Errorf("backfilling raw_response_gz: %w", err)
		}

		if _, err := tx.Exec(`ALTER TABLE windows_mdm_responses DROP COLUMN raw_response, ALGORITHM=INSTANT`); err != nil {
			return fmt.Errorf("dropping raw_response column: %w", err)
		}
	}

	return nil
}

// backfillWindowsMDMResponsesGz gzip-compresses each existing plaintext raw_response into raw_response_gz, walking the table in id-keyed
// batches. The raw_response_gz IS NULL filter makes the walk resumable: rows already converted by an interrupted run are skipped.
func backfillWindowsMDMResponsesGz(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	const batchSize = 500
	var lastID uint64
	for {
		var batch []struct {
			ID          uint64 `db:"id"`
			RawResponse []byte `db:"raw_response"`
		}
		if err := txx.Select(&batch,
			`SELECT id, raw_response FROM windows_mdm_responses
			 WHERE id > ? AND raw_response_gz IS NULL
			 ORDER BY id LIMIT ?`, lastID, batchSize); err != nil {
			return fmt.Errorf("selecting batch starting after id %d: %w", lastID, err)
		}
		if len(batch) == 0 {
			return nil
		}

		for _, row := range batch {
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			if _, err := gw.Write(row.RawResponse); err != nil {
				return fmt.Errorf("gzip-compressing response id %d: %w", row.ID, err)
			}
			if err := gw.Close(); err != nil {
				return fmt.Errorf("closing gzip writer for response id %d: %w", row.ID, err)
			}
			if _, err := txx.Exec(`UPDATE windows_mdm_responses SET raw_response_gz = ? WHERE id = ?`, buf.Bytes(), row.ID); err != nil {
				return fmt.Errorf("storing compressed response id %d: %w", row.ID, err)
			}
			increment()
		}
		lastID = batch[len(batch)-1].ID
	}
}

func Down_20260626120000(tx *sql.Tx) error {
	return nil
}

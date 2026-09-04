package tables

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260904125235, Down_20260904125235)
}

func Up_20260904125235(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS nano_seen_times (
			id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			seen_time timestamp NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("creating nano_seen_times: %w", err)
	}

	if columnExists(tx, "nano_enrollments", "last_seen_at") {
		backfill := incrementalMigrationStep(
			func(tx *sql.Tx) (uint64, error) {
				var total uint64
				err := tx.QueryRow(`SELECT COUNT(*) FROM nano_enrollments`).Scan(&total)
				return total, err
			},
			backfillNanoSeenTimes,
		)
		if err := backfill(tx); err != nil {
			return fmt.Errorf("backfilling nano_seen_times: %w", err)
		}

		if _, err := tx.Exec(`ALTER TABLE nano_enrollments DROP COLUMN last_seen_at`); err != nil {
			return fmt.Errorf("dropping last_seen_at column: %w", err)
		}
	}

	return nil
}

// backfillNanoSeenTimes copies (id, last_seen_at) from nano_enrollments in id-keyed batches
func backfillNanoSeenTimes(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	const batchSize = 10000
	var lastID string
	for {
		var batch []struct {
			ID         string    `db:"id"`
			LastSeenAt time.Time `db:"last_seen_at"`
		}
		if err := txx.Select(&batch,
			`SELECT id, last_seen_at FROM nano_enrollments
			 WHERE id > ? ORDER BY id LIMIT ?`, lastID, batchSize); err != nil {
			return fmt.Errorf("selecting enrollments after id %q: %w", lastID, err)
		}
		if len(batch) == 0 {
			return nil
		}

		values := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*2)
		for _, row := range batch {
			values = append(values, "(?, ?)")
			args = append(args, row.ID, row.LastSeenAt)
		}
		stmt := `INSERT INTO nano_seen_times (id, seen_time) VALUES ` + strings.Join(values, ", ") +
			` ON DUPLICATE KEY UPDATE seen_time = VALUES(seen_time)`
		if _, err := txx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("upserting seen times batch after id %q: %w", lastID, err)
		}
		for range batch {
			increment()
		}
		lastID = batch[len(batch)-1].ID
	}
}

func Down_20260904125235(tx *sql.Tx) error {
	return nil
}

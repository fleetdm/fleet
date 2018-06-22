package tables

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20180620175054, Down_20180620175054)
}

func Up_20180620175054(tx *sql.Tx) error {
	// Make sure all names are non-empty
	query := `
		UPDATE scheduled_queries
		SET name = COALESCE(name, query_name),
		description = COALESCE(description, ''),
		platform = COALESCE(platform, ''),
		version = COALESCE(version, '')
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "set name for all queries")
	}

	// Dedupe (name, pack_id)
	query = `
		SELECT name, pack_id
		FROM scheduled_queries
		GROUP BY name, pack_id
		HAVING count(pack_id) > 1
	`
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var dupes []struct {
		Name   string `db:"name"`
		PackID uint   `db:"pack_id"`
	}
	if err := txx.Select(&dupes, query); err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "get duplicate query names")
	}

	for _, dupe := range dupes {
		// Yes you really need to SELECT id FROM ( SELECT id FROM...
		// Otherwise MySQL errors
		query = `
			DELETE FROM scheduled_queries
			WHERE id IN
			( SELECT id FROM (
				SELECT id from scheduled_queries
				WHERE name = ? AND pack_id = ?
				ORDER BY id DESC
				LIMIT 9999 OFFSET 1
				) AS t
			)
		`
		if _, err := tx.Exec(query, dupe.Name, dupe.PackID); err != nil {
			return errors.Wrapf(err, "delete dupe %s", dupe.Name)
		}
	}

	// Enforce not-null, uniqueness and add column defaults
	query = `
		ALTER TABLE scheduled_queries
		MODIFY name varchar(255) NOT NULL,
		MODIFY description varchar(1023) DEFAULT '',
		MODIFY platform varchar(255) DEFAULT '',
		MODIFY version varchar(255) DEFAULT '',
		ADD UNIQUE KEY unique_names_in_packs (name, pack_id)
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrapf(err, "altering table")
	}

	return nil
}

func Down_20180620175054(tx *sql.Tx) error {
	return nil
}

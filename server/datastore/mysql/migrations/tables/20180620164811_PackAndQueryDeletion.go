package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20180620164811, Down_20180620164811)
}

func Up_20180620164811(tx *sql.Tx) error {
	// Drop the old foreign key for query name (to be replaced later)
	query := `
		ALTER TABLE scheduled_queries
		DROP FOREIGN KEY scheduled_queries_ibfk_1
	`
	if _, err := tx.Exec(query); err != nil {
		// If the foreign key doesn't exist (or exists under a
		// different name), we can just allow it to dupe and move on
		// rather than failing and requiring manual intervention.
		fmt.Println("Skipped deleting foreign key `scheduled_queries_ibfk_1`: " + err.Error())
	}

	// Delete any scheduled queries where the pack is already deleted
	query = `
		DELETE FROM scheduled_queries
		WHERE pack_id NOT IN (SELECT id FROM packs)
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "delete dangling scheduled queries")
	}

	query = `
		ALTER TABLE scheduled_queries
		ADD CONSTRAINT scheduled_queries_query_name
		FOREIGN KEY (query_name) REFERENCES queries (name)
		ON DELETE CASCADE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "add foreign key to query name")
	}

	query = `
		ALTER TABLE scheduled_queries
		ADD CONSTRAINT scheduled_queries_pack_id
		FOREIGN KEY (pack_id) REFERENCES packs (id)
		ON DELETE CASCADE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "add foreign key to pack ID")
	}

	return nil
}

func Down_20180620164811(tx *sql.Tx) error {
	return nil
}

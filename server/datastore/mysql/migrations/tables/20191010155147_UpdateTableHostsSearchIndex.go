package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20191010155147, Down20191010155147)
}

func Up20191010155147(tx *sql.Tx) error {
	// Update hosts_search fulltext index to allow search by UUID
	sql := `ALTER TABLE hosts DROP INDEX hosts_search`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "drop old index")
	}

	sql = `CREATE FULLTEXT INDEX hosts_search ON hosts(host_name, uuid)`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add updated index")
	}

	return nil
}

func Down20191010155147(tx *sql.Tx) error {
	// Update hosts_search fulltext index to allow search by UUID
	sql := `ALTER TABLE hosts DROP INDEX hosts_search`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "drop old index")
	}

	sql = `CREATE FULLTEXT INDEX hosts_search ON hosts(host_name)`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add updated index")
	}

	return nil
}

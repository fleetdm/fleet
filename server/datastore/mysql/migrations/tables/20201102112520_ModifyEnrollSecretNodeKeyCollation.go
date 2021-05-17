package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20201102112520, Down_20201102112520)
}

func Up_20201102112520(tx *sql.Tx) error {
	query := `
		ALTER TABLE enroll_secrets
		MODIFY secret VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "alter enroll secret collation")
	}

	query = `
		ALTER TABLE hosts
		MODIFY node_key VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "alter node key collation")
	}

	return nil
}

func Down_20201102112520(tx *sql.Tx) error {
	return nil
}

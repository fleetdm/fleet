package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220503134048, Down_20220503134048)
}

func Up_20220503134048(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE cve_scores (
    cve varchar(20) PRIMARY KEY,
    cvss_score double,
    epss_probability double,
    cisa_known_exploit boolean
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220503134048(tx *sql.Tx) error {
	return nil
}

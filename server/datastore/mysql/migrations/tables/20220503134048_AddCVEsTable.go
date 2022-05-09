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
CREATE TABLE cves (
    cve varchar(20) PRIMARY KEY,
    cvss_score double(4,2),
    epss_probability double(6,5),
    cisa_known_exploit boolean NOT NULL DEFAULT FALSE
)
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220503134048(tx *sql.Tx) error {
	return nil
}

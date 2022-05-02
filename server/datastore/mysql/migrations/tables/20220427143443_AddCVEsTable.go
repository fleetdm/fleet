package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220427143443, Down_20220427143443)
}

func Up_20220427143443(tx *sql.Tx) error {
	query := `
CREATE TABLE cves (
    cve varchar(20) PRIMARY KEY,
    cvss_score double(4,2),
    cvss_vector varchar(45),
    epss_score double(4,2),
    epss_percentile double(4,2)
)
`

	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "create cves table")
	}
	return nil
}

func Down_20220427143443(tx *sql.Tx) error {
	return nil
}

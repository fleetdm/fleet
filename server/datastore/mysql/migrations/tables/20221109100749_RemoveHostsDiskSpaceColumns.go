package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221109100749, Down_20221109100749)
}

func Up_20221109100749(tx *sql.Tx) error {
	// NOTE: *not* specifying the ALGORITHM option is better, as for mysql 5.7 it
	// will use the default of INPLACE, and for mysql 8.0.12+ it will use the
	// default of INSTANT, which is faster (only updates metadata), and fallback
	// on INPLACE otherwise.
	//
	// Same for the LOCK option, the default value is to allow the maximum level
	// of concurrency allowed by the algorithm (which will be INPLACE or INSTANT,
	// and both allow concurrent DML).
	//
	// Also, specifying an explicit LOCK and/or ALGORITHM means that the
	// operation would fail if the requested option is not supported for the
	// operation.
	const removeColStmt = `
ALTER TABLE hosts
	DROP COLUMN gigs_disk_space_available,
	DROP COLUMN percent_disk_space_available;
`
	if _, err := tx.Exec(removeColStmt); err != nil {
		return errors.Wrapf(err, "removing columns from hosts")
	}
	return nil
}

func Down_20221109100749(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210819131107, Down_20210819131107)
}

func Up_20210819131107(tx *sql.Tx) error {
	referencedTables := map[string]struct{}{"hosts": {}, "software": {}}
	table := "host_software"

	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return err
	}

	for _, constraint := range constraints {
		_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE host_software DROP FOREIGN KEY %s;`, constraint))
		if err != nil {
			if !strings.Contains(err.Error(), "check that column/key exists") {
				return errors.Wrapf(err, "dropping fk %s", constraint)
			}
		}
	}

	// Clear any orphan software and host_software
	_, err = tx.Exec(`CREATE TEMPORARY TABLE temp_host_software AS SELECT * FROM host_software;`)
	if err != nil {
		return errors.Wrap(err, "save current host software to a temp table")
	}
	_, err = tx.Exec(`DELETE FROM host_software;`)
	if err != nil {
		return errors.Wrap(err, "clear all host software")
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_software
		ADD FOREIGN KEY host_software_hosts_fk(host_id) REFERENCES hosts (id) ON DELETE CASCADE,
		ADD FOREIGN KEY host_software_software_fk(software_id) REFERENCES software (id) ON DELETE CASCADE
	`); err != nil {
		return errors.Wrap(err, "add fk on host_software hosts & software")
	}

	_, err = tx.Exec(`INSERT IGNORE INTO host_software SELECT * FROM temp_host_software;`)
	if err != nil {
		return errors.Wrap(err, "reinserting host software")
	}

	_, err = tx.Exec(`DROP TABLE temp_host_software;`)
	if err != nil {
		return errors.Wrap(err, "dropping temp table")
	}

	return nil
}

func Down_20210819131107(tx *sql.Tx) error {
	return nil
}

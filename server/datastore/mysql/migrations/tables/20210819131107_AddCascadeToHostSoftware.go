package tables

import (
	"database/sql"
	"fmt"

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
			return errors.Wrapf(err, "dropping fk %s", constraint)
		}
	}

	if _, err := tx.Exec(`
		ALTER TABLE host_software
		ADD FOREIGN KEY host_software_hosts_fk(host_id) REFERENCES hosts (id) ON DELETE CASCADE,
		ADD FOREIGN KEY host_software_software_fk(software_id) REFERENCES software (id) ON DELETE CASCADE
	`); err != nil {
		return errors.Wrap(err, "add foreign key on pack_targets pack_id")
	}

	return nil
}

func Down_20210819131107(tx *sql.Tx) error {
	return nil
}

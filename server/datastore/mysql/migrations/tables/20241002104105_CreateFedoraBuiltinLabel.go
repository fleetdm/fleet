package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20241002104105, Down_20241002104105)
}

func Up_20241002104105(tx *sql.Tx) error {
	const stmt = `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

	// hard-coded timestamps are used so that schema.sql is stable
	ts := time.Date(2024, 9, 27, 0, 0, 0, 0, time.UTC)
	_, err := tx.Exec(
		stmt,
		fleet.BuiltinLabelFedoraLinux,
		"All Fedora hosts",
		`select 1 from os_version where name = 'Fedora Linux';`,
		"rhel",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeDynamic,
		ts,
		ts,
	)
	if err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == mysqlerr.ER_DUP_ENTRY {
				return fmt.Errorf("a label with the name %q already exists, please rename it before applying this migration: %w", fleet.BuiltinLabelFedoraLinux, err)
			}
		}
		return err
	}
	return nil
}

func Down_20241002104105(tx *sql.Tx) error {
	return nil
}

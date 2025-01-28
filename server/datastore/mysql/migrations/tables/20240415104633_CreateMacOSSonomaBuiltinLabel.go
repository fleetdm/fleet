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
	MigrationClient.AddMigration(Up_20240415104633, Down_20240415104633)
}

func Up_20240415104633(tx *sql.Tx) error {
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
	ts := time.Date(2024, 4, 3, 0, 0, 0, 0, time.UTC)
	_, err := tx.Exec(
		stmt,
		fleet.BuiltinLabelMacOS14Plus,
		"macOS hosts with version 14 and above",
		`select 1 from os_version where platform = 'darwin' and major >= 14;`,
		"darwin",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeDynamic,
		ts,
		ts,
	)
	if err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == mysqlerr.ER_DUP_ENTRY {
				// TODO(mna): how do we feel about this approach to ensure the new
				// Fleet-reserved name is unique? All label names need to be unique
				// across built-in and regular. (I don't think we've done anything
				// special before, but this seems a bit nicer/clearer as to why the
				// migration may have failed and how to fix it)
				return fmt.Errorf("a label with the name %q already exists, please rename it before applying this migration: %w", fleet.BuiltinLabelMacOS14Plus, err)
			}
		}
		return err
	}
	return nil
}

func Down_20240415104633(tx *sql.Tx) error {
	return nil
}

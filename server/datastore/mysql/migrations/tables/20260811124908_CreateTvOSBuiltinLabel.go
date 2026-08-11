package tables

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20260811124908, Down_20260811124908)
}

func Up_20260811124908(tx *sql.Tx) error {
	// hard-coded timestamp is used so that schema.sql is stable
	stableTS := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

	// platform is deliberately left empty: Up_20251015103800 cleared it on every
	// builtin label and TestMigrations enforces that it stays empty. Membership
	// for this manual label is driven by the host's platform in
	// upsertMDMAppleHostLabelMembershipDB, not by this column.
	_, err := tx.Exec(`
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type,
			created_at,
			updated_at
		) VALUES (?, ?, '', '', ?, ?, ?, ?);`,
		fleet.BuiltinLabelTvOS,
		"All tvOS hosts",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeManual,
		stableTS,
		stableTS,
	)
	if err != nil {
		var driverErr *mysql.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == mysqlerr.ER_DUP_ENTRY {
			// All label names need to be unique across built-in and regular.
			// Thus we return an error and instruct the user how to solve the issue.
			return fmt.Errorf(
				"label with the name %q already exists, please rename it before applying this migration: %w",
				fleet.BuiltinLabelTvOS,
				err,
			)
		}
		return fmt.Errorf("failed to insert tvOS label: %w", err)
	}

	return nil
}

func Down_20260811124908(_ *sql.Tx) error {
	return nil
}

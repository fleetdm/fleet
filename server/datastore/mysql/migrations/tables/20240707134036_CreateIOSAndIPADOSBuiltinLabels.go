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
	MigrationClient.AddMigration(Up_20240707134036, Down_20240707134036)
}

func Up_20240707134036(tx *sql.Tx) error {
	// Create new builtin+manual labels for iOS/iPadOS
	iOSLabelID, iPadOSLabelID, err := createBuiltinManualIOSAndIPadOSLabels(tx)
	if err != nil {
		return fmt.Errorf("failed to create iOS/iPadOS labels: %w", err)
	}

	// Add label membership to existing iOS/iPadOS devices.
	if _, err := tx.Exec(`
		INSERT INTO label_membership (host_id, label_id)
		SELECT id AS host_id, IF(platform = 'ios', ?, ?) AS label_id
		FROM hosts WHERE platform = 'ios' OR platform = 'ipados';`,
		iOSLabelID, iPadOSLabelID,
	); err != nil {
		return fmt.Errorf("failed to insert label membership: %w", err)
	}

	// Move existing iOS/iPadOS profiles from "Verifying" to "Verified"
	// (there's no osquery in these devices).
	if _, err := tx.Exec(`
		UPDATE host_mdm_apple_profiles hmap
		JOIN hosts h ON hmap.host_uuid = h.uuid AND
		(h.platform = 'ios' OR h.platform = 'ipados') AND hmap.status = 'verifying'
		SET hmap.status = 'verified';`,
	); err != nil {
		return fmt.Errorf("failed to update host_mdm_apple_profiles: %w", err)
	}

	return nil
}

func createBuiltinManualIOSAndIPadOSLabels(tx *sql.Tx) (iOSLabelID uint, iPadOSLabelID uint, err error) {
	// hard-coded timestamps are used so that schema.sql is stable
	stableTS := time.Date(2024, 6, 28, 0, 0, 0, 0, time.UTC)
	for _, label := range []struct {
		name        string
		description string
		platform    string
	}{
		{
			fleet.BuiltinLabelIOS,
			"All iOS hosts",
			"ios",
		},
		{
			fleet.BuiltinLabelIPadOS,
			"All iPadOS hosts",
			"ipados",
		},
	} {
		res, err := tx.Exec(`
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type,
			created_at,
			updated_at
		) VALUES (?, ?, '', ?, ?, ?, ?, ?);`,
			label.name,
			label.description,
			label.platform,
			fleet.LabelTypeBuiltIn,
			fleet.LabelMembershipTypeManual,
			stableTS,
			stableTS,
		)
		if err != nil {
			if driverErr, ok := err.(*mysql.MySQLError); ok {
				if driverErr.Number == mysqlerr.ER_DUP_ENTRY {
					// All label names need to be unique across built-in and regular.
					// Thus we return an error and instruct the user how to solve the issue.
					//
					// NOTE(lucas): This is using the same approach we used when creating the Sonoma builtin label.
					return 0, 0, fmt.Errorf(
						"label with the name %q already exists, please rename it before applying this migration: %w",
						label.name,
						err,
					)
				}
			}
			return 0, 0, fmt.Errorf("failed to insert label: %w", err)
		}
		labelID, _ := res.LastInsertId()
		if label.name == fleet.BuiltinLabelIOS {
			iOSLabelID = uint(labelID) //nolint:gosec // dismiss G115
		} else {
			iPadOSLabelID = uint(labelID) //nolint:gosec // dismiss G115
		}
	}
	return iOSLabelID, iPadOSLabelID, nil
}

func Down_20240707134036(tx *sql.Tx) error {
	return nil
}

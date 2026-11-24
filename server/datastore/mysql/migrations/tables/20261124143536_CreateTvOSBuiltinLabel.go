package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20261124143536, Down_20261124143536)
}

func Up_20261124143536(tx *sql.Tx) error {
	// hard-coded timestamp is used so that schema.sql is stable
	stableTS := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

	if err := renameConflictingTvOSLabel20261124143536(tx); err != nil {
		return err
	}

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
		return fmt.Errorf("failed to insert tvOS label: %w", err)
	}

	return nil
}

// renameConflictingTvOSLabel20261124143536 frees the name for the built-in
// label. A label name can't be used by both an all-fleets label and a fleet's
// label, so labels in every scope are renamed, and the new name must be free
// in every scope.
func renameConflictingTvOSLabel20261124143536(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id FROM labels WHERE name = ? ORDER BY id`, fleet.BuiltinLabelTvOS)
	if err != nil {
		return fmt.Errorf("select labels named %q: %w", fleet.BuiltinLabelTvOS, err)
	}
	defer rows.Close()
	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan label named %q: %w", fleet.BuiltinLabelTvOS, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("select labels named %q: %w", fleet.BuiltinLabelTvOS, err)
	}

	n := 2
	for _, id := range ids {
		var newName string
		for ; ; n++ {
			newName = fmt.Sprintf("%s %d", fleet.BuiltinLabelTvOS, n)
			var taken bool
			// A profile or declaration keeps the name of a label that was since
			// deleted, and a name is unique per profile.
			if err := tx.QueryRow(`
				SELECT EXISTS (SELECT 1 FROM labels WHERE name = ?)
					OR EXISTS (SELECT 1 FROM mdm_configuration_profile_labels WHERE label_name = ?)
					OR EXISTS (SELECT 1 FROM mdm_declaration_labels WHERE label_name = ?)`,
				newName, newName, newName,
			).Scan(&taken); err != nil {
				return fmt.Errorf("check label name %q: %w", newName, err)
			}
			if !taken {
				break
			}
		}

		if _, err := tx.Exec(`UPDATE labels SET name = ? WHERE id = ?`, newName, id); err != nil {
			return fmt.Errorf("rename label %d to %q: %w", id, newName, err)
		}
		// These tables keep a copy of the label's name.
		for _, table := range []string{"mdm_configuration_profile_labels", "mdm_declaration_labels"} {
			if _, err := tx.Exec(fmt.Sprintf(`UPDATE %s SET label_name = ? WHERE label_id = ?`, table), newName, id); err != nil {
				return fmt.Errorf("rename label %d in %s: %w", id, table, err)
			}
		}
	}
	return nil
}

func Down_20261124143536(_ *sql.Tx) error {
	return nil
}

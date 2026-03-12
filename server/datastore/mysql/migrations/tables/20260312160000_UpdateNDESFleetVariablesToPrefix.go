package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260312160000, Down_20260312160000)
}

func Up_20260312160000(tx *sql.Tx) error {
	// This migration converts NDES SCEP from a singleton CA to support multiple
	// named CAs. The default name for existing NDES configurations is "NDES"
	// (the hardcoded name from the original 20250904091745 migration).
	//
	// Changes:
	// 1. fleet_variables: Replace exact-match entries (is_prefix=0) with
	//    prefix-based entries (is_prefix=1) with trailing underscore.
	// 2. Profile content: Update stored profiles that reference the old
	//    exact-match variable names to include the "_NDES" CA name suffix.
	//    e.g. $FLEET_VAR_NDES_SCEP_CHALLENGE -> $FLEET_VAR_NDES_SCEP_CHALLENGE_NDES
	//    This handles both $VAR and ${VAR} forms since REPLACE matches the
	//    inner substring in both cases.

	// Step 1: Delete old exact-match fleet variables.
	// Note: This cascades to mdm_configuration_profile_variables, removing
	// stale associations. The associations will be rebuilt when profiles are
	// next reconciled during batch set or the next cron cycle.
	_, err := tx.Exec(`DELETE FROM fleet_variables WHERE name IN ('FLEET_VAR_NDES_SCEP_CHALLENGE', 'FLEET_VAR_NDES_SCEP_PROXY_URL') AND is_prefix = 0`)
	if err != nil {
		return fmt.Errorf("failed to delete old NDES fleet variables: %w", err)
	}

	// Step 2: Insert new prefix-based fleet variables.
	insStmt := `
	INSERT INTO fleet_variables (
		name, is_prefix, created_at
	) VALUES
		('FLEET_VAR_NDES_SCEP_CHALLENGE_', 1, :created_at),
		('FLEET_VAR_NDES_SCEP_PROXY_URL_', 1, :created_at)
	`
	createdAt := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	stmt, args, err := sqlx.Named(insStmt, map[string]any{"created_at": createdAt})
	if err != nil {
		return fmt.Errorf("failed to prepare insert for NDES fleet_variables: %w", err)
	}
	_, err = tx.Exec(stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to insert NDES fleet_variables: %w", err)
	}

	// Step 3: Update profile content to use new variable names.
	// The default CA name is "NDES" so we append "_NDES" to the old variable names.
	// MySQL REPLACE() does a single left-to-right scan, so nested calls are safe.
	// Both $FLEET_VAR_X and ${FLEET_VAR_X} forms are handled because REPLACE
	// matches the inner "$FLEET_VAR_X" substring in the braced form too.
	for _, tbl := range []struct {
		name   string
		column string
	}{
		{"mdm_apple_configuration_profiles", "mobileconfig"},
		{"mdm_windows_configuration_profiles", "syncml"},
		{"mdm_apple_declarations", "raw_json"},
	} {
		updateStmt := fmt.Sprintf(`
			UPDATE %s SET %s = REPLACE(REPLACE(%s,
				'$FLEET_VAR_NDES_SCEP_CHALLENGE', '$FLEET_VAR_NDES_SCEP_CHALLENGE_NDES'),
				'$FLEET_VAR_NDES_SCEP_PROXY_URL', '$FLEET_VAR_NDES_SCEP_PROXY_URL_NDES')
			WHERE %s LIKE '%%FLEET_VAR_NDES_SCEP_CHALLENGE%%'
			   OR %s LIKE '%%FLEET_VAR_NDES_SCEP_PROXY_URL%%'
		`, tbl.name, tbl.column, tbl.column, tbl.column, tbl.column)

		_, err = tx.Exec(updateStmt)
		if err != nil {
			return fmt.Errorf("failed to update NDES variable references in %s.%s: %w", tbl.name, tbl.column, err)
		}
	}

	// Step 4: Recompute checksums for Apple profiles whose content was updated.
	// The checksum column is binary(16) = UNHEX(MD5(mobileconfig)).
	_, err = tx.Exec(`
		UPDATE mdm_apple_configuration_profiles SET checksum = UNHEX(MD5(mobileconfig))
		WHERE mobileconfig LIKE '%FLEET_VAR_NDES_SCEP_CHALLENGE_NDES%'
		   OR mobileconfig LIKE '%FLEET_VAR_NDES_SCEP_PROXY_URL_NDES%'
	`)
	if err != nil {
		return fmt.Errorf("failed to recompute checksums for Apple profiles: %w", err)
	}
	// Windows profiles and Apple declarations have GENERATED checksum/token
	// columns that auto-update, so no manual recomputation needed.

	return nil
}

func Down_20260312160000(tx *sql.Tx) error {
	return nil
}

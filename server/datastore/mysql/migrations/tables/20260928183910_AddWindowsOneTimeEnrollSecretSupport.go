package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260928183910, Down_20260928183910)
}

// Up_20260928183910 migration: Bind one-time enroll secrets to the Windows MDM enrollment they were minted for,
// and clear the profile name this release reserves.
//
// Windows MDM mints a one-time enroll secret before a hosts row exists: the automatic enrollment flows carry no Fleet host UUID
// (authBinarySecurityToken returns an empty one), so the enrollment is inserted unlinked and only acquires host_uuid later. The
// secret therefore binds to the MDM enrollment rather than to a host, and host_id stays NULL until the agent enrolls with it.
func Up_20260928183910(tx *sql.Tx) error {
	const table = "host_one_time_enroll_secrets"

	if !columnExists(tx, table, "mdm_windows_enrollment_id") {
		// mdm_windows_enrollments.id is INT UNSIGNED, so the referencing column must match exactly or the FK is rejected.
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD COLUMN mdm_windows_enrollment_id INT UNSIGNED NULL AFTER host_id`); err != nil {
			return fmt.Errorf("adding mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	if !indexExistsTx(tx, table, "idx_hotes_mdm_windows_enrollment_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD KEY idx_hotes_mdm_windows_enrollment_id (mdm_windows_enrollment_id)`); err != nil {
			return fmt.Errorf("adding idx_hotes_mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	if !constraintExists(tx, table, "fk_hotes_mdm_windows_enrollment_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_one_time_enroll_secrets
				ADD CONSTRAINT fk_hotes_mdm_windows_enrollment_id
					FOREIGN KEY (mdm_windows_enrollment_id) REFERENCES mdm_windows_enrollments (id) ON DELETE CASCADE`); err != nil {
			return fmt.Errorf("adding fk_hotes_mdm_windows_enrollment_id to %s: %w", table, err)
		}
	}

	return renameConflictingWindowsEnrollSecretProfiles(tx)
}

// "Fleetd enroll secret" becomes a Fleet-reserved Windows profile name in this release, and the reconciler both upserts and
// deletes by that name. Nothing stopped a customer authoring their own Windows profile under it before now, and that profile
// would be silently overwritten, or deleted off every host it is on, the first time the reconciler ran. Renaming theirs out of
// the way first is what makes the name safe to claim.
//
// Apple, declaration, and Android profiles are renamed too: profile names are unique across platforms within a team, so one of
// those under the name would block the reconciler from creating the Windows profile in that team.
func renameConflictingWindowsEnrollSecretProfiles(tx *sql.Tx) error {
	for _, t := range []struct {
		profiles, uuidColumn, hostProfiles, hostNameColumn string
	}{
		{"mdm_windows_configuration_profiles", "profile_uuid", "host_mdm_windows_profiles", "profile_name"},
		{"mdm_apple_configuration_profiles", "profile_uuid", "host_mdm_apple_profiles", "profile_name"},
		{"mdm_apple_declarations", "declaration_uuid", "host_mdm_apple_declarations", "declaration_name"},
		{"mdm_android_configuration_profiles", "profile_uuid", "host_mdm_android_profiles", "profile_name"},
	} {
		if err := renameConflictingEnrollSecretProfilesIn(tx, t.profiles, t.uuidColumn, t.hostProfiles, t.hostNameColumn); err != nil {
			return err
		}
	}
	return nil
}

func renameConflictingEnrollSecretProfilesIn(tx *sql.Tx, profiles, uuidColumn, hostProfiles, hostNameColumn string) error {
	//nolint:gosec // table and column names are constants from the list above, never user input
	selectConflicting := fmt.Sprintf(`SELECT %s FROM %s WHERE name = 'Fleetd enroll secret'`, uuidColumn, profiles)

	var uuids []string
	rows, err := tx.Query(selectConflicting)
	if err != nil {
		return fmt.Errorf("selecting conflicting enroll secret profiles in %s: %w", profiles, err)
	}
	defer rows.Close()
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return fmt.Errorf("scanning conflicting enroll secret profile in %s: %w", profiles, err)
		}
		uuids = append(uuids, uuid)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading conflicting enroll secret profiles in %s: %w", profiles, err)
	}

	// The overwhelmingly common case.
	if len(uuids) == 0 {
		return nil
	}

	// The whole UUID goes in the name, not a fragment of it. The UUID is the primary key, so the result cannot collide with
	// another row under the (team_id, name) unique key, and needs no probing for a free name.
	renamed := func(alias string) string {
		return `CONCAT('Fleetd enroll secret (renamed ', ` + alias + uuidColumn + `, ')')`
	}

	//nolint:gosec // table and column names are constants from the list above, never user input
	stmt, args, err := sqlx.In(fmt.Sprintf(`
		UPDATE %s h
		JOIN %s p ON p.%s = h.%s
		SET h.%s = %s
		WHERE h.%s IN (?)`, hostProfiles, profiles, uuidColumn, uuidColumn, hostNameColumn, renamed("p."), uuidColumn), uuids)
	if err != nil {
		return fmt.Errorf("building host profile rename for %s: %w", hostProfiles, err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("renaming host enroll secret profiles in %s: %w", hostProfiles, err)
	}

	//nolint:gosec // table and column names are constants from the list above, never user input
	stmt, args, err = sqlx.In(fmt.Sprintf(`
		UPDATE %s
		SET name = %s
		WHERE %s IN (?)`, profiles, renamed(""), uuidColumn), uuids)
	if err != nil {
		return fmt.Errorf("building profile rename for %s: %w", profiles, err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("renaming enroll secret profiles in %s: %w", profiles, err)
	}

	return nil
}

func Down_20260928183910(tx *sql.Tx) error {
	return nil
}

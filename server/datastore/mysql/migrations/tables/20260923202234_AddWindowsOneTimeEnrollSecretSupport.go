package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260923202234, Down_20260923202234)
}

// Up_20260923202234 migration: Bind one-time enroll secrets to the Windows MDM enrollment they were minted for,
// and clear the profile name this release reserves.
//
// Windows MDM mints a one-time enroll secret before a hosts row exists: the automatic enrollment flows carry no Fleet host UUID
// (authBinarySecurityToken returns an empty one), so the enrollment is inserted unlinked and only acquires host_uuid later. The
// secret therefore binds to the MDM enrollment rather than to a host, and host_id stays NULL until the agent enrolls with it.
func Up_20260923202234(tx *sql.Tx) error {
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
func renameConflictingWindowsEnrollSecretProfiles(tx *sql.Tx) error {
	const selectConflicting = `
		SELECT profile_uuid
		FROM mdm_windows_configuration_profiles
		WHERE name = 'Fleetd enroll secret'`

	var profileUUIDs []string
	rows, err := tx.Query(selectConflicting)
	if err != nil {
		return fmt.Errorf("selecting conflicting windows enroll secret profiles: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var profileUUID string
		if err := rows.Scan(&profileUUID); err != nil {
			return fmt.Errorf("scanning conflicting windows enroll secret profile: %w", err)
		}
		profileUUIDs = append(profileUUIDs, profileUUID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading conflicting windows enroll secret profiles: %w", err)
	}

	// The overwhelmingly common case.
	if len(profileUUIDs) == 0 {
		return nil
	}

	// The whole profile_uuid goes in the name, not a fragment of it. profile_uuid is the primary key, so the result cannot
	// collide with another row under the (team_id, name) unique key, and needs no probing for a free name.
	renamed := func(alias string) string {
		return `CONCAT('Fleetd enroll secret (renamed ', ` + alias + `profile_uuid, ')')`
	}

	stmt, args, err := sqlx.In(`
		UPDATE host_mdm_windows_profiles hmwp
		JOIN mdm_windows_configuration_profiles p ON p.profile_uuid = hmwp.profile_uuid
		SET hmwp.profile_name = `+renamed("p.")+`
		WHERE hmwp.profile_uuid IN (?)`, profileUUIDs)
	if err != nil {
		return fmt.Errorf("building host profile rename: %w", err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("renaming host windows enroll secret profiles: %w", err)
	}

	stmt, args, err = sqlx.In(`
		UPDATE mdm_windows_configuration_profiles
		SET name = `+renamed("")+`
		WHERE profile_uuid IN (?)`, profileUUIDs)
	if err != nil {
		return fmt.Errorf("building profile rename: %w", err)
	}
	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("renaming windows enroll secret profiles: %w", err)
	}

	return nil
}

func Down_20260923202234(tx *sql.Tx) error {
	return nil
}

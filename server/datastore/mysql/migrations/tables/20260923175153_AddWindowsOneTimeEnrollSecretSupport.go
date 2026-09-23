package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20260923175153, Down_20260923175153)
}

// Two things, because neither is useful without the other: bind one-time enroll secrets to the Windows MDM enrollment they were
// minted for, and clear the profile name this release reserves.
//
// Windows MDM mints a one-time enroll secret before a hosts row exists: the automatic enrollment flows carry no Fleet host UUID
// (authBinarySecurityToken returns an empty one), so the enrollment is inserted unlinked and only acquires host_uuid later. The
// secret therefore binds to the MDM enrollment rather than to a host, and host_id stays NULL until the agent enrolls with it.
//
// The foreign key is what invalidates a secret when the device re-enrolls: MDMWindowsDeleteEnrolledDeviceOnReenrollment deletes
// the enrollment row, and the cascade takes the secrets with it. That is the Windows equivalent of the explicit delete the Apple
// path does on re-enroll, and it cannot be forgotten by a future caller.
func Up_20260923175153(tx *sql.Tx) error {
	const table = "host_one_time_enroll_secrets"

	// Each piece is guarded separately so a run that failed partway can be retried. Adding a column that is already there, or a
	// key or constraint of a name already taken, is an error rather than a no-op in MySQL.
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
// Reusing the Apple name instead was not an option: NewMDMWindowsConfigProfile refuses a name already present in
// mdm_apple_configuration_profiles for the same team, and Fleet creates "Fleetd configuration" there for every team.
func renameConflictingWindowsEnrollSecretProfiles(tx *sql.Tx) error {
	// Fleet's own profile cannot exist yet, since this release introduces it, but matching on the payload rather than the name
	// alone keeps the statement honest if this is ever re-run against a database where it does.
	const selectConflicting = `
		SELECT profile_uuid
		FROM mdm_windows_configuration_profiles
		WHERE name = 'Fleetd enroll secret'
			AND syncml NOT LIKE '%ADMXInstall/FleetdEnrollSecret%'`

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

	// The overwhelmingly common case. host_mdm_windows_profiles has a row per host per profile, so it is worth not touching at
	// all rather than issuing an UPDATE that matches nothing.
	if len(profileUUIDs) == 0 {
		return nil
	}

	// The uuid fragment keeps the new name unique under the (team_id, name) key without having to probe for a free one, and
	// keeps it clear of the Apple and Android names that a Windows profile name also has to avoid. Both tables carry a
	// profile_uuid, so the column is qualified per statement rather than shared as one expression.
	renamed := func(alias string) string {
		return `CONCAT('Fleetd enroll secret (renamed ', SUBSTRING(` + alias + `profile_uuid, 2, 8), ')')`
	}

	// The host rows carry a denormalized copy of the name, so they move first, while the definitions still hold the old one.
	// profile_uuid leads idx_host_mdm_windows_profiles_profile_uuid_checksum, so this is an index lookup rather than a scan.
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

func Down_20260923175153(tx *sql.Tx) error {
	return nil
}

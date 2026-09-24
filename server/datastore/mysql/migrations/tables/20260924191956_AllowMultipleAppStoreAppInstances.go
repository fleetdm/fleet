package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260924191956, Down_20260924191956)
}

func Up_20260924191956(tx *sql.Tx) error {
	// Merge vpp_app_configurations, android_app_configurations, and software_update_schedules into vpp_apps_teams
	if !columnExists(tx, "vpp_apps_teams", "instance_name") {
		_, err := tx.Exec(`
			ALTER TABLE vpp_apps_teams
			ADD COLUMN instance_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			ADD COLUMN configuration mediumtext COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			ADD COLUMN update_schedule_enabled tinyint(1) NOT NULL DEFAULT '0',
			ADD COLUMN start_time char(5) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			ADD COLUMN end_time char(5) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
		`)
		if err != nil {
			return fmt.Errorf("adding version columns to vpp_apps_teams: %w", err)
		}
	}

	if indexExistsTx(tx, "vpp_apps_teams", "idx_global_or_team_id_adam_id") {
		_, err := tx.Exec(`
			ALTER TABLE vpp_apps_teams
			DROP INDEX idx_global_or_team_id_adam_id,
			ADD UNIQUE KEY idx_global_or_team_id_adam_id_instance_name (global_or_team_id, adam_id, platform, instance_name)
		`)
		if err != nil {
			return fmt.Errorf("replacing vpp_apps_teams unique key with instance_name: %w", err)
		}
	}

	if !columnExists(tx, "host_vpp_software_installs", "vpp_app_team_id") {
		_, err := tx.Exec(`
			ALTER TABLE host_vpp_software_installs
			ADD COLUMN vpp_app_team_id int unsigned DEFAULT NULL,
			ADD KEY fk_host_vpp_software_installs_vpp_app_team_id (vpp_app_team_id),
			ADD CONSTRAINT fk_host_vpp_software_installs_vpp_app_team_id
				FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE SET NULL
		`)
		if err != nil {
			return fmt.Errorf("adding vpp_app_team_id to host_vpp_software_installs: %w", err)
		}
	}

	if !columnExists(tx, "vpp_app_upcoming_activities", "vpp_app_team_id") {
		_, err := tx.Exec(`
			ALTER TABLE vpp_app_upcoming_activities
			ADD COLUMN vpp_app_team_id int unsigned DEFAULT NULL,
			ADD KEY fk_vpp_app_upcoming_activities_vpp_app_team_id (vpp_app_team_id),
			ADD CONSTRAINT fk_vpp_app_upcoming_activities_vpp_app_team_id
				FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE SET NULL
		`)
		if err != nil {
			return fmt.Errorf("adding vpp_app_team_id to vpp_app_upcoming_activities: %w", err)
		}
	}

	if !columnExists(tx, "mdm_configuration_profile_variables", "vpp_app_team_id") {
		_, err := tx.Exec(`ALTER TABLE mdm_configuration_profile_variables ADD COLUMN vpp_app_team_id int unsigned DEFAULT NULL`)
		if err != nil {
			return fmt.Errorf("adding vpp_app_team_id to mdm_configuration_profile_variables: %w", err)
		}
	}

	err := backfillAppStoreAppInstances(tx)
	if err != nil {
		return err
	}

	// Replace android_app_configuration_id with vpp_app_team_id and add back the check that each row links a fleet variable to exactly one profile, declaration, certificate template, or app
	if columnExists(tx, "mdm_configuration_profile_variables", "android_app_configuration_id") {
		// Delete variable links for Android app configurations whose app is no longer in the fleet
		_, err = tx.Exec(`
			DELETE FROM mdm_configuration_profile_variables
			WHERE android_app_configuration_id IS NOT NULL AND vpp_app_team_id IS NULL
		`)
		if err != nil {
			return fmt.Errorf("deleting unmatched android app configuration variable links: %w", err)
		}

		_, err = tx.Exec(`
			ALTER TABLE mdm_configuration_profile_variables
			DROP CHECK ck_mdm_configuration_profile_variables_exactly_one,
			DROP FOREIGN KEY fk_mdm_configuration_profile_variables_app_config_id,
			DROP INDEX idx_mdm_configuration_profile_variables_app_config_variable,
			DROP COLUMN android_app_configuration_id,
			ADD UNIQUE KEY idx_mdm_configuration_profile_variables_vpp_app_team_variable (vpp_app_team_id, fleet_variable_id),
			ADD CONSTRAINT fk_mdm_configuration_profile_variables_vpp_app_team_id
				FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE CASCADE,
			ADD CONSTRAINT ck_mdm_configuration_profile_variables_exactly_one
				CHECK ((
					(IF(apple_profile_uuid IS NULL, 0, 1) +
					 IF(windows_profile_uuid IS NULL, 0, 1) +
					 IF(apple_declaration_uuid IS NULL, 0, 1) +
					 IF(android_profile_uuid IS NULL, 0, 1) +
					 IF(certificate_template_id IS NULL, 0, 1) +
					 IF(vpp_app_team_id IS NULL, 0, 1) +
					 IF(apple_ddm_activation_uuid IS NULL, 0, 1)) = 1
				))
		`)
		if err != nil {
			return fmt.Errorf("replacing android_app_configuration_id with vpp_app_team_id: %w", err)
		}
	}

	// Drop the now unnecessary tables
	_, err = tx.Exec(`DROP TABLE IF EXISTS vpp_app_configurations, android_app_configurations, software_update_schedules`)
	if err != nil {
		return fmt.Errorf("dropping app configuration and update schedule tables: %w", err)
	}

	return nil
}

func Down_20260924191956(tx *sql.Tx) error {
	return nil
}

func backfillAppStoreAppInstances(tx *sql.Tx) error {
	// Name every existing app Default version
	_, err := tx.Exec(`UPDATE vpp_apps_teams SET instance_name = 'Default version' WHERE instance_name = ''`)
	if err != nil {
		return fmt.Errorf("backfilling vpp_apps_teams instance_name: %w", err)
	}

	// Backfill values from vpp_app_configurations
	_, err = tx.Exec(`
		UPDATE vpp_apps_teams vat
		JOIN vpp_app_configurations vac
			ON vac.application_id = vat.adam_id AND vac.platform = vat.platform AND vac.team_id = vat.global_or_team_id
		SET vat.configuration = vac.configuration
	`)
	if err != nil {
		return fmt.Errorf("backfilling vpp_apps_teams configuration from vpp_app_configurations: %w", err)
	}

	// Backfill values from android_app_configurations
	_, err = tx.Exec(`
		UPDATE vpp_apps_teams vat
		JOIN android_app_configurations aac
			ON aac.application_id = vat.adam_id AND aac.global_or_team_id = vat.global_or_team_id
		SET vat.configuration = aac.configuration
		WHERE vat.platform = 'android'
	`)
	if err != nil {
		return fmt.Errorf("backfilling vpp_apps_teams configuration from android_app_configurations: %w", err)
	}

	// Backfill values from software_update_schedules
	_, err = tx.Exec(`
		UPDATE vpp_apps_teams vat
		JOIN vpp_apps va
			ON va.adam_id = vat.adam_id AND va.platform = vat.platform
		JOIN software_update_schedules sus
			ON sus.title_id = va.title_id AND sus.team_id = vat.global_or_team_id
		SET
			vat.update_schedule_enabled = sus.enabled,
			vat.start_time = sus.start_time,
			vat.end_time = sus.end_time
	`)
	if err != nil {
		return fmt.Errorf("backfilling vpp_apps_teams update schedule: %w", err)
	}

	// Point Fleet variable links from the Android app configuration to the vpp_apps_teams row for the same app and fleet
	if columnExists(tx, "mdm_configuration_profile_variables", "android_app_configuration_id") {
		_, err = tx.Exec(`
			UPDATE mdm_configuration_profile_variables mcpv
			JOIN android_app_configurations aac
				ON aac.id = mcpv.android_app_configuration_id
			JOIN vpp_apps_teams vat
				ON vat.adam_id = aac.application_id AND vat.global_or_team_id = aac.global_or_team_id AND vat.platform = 'android'
			SET mcpv.vpp_app_team_id = vat.id
		`)
		if err != nil {
			return fmt.Errorf("backfilling mdm_configuration_profile_variables vpp_app_team_id: %w", err)
		}
	}

	const batchSize = 1000

	// Read the highest id to batch by id range, MySQL does not allow LIMIT on an UPDATE with a JOIN
	var maxInstallID uint64
	err = tx.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM host_vpp_software_installs`).Scan(&maxInstallID)
	if err != nil {
		return fmt.Errorf("getting max host_vpp_software_installs id: %w", err)
	}

	// Link installs to the version in the host's current fleet, installs from a fleet the host has since left get no version
	for startID := uint64(0); startID < maxInstallID; startID += batchSize {
		_, err = tx.Exec(`
			UPDATE host_vpp_software_installs hvsi
			JOIN hosts h
				ON h.id = hvsi.host_id
			JOIN vpp_apps_teams vat
				ON vat.adam_id = hvsi.adam_id AND vat.platform = hvsi.platform AND vat.global_or_team_id = COALESCE(h.team_id, 0)
			SET hvsi.vpp_app_team_id = vat.id
			WHERE hvsi.vpp_app_team_id IS NULL AND hvsi.id > ? AND hvsi.id <= ?
		`, startID, startID+batchSize)
		if err != nil {
			return fmt.Errorf("backfilling host_vpp_software_installs vpp_app_team_id after id %d: %w", startID, err)
		}
	}

	var maxUpcomingActivityID uint64
	err = tx.QueryRow(`SELECT COALESCE(MAX(upcoming_activity_id), 0) FROM vpp_app_upcoming_activities`).Scan(&maxUpcomingActivityID)
	if err != nil {
		return fmt.Errorf("getting max vpp_app_upcoming_activities id: %w", err)
	}

	// Link queued installs to the version in the host's current fleet
	for startID := uint64(0); startID < maxUpcomingActivityID; startID += batchSize {
		_, err = tx.Exec(`
			UPDATE vpp_app_upcoming_activities vaua
			JOIN upcoming_activities ua
				ON ua.id = vaua.upcoming_activity_id
			JOIN hosts h
				ON h.id = ua.host_id
			JOIN vpp_apps_teams vat
				ON vat.adam_id = vaua.adam_id AND vat.platform = vaua.platform AND vat.global_or_team_id = COALESCE(h.team_id, 0)
			SET vaua.vpp_app_team_id = vat.id
			WHERE vaua.vpp_app_team_id IS NULL AND vaua.upcoming_activity_id > ? AND vaua.upcoming_activity_id <= ?
		`, startID, startID+batchSize)
		if err != nil {
			return fmt.Errorf("backfilling vpp_app_upcoming_activities vpp_app_team_id after id %d: %w", startID, err)
		}
	}

	return nil
}

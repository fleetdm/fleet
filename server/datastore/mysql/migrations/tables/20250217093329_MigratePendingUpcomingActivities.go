package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250217093329, Down_20250217093329)
}

func Up_20250217093329(tx *sql.Tx) error {
	// this migration inserts pending software installs, software uninstalls,
	// VPP app installs and script executions in the upcoming_activities table
	// (inserts them already marked as "activated" since they are ready to be
	// processed). There is no ordering guarantee for those already-pending
	// activities, but any new upcoming activity will follow the unified queue
	// order.
	if err := migrateSoftwareInstalls(tx); err != nil {
		return err
	}
	if err := migrateSoftwareUninstalls(tx); err != nil {
		return err
	}
	if err := migrateVPPInstalls(tx); err != nil {
		return err
	}
	if err := migrateScriptExecs(tx); err != nil {
		return err
	}
	return nil
}

func migrateSoftwareInstalls(tx *sql.Tx) error {
	_, err := tx.Exec(`
INSERT INTO upcoming_activities
	(
		host_id,
		priority,
		user_id,
		fleet_initiated,
		activity_type,
		execution_id,
		payload,
		activated_at
	)
SELECT
	hsi.host_id,
	0,
	hsi.user_id,
	hsi.policy_id IS NOT NULL, -- true if fleet-initiated
	'software_install',
	hsi.execution_id,
	JSON_OBJECT(
		'self_service', hsi.self_service,
		'installer_filename', hsi.installer_filename,
		'version', hsi.version,
		'software_title_name', hsi.software_title_name,
		'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = hsi.user_id)
	),
	hsi.created_at
FROM
	host_software_installs hsi
	LEFT OUTER JOIN upcoming_activities ua
		ON hsi.execution_id = ua.execution_id
WHERE
	ua.id IS NULL AND
	hsi.status = 'pending_install' AND
	hsi.host_deleted_at IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending software installs: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO software_install_upcoming_activities
	(
		upcoming_activity_id,
		software_installer_id,
		policy_id,
		software_title_id,
		created_at
	)
SELECT
	ua.id,
	hsi.software_installer_id,
	hsi.policy_id,
	hsi.software_title_id,
	hsi.created_at
FROM
	upcoming_activities ua
	INNER JOIN host_software_installs hsi
		ON hsi.execution_id = ua.execution_id
	LEFT OUTER JOIN software_install_upcoming_activities sia
		ON sia.upcoming_activity_id = ua.id
WHERE
	ua.activity_type = 'software_install' AND
	hsi.status = 'pending_install' AND
	hsi.host_deleted_at IS NULL AND
	sia.upcoming_activity_id IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending software installs secondary table: %w", err)
	}
	return nil
}

func migrateSoftwareUninstalls(tx *sql.Tx) error {
	_, err := tx.Exec(`
INSERT INTO upcoming_activities
	(
		host_id,
		priority,
		user_id,
		fleet_initiated,
		activity_type,
		execution_id,
		payload,
		activated_at
	)
SELECT
	hsi.host_id,
	0,
	hsi.user_id,
	hsi.policy_id IS NOT NULL, -- true if fleet-initiated
	'software_uninstall',
	hsi.execution_id,
	JSON_OBJECT(
		'installer_filename', '',
		'version', 'unknown',
		'software_title_name', hsi.software_title_name,
		'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = hsi.user_id)
	),
	hsi.created_at
FROM
	host_software_installs hsi
	LEFT OUTER JOIN upcoming_activities ua
		ON hsi.execution_id = ua.execution_id
WHERE
	ua.id IS NULL AND
	hsi.status = 'pending_uninstall' AND
	hsi.host_deleted_at IS NULL 
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending software uninstalls: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO software_install_upcoming_activities
	(
		upcoming_activity_id,
		software_installer_id,
		software_title_id,
		created_at
	)
SELECT
	ua.id,
	hsi.software_installer_id,
	hsi.software_title_id,
	hsi.created_at
FROM
	upcoming_activities ua
	INNER JOIN host_software_installs hsi
		ON hsi.execution_id = ua.execution_id
	LEFT OUTER JOIN software_install_upcoming_activities sia
		ON sia.upcoming_activity_id = ua.id
WHERE
	ua.activity_type = 'software_uninstall' AND
	hsi.status = 'pending_uninstall' AND
	hsi.host_deleted_at IS NULL AND
	sia.upcoming_activity_id IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending software uninstalls secondary table: %w", err)
	}
	return nil
}

func migrateVPPInstalls(tx *sql.Tx) error {
	_, err := tx.Exec(`
INSERT INTO upcoming_activities
	(
		host_id,
		priority,
		user_id,
		fleet_initiated,
		activity_type,
		execution_id,
		payload,
		activated_at
	)
SELECT
	hvi.host_id,
	0,
	hvi.user_id,
	hvi.policy_id IS NOT NULL, -- true if fleet-initiated
	'vpp_app_install',
	hvi.command_uuid,
	JSON_OBJECT(
		'self_service', hvi.self_service,
		'associated_event_id', hvi.associated_event_id,
		'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = hvi.user_id)
	),
	COALESCE(hvi.created_at, NOW(6))
FROM
	host_vpp_software_installs hvi
	INNER JOIN
		nano_view_queue nvq ON nvq.command_uuid = hvi.command_uuid
	LEFT OUTER JOIN upcoming_activities ua
		ON hvi.command_uuid = ua.execution_id
WHERE
	ua.id IS NULL AND
	nvq.status IS NULL AND
	hvi.removed = 0
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending vpp app installs: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO vpp_app_upcoming_activities
	(
		upcoming_activity_id,
		adam_id,
		platform,
		policy_id,
		created_at
	)
SELECT
	ua.id,
	hvi.adam_id,
	hvi.platform,
	hvi.policy_id,
	hvi.created_at
FROM
	upcoming_activities ua
	INNER JOIN host_vpp_software_installs hvi
		ON hvi.command_uuid = ua.execution_id
	INNER JOIN
		nano_view_queue nvq ON nvq.command_uuid = hvi.command_uuid
	LEFT OUTER JOIN vpp_app_upcoming_activities vaua
		ON vaua.upcoming_activity_id = ua.id
WHERE
	ua.activity_type = 'vpp_app_install' AND
	hvi.removed = 0 AND
	nvq.status IS NULL AND
	vaua.upcoming_activity_id IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending vpp app installs secondary table: %w", err)
	}
	return nil
}

func migrateScriptExecs(tx *sql.Tx) error {
	// we don't want to migrate software uninstall scripts (those are already
	// covered by the software uninstalls), but we don't have anything special to
	// do as we will automatically ignore them with the left join on
	// upcoming_activities (and the fact that software uninstalls are processed
	// before scripts), because the uninstall scripts have the same execution id
	// as the corresponding software uninstall.
	_, err := tx.Exec(`
INSERT INTO upcoming_activities
	(
		host_id,
		priority,
		user_id,
		fleet_initiated,
		activity_type,
		execution_id,
		payload,
		activated_at
	)
SELECT
	hsr.host_id,
	0,
	hsr.user_id,
	hsr.policy_id IS NOT NULL, -- true if fleet-initiated
	'script',
	hsr.execution_id,
	JSON_OBJECT(
		'sync_request', hsr.sync_request,
		'is_internal', hsr.is_internal,
		'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = hsr.user_id)
	),
	hsr.created_at
FROM
	host_script_results hsr
	LEFT OUTER JOIN upcoming_activities ua
		ON hsr.execution_id = ua.execution_id
WHERE
	ua.id IS NULL AND
	hsr.exit_code IS NULL AND -- script is pending execution
	hsr.host_deleted_at IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending script executions: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO script_upcoming_activities
	(
		upcoming_activity_id,
		script_id,
		script_content_id,
		policy_id,
		setup_experience_script_id,
		created_at
	)
SELECT
	ua.id,
	hsr.script_id,
	hsr.script_content_id,
	hsr.policy_id,
	hsr.setup_experience_script_id,
	hsr.created_at
FROM
	upcoming_activities ua
	INNER JOIN host_script_results hsr
		ON hsr.execution_id = ua.execution_id
	LEFT OUTER JOIN script_upcoming_activities sua
		ON sua.upcoming_activity_id = ua.id
WHERE
	ua.activity_type = 'script' AND
	hsr.exit_code IS NULL AND
	hsr.host_deleted_at IS NULL AND
	sua.upcoming_activity_id IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to insert pending script executions secondary table: %w", err)
	}
	return nil
}

func Down_20250217093329(tx *sql.Tx) error {
	return nil
}

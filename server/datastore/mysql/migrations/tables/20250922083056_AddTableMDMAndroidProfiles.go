package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250922083056, Down_20250922083056)
}

func Up_20250922083056(tx *sql.Tx) error {
	// The AUTO_INCREMENT columns are used to determine if a row was updated by
	// an INSERT ... ON DUPLICATE KEY UPDATE statement. This is needed because we
	// are currently using CLIENT_FOUND_ROWS option to determine if a row was
	// found. And in order to find if the row was updated, we need to check
	// LAST_INSERT_ID(). MySQL docs:
	// https://dev.mysql.com/doc/refman/8.4/en/insert-on-duplicate.html

	// NOTE: not adding `secrets_updated_at` / `variables_updated_at` anywhere
	// just yet as we won't support them in Android profiles for now.

	createProfilesTable := `
CREATE TABLE mdm_android_configuration_profiles (
  -- profile_uuid is length 37 as it has a single char prefix (of 'g') before the actual uuid
  profile_uuid   VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  -- no FK constraint on teams, the profile is manually deleted when the team is deleted
  team_id        INT UNSIGNED NOT NULL DEFAULT '0',
  -- unique across all profiles (all platforms), must be checked on insert with the apple,
  -- windows and apple declaration names.
  name           VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  -- store as JSON so we can use JSON_CONTAINS to check if the applied JSON document
  -- contains this profile's JSON.
  raw_json       JSON NOT NULL,

  -- see note comment above, needed for upserts
  auto_increment BIGINT NOT NULL AUTO_INCREMENT UNIQUE,

  created_at     TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  uploaded_at    TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6),

  PRIMARY KEY (profile_uuid),
  UNIQUE KEY idx_mdm_android_configuration_profiles_team_id_name (team_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createProfilesTable); err != nil {
		return fmt.Errorf("create mdm_android_configuration_profiles table: %w", err)
	}

	// The table android_policy_requests tracks the API requests made to create
	// or update the policy to apply to a given host.
	createRequestsTable := `
CREATE TABLE android_policy_requests (
  request_uuid      VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  -- request_name is the policy_name (for patch policy) or device_name (for patch device)
  request_name      VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  -- the policy_id for which this request was made
  policy_id         VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  payload           JSON NOT NULL,

  -- track if API request was successful or not
  status_code       INT NOT NULL,
  -- in case of error, store details here
  error_details	    TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

  -- on success, the currently applied policy version for the device (only for patch device)
  applied_policy_version INT DEFAULT NULL,
  -- on success, the new version of the policy (only for patch policy)
  policy_version    INT DEFAULT NULL,

  created_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (request_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createRequestsTable); err != nil {
		return fmt.Errorf("create android_policy_requests table: %w", err)
	}

	// Note that the policy version ID seems to be managed by Google and AFAICT we only
	// receive it, we don't define it:
	//
	// See https://developers.google.com/android/management/reference/rest/v1/enterprises.policies
	// > The version of the policy. This is a read-only field. The version is
	// > incremented each time the policy is updated.
	//
	// Via the request_uuid and the policy_version that we could store on the requests
	// table after receiving the response from Google, we could track which version
	// a specific profile was associated with for a host.

	createHostProfilesTable := `
CREATE TABLE host_mdm_android_profiles (
  host_uuid      VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  status         VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  operation_type VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  -- same column name as for apple/windows, to store failure details if any
  detail         TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  profile_uuid   VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  profile_name   VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

  -- uuid of the corresponding android_policy_requests, supports NULL because
  -- we won't have the request uuid until the request is ready to be sent
  policy_request_uuid   VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  device_request_uuid   VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  -- counts the number of consecutive failures for AMAPI requests
  request_fail_count    TINYINT UNSIGNED NOT NULL DEFAULT '0',
  -- the latest policy version that includes this profile
  included_in_policy_version INT DEFAULT NULL,

  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (host_uuid, profile_uuid),
  FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
  FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE,
  FOREIGN KEY (policy_request_uuid) REFERENCES android_policy_requests (request_uuid) ON DELETE SET NULL,
  FOREIGN KEY (device_request_uuid) REFERENCES android_policy_requests (request_uuid) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createHostProfilesTable); err != nil {
		return fmt.Errorf("create host_mdm_android_profiles table: %w", err)
	}

	alterProfileLabelsTable := `
ALTER TABLE mdm_configuration_profile_labels
  ADD COLUMN android_profile_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  ADD FOREIGN KEY (android_profile_uuid) REFERENCES mdm_android_configuration_profiles(profile_uuid) ON DELETE CASCADE,
  ADD UNIQUE KEY idx_mdm_configuration_profile_labels_android_label_name (android_profile_uuid, label_name),
  -- only one of apple, android or windows profile uuid must be set
  ADD CONSTRAINT ck_mdm_configuration_profile_labels_profile_uuid
    CHECK (IF(ISNULL(apple_profile_uuid), 0, 1) + IF(ISNULL(windows_profile_uuid), 0, 1) + IF(ISNULL(android_profile_uuid), 0, 1) = 1)
`
	if _, err := tx.Exec(alterProfileLabelsTable); err != nil {
		return fmt.Errorf("alter mdm_configuration_profile_labels table: %w", err)
	}

	// our mysql version at the time this constraint was added did not support CHECK constraints so this may or may not exist for us
	// to delete, so we create the new wider constraint above then, optionally, delete the older narrow one
	checkIfOldConstraintExists := `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE CONSTRAINT_TYPE = 'CHECK' AND TABLE_NAME = 'mdm_configuration_profile_labels' AND CONSTRAINT_NAME = 'ck_mdm_configuration_profile_labels_apple_or_windows'`
	var constraintCount int
	if err := tx.QueryRow(checkIfOldConstraintExists).Scan(&constraintCount); err != nil {
		return fmt.Errorf("check for old CHECK constraint on mdm_configuration_profile_labels: %w", err)
	}
	if constraintCount > 0 {
		dropOldConstraint := `
			ALTER TABLE mdm_configuration_profile_labels
			DROP CONSTRAINT ck_mdm_configuration_profile_labels_apple_or_windows
		`
		if _, err := tx.Exec(dropOldConstraint); err != nil {
			return fmt.Errorf("drop old CHECK constraint on mdm_configuration_profile_labels: %w", err)
		}
	}

	alterAndroidDevicesTable := `
ALTER TABLE android_devices
	ADD COLUMN applied_policy_id VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
	ADD COLUMN applied_policy_version INT DEFAULT NULL
`
	if _, err := tx.Exec(alterAndroidDevicesTable); err != nil {
		return fmt.Errorf("alter android_devices table: %w", err)
	}

	// migrate android_policy_id to applied_policy_id, before removing the column
	migratePolicyID := `
UPDATE android_devices
	SET applied_policy_id = CAST(android_policy_id AS CHAR(100))
	WHERE android_policy_id IS NOT NULL
`
	if _, err := tx.Exec(migratePolicyID); err != nil {
		return fmt.Errorf("migrate android_policy_id to applied_policy_id: %w", err)
	}

	cleanupAndroidDevicesTable := `
ALTER TABLE android_devices
	DROP COLUMN android_policy_id
`
	if _, err := tx.Exec(cleanupAndroidDevicesTable); err != nil {
		return fmt.Errorf("cleanup android_devices table: %w", err)
	}

	// backfill the hosts.uuid column for pre-existing Android hosts that may not
	// have it set (we now use the enterprise_specific_id for that).
	updateHostUUIDs := `
UPDATE hosts
	JOIN android_devices ON android_devices.host_id = hosts.id
	SET uuid = android_devices.enterprise_specific_id
	WHERE
		hosts.platform = ? AND
		hosts.uuid = '' AND
		COALESCE(android_devices.enterprise_specific_id, '') != ''
`
	if _, err := tx.Exec(updateHostUUIDs, "android"); err != nil {
		return fmt.Errorf("backfill missing hosts.uuid for android hosts: %w", err)
	}
	return nil
}

func Down_20250922083056(tx *sql.Tx) error {
	return nil
}

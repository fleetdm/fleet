package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250825165154, Down_20250825165154)
}

func Up_20250825165154(tx *sql.Tx) error {
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
  android_policy_id INT UNSIGNED NOT NULL,
  policy_version    INT UNSIGNED NOT NULL,
  payload           JSON NOT NULL,

  -- track if API request was successful or not
  status_code       INT NOT NULL,
  -- in case of error, store (part of) the returned body
  error_body				TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

  -- from figma, retry up to 3 times before marking profile as failed
  -- (since the policy is a merge of all profiles, all profiles
  -- associated with this request would be failed)
  retries           TINYINT UNSIGNED NOT NULL DEFAULT '0',

  created_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (request_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createRequestsTable); err != nil {
		return fmt.Errorf("create android_policy_requests table: %w", err)
	}

	// TODO: do we get to define the policyId? Could it simply be the host's
	// uuid? TBD if we have to store this info, and where. I see that we already
	// have android_policy_id in the android_devices table, my guess is that this
	// is where we'll store that info (AFAIK it is a 1:1 relationship with devices).
	// See https://github.com/fleetdm/fleet/blob/49369c43b090f2bdf3c4de200046214e99252e1e/server/datastore/mysql/migrations/tables/20250219142401_UpdateAndroidTables.go#L31-L32

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
  request_uuid   VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,

  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),


  PRIMARY KEY (host_uuid, profile_uuid),
  FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
  FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE,
  FOREIGN KEY (request_uuid) REFERENCES android_policy_requests (request_uuid) ON DELETE SET NULL
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
  DROP CONSTRAINT ck_mdm_configuration_profile_labels_apple_or_windows,
  -- only one of apple, android or windows profile uuid must be set
  ADD CONSTRAINT ck_mdm_configuration_profile_labels_profile_uuid
    CHECK (IF(ISNULL(apple_profile_uuid), 0, 1) + IF(ISNULL(windows_profile_uuid), 0, 1) + IF(ISNULL(android_profile_uuid), 0, 1) = 1)
`
	if _, err := tx.Exec(alterProfileLabelsTable); err != nil {
		return fmt.Errorf("alter mdm_configuration_profile_labels table: %w", err)
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

func Down_20250825165154(tx *sql.Tx) error {
	return nil
}

package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260825213833, Down_20260825213833)
}

func Up_20260825213833(tx *sql.Tx) error {
	// These columns hold random values over a case-sensitive alphabet (base64 /
	// base64url or lowercase UUIDs), but their columns defaulted to (or were
	// declared with) the case-insensitive utf8mb4_unicode_ci collation. Lookups
	// use plain `= ?`, so a case-mutated value matched the stored one. Switch to
	// byte-exact utf8mb4_bin. Each MODIFY restates the column's exact type,
	// nullability, and default so no attribute is dropped. Mirrors the earlier
	// password_reset_requests.token fix (20260729110229).
	//
	// host_certificate_templates.fleet_challenge is a denormalized copy of
	// challenges.challenge and is only ever compared against it (never looked up
	// by value), so both move together to keep that join on a single collation.
	migrations := []struct {
		what string
		stmt string
	}{
		{"invites.token", "ALTER TABLE invites MODIFY token VARCHAR(255) COLLATE utf8mb4_bin NOT NULL"},
		{"email_changes.token", "ALTER TABLE email_changes MODIFY token VARCHAR(128) COLLATE utf8mb4_bin NOT NULL"},
		{"verification_tokens.token", "ALTER TABLE verification_tokens MODIFY token VARCHAR(255) COLLATE utf8mb4_bin NOT NULL"},
		{"acme_challenges.token", "ALTER TABLE acme_challenges MODIFY token VARCHAR(64) COLLATE utf8mb4_bin NOT NULL"},
		{"sessions.key", "ALTER TABLE sessions MODIFY `key` VARCHAR(255) COLLATE utf8mb4_bin NOT NULL"},
		{"host_device_auth.token", "ALTER TABLE host_device_auth MODIFY token VARCHAR(255) COLLATE utf8mb4_bin NOT NULL"},
		{"host_device_auth.previous_token", "ALTER TABLE host_device_auth MODIFY previous_token VARCHAR(255) COLLATE utf8mb4_bin DEFAULT NULL"},
		{"challenges.challenge", "ALTER TABLE challenges MODIFY challenge CHAR(32) COLLATE utf8mb4_bin NOT NULL"},
		{"host_certificate_templates.fleet_challenge", "ALTER TABLE host_certificate_templates MODIFY fleet_challenge CHAR(32) COLLATE utf8mb4_bin DEFAULT NULL"},
		{"android_enterprises.signup_token", "ALTER TABLE android_enterprises MODIFY signup_token VARCHAR(64) COLLATE utf8mb4_bin NOT NULL DEFAULT ''"},
		{"mdm_apple_bootstrap_packages.token", "ALTER TABLE mdm_apple_bootstrap_packages MODIFY token VARCHAR(36) COLLATE utf8mb4_bin DEFAULT NULL"},
		{"mdm_apple_enrollment_profiles.token", "ALTER TABLE mdm_apple_enrollment_profiles MODIFY token VARCHAR(36) COLLATE utf8mb4_bin DEFAULT NULL"},
		{"mdm_apple_installers.url_token", "ALTER TABLE mdm_apple_installers MODIFY url_token VARCHAR(36) COLLATE utf8mb4_bin DEFAULT NULL"},
		{"in_house_app_install_tokens.token", "ALTER TABLE in_house_app_install_tokens MODIFY token VARCHAR(36) COLLATE utf8mb4_bin NOT NULL"},
		{"eulas.token", "ALTER TABLE eulas MODIFY token VARCHAR(36) COLLATE utf8mb4_bin DEFAULT NULL"},
	}
	for _, m := range migrations {
		if _, err := tx.Exec(m.stmt); err != nil {
			return fmt.Errorf("alter %s to case-sensitive collation: %w", m.what, err)
		}
	}
	return nil
}

func Down_20260825213833(tx *sql.Tx) error {
	return nil
}

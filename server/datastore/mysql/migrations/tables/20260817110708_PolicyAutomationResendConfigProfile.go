package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260817110708, Down_20260817110708)
}

func Up_20260817110708(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE policies
ADD COLUMN resend_apple_profile_uuid varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
ADD COLUMN resend_windows_profile_uuid varchar(37) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
ADD CONSTRAINT fk_policies_resend_apple_profile FOREIGN KEY (resend_apple_profile_uuid)
    REFERENCES mdm_apple_configuration_profiles (profile_uuid),
ADD CONSTRAINT fk_policies_resend_windows_profile FOREIGN KEY (resend_windows_profile_uuid)
    REFERENCES mdm_windows_configuration_profiles (profile_uuid),
ADD CONSTRAINT ck_policies_resend_profile_uuid
    CHECK ((if((resend_apple_profile_uuid is null),0,1) + if((resend_windows_profile_uuid is null),0,1)) <= 1)`)
	return err
}

func Down_20260817110708(tx *sql.Tx) error {
	return nil
}

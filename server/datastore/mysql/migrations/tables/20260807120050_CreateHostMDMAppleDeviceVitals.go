package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260807120050, Down_20260807120050)
}

// Up_20260807120050 creates host_mdm_apple_device_vitals and
// host_mdm_apple_service_subscriptions, which hold the additional
// iOS/iPadOS host vitals collected via the expanded DeviceInformation MDM
// command (see #49984). Both tables are keyed by host_uuid, no FK, and are
// registered in additionalHostRefsByUUID for host-deletion cleanup.
func Up_20260807120050(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_mdm_apple_device_vitals (
  host_uuid                         varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  udid                              varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  model_number                      varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  modem_firmware_version            varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  supplemental_build_version        varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  supplemental_os_version_extra     varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  bluetooth_mac                     varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  wifi_mac                          varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  eas_device_identifier             varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  itunes_store_account_hash         varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  push_token                        blob,
  battery_level                     double DEFAULT NULL,
  cellular_technology               int DEFAULT NULL,
  app_analytics_enabled             tinyint(1) DEFAULT NULL,
  awaiting_configuration            tinyint(1) DEFAULT NULL,
  data_roaming_enabled              tinyint(1) DEFAULT NULL,
  diagnostic_submission_enabled     tinyint(1) DEFAULT NULL,
  is_cloud_backup_enabled           tinyint(1) DEFAULT NULL,
  is_device_locator_service_enabled tinyint(1) DEFAULT NULL,
  is_do_not_disturb_in_effect       tinyint(1) DEFAULT NULL,
  is_mdm_lost_mode_enabled          tinyint(1) DEFAULT NULL,
  is_network_tethered               tinyint(1) DEFAULT NULL,
  itunes_store_account_is_active    tinyint(1) DEFAULT NULL,
  personal_hotspot_enabled          tinyint(1) DEFAULT NULL,
  last_cloud_backup_date            datetime(6) DEFAULT NULL,
  accessibility_settings            json DEFAULT NULL,
  organization_info                 json DEFAULT NULL,
  mdm_options                       json DEFAULT NULL,
  device_properties_attestation     json DEFAULT NULL,
  created_at                        datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at                        datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_apple_device_vitals table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE host_mdm_apple_service_subscriptions (
  host_uuid                  varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  slot                       varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  carrier_settings_version   varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  current_carrier_network   varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  current_mcc                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  current_mnc                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  eid                        varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  iccid                      varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  imei                       varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  is_data_preferred          tinyint(1) DEFAULT NULL,
  is_roaming                 tinyint(1) DEFAULT NULL,
  is_voice_preferred         tinyint(1) DEFAULT NULL,
  label                      varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  label_id                   varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  meid                       varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  phone_number               varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  subscriber_carrier_network varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  created_at                 datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at                 datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (host_uuid, slot)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_apple_service_subscriptions table: %w", err)
	}
	return nil
}

func Down_20260807120050(tx *sql.Tx) error {
	return nil
}

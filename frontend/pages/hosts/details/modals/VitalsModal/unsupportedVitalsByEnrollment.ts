/** The 29 new iOS/iPadOS vitals fields (see HostMDMAppleDeviceVitals,
 * server/fleet/mdm_apple_device_vitals.go), keyed by their GET host API JSON
 * field name. */
export type VitalKey =
  | "udid"
  | "model_number"
  | "modem_firmware_version"
  | "supplemental_build_version"
  | "supplemental_os_version_extra"
  | "bluetooth_mac"
  | "wifi_mac"
  | "eas_device_identifier"
  | "itunes_store_account_hash"
  | "push_token"
  | "battery_level"
  | "cellular_technology"
  | "app_analytics_enabled"
  | "awaiting_configuration"
  | "data_roaming_enabled"
  | "diagnostic_submission_enabled"
  | "is_cloud_backup_enabled"
  | "is_device_locator_service_enabled"
  | "is_do_not_disturb_in_effect"
  | "is_mdm_lost_mode_enabled"
  | "is_network_tethered"
  | "itunes_store_account_is_active"
  | "personal_hotspot_enabled"
  | "last_cloud_backup_date"
  | "accessibility_settings"
  | "organization_info"
  | "mdm_options"
  | "device_properties_attestation"
  | "service_subscriptions";

/** The only enrollment methods an iOS/iPadOS host can report (see
 * host.mdm.enrollment_status, MDM_ENROLLMENT_STATUS_UI_MAP in
 * interfaces/mdm.ts). */
type IosOrIpadosEnrollmentStatus =
  | "On (manual)"
  | "On (automatic)"
  | "On (manual - personal)";

export const NOT_SUPPORTED_VITAL_TOOLTIP =
  "This property isn't supported for this device's enrollment method.";

/**
 * Ships empty (or with only the fields QA has already confirmed) pending
 * manual QA of each enrollment method — see sub-issue fleetdm/fleet#49987.
 * Populating this table is a fast-follow to this PR, not a blocker.
 *
 * A vital listed here for a given enrollment method always renders the
 * "Not supported" treatment for hosts enrolled that way, regardless of what
 * value the API actually returned for it.
 */
export const UNSUPPORTED_VITALS_BY_ENROLLMENT: Record<
  IosOrIpadosEnrollmentStatus,
  VitalKey[]
> = {
  "On (manual)": [],
  "On (automatic)": [],
  "On (manual - personal)": [],
};

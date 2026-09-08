import PropTypes from "prop-types";
import hostPolicyInterface, { IHostPolicy } from "./policy";
import hostUserInterface, { IHostUser } from "./host_users";
import labelInterface, { ILabel } from "./label";
import packInterface, { IPack } from "./pack";
import softwareInterface, { ISoftware } from "./software";
import hostQueryResult from "./campaign";
import queryStatsInterface, { IQueryStats } from "./query_stats";
import { ILicense, IDeviceGlobalConfig } from "./config";
import {
  IHostMdmProfile,
  MdmEnrollmentStatus,
  BootstrapPackageStatus,
  DiskEncryptionStatus,
  HostNameSettingStatus,
} from "./mdm";
import { HostPlatform } from "./platform";
import { IHostCustomVital } from "./custom_host_vitals";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  detail_updated_at: PropTypes.string,
  last_restarted_at: PropTypes.string,
  label_updated_at: PropTypes.string,
  policy_updated_at: PropTypes.string,
  last_enrolled_at: PropTypes.string,
  seen_time: PropTypes.string,
  refetch_requested: PropTypes.bool,
  hostname: PropTypes.string,
  uuid: PropTypes.string,
  platform: PropTypes.string,
  osquery_version: PropTypes.string,
  orbit_version: PropTypes.string,
  fleet_desktop_version: PropTypes.string,
  os_version: PropTypes.string,
  build: PropTypes.string,
  platform_like: PropTypes.string,
  code_name: PropTypes.string,
  uptime: PropTypes.number,
  memory: PropTypes.number,
  cpu_type: PropTypes.string,
  cpu_subtype: PropTypes.string,
  cpu_brand: PropTypes.string,
  cpu_physical_cores: PropTypes.number,
  cpu_logical_cores: PropTypes.number,
  hardware_vendor: PropTypes.string,
  hardware_model: PropTypes.string,
  hardware_version: PropTypes.string,
  hardware_serial: PropTypes.string,
  computer_name: PropTypes.string,
  primary_ip: PropTypes.string,
  primary_mac: PropTypes.string,
  distributed_interval: PropTypes.number,
  config_tls_refresh: PropTypes.number,
  logger_tls_period: PropTypes.number,
  team_id: PropTypes.number,
  pack_stats: PropTypes.arrayOf(
    PropTypes.shape({
      pack_id: PropTypes.number,
      pack_name: PropTypes.string,
      query_stats: PropTypes.arrayOf(queryStatsInterface),
    })
  ),
  team_name: PropTypes.string,
  additional: PropTypes.object, // eslint-disable-line react/forbid-prop-types
  percent_disk_space_available: PropTypes.number,
  gigs_disk_space_available: PropTypes.number,
  // On Linux hosts, `gigs_total_disk_space` only includes disk space from the "/" partition
  gigs_total_disk_space: PropTypes.number,
  // `gigs_all_disk_space` includes disk space from all partitions
  gigs_all_disk_space: PropTypes.number,
  labels: PropTypes.arrayOf(labelInterface),
  packs: PropTypes.arrayOf(packInterface),
  software: PropTypes.arrayOf(softwareInterface),
  status: PropTypes.string,
  display_name: PropTypes.string,
  users: PropTypes.arrayOf(hostUserInterface),
  policies: PropTypes.arrayOf(hostPolicyInterface),
  query_results: PropTypes.arrayOf(hostQueryResult),
  batteries: PropTypes.arrayOf(
    PropTypes.shape({
      cycle_count: PropTypes.number,
      health: PropTypes.string,
    })
  ),
});

export type HostStatus = "online" | "offline" | "new" | "missing";
export interface IDeviceUser {
  email: string;
  source: string;
}

export interface IMunkiData {
  version: string;
}

export type MacDiskEncryptionActionRequired = "log_out" | "rotate_key";

/** What the END USER can do about a disk encryption problem. Only set when there is something they can do: a Windows
 * host also reaches action_required when the TPM is not ready or policy forbids a TPM-only protector, and neither is
 * fixable from the My device page. Absent means show the reason without offering a call to action. */
export type DiskEncryptionActionRequired =
  | MacDiskEncryptionActionRequired
  | "create_pin"
  | "restart";

export type HostAndroidCertStatus =
  | "verified"
  | "failed"
  //  all below display "pending" in UI
  | "pending"
  | "delivering"
  | "delivered";

export interface IHostAndroidCert {
  name: string;
  status: HostAndroidCertStatus;
  operation_type: "install" | "remove";
  detail: string;
}

export type RecoveryLockPasswordStatus =
  | "verified"
  | "verifying"
  | "pending"
  | "failed";

export interface IHostMdmHostNameSetting {
  status: HostNameSettingStatus;
  detail: string;
}

// Prefer this over IMdmMacOsSettings, introduced MDM has expanded to non-mac platforms
export interface IOSSettings {
  disk_encryption: {
    status: DiskEncryptionStatus | null;
    detail: string;
    action_required?: DiskEncryptionActionRequired | null;
  };
  recovery_lock_password?: {
    status: RecoveryLockPasswordStatus;
    detail: string;
    password_available: boolean;
  };
  host_name?: IHostMdmHostNameSetting;
  managed_local_account?: {
    status: string | null;
    detail?: string;
    password_available: boolean;
    auto_rotate_at?: string;
    pending_rotation?: boolean;
  };
  certificates: IHostAndroidCert[];
}

// Legacy Mac mdm settings. Prefer IOSSettings
interface IMdmMacOsSettings {
  disk_encryption: DiskEncryptionStatus | null;
  action_required: MacDiskEncryptionActionRequired | null;
}

interface IMdmMacOsSetup {
  bootstrap_package_status: BootstrapPackageStatus | "";
  details: string;
  bootstrap_package_name: string;
}

export type HostMdmDeviceStatus = "unlocked" | "locked" | "wiped";
export type HostMdmPendingAction =
  | "unlock"
  | "lock"
  | "wipe"
  | "clear_passcode"
  | "location"
  | "";

export interface IHostMdmData {
  encryption_key_available: boolean;
  /**
   * encryption_key_archived indicates where there is any archived key for the host. It is only
   * populated for GET /hosts/:id and GET /hosts/identifiers/:identifier endpoints. It is not
   * populated for list hosts or other hosts endpoints.
   */
  encryption_key_archived?: boolean;
  /**
   * bootstrap_token_escrowed indicates whether Fleet has escrowed a bootstrap token for
   * the macOS host. Only applicable to macOS hosts. It is only populated for
   * GET /hosts/:id and GET /hosts/identifiers/:identifier endpoints.
   */
  bootstrap_token_escrowed?: boolean;
  enrollment_status: MdmEnrollmentStatus | null;
  /**
   * is_personal_enrollment reports whether the last MDM enrollment Fleet recorded
   * for the host was personal (BYOD). Unlike enrollment_status it is not cleared
   * on unenrollment, so BYOD-only UI doesn't flip back once the host unenrolls.
   * That holds for Android and Apple mobile hosts; on macOS and Windows the fleetd
   * detail queries re-ingest MDM state and can reset it once the profile is gone.
   */
  is_personal_enrollment?: boolean;
  dep_profile_error?: boolean;
  name?: string;
  id?: number;
  server_url: string | null;
  profiles: IHostMdmProfile[] | null;
  os_settings?: IOSSettings;
  apple_settings?: IMdmMacOsSettings;
  setup_experience?: IMdmMacOsSetup;
  device_status: HostMdmDeviceStatus;
  pending_action: HostMdmPendingAction;
  connected_to_fleet?: boolean;
  /**
   * wipe/lock/clear_passcode_allowed indicate whether the corresponding MDM
   * commands are permitted for this host based on the AccessRights delivered
   * in the host's manual (SCEP/ACME) enrollment profile. They are only
   * populated for the host-details endpoint; absent on list-hosts payloads.
   */
  wipe_allowed?: boolean;
  lock_allowed?: boolean;
  clear_passcode_allowed?: boolean;
}

export interface IHostMaintenanceWindow {
  starts_at: string; // e.g. "2024-06-18T13:27:18−07:00"
  timezone: string | null; // e.g. "America/Los_Angeles"
}

export interface IMunkiIssue {
  id: number;
  name: string;
  type: "error" | "warning";
  created_at: string;
}

interface IMacadminMDMData {
  enrollment_status: MdmEnrollmentStatus | null;
  name?: string;
  server_url: string | null;
  id?: number;
}

export interface IMacadminsResponse {
  macadmins: null | {
    munki: null | IMunkiData;
    mobile_device_management: null | IMacadminMDMData;
    munki_issues: IMunkiIssue[];
  };
}

export interface IPackStats {
  pack_id: number;
  pack_name: string;
  query_stats: IQueryStats[];
  type: string;
}

export interface IPolicyHostResponse {
  id: number;
  display_name: string;
  query_results?: unknown[];
  status?: string;
}

export interface IGeoLocation {
  country_iso: string;
  city_name: string;
  geometry?: {
    type: string;
    coordinates: number[];
  };
}

interface IBattery {
  cycle_count: number;
  health: string;
}

export interface IHostResponse {
  host: IHost;
}

// Device User Page
export interface IDUPDetails {
  host: IHostDevice;
  license: ILicense;
  /** @deprecated use `org_logo_url_dark_mode` */
  org_logo_url: string;
  /** @deprecated use `org_logo_url_light_mode` */
  org_logo_url_light_background: string;
  org_logo_url_dark_mode?: string;
  org_logo_url_light_mode?: string;
  org_contact_url: string;
  disk_encryption_enabled?: boolean;
  platform?: HostPlatform;
  global_config: IDeviceGlobalConfig;
  self_service: boolean;
}

export interface IHostEncrpytionKeyResponse {
  host_id: number;
  encryption_key: {
    updated_at: string;
    key: string;
  };
}

export interface IHostRecoveryLockPasswordResponse {
  host_id: number;
  recovery_lock_password: {
    updated_at: string;
    password: string;
    auto_rotate_at?: string;
  };
}

export interface IHostManagedAccountPasswordResponse {
  host_id: number;
  managed_account_password: {
    username: string;
    password: string;
    updated_at: string;
    auto_rotate_at?: string;
    pending_rotation?: boolean;
  };
}

export interface IHostIssues {
  total_issues_count: number;
  critical_vulnerabilities_count?: number; // Premium
  failing_policies_count: number;
}
export interface IHostEndUser {
  idp_id?: string;
  idp_username?: string;
  idp_full_name?: string;
  idp_info_updated_at: string | null;
  idp_groups?: string[];
  idp_department?: string;
  other_emails?: Array<{
    email: string;
    source: string;
  }>;
}

/** Cellular radio technology an iOS/iPadOS device supports. Apple reports an
 * integer code, which the API maps to these labels — `"unknown"` covers a code
 * Apple has added that Fleet doesn't recognize yet (see
 * fleet.MDMAppleCellularTechnology).
 * https://developer.apple.com/documentation/devicemanagement/deviceinformationresponse/queryresponses-data.dictionary */
export type HostMdmAppleCellularTechnology =
  | "None"
  | "GSM"
  | "CDMA"
  | "GSM and CDMA"
  | "unknown";

export interface IHostMdmAppleAccessibilitySettings {
  bold_text_enabled?: boolean;
  grayscale_enabled?: boolean;
  increase_contrast_enabled?: boolean;
  reduce_motion_enabled?: boolean;
  reduce_transparency_enabled?: boolean;
  text_size?: number;
  touch_accommodations_enabled?: boolean;
  voice_over_enabled?: boolean;
  zoom_enabled?: boolean;
}

export interface IHostMdmAppleOrganizationInfo {
  organization_name?: string;
  organization_address?: string;
  organization_phone?: string;
  organization_email?: string;
  organization_magic?: string;
}

export interface IHostMdmAppleDeviceVitalsMdmOptions {
  activation_lock_allowed_while_supervised?: boolean;
  bootstrap_token_allowed?: boolean;
  prompt_user_to_allow_bootstrap_token_for_authentication?: boolean;
}

export interface IHostMdmAppleServiceSubscription {
  slot: string;
  carrier_settings_version?: string;
  current_carrier_network?: string;
  current_mcc?: string;
  current_mnc?: string;
  eid?: string;
  iccid?: string;
  imei?: string;
  is_data_preferred?: boolean;
  is_roaming?: boolean;
  is_voice_preferred?: boolean;
  label?: string;
  label_id?: string;
  meid?: string;
  phone_number?: string;
  subscriber_carrier_network?: string;
}

/** AMAPI's DevicePosture enum: Google's overall risk assessment of the device.
 * The server blanks POSTURE_UNSPECIFIED, its "no data" member, so it never
 * reaches the UI.
 * https://developers.google.com/android/management/reference/rest/v1/enterprises.devices#DevicePosture */
export type AndroidSecurityPosture =
  | "SECURE"
  | "AT_RISK"
  | "POTENTIALLY_COMPROMISED";

/** AMAPI's EncryptionStatus enum, read from the device's DevicePolicyManager.
 * The server blanks ENCRYPTION_STATUS_UNSPECIFIED.
 * https://developers.google.com/android/management/reference/rest/v1/enterprises.devices#EncryptionStatus */
export type AndroidEncryptionStatus =
  | "UNSUPPORTED"
  | "INACTIVE"
  | "ACTIVATING"
  | "ACTIVE"
  | "ACTIVE_DEFAULT_KEY"
  | "ACTIVE_PER_USER";

/** AMAPI's SystemUpdateInfo.UpdateStatus enum. The server blanks
 * UPDATE_STATUS_UNKNOWN, its "no data" member.
 * https://developers.google.com/android/management/reference/rest/v1/enterprises.devices#UpdateStatus */
export type AndroidSystemUpdateStatus =
  | "UP_TO_DATE"
  | "UNKNOWN_UPDATE_AVAILABLE"
  | "SECURITY_UPDATE_AVAILABLE"
  | "OS_UPDATE_AVAILABLE";

/** One SIM card's telephony information. A device may report more than one
 * (dual-SIM). AMAPI only populates this for fully managed (company-owned)
 * devices, and the server withholds it for a personal enrollment. */
export interface IHostMdmAndroidTelephonyInfo {
  phone_number?: string;
  carrier_name?: string;
  iccid?: string;
  activation_state?: string;
  config_mode?: string;
}

/** One security risk contributing to the device's security posture, with the
 * admin-facing advice AMAPI attaches to it. */
export interface IHostMdmAndroidPostureDetail {
  security_risk?: string;
  advice?: string[];
}

export interface IHost {
  created_at: string;
  updated_at: string;
  software_updated_at?: string;
  id: number;
  detail_updated_at: string;
  last_restarted_at: string;
  label_updated_at: string;
  policy_updated_at: string;
  last_enrolled_at: string;
  last_mdm_enrolled_at: string;
  last_mdm_checked_in_at: string | null;
  last_mdm_enrollment_type?: string | null;
  seen_time: string;
  refetch_requested: boolean;
  refetch_critical_queries_until: string | null;
  hostname: string;
  uuid: string;
  platform: HostPlatform;
  osquery_version: string;
  orbit_version: string | null;
  fleet_desktop_version: string | null;
  os_version: string;
  build: string;
  platform_like: string; // TODO: replace with more specific union type
  code_name: string;
  uptime: number;
  memory: number;
  cpu_type: string;
  cpu_subtype: string;
  cpu_brand: string;
  cpu_physical_cores: number;
  cpu_logical_cores: number;
  hardware_vendor: string;
  hardware_model: string;
  hardware_marketing_name: string;
  hardware_version: string;
  hardware_serial: string;
  computer_name: string;
  timezone: string | null;
  public_ip: string;
  primary_ip: string;
  primary_mac: string;
  distributed_interval: number;
  config_tls_refresh: number;
  logger_tls_period: number;
  team_id: number | null;
  pack_stats: IPackStats[] | null;
  team_name: string | null;
  additional?: object; // eslint-disable-line @typescript-eslint/ban-types
  percent_disk_space_available: number;
  gigs_disk_space_available: number;
  // On Linux hosts, `gigs_total_disk_space` only includes disk space from the "/" partition
  gigs_total_disk_space?: number;
  // `gigs_all_disk_space` includes disk space from all partitions
  gigs_all_disk_space?: number;
  labels: ILabel[];
  packs: IPack[];
  software?: ISoftware[];
  issues: IHostIssues;
  status: HostStatus;
  display_text: string;
  display_name: string;
  target_type?: string;
  scripts_enabled: boolean | null;
  users: IHostUser[];
  device_users?: IDeviceUser[];
  munki?: IMunkiData;
  maintenance_window?: IHostMaintenanceWindow;
  mdm: IHostMdmData;
  policies: IHostPolicy[];
  query_results?: unknown[];
  geolocation?: IGeoLocation;
  batteries?: IBattery[];
  disk_encryption_enabled?: boolean;
  device_mapping: IDeviceUser[] | null;
  /** There will be at most 1 end user */
  end_users?: IHostEndUser[];
  custom_host_vitals?: IHostCustomVital[];
  conditional_access_bypassed: boolean;
  mdm_enrollment_hardware_attested?: boolean;
  dep_assigned_to_fleet: boolean;
  /** The OS version this host is required to reach. Null when OS updates
   * aren't configured for the host's fleet, and "Pending" while Fleet is still
   * resolving the target for a "latest" requirement. */
  os_update_minimum_version?: string | null;
  /** The date by which os_update_minimum_version must be installed, in
   * YYYY-MM-DD. Null and "Pending" follow os_update_minimum_version. */
  os_update_deadline?: string | null;
  // iOS/iPadOS-only vitals collected via the DeviceInformation MDM command.
  // Omitted entirely (not just null) for every other platform.
  udid?: string;
  model_number?: string;
  modem_firmware_version?: string;
  supplemental_build_version?: string;
  supplemental_os_version_extra?: string;
  bluetooth_mac?: string;
  wifi_mac?: string;
  eas_device_identifier?: string;
  itunes_store_account_hash?: string;
  push_token?: string;
  battery_level?: number;
  cellular_technology?: HostMdmAppleCellularTechnology;
  app_analytics_enabled?: boolean;
  awaiting_configuration?: boolean;
  data_roaming_enabled?: boolean;
  diagnostic_submission_enabled?: boolean;
  is_cloud_backup_enabled?: boolean;
  is_device_locator_service_enabled?: boolean;
  is_do_not_disturb_in_effect?: boolean;
  is_mdm_lost_mode_enabled?: boolean;
  is_network_tethered?: boolean;
  itunes_store_account_is_active?: boolean;
  personal_hotspot_enabled?: boolean;
  last_cloud_backup_date?: string;
  accessibility_settings?: IHostMdmAppleAccessibilitySettings;
  organization_info?: IHostMdmAppleOrganizationInfo;
  mdm_options?: IHostMdmAppleDeviceVitalsMdmOptions;
  device_properties_attestation?: string[];
  service_subscriptions?: IHostMdmAppleServiceSubscription[];
  // Android-only vitals collected from AMAPI status reports.
  // Omitted entirely (not just null) for every other platform, and for any
  // field the device didn't report.
  adb_enabled?: boolean;
  passcode_protected?: boolean;
  play_protect_enabled?: boolean;
  encryption_type?: AndroidEncryptionStatus;
  manufacturer?: string;
  /** The device's security patch level, as a YYYY-MM-DD date. */
  security_update_version?: string;
  device_kernel_version?: string;
  bootloader_version?: string;
  system_update_status?: AndroidSystemUpdateStatus;
  security_posture?: AndroidSecurityPosture;
  /** IMEI (GSM) and MEID (CDMA) are alternatives — a device reports at most
   * one. Both are withheld for personally-owned hosts, like telephony_infos. */
  imei?: string;
  meid?: string;
  api_level?: number;
  security_posture_details?: IHostMdmAndroidPostureDetail[];
  telephony_infos?: IHostMdmAndroidTelephonyInfo[];
}

/*
 * IHostDevice is an extension of IHost that is returned by the /devices endpoint. It includes the
 * dep_assigned_to_fleet field, which is not returned by the /hosts endpoint.
 */
export interface IHostDevice extends IHost {
  dep_assigned_to_fleet: boolean;
}

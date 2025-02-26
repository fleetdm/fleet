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
} from "./mdm";
import { HostPlatform } from "./platform";

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

const DEVICE_USER_SOURCE_TO_DISPLAY: { [key: string]: string } = {
  google_chrome_profiles: "Google Chrome",
  mdm_idp_accounts: "identity provider",
  custom: "custom",
} as const;

const getDeviceUserSourceForDisplay = (s: string): string => {
  return DEVICE_USER_SOURCE_TO_DISPLAY[s] || s;
};

const getDeviceUserForDisplay = (d: IDeviceUser): IDeviceUser => {
  return { ...d, source: getDeviceUserSourceForDisplay(d.source) };
};

/*
 * mapDeviceUsersForDisplay is a helper function that takes an array of device users and returns a
 * new array of device users with the source field mapped to a more user-friendly value. It also
 * ensures that the resulting array is ordered by source as follows: mdm_idp_accounts, if any,
 * custom, if any, then any remaining elements. Note that emails are not deduped.
 */
export const mapDeviceUsersForDisplay = (
  deviceMapping: IDeviceUser[]
): IDeviceUser[] => {
  const newDeviceMapping: IDeviceUser[] = [];
  let idpUser: IDeviceUser | undefined;
  let customUser: IDeviceUser | undefined;
  deviceMapping.forEach((d) => {
    switch (d.source) {
      case "mdm_idp_accounts":
        idpUser = d;
        break;
      case "custom":
        // exclude custom user without email
        if (d.email) {
          customUser = d;
        }
        break;
      default:
        newDeviceMapping.push(getDeviceUserForDisplay(d));
    }
  });
  // add idpUser and customUser to the front of the array, if they exist
  customUser && newDeviceMapping.unshift(getDeviceUserForDisplay(customUser));
  idpUser && newDeviceMapping.unshift(getDeviceUserForDisplay(idpUser));

  return newDeviceMapping;
};

export interface IDeviceMappingResponse {
  device_mapping: IDeviceUser[];
}

export interface IMunkiData {
  version: string;
}

export type MacDiskEncryptionActionRequired = "log_out" | "rotate_key";

export interface IOSSettings {
  disk_encryption: {
    status: DiskEncryptionStatus | null;
    detail: string;
  };
}

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
export type HostMdmPendingAction = "unlock" | "lock" | "wipe" | "";

export interface IHostMdmData {
  encryption_key_available: boolean;
  enrollment_status: MdmEnrollmentStatus | null;
  dep_profile_error?: boolean;
  name?: string;
  id?: number;
  server_url: string | null;
  profiles: IHostMdmProfile[] | null;
  os_settings?: IOSSettings;
  macos_settings?: IMdmMacOsSettings;
  macos_setup?: IMdmMacOsSetup;
  device_status: HostMdmDeviceStatus;
  pending_action: HostMdmPendingAction;
  connected_to_fleet?: boolean;
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

interface IGeoLocation {
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

export interface IDeviceUserResponse {
  host: IHostDevice;
  license: ILicense;
  org_logo_url: string;
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

export interface IHostIssues {
  total_issues_count: number;
  critical_vulnerabilities_count?: number; // Premium
  failing_policies_count: number;
}

export interface IHost {
  disk_encryption_status: string;
  created_at: string;
  updated_at: string;
  software_updated_at?: string;
  id: number;
  detail_updated_at: string;
  last_restarted_at: string;
  label_updated_at: string;
  policy_updated_at: string;
  last_enrolled_at: string;
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
  hardware_version: string;
  hardware_serial: string;
  computer_name: string;
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
}

/*
 * IHostDevice is an extension of IHost that is returned by the /devices endpoint. It includes the
 * dep_assigned_to_fleet field, which is not returned by the /hosts endpoint.
 */
export interface IHostDevice extends IHost {
  dep_assigned_to_fleet: boolean;
}

import PropTypes from "prop-types";
import hostPolicyInterface, { IHostPolicy } from "./policy";
import hostUserInterface, { IHostUser } from "./host_users";
import labelInterface, { ILabel } from "./label";
import packInterface, { IPack } from "./pack";
import softwareInterface, { ISoftware } from "./software";
import hostQueryResult from "./campaign";
import queryStatsInterface, { IQueryStats } from "./query_stats";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  detail_updated_at: PropTypes.string,
  label_updated_at: PropTypes.string,
  last_enrolled_at: PropTypes.string,
  seen_time: PropTypes.string,
  refetch_requested: PropTypes.bool,
  hostname: PropTypes.string,
  uuid: PropTypes.string,
  platform: PropTypes.string,
  osquery_version: PropTypes.string,
  os_version: PropTypes.string,
  build: PropTypes.string,
  platform_like: PropTypes.string,
  code_name: PropTypes.string,
  uptime: PropTypes.number,
  memory: PropTypes.number,
  cpu_type: PropTypes.string,
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
  display_text: PropTypes.string,
  users: PropTypes.arrayOf(hostUserInterface),
  policies: PropTypes.arrayOf(hostPolicyInterface),
  query_results: PropTypes.arrayOf(hostQueryResult),
});

export interface IDeviceUser {
  email: string;
}

export interface IMunkiData {
  version: string;
  last_run_time: string;
  packages_intalled_count: number;
  errors_count: number;
}

export interface IMDMData {
  health: string;
  enrollment_url: string;
}

export interface IPackStats {
  pack_id: number;
  pack_name: string;
  query_stats: IQueryStats[];
  type: string;
}

export interface IHostPolicyQuery {
  id: number;
  hostname: string;
  status?: string;
}

export interface IHostPolicyQueryError {
  host_hostname: string;
  osquery_version: string;
  error: string;
}

export interface IHost {
  created_at: string;
  updated_at: string;
  id: number;
  detail_updated_at: string;
  label_updated_at: string;
  last_enrolled_at: string;
  seen_time: string;
  refetch_requested: boolean;
  hostname: string;
  uuid: string;
  platform: string;
  osquery_version: string;
  os_version: string;
  build: string;
  platform_like: string;
  code_name: string;
  uptime: number;
  memory: number;
  cpu_type: string;
  cpu_brand: string;
  cpu_physical_cores: number;
  cpu_logical_cores: number;
  hardware_vendor: string;
  hardware_model: string;
  hardware_version: string;
  hardware_serial: string;
  computer_name: string;
  primary_ip: string;
  primary_mac: string;
  distributed_interval: number;
  config_tls_refresh: number;
  logger_tls_period: number;
  team_id: number;
  pack_stats: IPackStats[];
  team_name: string;
  additional: object; // eslint-disable-line @typescript-eslint/ban-types
  percent_disk_space_available: number;
  gigs_disk_space_available: number;
  labels: ILabel[];
  packs: IPack[];
  software: ISoftware[];
  issues: {
    total_issues_count: number;
    failing_policies_count: number;
  };
  status: string;
  display_text: string;
  users: IHostUser[];
  device_users?: IDeviceUser[];
  munki?: IMunkiData;
  mdm?: IMDMData;
  policies: IHostPolicy[];
  query_results?: [];
}

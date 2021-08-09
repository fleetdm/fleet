import PropTypes, { string } from "prop-types";
import hostUserInterface, { IHostUser } from "./host_users";
import labelInterface, { ILabel } from "./label";
import packInterface, { IPack } from "./pack";
import softwareInterface, { ISoftware } from "./software";

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
  // pack_stats: returns null HELP
  team_name: PropTypes.string,
  additional: PropTypes.object, // on /hosts/{id}
  labels: PropTypes.arrayOf(labelInterface), // on /hosts/{id}
  packs: PropTypes.arrayOf(packInterface), // on /hosts/{id}
  software: PropTypes.arrayOf(softwareInterface), // on /hosts/{id}
  status: PropTypes.string,
  display_text: PropTypes.string,
  ip: PropTypes.string, // is this outdated?
  mac: PropTypes.string, // is this outdated?
  users: PropTypes.arrayOf(hostUserInterface), // is this outdated?
});

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
  // pack_stats: returns null HELP
  team_name: string;
  additional: object; // on /hosts/{id}
  labels: ILabel[]; // on /hosts/{id}
  packs: IPack[]; // on /hosts/{id}
  software: ISoftware[]; // on /hosts/{id}
  status: string;
  display_text: string;
  ip: string; // is this outdated?
  mac: string; // is this outdated?
  users: IHostUser[]; // is this outdated?
}

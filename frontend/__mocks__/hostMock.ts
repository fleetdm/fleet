import { IHost } from "interfaces/host";
import { IHostMdmProfile } from "interfaces/mdm";
import { pick } from "lodash";

import { normalizeEmptyValues } from "utilities/helpers";
import { HOST_SUMMARY_DATA } from "utilities/constants";

const DEFAULT_HOST_PROFILE_MOCK: IHostMdmProfile = {
  profile_uuid: "123-abc",
  name: "Test Profile",
  operation_type: "install",
  platform: "darwin",
  status: "verified",
  detail: "This is verified",
};

export const createMockHostMdmProfile = (
  overrides?: Partial<IHostMdmProfile>
): IHostMdmProfile => {
  return { ...DEFAULT_HOST_PROFILE_MOCK, ...overrides };
};

const DEFAULT_HOST_MOCK: IHost = {
  id: 1,
  created_at: "2022-01-01T12:00:00Z",
  updated_at: "2022-01-02T12:00:00Z",
  detail_updated_at: "2022-01-02T12:00:00Z",
  last_restarted_at: "2022-01-02T12:00:00Z",
  label_updated_at: "2022-01-02T12:00:00Z",
  policy_updated_at: "2022-01-02T12:00:00Z",
  last_enrolled_at: "2022-01-02T12:00:00Z",
  seen_time: "2022-04-06T02:11:41Z",
  refetch_requested: false,
  refetch_critical_queries_until: null,
  hostname: "9b20fc72a247",
  display_name: "9b20fc72a247",
  display_text: "mock host 1",
  uuid: "09b244f8-0000-0000-b5cc-791a15f11073",
  platform: "ubuntu",
  osquery_version: "4.9.0",
  orbit_version: "1.22.0",
  fleet_desktop_version: "1.22.1",
  os_version: "Ubuntu 18.4.0",
  build: "",
  platform_like: "debian",
  code_name: "",
  uptime: 281037000000000,
  memory: 6232231936,
  cpu_type: "x86_64",
  cpu_subtype: "158",
  cpu_brand: "Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz",
  cpu_physical_cores: 8,
  cpu_logical_cores: 8,
  hardware_vendor: "",
  hardware_model: "",
  hardware_version: "",
  hardware_serial: "",
  computer_name: "9b20fc72a247",
  mdm: {
    encryption_key_available: false,
    enrollment_status: "Off",
    server_url: "https://www.example.com/1",
    profiles: [],
    os_settings: {
      disk_encryption: {
        status: null,
        detail: "",
      },
    },
    macos_settings: {
      disk_encryption: null,
      action_required: null,
    },
    macos_setup: {
      bootstrap_package_status: "",
      details: "",
      bootstrap_package_name: "",
    },
    device_status: "unlocked",
    pending_action: "",
  },
  public_ip: "",
  primary_ip: "172.23.0.3",
  primary_mac: "02:42:ac:17:00:03",
  distributed_interval: 10,
  config_tls_refresh: 10,
  logger_tls_period: 10,
  team_id: null,
  pack_stats: null,
  team_name: null,
  gigs_disk_space_available: 100.0,
  percent_disk_space_available: 50,
  issues: {
    total_issues_count: 0,
    failing_policies_count: 0,
  },
  status: "offline",
  scripts_enabled: false,
  labels: [],
  packs: [],
  software: [],
  users: [],
  policies: [],
  device_mapping: [],
};

const createMockHost = (overrides?: Partial<IHost>): IHost => {
  return { ...DEFAULT_HOST_MOCK, ...overrides };
};

export const createMockHostResponse = { host: createMockHost() };

export const createMockHostSummary = (overrides?: Partial<IHost>) => {
  return normalizeEmptyValues(
    pick(createMockHost(overrides), HOST_SUMMARY_DATA)
  );
};

export default createMockHost;

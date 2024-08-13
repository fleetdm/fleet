import { IHost } from "interfaces/host";
import { IHostMdmProfile } from "interfaces/mdm";
import { pick } from "lodash";

import { normalizeEmptyValues } from "utilities/helpers";
import { HOST_SUMMARY_DATA } from "utilities/constants";
import { IGetHostSoftwareResponse } from "services/entities/hosts";
import {
  IHostAppStoreApp,
  IHostSoftware,
  IHostSoftwarePackage,
} from "interfaces/software";

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
    critical_vulnerabilities_count: 0,
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

export const createMockIosHostResponse = {
  host: createMockHost({
    hostname: "Test device (iPhone)",
    display_name: "Test device (iPhone)",
    team_id: 2,
    team_name: "Mobile",
    platform: "ios",
    os_version: "iOS 14.7.1",
    hardware_serial: "C8QH6T96DPNA",
    created_at: "2024-01-01T12:00:00Z",
    updated_at: "2024-05-02T12:00:00Z",
    detail_updated_at: "2024-05-02T12:00:00Z",
    last_restarted_at: "2024-04-02T12:00:00Z",
    last_enrolled_at: "2024-01-02T12:00:00Z",
  }),
};

export const createMockHostSummary = (overrides?: Partial<IHost>) => {
  return normalizeEmptyValues(
    pick(createMockHost(overrides), HOST_SUMMARY_DATA)
  );
};

const DEFAULT_HOST_SOFTWARE_PACKAGE_MOCK: IHostSoftwarePackage = {
  name: "mock software.app",
  version: "1.0.0",
  self_service: false,
  icon_url: "https://example.com/icon.png",
  last_install: {
    install_uuid: "123-abc",
    installed_at: "2022-01-01T12:00:00Z",
  },
};

export const createMockHostSoftwarePackage = (
  overrides?: Partial<IHostSoftwarePackage>
): IHostSoftwarePackage => {
  return { ...DEFAULT_HOST_SOFTWARE_PACKAGE_MOCK, ...overrides };
};

const DEFAULT_HOST_APP_STORE_APP_MOCK: IHostAppStoreApp = {
  app_store_id: "123456789",
  version: "1.0.0",
  self_service: false,
  icon_url: "https://via.placeholder.com/512",
  last_install: null,
};

export const createMockHostAppStoreApp = (
  overrides?: Partial<IHostAppStoreApp>
): IHostAppStoreApp => {
  return { ...DEFAULT_HOST_APP_STORE_APP_MOCK, ...overrides };
};

const DEFAULT_HOST_SOFTWARE_MOCK: IHostSoftware = {
  id: 1,
  name: "mock software.app",
  software_package: createMockHostSoftwarePackage(),
  app_store_app: null,
  source: "apps",
  bundle_identifier: "com.test.mock",
  status: "installed",
  installed_versions: [
    {
      version: "1.0.0",
      last_opened_at: "2022-01-01T12:00:00Z",
      vulnerabilities: ["CVE-2020-0001"],
      installed_paths: ["/Applications/mock.app"],
    },
  ],
};

export const createMockHostSoftware = (
  overrides?: Partial<IHostSoftware>
): IHostSoftware => {
  return {
    ...DEFAULT_HOST_SOFTWARE_MOCK,
    ...overrides,
  };
};

const DEFAULT_GET_HOST_SOFTWARE_RESPONSE_MOCK: IGetHostSoftwareResponse = {
  count: 1,
  software: [createMockHostSoftware()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockGetHostSoftwareResponse = (
  overrides?: Partial<IGetHostSoftwareResponse>
): IGetHostSoftwareResponse => {
  return {
    ...DEFAULT_GET_HOST_SOFTWARE_RESPONSE_MOCK,
    ...overrides,
  };
};

export default createMockHost;

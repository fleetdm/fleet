import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IOSVersionsResponse } from "services/entities/operating_systems";
import { createMockSoftwareVulnerability } from "./softwareMock";

const DEFAULT_OS_VERSION: IOperatingSystemVersion = {
  os_version_id: 1,
  name: "Mac OS X",
  name_only: "Mac OS X",
  version: "10.15.7",
  platform: "darwin",
  hosts_count: 1,
  generated_cpes: ["cpe:/o:apple:mac_os_x:10.15.7"],
  kernels: [],
  vulnerabilities: [createMockSoftwareVulnerability()],
};

export const createMockOSVersion = (
  overrides?: Partial<IOperatingSystemVersion>
): IOperatingSystemVersion => {
  return { ...DEFAULT_OS_VERSION, ...overrides };
};

const DEFAULT_OS_VERSIONS_RESPONSE: IOSVersionsResponse = {
  count: 1,
  counts_updated_at: "2021-01-01T00:00:00Z",
  os_versions: [createMockOSVersion()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

// eslint-disable-next-line import/prefer-default-export
export const createMockOSVersionsResponse = (
  overrides?: Partial<IOSVersionsResponse>
): IOSVersionsResponse => {
  return { ...DEFAULT_OS_VERSIONS_RESPONSE, ...overrides };
};

const DEFAULT_LINUX_OS_VERSION: IOperatingSystemVersion = {
  os_version_id: 3,
  hosts_count: 2,
  name: "Ubuntu 24.04.1 LTS",
  name_only: "Ubuntu",
  version: "24.04.1 LTS",
  platform: "ubuntu",
  generated_cpes: [],
  kernels: [
    {
      id: 561703,
      version: "6.11.0-26.26~24.04.1",
      // 14 total, some repeated
      vulnerabilities: [
        "CVE-2023-53034", // repeat
        "CVE-2024-53222", // repeat
        "CVE-2024-58092", // repeat
        "CVE-2024-58093", // repeat
        "CVE-2025-21893",
        "CVE-2025-21894",
        "CVE-2025-21902", // repeat
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908", // repeat
        "CVE-2025-21909", // repeat
        "CVE-2025-21910", // repeat
      ],
      hosts_count: 1,
    },
    {
      id: 568098,
      version: "6.11.0-28.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 561709,
      version: "6.11.0-27.26~24.04.1",
      // purposefully create some repeats
      vulnerabilities: [
        "CVE-2023-53034", // repeat
        "CVE-2024-53222", // repeat
        "CVE-2024-58092", // repeat
        "CVE-2024-58093", // repeat
        "CVE-2025-21902", // repeat
        "CVE-2025-21908", // repeat
        "CVE-2025-21909", // repeat
        "CVE-2025-21910", // repeat
        "CVE-2025-21911",
      ],
      hosts_count: 1,
    },
  ],
  vulnerabilities: [
    {
      cve: "CVE-2023-53034",
      details_link: "https://nvd.nist.gov/vuln/detail/CVE-2023-53034",
      created_at: "2023-07-01T00:15:00Z",
      cvss_score: 7.8,
      epss_probability: 0.9729,
      cisa_known_exploit: false,
      cve_published: "2023-06-01T00:15:00Z",
      cve_description: "A description",
      resolved_in_version: "",
    },
  ],
};

export const createMockLinuxOSVersion = (
  overrides?: Partial<IOperatingSystemVersion>
): IOperatingSystemVersion => {
  return { ...DEFAULT_LINUX_OS_VERSION, ...overrides };
};

const DEFAULT_LINUX_OS_VERSIONS_RESPONSE: IOSVersionsResponse = {
  count: 1,
  counts_updated_at: "2021-01-01T00:00:00Z",
  os_versions: [createMockLinuxOSVersion()],
  meta: {
    has_next_results: false,
    has_previous_results: false,
  },
};

export const createMockLinuxOSVersionsResponse = (
  overrides?: Partial<IOSVersionsResponse>
): IOSVersionsResponse => {
  return { ...DEFAULT_LINUX_OS_VERSIONS_RESPONSE, ...overrides };
};

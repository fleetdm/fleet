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
      id: 561703, // the software version ID of the kernel
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21893",
        "CVE-2025-21894",
        "CVE-2025-21902",
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
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
      id: 561709, // the software version ID of the kernel
      version: "6.11.0-27.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21893",
        "CVE-2025-21894",
        "CVE-2025-21902",
        "CVE-2025-21910",
      ],
      hosts_count: 1,
    },
    {
      id: 561703, // the software version ID of the kernel
      version: "6.11.0-25.26~24.04.1",
      vulnerabilities: [
        "CVE-2025-21902",
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 1,
    },
    {
      id: 568096,
      version: "6.11.0-24.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 561703, // the software version ID of the kernel
      version: "6.11.0-23.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 1,
    },
    {
      id: 568096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 3,
    },
    {
      id: 56173, // the software version ID of the kernel
      version: "6.11.0-21.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21893",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 7,
    },
    {
      id: 58096,
      version: "6.11.0-22.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 20,
    },
    {
      id: 61703, // the software version ID of the kernel
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2025-21910"],
      hosts_count: 1,
    },
    {
      id: 56806,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 10,
    },
    {
      id: 56096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 56273,
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2023-53034", "CVE-2024-53222"],
      hosts_count: 17,
    },
    {
      id: 568096,
      version: "6.11.0-24.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 561703, // the software version ID of the kernel
      version: "6.11.0-23.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 1,
    },
    {
      id: 568096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 3,
    },
    {
      id: 56173, // the software version ID of the kernel
      version: "6.11.0-21.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21893",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 7,
    },
    {
      id: 58096,
      version: "6.11.0-22.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 20,
    },
    {
      id: 61703, // the software version ID of the kernel
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2025-21910"],
      hosts_count: 1,
    },
    {
      id: 56806,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 10,
    },
    {
      id: 56096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 56273,
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2023-53034", "CVE-2024-53222"],
      hosts_count: 17,
    },
    {
      id: 568096,
      version: "6.11.0-24.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 2,
    },
    {
      id: 561703, // the software version ID of the kernel
      version: "6.11.0-23.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21903",
        "CVE-2025-21904",
        "CVE-2025-21905",
        "CVE-2025-21906",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 1,
    },
    {
      id: 5686096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 3,
    },
    {
      id: 57173, // the software version ID of the kernel
      version: "6.11.0-21.26~24.04.1",
      vulnerabilities: [
        "CVE-2023-53034",
        "CVE-2024-53222",
        "CVE-2024-58092",
        "CVE-2024-58093",
        "CVE-2025-21893",
        "CVE-2025-21908",
        "CVE-2025-21909",
        "CVE-2025-21910",
      ],
      hosts_count: 71,
    },
    {
      id: 518096,
      version: "6.11.0-22.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 20,
    },
    {
      id: 612703, // the software version ID of the kernel
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2025-21910"],
      hosts_count: 1,
    },
    {
      id: 564806,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 1,
    },
    {
      id: 560396,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 5,
    },
    {
      id: 565273,
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2023-53034", "CVE-2024-53222"],
      hosts_count: 37,
    },
    {
      id: 568906,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 3,
    },
    {
      id: 506096,
      version: "6.11.0-29.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 12,
    },
    {
      id: 516273,
      version: "6.11.0-26.26~24.04.1",
      vulnerabilities: ["CVE-2023-53034", "CVE-2024-53222"],
      hosts_count: 117,
    },
    {
      id: 568096,
      version: "6.11.0-24.29~24.04.1",
      vulnerabilities: null,
      hosts_count: 22,
    },
  ],
  vulnerabilities: [
    {
      cve: "CVE-2023-53034",
      details_link: "https://nvd.nist.gov/vuln/detail/CVE-2023-53034",
      created_at: "2023-07-01T00:15:00Z",
      cvss_score: 7.8, // Available in Fleet Premium
      epss_probability: 0.9729, // Available in Fleet Premium
      cisa_known_exploit: false, // Available in Fleet Premium
      cve_published: "2023-06-01T00:15:00Z", // Available in Fleet Premium
      cve_description: "A description", // Available in Fleet Premium
      resolved_in_version: "", // Available in Fleet Premium
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

import { IVulnerability } from "interfaces/vulnerability";
import {
  IVulnerabilitiesResponse,
  IVulnerabilityResponse,
} from "services/entities/vulnerabilities";

const DEFAULT_VULNERABILITY: IVulnerability = {
  cve: "CVE-2022-30190",
  created_at: "2022-06-01T00:15:00Z",
  hosts_count: 1234,
  hosts_count_updated_at: "2023-12-20T15:23:57Z",
  details_link: "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
  cvss_score: 7.8, // Available in Fleet Premium
  epss_probability: 0.9729, // Available in Fleet Premium
  cisa_known_exploit: true, // Available in Fleet Premium
  cve_published: "2022-06-01T00:15:00Z", // Available in Fleet Premium
  cve_description:
    "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.", // Available in Fleet Premium
  resolved_in_version: "", // Available in Fleet Premium
  os_versions: [
    {
      os_version_id: 1,
      name: "bad version",
      name_only: "bad version",
      version: "1",
      platform: "windows",
      hosts_count: 5,
      resolved_in_version: "2",
      generated_cpes: [],
    },
  ],
  software: [
    {
      id: 1,
      name: "bad software",
      version: "1.1.1",
      bundle_identifier: "com.bad.software",
      source: "apps",
      generated_cpe: "cpe:/a:bad:software:1.1.1",
      hosts_count: 5,
      last_opened_at: "2021-08-18T15:11:35Z",
      installed_paths: ["/Applications/BadSoftware.app"],
      resolved_in_version: "2",
    },
  ],
};

export const createMockVulnerability = (
  overrides?: Partial<IVulnerability>
): IVulnerability => {
  return { ...DEFAULT_VULNERABILITY, ...overrides };
};

const DEFAULT_VULNERABILITIES_RESPONSE: IVulnerabilitiesResponse = {
  count: 1,
  counts_updated_at: "2024-02-01T00:00:00Z",
  vulnerabilities: [createMockVulnerability()],
  meta: {
    has_next_results: true,
    has_previous_results: false,
  },
};

export const createMockVulnerabilityResponse = (
  overrides?: Partial<IVulnerability>
): IVulnerabilityResponse => {
  return { vulnerability: { ...DEFAULT_VULNERABILITY, ...overrides } };
};

// eslint-disable-next-line import/prefer-default-export
export const createMockVulnerabilitiesResponse = (
  overrides?: Partial<IVulnerabilitiesResponse>
): IVulnerabilitiesResponse => {
  return { ...DEFAULT_VULNERABILITIES_RESPONSE, ...overrides };
};

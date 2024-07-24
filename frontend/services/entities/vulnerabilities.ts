/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IVulnerability } from "interfaces/vulnerability";
import { buildQueryStringFromParams } from "utilities/url";
import { IVulnerabilityOSVersion } from "interfaces/operating_system";
import { IVulnerabilitySoftware } from "interfaces/software";

export interface IGetVulnerabilitiesQueryParams {
  teamId?: number;
  order_key?: string;
  order_direction?: string;
  page?: number;
  per_page?: number;
  exploit?: boolean;
  query?: string;
}

export interface IGetVulnerabilitiesQueryKey
  extends IGetVulnerabilitiesQueryParams {
  scope: string;
}

interface IGetVulnerabilityOptions {
  vulnerability: string;
  teamId?: number;
}

export interface IGetVulnerabilityQueryKey extends IGetVulnerabilityOptions {
  scope: "softwareVulnByCVE";
}

export interface IVulnerabilitiesResponse {
  count: number;
  counts_updated_at: string;
  vulnerabilities: IVulnerability[] | null; // API can return null
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

// `GET /api/v1/fleet/vulnerabilities/:cve`
export interface IVulnerabilityResponse {
  vulnerability: IVulnerability;
  os_versions: IVulnerabilityOSVersion[];
  software: IVulnerabilitySoftware[];
}
// "vulnerability": {
//   "cve": "CVE-2022-30190",
//   "created_at": "2022-06-01T00:15:00Z",
//   "hosts_count": 1234,
//   "hosts_count_updated_at": "2023-12-20T15:23:57Z",
//   "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
//   "cvss_score": 7.8,// Available in Fleet Premium
//   "epss_probability": 0.9729,// Available in Fleet Premium
//   "cisa_known_exploit": false,// Available in Fleet Premium
//   "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
//   "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
//   "os_versions" : [
//     {
//       "os_version_id": 6,
//       "hosts_count": 200,
//       "name": "iPadOS 17.0.1", #iOS 17.0.1,#macOS 14.1.2
//       "name_only": "iPadOS", #iOS,macOS
//       "version": "14.1.2",
//       "resolved_in_version": "14.2",
//       "generated_cpes": [
//         "cpe:2.3:o:apple:macos:*:*:*:*:*:14.2:*:*",
//         "cpe:2.3:o:apple:mac_os_x:*:*:*:*:*:14.2:*:*"
//       ]
//     }
//   ],
//   "software": [
//     {
//       "id": 2363,
//       "name": "Docker Desktop",
//       "version": "4.9.1",
//       "source": "ipados_apps", # | ios_apps | programs | ...
//       "browser": "",
//       "generated_cpe": "cpe:2.3:a:docker:docker_desktop:4.9.1:*:*:*:*:windows:*:*",
//       "hosts_count": 50,
//       "resolved_in_version": "5.0.0"
//     }
//   ]
// }

export const getVulnerabilities = ({
  teamId,
  order_key,
  order_direction,
  page,
  per_page,
  exploit,
  query,
}: IGetVulnerabilitiesQueryParams = {}): Promise<IVulnerabilitiesResponse> => {
  const { VULNERABILITIES } = endpoints;
  let path = VULNERABILITIES;

  const queryString = buildQueryStringFromParams({
    team_id: teamId,
    order_key,
    order_direction,
    page,
    per_page,
    exploit,
    query,
  });

  if (queryString) path += `?${queryString}`;

  return sendRequest("GET", path);
};

const getVulnerability = ({
  vulnerability,
  teamId,
}: IGetVulnerabilityOptions): Promise<IVulnerabilityResponse> => {
  const endpoint = endpoints.VULNERABILITY(vulnerability);
  const path = teamId ? `${endpoint}?team_id=${teamId}` : endpoint;

  return sendRequest("GET", path);
};

export default {
  getVulnerabilities,
  getVulnerability,
};

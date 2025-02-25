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

export interface IVulnerabilityResponse {
  vulnerability: IVulnerability;
  os_versions: IVulnerabilityOSVersion[];
  software: IVulnerabilitySoftware[];
}

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
  const queryString = buildQueryStringFromParams({ team_id: teamId });
  const path =
    typeof teamId === "undefined" ? endpoint : `${endpoint}?${queryString}`;

  return sendRequest("GET", path);
};

export type IVulnerabilitiesEmptyStateReason =
  | "unknown-cve"
  | "invalid-cve"
  | "known-vuln"
  | "no-matching-items"
  | "no-vulns-detected";

export default {
  getVulnerabilities,
  getVulnerability,
};

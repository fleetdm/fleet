/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IVulnerability } from "interfaces/vulnerability";
import { buildQueryStringFromParams } from "utilities/url";

export interface IGetVulnerabilitiesQueryParams {
  teamId?: number;
  order_key?: string;
  order_direction?: string;
  page?: number;
  per_page?: number;
  exploited?: boolean;
  query?: string;
}

export interface IGetVulnerabilitiesQueryKey
  extends IGetVulnerabilitiesQueryParams {
  scope: string;
}

interface IGetVulnerabilityOptions {
  cve: string;
  teamId?: number;
}

export interface IGetVulnerabilityQueryKey extends IGetVulnerabilityOptions {
  scope: "softwareVulnByCVE";
}

export interface IVulnerabilitiesResponse {
  count: number;
  counts_updated_at: string;
  vulnerabilities: IVulnerability[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IVulnerabilityResponse {
  vulnerability: IVulnerability;
}

export const getVulnerabilities = ({
  teamId,
  order_key,
  order_direction,
  page,
  per_page,
  exploited,
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
    exploited,
    query,
  });

  if (queryString) path += `?${queryString}`;

  return sendRequest("GET", path);
};

const getVulnerability = ({
  cve,
  teamId,
}: IGetVulnerabilityOptions): Promise<IVulnerabilityResponse> => {
  const endpoint = endpoints.VULNERABILITY(cve);
  const path = teamId ? `${endpoint}?team_id=${teamId}` : endpoint;

  return sendRequest("GET", path);
};

export default {
  getVulnerabilities,
  getVulnerability,
};

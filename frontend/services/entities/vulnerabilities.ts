/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IVulnerability } from "interfaces/vulnerability";
import { buildQueryStringFromParams } from "utilities/url";
import {
  createMockVulnerabilitiesResponse,
  createMockVulnerabilityResponse,
} from "__mocks__/vulnerabilitiesMock";

export interface IGetVulnerabilitiesQueryParams {
  teamId?: number;
  order_key?: string;
  order_direction?: string;
  page?: number;
  per_page?: number;
}

export interface IGetVulnerabilitiesQueryKey
  extends IGetVulnerabilitiesQueryParams {
  scope: string;
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
}: IGetVulnerabilitiesQueryParams = {}): Promise<IVulnerabilitiesResponse> => {
  const { VULNERABILITIES } = endpoints;
  let path = VULNERABILITIES;

  const queryString = buildQueryStringFromParams({
    team_id: teamId,
    order_key,
    order_direction,
    page,
    per_page,
  });

  if (queryString) path += `?${queryString}`;

  // return sendRequest("GET", path); // TODO: API INTEGRATION: uncomment when API is ready
  return new Promise((resolve, reject) => {
    resolve(createMockVulnerabilitiesResponse());
  });
};

const getVulnerability = (id: number): Promise<IVulnerabilityResponse> => {
  const { VULNERABILITY } = endpoints;

  // return sendRequest("GET", VULNERABILITY(id)); // TODO: API INTEGRATION: uncomment when API is ready
  return new Promise((resolve, reject) => {
    resolve(createMockVulnerabilityResponse());
  });
};

export default {
  getVulnerabilities,
  getVulnerability,
};

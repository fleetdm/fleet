/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { Platform } from "interfaces/platform";
import { buildQueryStringFromParams } from "utilities/url";

// TODO: add platforms to this constant as new ones are supported
export const OS_VERSIONS_API_SUPPORTED_PLATFORMS = [
  "darwin",
  "windows",
  "chrome",
  "ios",
  "ipados",
];

export interface IGetOSVersionsQueryParams {
  platform?: Platform | "";
  teamId?: number;
  os_name?: string;
  os_version?: string;
  order_key?: string;
  order_direction?: string;
  page?: number;
  per_page?: number;
}

export interface IGetOSVersionsQueryKey extends IGetOSVersionsQueryParams {
  scope: string;
}

export interface IOSVersionsResponse {
  count: number;
  counts_updated_at: string;
  os_versions: IOperatingSystemVersion[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}
interface IGetOsVersionOptions {
  os_version_id: number;
  teamId?: number;
}

export interface IGetOsVersionQueryKey extends IGetOsVersionOptions {
  scope: "osVersionDetails";
}

export interface IOSVersionResponse {
  counts_updated_at?: string;
  os_version: IOperatingSystemVersion;
}

type IGetOSVersionsRequestQueryParams = Record<
  string,
  string | number | undefined
>;

export const getOSVersions = ({
  platform,
  teamId,
  os_name,
  os_version,
  order_key,
  order_direction,
  page,
  per_page,
}: IGetOSVersionsQueryParams = {}): Promise<IOSVersionsResponse> => {
  const { OS_VERSIONS } = endpoints;
  let path = OS_VERSIONS;

  const params: IGetOSVersionsRequestQueryParams = {
    platform,
    team_id: teamId,
    os_name,
    os_version,
    order_key,
    order_direction,
    page,
    per_page,
  };

  const queryString = buildQueryStringFromParams(params);

  if (queryString) path += `?${queryString}`;

  return sendRequest("GET", path);
};

const getOSVersion = ({
  os_version_id,
  teamId,
}: IGetOsVersionOptions): Promise<IOSVersionResponse> => {
  const endpoint = endpoints.OS_VERSION(os_version_id);
  const queryString = buildQueryStringFromParams({ team_id: teamId });
  const path = teamId !== undefined ? `${endpoint}?${queryString}` : endpoint;

  return sendRequest("GET", path);
};

export default {
  getOSVersions,
  getOSVersion,
};

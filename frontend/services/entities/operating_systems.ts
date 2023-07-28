/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { OsqueryPlatform } from "interfaces/platform";
import { buildQueryStringFromParams } from "utilities/url";

// TODO: add platforms to this constant as new ones are supported
export const OS_VERSIONS_API_SUPPORTED_PLATFORMS = [
  "darwin",
  "windows",
  "chrome",
];

export interface IGetOSVersionsRequest {
  id?: number;
  platform?: OsqueryPlatform;
  teamId?: number;
}

export interface IGetOSVersionsQueryKey extends IGetOSVersionsRequest {
  scope: string;
}
export interface IOSVersionsResponse {
  counts_updated_at: string;
  os_versions: IOperatingSystemVersion[];
}

export const getOSVersions = async ({
  id,
  platform,
  teamId,
}: IGetOSVersionsRequest = {}): Promise<IOSVersionsResponse> => {
  const { OS_VERSIONS } = endpoints;
  let path = OS_VERSIONS;

  const queryParams = { id, platform, team_id: teamId };
  const queryString = buildQueryStringFromParams(queryParams);

  if (queryString) path += `?${queryString}`;

  return sendRequest("GET", path);
};

export default {
  getOSVersions,
};

/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IOsqueryPlatform } from "interfaces/platform";
import { buildQueryStringFromParams } from "utilities/url";

export interface IOperatingSystemsResponse {
  counts_updated_at: string;
  os_versions: IOperatingSystemVersion[];
}

interface IGetVersionParams {
  platform: IOsqueryPlatform;
  teamId?: number;
}

export default {
  getVersions: async ({
    platform,
    teamId,
  }: IGetVersionParams): Promise<IOperatingSystemsResponse> => {
    const { OS_VERSIONS } = endpoints;
    const queryParams = { platform, team_id: teamId };
    const queryString = buildQueryStringFromParams(queryParams);
    const path = `${OS_VERSIONS}?${queryString}`;

    try {
      return sendRequest("GET", path);
    } catch (error) {
      return Promise.reject(error);
    }
  },
};

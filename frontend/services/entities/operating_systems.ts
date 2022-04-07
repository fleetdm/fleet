/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IOsqueryPlatform } from "interfaces/platform";

export interface IOperatingSystemsResponse {
  counts_updated_at: string;
  os_versions: IOperatingSystemVersion[];
}

interface IGetOperatingSystemProps {
  platform: IOsqueryPlatform;
  teamId?: number;
}

export default {
  getVersions: async ({
    platform,
    teamId,
  }: IGetOperatingSystemProps): Promise<IOperatingSystemsResponse> => {
    const { OS_VERSIONS } = endpoints;
    let path = OS_VERSIONS;

    const queryParams = [`platform=${platform}`];

    if (teamId) {
      queryParams.push(`team_id=${teamId}`);
    }

    const queryString = `?${queryParams.join("&")}`;
    path += queryString;

    try {
      return sendRequest("GET", path);
    } catch (error) {
      return Promise.reject(error);
    }
  },
};

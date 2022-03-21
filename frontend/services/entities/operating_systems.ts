/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IOperatingSystemVersion } from "interfaces/operating_system";

export interface IOperatingSystemsResponse {
  counts_updated_at: string;
  os_versions: IOperatingSystemVersion[];
}

type IPlatformParam = "darwin" | "windows" | "linux";

interface IGetOperatingSystemProps {
  platform: IPlatformParam;
  teamId?: number;
}

const MOCK_DATA: IOperatingSystemsResponse = {
  counts_updated_at: "2022-03-20",
  os_versions: [
    {
      id: 1,
      name: "macOS 11.6",
      hosts_count: 8360,
    },
    {
      id: 2,
      name: "macOS 11.6.1",
      hosts_count: 2000,
    },
    {
      id: 3,
      name: "macOS 12.0",
      hosts_count: 400,
    },
    {
      id: 4,
      name: "macOS 12.0.1",
      hosts_count: 112,
    },
    {
      id: 5,
      name: "macOS 12.1",
      hosts_count: 50,
    },
  ],
};

export default {
  getVersions: ({
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

    return Promise.resolve(MOCK_DATA);

    // try {
    //   return sendRequest("GET", path);
    // } catch (error) {
    //   throw error;
    // }
  },
};

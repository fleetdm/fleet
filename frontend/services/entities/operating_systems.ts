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
    {
      id: 6,
      name: "macOS 12.1",
      hosts_count: 50,
    },
    {
      id: 7,
      name: "macOS 12.1",
      hosts_count: 50,
    },
    {
      id: 8,
      name: "macOS 12.1",
      hosts_count: 50,
    },
    // {
    //   id: 9,
    //   name: "macOS 12.1",
    //   hosts_count: 50,
    // },
    // {
    //   id: 10,
    //   name: "macOS 12.1",
    //   hosts_count: 50,
    // },
  ],
};

export default {
  // TODO: enable with real API endpoint
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

    // try {
    //   return sendRequest("GET", path);
    // } catch (error) {
    //   throw error;
    // }

    if (teamId === 2) {
      return Promise.resolve({
        counts_updated_at: "2022-03-20",
        os_versions: [
          {
            id: 1,
            name: "macOS 11.6",
            hosts_count: 8360,
          },
        ],
      });
    }
    if (teamId === 8) {
      return Promise.resolve({} as IOperatingSystemsResponse);
    }

    if (teamId === 9) {
      return Promise.reject("error!!!");
    }

    console.log(path);

    return Promise.resolve(MOCK_DATA);
    // return Promise.resolve({} as IOperatingSystemsResponse);
    // return Promise.reject("foo");
  },
};

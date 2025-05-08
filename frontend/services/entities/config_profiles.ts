import { getConfig } from "@testing-library/react";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetConfigProfileStatusResponse {
  verified: number;
  verifying: number;
  failed: number;
  pending: number;
  counts_updated_at: string;
}

export default {
  getConfigProfileStatus: (
    uuid: string
  ): Promise<IGetConfigProfileStatusResponse> => {
    const { CONFIG_PROFILE_STATUS } = endpoints;
    // return sendRequest("GET", CONFIG_PROFILE_STATUS(uuid));

    return new Promise((resolve) => {
      resolve({
        verified: 0,
        verifying: 1,
        failed: 2,
        pending: 3,
        counts_updated_at: "2023-10-01T00:00:00Z",
      });
    });
  },
};

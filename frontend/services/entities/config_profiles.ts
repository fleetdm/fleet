import { getConfig } from "@testing-library/react";
import { profile } from "console";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetConfigProfileStatusResponse {
  verified: number;
  verifying: number;
  failed: number;
  pending: number;
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
      });
    });
  },

  batchResendConfigProfile: (uuid: string, status: string): Promise<void> => {
    const { CONFIG_PROFILE_BATCH_RESEND } = endpoints;
    const body = {
      profile_uuid: uuid,
      filters: {
        profile_status: status,
      },
    };
    return sendRequest("POST", CONFIG_PROFILE_BATCH_RESEND, body);
  },
};

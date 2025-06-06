import { IMdmProfile } from "interfaces/mdm";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export type IGetConfigProfileResponse = IMdmProfile;

export interface IGetConfigProfileStatusResponse {
  verified: number;
  verifying: number;
  failed: number;
  pending: number;
}

export default {
  getConfigProfile: (uuid: string): Promise<IGetConfigProfileResponse> => {
    const { CONFIG_PROFILE } = endpoints;
    return sendRequest("GET", CONFIG_PROFILE(uuid));
  },

  getConfigProfileStatus: (
    uuid: string
  ): Promise<IGetConfigProfileStatusResponse> => {
    const { CONFIG_PROFILE_STATUS } = endpoints;
    return sendRequest("GET", CONFIG_PROFILE_STATUS(uuid));
  },

  batchResendConfigProfile: (uuid: string): Promise<void> => {
    const { CONFIG_PROFILE_BATCH_RESEND } = endpoints;
    const body = {
      profile_uuid: uuid,
      filters: {
        profile_status: "failed",
      },
    };
    return sendRequest("POST", CONFIG_PROFILE_BATCH_RESEND, body);
  },
};

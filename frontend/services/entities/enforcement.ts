import {
  IEnforcementProfile,
  IEnforcementProfilesResponse,
} from "interfaces/enforcement";

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IGetEnforcementProfilesParams {
  page?: number;
  per_page?: number;
  team_id?: number;
}

export interface IUploadEnforcementProfileParams {
  file: File;
  teamId?: number;
}

const enforcementAPI = {
  getProfiles: (
    params: IGetEnforcementProfilesParams
  ): Promise<IEnforcementProfilesResponse> => {
    const { ENFORCEMENT_PROFILES } = endpoints;
    const path = `${ENFORCEMENT_PROFILES}?${buildQueryStringFromParams({
      ...params,
    })}`;
    return sendRequest("GET", path);
  },

  uploadProfile: ({
    file,
    teamId,
  }: IUploadEnforcementProfileParams): Promise<{ profile_uuid: string }> => {
    const { ENFORCEMENT_PROFILES } = endpoints;

    const formData = new FormData();
    formData.append("profile", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", ENFORCEMENT_PROFILES, formData);
  },

  getProfile: (profileUUID: string): Promise<IEnforcementProfile> => {
    const { ENFORCEMENT_PROFILE } = endpoints;
    return sendRequest("GET", ENFORCEMENT_PROFILE(profileUUID));
  },

  downloadProfile: (profileUUID: string) => {
    const { ENFORCEMENT_PROFILE } = endpoints;
    const path = `${ENFORCEMENT_PROFILE(profileUUID)}?${buildQueryStringFromParams({
      alt: "media",
    })}`;
    return sendRequest("GET", path);
  },

  deleteProfile: (profileUUID: string) => {
    const { ENFORCEMENT_PROFILE } = endpoints;
    return sendRequest("DELETE", ENFORCEMENT_PROFILE(profileUUID));
  },
};

export default enforcementAPI;

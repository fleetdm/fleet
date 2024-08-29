import { IMdmVppToken } from "interfaces/mdm";
import { ApplePlatform } from "interfaces/platform";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetVppInfoResponse {
  org_name: string;
  renew_date: string;
  location: string;
}

export interface IVppApp {
  name: string;
  bundle_identifier: string;
  icon_url: string;
  latest_version: string;
  app_store_id: string;
  added: boolean;
  platform: ApplePlatform;
}

interface IAddVppAppPostBody {
  app_store_id: string;
  team_id: number;
  platform: ApplePlatform;
  self_service?: boolean;
}

export interface IGetVppAppsResponse {
  app_store_apps: IVppApp[];
}

export interface IGetVppTokensResponse {
  vpp_tokens: IMdmVppToken[];
}

export interface IUploadVppTokenReponse {
  vpp_token: IMdmVppToken;
}

export type IRenewVppTokenResponse = IUploadVppTokenReponse;

export default {
  getAppleAPNInfo: () => {
    const { MDM_APPLE_PNS } = endpoints;
    const path = MDM_APPLE_PNS;
    return sendRequest("GET", path);
  },

  uploadApplePushCertificate: (certificate: File) => {
    const { MDM_APPLE_APNS_CERTIFICATE } = endpoints;
    const formData = new FormData();
    formData.append("certificate", certificate);
    return sendRequest("POST", MDM_APPLE_APNS_CERTIFICATE, formData);
  },

  deleteApplePushCertificate: () => {
    const { MDM_APPLE_APNS_CERTIFICATE } = endpoints;
    return sendRequest("DELETE", MDM_APPLE_APNS_CERTIFICATE);
  },

  requestCSR: () => {
    const { MDM_REQUEST_CSR } = endpoints;
    return sendRequest("GET", MDM_REQUEST_CSR);
  },

  disableVpp: () => {
    const { MDM_APPLE_VPP_TOKEN } = endpoints;
    return sendRequest("DELETE", MDM_APPLE_VPP_TOKEN);
  },

  getVppApps: (teamId: number): Promise<IGetVppAppsResponse> => {
    const { MDM_APPLE_VPP_APPS } = endpoints;
    const path = `${MDM_APPLE_VPP_APPS}?team_id=${teamId}`;
    return sendRequest("GET", path);
  },

  addVppApp: (
    teamId: number,
    appStoreId: string,
    platform: ApplePlatform,
    isSelfService: boolean
  ) => {
    const { MDM_APPLE_VPP_APPS } = endpoints;
    const postBody: IAddVppAppPostBody = {
      app_store_id: appStoreId,
      team_id: teamId,
      platform,
    };

    if (isSelfService) {
      postBody.self_service = isSelfService;
    }

    return sendRequest("POST", MDM_APPLE_VPP_APPS, postBody);
  },

  getVppTokens: (): Promise<IGetVppTokensResponse> => {
    const { MDM_VPP_TOKENS } = endpoints;
    return sendRequest("GET", MDM_VPP_TOKENS);
  },

  uploadVppToken: (token: File): Promise<IUploadVppTokenReponse> => {
    const { MDM_VPP_TOKENS } = endpoints;
    const formData = new FormData();
    formData.append("token", token);
    return sendRequest("POST", MDM_VPP_TOKENS, formData);
  },

  renewVppToken(id: number, token: File): Promise<IRenewVppTokenResponse> {
    const { MDM_VPP_TOKENS_RENEW } = endpoints;
    const path = MDM_VPP_TOKENS_RENEW(id);
    const formData = new FormData();
    formData.append("token", token);
    return sendRequest("PATCH", path, formData);
  },

  deleteVppToken: (id: number): Promise<void> => {
    const { MDM_VPP_TOKEN } = endpoints;
    const path = MDM_VPP_TOKEN(id);
    return sendRequest("DELETE", path);
  },

  editVppTeams: async (params: {
    tokenId: number;
    teamIds: number[] | null;
  }) => {
    const { MDM_VPP_TOKEN_TEAMS } = endpoints;
    const path = MDM_VPP_TOKEN_TEAMS(params.tokenId);
    return sendRequest("PATCH", path, { teams: params.teamIds });
  },
};

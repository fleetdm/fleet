import { IMdmVppToken } from "interfaces/mdm";
import { ApplePlatform } from "interfaces/platform";
import { SoftwareCategory } from "interfaces/software";
import { ISoftwareVppFormData } from "pages/SoftwarePage/components/forms/SoftwareVppForm/SoftwareVppForm";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { listNamesFromSelectedLabels } from "components/TargetLabelSelector/TargetLabelSelector";

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

export interface IAddVppAppPostBody {
  app_store_id: string;
  team_id: number;
  platform: ApplePlatform;
  self_service?: boolean;
  automatic_install?: boolean;
  labels_include_any?: string[];
  labels_exclude_any?: string[];
  categories?: SoftwareCategory[];
}

export interface IEditVppAppPostBody {
  team_id: number;
  self_service?: boolean;
  // No automatic_install on edit VPP app
  labels_include_any?: string[];
  labels_exclude_any?: string[];
  categories?: SoftwareCategory[];
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

  addVppApp: (teamId: number, formData: ISoftwareVppFormData) => {
    const { MDM_APPLE_VPP_APPS } = endpoints;

    if (!formData.selectedApp) {
      throw new Error("Selected app is required. This should not happen.");
    }

    const body: IAddVppAppPostBody = {
      app_store_id: formData.selectedApp.app_store_id,
      team_id: teamId,
      platform: formData.selectedApp?.platform,
      self_service: formData.selfService,
      automatic_install: formData.automaticInstall,
    };

    // Add categories if present
    if (formData.categories && formData.categories.length > 0) {
      body.categories = formData.categories as SoftwareCategory[];
    }

    if (formData.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
      if (formData.customTarget === "labelsIncludeAny") {
        body.labels_include_any = selectedLabels;
      } else {
        body.labels_exclude_any = selectedLabels;
      }
    }

    return sendRequest("POST", MDM_APPLE_VPP_APPS, body);
  },

  editVppApp: (
    softwareId: number,
    teamId: number,
    formData: ISoftwareVppFormData
  ) => {
    const { EDIT_SOFTWARE_VPP } = endpoints;

    const body: IEditVppAppPostBody = {
      self_service: formData.selfService,
      team_id: teamId,
    };

    // Add categories if present
    if (formData.categories && formData.categories.length > 0) {
      body.categories = formData.categories as SoftwareCategory[];
    }

    if (formData.targetType === "Custom") {
      const selectedLabels = listNamesFromSelectedLabels(formData.labelTargets);
      if (formData.customTarget === "labelsIncludeAny") {
        body.labels_include_any = selectedLabels;
      } else {
        body.labels_exclude_any = selectedLabels;
      }
    }

    return sendRequest("PATCH", EDIT_SOFTWARE_VPP(softwareId), body);
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

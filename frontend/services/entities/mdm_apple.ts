import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IGetVppInfoResponse {
  org_name: string;
  renew_date: string;
  location: string;
}

export interface IVppApp {
  name: string;
  icon_url: string;
  latest_version: string;
  app_store_id: number; // FIXME: This seems to be coming back as a string but we're not seeing TS errors because there's an implicit any cast
  added: boolean;
  platform: string; // "darwin" | "ios" | "ipados"
}

interface IGetVppAppsResponse {
  app_store_apps: IVppApp[];
}

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

  getVppInfo: (): Promise<IGetVppInfoResponse> => {
    const { MDM_APPLE_VPP } = endpoints;
    return sendRequest("GET", MDM_APPLE_VPP);
  },

  uploadVppToken: (token: File) => {
    const { MDM_APPLE_VPP_TOKEN } = endpoints;
    const formData = new FormData();
    formData.append("token", token);
    return sendRequest("POST", MDM_APPLE_VPP_TOKEN, formData);
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

  addVppApp: (teamId: number, appStoreId: number) => {
    // FIXME: API seems to expect app_store_id to be a string but we're not seeing TS errors because
    // there's an implicit any cast on getVppApps response that is being fed into the appStoreId
    // param (so it's being treated as a number even though it is really a string being passed in here)
    const { MDM_APPLE_VPP_APPS } = endpoints;
    return sendRequest("POST", MDM_APPLE_VPP_APPS, {
      app_store_id: appStoreId,
      team_id: teamId,
    });
  },
};

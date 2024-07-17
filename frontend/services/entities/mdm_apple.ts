import { createMockVppApp } from "__mocks__/appleMdm";
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
  app_store_id: number;
  added: boolean;
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
    // return sendRequest("GET", MDM_APPLE_VPP_APPS(teamId));

    return new Promise((resolve) =>
      resolve({
        app_store_apps: [
          createMockVppApp({ name: "Test App 1", app_store_id: 1 }),
          createMockVppApp({ name: "Test App 2", app_store_id: 2 }),
          createMockVppApp({ name: "Test App 3", app_store_id: 3 }),
        ],
      })
    );
  },

  addVppApp: (teamId: number, appStoreId: number) => {
    const { MDM_APPLE_VPP_APPS } = endpoints;
    return sendRequest("POST", MDM_APPLE_VPP_APPS, {
      app_store_id: appStoreId,
      team_id: teamId,
    });
  },
};

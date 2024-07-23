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
  platform: string; // darwin | ios | ipados
}

// `GET /api/v1/fleet/software/app_store_apps`
interface IGetVppAppsResponse {
  app_store_apps: IVppApp[];
}

// {
//   "app_store_apps": [
//     {
//       "name": "Xcode",
//       "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
//       "latest_version": "15.4",
//       "app_store_id": 497799835,
//       "added": true,
//       "platform": "darwin"
//     },
//     {
//       "name": "Logic Pro",
//       "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
//       "latest_version": "2.04",
//       "app_store_id": 634148309,
//       "added": false,
//       "platform": "ios"
//     },
//     {
//       "name": "Logic Pro",
//       "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
//       "latest_version": "2.04",
//       "app_store_id": 634148309,
//       "added": false,
//       "platform": "ipados"
//     },
//   ]
// }

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
    const { MDM_APPLE_VPP_APPS } = endpoints;
    return sendRequest("POST", MDM_APPLE_VPP_APPS, {
      app_store_id: appStoreId,
      team_id: teamId,
    });
  },
};

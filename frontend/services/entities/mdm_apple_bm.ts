/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { IMdmAbmToken } from "interfaces/mdm";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IAppleBusinessManagerTokenFormData {
  token: File | null;
}

export interface IGetAppleBMInfoResponse {
  apple_id: string;
  default_team: string;
  mdm_server_url: string;
  org_name: string;
  renew_date: string;
}

export default {
  getAppleBMInfo: (): Promise<IGetAppleBMInfoResponse> => {
    const { MDM_APPLE_BM } = endpoints;
    const path = MDM_APPLE_BM;
    return sendRequest("GET", path);
  },
  loadKeys: () => {
    const { MDM_APPLE_BM_KEYS } = endpoints;
    const path = MDM_APPLE_BM_KEYS;

    return sendRequest("POST", path).then(({ private_key, public_key }) => {
      let decodedPublic;
      let decodedPrivate;
      try {
        decodedPublic = global.window.atob(public_key);
        decodedPrivate = global.window.atob(private_key);
      } catch (err) {
        return Promise.reject(`Unable to decode keys: ${err}`);
      }
      if (!decodedPrivate || !decodedPublic) {
        return Promise.reject("Missing or undefined keys.");
      }

      return Promise.resolve({ decodedPublic, decodedPrivate });
    });
  },

  downloadPublicKey: () => {
    const { MDM_APPLE_ABM_PUBLIC_KEY } = endpoints;
    return sendRequest("GET", MDM_APPLE_ABM_PUBLIC_KEY);
  },

  uploadToken: (token: File) => {
    const { MDM_APPLE_ABM_TOKEN: MDM_APPLE_BM_TOKEN } = endpoints;
    const formData = new FormData();
    formData.append("token", token);

    return sendRequest("POST", MDM_APPLE_BM_TOKEN, formData);
  },

  disableAutomaticEnrollment: () => {
    const { MDM_APPLE_ABM_TOKEN: MDM_APPLE_BM_TOKEN } = endpoints;
    return sendRequest("DELETE", MDM_APPLE_BM_TOKEN);
  },

  getTokens: (): Promise<IMdmAbmToken[]> => {
    const { MDM_ABM_TOKENS } = endpoints;
    console.log("Fetching ABM tokens from:", MDM_ABM_TOKENS);
    // return sendRequest("GET", MDM_ABM_TOKENS);
    return Promise.resolve([
      {
        id: 1,
        apple_id: "apple@example.com",
        org_name: "Fleet Device Management Inc.",
        mdm_server_url: "https://example.com/mdm/apple/mdm",
        renew_date: "foo", // TODO: test what happens with invalid date
        terms_expired: false,
        macos_team: "💻 Workstations",
        ios_team: "📱🏢 Company-owned iPhones",
        ipados_team: "🔳🏢 Company-owned iPads",
      },
      {
        id: 2,
        apple_id: "apple@example.com",
        org_name: "Fleet Device Management Inc.",
        mdm_server_url: "https://example.com/mdm/apple/mdm",
        renew_date: "2023-11-29T00:00:00Z",
        terms_expired: false,
        macos_team: "💻 Workstations",
        ios_team: "📱🏢 Company-owned iPhones",
        ipados_team: "🔳🏢 Company-owned iPads",
      },
    ]); // Temporary stub
  },
};

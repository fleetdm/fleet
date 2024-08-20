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

  uploadToken: (token: File): Promise<IMdmAbmToken> => {
    const { MDM_ABM_TOKENS } = endpoints;
    const formData = new FormData();
    formData.append("token", token);

    return sendRequest("POST", MDM_ABM_TOKENS, formData);
  },

  renewToken: (id: number, token: File): Promise<void> => {
    const { MDM_ABM_TOKEN_RENEW } = endpoints;
    const path = MDM_ABM_TOKEN_RENEW(id);

    const formData = new FormData();
    formData.append("token", token);

    return sendRequest("PATCH", path, formData);
  },

  deleteToken: (id: number): Promise<void> => {
    const { MDM_ABM_TOKEN } = endpoints;
    const path = MDM_ABM_TOKEN(id);
    return sendRequest("DELETE", path);
  },

  getTokens: (): Promise<IMdmAbmToken[]> => {
    const { MDM_ABM_TOKENS } = endpoints;
    console.log("Fetching ABM tokens from:", MDM_ABM_TOKENS);
    // return sendRequest("GET", MDM_ABM_TOKENS);
    return Promise.resolve([]); // TODO: remove when API is ready
  },
};

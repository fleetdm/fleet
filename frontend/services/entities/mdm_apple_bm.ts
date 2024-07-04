/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IAppleBusinessManagerTokenFormData {
  token: File | null;
}

export default {
  getAppleBMInfo: () => {
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
};

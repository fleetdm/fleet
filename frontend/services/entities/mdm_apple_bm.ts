/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

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
};

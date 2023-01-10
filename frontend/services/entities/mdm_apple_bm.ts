/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { sendRequest } from "services/mock_service/service/service"; // MDM TODO: Replace when backend is merged
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  // TODO: set up full MDM testing environment, including all keys/credentials/tokens, to be able to
  // get proper data from this API as opposed to the mock backend above
  getAppleBMInfo: () => {
    const { MDM_APPLE_BM } = endpoints;
    const path = MDM_APPLE_BM;
    return sendRequest("GET", path);
  },
  loadKeys: () => {
    const { MDM_APPLE_BM_KEYS } = endpoints;
    const path = MDM_APPLE_BM_KEYS;

    // MDM TODO: Originally written for certificate_chain for certificate, refactor for keys when backend is merged
    return sendRequest("GET", path).then(({ certificate_chain }) => {
      let decodedKeys;
      try {
        decodedKeys = global.window.atob(certificate_chain);
      } catch (err) {
        return Promise.reject(`Unable to decode keys: ${err}`);
      }
      if (!decodedKeys) {
        return Promise.reject("Missing or undefined keys.");
      }

      return Promise.resolve(decodedKeys);
    });
  },
};

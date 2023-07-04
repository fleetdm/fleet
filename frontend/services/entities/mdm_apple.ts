/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  getAppleAPNInfo: () => {
    const { MDM_APPLE } = endpoints;
    const path = MDM_APPLE;
    return sendRequest("GET", path);
  },
};

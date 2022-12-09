/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { sendRequest } from "services/mock_service/service/service";
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  loadAll: () => {
    const { MDM_APPLE_BM } = endpoints;
    const path = MDM_APPLE_BM;
    return sendRequest("GET", path);
  },
};

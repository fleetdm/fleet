/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { sendRequest } from "services/mock_service/service/service"; // MDM TODO: Replace when backend is merged
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  loadAll: () => {
    const { MDM_APPLE } = endpoints;
    const path = MDM_APPLE;
    return sendRequest("GET", path);
  },
};

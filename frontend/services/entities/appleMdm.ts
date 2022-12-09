// TODO: Correct API call once backend is done

/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { sendRequest } from "services/mock_service/service/service";
// import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  loadAll: () => {
    const { APPLE_MDM } = endpoints;
    const path = APPLE_MDM;
    return sendRequest("GET", path);
  },
};

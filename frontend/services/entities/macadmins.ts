/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";

export default {
  loadAll: () => {
    const { MACADMINS } = endpoints;

    return sendRequest("GET", MACADMINS);
  },
};

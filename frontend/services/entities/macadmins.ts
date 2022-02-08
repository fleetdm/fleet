/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";

export default {
  loadAll: (teamId?: number) => {
    const { MACADMINS } = endpoints;
    let path = MACADMINS;

    if (teamId) {
      path += `?team_id=${teamId}`;
    }

    return sendRequest("GET", path);
  },
};

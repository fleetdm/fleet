/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export default {
  loadAll: (teamId?: number) => {
    const { MACADMINS } = endpoints;
    const queryString = buildQueryStringFromParams({ team_id: teamId });
    const path = `${MACADMINS}?${queryString}`;

    return sendRequest("GET", path);
  },
};

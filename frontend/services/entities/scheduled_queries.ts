/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";

import endpoints from "utilities/endpoints";
import {
  IPackQueryFormData,
  IScheduledQuery,
} from "interfaces/scheduled_query";
import helpers from "utilities/helpers";

export default {
  create: (packQueryFormData: IPackQueryFormData) => {
    const { SCHEDULE_QUERY } = endpoints;

    return sendRequest("POST", SCHEDULE_QUERY, packQueryFormData);
  },
  destroy: (packQueryId: number) => {
    const { SCHEDULE_QUERY } = endpoints;
    const path = `${SCHEDULE_QUERY}/${packQueryId}`;

    return sendRequest("DELETE", path);
  },
  loadAll: (packId: number) => {
    const { SCHEDULED_QUERIES } = endpoints;
    const path = SCHEDULED_QUERIES(packId);

    return sendRequest("GET", path);
  },
  update: (
    scheduledQuery: IScheduledQuery,
    updatedAttributes: IPackQueryFormData
  ) => {
    const { SCHEDULE_QUERY } = endpoints;
    const path = `${SCHEDULE_QUERY}/${scheduledQuery.id}`;
    const params = helpers.formatScheduledQueryForServer(updatedAttributes);

    return sendRequest("PATCH", path, params);
  },
};

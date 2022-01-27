/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import { omit } from "lodash";

import endpoints from "fleet/endpoints";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import helpers from "fleet/helpers";

export default {
  create: (formData: any) => {
    const { GLOBAL_SCHEDULE } = endpoints;

    const {
      interval,
      logging_type: loggingType,
      platform,
      query_id: queryID,
      shard,
      version,
    } = formData;

    const removed = loggingType === "differential";
    const snapshot = loggingType === "snapshot";

    const params = {
      interval: Number(interval),
      platform,
      query_id: Number(queryID),
      removed,
      snapshot,
      shard: Number(shard),
      version,
    };

    return sendRequest("POST", GLOBAL_SCHEDULE, params);
  },
  destroy: ({ id }: { id: number }) => {
    const { GLOBAL_SCHEDULE } = endpoints;
    const path = `${GLOBAL_SCHEDULE}/${id}`;

    return sendRequest("DELETE", path);
  },
  loadAll: () => {
    const { GLOBAL_SCHEDULE } = endpoints;
    const path = GLOBAL_SCHEDULE;

    return sendRequest("GET", path);
  },
  update: (
    globalScheduledQuery: IGlobalScheduledQuery,
    updatedAttributes: any
  ) => {
    const { GLOBAL_SCHEDULE } = endpoints;
    const path = `${GLOBAL_SCHEDULE}/${globalScheduledQuery.id}`;
    const params = helpers.formatScheduledQueryForServer(updatedAttributes);

    return sendRequest("PATCH", path, params);
  },
};

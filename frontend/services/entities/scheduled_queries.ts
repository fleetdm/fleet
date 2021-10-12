/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import { omit } from "lodash";

import endpoints from "fleet/endpoints";
import {
  IPackQueryFormData,
  IScheduledQuery,
} from "interfaces/scheduled_query";
import helpers from "fleet/helpers";

export default {
  create: (packQueryFormData: IPackQueryFormData) => {
    const { SCHEDULED_QUERIES } = endpoints;

    return sendRequest("POST", SCHEDULED_QUERIES, packQueryFormData);
  },
  destroy: (packQueryId: number) => {
    const { SCHEDULED_QUERIES } = endpoints;
    const path = `${SCHEDULED_QUERIES}/${packQueryId}`;

    return sendRequest("DELETE", path);
  },
  loadAll: (packId: number) => {
    const { SCHEDULED_QUERY } = endpoints;
    const path = SCHEDULED_QUERY(packId);

    return sendRequest("GET", path);
  },
  update: (
    scheduledQuery: IScheduledQuery,
    updatedAttributes: IPackQueryFormData
  ) => {
    const { SCHEDULED_QUERIES } = endpoints;
    const path = `${SCHEDULED_QUERIES}/${scheduledQuery.id}`;
    const params = helpers.formatScheduledQueryForServer(updatedAttributes);

    return sendRequest("PATCH", path, params);
  },
};

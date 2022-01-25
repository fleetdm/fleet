/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import { omit } from "lodash";

import endpoints from "fleet/endpoints";
import {
  // IPackQueryFormData,
  ITeamScheduledQuery,
} from "interfaces/team_scheduled_query";
import helpers from "fleet/helpers";

export default {
  create: (formData: any) => {
    const { TEAM_SCHEDULE } = endpoints;

    const {
      interval,
      logging_type: loggingType,
      platform,
      query_id: queryID,
      shard,
      version,
      team_id: teamID,
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
      team_id: Number(teamID),
    };

    return sendRequest("POST", TEAM_SCHEDULE(teamID), params);
  },
  destroy: (teamID: number, queryID: number) => {
    const { TEAM_SCHEDULE } = endpoints;
    const path = `${TEAM_SCHEDULE(teamID)}/${queryID}`;

    return sendRequest("DELETE", path);
  },
  loadAll: (teamID: number) => {
    const { TEAM_SCHEDULE } = endpoints;
    const path = TEAM_SCHEDULE(teamID);

    return sendRequest("GET", path);
  },
  update: (teamScheduledQuery: ITeamScheduledQuery, updatedAttributes: any) => {
    const { team_id } = updatedAttributes;
    const { TEAM_SCHEDULE } = endpoints;
    const path = `${TEAM_SCHEDULE(team_id)}/${teamScheduledQuery.id}`;
    const params = helpers.formatScheduledQueryForServer(updatedAttributes);

    return sendRequest("PATCH", path, params);
  },
};

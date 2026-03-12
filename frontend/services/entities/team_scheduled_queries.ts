/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";

import endpoints from "utilities/endpoints";
import {
  IScheduledQuery,
  IUpdateTeamScheduledQuery,
} from "interfaces/scheduled_query";
import helpers from "utilities/helpers";
import { API_NO_TEAM_ID } from "interfaces/team";

interface ICreateTeamScheduledQueryFormData {
  interval: number;
  logging_type: string;
  name?: string;
  platform: string;
  report_id: number;
  shard: number;
  fleet_id: number;
  version: string;
}

export default {
  create: (formData: ICreateTeamScheduledQueryFormData) => {
    const { TEAM_SCHEDULE } = endpoints;

    const {
      interval,
      logging_type: loggingType,
      platform,
      report_id: reportID,
      shard,
      version,
      fleet_id: fleetID,
    } = formData;

    if (fleetID <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${fleetID} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }

    const removed = loggingType === "differential";
    const snapshot = loggingType === "snapshot";

    const params = {
      interval: Number(interval),
      platform,
      report_id: reportID,
      removed,
      snapshot,
      shard: Number(shard),
      version,
      fleet_id: fleetID,
    };

    return sendRequest("POST", TEAM_SCHEDULE(fleetID), params);
  },
  destroy: (teamId: number | undefined, queryID: number) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAM_SCHEDULE } = endpoints;
    const path = `${TEAM_SCHEDULE(teamId)}/${queryID}`;

    return sendRequest("DELETE", path);
  },
  loadAll: (teamId?: number) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAM_SCHEDULE } = endpoints;
    const path = TEAM_SCHEDULE(teamId);

    return sendRequest("GET", path);
  },
  update: (
    teamScheduledQuery: IScheduledQuery,
    updatedAttributes: IUpdateTeamScheduledQuery
  ) => {
    const { team_id } = updatedAttributes;
    const { TEAM_SCHEDULE } = endpoints;
    const path = `${TEAM_SCHEDULE(team_id)}/${teamScheduledQuery.id}`;
    const params = helpers.formatScheduledQueryForServer(updatedAttributes);

    return sendRequest("PATCH", path, params);
  },
};

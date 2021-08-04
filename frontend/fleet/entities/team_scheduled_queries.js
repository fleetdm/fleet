import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    create: (formData) => {
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

      return client
        .authenticatedPost(
          client._endpoint(TEAM_SCHEDULE(teamID)),
          JSON.stringify(params)
        )
        .then((response) => response.scheduled);
    },
    destroy: (teamID, queryID) => {
      const { TEAM_SCHEDULE } = endpoints;
      const endpoint = `${client._endpoint(TEAM_SCHEDULE(teamID))}/${queryID}`;

      return client.authenticatedDelete(endpoint);
    },
    loadAll: (teamID) => {
      const { TEAM_SCHEDULE } = endpoints;
      const teamScheduledQueryPath = TEAM_SCHEDULE(teamID);

      return client
        .authenticatedGet(client._endpoint(teamScheduledQueryPath))
        .then((response) => response.scheduled);
    },
    update: (teamScheduledQuery, updatedAttributes) => {
      const { team_id } = updatedAttributes;
      const { TEAM_SCHEDULE } = endpoints;
      const endpoint = client._endpoint(
        `${TEAM_SCHEDULE(team_id)}/${teamScheduledQuery.id}`
      );
      const params = helpers.formatTeamScheduledQueryForServer(
        updatedAttributes
      );

      return client
        .authenticatedPatch(endpoint, JSON.stringify(params))
        .then((response) => response.scheduled);
    },
  };
};

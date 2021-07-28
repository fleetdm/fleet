import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    create: (formData) => {
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

      return client
        .authenticatedPost(
          client._endpoint(GLOBAL_SCHEDULE),
          JSON.stringify(params)
        )
        .then((response) => response.scheduled);
    },
    destroy: ({ id }) => {
      const { GLOBAL_SCHEDULE } = endpoints;
      const endpoint = `${client._endpoint(GLOBAL_SCHEDULE)}/${id}`;

      return client.authenticatedDelete(endpoint);
    },
    loadAll: () => {
      const { GLOBAL_SCHEDULE } = endpoints;
      const globalScheduledQueryPath = GLOBAL_SCHEDULE;

      return client
        .authenticatedGet(client._endpoint(globalScheduledQueryPath))
        .then((response) => response.global_schedule);
    },
    update: (globalScheduledQuery, updatedAttributes) => {
      const { GLOBAL_SCHEDULE } = endpoints;
      const endpoint = client._endpoint(
        `${GLOBAL_SCHEDULE}/${globalScheduledQuery.id}`
      );
      const params = helpers.formatGlobalScheduledQueryForServer(
        updatedAttributes
      );

      return client
        .authenticatedPatch(endpoint, JSON.stringify(params))
        .then((response) => response.scheduled);
    },
  };
};

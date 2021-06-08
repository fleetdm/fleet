import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    create: (formData) => {
      const { SCHEDULED_QUERIES } = endpoints;
      const {
        interval,
        logging_type: loggingType,
        pack_id: packID,
        platform,
        query_id: queryID,
        shard,
        version,
      } = formData;
      const removed = loggingType === "differential";
      const snapshot = loggingType === "snapshot";

      const params = {
        interval: Number(interval),
        pack_id: Number(packID),
        platform,
        query_id: Number(queryID),
        removed,
        snapshot,
        shard: Number(shard),
        version,
      };

      return client
        .authenticatedPost(
          client._endpoint(SCHEDULED_QUERIES),
          JSON.stringify(params)
        )
        .then((response) => response.scheduled);
    },
    destroy: ({ id }) => {
      const { SCHEDULED_QUERIES } = endpoints;
      const endpoint = `${client._endpoint(SCHEDULED_QUERIES)}/${id}`;

      return client.authenticatedDelete(endpoint);
    },
    loadAll: (pack) => {
      const { SCHEDULED_QUERY } = endpoints;
      const scheduledQueryPath = SCHEDULED_QUERY(pack);

      return client
        .authenticatedGet(client._endpoint(scheduledQueryPath))
        .then((response) => response.scheduled);
    },
    update: (scheduledQuery, updatedAttributes) => {
      const { SCHEDULED_QUERIES } = endpoints;
      const endpoint = client._endpoint(
        `${SCHEDULED_QUERIES}/${scheduledQuery.id}`
      );
      const params = helpers.formatScheduledQueryForServer(updatedAttributes);

      return client
        .authenticatedPatch(endpoint, JSON.stringify(params))
        .then((response) => response.scheduled);
    },
  };
};

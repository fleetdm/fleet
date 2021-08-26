import endpoints from "fleet/endpoints";

export default (client) => {
  return {
    create: ({ description, name, query, observer_can_run }) => {
      const { QUERIES } = endpoints;

      return client
        .authenticatedPost(
          client._endpoint(QUERIES),
          JSON.stringify({ description, name, query, observer_can_run })
        )
        .then((response) => response.query);
    },
    destroy: ({ id }) => {
      const { QUERIES } = endpoints;
      const endpoint = `${client._endpoint(QUERIES)}/id/${id}`;

      return client.authenticatedDelete(endpoint);
    },
    load: (queryID) => {
      const { QUERIES } = endpoints;
      const endpoint = `${client.baseURL}${QUERIES}/${queryID}`;

      return client
        .authenticatedGet(endpoint)
        .then((response) => response.query);
    },
    loadAll: () => {
      const { QUERIES } = endpoints;

      return client
        .authenticatedGet(client._endpoint(QUERIES))
        .then((response) => response.queries);
    },
    run: ({ query, selected, query_id }) => {
      const { RUN_QUERY } = endpoints;

      return client
        .authenticatedPost(
          client._endpoint(RUN_QUERY),
          JSON.stringify({ query, selected, query_id })
        )
        .then((response) => {
          const { campaign } = response;

          return {
            ...campaign,
            hosts_count: {
              successful: 0,
              failed: 0,
              total: 0,
            },
          };
        });
    },

    update: ({ id }, updateParams) => {
      const { QUERIES } = endpoints;
      const updateQueryEndpoint = `${client.baseURL}${QUERIES}/${id}`;

      return client
        .authenticatedPatch(updateQueryEndpoint, JSON.stringify(updateParams))
        .then((response) => response.query);
    },
  };
};

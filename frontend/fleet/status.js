import endpoints from "fleet/endpoints";

export default (client) => {
  return {
    result_store: () => {
      const { STATUS_RESULT_STORE } = endpoints;
      const endpoint = client.baseURL + STATUS_RESULT_STORE;

      return client.authenticatedGet(endpoint);
    },
    live_query: () => {
      const { STATUS_LIVE_QUERY } = endpoints;
      const endpoint = client.baseURL + STATUS_LIVE_QUERY;

      return client.authenticatedGet(endpoint);
    },
  };
};

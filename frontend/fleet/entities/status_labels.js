import endpoints from "fleet/endpoints";

export default (client) => {
  return {
    getCounts: () => {
      const { STATUS_LABEL_COUNTS } = endpoints;

      return client
        .authenticatedGet(client._endpoint(STATUS_LABEL_COUNTS))
        .then((response) => {
          return {
            ...response,
            total_count: response.online_count + response.offline_count,
          };
        })
        .catch(() => false);
    },
  };
};

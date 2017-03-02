import endpoints from 'kolide/endpoints';

export default (client) => {
  return {
    getCounts: () => {
      const { STATUS_LABEL_COUNTS } = endpoints;

      return client.authenticatedGet(client._endpoint(STATUS_LABEL_COUNTS))
        .catch(() => false);
    },
  };
};

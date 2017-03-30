import endpoints from 'kolide/endpoints';

export default (client) => {
  return {
    loadAll: () => {
      const { CONFIG_OPTIONS } = endpoints;

      return client.authenticatedGet(client._endpoint(CONFIG_OPTIONS))
        .then(response => response.options);
    },
    update: (options) => {
      const { CONFIG_OPTIONS } = endpoints;

      return client.authenticatedPatch(client._endpoint(CONFIG_OPTIONS), JSON.stringify({ options }))
        .then(response => response.options);
    },
    reset: () => {
      const { CONFIG_OPTIONS_RESET } = endpoints;

      return client.authenticatedGet(client._endpoint(CONFIG_OPTIONS_RESET))
        .then(response => response.options);
    },
  };
};

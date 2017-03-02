import endpoints from 'kolide/endpoints';

export default (client) => {
  return {
    destroy: (host) => {
      const { HOSTS } = endpoints;
      const endpoint = client._endpoint(`${HOSTS}/${host.id}`);

      return client.authenticatedDelete(endpoint);
    },
    loadAll: () => {
      const { HOSTS } = endpoints;

      return client.authenticatedGet(client._endpoint(HOSTS))
        .then(response => response.hosts);
    },
  };
};

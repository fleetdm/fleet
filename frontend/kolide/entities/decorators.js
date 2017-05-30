import endpoints from 'kolide/endpoints';

export default (client) => {
  return {
    loadAll: () => {
      const { DECORATORS } = endpoints;
      return client.authenticatedGet(client._endpoint(DECORATORS))
        .then(response => response.decorators);
    },
    create: (formData) => {
      const { DECORATORS } = endpoints;
      const request = { payload: formData };
      return client.authenticatedPost(client._endpoint(DECORATORS), JSON.stringify(request))
        .then(response => response.decorator);
    },
    destroy: ({ id }) => {
      const { DECORATORS } = endpoints;
      const endpoint = `${client._endpoint(DECORATORS)}/${id}`;
      return client.authenticatedDelete(endpoint);
    },
    update: (formData) => {
      const { DECORATORS } = endpoints;
      const endpoint = `${client._endpoint(DECORATORS)}/${formData.id}`;
      const request = { payload: formData };
      return client.authenticatedPatch(endpoint, JSON.stringify(request))
        .then(response => response.decorator);
    },
  };
};

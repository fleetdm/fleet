import endpoints from "fleet/endpoints";

export default (client) => {
  return {
    load: () => {
      const { VERSION } = endpoints;
      return client.authenticatedGet(client._endpoint(VERSION));
    },
  };
};

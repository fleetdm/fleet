import endpoints from 'kolide/endpoints';
import helpers from 'kolide/helpers';

export default (client) => {
  return {
    loadAll: () => {
      const { OSQUERY_OPTIONS } = endpoints;
      
      return client.authenticatedGet(client._endpoint(OSQUERY_OPTIONS));
    },
    update: (formData) => {
      const { OSQUERY_OPTIONS } = endpoints;
      const osqueryOptionsData = helpers.formatOsqueryOptionsDataForServer(formData);

      return client.authenticatedPatch(client._endpoint(OSQUERY_OPTIONS), JSON.stringify(osqueryOptionsData));
    },
  };
};

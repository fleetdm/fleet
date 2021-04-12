import yaml from "js-yaml";

import endpoints from "kolide/endpoints";

export default (client) => {
  return {
    loadAll: () => {
      const { OSQUERY_OPTIONS } = endpoints;
      return client.authenticatedGet(client._endpoint(OSQUERY_OPTIONS));
    },
    update: (formData) => {
      const { OSQUERY_OPTIONS } = endpoints;
      const osqueryOptionsData = yaml.safeLoad(formData.osquery_options);

      return client.authenticatedPost(
        client._endpoint(OSQUERY_OPTIONS),
        JSON.stringify(osqueryOptionsData)
      );
    },
  };
};

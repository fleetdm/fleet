import yaml from "js-yaml";

import endpoints from "fleet/endpoints";

export default (client: any) => {
  return {
    // Unneeded for teams, but might need this for global
    loadAll: () => {
      const { OSQUERY_OPTIONS } = endpoints;
      return client.authenticatedGet(client._endpoint(OSQUERY_OPTIONS));
    },
    update: (osqueryOptionsData: any, endpoint: string) => {
      const yamlOptions = yaml.load(osqueryOptionsData.osquery_options);
      return client.authenticatedPost(
        client._endpoint(endpoint),
        JSON.stringify(yamlOptions)
      );
    },
  };
};

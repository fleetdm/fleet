import yaml from "js-yaml";

import endpoints from "kolide/endpoints";

export default (client) => {
  return {
    loadAll: () => {
      const { OSQUERY_OPTIONS } = endpoints;
      return client.authenticatedGet(client._endpoint(OSQUERY_OPTIONS));
    },
    // network tab when saving
    // post request passing config as the json object
    // first keys needs to be teamId
    // second key should be config for the post body
    // if there's no team id, need condition here to send to somewhere
    update: (formData, teamId) => {
      const { OSQUERY_OPTIONS } = endpoints;
      const osqueryOptionsData = yaml.safeLoad(formData.osquery_options);

      return client.authenticatedPost(
        client._endpoint(OSQUERY_OPTIONS),
        JSON.stringify(osqueryOptionsData)
      );
    },
  };
};

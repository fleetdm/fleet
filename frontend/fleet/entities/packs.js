import { omit } from "lodash";

import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    addLabel: ({ packID, labelID }) => {
      const path = `/v1/fleet/packs/${packID}/labels/${labelID}`;

      return client.authenticatedPost(client._endpoint(path));
    },
    addQuery: ({ packID, queryID }) => {
      const endpoint = `/v1/fleet/packs/${packID}/queries/${queryID}`;

      return client.authenticatedPost(client._endpoint(endpoint));
    },
    create: ({ name, description, targets }) => {
      const { PACKS } = endpoints;
      const packTargets = helpers.formatSelectedTargetsForApi(targets, true);

      return client
        .authenticatedPost(
          client._endpoint(PACKS),
          JSON.stringify({ description, name, ...packTargets })
        )
        .then((response) => response.pack);
    },
    destroy: ({ id }) => {
      const { PACKS } = endpoints;
      const endpoint = `${client._endpoint(PACKS)}/id/${id}`;

      return client.authenticatedDelete(endpoint);
    },
    load: (packID) => {
      const { PACKS } = endpoints;
      const getPackEndpoint = `${client.baseURL}${PACKS}/${packID}`;

      return client
        .authenticatedGet(getPackEndpoint)
        .then((response) => response.pack);
    },
    loadAll: () => {
      const { PACKS } = endpoints;

      return client
        .authenticatedGet(client._endpoint(PACKS))
        .then((response) => response.packs);
    },
    update: (pack, updatedPack) => {
      const { PACKS } = endpoints;
      const { targets } = updatedPack;
      const updatePackEndpoint = `${client.baseURL}${PACKS}/${pack.id}`;
      let packTargets = null;
      if (targets) {
        packTargets = helpers.formatSelectedTargetsForApi(targets, true);
      }

      const packWithoutTargets = omit(updatedPack, "targets");
      const packParams = { ...packWithoutTargets, ...packTargets };

      return client
        .authenticatedPatch(updatePackEndpoint, JSON.stringify(packParams))
        .then((response) => response.pack);
    },
  };
};

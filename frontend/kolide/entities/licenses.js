import { trim } from 'lodash';

import endpoints from 'kolide/endpoints';
import helpers from 'kolide/helpers';

export default (client) => {
  return {
    create: (jwtToken) => {
      const { LICENSE } = endpoints;

      return client.authenticatedPost(client._endpoint(LICENSE), JSON.stringify({ license: trim(jwtToken) }))
        .then(response => helpers.parseLicense(response.license));
    },

    load: () => {
      const { LICENSE } = endpoints;

      return client.authenticatedGet(client._endpoint(LICENSE))
        .then(response => helpers.parseLicense(response.license));
    },
    setup: (jwtToken) => {
      const { SETUP_LICENSE } = endpoints;

      return client.authenticatedPost(client._endpoint(SETUP_LICENSE), JSON.stringify({ license: trim(jwtToken) }))
        .then(response => helpers.parseLicense(response.license));
    },
  };
};
